package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/apognu/gocal"
)

type Event struct {
	Title    string    `json:"title"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"allDay"`
	Location string    `json:"location,omitempty"`
	// Whose calendar it's on, by name. Two names means it's on both.
	Who []string `json:"who"`
}

type CalendarResponse struct {
	// In config order. The first person is drawn as the solid line.
	People []string `json:"people"`
	Events []Event  `json:"events"`
}

func calendarHandler(cfg CalendarConfig) http.Handler {
	c := &cached[CalendarResponse]{
		ttl: 5 * time.Minute,
		fetch: func(ctx context.Context) (CalendarResponse, error) {
			return fetchCalendar(ctx, cfg)
		},
	}
	return serveCached(c)
}

// people turns the config into one list, accepting the older flat
// icsUrls list as a single unnamed person.
func people(cfg CalendarConfig) []Person {
	var out []Person
	for _, p := range cfg.People {
		if urls := realURLs(p.ICSURLs); len(urls) > 0 {
			out = append(out, Person{Name: p.Name, ICSURLs: urls})
		}
	}
	if len(out) == 0 {
		if urls := realURLs(cfg.ICSURLs); len(urls) > 0 {
			out = append(out, Person{Name: "", ICSURLs: urls})
		}
	}
	return out
}

// realURLs skips placeholders, so a person without an address yet is
// simply left off instead of breaking everyone's calendar.
func realURLs(urls []string) []string {
	var out []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		// Apple Calendar's share links start with webcal://, which is
		// plain HTTPS underneath.
		if strings.HasPrefix(u, "webcal://") {
			u = "https://" + strings.TrimPrefix(u, "webcal://")
		}
		if strings.HasPrefix(u, "https://") && !strings.Contains(u, "YOUR_ID") {
			out = append(out, u)
		}
	}
	return out
}

func fetchCalendar(ctx context.Context, cfg CalendarConfig) (CalendarResponse, error) {
	ppl := people(cfg)
	if len(ppl) == 0 {
		return CalendarResponse{}, errors.New("calendar: no calendar addresses in config")
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, cfg.DaysAhead)

	resp := CalendarResponse{People: []string{}, Events: []Event{}}
	// The same event on both calendars (an invite) becomes one event
	// with both names.
	byKey := map[string]int{}
	// One calendar failing (a typo, Apple being slow) shouldn't blank the
	// others, so failures are logged and skipped. Only if every calendar
	// fails is it an error.
	var failures []error
	tried := 0
	for _, p := range ppl {
		resp.People = append(resp.People, p.Name)
		for _, u := range p.ICSURLs {
			tried++
			evs, err := fetchICS(ctx, u, start, end)
			if err != nil {
				log.Printf("calendar for %q: %v", p.Name, err)
				failures = append(failures, err)
				continue
			}
			for _, e := range evs {
				key := e.Title + "|" + e.Start.Format(time.RFC3339)
				if i, ok := byKey[key]; ok {
					if !contains(resp.Events[i].Who, p.Name) {
						resp.Events[i].Who = append(resp.Events[i].Who, p.Name)
					}
					continue
				}
				e.Who = []string{p.Name}
				byKey[key] = len(resp.Events)
				resp.Events = append(resp.Events, e)
			}
		}
	}

	if tried > 0 && len(failures) == tried {
		return CalendarResponse{}, failures[0]
	}

	// Today's finished events are kept on purpose: the terrain draws the
	// whole day. Widgets that only want what's ahead filter on their own.
	events := resp.Events
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Start.Equal(events[j].Start) {
			return events[i].AllDay && !events[j].AllDay
		}
		return events[i].Start.Before(events[j].Start)
	})
	if cfg.MaxEvents > 0 && len(events) > cfg.MaxEvents {
		events = events[:cfg.MaxEvents]
	}
	resp.Events = events
	return resp, nil
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func fetchICS(ctx context.Context, u string, start, end time.Time) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		// The raw error includes the full URL, which is secret. Keep the host.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("calendar: %s: %w", hostOf(u), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar: %s returned %s", hostOf(u), resp.Status)
	}

	// gocal expands recurring events within [start, end].
	p := gocal.NewParser(resp.Body)
	p.Start, p.End = &start, &end
	// Skip an event gocal can't read instead of rejecting the whole feed.
	p.Strict.Mode = gocal.StrictModeFailEvent
	if err := p.Parse(); err != nil {
		return nil, fmt.Errorf("calendar: parse: %w", err)
	}

	var out []Event
	for _, e := range p.Events {
		if e.Start == nil {
			continue
		}
		ev := Event{Title: e.Summary, Start: e.Start.Local(), Location: e.Location}
		if e.End != nil {
			ev.End = e.End.Local()
		}
		ev.AllDay = isAllDay(ev.Start, ev.End)
		out = append(out, ev)
	}
	return out, nil
}

// isAllDay treats midnight-to-midnight events as all-day.
func isAllDay(start, end time.Time) bool {
	midnight := func(t time.Time) bool { return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 }
	return midnight(start) && !end.IsZero() && midnight(end) && end.Sub(start) >= 24*time.Hour
}

// hostOf names a calendar in logs without printing its secret path.
func hostOf(u string) string {
	if parsed, err := url.Parse(u); err == nil {
		return parsed.Host
	}
	return "calendar"
}

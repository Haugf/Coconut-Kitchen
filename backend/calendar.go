package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/apognu/gocal"
)

type Event struct {
	Title    string    `json:"title"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"allDay"`
	Location string    `json:"location,omitempty"`
}

func calendarHandler(cfg CalendarConfig) http.Handler {
	c := &cached[[]Event]{
		ttl: 5 * time.Minute,
		fetch: func(ctx context.Context) ([]Event, error) {
			return fetchCalendar(ctx, cfg)
		},
	}
	return serveCached(c)
}

func fetchCalendar(ctx context.Context, cfg CalendarConfig) ([]Event, error) {
	if len(cfg.ICSURLs) == 0 {
		return nil, errors.New("calendar: no icsUrls in config")
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, cfg.DaysAhead)

	var events []Event
	for _, u := range cfg.ICSURLs {
		evs, err := fetchICS(ctx, u, start, end)
		if err != nil {
			return nil, err
		}
		events = append(events, evs...)
	}

	// Today's finished events are kept on purpose: the terrain draws the
	// whole day. Widgets that only want what's ahead filter on their own.

	sort.Slice(events, func(i, j int) bool {
		if events[i].Start.Equal(events[j].Start) {
			return events[i].AllDay && !events[j].AllDay
		}
		return events[i].Start.Before(events[j].Start)
	})
	if cfg.MaxEvents > 0 && len(events) > cfg.MaxEvents {
		events = events[:cfg.MaxEvents]
	}
	if events == nil {
		events = []Event{}
	}
	return events, nil
}

func fetchICS(ctx context.Context, u string, start, end time.Time) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calendar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar: upstream returned %s", resp.Status)
	}

	// gocal expands recurring events within [start, end].
	p := gocal.NewParser(resp.Body)
	p.Start, p.End = &start, &end
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

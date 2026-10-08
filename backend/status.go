package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// The page checks in every minute so the Pi can tell whether the screen
// is actually showing the mirror, and which build.
type heartbeat struct {
	At    time.Time      `json:"at"`
	Build string         `json:"build"`
	Seen  map[string]any `json:"seen,omitempty"`
}

type statusBoard struct {
	started  time.Time
	calendar *cached[CalendarResponse]
	weather  *cached[Weather]
	transit  *transit

	mu   sync.Mutex
	beat heartbeat
}

func (s *statusBoard) heartbeat(w http.ResponseWriter, r *http.Request) {
	var hb heartbeat
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&hb); err != nil {
		http.Error(w, "bad heartbeat", http.StatusBadRequest)
		return
	}
	hb.At = time.Now()
	s.mu.Lock()
	s.beat = hb
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// status is safe to publish: counts and error kinds only, never event
// titles or calendar addresses.
func (s *statusBoard) status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	out := map[string]any{"now": time.Now(), "serverStarted": s.started}

	if b, err := os.ReadFile(".installed"); err == nil {
		out["installed"] = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("web/index.html"); err == nil {
		out["serverBuild"] = scriptSrc(string(b))
	}

	s.mu.Lock()
	beat := s.beat
	s.mu.Unlock()
	page := map[string]any{"lastSeen": nil}
	if !beat.At.IsZero() {
		page["lastSeen"] = beat.At
		page["secondsAgo"] = int(time.Since(beat.At).Seconds())
		page["build"] = beat.Build
		page["seen"] = beat.Seen
	}
	out["page"] = page

	cal, calErr := s.calendar.get(ctx)
	calOut := map[string]any{"health": s.calendar.health()}
	if calErr == nil {
		counts := map[string]int{}
		for _, p := range cal.People {
			counts[p] = 0
		}
		for _, e := range cal.Events {
			for _, who := range e.Who {
				counts[who]++
			}
		}
		calOut["eventsThisWeek"] = counts
		calOut["failed"] = cal.Failed
	}
	out["calendar"] = calOut

	_, _ = s.weather.get(ctx)
	out["weather"] = s.weather.health()

	tr := s.transit.snapshot(ctx)
	rows := []map[string]any{}
	for _, row := range tr.Rows {
		rows = append(rows, map[string]any{"route": row.Route, "label": row.Label, "ok": row.OK, "minutes": row.Minutes, "error": row.Error})
	}
	out["transit"] = rows

	writeJSON(w, http.StatusOK, out)
}

func scriptSrc(html string) string {
	i := strings.Index(html, `type="module"`)
	if i < 0 {
		return ""
	}
	rest := html[i:]
	j := strings.Index(rest, `src="`)
	if j < 0 {
		return ""
	}
	rest = rest[j+5:]
	if k := strings.Index(rest, `"`); k >= 0 {
		return rest[:k]
	}
	return ""
}

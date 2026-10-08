package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type TransitConfig struct {
	Subway []SubwayStop `json:"subway"`
	Bus    []BusStop    `json:"bus"`
	// Free key from https://register.developer.obanyc.com, needed for buses only.
	BusAPIKey string `json:"busApiKey"`
}

type SubwayStop struct {
	Route string `json:"route"`
	// GTFS stop ID plus direction: "L17N" is Myrtle-Wyckoff toward Manhattan.
	Stop     string `json:"stop"`
	Label    string `json:"label"`
	StopName string `json:"stopName"`
}

type BusStop struct {
	Route string `json:"route"`
	// The 6-digit code on the bus stop sign (or on bustime.mta.info).
	StopCode string `json:"stopCode"`
	// Optional; otherwise the destination from the feed.
	Label    string `json:"label"`
	StopName string `json:"stopName"`
}

func defaultTransit() TransitConfig {
	return TransitConfig{
		Subway: []SubwayStop{
			{Route: "L", Stop: "L17N", Label: "Manhattan", StopName: "Myrtle–Wyckoff"},
			{Route: "L", Stop: "L17S", Label: "Canarsie", StopName: "Myrtle–Wyckoff"},
			{Route: "M", Stop: "M05N", Label: "Manhattan", StopName: "Forest Av"},
		},
	}
}

type TransitRow struct {
	Kind     string `json:"kind"` // "subway" or "bus"
	Route    string `json:"route"`
	Label    string `json:"label"`
	StopName string `json:"stopName"`
	// Whole minutes until each of the next few arrivals.
	Minutes []int `json:"minutes"`
	OK      bool  `json:"ok"`
}

type TransitResponse struct {
	Updated time.Time    `json:"updated"`
	Rows    []TransitRow `json:"rows"`
}

const feedBase = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/"

// feedFor maps a subway route to its GTFS-realtime feed.
func feedFor(route string) string {
	switch strings.ToUpper(route) {
	case "A", "C", "E", "H", "FS":
		return "nyct%2Fgtfs-ace"
	case "B", "D", "F", "M":
		return "nyct%2Fgtfs-bdfm"
	case "G":
		return "nyct%2Fgtfs-g"
	case "J", "Z":
		return "nyct%2Fgtfs-jz"
	case "N", "Q", "R", "W":
		return "nyct%2Fgtfs-nqrw"
	case "L":
		return "nyct%2Fgtfs-l"
	case "SI", "SIR":
		return "nyct%2Fgtfs-si"
	default:
		return "nyct%2Fgtfs"
	}
}

func transitHandler(cfg TransitConfig) http.Handler {
	t := &transit{cfg: cfg, feeds: map[string]*cached[[]rtTrip]{}, buses: map[string]*cached[[]busArrival]{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, t.snapshot(r.Context()))
	})
}

type transit struct {
	cfg   TransitConfig
	mu    sync.Mutex
	feeds map[string]*cached[[]rtTrip]
	buses map[string]*cached[[]busArrival]
}

func (t *transit) feed(name string) *cached[[]rtTrip] {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.feeds[name]; ok {
		return c
	}
	c := &cached[[]rtTrip]{ttl: 30 * time.Second, fetch: func(ctx context.Context) ([]rtTrip, error) {
		return fetchSubwayFeed(ctx, feedBase+name)
	}}
	t.feeds[name] = c
	return c
}

func (t *transit) bus(code, route string) *cached[[]busArrival] {
	key := code + "|" + route
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.buses[key]; ok {
		return c
	}
	c := &cached[[]busArrival]{ttl: 30 * time.Second, fetch: func(ctx context.Context) ([]busArrival, error) {
		return fetchBusStop(ctx, t.cfg.BusAPIKey, code, route)
	}}
	t.buses[key] = c
	return c
}

func (t *transit) snapshot(ctx context.Context) TransitResponse {
	now := time.Now()
	resp := TransitResponse{Updated: now, Rows: []TransitRow{}}

	for _, s := range t.cfg.Subway {
		row := TransitRow{Kind: "subway", Route: s.Route, Label: s.Label, StopName: s.StopName, Minutes: []int{}}
		trips, err := t.feed(feedFor(s.Route)).get(ctx)
		if err == nil {
			row.OK = true
			var times []int64
			for _, tr := range trips {
				if !strings.EqualFold(tr.Route, s.Route) {
					continue
				}
				for _, st := range tr.Stops {
					if st.ID == s.Stop {
						times = append(times, st.Time)
					}
				}
			}
			row.Minutes = nextMinutes(times, now)
		}
		resp.Rows = append(resp.Rows, row)
	}

	if t.cfg.BusAPIKey != "" {
		for _, s := range t.cfg.Bus {
			if s.StopCode == "" {
				continue
			}
			row := TransitRow{Kind: "bus", Route: s.Route, Label: s.Label, StopName: s.StopName, Minutes: []int{}}
			arrivals, err := t.bus(s.StopCode, s.Route).get(ctx)
			if err == nil {
				row.OK = true
				var times []int64
				for _, a := range arrivals {
					times = append(times, a.At.Unix())
					if row.Label == "" && a.Destination != "" {
						row.Label = a.Destination
					}
				}
				row.Minutes = nextMinutes(times, now)
			}
			resp.Rows = append(resp.Rows, row)
		}
	}
	return resp
}

// nextMinutes turns arrival times into the next three, in whole minutes,
// within the coming hour. A train due within the next half minute is 0.
func nextMinutes(times []int64, now time.Time) []int {
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	out := []int{}
	for _, t := range times {
		secs := t - now.Unix()
		if secs < -30 || secs > 3600 {
			continue
		}
		m := int((secs + 30) / 60)
		if m < 0 {
			m = 0
		}
		out = append(out, m)
		if len(out) == 3 {
			break
		}
	}
	return out
}

func fetchSubwayFeed(ctx context.Context, u string) ([]rtTrip, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("subway feed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subway feed: %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("subway feed: %w", err)
	}
	return parseFeed(b)
}

type busArrival struct {
	At          time.Time
	Destination string
}

// fetchBusStop asks MTA Bus Time (SIRI StopMonitoring) for upcoming buses.
func fetchBusStop(ctx context.Context, key, code, route string) ([]busArrival, error) {
	q := url.Values{}
	q.Set("key", key)
	q.Set("version", "2")
	q.Set("OperatorRef", "MTA")
	q.Set("MonitoringRef", code)
	if route != "" {
		q.Set("LineRef", "MTA NYCT_"+strings.ToUpper(route))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://bustime.mta.info/api/siri/stop-monitoring.json?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bus time: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bus time: %s", resp.Status)
	}
	var body struct {
		Siri struct {
			ServiceDelivery struct {
				StopMonitoringDelivery []struct {
					MonitoredStopVisit []struct {
						MonitoredVehicleJourney struct {
							DestinationName json.RawMessage
							MonitoredCall   struct {
								ExpectedArrivalTime string
								AimedArrivalTime    string
							}
						}
					}
				}
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("bus time: decode: %w", err)
	}
	var out []busArrival
	for _, d := range body.Siri.ServiceDelivery.StopMonitoringDelivery {
		for _, v := range d.MonitoredStopVisit {
			j := v.MonitoredVehicleJourney
			ts := j.MonitoredCall.ExpectedArrivalTime
			if ts == "" {
				ts = j.MonitoredCall.AimedArrivalTime
			}
			at, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				continue
			}
			out = append(out, busArrival{At: at, Destination: titleCase(firstString(j.DestinationName))})
		}
	}
	return out, nil
}

// firstString reads a SIRI name that may be "X" or ["X"].
func firstString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil && len(list) > 0 {
		return list[0]
	}
	return ""
}

// titleCase turns "WYCKOFF HOSP" into "Wyckoff Hosp".
func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

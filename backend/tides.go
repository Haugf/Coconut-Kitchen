package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// TidesConfig picks the NOAA tide station drawn behind the day. An empty
// station turns the tide off.
type TidesConfig struct {
	Station string `json:"station"`
	Place   string `json:"place"`
}

func defaultTides() TidesConfig {
	return TidesConfig{Station: "8518750", Place: "The Battery"}
}

// TidePoint is one predicted water level. Type is "H" or "L" for the
// day's turns and empty for the half-hourly curve.
type TidePoint struct {
	T    time.Time `json:"t"`
	V    float64   `json:"v"`
	Type string    `json:"type,omitempty"`
}

type Tides struct {
	Place  string      `json:"place"`
	Points []TidePoint `json:"points"`
	Turns  []TidePoint `json:"turns"`
}

// noaaBase is a variable so tests can point it at a local server.
var noaaBase = "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter"

func newTidesSource(cfg TidesConfig) *cached[Tides] {
	return &cached[Tides]{
		// Predictions don't change; an hour keeps "today" right after midnight.
		ttl: time.Hour,
		fetch: func(ctx context.Context) (Tides, error) {
			return fetchTides(ctx, cfg, time.Now())
		},
	}
}

// fetchTides reads today and tomorrow, so the curve can run to midnight.
func fetchTides(ctx context.Context, cfg TidesConfig, now time.Time) (Tides, error) {
	out := Tides{Place: cfg.Place, Points: []TidePoint{}, Turns: []TidePoint{}}
	if cfg.Station == "" {
		return out, nil
	}
	from := now.Format("20060102")
	to := now.AddDate(0, 0, 1).Format("20060102")
	points, err := noaaPredictions(ctx, cfg.Station, from, to, "30")
	if err != nil {
		return Tides{}, err
	}
	turns, err := noaaPredictions(ctx, cfg.Station, from, to, "hilo")
	if err != nil {
		return Tides{}, err
	}
	out.Points, out.Turns = points, turns
	return out, nil
}

func noaaPredictions(ctx context.Context, station, from, to, interval string) ([]TidePoint, error) {
	q := url.Values{
		"product":     {"predictions"},
		"application": {"coconut-kitchen"},
		"begin_date":  {from},
		"end_date":    {to},
		"datum":       {"MLLW"},
		"station":     {station},
		"time_zone":   {"lst_ldt"},
		"units":       {"english"},
		"interval":    {interval},
		"format":      {"json"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, noaaBase+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("tides: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tides: NOAA returned %s", resp.Status)
	}
	var body struct {
		Predictions []struct {
			T    string `json:"t"`
			V    string `json:"v"`
			Type string `json:"type"`
		} `json:"predictions"`
		// NOAA answers 200 with an error object for a bad station or date.
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&body); err != nil {
		return nil, fmt.Errorf("tides: %w", err)
	}
	if body.Error != nil {
		return nil, fmt.Errorf("tides: %s", body.Error.Message)
	}
	out := make([]TidePoint, 0, len(body.Predictions))
	for _, p := range body.Predictions {
		// Station local time; the mirror shares the station's time zone.
		t, err := time.ParseInLocation("2006-01-02 15:04", p.T, time.Local)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(p.V, 64)
		if err != nil {
			continue
		}
		out = append(out, TidePoint{T: t, V: v, Type: p.Type})
	}
	if len(out) == 0 {
		return nil, errors.New("tides: NOAA returned no predictions")
	}
	return out, nil
}

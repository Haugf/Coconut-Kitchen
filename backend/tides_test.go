package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchTides(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("station") != "8518750" {
			t.Errorf("station = %q", r.URL.Query().Get("station"))
		}
		if r.URL.Query().Get("interval") == "hilo" {
			w.Write([]byte(`{"predictions":[{"t":"2026-10-11 09:11","v":"5.794","type":"H"},{"t":"2026-10-11 15:40","v":"0.08","type":"L"}]}`))
			return
		}
		w.Write([]byte(`{"predictions":[{"t":"2026-10-11 09:00","v":"5.781"},{"t":"2026-10-11 09:30","v":"5.76"},{"t":"bad","v":"1"}]}`))
	}))
	defer srv.Close()
	old := noaaBase
	noaaBase = srv.URL
	defer func() { noaaBase = old }()

	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.Local)
	got, err := fetchTides(context.Background(), defaultTides(), now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Place != "The Battery" || len(got.Points) != 2 || len(got.Turns) != 2 {
		t.Fatalf("got place %q, %d points, %d turns; want The Battery, 2, 2", got.Place, len(got.Points), len(got.Turns))
	}
	if got.Turns[0].Type != "H" || got.Turns[0].V != 5.794 || got.Turns[0].T.Hour() != 9 || got.Turns[0].T.Minute() != 11 {
		t.Fatalf("first turn = %+v", got.Turns[0])
	}
}

func TestFetchTidesNOAAError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"error":{"message":"No Predictions data was found."}}`))
	}))
	defer srv.Close()
	old := noaaBase
	noaaBase = srv.URL
	defer func() { noaaBase = old }()

	if _, err := fetchTides(context.Background(), defaultTides(), time.Now()); err == nil {
		t.Fatal("want an error for NOAA's error object")
	}
}

func TestFetchTidesOff(t *testing.T) {
	got, err := fetchTides(context.Background(), TidesConfig{}, time.Now())
	if err != nil || len(got.Points) != 0 {
		t.Fatalf("an empty station should turn tides off quietly, got %v %+v", err, got)
	}
}

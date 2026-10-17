package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCurrentKeepsMessageUntilItExpires(t *testing.T) {
	h := newHub("")
	if _, ok := h.current(time.Now()); ok {
		t.Fatal("empty hub should have no current message")
	}
	sent := time.Date(2026, 10, 11, 15, 0, 0, 0, time.UTC)
	h.last = Message{Text: "Laundry is done", TTL: 600, At: sent}

	m, ok := h.current(sent.Add(4 * time.Minute))
	if !ok || m.TTL != 360 {
		t.Fatalf("4 minutes in: got ok=%v ttl=%d, want true 360", ok, m.TTL)
	}
	if _, ok := h.current(sent.Add(11 * time.Minute)); ok {
		t.Fatal("expired message should not be replayed")
	}
}

func TestCurrentUsesDefaultTTL(t *testing.T) {
	h := newHub("")
	sent := time.Now()
	h.last = Message{Text: "Hi", At: sent}
	if _, ok := h.current(sent.Add(10 * time.Second)); !ok {
		t.Fatal("message without ttl should last the default 20s")
	}
	if _, ok := h.current(sent.Add(25 * time.Second)); ok {
		t.Fatal("message without ttl should expire after 20s")
	}
}

func TestFromLoopback(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:5000":    true,
		"[::1]:5000":        true,
		"192.168.1.42:5000": false,
	} {
		r := httptest.NewRequest("POST", "/api/heartbeat", nil)
		r.RemoteAddr = addr
		if got := fromLoopback(r); got != want {
			t.Errorf("%s: got %v, want %v", addr, got, want)
		}
	}
}

func TestCalendarPrivateDuringParty(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	cfg := &Config{Style: "haunted"}
	h := partyPrivate(cfg, ok)
	for addr, want := range map[string]int{"192.168.68.20:5000": http.StatusForbidden, "127.0.0.1:5000": http.StatusOK} {
		r := httptest.NewRequest("GET", "/api/calendar", nil)
		r.RemoteAddr = addr
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != want {
			t.Errorf("%s during the party: %d, want %d", addr, rec.Code, want)
		}
	}
	cfg.Style = "kitchen"
	r := httptest.NewRequest("GET", "/api/calendar", nil)
	r.RemoteAddr = "192.168.68.20:5000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Errorf("kitchen style from the network: %d", rec.Code)
	}
}

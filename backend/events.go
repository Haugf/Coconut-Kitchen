package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Message is anything that should appear on the mirror right now:
// an assistant reply, a doorbell ring, a timer finishing.
type Message struct {
	Text   string    `json:"text"`
	Source string    `json:"source,omitempty"`
	TTL    int       `json:"ttl,omitempty"` // seconds on screen, default 20
	At     time.Time `json:"at"`
}

type hub struct {
	token string
	mu    sync.Mutex
	subs  map[chan Message]struct{}
}

func newHub(token string) *hub {
	return &hub{token: token, subs: make(map[chan Message]struct{})}
}

// stream is the SSE endpoint the frontend listens on.
func (h *hub) stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan Message, 8)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case m := <-ch:
			b, _ := json.Marshal(m)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// publish accepts a message from anything on the network.
//
//	curl -X POST localhost:8080/api/events -d '{"text":"Hello","source":"Test"}'
func (h *hub) publish(w http.ResponseWriter, r *http.Request) {
	if h.token != "" && r.Header.Get("Authorization") != "Bearer "+h.token {
		http.Error(w, "missing or wrong bearer token", http.StatusUnauthorized)
		return
	}
	var m Message
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&m); err != nil {
		http.Error(w, "body must be JSON like {\"text\":\"...\"}", http.StatusBadRequest)
		return
	}
	m.Text = strings.TrimSpace(m.Text)
	if m.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}
	m.At = time.Now()

	h.mu.Lock()
	for ch := range h.subs {
		select {
		case ch <- m:
		default: // slow client; drop rather than block
		}
	}
	h.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

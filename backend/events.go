package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Message is anything that should appear on the mirror right now:
// an assistant reply, a doorbell ring, a timer finishing.
type Message struct {
	Text   string `json:"text"`
	Source string `json:"source,omitempty"`
	TTL    int    `json:"ttl,omitempty"` // seconds on screen, default 20
	// Also say it out loud on the Google speaker, if there is one.
	Speak bool      `json:"speak,omitempty"`
	At    time.Time `json:"at"`
	// Event names a non-message event on the stream (like "summon"), sent
	// as its own SSE event type so the message bar never shows it.
	Event string `json:"-"`
}

type hub struct {
	token string
	mu    sync.Mutex
	subs  map[chan Message]struct{}
	// The most recent message, kept until it expires so a screen that
	// connects late (a phone waking up, a browser restarting) still sees it.
	last Message
	// The Google speaker, for messages sent with "speak": true.
	speaker speaker
}

type speaker interface {
	say(ctx context.Context, text string) error
}

// defaultTTL matches the frontend's default when a message has no ttl.
const defaultTTL = 20

// current returns the latest message with its remaining time on screen,
// or false if there is none or it has expired.
func (h *hub) current(now time.Time) (Message, bool) {
	m := h.last
	if m.Text == "" {
		return Message{}, false
	}
	ttl := m.TTL
	if ttl <= 0 {
		ttl = defaultTTL
	}
	left := int(m.At.Add(time.Duration(ttl) * time.Second).Sub(now).Seconds())
	if left <= 0 {
		return Message{}, false
	}
	m.TTL = left
	return m, true
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
	if m, ok := h.current(time.Now()); ok {
		ch <- m
	}
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
			if m.Event != "" {
				fmt.Fprintf(w, "event: %s\n", m.Event)
			}
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
	h.send(m)
	w.WriteHeader(http.StatusAccepted)
}

// send shows a message on every screen and keeps it for late arrivals.
func (h *hub) send(m Message) {
	if m.Speak && h.speaker != nil {
		go func(sp speaker, text string) {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			if err := sp.say(ctx, text); err != nil {
				log.Printf("speaker: couldn't say the message: %v", err)
			}
		}(h.speaker, spoken(m))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.last = m
	h.broadcast(m)
}

// event sends something to every screen that's listening now, without
// keeping it: a summoned visitor shouldn't replay on a page that loads later.
func (h *hub) event(name, text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcast(Message{Event: name, Text: text, At: time.Now()})
}

// broadcast needs h.mu held.
func (h *hub) broadcast(m Message) {
	for ch := range h.subs {
		select {
		case ch <- m:
		default: // slow client; drop rather than block
		}
	}
}

// spoken is how a message sounds: "Ally says: dinner's ready".
func spoken(m Message) string {
	if src := strings.TrimSpace(m.Source); src != "" {
		return src + " says: " + m.Text
	}
	return m.Text
}

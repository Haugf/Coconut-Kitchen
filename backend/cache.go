package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// cached wraps an upstream fetch with a TTL. If a refresh fails it keeps
// serving the last good value, so the mirror rides out network blips.
type cached[T any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	fetch func(context.Context) (T, error)
	val   T
	at    time.Time
	ok    bool
}

func (c *cached[T]) get(ctx context.Context) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ok && time.Since(c.at) < c.ttl {
		return c.val, nil
	}
	v, err := c.fetch(ctx)
	if err != nil {
		if c.ok {
			return c.val, nil
		}
		return v, err
	}
	c.val, c.at, c.ok = v, time.Now(), true
	return v, nil
}

func serveCached[T any](c *cached[T]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, err := c.get(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, v)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

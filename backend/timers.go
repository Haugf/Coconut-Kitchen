package main

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Google's own timers ("Hey Google, set a pasta timer"), read from the
// speaker's local API on port 8443. The speaker only answers with a local
// token that Google hands out to the account, so a small Python helper
// (glocaltokens, installed in ~/mirror/venv by `mirror google`) fetches
// one with the sign-in saved in ~/mirror/google.json. Both are unofficial;
// Google can change them.

// The speaker's local API; a variable so tests can point it elsewhere.
var timersPort = "8443"

// The Python that has glocaltokens (installed by `mirror google`).
var googlePython = "venv/bin/python"

//go:embed google_tokens.py
var googleTokensScript []byte

const (
	googleAuthFile = "google.json"
	staleTimers    = 30 * time.Second
)

// Timer is one of the speaker's timers as /api/timers serves it.
type Timer struct {
	Label    string    `json:"label,omitempty"`
	State    string    `json:"state"` // running, paused, ringing
	FireAt   time.Time `json:"fireAt,omitempty"`
	Duration float64   `json:"duration"`       // seconds, as set
	Left     float64   `json:"left,omitempty"` // seconds, while paused
}

type TimersResponse struct {
	Enabled bool      `json:"enabled"`
	Timers  []Timer   `json:"timers"`
	At      time.Time `json:"at"`
}

type googleTimers struct {
	speaker  *caster
	authFile string
	every    time.Duration
	client   *http.Client
	// fetchToken asks Google for the speaker's local token; tests replace it.
	fetchToken func(ctx context.Context, speaker string) (string, error)

	mu        sync.Mutex
	token     string
	tokenTry  time.Time
	timers    []Timer
	lastFire  map[string]time.Time
	pausedAt  map[string]float64
	ok        bool
	lastOK    time.Time
	lastErr   error
	errAt     time.Time
	tokenWait time.Duration
}

func newGoogleTimers(speaker *caster) *googleTimers {
	g := &googleTimers{
		speaker:   speaker,
		authFile:  googleAuthFile,
		every:     5 * time.Second,
		tokenWait: 5 * time.Minute,
		client: &http.Client{
			Timeout: 8 * time.Second,
			// The speaker's certificate is its own; the request stays on
			// the home network.
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
		},
		lastFire: map[string]time.Time{},
		pausedAt: map[string]float64{},
	}
	g.fetchToken = g.tokenFromHelper
	return g
}

func (g *googleTimers) enabled() bool {
	_, err := os.Stat(g.authFile)
	return err == nil
}

func (g *googleTimers) run(ctx context.Context) {
	t := time.NewTicker(g.every)
	defer t.Stop()
	for {
		if g.enabled() {
			err := g.poll(ctx)
			g.mu.Lock()
			if err != nil {
				if g.lastErr == nil || g.lastErr.Error() != err.Error() {
					log.Printf("timers: %v", err)
				}
				g.ok, g.lastErr, g.errAt = false, err, time.Now()
			} else {
				if g.lastErr != nil {
					log.Printf("timers: reading the speaker's timers again")
				}
				g.ok, g.lastErr, g.lastOK = true, nil, time.Now()
			}
			g.mu.Unlock()
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

var errNotConnected = errors.New("waiting for the speaker")

// poll reads the timers once, getting a fresh token when the speaker
// turns the old one away.
func (g *googleTimers) poll(ctx context.Context) error {
	name, host, ok := g.speaker.address()
	if !ok {
		return errNotConnected
	}
	for attempt := 0; attempt < 2; attempt++ {
		token, err := g.currentToken(ctx, name, attempt > 0)
		if err != nil {
			return err
		}
		body, status, err := g.get(ctx, host, token)
		if err != nil {
			return err
		}
		switch status {
		case http.StatusOK:
			return g.apply(body, time.Now())
		case http.StatusUnauthorized, http.StatusForbidden:
			g.mu.Lock()
			g.token = ""
			g.mu.Unlock()
			continue
		case http.StatusNotFound:
			return errors.New("this speaker doesn't keep timers")
		default:
			return fmt.Errorf("the speaker answered %d", status)
		}
	}
	return errors.New("the speaker turned down a fresh token; run `mirror google` again")
}

func (g *googleTimers) get(ctx context.Context, host, token string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+net.JoinHostPort(host, timersPort)+"/setup/assistant/alarms", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("cast-local-authorization-token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("asking the speaker for timers: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return b, resp.StatusCode, err
}

// currentToken returns the token in hand, or asks Google for one, at most
// every few minutes so a broken sign-in doesn't hammer Google.
func (g *googleTimers) currentToken(ctx context.Context, speaker string, fresh bool) (string, error) {
	g.mu.Lock()
	if g.token != "" && !fresh {
		t := g.token
		g.mu.Unlock()
		return t, nil
	}
	if !g.tokenTry.IsZero() && time.Since(g.tokenTry) < g.tokenWait {
		g.mu.Unlock()
		return "", errors.New("waiting a few minutes before asking Google for a new token")
	}
	g.tokenTry = time.Now()
	g.mu.Unlock()

	tctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	t, err := g.fetchToken(tctx, speaker)
	if err != nil {
		return "", err
	}
	g.mu.Lock()
	g.token = t
	g.mu.Unlock()
	return t, nil
}

// tokenFromHelper runs google_tokens.py and picks the speaker's token.
func (g *googleTimers) tokenFromHelper(ctx context.Context, speaker string) (string, error) {
	if _, err := os.Stat(googlePython); err != nil {
		return "", errors.New("the Google helper isn't installed; run `mirror google` on the Pi")
	}
	f, err := os.CreateTemp("", "google_tokens-*.py")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(googleTokensScript); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, googlePython, f.Name(), g.authFile)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("getting a token from Google: %s", lastLine(hideGoogleTokens(stderr.String()), err))
	}
	var devices []tokenDevice
	if err := json.Unmarshal(stdout.Bytes(), &devices); err != nil {
		return "", errors.New("getting a token from Google: the helper's answer wasn't readable")
	}
	if len(devices) == 0 {
		// glocaltokens prints why (a bad sign-in, usually) and returns none.
		return "", fmt.Errorf("Google returned no speakers: %s", lastLine(hideGoogleTokens(stderr.String()), errors.New("no reason given")))
	}
	return pickToken(devices, speaker)
}

type tokenDevice struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

func pickToken(devices []tokenDevice, speaker string) (string, error) {
	names := []string{}
	for _, d := range devices {
		if strings.EqualFold(strings.TrimSpace(d.Name), strings.TrimSpace(speaker)) {
			return d.Token, nil
		}
		names = append(names, fmt.Sprintf("%q", d.Name))
	}
	return "", fmt.Errorf("Google has no token for %q (it knows %s)", speaker, strings.Join(names, ", "))
}

var secretRe = regexp.MustCompile(`(aas_et/|oauth2[a-z0-9_]*/|ya29\.)[A-Za-z0-9._~+/=-]+`)

// hideGoogleTokens keeps Google tokens out of logs and /api/status.
func hideGoogleTokens(s string) string { return secretRe.ReplaceAllString(s, "[hidden]") }

func lastLine(s string, fallback error) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if l := strings.TrimSpace(lines[len(lines)-1]); l != "" {
		if len(l) > 200 {
			l = l[:200]
		}
		return l
	}
	return fallback.Error()
}

type rawTimer struct {
	ID               string `json:"id"`
	FireTime         *int64 `json:"fire_time"`
	OriginalDuration int64  `json:"original_duration"`
	Status           int    `json:"status"`
	Label            string `json:"label"`
}

// apply turns the speaker's answer into timers. It leaves out a paused
// timer's time left, so that's worked out from when it was last running.
func (g *googleTimers) apply(body []byte, now time.Time) error {
	var resp struct {
		Timer []rawTimer `json:"timer"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return errors.New("the speaker's timers weren't readable")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	seen := map[string]bool{}
	out := []Timer{}
	for _, r := range resp.Timer {
		seen[r.ID] = true
		t := Timer{Label: strings.TrimSpace(r.Label), Duration: float64(r.OriginalDuration) / 1000}
		var fire time.Time
		if r.FireTime != nil {
			fire = time.UnixMilli(*r.FireTime)
		}
		switch r.Status {
		case 1:
			t.State, t.FireAt = "running", fire
			g.lastFire[r.ID] = fire
			delete(g.pausedAt, r.ID)
		case 2:
			t.State = "paused"
			if _, ok := g.pausedAt[r.ID]; !ok {
				left := 0.0
				if f, ok := g.lastFire[r.ID]; ok && f.After(now) {
					left = f.Sub(now).Seconds()
				}
				g.pausedAt[r.ID] = left
			}
			t.Left = g.pausedAt[r.ID]
		case 3:
			t.State, t.FireAt = "ringing", fire
		default:
			continue
		}
		out = append(out, t)
	}
	for id := range g.lastFire {
		if !seen[id] {
			delete(g.lastFire, id)
			delete(g.pausedAt, id)
		}
	}
	// Ringing first, then the one that ends soonest.
	rank := map[string]int{"ringing": 0, "running": 1, "paused": 2}
	sort.SliceStable(out, func(i, j int) bool {
		if rank[out[i].State] != rank[out[j].State] {
			return rank[out[i].State] < rank[out[j].State]
		}
		return out[i].FireAt.Before(out[j].FireAt)
	})
	g.timers = out
	return nil
}

func (g *googleTimers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	timers := []Timer{}
	// A missed check or two is fine; after that, a countdown we can no
	// longer confirm comes off the screen.
	if time.Since(g.lastOK) < staleTimers {
		timers = append(timers, g.timers...)
	}
	g.mu.Unlock()
	writeJSON(w, http.StatusOK, TimersResponse{Enabled: g.enabled(), Timers: timers, At: time.Now()})
}

// health is for /api/status: whether timers are being read and how many
// are set, never their labels.
func (g *googleTimers) health() map[string]any {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := map[string]any{"enabled": g.enabled()}
	if !out["enabled"].(bool) {
		return out
	}
	out["ok"] = g.ok
	out["timers"] = len(g.timers)
	if !g.lastOK.IsZero() {
		out["lastOk"] = g.lastOK
	}
	if g.lastErr != nil {
		out["error"], out["errorAt"] = hideGoogleTokens(g.lastErr.Error()), g.errAt
	}
	return out
}

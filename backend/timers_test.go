package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeAlarmsAPI answers like the speaker's port 8443, accepting only the
// token "good".
func fakeAlarmsAPI(t *testing.T, body func() string) (host, port string) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/setup/assistant/alarms" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("cast-local-authorization-token") != "good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(body()))
	}))
	t.Cleanup(srv.Close)
	h, p, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	return h, p
}

func timersFor(t *testing.T, host string) (*googleTimers, *atomic.Int32) {
	dir := t.TempDir()
	auth := filepath.Join(dir, "google.json")
	if err := os.WriteFile(auth, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c := newCaster(CastConfig{})
	c.device, c.connected = castDevice{Name: "Kitchen speaker", Addr: net.JoinHostPort(host, "8009")}, true
	g := newGoogleTimers(c)
	g.authFile = auth
	calls := &atomic.Int32{}
	tokens := []string{"stale", "good"}
	g.fetchToken = func(ctx context.Context, speaker string) (string, error) {
		if speaker != "Kitchen speaker" {
			t.Errorf("token asked for %q", speaker)
		}
		n := calls.Add(1)
		return tokens[min(int(n)-1, len(tokens)-1)], nil
	}
	g.tokenWait = 0
	return g, calls
}

func TestTimersRefreshTokenAndRead(t *testing.T) {
	fire := time.Now().Add(9 * time.Minute).UnixMilli()
	host, port := fakeAlarmsAPI(t, func() string {
		return `{"alarm":[],"timer":[{"id":"t1","fire_time":` + jsonInt(fire) + `,"original_duration":540000,"status":1,"label":"pasta"}]}`
	})
	defer swapPort(port)()
	g, calls := timersFor(t, host)

	if err := g.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	// The first token was turned away, so it asked for a second.
	if calls.Load() != 2 {
		t.Fatalf("token fetched %d times", calls.Load())
	}
	g.lastOK = time.Now()
	rec := httptest.NewRecorder()
	g.ServeHTTP(rec, httptest.NewRequest("GET", "/api/timers", nil))
	var resp TimersResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Enabled || len(resp.Timers) != 1 {
		t.Fatalf("resp = %+v", resp)
	}
	tm := resp.Timers[0]
	if tm.Label != "pasta" || tm.State != "running" || tm.Duration != 540 || tm.FireAt.UnixMilli() != fire {
		t.Fatalf("timer = %+v", tm)
	}
	// The token is kept: no new fetch on the next poll.
	if err := g.poll(context.Background()); err != nil || calls.Load() != 2 {
		t.Fatalf("second poll: %v, %d fetches", err, calls.Load())
	}
	if h := g.health(); h["timers"] != 1 || h["ok"] != false && h["ok"] != true {
		t.Errorf("health = %v", h)
	}
}

func TestTimerStates(t *testing.T) {
	g := newGoogleTimers(newCaster(CastConfig{}))
	now := time.Now()
	fire := now.Add(5 * time.Minute)
	running := `{"timer":[{"id":"a","fire_time":` + jsonInt(fire.UnixMilli()) + `,"original_duration":600000,"status":1},` +
		`{"id":"b","fire_time":` + jsonInt(now.Add(-time.Second).UnixMilli()) + `,"original_duration":60000,"status":3,"label":"eggs"}]}`
	if err := g.apply([]byte(running), now); err != nil {
		t.Fatal(err)
	}
	if g.timers[0].State != "ringing" || g.timers[0].Label != "eggs" || g.timers[1].State != "running" {
		t.Fatalf("order = %+v", g.timers)
	}
	// Paused: the speaker sends no fire time, so the time left comes from
	// when it was last running.
	paused := `{"timer":[{"id":"a","original_duration":600000,"status":2}]}`
	if err := g.apply([]byte(paused), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(g.timers) != 1 || g.timers[0].State != "paused" || g.timers[0].Left < 239 || g.timers[0].Left > 241 {
		t.Fatalf("paused = %+v", g.timers)
	}
	// Still paused a minute later: the time left doesn't move.
	_ = g.apply([]byte(paused), now.Add(2*time.Minute))
	if l := g.timers[0].Left; l < 239 || l > 241 {
		t.Fatalf("left drifted to %v", l)
	}
	_ = g.apply([]byte(`{"timer":[]}`), now)
	if len(g.timers) != 0 || len(g.lastFire) != 0 {
		t.Fatalf("cleared = %+v %v", g.timers, g.lastFire)
	}
}

func TestStaleTimersComeOff(t *testing.T) {
	g := newGoogleTimers(newCaster(CastConfig{}))
	g.authFile = filepath.Join(t.TempDir(), "none.json")
	g.timers = []Timer{{State: "running"}}
	g.lastOK = time.Now().Add(-time.Minute)
	rec := httptest.NewRecorder()
	g.ServeHTTP(rec, httptest.NewRequest("GET", "/api/timers", nil))
	if !strings.Contains(rec.Body.String(), `"timers":[]`) || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestPickTokenAndHideTokens(t *testing.T) {
	devs := []tokenDevice{{Name: "Living Room TV", Token: "x"}, {Name: "Kitchen speaker", Token: "y"}}
	if tok, err := pickToken(devs, "kitchen speaker"); err != nil || tok != "y" {
		t.Fatalf("%q %v", tok, err)
	}
	if _, err := pickToken(devs, "Bedroom"); err == nil || !strings.Contains(err.Error(), `"Living Room TV"`) {
		t.Fatalf("err = %v", err)
	}
	s := hideGoogleTokens("BadAuthentication for aas_et/AKppINabc123-_x and ya29.a0AfB_byC and oauth2_4/0AeaYSHx")
	if strings.Contains(s, "AKpp") || strings.Contains(s, "a0AfB") || strings.Contains(s, "0AeaY") || !strings.Contains(s, "BadAuthentication") {
		t.Fatalf("not hidden: %s", s)
	}
}

func TestTimersNeedSpeaker(t *testing.T) {
	g := newGoogleTimers(newCaster(CastConfig{}))
	if err := g.poll(context.Background()); err != errNotConnected {
		t.Fatalf("err = %v", err)
	}
}

func jsonInt(n int64) string { b, _ := json.Marshal(n); return string(b) }

// swapPort points the timers at the test server's port.
func swapPort(p string) func() {
	old := timersPort
	timersPort = p
	return func() { timersPort = old }
}

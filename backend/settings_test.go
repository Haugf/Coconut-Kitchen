package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testConfig = `{
  "addr": ":8080",
  "eventsToken": "keep-me",
  "calendar": {
    "people": [
      {"name": "Fred", "icsUrls": ["https://calendar.google.com/calendar/ical/SECRET-FRED/basic.ics"]},
      {"name": "Ally", "icsUrls": "webcal://p111-caldav.icloud.com/published/2/SECRET-ALLY"}
    ],
    "daysAhead": 7
  },
  "map": {"radiusMeters": 1300},
  "widgets": [{"id": "transit"}]
}
`

func newTestSettings(t *testing.T) (*settingsAPI, *http.ServeMux, string, *int) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(testConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cal := &cached[CalendarResponse]{fetch: func(context.Context) (CalendarResponse, error) {
		return CalendarResponse{People: []string{"Fred", "Ally"}, Events: []Event{{Title: "x", Who: []string{"Fred"}}}}, nil
	}}
	s := newSettingsAPI(path, cfg, newHub(""), cal)
	restarts := 0
	s.restart = func() { restarts++ }
	mux := http.NewServeMux()
	s.routes(mux)
	return s, mux, path, &restarts
}

func call(mux *http.ServeMux, method, path, body, from string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = from
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestSettingsOnlyFromHomeNetwork(t *testing.T) {
	_, mux, _, _ := newTestSettings(t)
	if w := call(mux, "GET", "/api/settings", "", "8.8.8.8:5000"); w.Code != http.StatusForbidden {
		t.Fatalf("public address got %d, want 403", w.Code)
	}
	if w := call(mux, "GET", "/api/settings", "", "192.168.1.20:5000"); w.Code != http.StatusOK {
		t.Fatalf("home address got %d, want 200", w.Code)
	}
}

func TestSettingsNeverShowSecrets(t *testing.T) {
	_, mux, _, _ := newTestSettings(t)
	w := call(mux, "GET", "/api/settings", "", "127.0.0.1:5000")
	body := w.Body.String()
	if strings.Contains(body, "SECRET") || strings.Contains(body, "keep-me") {
		t.Fatalf("settings leaked a secret: %s", body)
	}
	var got struct {
		People []settingsPerson `json:"people"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.People) != 2 || got.People[1].Links[0].Host != "p111-caldav.icloud.com" || got.People[0].Events != 1 {
		t.Fatalf("people = %+v", got.People)
	}
}

func TestSettingsAddRemoveKeepsOtherSettings(t *testing.T) {
	_, mux, path, _ := newTestSettings(t)
	add := `{"person":"Ally","add":"webcal://p111-caldav.icloud.com/published/2/NEW"}`
	if w := call(mux, "POST", "/api/settings/calendar", add, "10.0.0.5:1"); w.Code != http.StatusOK {
		t.Fatalf("add: %d %s", w.Code, w.Body)
	}
	if w := call(mux, "POST", "/api/settings/calendar", `{"person":"Ally","remove":0}`, "10.0.0.5:1"); w.Code != http.StatusOK {
		t.Fatalf("remove: %d %s", w.Code, w.Body)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	ally := cfg.Calendar.People[1].ICSURLs
	if len(ally) != 1 || !strings.HasSuffix(ally[0], "/NEW") {
		t.Fatalf("Ally's links = %v", ally)
	}
	if cfg.EventsToken != "keep-me" || cfg.Calendar.DaysAhead != 7 || len(cfg.Widgets) != 1 || cfg.Map.RadiusMeters != 1300 {
		t.Fatalf("other settings changed: %+v", cfg)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "SECRET-FRED") {
		t.Fatal("Fred's link was lost")
	}
}

func TestSettingsRejectsBadInput(t *testing.T) {
	_, mux, _, _ := newTestSettings(t)
	for _, c := range []struct{ path, body string }{
		{"/api/settings/calendar", `{"person":"Ally","add":"not a link"}`},
		{"/api/settings/calendar", `{"person":"Nobody","add":"https://example.com/a.ics"}`},
		{"/api/settings/calendar", `{"person":"Ally","remove":5}`},
		{"/api/settings/style", `{"style":"neon"}`},
		{"/api/settings/sleep", `{"from":"25:00","to":"06:30"}`},
		{"/api/settings/say", `{"text":"   "}`},
	} {
		if w := call(mux, "POST", c.path, c.body, "127.0.0.1:1"); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s: got %d, want 400", c.path, c.body, w.Code)
		}
	}
}

func TestSettingsStyleAndSleep(t *testing.T) {
	_, mux, path, _ := newTestSettings(t)
	call(mux, "POST", "/api/settings/style", `{"style":"terrain"}`, "127.0.0.1:1")
	call(mux, "POST", "/api/settings/sleep", `{"from":"","to":""}`, "127.0.0.1:1")
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Style != "terrain" || cfg.Sleep == nil || cfg.Sleep.From != "" {
		t.Fatalf("style %q sleep %+v", cfg.Style, cfg.Sleep)
	}
}

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, folder, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, folder), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, folder, "widget.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceURLRules(t *testing.T) {
	for _, bad := range []string{
		"http://api.example.com/x",
		"https://{settings.host}/x",
		"https://api.example.com@evil.com/x",
		"ftp://x",
	} {
		if checkSourceURL(bad) == nil {
			t.Errorf("%s should be refused", bad)
		}
	}
	if err := checkSourceURL("https://api.example.com/v1?q={settings.city}&key={secrets.key}"); err != nil {
		t.Errorf("good URL refused: %v", err)
	}
	got := fill("https://a.com/x?q={settings.city}&k={secrets.key}", map[string]any{"city": "New York/../evil"}, map[string]string{"key": "s3cret"}, func(s string) string { return strings.ReplaceAll(s, "/", "%2F") })
	if strings.Contains(got, "/../") || !strings.Contains(got, "k=s3cret") {
		t.Errorf("fill: %s", got)
	}
}

func TestManifestValidation(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "plants", `{"id":"plants","name":"Plants"}`)
	writeManifest(t, dir, "_template", `{"id":"example","name":"Example"}`) // skipped
	m, err := loadManifests(dir)
	if err != nil || len(m) != 1 {
		t.Fatalf("got %v, %v", m, err)
	}
	writeManifest(t, dir, "wrong", `{"id":"other","name":"X"}`)
	if _, err := loadManifests(dir); err == nil {
		t.Error("an id that doesn't match its folder should fail")
	}
}

func TestWidgetHostServesSettingsAndProxiesData(t *testing.T) {
	var gotKey string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Key")
		w.Write([]byte(`{"city":"` + r.URL.Query().Get("q") + `"}`))
	}))
	defer upstream.Close()
	old := httpClient
	httpClient = upstream.Client()
	defer func() { httpClient = old }()

	dir := t.TempDir()
	writeManifest(t, dir, "air", `{
		"id": "air", "name": "Air", "width": "fit",
		"settings": {"city": {"default": "Queens"}, "units": {"default": "us"}},
		"secrets": ["key"],
		"data": {
			"now": {"url": "`+upstream.URL+`/v1?q={settings.city}", "headers": {"X-Key": "{secrets.key}"}, "every": "1m"},
			"core": {"url": "/api/transit"}
		}}`)
	h := newWidgetHost(dir, []WidgetConfig{
		{ID: "air", Settings: map[string]any{"city": "Ridgewood"}, Secrets: map[string]string{"key": "s3cret"}},
		{ID: "missing"},
	})

	rec := httptest.NewRecorder()
	h.list(rec, httptest.NewRequest("GET", "/api/widgets", nil))
	var list []WidgetInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list: %s", rec.Body)
	}
	if strings.Contains(rec.Body.String(), "s3cret") {
		t.Fatal("secrets must never reach the page")
	}
	w := list[0]
	if w.Settings["city"] != "Ridgewood" || w.Settings["units"] != "us" || w.Width != "fit" {
		t.Errorf("settings: %+v", w)
	}
	if w.Sources["now"] != "/api/w/air/now" || w.Sources["core"] != "/api/transit" {
		t.Errorf("sources: %+v", w.Sources)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/w/{id}/{source}", h.source)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/w/air/now", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"city":"Ridgewood"`) || gotKey != "s3cret" {
		t.Errorf("proxy: %d %s key=%q", rec.Code, rec.Body, gotKey)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/w/air/nope", nil))
	if rec.Code != 404 {
		t.Errorf("unknown source: %d", rec.Code)
	}
}

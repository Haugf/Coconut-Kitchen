package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Widgets are the pieces people add to the mirror. Each one is a folder
// with a widget.json manifest (see WIDGETS.md). The server reads the
// manifests, merges each widget's settings from config.json, and fetches
// the data sources a manifest declares, so API keys stay on the Pi and
// every source is cached.

type WidgetManifest struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Author      string                   `json:"author"`
	Width       string                   `json:"width"` // "fill" (share the row) or "fit" (its own size)
	Settings    map[string]WidgetSetting `json:"settings"`
	Secrets     []string                 `json:"secrets"`
	Data        map[string]WidgetSource  `json:"data"`
}

type WidgetSetting struct {
	Description string `json:"description"`
	Default     any    `json:"default"`
}

type WidgetSource struct {
	// Either a core endpoint ("/api/transit") or an https URL that may use
	// {settings.name} and {secrets.name}. The scheme and host must be
	// written out in full: settings can't change where a request goes.
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	// How long to keep an answer, like "10m". At least 30s; default 5m.
	Every string `json:"every"`
}

// WidgetConfig is one entry in config.json's "widgets" list. The list's
// order is the order on screen.
type WidgetConfig struct {
	ID       string            `json:"id"`
	Settings map[string]any    `json:"settings"`
	Secrets  map[string]string `json:"secrets"`
}

func defaultWidgets() []WidgetConfig {
	return []WidgetConfig{{ID: "transit"}, {ID: "minimap"}}
}

// What the page gets for each enabled widget. Never includes secrets.
type WidgetInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Width    string            `json:"width"`
	Settings map[string]any    `json:"settings"`
	Sources  map[string]string `json:"sources"` // source name -> path to fetch
}

type widgetHost struct {
	manifests map[string]WidgetManifest
	enabled   []WidgetConfig

	mu    sync.Mutex
	cache map[string]*cached[json.RawMessage]
}

var widgetID = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}$`)

// widgetsDir finds the manifests: ~/mirror/widgets on the Pi (install.sh
// copies them there), the source tree when developing.
func widgetsDir() string {
	for _, d := range []string{"widgets", "../frontend/src/widgets"} {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return "widgets"
}

func loadManifests(dir string) (map[string]WidgetManifest, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*", "widget.json"))
	if err != nil {
		return nil, err
	}
	out := map[string]WidgetManifest{}
	for _, f := range files {
		folder := filepath.Base(filepath.Dir(f))
		if strings.HasPrefix(folder, "_") {
			continue // the template
		}
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var m WidgetManifest
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if err := m.validate(folder); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		out[m.ID] = m
	}
	return out, nil
}

func (m WidgetManifest) validate(folder string) error {
	if !widgetID.MatchString(m.ID) {
		return fmt.Errorf("id %q must be lowercase letters, digits and dashes", m.ID)
	}
	if m.ID != folder {
		return fmt.Errorf("id %q must match its folder %q", m.ID, folder)
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Width != "" && m.Width != "fill" && m.Width != "fit" {
		return fmt.Errorf(`width must be "fill" or "fit"`)
	}
	for name, s := range m.Data {
		if !widgetID.MatchString(name) {
			return fmt.Errorf("data source name %q must be lowercase letters, digits and dashes", name)
		}
		if strings.HasPrefix(s.URL, "/api/") {
			continue
		}
		if err := checkSourceURL(s.URL); err != nil {
			return fmt.Errorf("data source %q: %w", name, err)
		}
		if _, err := sourceTTL(s.Every); err != nil {
			return fmt.Errorf("data source %q: %w", name, err)
		}
	}
	return nil
}

// checkSourceURL insists on a literal https scheme and host, so filling in
// settings can only change the path and query.
func checkSourceURL(raw string) error {
	if !strings.HasPrefix(raw, "https://") {
		return fmt.Errorf("url must start with https:// (or be a core /api/ path)")
	}
	rest := strings.TrimPrefix(raw, "https://")
	host := rest
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		host = rest[:i]
	}
	if host == "" || strings.ContainsAny(host, "{}@") {
		return fmt.Errorf("the host must be written out in full, without {placeholders}")
	}
	return nil
}

func sourceTTL(s string) (time.Duration, error) {
	if s == "" {
		return 5 * time.Minute, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf(`every: use a duration like "30s" or "10m"`)
	}
	if d < 30*time.Second {
		d = 30 * time.Second
	}
	return d, nil
}

func newWidgetHost(dir string, enabled []WidgetConfig) *widgetHost {
	manifests, err := loadManifests(dir)
	if err != nil {
		log.Printf("widgets: %v", err)
	}
	if enabled == nil {
		enabled = defaultWidgets()
	}
	h := &widgetHost{manifests: manifests, cache: map[string]*cached[json.RawMessage]{}}
	for _, w := range enabled {
		if _, ok := manifests[w.ID]; !ok {
			log.Printf("widgets: %q is in config.json but no widget has that id; skipping it", w.ID)
			continue
		}
		h.enabled = append(h.enabled, w)
	}
	return h
}

func (h *widgetHost) settingsFor(w WidgetConfig) map[string]any {
	m := h.manifests[w.ID]
	out := map[string]any{}
	for k, s := range m.Settings {
		out[k] = s.Default
	}
	for k, v := range w.Settings {
		out[k] = v
	}
	return out
}

// list serves GET /api/widgets: the enabled widgets in screen order.
func (h *widgetHost) list(w http.ResponseWriter, r *http.Request) {
	out := []WidgetInfo{}
	for _, cfg := range h.enabled {
		m := h.manifests[cfg.ID]
		info := WidgetInfo{ID: m.ID, Name: m.Name, Width: m.Width, Settings: h.settingsFor(cfg), Sources: map[string]string{}}
		if info.Width == "" {
			info.Width = "fill"
		}
		names := make([]string, 0, len(m.Data))
		for n := range m.Data {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			src := m.Data[n]
			if strings.HasPrefix(src.URL, "/api/") {
				info.Sources[n] = src.URL
			} else {
				info.Sources[n] = "/api/w/" + m.ID + "/" + n
			}
		}
		out = append(out, info)
	}
	writeJSON(w, http.StatusOK, out)
}

// source serves GET /api/w/{id}/{source}: the widget's data, fetched and
// cached here with its secrets filled in.
func (h *widgetHost) source(w http.ResponseWriter, r *http.Request) {
	id, name := r.PathValue("id"), r.PathValue("source")
	var cfg *WidgetConfig
	for i := range h.enabled {
		if h.enabled[i].ID == id {
			cfg = &h.enabled[i]
		}
	}
	if cfg == nil {
		http.Error(w, "no such widget enabled", http.StatusNotFound)
		return
	}
	src, ok := h.manifests[id].Data[name]
	if !ok || strings.HasPrefix(src.URL, "/api/") {
		http.Error(w, "no such data source", http.StatusNotFound)
		return
	}
	c := h.cacheFor(*cfg, name, src)
	v, err := c.get(r.Context())
	if err != nil {
		// Errors can carry the URL; keep secrets out of what we return and log.
		msg := scrubSecrets(err.Error(), cfg.Secrets)
		log.Printf("widget %s/%s: %s", id, name, msg)
		http.Error(w, msg, http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(v)
}

func (h *widgetHost) cacheFor(cfg WidgetConfig, name string, src WidgetSource) *cached[json.RawMessage] {
	key := cfg.ID + "/" + name
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.cache[key]; ok {
		return c
	}
	ttl, _ := sourceTTL(src.Every)
	settings := h.settingsFor(cfg)
	c := &cached[json.RawMessage]{ttl: ttl, fetch: func(ctx context.Context) (json.RawMessage, error) {
		u := fill(src.URL, settings, cfg.Secrets, url.QueryEscape)
		headers := map[string]string{}
		for k, v := range src.Headers {
			headers[k] = fill(v, settings, cfg.Secrets, func(s string) string { return s })
		}
		return fetchWidgetJSON(ctx, u, headers)
	}}
	h.cache[key] = c
	return c
}

var placeholder = regexp.MustCompile(`\{(settings|secrets)\.([A-Za-z0-9_]+)\}`)

// fill swaps {settings.x} and {secrets.y} for their values. In URLs each
// value is escaped, so it can't break out of its spot.
func fill(s string, settings map[string]any, secrets map[string]string, escape func(string) string) string {
	return placeholder.ReplaceAllStringFunc(s, func(m string) string {
		p := placeholder.FindStringSubmatch(m)
		var v string
		if p[1] == "secrets" {
			v = secrets[p[2]]
		} else if x, ok := settings[p[2]]; ok && x != nil {
			v = fmt.Sprint(x)
		}
		return escape(v)
	})
}

func scrubSecrets(s string, secrets map[string]string) string {
	for _, v := range secrets {
		if v != "" {
			s = strings.ReplaceAll(s, v, "SECRET")
			s = strings.ReplaceAll(s, url.QueryEscape(v), "SECRET")
		}
	}
	return s
}

func fetchWidgetJSON(ctx context.Context, u string, headers map[string]string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "coconut-kitchen-mirror/1")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", req.URL.Host, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 2<<20 {
		return nil, fmt.Errorf("%s: answer is over 2 MB", req.URL.Host)
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("%s: answer isn't JSON", req.URL.Host)
	}
	return json.RawMessage(b), nil
}

// health lists each enabled widget's external sources for /api/status.
func (h *widgetHost) health() map[string]any {
	out := map[string]any{}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, cfg := range h.enabled {
		entry := map[string]any{}
		for name, src := range h.manifests[cfg.ID].Data {
			if strings.HasPrefix(src.URL, "/api/") {
				continue
			}
			if c, ok := h.cache[cfg.ID+"/"+name]; ok {
				hl := c.health()
				hl.Error = scrubSecrets(hl.Error, cfg.Secrets)
				entry[name] = hl
			} else {
				entry[name] = "not fetched yet"
			}
		}
		out[cfg.ID] = entry
	}
	return out
}

package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// settingsAPI backs the phone page at /settings: anyone on the home
// network can send a message, fix a calendar link, switch the style or
// the screen. It never hands out a calendar's secret address, only its
// host. Changes are written to config.json, then the server exits and
// systemd starts it again with the new settings (Restart=always).
type settingsAPI struct {
	path     string
	cfg      *Config
	hub      *hub
	calendar *cached[CalendarResponse]
	// The Google speaker, if the mirror found one; nil when it's off.
	speaker *caster
	mu      sync.Mutex
	// restart is os.Exit in production; tests replace it.
	restart func()

	// Wrong PINs: a pause each, and a minute's lockout after a few, so a
	// short PIN can't be guessed by trying them all.
	pinMu       sync.Mutex
	wrongPins   int
	lockedUntil time.Time
}

// wrongPinDelay is a variable so tests don't wait.
var wrongPinDelay = time.Second

const (
	pinCookie   = "mirror_settings"
	maxWrongPin = 5
)

func newSettingsAPI(path string, cfg *Config, h *hub, cal *cached[CalendarResponse]) *settingsAPI {
	return &settingsAPI{path: path, cfg: cfg, hub: h, calendar: cal, restart: func() { os.Exit(1) }}
}

func (s *settingsAPI) routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/settings/unlock", s.homeOnly(s.unlock))
	mux.HandleFunc("POST /api/settings/lock", s.homeOnly(s.lock))
	mux.HandleFunc("GET /api/settings", s.local(s.get))
	mux.HandleFunc("POST /api/settings/calendar", s.local(s.editCalendar))
	mux.HandleFunc("POST /api/settings/style", s.local(s.setStyle))
	mux.HandleFunc("POST /api/settings/sleep", s.local(s.setSleep))
	mux.HandleFunc("POST /api/settings/say", s.local(s.say))
	mux.HandleFunc("POST /api/settings/screen", s.local(s.screen))
	mux.HandleFunc("POST /api/settings/summon", s.local(s.summon))
}

// homeOnly refuses anything that doesn't come from the home network.
func (s *settingsAPI) homeOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !fromHomeNetwork(r) {
			http.Error(w, "settings are only available on the home network", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// local also needs the PIN: a phone that entered it carries a cookie.
func (s *settingsAPI) local(next http.HandlerFunc) http.HandlerFunc {
	return s.homeOnly(func(w http.ResponseWriter, r *http.Request) {
		if !s.unlocked(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"locked": true, "pinSet": s.pin() != ""})
			return
		}
		next(w, r)
	})
}

func (s *settingsAPI) pin() string { return strings.TrimSpace(s.cfg.SettingsPin) }

// pinToken is what the cookie holds. It changes when the PIN does, so a
// new PIN signs every phone out. Letters aren't case-sensitive.
func pinToken(pin string) string {
	sum := sha256.Sum256([]byte("coconut-kitchen-settings:" + strings.ToLower(pin)))
	return hex.EncodeToString(sum[:])
}

func (s *settingsAPI) unlocked(r *http.Request) bool {
	pin := s.pin()
	if pin == "" {
		return false
	}
	c, err := r.Cookie(pinCookie)
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(pinToken(pin))) == 1
}

func (s *settingsAPI) unlock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pin string `json:"pin"`
	}
	if !decode(w, r, &req) {
		return
	}
	pin := s.pin()
	if pin == "" {
		http.Error(w, "No PIN is set yet. On the Pi, run: mirror pin <your PIN>", http.StatusConflict)
		return
	}
	s.pinMu.Lock()
	waiting := time.Now().Before(s.lockedUntil)
	s.pinMu.Unlock()
	if waiting {
		http.Error(w, "Too many tries. Wait a minute and try again.", http.StatusTooManyRequests)
		return
	}

	given := strings.ToLower(strings.TrimSpace(req.Pin))
	if subtle.ConstantTimeCompare([]byte(given), []byte(strings.ToLower(pin))) != 1 {
		time.Sleep(wrongPinDelay)
		s.pinMu.Lock()
		s.wrongPins++
		if s.wrongPins >= maxWrongPin {
			s.wrongPins = 0
			s.lockedUntil = time.Now().Add(time.Minute)
		}
		s.pinMu.Unlock()
		http.Error(w, "That’s not the PIN.", http.StatusUnauthorized)
		return
	}
	s.pinMu.Lock()
	s.wrongPins = 0
	s.pinMu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: pinCookie, Value: pinToken(pin), Path: "/",
		MaxAge: 30 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// lock forgets the PIN on this phone.
func (s *settingsAPI) lock(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: pinCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func fromHomeNetwork(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

type settingsLink struct {
	// Only the host is shared: the rest of the address is the secret.
	Host string `json:"host"`
	// False for a placeholder that isn't a calendar address yet.
	Real bool `json:"real"`
}

// lookPath is a variable so tests can pretend wlopm is missing.
var lookPath = exec.LookPath

type settingsPerson struct {
	Name   string         `json:"name"`
	Links  []settingsLink `json:"links"`
	Events int            `json:"events"`
	Failed bool           `json:"failed"`
}

func (s *settingsAPI) get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	raw, err := s.read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cal, calErr := s.calendar.get(ctx)
	people := []settingsPerson{}
	for _, p := range rawPeople(raw) {
		sp := settingsPerson{Name: str(p["name"]), Links: []settingsLink{}}
		for _, u := range urlList(p["icsUrls"]) {
			sp.Links = append(sp.Links, settingsLink{Host: hostOf(u), Real: len(realURLs([]string{u})) == 1})
		}
		if calErr == nil {
			for _, e := range cal.Events {
				if contains(e.Who, sp.Name) {
					sp.Events++
				}
			}
			sp.Failed = contains(cal.Failed, sp.Name)
		} else {
			sp.Failed = len(sp.Links) > 0
		}
		people = append(people, sp)
	}
	sleep := defaultSleep()
	if s.cfg.Sleep != nil {
		sleep = *s.cfg.Sleep
	}
	_, wlopmErr := lookPath("wlopm")
	writeJSON(w, http.StatusOK, map[string]any{
		"people":        people,
		"style":         s.cfg.Style,
		"sleep":         sleep,
		"screenControl": wlopmErr == nil,
		"speaker":       s.speakerName(),
	})
}

// speakerName is the speaker messages can be said on, or "" if none is
// connected.
func (s *settingsAPI) speakerName() string {
	if s.speaker == nil {
		return ""
	}
	if np := s.speaker.snapshot(time.Now()); np.Connected {
		return np.Speaker
	}
	return ""
}

func (s *settingsAPI) editCalendar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Person string `json:"person"`
		Add    string `json:"add"`
		Remove *int   `json:"remove"`
	}
	if !decode(w, r, &req) {
		return
	}
	add := strings.TrimSpace(req.Add)
	if add != "" && !validCalendarURL(add) {
		http.Error(w, "That doesn't look like a calendar link. It should start with https:// or webcal://.", http.StatusBadRequest)
		return
	}
	err := s.change(func(raw map[string]any) error {
		for _, p := range rawPeople(raw) {
			if str(p["name"]) != req.Person {
				continue
			}
			urls := urlList(p["icsUrls"])
			if add != "" {
				for _, u := range urls {
					if u == add {
						return errors.New("That link is already there.")
					}
				}
				urls = append(urls, add)
			}
			if req.Remove != nil {
				i := *req.Remove
				if i < 0 || i >= len(urls) {
					return errors.New("That link is no longer there. Reload and try again.")
				}
				urls = append(urls[:i:i], urls[i+1:]...)
			}
			anyURLs := make([]any, len(urls))
			for i, u := range urls {
				anyURLs[i] = u
			}
			p["icsUrls"] = anyURLs
			return nil
		}
		return errors.New("No one by that name is set up on the mirror.")
	})
	s.finish(w, err)
}

func (s *settingsAPI) setStyle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Style string `json:"style"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Style != "kitchen" && req.Style != "terrain" && req.Style != "haunted" {
		http.Error(w, "style must be kitchen, terrain or haunted", http.StatusBadRequest)
		return
	}
	s.finish(w, s.change(func(raw map[string]any) error {
		raw["style"] = req.Style
		return nil
	}))
}

func (s *settingsAPI) setSleep(w http.ResponseWriter, r *http.Request) {
	var req SleepConfig
	if !decode(w, r, &req) {
		return
	}
	never := req.From == "" && req.To == ""
	_, okFrom := clockMinutes(req.From)
	_, okTo := clockMinutes(req.To)
	if !never && !(okFrom && okTo) {
		http.Error(w, "Use 24-hour times like 01:00 and 06:30.", http.StatusBadRequest)
		return
	}
	s.finish(w, s.change(func(raw map[string]any) error {
		raw["sleep"] = map[string]any{"from": req.From, "to": req.To}
		return nil
	}))
}

// say puts a message on the mirror right away; nothing is saved.
func (s *settingsAPI) say(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text    string `json:"text"`
		From    string `json:"from"`
		Minutes int    `json:"minutes"`
		Speak   bool   `json:"speak"`
	}
	if !decode(w, r, &req) {
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" || len(text) > 280 {
		http.Error(w, "Write a short message (up to 280 characters).", http.StatusBadRequest)
		return
	}
	minutes := req.Minutes
	if minutes <= 0 || minutes > 24*60 {
		minutes = 10
	}
	s.hub.send(Message{Text: text, Source: strings.TrimSpace(req.From), TTL: minutes * 60, At: time.Now(), Speak: req.Speak})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// visitors are who can be sent across the haunted page's grounds; the
// same names as frontend/src/widgets/haunted/Visitors.tsx.
var visitors = map[string]bool{"zombie": true, "cat": true, "crow": true, "witch": true, "hand": true, "balloon": true, "bats": true}

// summon sends a visitor across the haunted page right now.
func (s *settingsAPI) summon(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Visitor string `json:"visitor"`
	}
	if !decode(w, r, &req) {
		return
	}
	if !visitors[req.Visitor] {
		http.Error(w, "No visitor by that name.", http.StatusBadRequest)
		return
	}
	if s.cfg.Style != "haunted" {
		http.Error(w, "Visitors only come to the Haunted page. Switch to it under Look.", http.StatusConflict)
		return
	}
	s.hub.event("summon", req.Visitor)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// screen switches the monitor now; the sleep schedule takes over again
// at its next boundary.
func (s *settingsAPI) screen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		On bool `json:"on"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := setDisplay(req.On); err != nil {
		log.Printf("settings: screen: %v", err)
		http.Error(w, "The screen couldn't be switched from here.", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// change edits config.json as plain JSON, so settings this page doesn't
// know about are kept exactly, and writes it atomically.
func (s *settingsAPI) change(edit func(map[string]any) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := s.read()
	if err != nil {
		return err
	}
	if err := edit(raw); err != nil {
		return userError{err}
	}
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if info, err := os.Stat(s.path); err == nil {
		_ = os.Chmod(tmp.Name(), info.Mode().Perm())
	}
	return os.Rename(tmp.Name(), s.path)
}

func (s *settingsAPI) read() (map[string]any, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	raw := map[string]any{}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber() // keep numbers exactly as written
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type userError struct{ error }

// finish answers, then restarts the server so the new config takes effect.
func (s *settingsAPI) finish(w http.ResponseWriter, err error) {
	var ue userError
	if errors.As(err, &ue) {
		http.Error(w, ue.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("settings: %v", err)
		http.Error(w, "The change couldn't be saved.", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"restarting": true})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	log.Printf("settings: config.json changed from the settings page; restarting")
	go func() {
		time.Sleep(500 * time.Millisecond)
		s.restart()
	}()
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func validCalendarURL(u string) bool {
	return (strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "webcal://")) &&
		len(u) < 2048 && !strings.ContainsAny(u, " \t\n\"")
}

func rawPeople(raw map[string]any) []map[string]any {
	cal, _ := raw["calendar"].(map[string]any)
	list, _ := cal["people"].([]any)
	out := []map[string]any{}
	for _, p := range list {
		if m, ok := p.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// urlList reads icsUrls as either one address or a list, like stringList.
func urlList(v any) []string {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return []string{}
		}
		return []string{t}
	case []any:
		out := []string{}
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return []string{}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

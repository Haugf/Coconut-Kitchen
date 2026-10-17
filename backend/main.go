package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	cfgPath := flag.String("config", "config.json", "path to config file")
	static := flag.String("static", "", "override staticDir from config (used on the Pi)")
	flag.Parse()

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *static != "" {
		cfg.StaticDir = *static
	}

	hub := newHub(cfg.EventsToken)

	mux := http.NewServeMux()
	weather := newWeatherSource(cfg.Weather)
	calendar := newCalendarSource(cfg.Calendar)
	transitCfg := defaultTransit()
	if cfg.Transit != nil {
		transitCfg = *cfg.Transit
	}
	trains := newTransit(transitCfg)
	minimap := newMinimap(cfg.Map, trains)
	sleep := newSleeper(cfg.Sleep)
	tidesCfg := defaultTides()
	if cfg.Tides != nil {
		tidesCfg = *cfg.Tides
	}
	tides := newTidesSource(tidesCfg)
	castCfg := defaultCast()
	if cfg.Cast != nil {
		castCfg = *cfg.Cast
	}
	var speakerSrc *caster
	if !castCfg.Off {
		speakerSrc = newCaster(castCfg)
		hub.speaker = speakerSrc
		go speakerSrc.run(context.Background())
	}
	// Google's timers need the speaker; `mirror google` turns them on.
	var timers *googleTimers
	if speakerSrc != nil {
		timers = newGoogleTimers(speakerSrc)
		go timers.run(context.Background())
	}
	board := &statusBoard{started: time.Now(), calendar: calendar, weather: weather, transit: trains, minimap: minimap, sleep: sleep, tides: tides, speaker: speakerSrc, timers: timers}

	mux.Handle("GET /api/weather", serveCached(weather))
	mux.Handle("GET /api/calendar", partyPrivate(cfg, serveCached(calendar)))
	mux.Handle("GET /api/tides", serveCached(tides))
	mux.Handle("GET /api/transit", trains)
	if speakerSrc != nil {
		mux.Handle("GET /api/cast", speakerSrc)
	} else {
		mux.HandleFunc("GET /api/cast", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, NowPlaying{At: time.Now()})
		})
	}
	if timers != nil {
		mux.Handle("GET /api/timers", timers)
	} else {
		mux.HandleFunc("GET /api/timers", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, TimersResponse{Timers: []Timer{}, At: time.Now()})
		})
	}
	mux.Handle("GET /api/map", minimap)
	go sleep.run()
	go board.watchBrowser()
	mux.Handle("GET /api/sleep", sleep)
	mux.HandleFunc("GET /api/style", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"style": cfg.Style})
	})
	widgets := newWidgetHost(widgetsDir(), cfg.Widgets)
	board.widgets = widgets
	mux.HandleFunc("GET /api/widgets", widgets.list)
	mux.HandleFunc("GET /api/w/{id}/{source}", widgets.source)
	mux.HandleFunc("GET /api/status", board.status)
	mux.HandleFunc("POST /api/heartbeat", board.heartbeat)
	settings := newSettingsAPI(*cfgPath, cfg, hub, calendar)
	settings.speaker = speakerSrc
	settings.routes(mux)
	mux.HandleFunc("GET /api/events", hub.stream)
	mux.HandleFunc("POST /api/events", hub.publish)
	mux.Handle("/", spaHandler(cfg.StaticDir))

	// No WriteTimeout: /api/events is a long-lived SSE stream.
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("mirror listening on %s, serving %s", cfg.Addr, cfg.StaticDir)
	log.Fatal(srv.ListenAndServe())
}

// spaHandler serves the built frontend, falling back to index.html.
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}

// partyPrivate keeps the calendar off the network while the haunted
// party page is on: guests share the Wi-Fi, and that page never shows
// the calendar anyway. The Pi itself can still ask.
func partyPrivate(cfg *Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Style == "haunted" && !fromLoopback(r) {
			http.Error(w, "The calendar is private during the party.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

package main

import (
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
	board := &statusBoard{started: time.Now(), calendar: calendar, weather: weather, transit: trains, minimap: minimap}

	mux.Handle("GET /api/weather", serveCached(weather))
	mux.Handle("GET /api/calendar", serveCached(calendar))
	mux.Handle("GET /api/transit", trains)
	mux.Handle("GET /api/map", minimap)
	mux.HandleFunc("GET /api/status", board.status)
	mux.HandleFunc("POST /api/heartbeat", board.heartbeat)
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

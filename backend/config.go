package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Addr      string `json:"addr"`
	StaticDir string `json:"staticDir"`
	// Optional. When set, POST /api/events requires
	// "Authorization: Bearer <token>".
	EventsToken string         `json:"eventsToken"`
	Weather     WeatherConfig  `json:"weather"`
	Calendar    CalendarConfig `json:"calendar"`
}

type WeatherConfig struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Units     string  `json:"units"` // "fahrenheit" or "celsius"
}

type CalendarConfig struct {
	// Google Calendar: Settings > your calendar > "Secret address in iCal format".
	ICSURLs   []string `json:"icsUrls"`
	DaysAhead int      `json:"daysAhead"`
	MaxEvents int      `json:"maxEvents"`
}

func loadConfig(path string) (*Config, error) {
	cfg := &Config{
		Addr:      ":8080",
		StaticDir: "../frontend/dist",
		Weather:   WeatherConfig{Latitude: 40.7128, Longitude: -74.0060, Units: "fahrenheit"},
		Calendar:  CalendarConfig{DaysAhead: 7, MaxEvents: 8},
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w (copy config.example.json to get started)", path, err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

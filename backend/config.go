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
	// One entry per person whose day shows on the mirror. Each URL is a
	// Google Calendar "Secret address in iCal format" or an Apple
	// Calendar public link (webcal:// is fine).
	People []Person `json:"people"`
	// Older single-person form, still accepted.
	ICSURLs   stringList `json:"icsUrls"`
	DaysAhead int      `json:"daysAhead"`
	MaxEvents int      `json:"maxEvents"`
}

type Person struct {
	Name    string     `json:"name"`
	ICSURLs stringList `json:"icsUrls"`
}

// stringList accepts either ["a", "b"] or a single "a", since a lone
// address without brackets is an easy mistake to make by hand.
type stringList []string

func (l *stringList) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*l = stringList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("expected an address or a list of addresses: %w", err)
	}
	*l = many
	return nil
}

func loadConfig(path string) (*Config, error) {
	cfg := &Config{
		Addr:      ":8080",
		StaticDir: "../frontend/dist",
		Weather:   WeatherConfig{Latitude: 40.7128, Longitude: -74.0060, Units: "fahrenheit"},
		Calendar:  CalendarConfig{DaysAhead: 7, MaxEvents: 30},
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

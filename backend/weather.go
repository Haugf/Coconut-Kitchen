package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Weather struct {
	Temp      float64      `json:"temp"`
	FeelsLike float64      `json:"feelsLike"`
	Code      int          `json:"code"`
	Units     string       `json:"units"`
	Days      []WeatherDay `json:"days"`
}

type WeatherDay struct {
	Date       string  `json:"date"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Code       int     `json:"code"`
	RainChance int     `json:"rainChance"`
}

// Open-Meteo: free, no API key.
type openMeteo struct {
	Current struct {
		Temp      float64 `json:"temperature_2m"`
		FeelsLike float64 `json:"apparent_temperature"`
		Code      int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		Time []string   `json:"time"`
		Max  []*float64 `json:"temperature_2m_max"`
		Min  []*float64 `json:"temperature_2m_min"`
		Code []*float64 `json:"weather_code"`
		Rain []*float64 `json:"precipitation_probability_max"`
	} `json:"daily"`
}

func newWeatherSource(cfg WeatherConfig) *cached[Weather] {
	return &cached[Weather]{
		ttl: 10 * time.Minute,
		fetch: func(ctx context.Context) (Weather, error) {
			return fetchWeather(ctx, cfg)
		},
	}
}

func fetchWeather(ctx context.Context, cfg WeatherConfig) (Weather, error) {
	q := url.Values{}
	q.Set("latitude", fmt.Sprint(cfg.Latitude))
	q.Set("longitude", fmt.Sprint(cfg.Longitude))
	q.Set("current", "temperature_2m,apparent_temperature,weather_code")
	q.Set("daily", "temperature_2m_max,temperature_2m_min,weather_code,precipitation_probability_max")
	q.Set("temperature_unit", cfg.Units)
	q.Set("timezone", "auto")
	q.Set("forecast_days", "5")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), nil)
	if err != nil {
		return Weather{}, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Weather{}, fmt.Errorf("weather: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Weather{}, fmt.Errorf("weather: upstream returned %s", resp.Status)
	}

	var om openMeteo
	if err := json.NewDecoder(resp.Body).Decode(&om); err != nil {
		return Weather{}, fmt.Errorf("weather: decode: %w", err)
	}

	w := Weather{
		Temp:      om.Current.Temp,
		FeelsLike: om.Current.FeelsLike,
		Code:      om.Current.Code,
		Units:     cfg.Units,
	}
	for i, date := range om.Daily.Time {
		w.Days = append(w.Days, WeatherDay{
			Date:       date,
			High:       at(om.Daily.Max, i),
			Low:        at(om.Daily.Min, i),
			Code:       int(at(om.Daily.Code, i)),
			RainChance: int(at(om.Daily.Rain, i)),
		})
	}
	return w, nil
}

// at safely reads a nullable value from an Open-Meteo daily series.
func at(s []*float64, i int) float64 {
	if i < len(s) && s[i] != nil {
		return *s[i]
	}
	return 0
}

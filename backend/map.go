package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// The minimap: where the trains and buses heading to your stops are right
// now, around home.
//
// Subway feeds don't carry GPS, so a train's position is estimated: it
// sits between the station it last left and the one it's heading to,
// placed by how many seconds it has left to get there. A train the feed
// lists but hasn't assigned yet is still waiting at the start of its line.
// Buses report real GPS through Bus Time.

type MapConfig struct {
	// Home is the centre of the map. Give an address, or lat/lon.
	Home *HomeConfig `json:"home"`
	// How far the map reaches from the centre, in metres.
	RadiusMeters float64 `json:"radiusMeters"`
}

type HomeConfig struct {
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

type latLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type station struct {
	Name string
	latLon
}

// From the MTA's station list (data.ny.gov, GTFS stop IDs).
var stations = map[string]station{
	"L13": {"Montrose Av", latLon{40.707739, -73.93985}},
	"L14": {"Morgan Av", latLon{40.706152, -73.933147}},
	"L15": {"Jefferson St", latLon{40.706607, -73.922913}},
	"L16": {"DeKalb Av", latLon{40.703811, -73.918425}},
	"L17": {"Myrtle–Wyckoff", latLon{40.699814, -73.911586}},
	"L19": {"Halsey St", latLon{40.695602, -73.904084}},
	"L20": {"Wilson Av", latLon{40.688764, -73.904046}},
	"L21": {"Bushwick Av", latLon{40.682829, -73.905249}},
	"M01": {"Metropolitan Av", latLon{40.711396, -73.889601}},
	"M04": {"Fresh Pond Rd", latLon{40.706186, -73.895877}},
	"M05": {"Forest Av", latLon{40.704423, -73.903077}},
	"M06": {"Seneca Av", latLon{40.702762, -73.90774}},
	"M08": {"Myrtle–Wyckoff", latLon{40.69943, -73.912385}},
	"M09": {"Knickerbocker Av", latLon{40.698664, -73.919711}},
	"M10": {"Central Av", latLon{40.697857, -73.927397}},
	"M11": {"Myrtle Av", latLon{40.697207, -73.935657}},
}

// Each line in order. Trains toward Manhattan ("N") run from the end of
// the list to the start; "S" runs the other way.
var lineStops = map[string][]string{
	"L": {"L13", "L14", "L15", "L16", "L17", "L19", "L20", "L21"},
	"M": {"M11", "M10", "M09", "M08", "M06", "M05", "M04", "M01"},
}

// The stations named on the map. Everything else stays unlabeled.
var labeledStations = []string{"L17", "M05", "M04"}

type MapVehicle struct {
	ID    string  `json:"id"`
	Kind  string  `json:"kind"` // subway or bus
	Route string  `json:"route"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	// Where it's coming from, for the short trail behind the dot.
	From *latLon `json:"from,omitempty"`
	// Minutes until it reaches your stop.
	Minutes int `json:"minutes"`
	// moving, stopped (at a station), waiting (hasn't left the start of
	// the line) or far (beyond the stations we know; drawn at the edge).
	State string `json:"state"`
	Note  string `json:"note,omitempty"`
}

type MapLine struct {
	Route  string   `json:"route"`
	Points []latLon `json:"points"`
}

type MapStation struct {
	Name string `json:"name"`
	latLon
}

type MapResponse struct {
	Updated      time.Time    `json:"updated"`
	Center       latLon       `json:"center"`
	Home         *latLon      `json:"home,omitempty"`
	RadiusMeters float64      `json:"radiusMeters"`
	Streets      [][]latLon   `json:"streets"`
	Lines        []MapLine    `json:"lines"`
	Stations     []MapStation `json:"stations"`
	Vehicles     []MapVehicle `json:"vehicles"`
}

type minimap struct {
	cfg     MapConfig
	transit *transit

	mu        sync.Mutex
	home      *latLon
	homeTried time.Time
	streets   [][]latLon
	streetsAt time.Time
	tried     time.Time
}

func newMinimap(cfg MapConfig, t *transit) *minimap {
	if cfg.RadiusMeters <= 0 {
		cfg.RadiusMeters = 1300
	}
	m := &minimap{cfg: cfg, transit: t}
	if h := cfg.Home; h != nil && h.Lat != 0 && h.Lon != 0 {
		m.home = &latLon{h.Lat, h.Lon}
	}
	m.loadDisk()
	return m
}

func (m *minimap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, m.snapshot(r.Context()))
}

func (m *minimap) snapshot(ctx context.Context) MapResponse {
	now := time.Now()
	home := m.homeLocation(ctx)
	center := stations["M05"].latLon // Forest Av until home is known
	if home != nil {
		center = *home
	}
	resp := MapResponse{
		Updated:      now,
		Center:       center,
		Home:         home,
		RadiusMeters: m.cfg.RadiusMeters,
		Streets:      m.streetLines(ctx, center),
		Vehicles:     []MapVehicle{},
	}
	for _, route := range []string{"L", "M"} {
		line := MapLine{Route: route}
		for _, id := range lineStops[route] {
			line.Points = append(line.Points, stations[id].latLon)
		}
		resp.Lines = append(resp.Lines, line)
	}
	for _, id := range labeledStations {
		resp.Stations = append(resp.Stations, MapStation{Name: stations[id].Name, latLon: stations[id].latLon})
	}
	if resp.Streets == nil {
		resp.Streets = [][]latLon{}
	}
	resp.Vehicles = append(resp.Vehicles, m.trains(ctx, now)...)
	resp.Vehicles = append(resp.Vehicles, m.buses(ctx, now)...)
	return resp
}

// trains places every train that's on its way to one of your stops.
func (m *minimap) trains(ctx context.Context, now time.Time) []MapVehicle {
	best := map[string]MapVehicle{}
	for _, s := range m.transit.cfg.Subway {
		trips, err := m.transit.feed(feedFor(s.Route)).get(ctx)
		if err != nil {
			continue
		}
		for _, tr := range trips {
			if !strings.EqualFold(tr.Route, s.Route) {
				continue
			}
			v, ok := placeTrain(tr, s.Stop, now)
			if !ok {
				continue
			}
			if old, seen := best[v.ID]; !seen || v.Minutes < old.Minutes {
				best[v.ID] = v
			}
		}
	}
	out := make([]MapVehicle, 0, len(best))
	for _, v := range best {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Minutes < out[j].Minutes })
	return out
}

// placeTrain works out where trip tr is, if it's still heading to
// myStop (like "M05N") within the next half hour.
func placeTrain(tr rtTrip, myStop string, now time.Time) (MapVehicle, bool) {
	var arrive int64
	for _, st := range tr.Stops {
		if st.ID == myStop {
			arrive = st.Time
		}
	}
	secsToMine := arrive - now.Unix()
	if arrive == 0 || secsToMine < -30 || secsToMine > 30*60 {
		return MapVehicle{}, false
	}
	route := strings.ToUpper(tr.Route)
	order := lineStops[route]
	if len(order) == 0 || len(myStop) < 4 {
		return MapVehicle{}, false
	}
	dir := myStop[len(myStop)-1:]

	// The next stop is the first one the train hasn't reached yet.
	var next rtStop
	nextIdx := -1
	for i, st := range tr.Stops {
		if st.Time >= now.Unix()-20 {
			next, nextIdx = st, i
			break
		}
	}
	if nextIdx < 0 {
		return MapVehicle{}, false
	}

	v := MapVehicle{
		ID:      tr.ID,
		Kind:    "subway",
		Route:   route,
		Minutes: max(0, int((secsToMine+30)/60)),
		State:   "moving",
	}
	if v.ID == "" {
		v.ID = fmt.Sprintf("%s-%s-%d", route, myStop, arrive)
	}

	base := strings.TrimRight(next.ID, "NS")
	idx := indexOf(order, base)
	if idx < 0 {
		// Still beyond the stations we draw: park it at the far end of
		// the line it's coming from, and let the map show an edge arrow.
		end := order[len(order)-1]
		if dir == "S" {
			end = order[0]
		}
		v.Lat, v.Lon, v.State = stations[end].Lat, stations[end].Lon, "far"
		if !tr.Assigned {
			v.Note = "hasn't left yet"
		}
		return v, true
	}
	here := stations[base]
	prevIdx := idx + 1 // toward Manhattan, trains come from the end of the list
	if dir == "S" {
		prevIdx = idx - 1
	}

	switch {
	case !tr.Assigned && nextIdx == 0:
		v.Lat, v.Lon, v.State = here.Lat, here.Lon, "waiting"
		v.Note = "hasn't left " + here.Name
	case tr.Assigned && tr.Status == statusStopped && strings.TrimRight(tr.AtStop, "NS") == base:
		v.Lat, v.Lon, v.State = here.Lat, here.Lon, "stopped"
		v.Note = "at " + here.Name
	case prevIdx < 0 || prevIdx >= len(order):
		v.Lat, v.Lon = here.Lat, here.Lon
	default:
		prev := stations[order[prevIdx]]
		// A rough time between stations from the distance, about 30 km/h.
		seg := math.Max(60, math.Min(240, metersBetween(prev.latLon, here.latLon)/8.5))
		left := float64(next.Time - now.Unix())
		f := math.Max(0, math.Min(1, left/seg)) // 0 at the next station, 1 at the last
		v.Lat = here.Lat + (prev.Lat-here.Lat)*f
		v.Lon = here.Lon + (prev.Lon-here.Lon)*f
		v.From = &latLon{prev.Lat, prev.Lon}
		v.Note = "to " + here.Name
	}
	return v, true
}

func (m *minimap) buses(ctx context.Context, now time.Time) []MapVehicle {
	t := m.transit
	if t.cfg.BusAPIKey == "" {
		return nil
	}
	var out []MapVehicle
	seen := map[string]bool{}
	for _, s := range t.cfg.Bus {
		if s.StopCode == "" {
			continue
		}
		arrivals, err := t.bus(s.StopCode, s.Route).get(ctx)
		if err != nil {
			continue
		}
		for _, a := range arrivals {
			if a.Lat == 0 || a.Lon == 0 || seen[a.Vehicle] {
				continue
			}
			mins := int((a.At.Unix() - now.Unix() + 30) / 60)
			if mins < 0 || mins > 30 {
				continue
			}
			seen[a.Vehicle] = true
			out = append(out, MapVehicle{
				ID: a.Vehicle, Kind: "bus", Route: strings.ToUpper(s.Route),
				Lat: a.Lat, Lon: a.Lon, Minutes: mins, State: "moving", Note: a.Away,
			})
		}
	}
	return out
}

// --- Home --------------------------------------------------------------

const homeCacheFile = "map-home.json"
const streetsCacheFile = "map-streets.json"

func (m *minimap) loadDisk() {
	if m.home == nil && m.cfg.Home != nil && m.cfg.Home.Address != "" {
		var saved struct {
			Address string `json:"address"`
			latLon
		}
		if b, err := os.ReadFile(homeCacheFile); err == nil && json.Unmarshal(b, &saved) == nil &&
			saved.Address == m.cfg.Home.Address && saved.Lat != 0 {
			m.home = &latLon{saved.Lat, saved.Lon}
		}
	}
	var saved struct {
		At      time.Time  `json:"at"`
		Center  latLon     `json:"center"`
		Streets [][]latLon `json:"streets"`
	}
	if b, err := os.ReadFile(streetsCacheFile); err == nil && json.Unmarshal(b, &saved) == nil {
		m.streets, m.streetsAt = saved.Streets, saved.At
	}
}

// homeLocation looks the address up once and remembers it on disk.
func (m *minimap) homeLocation(ctx context.Context) *latLon {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.home != nil || m.cfg.Home == nil || m.cfg.Home.Address == "" {
		return m.home
	}
	if time.Since(m.homeTried) < 15*time.Minute {
		return nil
	}
	m.homeTried = time.Now()
	ll, err := geocode(ctx, m.cfg.Home.Address)
	if err != nil {
		log.Printf("map: couldn't find home on the map: %v", err)
		return nil
	}
	m.home = &ll
	b, _ := json.Marshal(map[string]any{"address": m.cfg.Home.Address, "lat": ll.Lat, "lon": ll.Lon})
	_ = os.WriteFile(homeCacheFile, b, 0o600)
	// The map's centre moved, so fetch streets for the new area.
	m.streetsAt = time.Time{}
	return m.home
}

func geocode(ctx context.Context, address string) (latLon, error) {
	// US Census geocoder first (no key, good with US street addresses),
	// then OpenStreetMap's.
	q := url.Values{"address": {address}, "benchmark": {"Public_AR_Current"}, "format": {"json"}}
	var census struct {
		Result struct {
			AddressMatches []struct {
				Coordinates struct{ X, Y float64 }
			} `json:"addressMatches"`
		} `json:"result"`
	}
	if err := getJSON(ctx, "https://geocoding.geo.census.gov/geocoder/locations/onelineaddress?"+q.Encode(), &census); err == nil &&
		len(census.Result.AddressMatches) > 0 {
		c := census.Result.AddressMatches[0].Coordinates
		return latLon{c.Y, c.X}, nil
	}
	var osm []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	q = url.Values{"q": {address}, "format": {"json"}, "limit": {"1"}}
	if err := getJSON(ctx, "https://nominatim.openstreetmap.org/search?"+q.Encode(), &osm); err != nil {
		return latLon{}, err
	}
	if len(osm) == 0 {
		return latLon{}, fmt.Errorf("no match for the home address")
	}
	var ll latLon
	fmt.Sscan(osm[0].Lat, &ll.Lat)
	fmt.Sscan(osm[0].Lon, &ll.Lon)
	return ll, nil
}

// --- Streets -----------------------------------------------------------

// streetLines returns the main streets around the centre, from
// OpenStreetMap. Fetched once and kept on disk; streets don't move.
func (m *minimap) streetLines(ctx context.Context, center latLon) [][]latLon {
	m.mu.Lock()
	defer m.mu.Unlock()
	fresh := !m.streetsAt.IsZero() && time.Since(m.streetsAt) < 60*24*time.Hour
	if fresh || time.Since(m.tried) < 10*time.Minute {
		return m.streets
	}
	m.tried = time.Now()
	lines, err := fetchStreets(ctx, center, m.cfg.RadiusMeters*1.2)
	if err != nil {
		log.Printf("map: streets: %v", err)
		return m.streets
	}
	m.streets, m.streetsAt = lines, time.Now()
	b, _ := json.Marshal(map[string]any{"at": m.streetsAt, "center": center, "streets": lines})
	_ = os.WriteFile(streetsCacheFile, b, 0o644)
	return m.streets
}

func fetchStreets(ctx context.Context, c latLon, radius float64) ([][]latLon, error) {
	around := fmt.Sprintf("around:%.0f,%.5f,%.5f", radius, c.Lat, c.Lon)
	query := fmt.Sprintf(`[out:json][timeout:25];(`+
		`way["highway"~"^(primary|secondary|tertiary)$"](%s);`+
		`way["highway"]["name"~"^(Palmetto Street|Gates Avenue|Fairview Avenue|Forest Avenue|Wyckoff Avenue|Fresh Pond Road|Myrtle Avenue)$"](%s);`+
		`);out geom;`, around, around)
	var body struct {
		Elements []struct {
			Geometry []latLon `json:"geometry"`
		} `json:"elements"`
	}
	if err := getJSON(ctx, "https://overpass-api.de/api/interpreter?data="+url.QueryEscape(query), &body); err != nil {
		return nil, err
	}
	var out [][]latLon
	for _, e := range body.Elements {
		if len(e.Geometry) < 2 {
			continue
		}
		line := make([]latLon, len(e.Geometry))
		for i, p := range e.Geometry {
			line[i] = latLon{math.Round(p.Lat*1e5) / 1e5, math.Round(p.Lon*1e5) / 1e5}
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no streets returned")
	}
	return out, nil
}

func getJSON(ctx context.Context, u string, into any) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "coconut-kitchen-mirror/1 (personal magic mirror)")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", req.URL.Host, resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(into)
}

// --- Helpers -----------------------------------------------------------

func indexOf(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

func metersBetween(a, b latLon) float64 {
	const r = 6371000
	la1, la2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dla, dlo := la2-la1, (b.Lon-a.Lon)*math.Pi/180
	h := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

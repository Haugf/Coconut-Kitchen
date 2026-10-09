# Working on Coconut Kitchen

Notes for any coding agent (Claude, Codex, Cursor, and so on) that clones
this repo to change the mirror. Read this before you touch anything. The
README covers what the mirror is and how a person runs it; this file covers
how to change it safely.

## The one thing to know first

**A push to `main` goes live on a real screen in about two minutes.** A
Raspberry Pi in Fred's apartment checks GitHub every minute, rebuilds the
Go server, copies `frontend/dist`, and restarts. Nobody reviews it in
between. So:

- Work on a branch and open a pull request unless you were told to push
  to `main`.
- Never push a broken build. Run the checks below first.
- If you change the frontend, rebuild it and commit `frontend/dist`. The Pi
  has no Node; it serves exactly what's committed there.

## What it is

A single full-screen page ("terrain") on a 24" monitor, landscape at
1280x720 @ 50Hz (the cable shimmers at 1080p). It shows:

- the date, weather and a one-line headline about the day
- the day drawn as a ridge from 6 am to midnight, one line per person (Fred
  solid, Ally dashed)
- Today's events, train times, and a round minimap of trains and buses
  heading to the configured stops

The day scene runs from 5 am and night from 10 pm (dark palette, dimmed).
The screen sleeps from 1:00 to 6:30.

```
backend/    Go server (one dependency: github.com/apognu/gocal)
  main.go       routes
  config.go     config.json shape
  cache.go      cached[T]: TTL + last-good fallback + health()
  calendar.go   iCal feeds per person, merged
  weather.go    Open-Meteo
  gtfsrt.go     hand-rolled GTFS-realtime protobuf reader
  transit.go    subway (GTFS-rt) and bus (MTA Bus Time SIRI) arrivals
  map.go        minimap: train positions, bus GPS, OSM streets, home
  sleep.go      turns the HDMI output off overnight (wlr-randr)
  status.go     /api/status and the page heartbeat
frontend/   Vite + React + TypeScript
  src/App.tsx            scene switch, sleep, live messages
  src/scenes.ts          which widgets show when
  src/widgets/terrain/   the page: Terrain, Transit, Minimap, terrain.css
  dist/                  committed build the Pi serves
deploy/     setup-pi.sh (one-time, needs sudo), mirror-status.sh, status reporting
install.sh  builds and installs on the Pi (runs on every update)
update.sh   the every-minute check: origin/main vs ~/mirror/.installed
```

### API

| Route | What |
|---|---|
| `GET /api/weather` | current and daily weather |
| `GET /api/calendar` | `{people, events[{title,start,end,allDay,who}], failed}` |
| `GET /api/transit` | `{rows[{kind,route,label,stopName,minutes[],ok}]}` |
| `GET /api/map` | `{center, home?, radiusMeters, streets, lines, stations, vehicles[{route,lat,lon,from?,minutes,state}]}` |
| `GET /api/sleep` | `{asleep, from, to}` |
| `GET /api/status` | health summary (counts only) |
| `POST /api/heartbeat` | the page checks in every minute |
| `GET/POST /api/events` | live text messages over SSE |

## Checks before you commit

```
cd backend && go vet ./... && go test ./...
cd frontend && npx tsc --noEmit -p . && npm run build
```

Then look at the page. Sandboxes usually can't reach the MTA, Google or
OpenStreetMap, so use the mock API, which serves `frontend/dist` with
sample data:

```
python3 dev/mock_api.py            # http://127.0.0.1:8099
```

Screenshot it at 1280x720, and at night by faking the clock (for example
with Playwright's `addInitScript` overriding `Date`). Check that nothing
clips or overlaps, in both palettes.

## Rules of the house

**Privacy. The repo is public.**
- Never commit `config.json`, calendar addresses, API keys, a home address
  or home coordinates. They live only in `~/mirror/config.json` on the Pi.
- `/api/status` and the `pi-status` branch report counts and health only,
  never event titles, addresses, locations or keys.
- Logs name a calendar by host only, never its secret URL. The Bus Time key
  is scrubbed from errors.

**Design.** The page follows a terrain style guide:
- Two bands (a sand wash on top, ivory below) meeting at a 1px edge, with
  no cards, shadows or radii.
- A closed palette in `terrain.css`, with clay (`--clay`) as the only accent.
- Fraunces for the headline, a sans for everything else.
- An observational voice ("A climb to Dinner at 7, then the day opens up.").

Transit colors are the MTA's: M `#FF6319`, L `#A7A9AC`, buses `#2F6BFF`.
Keep things minimal; Fred has said more than once not to over-engineer it.
New colors or decoration need a reason.

**Resilience.** Every upstream goes through `cached[T]`, so a failure keeps
showing the last good data. One bad calendar must not blank the others.
Anything slow or rate-limited (geocoding, OpenStreetMap) is fetched rarely
and cached on disk in `~/mirror`.

**Pi constraints.**
- Raspberry Pi 4 (4 GB) running Raspberry Pi OS 64-bit, with labwc on
  Wayland and Chromium in kiosk mode.
- The server runs as user `haugf` from `~/mirror` (that's its working
  directory).
- Automatic updates have passwordless sudo for exactly one command:
  `systemctl restart mirror.service`. Anything else needing root (apt,
  systemd units, kiosk.sh) lives in `deploy/setup-pi.sh`, which only reruns
  when its hash changes and sudo works without a password, so it usually
  won't run. Prefer changes that need no root.
- There's no emoji font on the Pi, so `plainTitle` strips emoji from event
  titles.
- Don't kill Chromium from an update. The page reloads itself when
  `index.html` changes (`useReloadOnUpdate`).

**Commits.** Write plain messages that say what changed and why. Keep the
Go server at one dependency. Don't add Node to the Pi.

## Adding something

1. **Data:** add a source in `backend/`, wrapped in `cached[T]`, with a
   route in `main.go`. Add its settings to `Config` with a working default,
   and to `config.example.json`.
2. **Display:** add it to `frontend/src/widgets/terrain/` and poll it with
   `useWidgetData(url, ms)`. Style it with the tokens in `terrain.css` and
   add night overrides under `.mirror-page.is-dim .terrain` if needed.
3. **Health:** if it can fail, add it to `/api/status` and a line in
   `deploy/mirror-status.sh`.
4. **Docs:** add a line to the README's "What's on the screen" table, and
   to the settings section if it needs config.
5. Run the checks, rebuild `frontend/dist`, commit.

## Checking the live mirror

Fred runs `mirror-status` on the Pi (over SSH). If he has set up
`deploy/report-setup.sh`, the same report is pushed every 5 minutes to the
`pi-status` branch (`status.json`, `status.txt`, `recent.log`), which an
agent can read with `git fetch origin pi-status`.

## Ideas on the list

- The B13 bus needs a Bus Time key and two stop codes in config. The code
  is ready; buses then appear on the map with real GPS.
- Portrait/wall mode (`ROTATION=90 bash ~/mirror/setup-pi.sh`); the layout
  already has portrait CSS but hasn't been tuned since the minimap.
- Voice or Google Home replies shown on screen via `POST /api/events`.

# Working on Coconut Kitchen

Notes for any coding agent (Claude, Codex, Cursor, and so on) that clones
this repo to change the mirror. Read this before you touch anything. The
README covers what the mirror is and how a person runs it; this file covers
how to change it safely.

## The one thing to know first

**Only Fred merges into `main`.** Whatever lands on `main` goes live on a
real screen in about two minutes: a Raspberry Pi in Fred's apartment checks
GitHub every minute, rebuilds the Go server, copies `frontend/dist`, and
restarts. So `main` is protected:

- Never push to `main`, and never merge a pull request, even if your token
  would allow it. Push a branch, open a pull request, and tell Fred it's
  ready. He reviews and merges.
- Don't change branch protection, repository settings, deploy keys, or the
  Pi's update scripts (`update.sh`, `install.sh`, `bootstrap.sh`,
  `deploy/`) unless Fred asked for that specific change, and say so in
  the pull request.
- Never open a pull request with a broken build. Run the checks below first.
- If you change the frontend, rebuild it and commit `frontend/dist`. The Pi
  has no Node; it serves exactly what's committed there.

## What it is

A single full-screen page ("kitchen", Coconut Kitchen) on a 24" monitor,
landscape at 1280x720 @ 50Hz (the cable shimmers at 1080p). It shows:

- the date, weather, a "Coconut Kitchen" mark and a one-line headline
- Today's events, then the latest live message, what the Google speaker is
  playing, or tomorrow's events, then
  the enabled widgets in a row (by default train times and a round minimap
  of trains and buses heading to the configured stops)
- along the bottom, the day drawn as a line from 6 am to midnight, one per
  person (Fred solid, Ally dashed), with the tide behind it on the same clock

The kitchen page reuses the terrain page's logic (`terrain/day.ts`,
`shape.ts`, `voice.ts`, `lists.tsx`) and restyles everything, widgets
included, by redefining the same tokens. Both pages stay: `"style"` in
config.json picks one (`mirror style kitchen|terrain`), served at
`/api/style`; scenes ask for `'page'` and App resolves it. Change one page
without breaking the other.

The day scene runs from 5 am and night from 10 pm (dark palette, dimmed).
The screen sleeps from 1:00 to 6:30.

```
backend/    Go server (one dependency: github.com/apognu/gocal)
  main.go       routes
  config.go     config.json shape
  cache.go      cached[T]: TTL + last-good fallback + health()
  calendar.go   iCal feeds per person, merged
  tides.go      NOAA tide predictions for today and tomorrow
  weather.go    Open-Meteo
  gtfsrt.go     hand-rolled GTFS-realtime protobuf reader
  transit.go    subway (GTFS-rt) and bus (MTA Bus Time SIRI) arrivals
  map.go        minimap: train positions, bus GPS, OSM streets, home
  widgets.go    widget manifests, /api/widgets, cached data proxy for widgets
  cast.go       the Google speaker over Cast: what's playing, saying messages
  mdns.go       finds Cast speakers on the home network by name
  timers.go     the speaker's Google timers (local API, token via google_tokens.py)
  sleep.go      puts the screen in standby overnight (wlopm)
  status.go     /api/status and the page heartbeat
  settings.go   the phone settings page's API (home network only)
frontend/   Vite + React + TypeScript
  src/App.tsx            scene switch, sleep, live messages
  src/scenes.ts          which widgets show when
  src/settings/          the phone page at /settings
  src/widgets/kitchen/   the page: Kitchen, kitchen.css, tide.ts
  src/widgets/terrain/   the earlier page, plus the parts both pages share
  src/widgets/<id>/      one folder per widget (widget.json, index.tsx, sample.json)
  src/widgets/_template/ what `npm run new-widget -- <id>` copies
  src/widgets/loader.tsx finds widget folders; hides a widget that crashes
  scripts/               new-widget.mjs, check-widgets.mjs
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
| `GET /api/tides` | `{place, points[{t,v}], turns[{t,v,type}]}`, today and tomorrow |
| `GET /api/transit` | `{rows[{kind,route,label,stopName,minutes[],ok}]}` |
| `GET /api/map` | `{center, home?, radiusMeters, streets, lines, stations, vehicles[{route,lat,lon,from?,minutes,state}]}` |
| `GET /api/cast` | the Google speaker: `{speaker, connected, state?, app?, title?, subtitle?, live?, duration?, position?, at}` (never its address) |
| `GET /api/timers` | timers set by voice on the speaker: `{enabled, timers[{label?, state, fireAt?, duration, left?}], at}` |
| `GET /api/sleep` | `{asleep, from, to}` |
| `GET /api/style` | `{style}`: `kitchen` or `terrain` |
| `GET /api/widgets` | enabled widgets in order: `{id, name, width, settings, sources}` (never secrets) |
| `GET /api/w/{id}/{source}` | a widget's data source, fetched and cached by the Pi |
| `GET /api/status` | health summary (counts only) |
| `GET /api/settings` | the phone settings page's view: people with link hosts and event counts, style, sleep (never a full calendar address) |
| `POST /api/settings/unlock`, `/lock` | the PIN: sets or clears a cookie (30 days). Without it every other settings route answers 401 |
| `POST /api/settings/{calendar,style,sleep,say,screen}` | changes from the settings page; home network and PIN. The first three rewrite config.json and restart the server |
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
python3 dev/mock_api.py              # http://127.0.0.1:8099
python3 dev/mock_api.py --with <id>  # plus your widget
```

Screenshot it at 1280x720, and at night by faking the clock (for example
with Playwright's `addInitScript` overriding `Date`). Check that nothing
clips or overlaps, in both palettes.

## Rules of the house

**Privacy. The repo is public.**
- Never commit `config.json`, `google.json` (a Google sign-in token),
  calendar addresses, API keys, the settings PIN, a home address or home
  coordinates. They live only in `~/mirror/config.json` on the Pi.
- `/api/status` and the `pi-status` branch report counts and health only,
  never event titles, addresses, locations or keys.
- Logs and the settings page name a calendar by host only, never its
  secret URL. The Bus Time key
  is scrubbed from errors.

**Design.** The kitchen page:
- One dark husk-brown ground all day (a light ground lit up the kitchen
  too much), hairline column rules, no cards, shadows or radii.
- A closed palette at the top of `kitchen.css`: coconut-cream ink, toasted
  coconut (`--clay`) as the one accent, lagoon only for the tide. The
  tokens are registered with `@property` so day and night fade into each
  other; keep new tokens registered too.
- Young Serif for the headline, the mark and messages; Figtree for
  everything else.
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

Most new things should be a **widget**: follow WIDGETS.md. Start with
`npm run new-widget -- <id>`. A widget never edits core files, which
keeps pull requests small and safe to merge.

Change the core only for something no widget can do: a new kind of data
that needs real parsing (like `gtfsrt.go`), scenes, sleep, the page frame.
Then:

1. Add a source in `backend/`, wrapped in `cached[T]`, with a route in
   `main.go`. Give its settings in `Config` a working default, and add
   them to `config.example.json`.
2. If it can fail, add it to `/api/status` and a line in
   `deploy/mirror-status.sh`.
3. Add a line to the README's "What's on the screen" table.
4. Run the checks, rebuild `frontend/dist`, commit.

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

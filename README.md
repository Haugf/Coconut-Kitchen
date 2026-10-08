# Mirror

A personal magic mirror: a Vite + React frontend served by a small Go backend,
running full screen in Chromium on a Raspberry Pi 4 with a portrait monitor.

## Layout

```
frontend/   Vite + React + TS. Widgets, scenes, styles.
backend/    Go server. Fetches and caches APIs, serves the frontend, pushes live messages.
deploy/     One-time Pi setup (systemd service, kiosk autostart, rotation).
Makefile    setup, dev, build, deploy from the Mac.
```

## Set up a Pi in one line

On a fresh Raspberry Pi OS (64-bit) with SSH:

```
curl -fsSL https://raw.githubusercontent.com/Haugf/coconut-kitchen/main/bootstrap.sh | bash
```

It clones this repo to `~/mirror-src`, builds the server, installs the kiosk,
turns on auto-update, and reboots. After that the Pi checks GitHub every
minute and installs anything new pushed to `main`; the page reloads itself.
Anyone who can push to `main` can change what runs on the Pi, so keep push
access to yourself.

## Run it on the Mac first

```
make setup                  # go mod tidy, npm install, creates backend/config.json
# edit backend/config.json: add your calendar's iCal address
make dev-backend            # terminal 1, :8080
make dev-frontend           # terminal 2, open http://localhost:5173
```

Resize the browser window to a tall portrait shape to preview the layout.

**Calendar address:** Google Calendar on the web > Settings > pick the calendar >
"Secret address in iCal format". Add one URL per calendar to `icsUrls`.
This avoids OAuth entirely. Treat the URL like a password; config.json is gitignored.

**Weather:** Open-Meteo, no API key. Set latitude and longitude in config.json.

## Put it on the Pi

1. Flash **64-bit** Raspberry Pi OS with Raspberry Pi Imager (set WiFi, user, SSH).
2. Set `PI` in the Makefile (or `make deploy PI=you@raspberrypi.local`).
3. `make deploy`
4. On the Pi, once: `bash ~/mirror/setup-pi.sh` then `sudo reboot`.
   If the screen is upside down, rerun with `ROTATION=270 bash ~/mirror/setup-pi.sh`.

After that, every change is just `make deploy`. Logs: `make logs`.

The kiosk setup targets the default labwc desktop on current Pi OS. If your image
uses a different compositor, the autostart location and rotation command differ.

## Add a widget

1. Create `frontend/src/widgets/<name>/<Name>.tsx`.
2. If it needs external data or secrets, add a handler in `backend/` and wrap it
   in `cached` so the Pi never hammers an API.
3. Register it in `frontend/src/widgets/registry.ts`.
4. Add its id to one or more scenes in `frontend/src/scenes.ts`.

## Scenes

`scenes.ts` swaps which widgets show by time of day. Night dims the screen and
shows only the clock. Edit the hours and widget lists freely.

## Live messages (the v2 hook)

Anything on your network can put text on the mirror:

```
curl -X POST http://raspberrypi.local:8080/api/events \
  -H 'Content-Type: application/json' \
  -d '{"text":"Laundry is done","source":"Home","ttl":30}'
```

Set `eventsToken` in config.json to require `Authorization: Bearer <token>`.
This endpoint is how voice replies, smart home events, and timers will reach
the screen.

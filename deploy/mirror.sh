#!/usr/bin/env bash
# One command for everyday mirror chores. Run `mirror help` for the list.
set -euo pipefail

DIR="$HOME/mirror"
CONFIG="$DIR/config.json"
API="http://localhost:8080"

usage() {
  cat <<'EOF'
mirror status            is everything working? (add --json for JSON)
mirror off               turn the screen off now
mirror on                turn the screen back on
mirror sleep             show the nightly sleep hours
mirror sleep 00:30 07:00 set them (24-hour times)
mirror sleep never       keep the screen on all night
mirror say "text" [min]  put a message on the screen (default 10 minutes)
mirror home "address"    centre the minimap on this address
mirror style             show which page style is on
mirror style terrain     switch to the first design (or: kitchen)
mirror pin               is there a PIN for the phone settings page?
mirror pin ABCD          set it (letters or numbers, case doesn't matter)
mirror update            install the latest from GitHub now
mirror restart           restart the mirror server
mirror logs              follow the server log (Ctrl-C to stop)
mirror config            edit config.json, then restart
EOF
}

# Put the screen in standby or wake it, inside the desktop session (which
# SSH doesn't see). wlopm keeps the screen connected, so the browser
# carries on underneath.
power() {
  export XDG_RUNTIME_DIR="/run/user/$(id -u)"
  local sock
  sock=$(ls "$XDG_RUNTIME_DIR"/wayland-? 2>/dev/null | head -1 || true)
  if [ -z "$sock" ]; then
    echo "The desktop isn't running, so the screen can't be switched." >&2
    exit 1
  fi
  if ! command -v wlopm >/dev/null; then
    echo "wlopm isn't installed. Install it once with:  sudo apt install -y wlopm" >&2
    exit 1
  fi
  WAYLAND_DISPLAY=$(basename "$sock") wlopm "$1" '*'
}

restart() {
  if sudo -n systemctl restart mirror.service 2>/dev/null || sudo systemctl restart mirror.service; then
    echo "Restarted."
  fi
}

# Edit config.json with a small Python snippet; $1 is the code, the rest
# are its arguments (sys.argv[1:]).
edit_config() {
  local code="$1"; shift
  python3 - "$@" <<EOF
import json, sys
p = "$CONFIG"
c = json.load(open(p))
$code
json.dump(c, open(p, "w"), indent=2, ensure_ascii=False)
EOF
}

is_time() { [[ "$1" =~ ^([01]?[0-9]|2[0-3]):[0-5][0-9]$ ]]; }

cmd="${1:-help}"
shift || true

case "$cmd" in
  status)
    exec bash "$DIR/mirror-status.sh" "$@"
    ;;

  off)
    power --off
    echo "Screen off. 'mirror on' brings it back (or the schedule does in the morning)."
    ;;

  on)
    power --on
    echo "Screen on."
    ;;

  sleep)
    if [ $# -eq 0 ]; then
      curl -fsS "$API/api/sleep" && echo || echo "The server isn't answering. Try: mirror restart" >&2
    elif [ "$1" = "never" ]; then
      edit_config 'c["sleep"] = {"from": "", "to": ""}'
      echo "The screen will stay on all night."
      restart
    elif [ $# -eq 2 ] && is_time "$1" && is_time "$2"; then
      edit_config 'c["sleep"] = {"from": sys.argv[1], "to": sys.argv[2]}' "$1" "$2"
      echo "The screen will sleep from $1 to $2."
      restart
    else
      echo "Usage: mirror sleep 00:30 07:00   or   mirror sleep never" >&2
      exit 1
    fi
    ;;

  say)
    text="${1:-}"
    minutes="${2:-10}"
    if [ -z "$text" ]; then
      echo 'Usage: mirror say "Laundry is done" [minutes]' >&2
      exit 1
    fi
    token=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("eventsToken",""))' "$CONFIG" 2>/dev/null || true)
    body=$(python3 -c 'import json,sys; print(json.dumps({"text": sys.argv[1], "source": "Fred", "ttl": int(sys.argv[2]) * 60}))' "$text" "$minutes")
    curl -fsS -X POST "$API/api/events" -H 'Content-Type: application/json' \
      ${token:+-H "Authorization: Bearer $token"} -d "$body" >/dev/null
    echo "On the screen for $minutes minutes."
    ;;

  style)
    if [ $# -eq 0 ]; then
      python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("style") or "kitchen")' "$CONFIG"
    elif [ "$1" = "kitchen" ] || [ "$1" = "terrain" ]; then
      edit_config 'c["style"] = sys.argv[1]' "$1"
      echo "Switching to $1. The screen changes within a minute."
      restart
    else
      echo "Usage: mirror style kitchen   or   mirror style terrain" >&2
      exit 1
    fi
    ;;

  pin)
    if [ $# -eq 0 ]; then
      python3 -c 'import json,sys; p=(json.load(open(sys.argv[1])).get("settingsPin") or "").strip(); print("A PIN is set for the settings page." if p else "No PIN yet: the settings page stays locked. Set one with: mirror pin ABCD")' "$CONFIG"
    elif [[ "$1" =~ ^[A-Za-z0-9]{4,12}$ ]]; then
      edit_config 'c["settingsPin"] = sys.argv[1]' "$1"
      echo "PIN set. Phones that unlocked with an old PIN will need the new one."
      restart
    else
      echo "Usage: mirror pin ABCD   (4 to 12 letters or numbers)" >&2
      exit 1
    fi
    ;;

  home)
    if [ -z "${1:-}" ]; then
      echo 'Usage: mirror home "123 Example St, Ridgewood, NY 11385"' >&2
      exit 1
    fi
    edit_config 'c.setdefault("map", {})["home"] = {"address": sys.argv[1]}' "$1"
    rm -f "$DIR/map-home.json" "$DIR/map-streets.json"
    echo "Home set. The map looks it up when the server restarts."
    restart
    ;;

  update)
    bash "$HOME/mirror-src/update.sh"
    ;;

  restart)
    restart
    ;;

  logs)
    journalctl -u mirror.service -n 40 -f
    ;;

  config)
    "${EDITOR:-nano}" "$CONFIG"
    if python3 -c 'import json,sys; json.load(open(sys.argv[1]))' "$CONFIG" 2>/dev/null; then
      restart
    else
      echo "config.json has a mistake in it (not valid JSON). Not restarting; run 'mirror config' to fix it." >&2
      exit 1
    fi
    ;;

  help|-h|--help)
    usage
    ;;

  *)
    echo "Unknown command: $cmd" >&2
    usage >&2
    exit 1
    ;;
esac

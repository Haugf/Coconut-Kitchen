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
mirror google            show your Google speaker's timers (one-time sign-in)
mirror google off        sign out; timers stop showing
mirror cursor off        hide the mouse pointer on the screen (after a reboot)
mirror cursor on         bring it back
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

  google)
    # Timers set by voice live in Google; the speaker hands them to the Pi
    # only with a token from the Google account. This saves a sign-in for
    # that in google.json (only this user can read it) and installs the
    # helper that turns it into speaker tokens (glocaltokens, in a venv).
    auth="$DIR/google.json"
    if [ "${1:-}" = "off" ]; then
      rm -f "$auth"
      echo "Signed out of Google. Timers stop showing on the mirror."
      restart
      exit 0
    fi
    cat <<'EOF'
This shows the timers you set by voice ("Hey Google, set a pasta timer")
on the mirror.

The Pi keeps a Google sign-in token in ~/mirror/google.json. That token can
act as your Google account, so keep the Pi private. Undo it any time with
`mirror google off`; changing your Google password also cancels it.

Installing the helper (a minute or two the first time)...
EOF
    if [ ! -x "$DIR/venv/bin/python" ]; then
      python3 -m venv "$DIR/venv" || { echo "Couldn't make a Python venv. Install it once with:  sudo apt install -y python3-venv" >&2; exit 1; }
    fi
    "$DIR/venv/bin/pip" install -q --upgrade glocaltokens
    cat <<'EOF'

Now, on your Mac (a private window is easiest):
  1. Open https://accounts.google.com/EmbeddedSetup and sign in.
  2. Click "I agree". The page may spin forever; that's fine.
  3. Open the developer tools (Cmd-Option-I) › Application › Cookies ›
     https://accounts.google.com, and copy the value of "oauth_token"
     (it starts with oauth2_4/).

EOF
    read -rp "Google account email: " email
    read -rsp "oauth_token (hidden as you paste): " oauth_token
    echo
    ( umask 077
      "$DIR/venv/bin/python" - "$auth" "$email" "$oauth_token" <<'PY'
import json, os, secrets, sys
import gpsoauth
path, email, token = sys.argv[1], sys.argv[2].strip(), sys.argv[3].strip()
android_id = secrets.token_hex(8)
try:
    r = gpsoauth.exchange_token(email, token, android_id)
except Exception as e:
    sys.exit("Couldn't reach Google: %s" % type(e).__name__)
master = r.get("Token", "")
if not master.startswith("aas_et/"):
    sys.exit("Google didn't accept that (%s). Get a fresh oauth_token and try again; each one works once." % (r.get("Error") or "no token"))
tmp = path + ".new"
with open(tmp, "w") as f:
    json.dump({"email": email, "masterToken": master, "androidId": android_id}, f)
os.replace(tmp, path)
from glocaltokens.client import GLocalAuthenticationTokens
devices = GLocalAuthenticationTokens(username=email, master_token=master, android_id=android_id).get_google_devices(disable_discovery=True)
names = [d.device_name for d in devices if d.local_auth_token]
print("Signed in. Speakers on this account: " + (", ".join(names) or "none found"))
PY
    ) || exit 1
    echo "Restarting the mirror; a timer shows up a few seconds after you set it."
    restart
    ;;

  cursor)
    # The page hides the pointer, but Chromium only applies that once the
    # mouse moves, so after a restart the arrow sits on the screen. A blank
    # cursor theme for the desktop hides it from the start. No root needed.
    env_file="$HOME/.config/labwc/environment"
    case "${1:-}" in
      off)
        python3 - "$HOME/.icons/mirror-blank" <<'PY'
import os, struct, sys
d = os.path.join(sys.argv[1], "cursors")
os.makedirs(d, exist_ok=True)
# One fully transparent 1x1 Xcursor image.
image = struct.pack("<9I", 36, 0xFFFD0002, 24, 1, 1, 1, 0, 0, 0) + struct.pack("<I", 0)
data = b"Xcur" + struct.pack("<3I", 16, 0x10000, 1) + struct.pack("<3I", 0xFFFD0002, 24, 28) + image
with open(os.path.join(d, "default"), "wb") as f:
    f.write(data)
for name in ["left_ptr", "arrow", "top_left_arrow", "text", "xterm", "pointer", "hand1", "hand2",
             "watch", "wait", "progress", "left_ptr_watch", "crosshair", "grab", "grabbing",
             "move", "not-allowed", "help", "col-resize", "row-resize"]:
    link = os.path.join(d, name)
    if not os.path.lexists(link):
        os.symlink("default", link)
with open(os.path.join(sys.argv[1], "index.theme"), "w") as f:
    f.write("[Icon Theme]\nName=mirror-blank\n")
PY
        mkdir -p "$(dirname "$env_file")"
        touch "$env_file"
        sed -i '/^XCURSOR_THEME=/d' "$env_file"
        echo "XCURSOR_THEME=mirror-blank" >> "$env_file"
        echo "The pointer will be hidden after the next reboot (sudo reboot)."
        ;;
      on)
        [ -f "$env_file" ] && sed -i '/^XCURSOR_THEME=mirror-blank$/d' "$env_file"
        echo "The pointer comes back after the next reboot (sudo reboot)."
        ;;
      *)
        echo "Usage: mirror cursor off   or   mirror cursor on" >&2
        exit 1
        ;;
    esac
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

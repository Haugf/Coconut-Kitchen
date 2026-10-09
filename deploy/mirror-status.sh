#!/usr/bin/env bash
# One look at whether the mirror is healthy, without walking to the screen.
#   mirror-status          a checklist
#   mirror-status --json   the same, as JSON (used by the status report)
set -uo pipefail

MIRROR="$HOME/mirror"
SRC="$HOME/mirror-src"
STATUS=$(curl -s --max-time 25 localhost:8080/api/status || true)

server_active=$(systemctl is-active mirror 2>/dev/null)
browser=$(pgrep -c -x chromium 2>/dev/null | head -1)
browser=${browser:-0}
installed=$(cat "$MIRROR/.installed" 2>/dev/null || echo "")
latest=$(git -C "$SRC" ls-remote -q origin main 2>/dev/null | cut -f1)
update_result=$(systemctl show mirror-update --property=Result --value 2>/dev/null)
update_last=$(journalctl -u mirror-update -n 1 --no-pager -o cat 2>/dev/null | tail -1)
display=$(cat "$MIRROR/display.env" 2>/dev/null | tr '\n' ' ')
temp=$(vcgencmd measure_temp 2>/dev/null | cut -d= -f2)
throttled=$(vcgencmd get_throttled 2>/dev/null | cut -d= -f2)
errors=$(journalctl -u mirror --since "-30 min" --no-pager -o cat 2>/dev/null | grep -Ei 'error|fail|returned [45]' | tail -5)

export server_active browser installed latest update_result update_last display temp throttled errors STATUS
python3 - "$@" <<'PY'
import json, os, sys
e = os.environ
try:
    st = json.loads(e["STATUS"]) if e["STATUS"] else {}
except Exception:
    st = {}

checks = []
def check(ok, label, detail=""):
    checks.append({"ok": bool(ok), "label": label, "detail": detail})

check(e["server_active"] == "active", "Server running", e["server_active"])
check(int(e["browser"] or 0) > 0, "Browser on screen", f'{e["browser"]} chromium processes')

page = st.get("page") or {}
ago = page.get("secondsAgo")
check(ago is not None and ago < 180, "Page checking in",
      "never" if ago is None else f"{ago}s ago")
check(page.get("build") and page.get("build") == st.get("serverBuild"), "Screen shows the installed build",
      f'screen {page.get("build")}, installed {st.get("serverBuild")}')
inst, latest = e["installed"], e["latest"]
check(inst and inst == latest, "Up to date with GitHub",
      f"installed {inst[:7] or '?'}, GitHub {latest[:7] or '?'}")
check(e["update_result"] in ("success", ""), "Last auto-update", f'{e["update_result"]}: {e["update_last"]}')

cal = st.get("calendar") or {}
counts = cal.get("eventsThisWeek")
failed = cal.get("failed") or []
check(counts is not None and not failed, "Calendars",
      (", ".join(f"{k or 'calendar'}: {v} events" for k, v in (counts or {}).items()) or cal.get("health", {}).get("error", "no data"))
      + (f"; failing: {', '.join(failed)}" if failed else ""))
w = st.get("weather") or {}
check(w.get("ok"), "Weather", w.get("error", ""))
for row in st.get("transit") or []:
    mins = ", ".join(str(m) for m in row.get("minutes", [])) or "none soon"
    check(row.get("ok"), f'{row.get("route")} to {row.get("label")}', row.get("error") or f"{mins} min")
mp = st.get("map") or {}
check(mp.get("streets", 0) > 0, "Minimap",
      f'{mp.get("streets", 0)} streets, {mp.get("vehicles", 0)} trains and buses'
      + ("" if mp.get("homeSet") else ", home not set (centred on Forest Av)"))
seen = page.get("seen") or {}
check(seen.get("transitRows", 0) > 0, "Transit visible on screen", f'{seen.get("transitRows", 0)} rows')
check(e["throttled"] in ("0x0", ""), "Power", f'throttled={e["throttled"]}, {e["temp"]}')

report = {"now": st.get("now"), "display": e["display"].strip(), "checks": checks,
          "recentErrors": [l for l in e["errors"].splitlines() if l]}

if "--json" in sys.argv:
    print(json.dumps(report, indent=2))
else:
    for c in checks:
        print(("  ✓ " if c["ok"] else "  ✗ ") + c["label"] + (f'  ({c["detail"]})' if c["detail"] else ""))
    print(f'  · Display: {report["display"] or "default"}')
    if report["recentErrors"]:
        print("\nRecent errors:")
        for l in report["recentErrors"]:
            print("  " + l)
PY

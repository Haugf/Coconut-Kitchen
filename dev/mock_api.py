#!/usr/bin/env python3
"""Serve the built page (frontend/dist) with sample data, so you can see
the mirror without a Pi, API keys or network access.

    python3 dev/mock_api.py                 # transit and minimap, like the Pi
    python3 dev/mock_api.py --with sun      # add your widget after them
    python3 dev/mock_api.py --only sun      # just your widget

Then open http://127.0.0.1:8099. Every widget's data comes from its own
sample.json; calendar and weather come from the samples below.
Run `npm run build` in frontend/ first, and again after each change.
"""
import argparse
import datetime
import http.server
import json
import os

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..")
DIST = os.path.join(ROOT, "frontend", "dist")
WIDGETS = os.path.join(ROOT, "frontend", "src", "widgets")

ap = argparse.ArgumentParser()
ap.add_argument("--with", dest="extra", action="append", default=[], help="add a widget after the defaults")
ap.add_argument("--only", action="append", default=[], help="show only these widgets")
ap.add_argument("--port", type=int, default=8099)
args = ap.parse_args()


def load_widgets():
    found = {}
    for folder in sorted(os.listdir(WIDGETS)):
        mpath = os.path.join(WIDGETS, folder, "widget.json")
        if folder.startswith("_") or not os.path.exists(mpath):
            continue
        manifest = json.load(open(mpath))
        spath = os.path.join(WIDGETS, folder, "sample.json")
        sample = json.load(open(spath)) if os.path.exists(spath) else {}
        found[manifest["id"]] = (manifest, sample)
    return found


now = datetime.datetime.now().astimezone()


def at(h, m=0):
    return now.replace(hour=h, minute=m, second=0, microsecond=0).isoformat()


CORE = {
    "/api/weather": {"temp": 58, "feelsLike": 56, "code": 1, "units": "fahrenheit",
                     "days": [{"date": now.date().isoformat(), "high": 64, "low": 51, "code": 1, "rainChance": 5}]},
    "/api/calendar": {"people": ["Fred", "Ally"], "events": [
        {"title": "Laundry", "start": at(9), "end": at(10), "allDay": False, "who": ["Fred"]},
        {"title": "Dinner", "start": at(19), "end": at(21), "allDay": False, "who": ["Fred", "Ally"]}]},
    "/api/sleep": {"asleep": False, "from": "01:00", "to": "06:30"},
}


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *a, **k):
        super().__init__(*a, directory=DIST, **k)

    def log_message(self, *a):
        pass

    def send_json(self, obj, code=200):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        self.send_response(204)
        self.end_headers()

    def do_GET(self):
        path = self.path.split("?")[0]
        widgets = load_widgets()  # re-read so edits to sample.json show on reload
        enabled = args.only or (["transit", "minimap"] + args.extra)

        if path in CORE:
            return self.send_json(CORE[path])
        if path == "/api/widgets":
            out = []
            for wid in enabled:
                if wid not in widgets:
                    print(f"No widget called {wid!r}; skipping it")
                    continue
                m, _ = widgets[wid]
                settings = {k: v.get("default") for k, v in (m.get("settings") or {}).items()}
                out.append({"id": wid, "name": m["name"], "width": m.get("width", "fill"), "settings": settings,
                            "sources": {n: f"/api/w/{wid}/{n}" for n in (m.get("data") or {})}})
            return self.send_json(out)
        if path.startswith("/api/w/"):
            parts = path.split("/")
            if len(parts) == 5 and parts[3] in widgets and parts[4] in widgets[parts[3]][1]:
                return self.send_json(widgets[parts[3]][1][parts[4]])
            return self.send_json({"error": "no sample for this source"}, 404)
        if path.startswith("/api/"):
            return self.send_json({"error": "not in the mock"}, 404)
        if path == "/" or not os.path.exists(os.path.join(DIST, path.lstrip("/"))):
            self.path = "/index.html"
        return super().do_GET()


print(f"Serving the mirror with sample data on http://127.0.0.1:{args.port}")
http.server.ThreadingHTTPServer(("127.0.0.1", args.port), Handler).serve_forever()

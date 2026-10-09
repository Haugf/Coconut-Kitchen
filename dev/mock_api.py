#!/usr/bin/env python3
"""Serve frontend/dist with sample API data, for checking the page without
network access to the MTA, Google or OpenStreetMap.

    python3 dev/mock_api.py      # then open http://127.0.0.1:8099
"""
import http.server, json, datetime, os, sys
DIST=os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "frontend", "dist")
now=datetime.datetime.now().astimezone()
def iso(h,m=0,days=0):
    return (now.replace(hour=h,minute=m,second=0,microsecond=0)+datetime.timedelta(days=days)).isoformat()
st={"L17":(40.699814,-73.911586),"L19":(40.695602,-73.904084),"L20":(40.688764,-73.904046),"L16":(40.703811,-73.918425),"L15":(40.706607,-73.922913),"L14":(40.706152,-73.933147),"L13":(40.707739,-73.93985),"L21":(40.682829,-73.905249),
"M01":(40.711396,-73.889601),"M04":(40.706186,-73.895877),"M05":(40.704423,-73.903077),"M06":(40.702762,-73.90774),"M08":(40.69943,-73.912385),"M09":(40.698664,-73.919711),"M10":(40.697857,-73.927397),"M11":(40.697207,-73.935657)}
ll=lambda k:{"lat":st[k][0],"lon":st[k][1]}
def lerp(a,b,f): return {"lat":st[a][0]+(st[b][0]-st[a][0])*f,"lon":st[a][1]+(st[b][1]-st[a][1])*f}
home={"lat":40.7044,"lon":-73.9031}  # a sample centre near Forest Av, not anyone's home
# Rough street lines for a render check only; the Pi fetches real ones.
streets=[[ll("M08"),ll("M06"),ll("M05"),ll("M04")],
 [{"lat":40.6985,"lon":-73.9175},{"lat":40.6996,"lon":-73.9118},{"lat":40.7045,"lon":-73.8920}],
 [{"lat":40.7070,"lon":-73.9170},{"lat":40.6930,"lon":-73.9040}],
 [{"lat":40.7120,"lon":-73.8990},{"lat":40.6990,"lon":-73.8935}],
 [{"lat":40.7085,"lon":-73.9080},{"lat":40.6975,"lon":-73.8990}],
 [{"lat":40.7000,"lon":-73.9150},{"lat":40.7055,"lon":-73.8890}]]
data={
 "/api/map":{"center":home,"home":home,"radiusMeters":1300,"streets":streets,
  "lines":[{"route":"L","points":[ll(k) for k in ["L13","L14","L15","L16","L17","L19","L20","L21"]]},{"route":"M","points":[ll(k) for k in ["M11","M10","M09","M08","M06","M05","M04","M01"]]}],
  "stations":[{"name":"Myrtle–Wyckoff",**ll("L17")},{"name":"Forest Av",**ll("M05")},{"name":"Fresh Pond Rd",**ll("M04")}],
  "vehicles":[
   {"id":"m1","kind":"subway","route":"M","minutes":4,"state":"moving","from":ll("M04"),**lerp("M05","M04",0.55)},
   {"id":"m2","kind":"subway","route":"M","minutes":16,"state":"waiting","note":"hasn't left Metropolitan Av",**ll("M01")},
   {"id":"l1","kind":"subway","route":"L","minutes":6,"state":"stopped",**ll("L19")},
   {"id":"l2","kind":"subway","route":"L","minutes":2,"state":"moving","from":ll("L16"),**lerp("L17","L16",0.4)},
   {"id":"l3","kind":"subway","route":"L","minutes":13,"state":"far",**ll("L21")}]},
 "/api/transit":{"updated":now.isoformat(),"rows":[
   {"kind":"subway","route":"L","label":"Manhattan","stopName":"Myrtle–Wyckoff","minutes":[6,13],"ok":True},
   {"kind":"subway","route":"L","label":"Canarsie","stopName":"Myrtle–Wyckoff","minutes":[2,10],"ok":True},
   {"kind":"subway","route":"M","label":"Manhattan","stopName":"Forest Av","minutes":[4,16],"ok":True}]},
 "/api/weather":{"temp":58,"feelsLike":56,"code":1,"units":"fahrenheit","days":[{"date":now.date().isoformat(),"high":64,"low":51,"code":1,"rainChance":5}]},
 "/api/sleep":{"asleep":False,"from":"01:00","to":"06:30"},
 "/api/calendar":{"people":["Fred","Ally"],"events":[
   {"title":"Laundry","start":iso(9),"end":iso(10),"allDay":False,"who":["Fred"]},
   {"title":"Dinner","start":iso(19),"end":iso(21),"allDay":False,"who":["Fred","Ally"]}]},
}
class H(http.server.SimpleHTTPRequestHandler):
    def __init__(s,*a,**k): super().__init__(*a,directory=DIST,**k)
    def log_message(s,*a): pass
    def do_POST(s): s.send_response(204); s.end_headers()
    def do_GET(s):
        p=s.path.split("?")[0]
        if p in data:
            b=json.dumps(data[p]).encode(); s.send_response(200); s.send_header("Content-Type","application/json"); s.end_headers(); s.wfile.write(b); return
        if p.startswith("/api/"): s.send_response(404); s.end_headers(); return
        if not os.path.exists(DIST+p) or p=="/": s.path="/index.html"
        super().do_GET()
print("Serving the mirror with sample data on http://127.0.0.1:8099")
http.server.ThreadingHTTPServer(("127.0.0.1",8099),H).serve_forever()

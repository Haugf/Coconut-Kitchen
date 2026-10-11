# Tides

Today's tide as a filled curve, the way a printed tide table draws it,
with now marked in clay, the next high or low in a sentence, and the
day's turns underneath. Predictions come from NOAA (no key needed).

## Settings

| Setting | What it does | Default |
|---|---|---|
| `station` | NOAA tide station id. Search for one at tidesandcurrents.noaa.gov. | `8518750` (The Battery, NY) |
| `place` | What to call it on screen | `The Battery` |

Times are the station's local time, so the "now" marker lines up when the
station shares the mirror's time zone.

## Turn it on

In `~/mirror/config.json` on the Pi:

```json
"widgets": [
  { "id": "transit" },
  { "id": "minimap" },
  { "id": "tides" }
]
```

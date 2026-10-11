# Sun

Today's sunrise and sunset, from Open-Meteo (no key needed).

## Settings

| Setting | What it does | Default |
|---|---|---|
| `latitude` | Where you are, north–south | 40.7044 |
| `longitude` | Where you are, east–west | -73.9018 |

## Turn it on

In `~/mirror/config.json` on the Pi:

```json
"widgets": [
  { "id": "transit" },
  { "id": "example", "settings": { "latitude": 40.70, "longitude": -73.90 } }
]
```

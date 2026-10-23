# The forest

A plan, not code yet. This captures where the creature idea landed on
2026-10-11, so it can be picked up later.

## Two channels, not one screen

The idea started as a creature walking onto the mirror. It split into two
things that want different things from a screen:

| | The mirror | The forest |
|---|---|---|
| What it's for | A glance at the day | Who's here right now |
| Feel | Calm, but alive | Playful, social |
| Look | Terrain style, as it is | 32-bit pixel forest with a giant tree |
| Characters | None | One animal per phone |
| Where it lives | This repo, the 24" monitor | Its own screen (second display, a small Pi by the door, or a scene) |

**The mirror stays a glance**, with small motion so it feels alive (the
"now" dot drifting along the ridge, trains gliding on the minimap, a soft
change at sunset). No creatures on it.

**The forest is a living guest list.** Each person's phone has an animal.
When the phone is home, the animal runs around the giant tree; when it
leaves, the animal walks off the edge. When friends visit, their animals
show up too. Ambient, not needy: nothing gets hungry or sad.

Open `prototype.html` in a browser to see it: tap a name to bring that
animal home or send it away.

## Designing the animals

`creature-sheet.png` is a phone-sized sheet to draw on (Photos > Markup or
Procreate): the main look, walking, arriving, rain, train soon, asleep at
night, in your pocket, plus colors that read by day and on black at night.
One sheet per animal.

## How it knows who's here: AWS IoT Core

Chosen because it's the most interesting path, not the shortest. The phone
never talks to the screen directly; it changes a shadow in AWS, and the
screen reacts.

```
phone (Shortcuts) --HTTPS + token--> Lambda (function URL)
                                       | update desired state
                                       v
                              IoT Core named shadow per animal
                                       | delta over MQTT
                                       v
                       bridge on the forest's Pi (device cert)
                                       | local call
                                       v
                          forest page: the animal runs in
```

- **Arrive:** a Shortcuts automation on joining home Wi-Fi.
- **Leave:** a "when I leave" location automation. It works because the
  phone talks to AWS, not to the Pi.
- **Shadow:** `desired.location` (home/away) set by the phone, then
  `reported.location` set by the bridge once the animal has moved, so the
  shadow always shows the truth.

### Phases

0. **The forest, local only.** The pixel scene and a local endpoint to
   bring an animal in or out. Buttons, like the prototype.
1. **The Pi becomes a Thing.** Terraform: a Thing, a certificate, a policy
   scoped to its own topics. Cert and key live on the Pi, never in git. A
   small bridge subscribes to the shadow delta. Milestone: flip the shadow
   from the AWS console's MQTT test client and an animal runs onto the
   real screen.
2. **Phones.** One Lambda with a function URL and a per-person token; two
   Shortcuts automations per phone.
3. **Everyone.** One shadow per animal (Fred, Ally, then guests). The
   Lambda also answers "where's my animal" for a phone page while away.

### Decisions to keep

- **Separate bridge program.** MQTT needs a library (Paho); the mirror's
  Go server stays at one dependency. If the bridge is down, nothing else
  breaks.
- **Offline fallback.** If AWS or the internet is down, the forest keeps
  showing whoever was last home.
- **Privacy.** The repo is public: no certs, tokens or locations
  committed. Payloads carry only home/away and mood.
- **Cost.** IoT Core bills per connection-minute and message; tiny at this
  scale. Set a billing alarm anyway.

## Still open

- **Guests:** does a friend's animal appear automatically when their phone
  joins the Wi-Fi, or do they scan a QR code by the door?
- **The forest's screen:** second monitor, small display by the door, or a
  scene the mirror switches to?
- **Repo:** does the forest live here or in its own repo?
- **The animals:** waiting on drawings.

# Coconut Kitchen for iPhone

A small SwiftUI app with Home Screen and Lock Screen widgets for the
mirror. It reads the Pi's existing API, so nothing changes on the Pi.

- **Today widget** (small, medium, large, Lock Screen): the mirror's
  headline ("A climb to Dinner at 7, then the day opens up."), the day as
  a line from 6 AM to midnight (your line solid, the other dashed, a dot
  at now), and what's next.
- **Send to Mirror widget** (small, medium): one-tap messages that show
  on the mirror for 20 seconds, signed with your name. The pencil opens
  the app to write anything.
- **The app**: today and tomorrow, a line to write on the mirror (with
  "say it out loud" on the speaker), and settings.
- **Siri and Shortcuts**: "Send to the mirror in Coconut Kitchen".

It only talks to the mirror on the home Wi-Fi. Away from home the app and
widgets show the last thing the mirror said, and the Send buttons say
they couldn't reach it.

## Build it

On the Mac, with Xcode 16 or newer:

```
brew install xcodegen
cd ios
xcodegen
open CoconutKitchen.xcodeproj
```

`project.yml` is the source of truth; the `.xcodeproj` is generated and
ignored by git. Rerun `xcodegen` after adding files.

1. Select the **CoconutKitchen** target, then Signing & Capabilities, and
   pick your team (or set `DEVELOPMENT_TEAM` in `project.yml`). Do the
   same for **MirrorWidgets**. Xcode registers the App Group
   `group.com.haugf.coconutkitchen` the first time.
2. Plug in your iPhone, choose it as the run destination, and press Run.
3. The first launch asks for the local network: allow it. Then pick who
   this phone belongs to.
4. Long-press the Home Screen, tap Edit, Add Widget, and search for
   Coconut Kitchen.

## Ally's phone: TestFlight

With the paid Apple Developer account:

1. In App Store Connect, create an app with bundle ID
   `com.haugf.coconutkitchen`.
2. In Xcode: Product, Archive, then Distribute App, TestFlight Internal
   Only.
3. In App Store Connect, add Ally as an internal tester (they need to be
   a user on your team, role Developer or Marketing is enough). They get
   an email, install TestFlight, and tap Install.

Builds last 90 days; archive and upload again when you change things.
Internal testing skips App Review.

## Settings

| Setting | Default | |
|---|---|---|
| This phone is | (ask on first launch) | the names come from the Pi's calendar people |
| One-tap messages | Dinner's ready, Home soon, Love you | one per line; the widget shows the first 2 (small) or 4 (medium) |
| Mirror | `mirror.local:8080` | the Pi's address on the home Wi-Fi; an IP works too |
| Token | empty | only if `eventsToken` is set in the Pi's `config.json` |

## How it works

```
Shared/   used by both the app and the widgets
  Models.swift             the Pi's JSON (calendar, weather, messages)
  MirrorAPI.swift          GET /api/calendar, /api/weather; POST /api/events
  Store.swift              settings and the last good snapshot (App Group)
  Day.swift                Swift copy of terrain/day.ts, shape.ts, voice.ts
  Theme.swift              the kitchen palette and the day line
  SendToMirrorIntent.swift the App Intent behind the Send buttons and Siri
App/      the app: one screen, a compose bar, settings
Widgets/  Today and Send to Mirror
```

- Like the Pi's `cached[T]`, a failed fetch keeps the last good data.
  Calls time out after 6 seconds so a widget refresh off the home Wi-Fi
  gives up quickly.
- The Today widget asks for a new timeline every 30 minutes (iOS decides
  the real schedule) and has entries every 15 minutes, so the now dot
  moves and past events drop off without the network.
- `Day.swift` mirrors the frontend's headline and day-line logic. If
  `voice.ts` or `shape.ts` change, change it too.
- While the haunted party page is on, the Pi won't hand out the calendar;
  the phone keeps showing what it had.
- Nothing about the household is in this folder: the address, name and
  token live in the phone's settings.

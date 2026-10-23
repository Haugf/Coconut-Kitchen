# Drawing for the mirror

Ally made the haunted page what it is. This is how a drawing gets from
Ally's sketchbook onto the mirror and the phone widgets, with as little
fuss as possible.

## What to draw

Anything that should appear on the screen: a new visitor for the haunted
grounds (the bats, a ghost, a pumpkin rolling down the hill), a creature,
an icon for a phone widget, a whole new style for the page.

The haunted visitors are **solid black silhouettes** against the night,
so they read from across the kitchen. One shape, no outlines or shading.
The one exception so far is the red balloon.

## How to hand it over

1. Draw it however is comfortable (Procreate, Illustrator, Figma, paper).
2. Export **SVG** if the app can (Illustrator, Figma, Affinity, and
   Procreate's vector export all can). Paper or a PNG is fine too; it
   just gets traced, and some detail may soften.
3. For anything that moves, draw the poses separately: wings up and wings
   down, walking and stopping. Two or three poses are plenty. Say how it
   moves ("flutters out of the bell tower", "creeps left, stops to stare").
4. Drop the files in a GitHub issue, or send them to Fred.

From there, Fred or Claude turns it into code. Each visitor is a few
paths in `frontend/src/widgets/haunted/Visitors.tsx`, animated with CSS in
`haunted.css`, so the Pi stays cool. The phone widgets use the same
shapes as vector images in `ios/Assets.xcassets`.

## Sizes, roughly

- **Visitors:** the grounds along the bottom are 1200 by 200. A zombie is
  about 50 tall, a bat about 24 wide. Put the feet (or the middle, for
  fliers) at the bottom center of the drawing.
- **Widget icons:** square, simple enough to read at the size of a
  fingertip.

The palette is closed, so new colors need a reason, but a reason from the
artist counts.

# Thin blocks and flight corrections

Rails now use one alpha-tested surface, with the Bedrock `rail_direction`
controlling straight, curved and ascending geometry. All ten normal-rail states
and the six straight/ascending states for powered, detector and activator rails
are covered. `rail_data_bit` selects the powered texture. Bedrock's `blocks.json`
uses up/down texture slots for these variants; those slots are no longer treated
as cube faces. Existing pack textures and explicit custom geometry retain priority.

Ladders use one wall-mounted alpha-tested plane with full-face UVs, oriented by
`facing_direction`. Nether portals use a centered quarter-block-thick transparent
slab along `portal_axis`, rather than a full cube. The shared scene mesh feeds
the map viewer, Render editor and GLB export.

Stairs honor all five `minecraft:corner` states, both halves and all four
directions, using Dragonfly's vanilla stair bounds. Older palettes without a
corner state derive inner and outer corners from neighboring stairs, including
continuation checks. Buttons use their small mounted shape and pressed depth;
levers use a stone base and tilted handle. Tripwire hooks use a small wooden
mount and alpha-tested hook texture with attached/powered tilt.

Item frames have thin housing and persist their block-entity item data. Block
items use scaled geometry; ordinary items use their pack icon. Item rotation
and byte/bool block states survive JSON cache storage. Cake height and bite
count affect its mesh, including the exposed cut texture. Sea-pickle clusters
contain one to four small bodies with separate cap and side UV regions.

Frame contents distinguish inventory icons from world geometry: glass panes
use a full transparent sprite, while block items use a small GUI-style 3D view.
Commented item metadata, dye aliases, full HD icon UVs and upright mounting on
all walls are supported. Bedrock degree rotations persist explicitly through
the scene cache. The captured Hive police car was checked with its actual
orange panes and pack textures.

Doors use a 3/16-block slab, separate upper/lower textures, wood variants and
open/hinge orientation. The two halves share their authoritative state when
both are available. Azaleas use a raised canopy and narrow cutout stem, with
the model-part texture slots separated from the potted-plant slots. Bubble
columns use the water material and shared fluid-face culling, with small,
deterministically placed transparent bubble sprites. Sprites are static in the
scene mesh and GLB export. Existing explicit pack geometry still takes priority.

`TestCapturedPoliceCarGallery` provides a read-only optional captured-world
fixture via `CRAFT_BOX_FRAME_CAPTURE`, `CRAFT_BOX_FRAME_GALLERY_OUT` and
`CRAFT_BOX_3D_ASSETS`. The thin-model gallery also covers doors, azaleas, water
columns, panes and dyes. `tmp/frame-item-check/verify.py` renders both fixtures
using the actual Render editor engine.

Flight defaults to 6 blocks/second, with a 2.5x Shift boost, lower mouse
sensitivity and adjustable smoothing. Velocity uses exponential acceleration
and braking with integrated displacement; mouse deltas ease out separately.
Focus loss, teleporting and camera changes clear pending motion. Releasing the
right mouse button ends mouse look without cancelling movement keys. Cinematic
navigation retains its previous behavior unless smoothing is explicitly enabled.

Escape switches to orbit controls at the current flight position, preserving
the perspective camera and view direction. The orbit pivot follows the visible
surface under the center of the view, with a forward-distance fallback for sky.
Reentering flight also keeps the position. Initial entry from an orthographic
view matches perspective FOV to the previous scale at the orbit pivot and
updates the flight FOV control. Pending orbit damping is consumed
without changing the pose. Return to start remains an explicit navigation action.
Orthographic zoom no longer widens the panoramic sky projection.

Explicit View presets use orthographic projection; the 2D map temporarily uses
an orthographic camera and restores the previous 3D camera on return. Saved views
include projection and orthographic span; older documents retain their default
orthographic behavior.

The First person button enters flight and requests mouse capture directly within
the click gesture. The compact bottom toolbar exposes speed and an expandable
settings panel; Tab releases the cursor for controls without exiting flight.

Regression tests cover geometry, rail texture variants, curve endpoints, ladder
wall placement, portal axes, motion easing and orthographic sky projection.
`TestRailLadderPortalGallery` creates a real-asset scene page and GLB when
`CRAFT_BOX_THIN_GALLERY_OUT` and `CRAFT_BOX_3D_ASSETS` are set.
`tmp/thin-block-check/verify.py` renders the scene on a real GPU, flies far away,
presses Escape and compares the orbit frame against the immediately preceding
flight frame. Repeated switching, 2D map round trips, saved perspective views and
deferred exits during export are checked as well.

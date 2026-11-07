# Desktop UI development

The desktop GUI uses Wails v2, React, TypeScript and shadcn/ui (Base UI).
The concrete layout and colour decisions are in [wails-ui-plan.md](wails-ui-plan.md).
The Go command handlers remain the source of truth for options and defaults;
the GUI reads their schema rather than maintaining a separate flag list.

## Where to edit

| File | Responsibility |
| --- | --- |
| `ui/desktop/frontend/src/App.tsx` | Navigation, source selection, command forms, account, results and logs |
| `ui/desktop/frontend/src/workspace.css` | Layout, responsive breakpoints and workspace styling |
| `ui/desktop/frontend/src/index.css` | Dark/light colour tokens, fonts and shadcn theme |
| `ui/desktop/frontend/src/components/ui/` | Editable shadcn components |
| `ui/desktop/frontend/src/WorldMap.tsx` | Pixel-sharp live terrain, player position, pan and zoom |
| `ui/desktop/frontend/src/bridge.ts` | Typed Wails calls and events |
| `ui/desktop/app.go`, `sources.go` | Command lifecycle, snapshots, source lookup and authentication |
| `ui/desktop/shell.go` | Native window, embedded assets, file picker, links and shutdown |

The main UI uses in-process Wails IPC. The independent map window runs the
same EXE as a child and receives snapshots through a bearer-protected
HTTP endpoint bound to a random localhost port. It has no capture or account
controls. Its only editing operation registers a named world region; see
[world-region-editor.md](world-region-editor.md) and
[map-window-api.yaml](map-window-api.yaml). Fonts and frontend
assets are bundled in the EXE.

## Build and test

Use Go 1.26 and Node 22. Windows requires Microsoft Edge WebView2 Runtime;
Linux requires GTK 3 and WebKitGTK 4.1 development packages. From the repository:

```powershell
cd ui/desktop/frontend
npm ci
npm test
npm run build
cd ../../..
go test ./ui/desktop ./utils/auth ./utils/proxy ./handlers/worlds
go run ./build.go windows gui,cli
```

`build.go` runs `npm ci` and `npm run build` automatically for desktop GUI builds,
then embeds `frontend/dist`. Its existing `builds` directory is regenerated.
`npm run dev` can preview frontend edits, but native operations require the
compiled desktop app; it is not a complete standalone browser application.

Portable GUI builds also bundle the complete locally prepared `assets/vanilla-3d`
tree. Prepare it with `tools/prepare-3d-assets.py` before building; `build.go`
generates `assets/bundle/vanilla-3d.zip` automatically. The ZIP and original
Mojang runtime data remain ignored by Git. For direct `go build`, run
`go run ./tools/bundle-assets` first. A missing source fails the standard GUI
build rather than silently shipping a textureless portable installation.

On launch the GUI restores missing packaged files to `assets/vanilla-3d` beside
the EXE in the background. Changed bundles replace stock files atomically under
an OS process lock. Rendering uses an embedded vanilla base immediately; local
overrides and captured resource packs retain priority. A partially deleted or
unwritable installation folder therefore cannot hide embedded textures.

`GetBootstrap` includes `installationId`, an opaque SHA-256 identifier of the
normalized executable directory. Account-onboarding completion is stored under
that scope, so a new portable folder shows the guide again when signed out.
Renaming or updating the executable in the same folder preserves completion.
Existing unscoped completion flags do not suppress a new scoped installation.
Signed-in users and secondary map windows continue to skip the account guide.

For a standalone build with the checked-in official dependency snapshots:

```powershell
$env:GOWORK = 'off'
go run ./tools/bundle-assets
go build -mod=readonly -tags 'gui,production' -ldflags '-H=windowsgui -X github.com/Daniel-Ric/CraftBox/utils.CmdName=CraftBox-gui -X github.com/Daniel-Ric/CraftBox/utils.Version=1.62.0-wails-dev' -o builds/CraftBox-wails-gui.exe ./cmd/CraftBox
```

Versions ending in `-dev` skip automatic update checks. Explicit update actions
use this repository's releases. Keep `CmdName=CraftBox-gui` for GUI assets.

## Preserved workflows and verification

Worlds, resource packs and skins retain all registered command options, including
advanced settings and capture replay. Realms, events, featured servers and
friends' worlds use the existing backend services. Device-code login, named
accounts, cancellation, update actions, progress, result folders, logs and live
map controls are available in the new interface. CLI behaviour is preserved.
Android retains its existing Gio UI because Wails v2 targets desktop platforms.

The world session automatically opens an independent, resizable native map
window, initially 1280 by 760 pixels. Once the local listener is ready, Minecraft is opened once with
`minecraft://connect?serverUrl=<address>&serverPort=<port>`; wildcard listeners
use `127.0.0.1`. The compact connection status appears in the map window header.
The map can be reopened with the session header's map button,
with live chunks, coordinates, pan/zoom and player-follow controls. Closing it
leaves capture running; reopening reads the current retained terrain data.

Each world visit has its own UUID, tab, terrain and player snapshot. Real server
transfers (including both ReloadWorld flag values), dimension changes and
`/save-world` start new visits. Returning to a
same-name world creates a separate visit rather than merging coordinates or
overwriting earlier exports. Files use `worlds/<server>/<name>--<UUID>.mcworld`
and, when image saving is enabled, the matching `.png`. Empty visits retain a
tab but do not create an export until chunks have been received. Plain teleports
inside one dimension do not reliably identify a new world; use `/save-world`
to explicitly split arenas in that case.

Mutable world/player data and final map changes are finalized before switching
visits. Only archiving the finalized world runs in the background. Regression
tests verify same-name/same-coordinate visits, independent real PNG/ZIP files,
player snapshots and tab selection.

Regression tests cover command defaults and validation, overlapping operations,
account selection and cancellation, optional pack icons, versioned pack progress,
map/session reset, spawn coordinates and bootstrap/event ordering. The user
confirmed a real CubeCraft join with terrain and coordinates visible in the
native Wails window. This is distinct from frontend tests with a mocked bridge.

The existing NetherNet server compatibility and encrypted-pack restrictions
remain documented in [nethernet-migration.md](nethernet-migration.md). The UI
refresh does not change those transport or decryption limitations.

Normal client disconnects, including the exact NetherNet SCTP transport closed
message, finish the session without an error dialog. The same upstream closure
and failed transports remain errors.

Listener events received before bootstrap completes cannot trigger main-window
actions in the read-only map process. A regression test covers this startup
ordering, which previously caused a blank window on a live join. Export folder
buttons use the native file manager because Wails rejects file:// browser URLs.

Windows GUI builds generate ICO resources from the transparent pixel icon in
[../assets/](../assets/). Both native windows use this taskbar icon. The generated
.syso is temporary and excluded from Git. Image provenance and prompt are in
[../assets/README.md](../assets/README.md).

## GLB models from files and captures

The Render and Cinematic editors export GLB from an imported `.mcworld`, a world
folder, a saved CraftBox project, or a retained capture. Use **Export GLB** in
the editor toolbar or **File → Export 3D map as GLB**. The native save dialog
selects the destination; the export includes geometry and an embedded texture
atlas, without baking the editor's lighting and camera effects.

The native bridge contract is `ExportRenderGLB(worldId) -> path`,
`GetModelExportProgress() -> {running, done, total, bytes, stage}`, and
`CancelGLBExport()`. Cancelling the save dialog returns an empty path. Only one
export runs at a time. Failed or cancelled exports remove temporary files and
preserve any previous destination. The map-window HTTP bridge retains its
server-selected destination and accepts no caller-provided file path.

Exports spool one chunk at a time to a temporary binary file beside the output,
then assemble the final GLB atomically. The old 256 MiB whole-scene export limit
does not apply. Available disk space and the GLB format's 32-bit file length
(less than 4 GiB per file) remain the practical limits. The desktop viewport
separately budgets one eighth of available RAM at startup, bounded between
128 MiB and 1 GiB (at most 256 MiB on 32-bit systems), with a 512 MiB fallback
when memory information is unavailable. Large imported worlds can be exported
from the toolbar even while the viewport is loading.

Validation includes a complete 1,755-chunk Hive archive exported to a
470,444,448-byte GLB, with 6,139,178 triangles and an embedded PNG atlas.

### Water and still render quality

The shared Render/Cinematic inspector's Details tab now offers separate water
opacity, visibility in blocks, roughness, HDR reflections, static ripples and tint.
The mesher distinguishes vanilla water from glass and other blended surfaces.
Water uses an opaque-depth prepass and exponential screen-space depth fading;
environment reflections do not include nearby world geometry.

Output → Image sampling selects 1, 4 or 16 subpixel samples for still PNGs.
Sampling holds atmosphere time fixed and averages HDR before bloom and grading,
without enlarging the render targets. Cinematic shutter accumulation retains
its existing path. Saved styles without the new settings receive defaults.
Old packed projects with combined water/glass geometry need a world reimport
to use separate water shading. Bundled scene presets include tuned water values.

Orthographic Fit uses projected bounds with an eight percent margin. Saved views
retain their framing. Research, scope and limitations are documented in
[render-quality-research.md](render-quality-research.md).

Water controls now additionally separate shallow/deep colors, palette strength,
color depth, atlas texture detail, ripple size, artistic depth emission and overall
transparency. Zero atlas detail uses the isolated tile's mean color. Full water
transparency also removes its depth from the DOF prepass. The composed atlas and
water shader hooks are regression-tested together; the depth hook uses the
preserved color fragment anchor after texture sampling.

Fog surface protection reduces only fog near visible geometry, using the actual
surface distance independently of the integration cutoff. Fine-detail protection
limits sharpening of high-frequency contrast. Output offers adjustable fit margin
and vertical placement, preserved across look changes. See
[render-water-controls.md](render-water-controls.md) for the complete controls.

### Console workspace

The console is now a responsive tool window with independent CraftBox views and
native PowerShell terminals. Its output area flexes with the dock; the previous
230-pixel cap no longer applies. Console filters/pause/clear/bookmarks and reading
state are scoped to a tab. PowerShell tabs use ConPTY with bounded pull-based
output and xterm.js; each owns its shell process tree. All are closed on shutdown.
See [console-workspace.md](console-workspace.md) for controls and shortcuts and
[console-terminal-api.md](console-terminal-api.md) for native contracts.

### Capture removal and free flight

World-map tabs have individual delete buttons with an English confirmation
dialog. `DeleteCapturedWorld(worldId)` removes that capture and child regions,
including its recording directory, archive and map image. The active connection
continues; a discarded current visit resumes recording on the next world.
Both the main Wails app and separate map window use the same operation.
The map bridge accepts authenticated POST `/world-delete` with only `worldId`;
unexpected fields, missing IDs, browser origins and unauthorized requests fail.
File removal is restricted to that capture's `--ID` folder under `worlds` and
its `.mcworld`/`.png` siblings; linked paths outside the capture root are rejected.
Tombstones suppress late world events and archive writes. Writer shutdown and
authoritative paths are synchronized with rename before removal. Long deletes
use a separate timeout from the map's short polling requests.

The Render editor's First person button switches its existing camera to free
perspective flight without rebuilding terrain. WASD moves, E/Space rises,
Q/C descends, Shift boosts and Alt slows. Right-drag looks; Mouse lock captures
the pointer, and Escape returns to Orbit. Scrolling adjusts speed. Flight settings
include FOV, sensitivity, level flight, world-coordinate teleport, current-position
copy and Return to start. Flight ignores collisions. Coordinates include the
renderer origin offset. Camera swaps update render passes and gizmos together;
export completion reconciles a navigation change deferred during capture.
Cinematic Space/play behavior is preserved.

Minecraft auto-join is monitored separately from the OS launch request. The
main workspace shows a persistent flyout until the local Minecraft connection
starts. After 20 seconds without a connection it offers Retry join and Copy
connection address, with guidance to return an already running game to its
main menu. The listener remains ready for slower starts. Retry sends one new
join request; there are no automatic retries or forced Minecraft restarts.
The hint disappears on connection progress, stopping the capture, or changing
its ID or endpoint. Map windows never launch or monitor another join attempt.

Inline operational errors and warnings use FeedbackFlyout in the shared
notification stack. Recovery controls remain live through portals, and input
validation descriptions retain their IDs for assistive technology. The stack
is hosted within an open modal's focus boundary; a manual popover keeps it
positioned against the viewport. Long project names are clamped inside equal
height cards, with full names in tooltips and stable metadata rows.

Collection browsers share a Select/Done mode, Ctrl/Cmd-click toggles and Shift
ranges. Batch actions have one toolbar instead of duplicated delete buttons or
row checkboxes. World capture batches omit descendants of selected parents;
recording batches retain only remaining targets after a partial failure. Left
sidebars share search, row, section and footer styling. Navigation and editing
tools retain their own applicable actions.

View headings use compact icon actions with tooltips while export actions keep
their text. Project tabs live inside the active editor rather than above the
whole collection. The notification bell stays in the application status bar
when the console opens. Text fields, selects and buttons share a thin neutral
focus line, while invalid fields retain their error border. Dialogs can grow
from their default size using the bottom-right handle or its arrow keys; Home
restores the original dimensions. Resizing remains within the viewport.

Console completion is capped at 580 pixels and adapts to smaller panes. The
command input uses a two-pixel blinking caret, follows text selection and
horizontal scrolling, and keeps the native caret during IME composition.
Reduced-motion settings disable the blink. Appearance precedes Connection in
settings, and Structure Studio precedes Marketplace in the main navigation.

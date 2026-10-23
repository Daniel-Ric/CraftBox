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

For a standalone build with the checked-in official dependency snapshots:

```powershell
$env:GOWORK = 'off'
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

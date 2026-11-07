# CraftBox icon audit

The request covered the whole visible desktop app, starting with the generic
globe inside a framed recent-project placeholder. All 28 existing icon-bearing
TSX modules, shared UI primitives and the custom SVG/chart contexts were reviewed.

## Direction

Keep one Lucide outline family, using the existing theme's foreground, muted,
selected and card colors. No new font, decorative color palette or icon font.
Use one subject identity across tabs, cards, libraries, editor headings and help.
Normalize Lucide stroke weight to 1.75, with 16px workspace icons and a lighter
36px project-card glyph. Recent project glyphs are unboxed; actual output-aspect
previews keep their meaningful frame. Existing brand artwork and captured pack/
skin images retain precedence over fallback icons.

`src/studio-icons.tsx` is the shared subject vocabulary:

| Meaning | Icon | Reason |
|---|---|---|
| Recorded world / terrain | Boxes | Voxel/block construction rather than a planet |
| Resource pack | Layers3 | Layered textures/resources rather than a shipping parcel |
| Player skin | Shirt | Appearance rather than a group of people |
| Still render | Camera | Image capture rather than an aperture setting |
| Camera sequence | Clapperboard | Consistent cinematic identity across all entry points |
| Capture workflow | ScanLine | Recording/scanning rather than an internet globe |
| Account | UserRound | One signed-in identity rather than a social group |
| Server / Realms / friends / experiences | Server / Cloud / UsersRound / Compass | Distinguish source types instead of repeating one globe |

## Application coverage

| Surfaces reviewed | Result |
|---|---|
| StartScreen, StartDialog, NewProjectDialog | Shared subject icons in mode tabs, project cards, recent rows, capture presets and session worlds. Corrected pack/skin projects previously falling through to the globe. Removed framed miniature from recent cards. |
| DesktopChrome | Same subjects in workspace navigation; account now uses a single-person identity. Search, Home, Settings, Terminal and check marks remain appropriate. |
| AssetLibrary | Same subjects in headings, rows, empty states, missing thumbnails and properties; actual thumbnails unchanged. Map opening now uses Map rather than Maximize; world-copy export has a download affordance. |
| App, SourcePicker | Source-kind glyphs now match the selected kind in shortcuts, result rows, empty states and details. Account switching uses the shared account glyph. Existing server-address and refresh/connection controls remain appropriate. |
| AccountIdentity | Shared account subject; signed-in checks, lock and retry statuses retained. |
| RenderEditor, WorldScene | Render/cinematic headings differ correctly; missing/captured terrain uses the world subject. Overview, centered orbit, player targeting, GLB download and loading controls remain appropriate. |
| AtlasMap | Pan uses Hand rather than the selection pointer. Empty terrain uses the world subject. Layers, area selection, location, ruler, coordinates and 3D Box controls retained. |
| CinematicStudio | Local path opening uses FolderOpen, project saving uses Save, WebM output uses FileVideo. Playback, pause, frame navigation, undo/redo, duplicate, delete and camera-to-keyframe controls retained. |
| LookLibrary, AtmosphereLibraryDialog, AutoDirectionControls, FlightControls | Looks now use Palette. Cloud/Haze, automatic direction's wand and flight settings' sliders retained. |
| StudioSupportDialogs | Help topic subjects match the real workspaces. Keyboard Command and brand identity retained. |
| MediaDialogs | FileArchive and FolderOpen accurately identify .mcworld archives and folders; retained. |
| PreferencesDialog | Sun/Moon/Monitor, update/download, reset, search and external links retained. |
| ConsoleDock, ShellTerminal | Transport, output following, find, copy/paste, wrapping, clear, export, tab and pane controls retained. |
| ViewportMenu | View/framing, selection and tilt lock glyphs retained. |
| WorldComposer | Scan/Move, directional composition, copy and removal glyphs retained. |
| StudioFlyouts, dialog, sonner | Existing success/info/warning/error meaning and color retained; only stroke family normalized. |
| select, dropdown-menu | Familiar disclosure arrows and selected-state check marks retained. |
| EditorLoadingOverlay | Spinner retained, including its animation. |
| CurveEditor, CinematicTimeline, CinematicAudioTimeline, WorldCopyDialog | Custom SVG plots, keyframes, waveforms and spatial previews are information graphics, not replaceable UI glyphs; retained. |

## Verification

- TypeScript typecheck passes; full frontend suite: 102 files / 348 tests pass.
- Additional AccountIdentity/StartScreen checks pass after final shared aliases.
- `tools/verify-studio-icons.mjs` renders actual components in Edge, verifies all
  five project meanings and library fallbacks, and captures both themes and all
  three libraries. Results/screenshots: `tools/light-audit/icons-*`.
- An independent read-only review found no important semantic, accessibility or
  integration defect. Surrounding labels and captured-image precedence remain.
- Production frontend and GUI build pass; desktop Go tests and Windows resource
  validation pass. New EXE: `1.63.1-studio-icons-dev`, 85,552,640 bytes, SHA256
  `4652D2328575E755693EFA708A84D13000EC888E1E43BCEBBD94A5759BA13512`.
  The local installation script is updated to this verified artifact. The session
  does not have write access to D:\bedrock-tools, so the new EXE must be installed
  locally rather than claiming it has already replaced the installed version.

Primary icon references: [Boxes](https://lucide.dev/icons/boxes),
[Layers](https://lucide.dev/icons/layers), [Shirt](https://lucide.dev/icons/shirt),
[Camera](https://lucide.dev/icons/camera),
[Clapperboard](https://lucide.dev/icons/clapperboard),
[Scan line](https://lucide.dev/icons/scan-line).

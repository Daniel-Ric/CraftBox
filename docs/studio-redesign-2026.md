# CraftBox Studio redesign — Cape Editor / Adobe 2026

Status: complete and delivered, 20 October 2026. The installed executable matches the verified production build; delivery details are recorded below.

## Evidence and direction

The user's reference is `D:/Code Projekte/WebStorm/Cape Editor`. Read-only source inspection covered `assets/dialogs.css`, `controls.css`, `photoshop.css`, `StudioWindow.vue`, `StudioMenu.vue`, `NewProjectPanel.vue`, and export/settings components. Screenshots of its current dist are in `tmp/studio-redesign/reference/`.

Cape succeeds through a consistent 40 px window title strip, a task-specific content area, permanent 52 px actions, readable 12 px controls, neutral panel tabs, a separate document/context strip, subdued blue selection, and blue reserved for primary actions. CraftBox currently has overlapping CSS patches, oversized descriptions in dialog headers, tiny captions used as controls, repeated source/header bars, empty export previews, and preferences hidden using positional CSS selectors. Replace those structures, not just their colors.

Official 2026 references inspected:

- [Photoshop preferences](https://helpx.adobe.com/photoshop/desktop/get-started/settings-and-preferences/adjust-preferences.html), updated 28 September 2026: category navigation, content pane, explicit confirmation.
- [Photoshop new-document presets](https://helpx.adobe.com/photoshop/desktop/create-open-import-images/create-images/create-documents-using-presets.html): categorized presets and editable document details.
- [Premiere workspaces](https://helpx.adobe.com/premiere/desktop/get-started/tour-the-workspace/what-are-workspaces.html): panels grouped into purpose-specific workspaces.
- [Premiere export](https://helpx.adobe.com/premiere/desktop/render-and-export/export-files/export-video-and-audio-files.html): preview, format/preset, file information and dimensions before export.

Design tokens: workspace #232323, panels #303030, raised surfaces #383838, input #242424, separator #505050, text #dedede, selection #496580 / border #52779a, action #267cce. Preserve four brightness levels and a white light-theme canvas. Segoe UI 12 px controls, 13 px panel headings, 11 px metadata, 4/8/12/16/24 px spacing. No global left navigation sidebar. Custom brand art must come from the imagegen skill; retain the existing standard Lucide icon system for ordinary controls.

## Individual view and dialog plans

| Area | Current assessment | Required implementation | Proof |
|---|---|---|---|
| Menubar / workspace chrome | Flat command list, repeated navigation actions, no searchable commands | Task-grouped File/Edit/Scene/Sequence/View/Window/Help, contextual enablement, Ctrl+K search, useful shortcuts; document/context strip; preserve Start overlay | Menu, keyboard, disabled-command tests; rendered menus |
| Welcome | Large empty recent area, duplicated entry points | Compact branded task launcher; recent/sitting worlds, search and meaningful empty state; cancel returns to live workspace | Open/close/New/cancel/reopen tests; desktop/compact screenshots |
| New Project | Preset numbers fixed, fields partly read-only | Cape-style preset browser and grouped editable details; orientation and valid custom dimensions; real output summary; visible actions | Custom preset/aspect/duration tests; no hidden fields at desktop sizes |
| World import | Two cards dominate the dialog | Source navigation and concise file/compatibility information; clearly named picker action | Native picker routing and cancellation tests; screenshot |
| Image export | Icon box in place of preview | Actual scene preview, filename, output summary, grouped PNG settings, transparency and exact pixel dimensions | Real PNG 4K/alpha export; preview screenshot |
| Cinematic export | Configuration duplicates inspector; preview is symbolic | Real frame preview, filename, format navigation, MP4/GIF settings, honest WebM preview dimensions, fixed actions | GIF/MP4/WebM file and metadata validation |
| Preferences | Positional CSS hides categories, no workspace preferences | Dedicated category component; theme swatches, startup/density/motion preferences, application updates in explicit content panes | Persistence and category tests; all themes/screenshots |
| Account / sign-in | Generic stacked form | Identity header, account status, device-code step, action area and useful progress/error states | Fake-backend signed-out/in/device-flow screenshots and interactions |
| Source picker | Generic list inside large empty well | True tab navigation, compact searchable source rows, availability state and stable footer | Source switching/stale response/keyboard tests; populated/empty/loading states |
| Errors | Raw error dump is primary content | Application alert layout, meaningful heading/message, expandable technical details, copy/close actions | Error interaction test and screenshot |
| Help / About | Long shortcut list; version mixed into settings | Concise workflow help, searchable commands, dedicated About with brand/version/build; actual links only | Open from Help, shortcuts and About screenshots |
| Worlds view | Sparse list beside an oversized form | World browser with search, grid/list and selection; actual world/chunk/entity details and direct Render/Cinematic actions; organized source inspector | World selection/open-render/filter tests; populated/empty views |
| Resource packs view | Progress rows have weak metadata hierarchy | Asset-library table/grid, icon/name/version/status/size, selected item details and existing folder action | Filter/status/folder tests and screenshots |
| Skins view | Small unorganized cards | Proper checkerboard gallery, selectable skin preview with pixel dimensions, name and PNG export | Skin preview and native PNG route test; populated screenshot |
| Render editor | Duplicated header/source bars, several panel visual languages | Compact document/source options strip; readable inspector groups; collapsible sections; consistent view controls, real preview/export integration | Real-terrain look controls, camera/PNG checks; 1240x820 and light/dark |
| Cinematic editor | Monitor/inspector/timeline compete for space | Premiere-like monitor and effect controls, adjustable timeline height, restrained track colors and consistent transport/status; keep continuous camera interpolation | Timeline resize/keyframe/undo/preview/export checks |
| Console dock | Good basic output, inconsistent sizing | Match workspace panel header; resize/close, search/level/copy/clear; readable monospace stream and real session state | Console controls and bounded viewport screenshots |
| Independent map window | Separate older toolbar visual language | Matching context/document bars, world tabs, 2D/3D/region/spawn panels, compact details/dialogs; preserve capture semantics | Existing map tests plus independent window screenshot |
| Region / rename / spawn editing | Form-heavy small dialogs | Grouped fields, clear selected world context, measured width and fixed actions | Existing map edit tests and rendered form states |
| Shared controls / windows | CSS overrides fight component defaults | Shared studio window classes with title/content/actions, consistent small rectangular secondary buttons, distinct primary actions, focus and disabled states; remove superseded theme layers | CSS import audit; modal bounds/focus tests across all dialogs |

## Execution and completion gates

- [x] Inspect Cape source and actual rendered new-document/workspace/export views.
- [x] Inspect current official Adobe 2026 references and record per-view assessment.
- [x] Build shared tokens/window/controls foundation and remove superseded visual layers from the active imports.
- [x] Implement each view/dialog row above and record authoritative evidence below.
- [x] Refine the existing grass-block icon using built-in imagegen, preserving its identity as the user's subsequent instruction requires; store prompt/provenance in `docs/icon-refinement.md`.
- [x] Run frontend build, all meaningful frontend tests and focused native checks if backend changes.
- [x] Capture and inspect light/dark windows, views and relevant populated/empty/progress states.
- [x] Review the final patch, fix important findings, repeat affected checks only.
- [x] Build EXE with icon resources, back up and replace `D:/CraftBoxs/craft-box.exe`, compare hashes.
- [x] Audit every explicit requirement against actual files and runtime evidence before marking the goal complete.

## Verification ledger

### Implemented views and dialogs

The assessment/plan table above records the starting problems and individual decisions. The resulting implementation and evidence are:

| Area | Implemented result | Evidence |
|---|---|---|
| Menubar and workspaces | File/Edit/Scene/Sequence/View/Window/Help; real contextual actions, command search with Ctrl+K, background-shortcut guard while modal; no global left sidebar | `DesktopChrome.tsx`, `DesktopChrome.test.tsx`; `current/menu-*.png`, `current/command-search.png` |
| Welcome | Compact task launcher, original refined icon, filtered recent projects, current-session worlds, returns to the live workspace | `StartScreen.tsx`, start/App tests; `current/welcome-dark.png`, `welcome-light.png` |
| New Project | Format browser, editable dimensions, orientation swap, renderer-compatible ratio validation, separate capture and sequence settings, fixed actions | Five StartScreen tests, including failing-before size/ratio regressions; nine compact-window captures plus light/dark presets |
| Import | Source navigation beside file/compatibility information; clearly named native picker action | Import routing test; `render/import-dark.png`, `render/import-light.png` |
| PNG export | Actual rendered preview, filename, exact dimensions, transparency preview refresh, native destination dialog | Preview/filename test; `render/png-export-dialog.png`, `render/png-export-light.png`; actual 3840×2160 and alpha 1920×1920 files |
| Cinematic export | Actual camera frame, format navigation, filename, encoder-specific settings and output information | `cinematic/gif-export-dialog.png`, light MP4/WebM captures; decoded GIF, MP4 metadata and WebM header checks |
| Preferences | Category navigation instead of positional CSS; five theme choices; persisted welcome/density/motion; real update actions | `PreferencesDialog.tsx`, persistence/category test; five appearance states plus workspace/version captures |
| Account | Identity/context area, separated account controls, readable device code, browser step and anchored close action | `current/account-signed-out.png`, `account-signed-in.png`, `account-device-code.png`, `account-light.png` |
| Source picker | Actual tabs, searchable source rows, availability states, bounded result pane and stable footer | Stale-response test retained; populated/empty/loading light captures and `source-picker.png` |
| Errors | Readable short messages; technical HTTP/stack details disclosed on demand; copy/close actions | Live-error race test retained; `current/error-details.png`, `error-light.png` |
| Help / About | Task-specific help categories and shortcuts; dedicated original-brand/version window | `StudioSupportDialogs.tsx`; four help topics in light, dark help, light/dark About |
| Worlds | Filterable grid/list browser, selection inspector, chunk/entity/spawn metadata, direct Render/Cinematic creation, native folder/map actions | AssetLibrary selection/filter/routing test; populated grid/detail and empty light/dark captures |
| Packs | Asset browser with version/size, receiving progress, saved/error status and selected folder action | Populated/progress/empty captures; native folder callback verified in browser |
| Skins | Checkerboard gallery, grid/list, selected texture with dimensions and PNG saving | AssetLibrary PNG routing test; native callback checked, populated/empty light/dark captures |
| Render editor | Compact source/document strips, common inspector tabs and field hierarchy, neutral light canvas, integrated import/export windows | Real 48-chunk fixture, editable looks/colors, camera change, 4K/alpha export, compact and light/dark screenshots |
| Cinematic editor | Common neutral monitor/inspector, blue/gold tracks, resizable timeline, consistent transport and integrated format export | Real terrain, key/undo/path tests and rendered checks; 1240×820 bounds; GIF/MP4/WebM successful |
| Console | Session tab, readable monospace output, search/level/copy/clear, close, pointer/keyboard height adjustment | `PanelResizeHandle` bounded keyboard test; browser filter and height check; `current/console-filtered.png` |
| Independent map | Shared context strips, compact document tabs, 2D/3D controls, consistent scene toolbar/status, neutral canvas in every theme | Actual BDS-derived 3D fixture, navigation/GLB/retry/live-delta checks; rendered theme pixels checked |
| Region / rename / spawn | Grouped coordinate fields, right-side task inspectors, shared rename window with anchored actions | Existing region/spawn/rename tests retained; six light/dark map-edit captures |
| Shared controls/windows | One foundation for colors, typography, input/button/focus states, 40px titles and 52px actions, bounded inner scrolling | Main import audit; modal/viewport bounds checked throughout 69-state browser run |

Screenshot paths in this table are relative to `tmp/studio-redesign/`. The core matrix and its exact screenshot inventory are recorded in `tmp/studio-redesign/current/verification.json`. Screenshots were visually inspected, including the reference app, welcome/presets, preferences, world selection, source picker, account/device flow, import/export, and map editing. Test screenshots use controlled transport responses; terrain/video/3D checks use recorded real geometry, not substitute meshes.

### Commands and observed results

- `npm.cmd test`: **87 tests passed in 28 files**. The two pre-existing jsdom canvas notices are covered by the actual WebGL browser runs.
- `npm.cmd run build`: TypeScript and Vite production build succeeded.
- `tools/verify-studio-redesign.py`: **69 states**, no browser errors, no outer-dialog or page overflow; includes 1182×695, 1024×640 and 760×560 document windows.
- `tools/verify-render-editor.py`: **48 chunks / 793,730 triangles**, actual 4K PNG and transparent square PNG passed; dynamic transparency preview, import windows and light/compact layout passed.
- `tools/verify-cinematic-editor.py`: timeline/key/undo/path and light/desktop bounds passed; GIF decoded, MP4 **720×406 / 0.25s** decoded, WebM signature checked; no browser errors.
- `tools/verify-3d-viewer.py`: real BDS fixture; camera navigation changes pixels, GLB action reaches backend, retry preserves terrain, live update and compact mode passed. Gray/graphite rendered background pixels match their theme, and light/dark views passed.
- Code review identified two important issues: extreme custom aspect ratios and preview camera mutation. Both have failing-before regression tests, fixes and a clean follow-up review.
- `go build -tags gui,production ... ./cmd/CraftBox`: native GUI production executable built, version `1.59.0-studio-workspaces.20261020-dev`.
- `tools/verify-windows-resources.go`: Wails icon group **3**, **9 icon sizes**, application manifest present.
- The active imports contain none of `studio-theme.css`, `desktop-chrome.css`, `adobe-dialogs.css`, `welcome-dialog.css` or `start-screen.css`; these superseded sheets are inactive. Scoped renderer/inspector styles remain for their actual layouts.

No MCP server was added. No Cape Editor source was modified. The original grass-block identity is retained; its imagegen refinement, file paths and prompt are in `docs/icon-refinement.md`.

### Final delivery and requirement audit

The final capture inspector keeps its start/stop action in a fixed footer, with source and recording options scrolling above it. Redundant download/source summaries were removed. The affected App/AssetLibrary suite passed all 17 tests after this last change, the production TypeScript/Vite build passed, and `tools/verify-capture-footer.py` verified six combinations of 1440×960, 1182×695 and 1024×640 with the console open/closed. The action remained inside the workspace in all cases, without browser errors. Screenshots and bounds are in `tmp/studio-redesign/capture-footer/`.

All 20 view/dialog assessment rows have implementation and runtime evidence in the ledger above. Cape source and rendered references, current official Adobe documentation, individual view plans, shared visual foundations, task-specific layouts, menu actions, modal behavior, custom icon provenance, render/export behavior, independent review and executable delivery have been checked. The final change affects capture layout only; the previously verified rendering and encoding paths are unchanged.

- Installed executable: `D:/CraftBoxs/craft-box.exe`
- Version: `1.59.0-studio-workspaces.20261020-dev`
- Previous executable backup: `D:/CraftBoxs/craft-box-before-studio-workspaces-20261020-064632.exe`
- Build, staged copy and installed SHA-256: `021E0138F7F93E89A1C9D2A9DEAD5B10B6828B4A6467FAD47D540020135B0657`
- Native resource verification: Wails icon group 3, nine icon images, application manifest present.

No running user application was terminated. Restart CraftBox to load the installed version.

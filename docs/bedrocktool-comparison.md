# CraftBox and bedrocktool

This comparison lists what the original [bedrocktool](https://github.com/bedrock-tool/bedrocktool) and CraftBox provide at the source revisions below. CraftBox is maintained by **SpindexGFX** and builds on bedrocktool's proxy and capture work.

## Source snapshot

**Reviewed on 14 November 2026.**

| Project | Reviewed source |
| --- | --- |
| Original bedrocktool | Default branch master, commit [64c3e7657577b159931adeed3b4a3044c751c862](https://github.com/bedrock-tool/bedrocktool/tree/64c3e7657577b159931adeed3b4a3044c751c862), committed 11 November 2026. |
| CraftBox | Commit [28276639ea12798d7c7f5910b929a31c2a45872f](https://github.com/Daniel-Ric/CraftBox/tree/28276639ea12798d7c7f5910b929a31c2a45872f), including the material/style fixes described below. The comparison records this source revision; published release binaries may differ. |
| Method | Read the upstream README, commands, capture handlers, proxy, scripting and Gio pages; compare with CraftBox's public commands, desktop bridge, workspaces and implementation guides. This is a source review, not a fresh live-server compatibility test. |

**Reading the tables:** **Yes** = implemented; **Partial** = the related capability exists, but the listed workflow or controls differ; **No** = no equivalent implementation was found in the reviewed source. **Not verified** marks a detail that this review did not establish. **Not applicable** means the corresponding module does not exist in that project. Notes in each cell explain the scope. An upstream proposal or pull request is not counted as already present in master.

This inventory covers the public modules and their user-facing functions. It groups related controls rather than counting every button or refactor as a separate feature. The dated revisions matter: later upstream releases may include some of these changes.

## Shared foundation

These are upstream capabilities too. They should not be presented as CraftBox inventions.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| World capture | **Yes** — received terrain as a Bedrock world. | **Yes** — Save received terrain as a Bedrock world. |
| Entities and inventories | **Yes** — selectable entity, player, equipment and inventory capture. | **Yes** — Selectable capture of entities, players, equipment and inventories. |
| Packs and skins | **Yes** — resource-pack and player-skin downloading. | **Yes** — Download server resource packs and collect player skins. |
| Microsoft sign-in | **Yes** — device sign-in and saved account selection. | **Yes** — Device sign-in, saved accounts and account selection. |
| Connection sources | **Yes** — custom addresses, Realms, friends, featured servers and Experiences. | **Yes** — Custom addresses, Realms, friends, featured servers and Experiences. |
| NetherNet | **Yes** — supported by the reviewed proxy. | **Yes** — Supported by the proxy. |
| Live map | **Yes** — tiled capture map. | **Yes** — Live map plus the atlas tools listed below. |
| Packet capture and replay | **Yes** — .pcap2 capture/replay; recording enabled by default. | **Yes** — Recording and .pcap2 replay are shared foundations. Packet recording is already enabled by default upstream. |
| World merging and PNG rendering | **Yes** — CLI merge and top-down PNG commands. | **Yes** — CLI commands plus desktop composition and renderer extensions. |
| JavaScript hooks | **Yes** — scriptable world capture. | **Yes** — Scriptable capture is inherited and extended. |
| Other CLI tools | **Yes** — chat logging, Realms listing and updates. | **Yes** — Retained chat logging, Realms listing and updates. |
| Graphical interface | **Yes** — Gio GUI. | **Yes** — Wails/React studio with dedicated workspaces and document tabs. |

Upstream: [README](https://github.com/bedrock-tool/bedrocktool/blob/64c3e7657577b159931adeed3b4a3044c751c862/Readme.md), [commands](https://github.com/bedrock-tool/bedrocktool/tree/64c3e7657577b159931adeed3b4a3044c751c862/subcommands), [GUI pages](https://github.com/bedrock-tool/bedrocktool/tree/64c3e7657577b159931adeed3b4a3044c751c862/ui/gui/pages), [source dialogs](https://github.com/bedrock-tool/bedrocktool/tree/64c3e7657577b159931adeed3b4a3044c751c862/ui/gui/popups), [proxy](https://github.com/bedrock-tool/bedrocktool/tree/64c3e7657577b159931adeed3b4a3044c751c862/utils/proxy).

## Worlds and capture

The following rows describe CraftBox's additions or changes to the shared capture system.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Source browser | **Partial** — individual source-selection dialogs; no shared searchable browser. | **Yes** — One searchable source picker with details and confirmation across Worlds, Resource packs and Player skins. |
| Recent addresses | **No** — no equivalent recent-address browser. | **Yes** — Save/select/remove recent custom destinations without automatically connecting to them. |
| Connection checks | **Partial** — proxy connection/status handling; no separate desktop status/transport/login inspection workflow. | **Yes** — Separate HTTP status, NetherNet transport and actual login results, including endpoint/version/timing details. |
| Account identity | **Partial** — saved account selection; no equivalent desktop identity/capability inspector. | **Yes** — Actual gamertag, Xbox ID and sign-in state remain separate from the local editable profile name. |
| Remembered capture options | **Partial** — command settings in the Gio UI; no per-mode persistence found. | **Yes** — Desktop settings persist per capture mode. |
| Generator | **Partial** — void-generator toggle; no explicit void/flat/normal selector. | **Yes** — Void, flat and normal choices, /generator and the retained /void command. |
| Custom entity option | **Partial** — entity capture; no separate custom-entity opt-in. | **Yes** — A separate opt-in for custom visual actors. |
| Recording defaults | Entities enabled; players disabled. No separate custom-entity option. | Entities, custom entities and players disabled. |
| Image/packet defaults | Final map image disabled; packet recording enabled. Both configurable. | Final map image enabled; packet recording enabled. Both configurable. |
| World visits | **Partial** — save/reset and dimension/transfer handling; no CraftBox visit-tab workflow. | **Yes** — Independent visit IDs and finished captures across transfers, dimensions and repeated names; no empty archives for visits without chunks. |
| Save while playing | **Yes** — /save-world saves and resets without disconnecting. | **Yes** — /save-world plus /cb save and desktop visit/output integration. |
| Merged dimensions | **No** — no equivalent option in the reviewed master settings. | **Yes** — Later captures become custom dimensions in the first world, with generated /craftbox:dim travel commands. |
| Recorded forms and scoreboards | **No** — no equivalent recording/export option in reviewed master. | **Yes** — Export form/navigation and scoreboard snapshots into a behavior pack. Server-side actions are not recreated. |
| PlayerAdd hook | **Partial** — entity hooks, but no separate PlayerAdd override. | **Yes** — A script can override the normal decision to save an individual player. |
| Replay without a client | **Partial** — .pcap2 replay; no corresponding replay-join choice. | **Yes** — Process a recording immediately; replay-join opts into waiting for a joining Minecraft client. |
| Realms metadata | **Partial** — Realms connections; no corresponding Realm Stories session handling found. | **Yes** — Realm Stories session handling accompanies the connection flow. |
| Map-art outputs | **Partial** — map-item capture; no equivalent tile/map-wall export workflow. | **Yes** — Individual item-frame map PNGs and assembled map walls. Upstream map-item capture itself is shared. |
| Native ingame menu | **No** — no /cb menu. | **Yes** — /cb menu with capture, map, points, blueprints, regions, entity exclusions, connection and help pages. |
| Ingame points | **No** — no equivalent named point/note browser. | **Yes** — Named markers/notes, search, filtering and detail pages. |
| Local capture commands | **Yes** — existing capture commands such as naming, void, saving and exclusions. | **Yes** — Naming, status, version, saving, map visibility, zoom and entity exclusion controls. |
| Connection details ingame | **No** — no /cb connection inspector. | **Yes** — /cb connection displays known endpoints, transport, version and latency. |
| Player map position | **No** — no equivalent optional position/follow workflow identified. | **Yes** — Optional current-position dot and desktop player follow, separate from exported image content. |
| Named capture regions | **No** — no equivalent split/copy region workflow. | **Yes** — Split or copy non-overlapping areas into independent regions and exports. |
| Region spawn placement | **No** — no equivalent named-region export workflow. | **Yes** — Find a standing position inside retained terrain when the original player position lies outside the region. |
| Singleplayer world copies | **No** — no installed-world copy workspace. | **Yes** — Discover installed worlds or select folders and export a separate .mcworld. |
| Dependency checks | **No** — no equivalent local-world dependency preview and backup workflow. | **Yes** — Exact pack versions/transitive add-ons, original-pack lookup and explicit backup-only output with a dependency report. |

Upstream: [world settings](https://github.com/bedrock-tool/bedrocktool/blob/64c3e7657577b159931adeed3b4a3044c751c862/subcommands/worlds.go), [handlers](https://github.com/bedrock-tool/bedrocktool/tree/64c3e7657577b159931adeed3b4a3044c751c862/handlers/worlds). CraftBox: [source browser](source-browser.md), [account identity](account-identity.md), [ingame commands](ingame-commands.md), [regions](world-region-editor.md), [singleplayer copies](singleplayer-world-copy.md), [settings](../subcommands/worlds.go).

## Maps and world composition

Upstream provides a live map and CLI merge/render commands. CraftBox adds these interactive workflows.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Separate map window | **No** — the reviewed Gio map is embedded in its Worlds page. | **Yes** — Keep the map open independently of the main workspace. |
| Detailed atlas | **Partial** — tiled live map; no equivalent raw-chunk inspection atlas. | **Yes** — Read captured raw chunks, block heights, biome IDs/names and block names without constructing 3D meshes. |
| Navigation | **Partial** — map pan/zoom; no equivalent coordinate/block inspection toolset. | **Yes** — Pointer-centered zoom, pan, coordinate jumps/copying and block/chunk inspection. |
| Overlays | **No** — no equivalent atlas/composition tool. | **Yes** — Adaptive grid, elevation colors, relief, contours, biomes, capture coverage and update highlights. |
| Annotations | **No** — no equivalent atlas/composition tool. | **Yes** — Positioned markers, symbols, labels, colors and saved map presentation. |
| Measurements | **No** — no equivalent atlas/composition tool. | **Yes** — Distance paths and closed-polygon areas. |
| Selections | **No** — no equivalent atlas/composition tool. | **Yes** — Resizable/numeric areas with real captured and missing chunk counts. |
| Atlas-to-3D | **No** — no equivalent atlas/composition tool. | **Yes** — Pick a map position and open it in the 3D view. |
| Map export | **No** — no equivalent atlas/composition tool. | **Yes** — Whole map or selection, texture detail, annotations and transparency. |
| Desktop composition | **No** — no equivalent atlas/composition tool. | **Yes** — Combine up to 16 world/region placements with X/Y/Z offsets. |
| Placement tools | **No** — no equivalent atlas/composition tool. | **Yes** — Drag placements, snap to chunks or arrange them with a defined gap. |
| Conflict review | **No** — no equivalent atlas/composition tool. | **Yes** — Count occupied-block conflicts and confirm a staged snapshot; later placements win confirmed conflicts. |
| Composed world output | **No** — no equivalent atlas/composition tool. | **Yes** — Save a new world with supported block entities, actor data and compatible packs while retaining source captures. |
| CLI renderer inputs | **Partial** — world-folder PNG rendering; no corresponding archive/dimension/map-art options. | **Yes** — World folder or .mcworld, dimension name/ID selection and independent map-art output. |
| Top-down shading | **Partial** — top-down PNG renderer; not the same biome/height/water-depth shading pipeline. | **Yes** — Biome-aware colors, height shading and depth-aware water appearance. |

Upstream: [renderer](https://github.com/bedrock-tool/bedrocktool/blob/64c3e7657577b159931adeed3b4a3044c751c862/subcommands/render.go), [merge](https://github.com/bedrock-tool/bedrocktool/blob/64c3e7657577b159931adeed3b4a3044c751c862/subcommands/merge.go). CraftBox: [atlas/composition](map-atlas-and-composition.md), [regions](world-region-editor.md), [renderer](../subcommands/render.go).

## Render editor

No corresponding interactive 3D render editor was found in the reviewed upstream GUI. Upstream's render command produces a top-down PNG.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Textured worlds | **No** — no interactive 3D render editor. | **Yes** — View captured/imported terrain, supported custom block geometry and studio scenes in 3D. |
| Camera navigation | **No** — no interactive 3D render editor. | **Yes** — Orbit, pan, zoom and configurable first-person flight. |
| Framing | **No** — no interactive 3D render editor. | **Yes** — Aspect ratio, output dimensions, views, pitch lock, margins and transparent backgrounds. |
| Look library | **No** — no interactive 3D render editor. | **Yes** — More than 100 looks, filtering/search, editable parameters and saved presets. |
| Automatic direction | **No** — no interactive 3D render editor. | **Yes** — Analyze actual geometry/texture evidence, select a look and fit a composition with a report and undo. |
| Local look preferences | **No** — no interactive 3D render editor. | **Yes** — Optional local preference learning for later style recommendations. |
| Day/night | **No** — no interactive 3D render editor. | **Yes** — Time controls, coordinated sun/moon lighting and a procedural star sky. |
| HDR skies | **No** — no interactive 3D render editor. | **Yes** — Offline panoramas and local HDR import, rotation and separate lighting/background intensity. |
| Scene lights | **No** — no interactive 3D render editor. | **Yes** — Directional, point, spot and area lights with placement controls. |
| Minecraft block light | **No** — no interactive 3D render editor. | **Yes** — Emissive colors, local lights, budgets, range and daylight adaptation. |
| Shadows | **No** — no interactive 3D render editor. | **Yes** — Resolution/softness, bounds fitted to the scene and filtered sampling. |
| Clouds | **No** — no interactive 3D render editor. | **Yes** — Volumetric banks, types, density, distribution, lighting and authored regions. |
| Cloud authoring | **No** — no interactive 3D render editor. | **Yes** — Transforms, tint, wind, scene gizmos and supported animated properties. |
| Model clearance | **No** — no interactive 3D render editor. | **Yes** — Geometry-derived footprint clearance and a fitted opening around the build. |
| Fog | **No** — no interactive 3D render editor. | **Yes** — Height/reference, density, falloff, color and surface protection. |
| Water | **No** — no interactive 3D render editor. | **Yes** — Dedicated water appearance and surface/reflection controls. |
| Color/detail | **No** — no interactive 3D render editor. | **Yes** — Exposure, tone mapping, grading, saturation/vibrance, contrast, highlights/shadows, temperature/tint, vignette, sharpening and bloom. |
| Depth of field | **No** — no interactive 3D render editor. | **Yes** — Focus picking/locking, aperture, near/far controls and virtual world scale. |
| Preview quality | **No** — no interactive 3D render editor. | **Yes** — Interactive/full modes, progressive accumulation, antialiasing, AO, shadows and export sampling. |
| Atlas filtering | **No** — no interactive 3D render editor. | **Yes** — Isolated mipmaps, alpha-weighted filtering and cutout coverage to reduce shimmer and neighboring texture bleed. |
| Resource-pack overrides | **No** — no interactive 3D render editor. | **Yes** — Apply/restore readable pack appearances for the scene. |
| Texture inspection | **No** — no interactive 3D render editor. | **Yes** — Pick a block face and inspect its state, mapping and available source pack. |
| 2D scene view | **No** — no interactive 3D render editor. | **Yes** — Orthographic map presentation with styling, grid and saved annotations. |
| Panorama | **No** — no interactive 3D render editor. | **Yes** — 360-degree image output using scene settings. |

CraftBox: [Render editor](render-editor.md), [looks](scene-looks-and-auto-learning.md), [atmosphere](atmosphere-rendering.md), [clouds](cloud-banks-and-daylight.md), [water](render-water-controls.md), [focus](render-focus.md), [renderer](../ui/desktop/frontend/src/RenderCanvas.tsx).

## Cinematic editor and exports

These editors and output workflows are additional to upstream's reviewed GUI. Shared top-down PNG rendering is listed above.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Timeline | **No** — no cinematic/3D export workspace. | **Yes** — Camera keyframes, playhead, duration, frame rate and sequence preview. |
| Paths and shots | **No** — no cinematic/3D export workspace. | **Yes** — Author camera movement, targets, shot framing and navigation. |
| Shared styling | **No** — no cinematic/3D export workspace. | **Yes** — Use the Render editor's lights, skies, water, atmosphere and grading. |
| Motion effects | **No** — no cinematic/3D export workspace. | **Yes** — Supported motion blur and animated scene/atmosphere properties. |
| Images | **Partial** — top-down PNG output, without the 3D image/panorama exporter. | **Yes** — PNG, JPEG, WebP, TIFF and BMP; transparency depends on the format. |
| Static models | **No** — no cinematic/3D export workspace. | **Yes** — GLB, glTF, OBJ, FBX and STL with supported texture/material data. |
| Video/animation | **No** — no cinematic/3D export workspace. | **Yes** — MP4, WebM, MOV, MKV and GIF, depending on runtime encoders. |
| Export tasks | **No** — no cinematic/3D export workspace. | **Yes** — Progress, cancellation and output-folder actions. |
| Portable projects | **No** — no cinematic/3D export workspace. | **Yes** — Scene pages, atlas, settings, supported custom HDR and sequences saved in .craftbox. |
| Independent tabs | **No** — no cinematic/3D export workspace. | **Yes** — Keep several render/sequence documents with their own state. |

CraftBox: [editor](render-editor.md), [cinematic navigation](cinematic-navigation-2026.md), [export formats](export-studio.md), [projects](studio-projects.md), [documents](document-tabs.md), [model formats](../internal/modelexport/).

## Structure Studio

Upstream has capture and merging, but no equivalent structure-editing workspace was found.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Documents | **No** — no structure editor. | **Yes** — Blank blueprints, .mcstructure imports and extraction from captured/local worlds. |
| Block palette | **No** — no structure editor. | **Yes** — Search, brush selection and identifier/state editing. |
| Direct editing | **No** — no structure editor. | **Yes** — Place/remove blocks and use an eyedropper in 3D. |
| Selections | **No** — no structure editor. | **Yes** — Numeric corners, draggable bounds, named subregions and extraction into a new document. |
| Bulk edits | **No** — no structure editor. | **Yes** — Fill, replace, copy, cut, paste, delete, rotate and mirror. |
| History/recovery | **No** — no structure editor. | **Yes** — Undo/redo and automatic recovery. |
| Layers | **No** — no structure editor. | **Yes** — Layer navigation, isolated/adjacent layers and top view. |
| Materials | **No** — no structure editor. | **Yes** — Required-block counts, CSV and printable material lists. |
| Comparison | **No** — no structure editor. | **Yes** — Missing, extra, different and incorrectly oriented blocks with overlays. |
| Preserved data | **No** — no structure editor. | **Yes** — Both block layers, supported block-entity NBT, sign text, containers and optional entities. Players are excluded. |
| Unknown cells | **No** — no structure editor. | **Yes** — Distinguish uncaptured cells, air and structure void. |
| Pack dependencies | **No** — no structure editor. | **Yes** — Retain available packs and locate/add original dependencies. |
| Blueprint library | **No** — no structure editor. | **Yes** — Portable projects with copied dependencies, independent of source-world lifetime. |
| Ingame selection | **No** — no structure editor. | **Yes** — /cb structure captures an area and can open it in the desktop. |
| World placement | **No** — no structure editor. | **Yes** — New world copy with replace-all, ignore-source-air or only-into-air modes. |

CraftBox: [Structure Studio](structure-studio.md), [validation](structure-studio-validation.md), [ingame commands](ingame-commands.md), [implementation](../internal/structure/).

## Resource packs, models and Blockbench

Pack downloading is shared. CraftBox adds a managed inspection/export workspace.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Library | **No** — no equivalent asset/model inspection workspace. | **Yes** — Owned copies, persistent metadata and capture provenance. |
| Imports | **No** — no equivalent asset/model inspection workspace. | **Yes** — Pack/ZIP archives, folders, PNG skins and supported skin packs. |
| Organization | **No** — no equivalent asset/model inspection workspace. | **Yes** — Search, sorting, grid/list, favorites, tags, duplicates and bulk selection/removal. |
| Pack inspection | **No** — no equivalent asset/model inspection workspace. | **Yes** — File tree, images/textures, checks, version comparison and pack-order tools. |
| Model picker | **No** — no equivalent asset/model inspection workspace. | **Yes** — Resolved blocks/entities, geometry, textures and material variants. |
| 3D previews | **No** — no equivalent asset/model inspection workspace. | **Yes** — Orbit, fit, turntable, lighting, wireframe and supported pose controls. |
| Texture resolution | **No** — no equivalent asset/model inspection workspace. | **Yes** — Source definitions/controllers and material mappings with missing/ambiguous binding diagnostics. |
| Custom blocks | **No** — no equivalent asset/model inspection workspace. | **Yes** — Supported permutations, bone visibility, face UVs, materials and transforms. |
| Animation preview | **No** — no equivalent asset/model inspection workspace. | **Yes** — Supported skeletal channels, scripts/controller transitions and bounded Molang/query evaluation. |
| Part extraction | **No** — no equivalent asset/model inspection workspace. | **Yes** — Select parts from a combined model before exporting. |
| Static outputs | **No** — no equivalent asset/model inspection workspace. | **Yes** — Displayed-model exports and preview images. |
| Editable Blockbench export | **No** — no equivalent asset/model inspection workspace. | **Yes** — Automatic, Bedrock Entity, Bedrock Block or Generic Model .bbmodel projects. |
| Blockbench geometry | **No** — no equivalent asset/model inspection workspace. | **Yes** — Supported bones/cubes, hierarchy, pivots, rotation, inflation, UVs, materials and embedded textures. |
| Blockbench animation data | **No** — no equivalent asset/model inspection workspace. | **Yes** — Supported editable source animations/controllers/expressions with compatibility warnings. |
| Original files | **Yes** — downloaded resource-pack files; no model-preview export workspace. | **Yes** — Source pack content and original model data remain separate from baked preview output. |
| Installation integration | **No** — no equivalent asset/model inspection workspace. | **Yes** — Identify supported Minecraft installations/content locations for available import/integration actions. |

CraftBox: [Asset Studio](asset-studio.md), [library](../internal/assetlibrary/), [model picker](../internal/modelpicker/), [Blockbench exporter](../ui/desktop/frontend/src/blockbench-project.ts), [model formats](../internal/modelexport/).

## Player skins and Audio Studio

Skin downloading is shared. These library/preview tools are additions.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Skin organization | **Partial** — captured skin files/list; no equivalent managed asset library. | **Yes** — Search, favorites, tags and provenance using the asset library. |
| Player variants | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Group available appearance captures by player UUID and retain alternate bodies. |
| Skin geometry | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Supported captured geometry, wide/slim arms, clothing layers and capes. |
| Pose/cosmetics | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Poses, supported equipment/attachments and supported cosmetic/particle previews. |
| Texture editing | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Save edited skin pixels as a separate item while retaining arm-model selection. |
| Skin packs | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Split into selectable skins while retaining the original archive. |
| Export choices | **Partial** — captured skin files; no displayed-model or Blockbench export. | **Yes** — Original files, preview images, displayed models and supported Blockbench projects. |
| Pack audio | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Browse/listen to supported sounds. |
| Waveforms | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Inspect playback position and sound waveform. |
| Multitrack audio | **No** — no equivalent 3D skin/audio workspace. | **Yes** — Arrange and preview several clips together. |

Upstream: [skin command](https://github.com/bedrock-tool/bedrocktool/blob/64c3e7657577b159931adeed3b4a3044c751c862/subcommands/skins.go). CraftBox: [Asset Studio](asset-studio.md), [skin corrections](slim-skins-and-decorations.md), [workspace](../ui/desktop/frontend/src/AssetStudio.tsx), [audio](../ui/desktop/frontend/src/AudioStudio.tsx).

## Packet Inspector

Recording itself is inherited. The following inspection workspace is additional.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Session archive | **Partial** — packet recordings as files; no equivalent inspector session library. | **Yes** — Persistent sessions, metadata and recording state. |
| Shared observation | **Partial** — proxy recording; no integrated observation/inspection workspace. | **Yes** — Observe the existing capture without creating another game connection. |
| Packet browser | **No** — no equivalent packet-inspection workspace. | **Yes** — Direction/type filters, search, timeline and statistics. |
| Evidence | **No** — no equivalent packet-inspection workspace. | **Yes** — Decoded fields, raw/hex bytes and exact large integer values where available. |
| Bookmarks and clips | **No** — no equivalent packet-inspection workspace. | **Yes** — Notes, bookmarks and saved portions of traffic. |
| Analysis views | **No** — no equivalent packet-inspection workspace. | **Yes** — Entity, movement, transaction and chunk evidence. |
| Comparison | **No** — no equivalent packet-inspection workspace. | **Yes** — Compare recorded sessions and differing evidence. |
| Offline playback | **Partial** — .pcap2 replay, without the inspector playback UI. | **Yes** — Navigate saved/imported traffic without a live game. |
| Import formats | **Partial** — .pcap2 replay; no equivalent ProxyPass/ProxyWeb inspector imports. | **Yes** — Supported native/ProxyWeb JSON, ProxyPass logs and .pcap2; text-only input remains marked partial. |
| Rules/triggers | **No** — no equivalent packet-inspection workspace. | **Yes** — Bounded recording windows and supported matching/intervention rules. |
| Intervention | **No** — no equivalent packet-inspection workspace. | **Yes** — Separately enable edits, delays, blocking and sends; previewed sends are bound to destination/session. |
| Content links | **No** — no equivalent packet-inspection workspace. | **Yes** — Open related skin/pack/world evidence and navigate back to packet records. |
| Skin extraction | **Partial** — skin capture command; no selected-packet extraction workflow. | **Yes** — Extract available PlayerSkin/PlayerList content with provenance. |
| Archive safeguards | **No** — no equivalent packet-inspection workspace. | **Yes** — Secret-aware views/exports, bounded storage/queries and recovery handling. |

CraftBox: [Packet Inspector](packet-inspector.md), [implementation](../internal/packetinspector/), [workspace](../ui/desktop/frontend/src/PacketInspector.tsx).

## Archive

No equivalent snapshot-analysis workspace was found in the reviewed upstream tree.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Snapshots | **No** — no snapshot-analysis workspace. | **Yes** — Capture supported installation/data sources and retain version snapshots. |
| File browser | **No** — no snapshot-analysis workspace. | **Yes** — Search/filter metadata and preview/export available files. |
| Version comparison | **No** — no snapshot-analysis workspace. | **Yes** — Added/removed/changed files and detailed content differences. |
| Identifiers | **No** — no snapshot-analysis workspace. | **Yes** — Compare available named identifiers across versions. |
| Decoded data | **No** — no snapshot-analysis workspace. | **Yes** — Supported binary/encoded content with format-specific decoding and source details. |
| JSON analysis | **No** — no snapshot-analysis workspace. | **Yes** — Field comparison and supported schema information. |
| Schemas | **No** — no snapshot-analysis workspace. | **Yes** — Prepare schema bundles, validate snapshots and re-decode with a selected set. |
| Binary strings | **No** — no snapshot-analysis workspace. | **Yes** — Search supported executable/binary strings and inspect matches. |
| Media preview | **No** — no snapshot-analysis workspace. | **Yes** — Supported images/audio and decoded WAV export when available. |
| Library handoff | **No** — no snapshot-analysis workspace. | **Yes** — Import an available snapshot pack into the asset library. |
| Long jobs | **No** — no snapshot-analysis workspace. | **Yes** — Progress/cancellation for supported capture and analysis jobs. |

CraftBox: [desktop backend](../ui/desktop/bedrock_lab.go), [analysis](../internal/bedrocklab/), [workspace](../ui/desktop/frontend/src/ArchiveStudio.tsx).

## Marketplace

Featured-server discovery is shared. Browsing these sources' supported online offers is an additional module.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Discovery | **Partial** — featured-server/Experience source selection; no offer-catalog workspace. | **Yes** — Supported server/Experience sources, details and available player counts. |
| Offers | **No** — no offer-catalog workspace. | **Yes** — Searchable grid/list, ordering, filters and item details. |
| Collections | **No** — no offer-catalog workspace. | **Yes** — Supported published collection pages and creator-scoped relationships. |
| Metadata | **No** — no offer-catalog workspace. | **Yes** — Images, descriptions, prices, ratings, platforms, availability and supplied compatibility information. |
| Previews | **No** — no offer-catalog workspace. | **Yes** — Available item images/content and technical details. |
| Favorites | **No** — no offer-catalog workspace. | **Yes** — Source/environment/item identifiers checked against the online catalog. |
| Comparison/export | **No** — no offer-catalog workspace. | **Yes** — Compare available offers and export metadata as JSON/CSV. |
| Environments | **No** — no offer-catalog workspace. | **Yes** — Separate Production/Staging queries and results. |
| Join | **Partial** — select a proxy connection source; no catalog-item join action. | **Yes** — Hand off the selected supported source to Minecraft. |

CraftBox: [Marketplace](marketplace.md), [service](../internal/marketplace/), [workspace](../ui/desktop/frontend/src/Marketplace.tsx).

## Desktop, console and projects

Upstream already has GUI settings/logging and update commands. CraftBox extends them and adds these studio workflows.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Desktop shell | **Yes** — Gio pages and controls; not the Wails/React studio shell. | **Yes** — Wails/React menus, command search, toolbars and workspaces as an alternative to Gio. |
| Document tabs | **No** — no equivalent studio workflow. | **Yes** — Independent documents and save state across supported editors/library areas. |
| Close protection | **No** — no equivalent studio workflow. | **Yes** — Save/discard/cancel, including a shared native-window close dialog. |
| Recent projects | **No** — no equivalent studio workflow. | **Yes** — Supported recent/pinned projects and startup workspace selection. |
| Preferences | **Partial** — GUI/command settings; no equivalent studio preference categories. | **Yes** — Searchable categories for scale/density, motion/tooltips, navigation, preview/export, console and updates. |
| Notifications | **No** — no equivalent studio workflow. | **Yes** — Actionable results, open-folder actions and searchable session notification history. |
| Export task UI | **No** — no equivalent studio workflow. | **Yes** — Progress, cancellation and result actions for supported operations. |
| Log consoles | **Partial** — console/GUI logging; no equivalent independent filtered console tabs. | **Yes** — Multiple purposes/tabs, filters, search, pause/follow, bookmarks, details and export. |
| Command Explorer | **No** — no equivalent studio workflow. | **Yes** — Inspect advertised server commands, overloads and arguments. |
| Command input | **Partial** — CLI and capture commands; no equivalent integrated completion/history workspace. | **Yes** — Completion, history/search and /cli integration. |
| Native terminal | **No** — no equivalent studio workflow. | **Yes** — PowerShell terminal tabs through the supported native bridge. |
| Diagnostics | **Partial** — console/GUI logs; no equivalent retained-session diagnostics viewer. | **Yes** — Retained session logs and in-app access from relevant errors. |
| Account guidance | **No** — no equivalent studio workflow. | **Yes** — Distinguish authenticated online workflows from local/offline work. |
| Updates | **Yes** — update command/GUI; no equivalent stable/preview studio updater workflow. | **Yes** — Stable/preview channels, release notes, scheduled checks and checksum-verified installation when compatible releases exist. |
| Offline rendering assets | **No** — no equivalent studio workflow. | **Yes** — Prepared/bundled vanilla rendering catalog and HDR resources. |

CraftBox: [desktop](wails-ui.md), [documents](document-tabs.md), [projects](studio-projects.md), [preferences/updates](preferences-and-updates.md), [console](console-workspace.md), [terminal](terminal-cli.md), [diagnostics](persistent-diagnostics.md).

## Export and reliability work

These rows distinguish CraftBox fixes from broader capabilities already present upstream. Where individual failure cases or internal safeguards were not compared directly, the bedrocktool column says so; they are not counted as missing upstream features.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Pack packaging | **Partial** — world/pack export exists; the specific packaging safeguards were not separately compared. | **Yes** — Matching manifests/versions and UUID-qualified folders; report copy/decode failures rather than claiming complete output. |
| Readable export copies | **Not verified** — pack handling exists; parity for this export-copy path is not established here. | **Yes** — Use content keys supplied during negotiation for supported server packs. Preserve originals and do not export keys; missing keys cannot be invented. |
| Generated pack identity | **Partial** — generated behavior packs exist; per-visit identity parity was not separately verified. | **Yes** — Keep behavior-pack identity and captured player/model state scoped to the world visit. |
| Custom block conversion | **Partial** — custom block export exists; the listed conversion details were not all verified for parity. | **Yes** — Supported geometry/material options, transform defaults and legacy component conversion for the targeted export format. |
| Actor data | **Partial** — actor capture/export exists; the listed property/physics fixes were not all verified for parity. | **Yes** — Available client definitions, catalog/property updates, enum/attribute precision and supported captured physics. |
| Sign visuals | **No** — no equivalent 3D sign-rendering/model-export module. Capturing sign data is a different capability. | **Yes** — Available sign text in scene generation and supported textured model output, including supported hanging signs. |
| Cache/transfers | **Partial** — cache/transfer handling exists; these individual failure cases were not re-tested upstream. | **Yes** — Additional cache-status bounds, transfer/session ownership and empty-visit handling. |
| Resource/spawn startup | **Partial** — login/resource/spawn handling exists; the investigated failures were not re-tested upstream. | **Yes** — Pack-completion/spawn ordering and diagnostics for the investigated join failures. |
| Save completion | **Not verified** — no blanket comparison of every upstream save/output path was performed. | **Yes** — Staged/atomic replacement in supported workflows, with cancellation and failed saves kept distinct from success. |
| Leaf-material fix in this update | **Not applicable** — no equivalent 3D renderer. | **Yes** — Binary-alpha blend textures become cutouts: visible pixels remain opaque and holes transparent. True intermediate alpha stays blended. Corrects CraftBox's glass-like custom leaves. |
| Style-clearance fix in this update | **Not applicable** — no equivalent scene-look/cloud renderer. | **Yes** — Keep model cloud opening/clearance and fog protection when changing looks, including fitted fog placement while the opening is active. |

CraftBox: [world packs](world-pack-export.md), [export contract](world-export-contract.md), [empty visits](empty-transfer-worlds.md), [cache investigation](client-cache-status-limit-fix-20261102.md), [spawn ordering](pack-completion-spawn-fix.md), [portable assets](portable-assets-fix-20261102.md), [materials](../internal/scene/block_geometry.go), [styles](../ui/desktop/frontend/src/render-settings.ts).

## Credit for upstream work

The generator, dimension, recorded-interface, renderer and player-hook updates include adaptations of work proposed in upstream [PR #361](https://github.com/bedrock-tool/bedrocktool/pull/361) and [PR #362](https://github.com/bedrock-tool/bedrocktool/pull/362). Those authors retain credit for their work. Availability in CraftBox does not imply that every feature originated in this fork.

## Current limits

The following workflows are not implemented in either reviewed project. They are listed explicitly to distinguish them from partially supported features above.

| Function | bedrocktool | CraftBox |
| --- | --- | --- |
| Ingame building hologram | **No** | **No** — Structure Studio's comparison overlay is inside the desktop editor. |
| Saved multitrack audio projects | **No** | **No** — Audio Studio arrangements are preview sessions. |
| Mixed audio export | **No** | **No** — Multitrack listening is available, but mix export is not. |
| Offline offer catalog | **No** | **No** — Marketplace requires online catalog access. |

- Both projects can capture only received chunks and data. Missing packs, never-received terrain and private server code cannot be reconstructed.
- Merged custom dimensions require compatible Bedrock and Script API support.
- Structure Studio's comparison overlay is in the editor. An ingame Litematica-style building hologram is not implemented.
- Static 3D outputs do not include rendered clouds, fog, postprocessing or cinematic camera sequences. STL is geometry only.
- Blockbench is separate software. Export cannot recover missing source authoring data or reproduce every Minecraft shader, Molang program or controller runtime behavior.
- Audio Studio does not currently save audio projects or export mixed tracks.
- Marketplace needs online catalog access. It does not retain an offline catalog; ordinary browser/network caching is separate.
- Archive decoding depends on supported formats, available data and schemas.
- Protected local content may still need the original files and account ownership.
- Current CraftBox source declares Minecraft **1.26.62**, protocol **2225**. Check the release notes of the actual binary you use.

## Verification of the renderer fixes

Checked on **14 November 2026** with failing-before/passing-after regressions, the public Go package suite with the GUI tag, 87 frontend tests, a production frontend build and actual WebGL renders. Three real captured Hive leaf textures retained their exact opaque/hole counts. A style-change cloud test kept protected model pixels unchanged; removing the opening visibly obscured the same model. An independent source review found no actionable regression in the changes.

Temporary regression tests and their fixture scripts were removed after successful execution at the maintainer's request. These checks do not establish compatibility with every live server. Both fixes are included in the CraftBox revision named above.

## Attribution

CraftBox remains an independently maintained GPL-3.0 derivative of bedrocktool. Original contributors, upstream pull-request authors and bundled third-party projects retain their credit and license notices. Blockbench belongs to its own developers. Minecraft and the server/pack assets in the screenshots belong to their respective owners.

There is no official affiliation with Mojang, Microsoft or the original bedrocktool developers. See the [README credits](../Readme.md#credits-and-license) and [LICENSE](../LICENSE).

Return to the [CraftBox README](../Readme.md) for downloads, screenshots and setup.

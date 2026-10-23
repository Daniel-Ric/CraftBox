# Live 3D and GLB export

Open **Live-Karte**, then **3D** in the separate world window. Each world visit
and manually selected region uses its own tab and coordinate space. Drag to
orbit, right-drag to pan, wheel to zoom. Focus the canvas to fly with WASD and
Q/E; Shift increases speed. **Übersicht** fits all captured geometry, and
**Spieler** moves the camera to the selected world's last player position.

**GLB exportieren** exports the complete selected world/region, including
embedded PNG textures. During capture, the result is written to the configured
data directory's `models` folder; after saving, it is written beside the known
`.mcworld`. This is a static model snapshot, not a Minecraft world replacement.
The export button reports the saved path and opens its containing folder.

## Official assets

The inspected official BDS 1.26.52.3 distribution contains `blocks.json`
definitions but **no PNG/TGA block texture images**. Running BDS does not generate
the missing client textures. The importer therefore combines BDS's vanilla
definitions with images and the terrain texture index from Mojang's official
[bedrock-samples](https://github.com/Mojang/bedrock-samples) repository.
Obtain BDS from the [official download page](https://www.minecraft.net/en-us/download/server/bedrock).

From the repository root, using Python 3.10 or newer:

```powershell
python tools/prepare-3d-assets.py --bds C:/Minecraft/BDS --probe tmp/bds-3d-test
```

`--probe` is optional, and must name a fresh directory. It copies BDS into an
isolated directory, starts a hidden test server, loads two columns, places a
stone floor, stairs, glass, water and a slab, saves the world and shuts down.
It never edits or starts the supplied source server directory. The probe uses
ports 19942/19943 and disables LAN visibility. Probe output is in `probe.log`.

The sample ref is resolved to an immutable commit before downloading. Pin a
release/commit explicitly with `--sample-ref`; use `--sample` for an existing
official `resource_pack` directory without a download. `provenance.json` records
the BDS executable SHA256, probe version (when run), sample commit and archive
SHA256. Local sample imports record their source instead. JSON comments,
trailing commas and numerically ordered BDS version overlays are supported.

Imported Mojang files are ignored by Git. Default destination is
`assets/vanilla-3d`; copy that directory beside a distributed EXE, preserving
`assets/vanilla-3d`, or set `CRAFT_BOX_3D_ASSETS` to its absolute location.
The configured data directory's `scene-assets` is also searched. Assets must
be present **before starting capture**. No asset downloads happen during a
capture session. Without assets, geometry uses a visible checkerboard fallback
and the viewer reports missing textures. Readable server packs overlay vanilla
textures; captured custom material instances supply per-face textures and alpha
modes. Unreadable encrypted packs cannot supply images.

## Geometry and lifecycle

- A common Go mesher produces the live viewer and glTF 2.0/GLB export geometry.
- Chunk-local vertices plus chunk node translations preserve world coordinates.
- Full cubes, separate face textures, half-slabs, straight stairs, snow layers,
  carpets, crossed vegetation, transparent glass and static water are supported.
- Pane connections, fences, doors, rails, signs, block entities and other special
  forms are still simplified. Static custom block cube/bone geometries from
  `models/blocks` and `models/block` are supported, including model UVs, pivots,
  bone parents, rotations, inflate and mirrored east/west faces. Material aliases,
  captured face mappings, per-material alpha modes and simple state permutations
  are resolved. A bounded expression reader supports block_state/block_property,
  comparisons, boolean/arithmetic operators and ternaries; arbitrary Molang,
  animated geometry and full client rendering rules are not evaluated. Unsupported
  conditions/models remain explicitly approximate.
- Vanilla textures use 16x16 tiles; custom model textures retain their aspect ratio
  and model UV dimensions, capped at 256 pixels on the longest side. Shelf atlas
  allocation and 18x18 RGBA patches support both sizes with gutters and nearest
  filtering. Animated textures use their first frame. Vegetation
  tint masks receive default grass/foliage colors, not exact per-biome colors.
  Grass-side alpha channels are treated as tint masks, preserving opaque dirt.
  Existing colored pack images are preserved. No mobs, skins or player models
  are included in this first terrain viewer/export.
- Hidden full-block faces are removed across captured chunk borders. Missing
  neighboring chunks are treated as open space. Region filtering happens before
  meshing; cut borders expose faces. Different fluid heights retain exposed
  walls. Region ownership changes invalidate client state and reject in-flight
  snapshots built with stale ownership.
- Live geometry arrives in near-player pages, normally at most eight chunks,
  with a 35 ms meshing and 2 MiB raw geometry budget (one chunk can exceed either).
  Only fully applied cycles are acknowledged; writes during paging are resent
  on the next cycle. Ownership changes reset the cursor and retained geometry.
  Request cancellation stops queued work and active meshing at subchunk boundaries.
- Geometry uses compact base64 binary buffers: exact float32 positions/UVs,
  normalized int8 normals and uint32 indices. A module Worker validates/decodes
  data and transfers buffers without cloning. Main-thread mesh commits are
  limited to two chunks/4 ms per frame; one large chunk/GPU draw can exceed this.
  Only changed atlas cells travel as raw RGBA patches. Idle views render on
  camera/scene changes; hidden views pause requests and mesh commits.
- Canvas, loading notices and captions follow the light/dark theme, including
  changes while open. Failed requests preserve already loaded geometry and
  retry automatically with backoff, with a manual retry button.
- Updates are polled every half-second after completing a cycle; queued block changes follow the existing
  capture update cadence (up to ten seconds). An emptied chunk replaces its mesh.
- Serialized chunks live in a compressed temporary LevelDB, not additional
  retained chunk clones. Immutable database snapshots allow meshing alongside
  capture. The progressive decoded cache holds at most 48 chunks; each mesh reads only
  its primary chunk and four neighbors. The mesh cache is limited to 64 MiB.
  The live viewer limits retained geometry buffers to 256 MiB and reports when
  a smaller selection is needed. Full snapshots/GLB exports retain the
  4096-chunk and 256 MiB geometry limits. Paging has no total chunk-count cap.
- The desktop retains caches until the next command or app shutdown; they are
  not restored after restarting. CLI capture caches are released at finalization.
  WebGL buffers, textures, controls and observers are released on view teardown.

The model file includes normals, UVs, uint32 indices, embedded PNG and separate
opaque/masked/blended materials. The format follows the
[glTF 2.0 specification](https://registry.khronos.org/glTF/specs/2.0/glTF-2.0.html).
The authenticated child-process operations are specified in
[`map-window-api.yaml`](map-window-api.yaml). No caller-supplied export path is
accepted through the bridge.

## Offline export and verification

An extracted `.mcworld` directory can also be exported without the GUI:

```powershell
go run ./cmd/CraftBox render3d --world path/to/world --assets assets/vanilla-3d --out world.glb --dimension 0
```

`--scene-json` additionally saves a complete viewer snapshot for reproducing
rendering issues. Geometry approximation and texture reporting remain the same.
For custom blocks, captured live material definitions provide more information
than disk palettes alone; offline exports may fall back to `blocks.json` mappings.

```powershell
go test ./internal/scene ./ui/desktop ./handlers/worlds ./handlers/worlds/worldstate ./subcommands
python -m unittest discover -s tools -p test_prepare_3d_assets.py
npm.cmd --prefix ui/desktop/frontend test
npm.cmd --prefix ui/desktop/frontend run build
```

The opt-in real BDS integration test reads the stopped probe world:

```powershell
$env:CRAFT_BOX_BDS_FIXTURE=(Resolve-Path tmp/bds-3d-test)
$env:CRAFT_BOX_3D_ASSETS=(Resolve-Path assets/vanilla-3d)
go test ./internal/scene -run TestOfficialBDSFixture -v
```

For browser verification, run the frontend server on port 5182 and execute
`python tools/verify-3d-viewer.py --scene path/to/scene.json`. It uses a test
desktop bridge with the real exported geometry/atlas, verifies camera movement,
updates, export dispatch and 2D selection, and saves screenshots. It does not
claim to exercise a real Microsoft-authenticated proxy session.

## Verified local run, 2026-10-15

- BDS 1.26.52.3 started in isolation, generated the fixture and exited cleanly.
  All eight fixture block types resolved textures; missing/approximate counts
  were zero in this scene. This is not a whole-block-catalog coverage claim.
- Mojang samples commit: `46ba6ea985fb5a92d79a9419198f10dda14c199d`.
- Real-world GLB passed the official Khronos validator with zero errors and
  warnings; one informational unused mask-material entry remains for scenes
  without masked geometry. The GLB contains opaque and blended fixture meshes.
- The affected Go packages, native GUI scene tests, two Python importer tests
  and all 27 frontend tests passed; TypeScript/Vite and Windows GUI builds passed.
- Headless Edge/WebGL verified the real BDS meshes, camera motion, a live chunk
  replacement, export dispatch, switching back to region selection and compact
  layout. No browser errors were recorded. Screenshots and probe data are in
  the ignored `tmp/scene-visual-check` and `tmp/bds-3d-final` directories.
- The global `go test ./...` run is blocked by pre-existing NUL bytes in
  `subcommands/resourcepack-d/resourcepack-d.go` and old incompatible Go files
  in `tmp`. The race detector could not run because the environment has CGO
  disabled and no C compiler; concurrent capture/snapshot behavior tests passed.
- Launchable local build: `builds/CraftBox-3D.exe`; matching imported assets:
  `builds/assets/vanilla-3d`. The original `.idea/vcs.xml` edits were left intact.

## Progressive viewer verification, 2026-10-15

- On this Ryzen 7 5800X3D, a repeatable 64-chunk solid-terrain fixture originally
  needed 804 ms for a full numeric JSON snapshot. After changes, the progressive
  path delivered its first page (including JSON encoding) in 23.32 ms and all
  11 pages in 235.55 ms, using 23,428,608 JSON bytes versus about 29.2 MB before.
  These are single-iteration backend measurements, not end-to-end frame times.
  Reproduce with `go test ./internal/scene -run '^$' -bench 'BenchmarkScene(Cold|Progressive)$' -benchtime=1x -benchmem`.
- Headless Edge/WebGL, using actual stopped-BDS fixture geometry and a test
  desktop bridge, verified a visible first chunk before the second page, worker
  decoding, dark/light canvas colors, camera movement, live replacement, export
  dispatch and recovery after an injected deadline error while preserving meshes.
  Screenshots: `tmp/scene-progressive-check`. The live CubeCraft connection was
  not replayed by this test.
- Regression tests cover cancellation while waiting for the renderer, changes
  before a paging cursor, future targets, region ownership resets, strict bridge
  input/authentication, memory accounting, bounded frame work and worker failure
  before its first response.

## Captured CubeCraft timeout correction, 2026-10-15

The small BDS/terrain fixtures and simulated browser transport did not exercise
large server resource packs. A replay of a local copy of the affected CubeCraft
database (275 chunks, five resource packs) reproduced a first-page deadline with
zero returned chunks. A CPU profile attributed most work to `Pack.ReadFile` and
`Pack.BaseDir`: each candidate file rebuilt the ZIP central directory while
searching for the base directory, causing quadratic work on missing overlay files.
The local Gophertunnel adapter now constructs one immutable ZIP reader, base path
and case-insensitive lookup index when parsing a pack. Copies with different
content keys share this read-only index; archive forwarding stays byte-identical.

The synthetic missing-file regression test previously used 43,156 allocations
with only 102 entries; the corrected lookup passes a ceiling of 50. The same
captured-world replay now returns its cold first page in about 1.5–2.5 ms. All
275 chunks / 1,008,236 triangles were read through the real authenticated
`MapWindowApp.GetScenePage` HTTP client in 39 pages / about 1.25 seconds locally.
This measures replayed capture data, not a new Minecraft login.
Headless Edge also rendered those saved responses without browser errors in
dark mode. The captured scene reports 81 blocks without textures and 2,391
simplified forms, which are existing geometry/asset limitations. Screenshot:
`tmp/cubecraft-repro/browser/cubecraft-275-chunks.png`.

To reproduce, copy a captured database into an isolated directory (skip its
locked `LOCK` file) and set `CRAFT_BOX_CAPTURE_FIXTURE` to that directory,
`CRAFT_BOX_CAPTURE_PACKS` to its readable `resource_packs` directory, and
`CRAFT_BOX_3D_ASSETS` to the imported vanilla assets. Run
`go test -tags gui ./ui/desktop -run TestCapturedScenePageReplay -count=1 -v`.
Set `CRAFT_BOX_CAPTURE_PAGES_OUT` to a JSON filename to record actual responses.
Replay those in a running frontend with
`python tools/verify-captured-3d.py --pages path/to/pages.json --url http://127.0.0.1:5191`.

## Foliage, water and block model corrections, 2026-10-15

- Mojang's oak mask includes RGB (104,99,104); the former exact-gray tolerance
  missed it. Near-gray foliage now receives color while transparent holes and
  existing colored pack pixels are preserved. Pale oak remains intentionally
  untinted. Water receives a default blue tint without discarding its alpha.
- Tall flowers/grass use separate lower/upper textures and crossed plant planes;
  sunflowers add their flower-head plane. Trapdoors use the existing Bedrock
  model's exact bounds for all 16 direction/open/half combinations. Small
  flowers, seagrass, webs and attached vines no longer become full solid cubes.
  Invisible light blocks are excluded. Fence/pane/wall arms follow captured
  neighbours. Partial cube UVs crop the texture instead of stretching it.
  Cutout textures do not incorrectly occlude geometry visible through holes.
  Renamed iron chains resolve the legacy chain1/chain2 textures and thin bounds;
  linked ring geometry is still approximate.
- The real CubeCraft pack gallery rendered all 96 custom default-state models
  with no missing materials. Separate tests cover state changes, aliases, mirrored
  textures, model UV dimensions, transformed normals and pivot defaults. This is
  not a proof of every possible state or animation in every resource pack.
- Headless Edge rendered the custom gallery and a separate real-texture fixture
  of tall flowers, trapdoors, leaves, connected fences/panes and blue water without
  browser errors. The custom GLB passed Khronos validation with zero errors,
  warnings or informational messages. Visual artifacts are in
  `tmp/cubecraft-materials`.

The implementation was checked against Microsoft's
[material instances](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/blockreference/examples/blockcomponents/minecraftblock_material_instances?view=minecraft-bedrock-stable),
[block transformation](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/blockreference/examples/blockcomponents/minecraftblock_transformation?view=minecraft-bedrock-stable), and
[geometry](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/visualreference/geometry.v1.19.30?view=minecraft-bedrock-stable) references.

## StartGame integer flag correction, 2026-10-15

An observed `interface conversion: interface {} is int32, not uint8` after
"using custom blocks" was reproduced before scene initialization, in behavior
definition normalization (`minecraft:placement_filter.allowed_faces`). The same
strict byte assumption existed in trait enablement and liquid-detection flags.
Those fields now accept integral NBT tags and booleans through an explicit
converter; placement filters retain all six face-mask bits. Strings/floats are
not silently converted. A regression test executes the StartGame processing
order (behavior definitions, runtime block registry, scene definitions) with
Byte/Int/Boolean flags, verifies four distinct directional states, and passes
without the former panic. This is a local login-processing test, not an
authenticated remote login.

## Fluid and block model correction, 2026-10-15

- Falling liquid depth uses a separate falling flag. Fluid under another fluid
  cell fills its full height; internal top/bottom surfaces are removed across
  water/flowing-water aliases, storage layers and chunk boundaries. This removes
  the repeated horizontal shelves in captured waterfalls.
- Pressure plates use inset 14-by-14 pixel footprints and 1/16 block height,
  reduced to 1/32 when powered. Lanterns use a six-pixel-wide body, a separate
  cap and crossed attachment quads, model UV regions and hanging/standing states.
- Captured CubeCraft portal corners exposed incorrect model-space rotation
  signs: the actual rotated cube extended to X=18 instead of fitting X=-2..8.
  Model X/Z angles are converted separately from world/block transformations.
  The coordinate conversion follows the
  [Blockbench Bedrock importer](https://github.com/JannisX11/blockbench/blob/master/js/formats/bedrock/bedrock.js).
- Resource-pack access now follows the server's UUID/version stack, rather
  than download completion order, and excludes downloaded inactive packs from
  rendering. Capture access retains every downloaded archive for replay.
- Replaying the latest saved world also revealed a stale linked function
  declaration in the merge registry. Its signature now matches Dragonfly's
  hash function, including the scratch argument and second return value.

Regression tests reproduced the former fluid shelves, full-cube forms, portal
corner protrusion and registry panic before the fixes. The latest captured world
replayed 735 chunks / 5,486,228 triangles through the actual local HTTP bridge in
6.10 seconds with no unresolved image references. Headless Edge checked the real
four portal chunks (39,670 triangles) and a separate real-texture forms fixture;
the latter GLB passed Khronos validation with zero errors or warnings. The full
735-chunk browser replay reached the existing 256 MiB geometry budget, so that
browser run is not claimed as a complete visual pass. Artifacts are under
`tmp/forms-repro`. Animated texture frames, entities and all native block shapes
are not fully reproduced by these changes.

## Portal handedness, crystals and planted pots, 2026-10-15

The preceding model-angle fix was incomplete: Bedrock model X points opposite
to world X. Model vertices now reflect X before block-component transforms;
triangle winding and east/west face lookup follow that reflection. UV corner
orders also follow Bedrock's explicit-face conventions, signed rectangles and
inverse texture rotation. Box-UV bottoms have their separate convention. These
were checked against Blockbench's Bedrock importer and
[cube UV mapping](https://github.com/JannisX11/blockbench/blob/master/js/outliner/types/cube.js).
An optional saved-world test verifies that 65 actual portal edge/inside pairs
point inward, including top beams. World/block-component rotations were not
changed to compensate for the formerly missing model conversion.

Amethyst clusters and all three bud sizes use two cutout planes, rotated to each
of six attachment faces. Anvils have a foot, shoulder, narrow stem and oriented
head. Flower pots have explicit walls/rim and recessed soil. Their `PlantBlock`
name is passed from captured block entities into the same scene snapshot as the
chunk, so live updates and GLB/offline export include the contained flowers.
Only the minimal pot visual metadata is retained, atomically with chunk data;
removing a flower invalidates the old mesh. Older worlds without PlantBlock
data cannot reconstruct their original contents from the empty pot alone.

Jungle leaf TGA images contain a mostly gray tint mask with colored gold details.
The former all-gray test disabled tinting for the whole image. Gray-mask-dominant
vegetation now tints its gray pixels while preserving colored details; explicit
material tint instructions still take precedence. Saplings remain cutout plants.

Regression tests cover the model coordinate/UV mapping, 24 crystal attachment
states, twelve anvil states, flower-pot changes and clearing, and mixed-color
leaf masks. The real latest capture's six portal chunks (134,988 triangles) and
17 pot-containing chunks (274,444 triangles) replayed with no unresolved image
references. The native detail gallery rendered in headless Edge; its GLB has
zero Khronos validation errors/warnings (one unused-material informational item).
Artifacts are under `tmp/model-corrections`.

The set-name chat string is valid UTF-8 in both YAML and Go. Its German wording
and prefix separator now use simple ASCII glyphs in addition to Minecraft's
section-sign formatting codes, avoiding the affected special glyphs.

## Gates, anvil skins and plant surfaces, 2026-10-15

The disconnected railing was confirmed against the actual capture: at X=1925,
Y=116, two birch fences at Z=928/936 enclose seven closed, east-facing birch
fence gates. Gates formerly fell back to one post. Gates now include posts,
rails and braces in all four directions, open/closed and wall-lowered states.
Fences attach to gate sides, and wooden/nether-brick fences remain separate.

Anvil heads sample a texture whose opaque horizontal interval is only pixels
3..13. World-space UV cropping after turning an anvil sampled the transparent
margin and exposed light surfaces underneath. Each component now receives UVs
in its canonical model coordinates before its geometry and normals rotate;
only the top of the head uses the head skin, while foot, stem and side faces
use the base texture.

Saplings use two inset diagonal planes with proper unit normals. Vines now emit
one zero-thickness quad per attachment bit, rather than six faces around a thin
box. The hanging plants beneath the captured anvils are jungle leaves and vines,
not saplings; actual leaf cubes keep their native shape.

Regression tests cover all sixteen gate state combinations, side attachment,
wood/nether separation, oriented anvil UV bounds and sapling/vine plane counts.
The gallery and 18 actual gate-containing chunks (551,090 triangles) rendered
in headless Edge with no browser errors or missing textures. The gallery GLB
passed Khronos validation with zero errors/warnings (one unused-material info).
Artifacts are in `tmp/gate-corrections`. An accidental cleanup deletion of model
branches was caught by the regression suite, restored and independently reviewed
before the production executable was built.

## Centered orbit mode, 2026-10-15

The "Zentriert drehen" toolbar toggle fits the geometric center of the loaded
model in all three axes. A bounding sphere and the narrower horizontal/vertical
field of view determine the camera distance, so wide, tall and rotated models
remain visible even in portrait windows. Orbit and wheel zoom remain enabled;
pan, modified drag, WASD/QE flight, arrow navigation and player recentering are
disabled while the mode is active. Residual pan damping is canceled without
moving the fixed target. Bounds changes at the end of a live page cycle and
window resizing recompute the framing while preserving the view direction.
The preference lives in App and survives world changes and 2D/3D remounts.

Six camera tests project the corners of wide/tall/cubic bounds across five
aspect ratios and three orbit directions. All 33 frontend tests and the
production frontend build pass. Headless Edge exercised the native gallery and
18 captured chunks / 551,090 triangles: right drag, Ctrl-drag and movement keys
leave the rendered view unchanged; orbit and zoom change it; unlocking restores
pan; portrait resizing and a 2D/3D remount preserve the mode. Image comparisons
exclude toolbar hover and the keyboard focus outline so those UI effects are
not mistaken for camera movement. Artifacts are under
`tmp/gate-corrections/centered-world-browser`.

## Recorded entities, signs and chains, 2026-10-15

Client entity definitions, entity geometry, render controllers and animation
controllers are read from the active pack stack. The shared geometry compiler
resolves variant arrays, property conditions, bone visibility, initial animation
controller states, constant pose channels and first keyframes. This is a static
snapshot renderer, not a complete Minecraft animation/Molang interpreter.
Camera-dependent banner entrance scripts use their visible base pose.
Entity yaw includes the 180-degree model-front correction: native model noses
point toward -Z, while Bedrock yaw zero faces +Z. Nametag quads already face +Z
and keep their separate orientation. Four cardinal yaw tests guard this mapping.
Pack-defined unlit and translucent material passes retain their material mode.

Live actor metadata is copied into immutable chunk sidecars. Packet updates do
not mesh on the network thread. Actor-only chunks and moves between chunks emit
proper deltas, including empty replacement geometry at the previous position.
Players, both CubeCraft buddy prefixes, owned entities and attachment chains
rooted at players/pets are excluded. PlayerList and StartGame Unique-IDs match
the Unique-IDs in riding links; Runtime-IDs are deliberately not substituted.
The exclusion is also saved as CraftBoxExcludeFrom3D for later offline exports.

Signs have boards/posts, orientation and front/back text instead of full cubes.
Native sign images are used where available. UTF-8 text uses an embedded font
fallback, section-code colors/bold/reset and pack-provided private glyph pages.
Nametags remain visible when their carrier has scale zero. In the live viewer
their unlit quads face the camera; GLB stores their captured static orientation
using KHR_materials_unlit. Dynamic labels reuse atlas slots, and each atlas cell
retains only its latest changed revision rather than an unbounded update log.
Chains use two diagonal cutout planes with explicit UVs for the three populated
pixel columns in Mojang's chain texture, oriented along the captured axis.

Evidence: the completed 11:40 capture contains 405,667 packets, 72 saved actors,
34 signs and 60 retained visible holograms. An asset gallery renders 22 recorded
model types and four real holograms with no missing image references. A 20-chunk
lobby replay and 48-chunk SkyWars replay (798,920 triangles) pass through the real
HTTP paging/decoding/rendering path with no browser or WebGL shader errors.
The gallery GLB passes Khronos validation with zero errors and zero warnings.
Regression tests cover player link ordering and distinct IDs, immutable actor
snapshots, movement deltas, invisible carriers, text slot reuse, UTF-8 signs and
billboard dimensions. Local artifacts: `tmp/entity-corrections`.

References: [render controllers](https://learn.microsoft.com/en-us/minecraft/creator/documents/behaviorrendercontrollers?view=minecraft-bedrock-stable)
and [Bedrock raw text formatting](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/rawmessagejson?view=minecraft-bedrock-stable).

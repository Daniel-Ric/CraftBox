# Volumetric atmosphere in Render and Cinematic

The shared **Bildgestaltung → Atmosphäre** tab controls both editors. Enable fog and clouds separately. Settings are stored with render projects. Heights use world Y in blocks; cloud height is the bottom of the layer, and thickness determines its top. To place clouds between towers, put the layer inside their world-height range. For a cloud ceiling, move the bottom above the tallest building.

## Controls

The atmosphere revision adds **14 cloud presets and 12 fog presets** in separate searchable library dialogs. The cloud library includes fair-weather puffs, cloud seas, broken ceilings, storms, anvil towers, cirrus, small cloudlets and lens clouds. Five fog density profiles provide mist, banks, valley layers, uniform haze and turbulent columns. Applying one library's preset preserves the other effect's settings.

Both effects now offer independent persistent random seeds and redistribution buttons. Additional controls tune cloud grouping, roundness, light absorption and fog warping. The eight cloud shapes have distinct density transforms; presets also tune their dimensions, coverage and color.

Height can be absolute world Y, or an offset from the loaded object's bottom, center or top. Negative offsets are supported using signed text entry, a slider, ±10-block steps and quick values. For example, choose “Ab Objektoberkante” and −32 to begin the cloud layer 32 blocks below the tallest point; use +24 to place it above the object. Valid offsets span −2048…4096 blocks. Old projects retain absolute world-height interpretation.

- Fog: color, density, height, height falloff, variation, size of drifting wisps, clear camera-near distance, light scattering and forward scattering.
- Clouds: Cumulus, Stratus, Cumulonimbus and Cirrus; color, density, coverage, base height, thickness, feature size, surface detail, edge erosion, shape warping and terrain-shadow strength.
- Wind/time: speed in blocks per second, direction, animation toggle and start-time offset. A paused Cinematic sequence holds the atmosphere at that sequence time. GIF/MP4 frames receive explicit timestamps; WebM follows playback time. Static renders can freeze the atmosphere using the animation toggle.

Start with coverage around 0.4–0.6 for separated clouds. High coverage intentionally produces a cloud sea or overcast layer. Reducing fog density preserves foreground contrast. “Fein” uses full-resolution volume rendering and more samples; other quality modes use depth-aware upsampling of a half-resolution volume buffer.

## Rendering and interaction

The volume pass integrates scattering and extinction through a deterministic periodic 3D density field in scene coordinates. Each cloud type changes its density profile. Six samples toward the current sun approximate cloud self-shadowing. Cloud density also attenuates directional lighting on loaded terrain materials, preserving their atlas and foliage shader hooks.

The revised field combines Perlin and Worley channels with nonperiodic seeded macro weather, independent cloud/fog offsets, variable lower/upper envelopes and type-specific shapes. Near-biased sampling and mipmapped 3D filtering reduce long-ray aliasing. A configurable soft distance fade and continuous grazing-angle attenuation soften the distant volume boundary; attenuation blends smoothly as a camera enters or leaves the layer.

The actual cutout-aware terrain depth texture limits integration at visible geometry. Rays are reconstructed from the active camera's inverse projection and world transform, supporting both orthographic framing and perspective flights. Depth-aware reconstruction reduces cloud bleeding along silhouettes. Occupied height intervals are sampled separately, so distant fog and thin cloud layers do not consume each other's sample budgets. The pass runs before bloom and output/color grading and participates in image/video export.

This is a real-time single-scattering approximation. Opaque/cutout terrain controls volume occlusion; blended glass/water do not supply opaque depth. Cloud terrain shadows approximate optical depth at the sun-projected layer position. World heights and weather layers are artistic scene controls, rather than a fluid simulation.

Native vine sheets at 1/64-block offsets now receive wrapped canopy lighting and corresponding AO normals. Crossed grasses retain their existing canopy treatment; solid cube faces retain their normals.

Primary technical references: [Three.js volumetric-cloud example](https://threejs.org/examples/webgl_volume_cloud.html), [DepthTexture](https://threejs.org/docs/pages/DepthTexture.html), [render targets](https://threejs.org/manual/pages/rendertargets.html). The shaders here are implemented for CraftBox's existing WebGL/postprocessing pipeline.

## Verification

- `RenderEditor.test.tsx`: shared cloud/fog controls for Render and Cinematic, including changing type, height and color.
- `foliage-lighting.test.ts`: native vine offsets recognized; solid cube faces preserved. Failed before the fix, passed afterward.
- `tools/verify-atmosphere-depth.py`: actual WebGL pixel checks for geometry occlusion, perspective/orthographic rays, deterministic identical timestamps, wind motion, thin clouds with zero/sparse fog, and improved vine lighting.
- `tools/verify-atmosphere.py`: actual 48-chunk terrain / 793,730 triangles; four distinct cloud outputs, fog/height changes, real loaded terrain shadow changes and PNG export at 720×405. No shader/browser errors.
- `tools/verify-cinematic-editor.py --atmosphere`: actual terrain with clouds and fog enabled during GIF/MP4/WebM checks.

Evidence is saved under `tmp/atmosphere/`. Independent review identified the atlas-hook ordering and disjoint-slab sampling problems; both have failing-before/passing-after GPU or real-terrain regressions, followed by a clean review.

Revision evidence: `tmp/atmosphere-v2/views/` contains below/above/horizon/inside/alternate-seed views. The updated real-terrain run checked both library dialogs, signed height entry, all eight distinct cloud outputs, terrain shadows and PNG export. GPU regression checks confirmed identical-time determinism, depth occlusion, thin layers, a changed random seed, and a mean channel delta of only 0.0137 while crossing a cloud boundary at a grazing angle. Review found duplicate shape branches and a binary boundary transition; both were corrected and the follow-up review found no important remaining issues.

The complete frontend suite passed **90 tests in 28 files** using `npm.cmd test -- --maxWorkers=2`; the production TypeScript/Vite build passed. An initial parallel run while software-GPU browser tests were active hit three existing UI test timeouts; limiting test worker concurrency resolved the resource contention without changing those tests. The subsequent focused Cinematic/export suite passed all five tests after the recording-start adjustment.

The final Cinematic run passed GIF frame decoding, MP4 decoding at 720×406 / 0.25 seconds, and WebM frame decoding at the preview's 637×358 resolution, with no shader/browser errors. The WebM test originally produced a header-only 110-byte file. Recording now waits for recorder start, renders/requests the first frame and yields before starting playback; the final WebM contains decodable video (14,240 bytes). A focused independent review found no important races in the correction.

The revised rendering also passed the same Cinematic GIF/MP4/WebM decoding checks (`tmp/atmosphere-v2/cinematic/`), again without shader/browser errors. The real-terrain library/height/eight-shape run and PNG export passed. The complete suite was repeated serially after the software-GPU checks, because a concurrent run hit one unchanged App test's 5-second timeout.

Final revision verification: `npm.cmd test -- --maxWorkers=1` passed **92 tests in 30 files**, with no changes to test timeouts. TypeScript/Vite production build passed. Library screenshots were visually inspected. The camera-below/above/horizon/inside views were inspected after the near-biased sampling and filtering changes, including removal of the previously visible interior sampling band. The production render, GIF, MP4 and decoded WebM outputs all passed.

## Original atmosphere build

- Installed: `D:/CraftBoxs/craft-box.exe`
- Version: `1.59.0-studio-atmosphere.20261020-dev`
- Previous version backup: `D:/CraftBoxs/craft-box-before-atmosphere-20261020-073539.exe`
- Build, staging and installed SHA-256: `0E482903AAEA60ACA3D589A29C23FB672103026B5F2474146088C23C68E07432`
- Native resource check passed: original icon identity, Wails group 3, nine images, manifest present.

Restart CraftBox to load the new executable. No running user application was terminated.

## Previous delivered revision (v2)

- Installed: `D:/CraftBoxs/craft-box.exe`
- Version: `1.59.0-studio-atmosphere-v2.20261020-dev`
- Previous executable backup: `D:/CraftBoxs/craft-box-before-atmosphere-v2-20261020-084653.exe`
- Build, staged and installed SHA-256: `67DDDF6146B67AB117D6FFA66B931C8D127AAA6E4B91EA0D1607133FEE57DDA3`
- Resource check: original icon identity, nine icon images and application manifest passed.

The current revision supersedes the original atmosphere build recorded above. Restart CraftBox to load it; no running application was terminated.

## Cloud bodies, reusable render styles and empty Cinematic layout

The puffy cloud families now use seeded clusters of overlapping three-dimensional lobes, with distinct vertical profiles, surface distortion and eroded edges. They occupy a soft bounded region around the object, controlled by **Wolkenbereich um Objekt**. This prevents a low cloud bank from extending to the horizon as a continuous slab. Stratus, cirrus and lenticular intentionally retain their layer-shaped profiles. Height, thickness, feature size, coverage, seed and wind remain independent controls.

The Looks panel includes **Wolkeninsel · Tageslicht**, **Wolkeninsel · Goldenes Gegenlicht** and **Wolkeninsel · Weicher Dunst**. These combine fine render quality, cloud bodies, object-relative heights, low fog and matching light/color settings as editable starting points. They approximate the reference atmosphere within the existing real-time renderer; they are not an offline multiple-scattering renderer.

**Voreinstellungen speichern / laden …**, above the inspector tabs in both editors, stores the complete render settings in IndexedDB. JSON import/export transfers the same settings between worlds or installations. Custom HDR sky files up to 16 MiB are embedded. Camera position and world selection remain independent; object-relative height references adapt to the destination world's bounds. Imports validate render domains, shadow-map sizes, atmosphere values and embedded asset limits before application.

The empty Cinematic workspace no longer relies on the lazy studio stylesheet to hide the lighting inspector. The inspector stays in its right-hand column when opened, even before a world is selected or while the console is open. `tools/verify-cinematic-empty.py` reproduced the original failure before the fix and passed eight viewport/console/inspector states afterward.

Additional verification: `tools/verify-render-styles.py` passed actual editor save, JSON export, application to a second world, IndexedDB persistence after reload, deletion and JSON reimport. Unit coverage includes complete snapshot isolation, bundled look roundtrips, invalid imported values and embedded custom HDR preservation. Actual WebGL checks passed occlusion, deterministic timestamps, wind/seed variation, thin layers, grazing transitions and vine lighting. Side, above, below and interior cloud views were inspected. The 48-chunk terrain run passed all eight cloud outputs, visible terrain shadow changes, fog-height changes and PNG export. The shadow fixture now uses scattered cumulus bodies so that the terrain under test is visible instead of hidden by an opaque stratus ceiling.

The final Cinematic run passed loaded-world layouts, camera/timeline editing, GIF frame decoding, MP4 decoding at 720×406 / 0.25 seconds, and WebM decoding at 637×358, with clouds and fog enabled and no browser/shader errors. Evidence: `tmp/cloud-bodies/cinematic/`, `tmp/cloud-bodies/empty/`, `tmp/cloud-bodies/styles/`, `tmp/atmosphere/terrain/`, `tmp/atmosphere-v2/views/`. The full frontend suite passed **94 tests in 31 files** (`npm.cmd test -- --maxWorkers=1`). TypeScript/Vite production build, native production build and Windows resource verification passed. Follow-up independent review found no important remaining issues.

## Current delivered revision (cloud bodies)

- Installed: `D:/CraftBoxs/craft-box.exe`
- Version: `1.59.0-studio-cloud-bodies.20261020-dev`
- Previous executable backup: `D:/CraftBoxs/craft-box-before-cloud-bodies-20261020-094415.exe`
- Build, staged and installed SHA-256: `DF4FEEF81508DE32559C4CF0372633B85E1121F01399BE29D4FCB02D5B225707`
- Original app icon identity retained; Wails icon group 3, nine icon images and manifest verified.

This revision supersedes v2 above. Restart CraftBox to load it. No running user application was terminated.

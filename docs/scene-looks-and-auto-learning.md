# Scene library and reference-based Auto

## Current update — 2026-11-06

### AAD cloud composition follow-up

The explicit Clouds option previously overwrote complete cloud recipes with a generic 48-block bank and multiplied cloud height/thickness by the entire map span. AAD now preserves the selected cloud recipe (including its base seed and volume settings). A non-cloud seasonal reference borrows a complete light-compatible cloud recipe. Optional tint behavior remains unchanged.

The feature worker builds a 24×24 upper-surface profile using upward-facing samples. Ground height comes from spatial upper-surface quartiles, not the minimum Y of a deep captured terrain box. AAD compares the curated recipe against 24 fitted layouts. Candidates retain rounded cloud bodies, add up to three editable cloud volumes around the terrain perimeter, and protect the build with an opening above the ground line. The geometric score evaluates physical intersections, camera-to-landmark ray/ellipsoid obstruction and foreground framing. Perspective and orthographic views use different sightlines. This is a bounded placement heuristic; it does not claim to score rendered artistic quality.

The current camera is attached after evidence-cache lookup, so navigation changes framing without rescanning terrain. Recipes remain the fallback if surface evidence is missing or the fit scores worse. The report identifies the selected composition and comparison count.

Validation: 62 frontend tests, TypeScript/Vite production build, tests for deep terrain, deterministic placement, preserved tint/seed, camera direction, clear-atmosphere overrides and near-cloud/far-landmark perspective alignment. A recorded full Hive fixture contains 1,755 chunks and 6,139,178 triangles; analysis plus layout fitting took approximately 640–783 ms across test runs (excluding initial loading). 25 candidates were evaluated using 462 upper-surface cells. A browser regression also verifies that a cache hit refreshes camera metadata without rescanning geometry. WebGL renders compare the curated recipe, previous AAD override and terrain-fitted AAD at the same camera and reduced preview quality. No page/shader errors were observed. Artifacts: `tmp/hive-aad-composition-proof.json`, `tmp/hive-cloud-curated.png`, `tmp/hive-cloud-old.png`, `tmp/hive-cloud-improved.png`. These figures are environment-specific, not latency guarantees.

Delivery: `builds/Craft-Box.exe`, version `1.65.2-aad-cloud-composition-dev`.

### Analysis performance and cloud integration follow-up

Auto analysis no longer awaits `ensureFullDetail()` / the unrelated secondary cloud-clearance rasterizer. The render scene is already fully loaded with `previewStep=0`. It uses at most 48,000 deterministic, mesh/material-stratified probes with interval-area weights, reusable geometry buffers, scalar triangle arithmetic and approximately 6 ms cooperative sampling slices. Packed samples (not the entire terrain) transfer to a dedicated feature-analysis worker. Area-weighted luminance percentiles now refine exposure and highlight protection without relying on average brightness alone.

Geometry commits and atlas patches invalidate a single-entry per-view evidence cache. A source-name change also prevents reuse. Both cached and computed results reject changed/disposed scenes; cancellation reaches sampling and terminates the worker. Style options rerank references against cached evidence. The modal reports actual preparation, surface, palette, spatial-shape, event-evidence and reference-matching phases, sample counts, elapsed time and completed steps. Progress is work-based rather than a guessed countdown.

Cloud integration now uses stable integer weather hashes, rotated 3D noise with independent warp axes, quintic noise interpolation, stratified ray/shadow samples and explicit noise LOD inside divergent marching branches. This removes coherent slice alignment and reduces directional noise ribs. Cloud banks have deeper variable bodies; dual-lobe scattering, interior attenuation and powder lighting improve depth. A small depth/transmission-aware reconstruction filter reduces sample grain while preserving geometry edges. These changes are shared by built-in cloud types and authored cloud volumes; existing colors, seeds and quality controls remain available, though the revised field changes the exact appearance of old seeds.

Validation: 56 frontend tests and TypeScript/Vite production build. Browser checks exercise actual worker progress, real abort, cache invalidation and rejection of stale cached results. A synthetic 1,755-mesh / 6,154,785-triangle scene took 117 ms for bounded analysis in the test browser (10 main-thread timer ticks); this is not a full Hive-lobby benchmark. A real 48-chunk / 793,730-triangle terrain fixture took 513 ms, with 0.2 ms cache reuse. Timings are environment-specific and exclude initial world loading. All eight cloud types, additional base models and an authored volume rendered in 12 WebGL cases without shader errors. Evidence: `tmp/analysis-performance-proof.json`, `tmp/actual-terrain-analysis.json`, `tmp/auto-analysis-live-progress.png`, `tmp/cloud-before-*.png`, `tmp/cloud-final-*.png`.

Delivery: `builds/Craft-Box.exe`, version `1.65.1-auto-performance-dev`.

### Scene library and evidence-based styling

The shared Render/Cinematic library now contains **100 looks**, with 40 new references for autumn, spring, summer, winter, Halloween, Easter, Christmas, architecture, interiors and film. The browser adds combined multi-word/category/time filters, sorting, persistent local favourites and an inspect-before-apply detail panel. Palette illustrations are labelled as palettes, not screenshots of the user's world. Camera and output settings remain intact when applying a look.

Auto now combines the existing area-weighted visual measurements with bounded block-family counts from the native mesh. Snow/ice coverage is measured on upper surfaces, so buried rock depth does not decide whether a map is winter. Hidden solid blocks do not inflate decoration evidence. Custom block identifiers can supply decoration hints, but arbitrary orange concrete, terracotta and white quartz do not establish events.

Halloween requires multiple independent decoration families; pumpkin and jack-o-lantern counts are correlated and count as one family. A pumpkin farm, orange palette or stale server title alone cannot enable Halloween looks. Easter combines egg/rabbit decorations with supporting signals; Christmas needs gift decorations plus snow or festive decorations. Autumn uses warm cutout foliage and named leaf hints rather than warm walls. The analysis reports evidence and uncertainty. Geometry-only/older projects without block metadata stay explicitly uncertain, with a manual scene-character override available.

Theme and cloud choices compose: a manually selected holiday can still use cloud banks. Auto refines reference lighting within a bounded 24-degree azimuth / 5-degree elevation envelope using measured normals, reduces oversaturation and limits sharpening on dense geometry. This remains a deterministic local heuristic, not a trained semantic image model or a guarantee that every custom event can be recognized.

Cloud color is directly accessible at the top of Atmosphere controls, with a custom color picker and swatches. The global tint also multiplies authored cloud-region colors while fog stays independent. **Automatically match cloud color** is optional and off by default. Auto preserves the current manual tint unless enabled; when enabled it derives a restrained tint from the selected light, scene evidence and fog palette. The existing project and render/export pipeline carries the color setting.

The original MCBE PvP Title pack also exposed a signed-size geometry bug: inward-facing negative cube shells were turned outward and hid the text atlas. The preview now retains the authored winding and matching normals. Positive and even/odd-negative cube cases are covered, and the original pack renders with its blue text.

Validation: 49 frontend tests, TypeScript/Vite build, native scene/asset-library/desktop tests and browser checks for 100 entries, filtering, favourites, explicit apply, theme evidence, optional tint, light/dark themes and smaller windows. Native tests check snow coverage over shallow/deep terrain; geometry tests check signed-shell winding. Visual evidence: `tmp/looks-*.png`, `tmp/title-pack-after.png`.

Three real-terrain browser renders (winterGlacier, halloweenLantern and springSakura) also exercise the volumetric cloud shader with color tinting. These are smoke checks, not an aesthetic review of every look. Evidence: `tmp/style-render-*.png`. Windows production binary: `builds/Craft-Box.exe`, version `1.65.0-scene-styling-dev`.

## Historical implementation — 2026-10-20

The shared Render/Cinematic inspector now offers 60 complete scene looks: 20 existing looks and 40 additions. The library groups them into cloud worlds, nature/coast, fantasy/history, winter/mountains, desert/savanna, city/night, interiors/studio and film/neutral. Search matches descriptions and tags as well as names. All looks are also available in the new-project dialog.

Applying a look starts from render defaults, then applies its complete reference configuration. This avoids inherited fog, cloud or studio-light settings from a previously selected look. Output size, aspect, transparency and pitch lock remain unchanged.

## Auto selection

This supersedes the former independent 120-direction lighting search described in earlier versions of `auto-art-direction.md`.

Auto still measures the actual mesh, texture atlas and emitter metadata locally. It ranks the scene library using authored feature exemplars, model form, vegetation/water/bright/warm/dark surface proportions, color diversity, user intent and optional time/atmosphere constraints. These are visual proxies, not semantic biome recognition.

The winning scene supplies the lighting and grading. Auto preserves its sun direction, energy and color treatment; bounded adjustments cover atmosphere dimensions, deterministic distribution seeds, bright-material protection, dark-material exposure and emitter budget. Explicit time overrides intentionally replace the reference's solar rig. The report names the selected scene and explains all resulting settings.

This is reference-based calibration and nearest-example preference learning, not neural-network training on rendered images. It neither evaluates rendered candidate images nor guarantees subjective perfection.

## Personal feedback

Select a scene look, then choose **Diesen Look für Auto merken**. CraftBox stores the look ID and eight normalized model features in localStorage (`craftbox-direction-examples`, schema version 1), capped at 64 examples. No model files or images leave the computer. Similar feature profiles increase that look's ranking. Explicit atmosphere constraints still take precedence.

Remembering another look for essentially identical features replaces the earlier preference, including when its look ID differs. Invalid stored data is ignored; storage failures produce a visible message. The memory belongs to the local application profile and persists across editor changes/restarts.

## Verification

- 117 frontend tests pass, including all 40 new profiles' schema roundtrips, deterministic reference selection, preservation of reference lighting/grading, model-scale adaptation, local learning and corrected preferences.
- Production TypeScript/Vite build and desktop integration tests pass.
- `tools/verify-scene-learning.py` exercises the production UI with the real 48-chunk terrain fixture, including both editors, all 60 library entries and remembered/corrected look selection.
- `tools/verify-scene-previews.py` captured and visually checked four representative new looks (peach clouds, clear winter, lagoon and warm interior), with no shader/browser errors. Evidence is under `tmp/scene-learning/`; these are samples, not an exhaustive visual rating of every look.

The profiles configure existing rendering features. They do not add weather simulation, rain geometry or new emissive blocks. The cloud looks reuse the existing continuous cloud-bank renderer.

## Delivery

Version `1.59.0-studio-scene-library.20261020-dev` installed at `D:\CraftBoxs\craft-box.exe`. SHA256 `86F9A97C279E1F091D6502B8DEB85AAC09296330644E0502D2BEF96C317BFDA8`. Previous binary retained as `craft-box-before-scene-library-20261020-200154.exe`. The existing icon group (nine images) and application manifest were verified in the Windows executable.

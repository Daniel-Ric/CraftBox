# Portable studio project contract

`.craftbox` is a ZIP container with `project.json` (format `craftbox-project`, version 1, opaque validated editor document) and sequential `scene/000000.json` pages containing the complete terrain geometry, texture atlas patches and emitter metadata used by the renderer. No external map path or resource pack is required to resume rendering. It is a studio document, not a replacement for the Minecraft world-editing format.

The editor document includes all RenderSettings, custom HDR bytes when present, cinematic sequence, selected keyframe/playhead, viewport pose and 2D map presentation settings. The reader validates the format/version, bounded JSON sizes, scene page sequence and atlas dimensions before adopting a document. Saving uses a temporary sibling and atomic replacement; cancelled/failed saves never mark a document clean.

Desktop bridge additions:

- `SaveStudioProject(worldId, name, documentJson, saveAs) -> path`: package a stable complete paged scene and editor document. Empty path means the native save dialog was cancelled.
- `OpenStudioProject() -> { world, document, path }`: pick and validate a portable project, staging its source until frontend document validation succeeds. The first `GetRenderScenePage` for its ID adopts the staged source. `CloseRenderWorld` for that ID cancels staging without changing the active project or save path. Cancellation returns an empty result.
- `SetStudioProjectDirty(bool)`: mirrors unsaved editor state to the native window close gate.
- `ConfirmStudioProjectClose()`: allow one native close after the frontend save/discard dialog succeeds.

Each editor now has independent document tabs. Switching tabs and opening another document retains existing state; closing dirty documents asks save/discard/cancel. Native window closure uses a shared selectable list across all five document areas. Save must finish successfully before continuing; discard continues without writing; cancel keeps the editor and its state. A saved snapshot establishes the clean baseline. Sessions start empty. See [Document tabs](document-tabs.md) for scoped bridge operations and referenced-content projects.

Loaded files establish their clean baseline immediately, before terrain readiness. Settings edited during loading remain unsaved. Free camera and orbit changes mark the document dirty immediately, with a debounced camera snapshot for storage. Export and initialization frames do not mark user camera edits.

## Maps, atmosphere and loading

The Render editor has a separate orthographic 2D map mode. Outside the terrain the output is always transparent, including PNG export. Camera rotation/zoom, grid, elevation tint, contour lines, terrain shading and saturation are editable. World-space markers support text, symbols, colors, scale and repositioning; annotations are included in PNGs and portable projects. Resetting map styling preserves annotations. Zoom is bounded to 0.1–20; marker placement can be cancelled.

Both editors defer scene effects, sky environment loading and drawing until all terrain pages and queued geometry are complete. A theme-colored translucent overlay covers the whole workspace, with centered progress and spinner. The Cinematic toolbar opens all 60 scene looks directly.

Cloud clearance uses a world-space distance field derived from the actual projected terrain triangles, with adjustable strength and soft clearance radius. It attenuates clouds near the model above its base while retaining clouds in surrounding space; it is a footprint approximation rather than a full 3D collision solver. Independent fog remains available. Night uses a dark procedural star panorama, a moon aligned to its light direction, faint environment illumination and the existing night lighting rig.

## Verification — 2026-10-20

- 130 frontend tests in 45 files; production TypeScript/Vite build.
- Full desktop Go tests, including atomic save failure, cancelled pending open, large 5,000-chunk project reopening and native close protection.
- Actual 48-chunk terrain archive roundtrip, comparing chunk geometry and atlas patches.
- `tools/verify-studio-workflow.py`: production UI, full overlay in dark/light themes, deferred HDR requests, transparent PNG markers, cancelled save, native close prompt, successive A/B project opening and free camera edits triggering protection.
- `tools/verify-night-clearance-gpu.py`: star visibility and dark background, moon alignment under horizon correction, local cloud attenuation with unchanged distant foreground; no shader errors.
- Windows icon group (nine images) and application manifest checked in the compiled executable.

Browser/GPU evidence is stored under `tmp/studio-workflow/` and `tmp/studio-atmosphere/`. Native dialogs are covered by backend tests and bridge-driven browser workflows; their OS interaction is not automated by those browser fixtures.

## Delivery

Version `1.59.0-studio-projects-map2d.20261020-dev` installed at `D:\CraftBoxs\craft-box.exe` (67,344,384 bytes). SHA256: `4DE4EBAD2106E0674D6A3F8A2621DD4F3ECB17501D86E716D4AEFA537F83A2CC`. Backup: `D:\CraftBoxs\craft-box-before-projects-map2d-20261020-215837.exe`. Staged and installed file hashes were checked before and after replacement.

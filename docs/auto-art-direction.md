# Auto-Art-Direction and daytime — 2026-10-20

> Historical implementation notes below. The current scene-reference selection and local preference learning are documented in [scene-looks-and-auto-learning.md](scene-looks-and-auto-learning.md); the independent 120-direction lighting search has been replaced.

Available in the header of both Render and Cinematic editors once the world has finished loading. **Auto-Art-Direction** analyses and applies in one action. **Erweitert …** opens preferences for intent, atmosphere, fixed/automatic time and output resolution. The resulting report exposes all 93 current scalar render/atmosphere decisions. Rückgängig restores the complete configuration from before the most recent application, including custom-sky selection; the loaded custom HDR file is retained by the parent editor.

## Model evidence and decision pipeline

1. Sample the actual committed terrain meshes, texture atlas and emitter metadata. Distribute a target budget of approximately 40,000 stratified triangle samples across the complete geometry, and weight by triangle area and sampling stride. Transparent texels are excluded. Work yields between mesh batches and ignores results from closed/replaced views.
2. Measure world extents, aspect/relief, footprint occupancy and symmetry on a 32×32 grid, geometric detail, normal distribution, luminance, saturation, whiteness/darkness, warm colors, vegetation-like colors/cutouts, transparent/blue surfaces, palette entropy and existing emitter count. These are visual proxies, not semantic identification of individual Minecraft biomes or buildings.
3. Score nature, winter, warm landscape, fantasy, architectural and dark-scene directions. User intent can guide vivid, cinematic or neutral output. Lighting evaluates 24 azimuths × 5 elevations against the measured unit normal distribution, rewarding shape contrast and penalising lost dark facets/bright-surface overexposure. Artistic tie-breaking remains deterministic.
4. Generate the entire `RenderSettings` object: sun, HDR environment, exposure, AO, shadow quality, material response, grading, bloom, sharpness, vignette, block lights, film spotlight, output dimensions and every atmosphere field. Each value has an audit entry. Seeds derive from model measurements, never random per click. Neutral/clear intent can deliberately disable effects while still specifying their parameters.
5. Scale cloud size, thickness, extent and fog distances from world dimensions. Cloud banks use the continuous field and start below the object, with their upper boundary near its base, avoiding fixed 130-block puffs on large worlds. Extremely large dimensions are clamped to supported parameter ranges. This does not replace the underlying atmosphere integrator with an offline renderer.
6. Apply atomically to the shared settings, including export aspect and size. Cinematic MP4 resolution follows shared output changes. Preserve camera pose, sequence duration, FPS, keyframes and camera motion; these are sequence authoring properties, not `RenderSettings`. Existing cinematic exposure animation multiplies the newly selected global exposure.

## Time control

The shared Light inspector has an editable HH:MM time, quarter-hour slider and morning/noon/evening/night shortcuts. Changing time updates sun or moon direction, energy, color, HDR selection/rotation, background, environment fill and color temperature together. Manual edits to those light controls mark the rig as manual. Existing profiles without time fields inherit defaults; new profiles roundtrip both values.

This is artistic local solar time with a simple day/night cycle. No latitude, season, date or astronomical precision is claimed. HDR skies are pre-existing panoramas, selected/rotated rather than simulated continuously; cloud/fog time and camera animation remain independent.

## Validation and limits

- 110 frontend tests passed, including complete profile coverage/roundtrips, model-dependent scale and material settings, invalid evidence, repeatability, normalisation of rotated geometry, daytime/night changes, one-action application, undo, stale/cancelled result handling and cinematic export-size propagation.
- TypeScript and Vite production build passed; desktop Go checks cover the embedded integration.
- Real 48-chunk terrain fixture: 38,459 accepted surface samples, 120 lighting candidates, 93 decision rows in each editor. Browser checks exercise intent/fixed time, actual HDR switches, output propagation, palette display and undo. Evidence is under `tmp/auto-direction/`.
- The result is an editable heuristic art direction, not a guarantee of subjective perfection or an image-trained semantic model. It does not iteratively render every candidate or optimise true global illumination. Sampling and very large source meshes can introduce approximation bias. Existing local-light shadow budgets still apply.

Delivered as `1.59.0-studio-auto-direction.20261020-dev` to `D:\CraftBoxs\craft-box.exe`. SHA256 `95E70B2CCA96906E7E15886134CB9C5988E0D5E66C87A344F67D81EF858D96BB`. Backup: `craft-box-before-auto-direction-20261020-132948.exe`. Embedded original-style icon group, nine images and application manifest verified.

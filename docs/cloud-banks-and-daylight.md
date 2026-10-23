# Cloud banks and daylight adaptation — 2026-10-20

## Cloud authoring expansion — requested 2026-10-22

The next revision retains all requested improvements: independent cloud layers,
placeable/transformable cloud volumes, an irregular editable opening around the
map, enhanced forms and light transport, volume-aware depth of field, separate
preview/export quality, and independent cinematic cloud animation. Existing
profiles remain a legacy base layer; new objects must round trip with projects
and contribute to PNG, panorama and deterministic video exports. Verification
must include real GPU images, object placement, occlusion/overlap, camera cuts,
transparent output, persistence, and preview/export quality separation.

Visual references inspected: Dragon Village (BuiltByBit resource 27010),
Skywarden Kingdom (MCModels product 13012), the supplied Netherix Christmas
render, and Head into the Clouds (Planet Minecraft). Their useful visual cues
are layered banks, a clear central opening, lit billows with shadowed valleys,
soft foreground framing, and optional voxel silhouettes. Technical references:
NVIDIA GPU Gems 3 chapter 13 (volume scattering) and Three.js render targets.

### Implemented cloud authoring

Both editors share **Atmosphäre → Wolkenbereiche**. Up to eight independently
named layers/volumes can be enabled, duplicated, removed, placed onto the scene,
and translated, rotated or scaled with viewport handles or numeric fields.
Transforms use the scene bounding-sphere radius as their unit. **Insel mit Wolken
einrahmen** creates an underside bank and two framing volumes as a starting point.

The common map opening has an editable center, elliptical radius, floor, soft
transition and irregular edge. Its lower floor allows clouds around the island
underside. New media share the same depth-clipped ray integration, including
overlapping volumes, colored scattering and sun self-shadowing. Billows, erosion,
soft edges, a backlit rim and approximate multiple scattering are adjustable.

Cloud depth is accumulated separately as a contribution-weighted camera-space
depth and consumed by depth of field. This is a real-time approximation for
translucent media. Preview quality and export quality select independent ray
budgets; Ultra export uses 192 steps at full output resolution.

In Cinematic, position XYZ, size XYZ, rotation XYZ, density, coverage and dissolve
have independent scalar tracks with Bézier, linear and hold interpolation and a
curve/velocity editor. Program time drives sampling during preview and export.
Cloud settings and tracks round trip in native studio projects. The 2D map view
derives cloud-free settings without deleting the 3D project's clouds.

Evidence: `tools/verify-cloud-authoring-gpu.py` checks combined volumes, opening,
repeatability, dissolve, cloud depth/DOF, alpha, depth occlusion and export budget.
`tools/verify-cloud-authoring-editors.py` exercises both editors on recorded
terrain, surface placement, transforms, duplication, keyframes, project saves,
actual PNG/GIF exports, project reopening and optional transparent panoramas.

### Cloud authoring verification and installation — 2026-10-22

- 212 frontend tests across 64 files pass; TypeScript/Vite production build and
  desktop Go tests pass. Final review found no remaining concrete blockers.
- Actual GPU: repeat-frame delta 0, fully dissolved delta 0, opaque-wall occlusion
  delta 0; 13,587 contributing cloud-depth pixels and a visible DOF-depth difference
  of 4.878. Ultra export selects 192 steps at full resolution. Transparent clouds
  retain 14,438 translucent pixels and 34,714 fully clear pixels in the fixture.
- Both real terrain editors pass authoring and PNG exports. A static-camera GIF
  changes by 0.730 mean RGB units through cloud dissolve alone. A transparent
  1024×512 panorama passes. Native projects reopen with cloud curves/opening and
  permit immediate placement in both editors (`verify-cloud-project-reopen.py`).
- Authored clouds at a held animation cut pass the cinematic motion audit:
  cut/temporal delta 0.659, map/temporal delta 0.997, and all eight shutter samples
  remain after the camera cut. Legacy cloud-bank/fog regression deltas remain 0.
- Loaded-object selection, stale hovered transform axes, animated picking time,
  2D map isolation, multi-turn rotation and authored-layer repainting were checked
  during review. Regression tests cover the reproduced failures.

Installed `1.59.0-cloud-authoring.20261022-dev` at
`D:\CraftBoxs\craft-box.exe` (67,855,872 bytes). SHA256
`76C3D49695834EDFE2EAF7ABEF6AB9025ED74981561CB325C28E1BFD44729496`.
Backup: `craft-box-before-cloud-authoring-20261022-094136.exe`.
Final embedded frontend asset names, Wails icon group, nine icon images and
application manifest were verified before installation; installed/source hashes match.

This revision addresses the flat sheet/fog horizon produced by the previous Auto-Art-Direction cloud configuration and the visible artificial glow of light blocks during bright daylight.

## Clouds and fog

### Cloud sampling regression fix — 2026-10-22

The cloud-authoring interval integrator had divided the cloud sample budget by
the combined cloud-plus-fog ray length. A long fog interval could therefore reduce
fine clouds to the twelve-step minimum, causing horizontal slices and frayed,
flat-looking island clouds. Cloud segments now use the union length of cloud
intervals for their own budget. Shared volume integration and fog remain enabled.
Completely empty authored layers are omitted after animation sampling, so they
cannot consume the budget of visible clouds or alter their shadow reach.

GPU reproduction: fog density `1e-8` changed an otherwise identical cloud image
by 1.78755 mean RGBA units before the fix and 0 afterward. Adding a huge empty
authored volume also produces delta 0. Existing overlap, opening, repeatability,
dissolve, DOF, alpha and depth-occlusion checks pass. Actual 720px PNGs of
`cloudIsland`, `cloudGolden` and `cloudSoft` with fog pass without GPU errors;
evidence is in `tmp/cloud-island-look-fix`. The ten focused cloud data tests and
the production TypeScript/Vite build pass.

Installed `1.59.0-cloud-sampling-fix.20261022-dev` at
`D:\CraftBoxs\craft-box.exe`; source/installed SHA256 matches
`58FFFED12712FA0602D8A390598E1AE9DA60716D71B83809CAEE12272CE070DD`.
Backup: `craft-box-before-cloud-sampling-fix-20261022-101809.exe`.
Embedded frontend assets, icon resources and application manifest pass verification.

- Added `cloudModel: banks`, selectable as **Wolkenmeer · räumliche Wolkenbänke**. It joins seeded, warped 3D lobe groups with broader overlapping bodies and varying vertical centres. It retains world-space density, geometry depth occlusion, sun self-shadowing and local volume lighting. This is an editable real-time volume approximation, not an offline scattering solver.
- Auto-Art-Direction, the three cloud-island looks and the Wolkenmeer library entry use the new model. Automatic thickness increases relative to world span/height and the bank extent is reduced so the bank does not become a thin, oversized disk. Old manually selected Classic/Natural profiles remain selectable; reapply Auto-Art-Direction or a cloud-island look to update an existing setup.
- Bank and valley fog now have finite world-relative horizontal bounds and decay both above and below their height. Their region derives from world bounds and fog size independently of cloud settings. Ordinary mist/haze remains intentionally global.
- Cloud/fog ray intervals are bounded before marching. Disabled clouds no longer partition the fog integration interval. The bank model remains visible at shallow viewing angles; the older horizon fade is retained for the older models.
- Sun attenuation uses optical depth integrated over geometrically distributed shadow samples. Ambient fill is cooler and a bounded soft scattering term keeps shadowed cloud volumes readable.

## Block lights

**An Tageslicht anpassen** is on by default in the shared Light inspector and in generated art directions. It smoothly suppresses point-light power and material emission with increasing actual directional-sun irradiance (`sun intensity × positive sine of its elevation`). Suppression is complete at strong daylight; the sun's actual direction is used when linked to an HDR sky. Night settings retain full artificial-light power.

This is a global artistic adaptation, not a per-emitter sunlight-visibility simulation. Disable it for interiors or deliberately luminous daylight scenes. Film spots are unaffected. New saved profiles include the switch; older profiles inherit its enabled default.

## Verification

- Regression tests first reproduced the old Auto selection and daylight power remaining unchanged. The linked-sky low-sun case was also reproduced before its fix.
- 111 frontend tests and the production TypeScript/Vite build passed.
- GPU regressions compare fog far outside the model, disabled-cloud invariance, repeatable volume frames, low-angle rolling silhouettes, perspective/orthographic occlusion and moving seeds/wind.
- Lighting GPU checks compare day/night and an explicit manual daylight override, in addition to wall shadows, alpha-cutout shadows and point/spot scattering.
- Actual terrain is exercised in both editor pipelines with signed bank heights and PNG export. Evidence scripts: `tools/verify-cloud-banks.py`, `tools/verify-lighting-gpu.py`, `tools/verify-bank-editors.py`.

Final results: distant bank-fog delta 0; disabled-cloud fog delta 0; repeat-frame delta 0; daylight block-glow delta 0; manual daylight override delta 3.669. Both actual editor PNG exports passed with no shader/browser errors. Desktop Go tests and embedded icon/manifest verification passed.

Installed `1.59.0-studio-cloud-banks-daylight.20261020-dev` at `D:\CraftBoxs\craft-box.exe`. SHA256 `60C4D743CB68186D62B9F390FDD2B88060BBBB4C4B69416119C3E65845F9B8CC`. Backup `craft-box-before-cloud-daylight-20261020-191621.exe`.

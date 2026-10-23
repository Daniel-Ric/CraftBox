# Render lighting investigation — 2026-10-20

## Requested work and current status

The shared Render and Cinematic pipeline now supports Minecraft block emission, nearby point-light illumination, bounded local shadows, a configurable film spotlight, and shadow-aware local scattering in fog/clouds. This is a real-time approximation; full indirect light transport and exact Cinema 4D/Cycles parity remain outside this implementation.

The `reverse-engineer-anything` skill was installed from the original `morluto/rea` repository, `.agents/skills/reverse-engineer-anything`, into `C:/Users/danie/.codex/skills/reverse-engineer-anything`. Installed frontmatter version: 43. The installer copied SKILL.md and its reference directory. This installs instructions, not a live MCP connection or a native analysis engine.

REA CLI 6.7.0 `doctor --client codex --json` reported unsupported Node 23.5.0 (supported ranges: 22.19+, 24.11+, 26+), no configured native engine, and configuration-read errors. These are diagnostics, not proof of a live connection. No Codex configuration, system Node installation, or native analysis engine was changed. The CLI release also differs from repository-main skill version; use its actual command catalog rather than assuming all skill-described commands exist.

## Observations and sources

- No Maxon/Cinema 4D directory was found in the bounded top-level `C:/Program Files` directory-name check. This is not proof that no installation exists elsewhere.
- Blender 4.3 and Blender 5.0 directories exist under `C:/Program Files/Blender Foundation`.
- Ordinary shell inspection, **not REA Evidence**, of `Blender 5.0/5.0/scripts/addons_core/cycles/shader/node_emission.oso` found an OSL 1.00 shader compiled by oslc 1.14.7.0. Its Color and Strength parameters multiply an emission closure. This establishes that packaged shader's expression only; it does not establish the integrator's execution or reproduce the renderer.
- The Blender 5.0 installation includes Cycles kernel source directories `integrator`, `light`, `closure`, `sample` and `bvh`. These are suitable primary implementation references without pretending binary decompilation recovered them. The REA skill specifically routes ordinary available-source analysis to normal repository tools.
- [Chunky material properties](https://chunky-dev.github.io/docs/user_guides/introduction/material_properties/) explain texture-derived emitter color, emittance and emitted surface area. [Next Event Estimation](https://chunky-dev.github.io/docs/user_guides/introduction/next_event_estimation/) describes spatial emitter cells, adjacent-cell sampling and convergence tradeoffs.
- [Maxon Redshift Physical Light](https://help.maxon.net/c4d/2025/en-us/Subsystems/Default/Content/html/Physical%2BLight.html) documents physical point/spot/area lights and inverse-square falloff. This is documentation evidence, not evidence about undisclosed implementation details.

The shared RenderEngine retains Standard/Physical materials, HDR environment lighting, directional sun shadows, GTAO, bloom and color grading. Optional scene-page `lights` metadata now conveys actual block identity/level. Registry values, conservative vanilla fallbacks and captured custom `minecraft:light_emission` components are respected, including switched-off states. Texture sampling supplies emitter color, not emission identity. Actor `unlit` materials are not used as emitter metadata.

## Implementation approach

1. Establish authoritative emitter data for vanilla states and captured custom block `minecraft:light_emission` values. Respect switched-off block states. Do not infer all luminous blocks from bright texture pixels or actor unlit materials.
2. Carry emitter position, energy/color and shape through the existing scene-page contract, with backwards-compatible validation and tests. Update the existing contract artifact before changing that interface, using the Postman api-engineer workflow already read in this session.
3. Separate visible material emission from actual illumination of surrounding surfaces. Test occlusion and energy falloff on a dark enclosure fixture, including off/on states and transparent cutouts.
4. Select nearby emitters deterministically under an explicit performance budget. Document approximation limits relative to path tracing instead of calling a bounded real-time selection exact Cycles/Redshift GI.
5. Integrate local lights and spot cones into volume scattering using the same scene coordinates and occlusion as surface lighting. A cone overlay or halo is not an adequate replacement.
6. Share controls and rendering implementation between Render and Cinematic. Preserve repeatable timeline timestamps and every export format. Verify saved style roundtrips after settings/contract additions.
7. If exact offline light transport is required, assess using an actual installed Cycles backend versus an explicitly labelled real-time approximation; do not silently claim equivalence.

## Validation and practical limits

- Native tests verify active/off block states, captured custom conditional emission, packed scene metadata, and unknown runtime IDs. The latter exposed an out-of-bounds registry access; a range guard fixes it.
- Frontend tests verify input validation, surface emission attribution, deterministic emitter selection, shadow-map refresh, and complete saved-style roundtrips including film-light settings.
- Real WebGL checks compare block lights on/off, solid-wall shadows, transparent cutout shadows, point-light fog and spotlight fog. No shader errors were reported. `MeshDepthMaterial` is deliberately used as the custom point-light distance material because this Three.js revision stores projected cube depth, verified against its shadow-map implementation and actual rendering.
- Generated Go scene pages were loaded through both production editors. GIF, MP4 and WebM exports with film light/fog were generated and decoded successfully; MP4 was 720×406, 0.25 seconds.
- Performance budgets select 4–24 nearby point lights deterministically; at most two cast local shadows. Additional unshadowed point lights can illuminate through intervening walls. Only shadowed local lights contribute to volumetric scattering. This is not full-scene ray tracing or bounced GI. Existing sun/cloud self-shadowing remains separate.
- Controls live in the shared Light inspector; presets store and transfer them across worlds. Film-light coordinates are relative to world size. Fog must have density for visible shafts.

No native REA decompilation, Cinema 4D execution, or production-lighting parity has been established. Skill installation and native tool readiness remain separate from the implemented real-time lighting.

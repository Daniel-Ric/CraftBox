# Render quality: Hive island analysis and implementation

Research date: 29 October 2026. Reference: the user's 2048 × 1152 CraftBox island render.

## Observations

The island silhouette and readable central castle already make a strong overview. The most distracting areas are the very transparent purple-blue ocean, the repeated seabed pattern, dense one-pixel vegetation and roof details, and pink haze crossing architecture. The right-hand castle could separate more clearly from the vegetation behind it. These are visual judgments, not proof of corrupt map data. The exact submitted camera/project was not supplied, so the validation renders use the same Hive world with a reproducible camera and look.

The scene should retain its Minecraft palette and deliberate lavender style. More saturation, sharpening, bloom or fog alone would amplify the busy details. A full-map image also benefits from broad depth of field; stronger miniature blur would hide the structures the map is meant to show.

## References and tools

| Reference | Relevant strengths | Implication for CraftBox |
| --- | --- | --- |
| [Chunky gallery](https://chunky-dev.github.io/gallery/), especially MAT1CSBuilds' *Japanese Castle* and Peregrine05's *Mountain Lake 2* | In the inspected gallery thumbnails, the castle is separated by light/shadow and the lake has a continuous reflective surface with readable shore transitions. These are different maps and cameras; they are visual references, not performance benchmarks. | Preserve large readable surfaces and lighting hierarchy; do not rely on foreground fog to create all depth. |
| [Chunky](https://chunky-dev.github.io/docs/) | Java-world path tracing; [water controls](https://chunky-dev.github.io/docs/reference/user_interface/chunky/render_controls/water/) distinguish waves, visibility, opacity and color. | Give water its own material and controls. CraftBox remains a raster renderer. |
| [Mineways rendering workflow](https://erich.realtimerendering.com/minecraft/public/mineways/mineways.html) | Geometry export to external renderers including Blender. Its documentation discusses atlas mip bleeding, texture filtering and alpha sorting. | Keep isolated atlas mipmaps, preserve cutout coverage and export separate water material for editing. |
| [uNmINeD](https://unmined.net/) | Reads Java and Bedrock. Current development notes describe isometric shadows, water transparency/depth fading and memory improvements; isometric image/web export is listed as unfinished. | Depth fading is useful even without path tracing. Avoid presenting all map tools as equivalent finished image exporters. |
| [BlueMap](https://bluemap.bluecolored.de/wiki/FAQ.html) and [installation requirements](https://bluemap.bluecolored.de/wiki/getting-started/Installation.html) | Preprocessed 3D tiles and browser navigation; Java worlds required. | A useful reference for scalable navigation, rather than a direct Bedrock final-render replacement. CraftBox already has chunk streaming and GLB export. |

The [Three.js SSAA documentation](https://threejs.org/docs/pages/SSAARenderPass.html) describes camera jitter and accumulated samples. The implementation uses that principle across the existing complete CraftBox render pipeline, rather than inserting an additional beauty-only pass. Accumulation uses a floating-point target before bloom, tone mapping and grading.

[Blender's volume coefficients documentation](https://docs.blender.org/manual/en/latest/render/shader_nodes/shader/volume_coefficients.html) describes absorption per distance. CraftBox uses exponential depth fading as a practical screen-space approximation; this is not a full volume simulation.

## Implemented

1. **Separate water from other transparent blocks.** The mesher emits a `water` part for vanilla still/flowing water, including waterlogged layers. Glass, portals, ice and custom translucent materials keep their existing classification. Internal fluid face culling remains intact.
2. **Depth-aware water.** The opaque depth prepass precedes beauty. Water reconstructs the visible bed position using the inverse projection matrix. Longer paths become less transparent, while shallow shores retain detail. Orthographic and perspective cameras use the appropriate path calculation. Controls expose base opacity, visibility in blocks, roughness, environment reflections, static ripple strength and multiplicative tint. White tint preserves the resource-pack color.
3. **Still PNG sampling: Quick / Clean / Print.** 1, 4 or 16 centered subpixel samples at the requested final resolution. The atmosphere clock is held fixed. HDR samples are averaged before postprocessing. Overlay text is drawn once. Extra samples increase render time rather than framebuffer dimensions. Preview quality remains independent. Cinematic stills without multi-sample motion blur also use this sampling; shutter accumulation retains its existing path.
4. **Existing scene looks updated.** All scene families have explicit water values. Nature looks retain more underwater visibility, cloud/island and winter looks fade earlier, and night/twilight water is rougher. Shared cloud-bank fog is less dense and lower; the lavender scene has lighter shadows and less global magenta tint. The original clear/soft/golden cloud presets receive the same direction of adjustment. Existing output size, sampling choice and framing survive look changes.
5. **Project/style compatibility.** Documents lacking the new settings receive validated defaults. Authored camera, lighting and atmosphere values are preserved on load. New settings round-trip with styles/projects. Older packed `.craftbox` geometry that combines glass and water into one `blend` mesh cannot be safely reclassified without its original world; reimport that world to obtain separate water. Existing look presets themselves work immediately.
6. **GLB material separation.** New water parts export as a named water material with independent roughness. The editor's screen-space absorption and atmosphere are not baked into GLB; external renderers can configure their own water volumes and lighting.
7. **Tighter automatic framing.** Fit projects the world bounds through the current orthographic camera and fits the limiting image axis with an eight percent margin. Wide maps no longer inherit the excessive empty border from a bounding sphere. Portrait output and tall structures remain inside the frame. Saved cameras retain their authored framing.

## Deliberately deferred

True scene reflections, refraction, caustics and global indirect lighting need a substantially different renderer or a carefully budgeted screen-space/ray-traced pipeline. The new reflections control uses the existing HDR environment; it does not reflect nearby castle geometry. Animated water is deferred to avoid unstable stills and extra timeline state. Very high resolution tiled output needs overlap-aware bloom, fog and DOF handling before it can be reliable. A denoiser is useful for path-tracing noise, not a substitute for raster edge sampling.

## Verification

- Unit tests cover fluid surface classification/culling, sample distribution, camera restoration after interruption, independent water controls, preset families and legacy setting defaults.
- The actual Hive `.mcworld` imports all 1,755 chunks with zero missing textures.
- Browser verification compares an emulation of the former material/fog values with the new renderer at identical framing, then exercises the 16-sample transparent path. See the generated artifacts under `tmp/hive-analysis/quality-browser`.
- No new resource-pack images, reference renders or third-party art are shipped in the application.

## Follow-up: editable water and confirmed depth shading

The next iteration adds independently editable shallow/deep colors, color-depth
distance, texture contrast, ripple size, depth-light color/intensity and whole-water
transparency. The cloud scene families fade the seabed earlier. Fog now has a
geometry-protection distance, sharpening has a fine-detail limiter, and Fit exposes
margin and vertical placement. Controls and compatibility are described in
[render-water-controls.md](render-water-controls.md).

An additional assembled-shader regression found that the atlas hook had already
removed the original water insertion marker. Depth and palette logic now use a
preserved marker. Previous no-error browser checks alone did not establish that
the depth code was actually active. The new controlled GPU fixture checks pixel
values for shallow/deep water, lighting, custom colors, full transparency and a
seam between two pieces. Full-world screenshots compare 1/4/16 samples at a fixed
atmosphere time. These checks complement the shader assembly regression.

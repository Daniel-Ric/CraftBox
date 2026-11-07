# Render focus

Depth of field now separates foreground and background blur. The near layer can
spread beyond its silhouette, while sharp foreground surfaces are excluded from
background bokeh. Samples adapt to the local blur radius; a sharp center sample
no longer remains mixed into every blurred pixel. Alpha is filtered in
premultiplied form, then returned as straight RGBA for the existing compositor.
The canvas explicitly uses straight alpha to prevent bright transparent fringes.

## Controls

- **Depth plane**: an artistic sharp depth band, with a transition in blocks.
- **Subject isolation**: a sharp sphere around a picked world point. Other
  surfaces at the same camera depth can still blur outside this sphere.
- **Physical lens**: signed thin-lens circle of confusion, aperture, focal
  length and miniature scene scale. Perspective focal length follows the FOV.
- **Foreground blur / Background blur**: independent strength, including zero.
- **Sharp radius / Focus transition**: the protected region and smooth falloff.
- **Blur limit**: radius at 1080p, scaled by actual output height for previews,
  supersampled captures and exports.

Picking stores an absolute world point, including the chunk origin offset.
Orbiting or flying recomputes its camera-space depth. Manual distance entry or
Unlock focus returns to an unlocked plane. In cinematic subject mode this saved
subject is the fallback; an animated tracking target takes priority.
Focus settings survive look changes and project/style save/load. Older styles
without a focus mode retain their physical lens and unit strength, including
aperture animation. Newly created settings use the depth-plane mode.

## Visual selection

**Pick subject** opens a sharp selection preview and suspends camera orbiting.
The crosshair follows the hovered visible surface. Atlas alpha testing skips
transparent cutout pixels instead of selecting invisible geometry. Hover picking
is throttled, with the final cursor position processed on the next update.

A yellow reticle and wire sphere preview the chosen point and the spatial sharp
radius. Clicking locks the target in cyan; the sphere updates when the radius
changes. The overlay also shows the current camera-to-focus depth. Escape cancels
selection while retaining the previous target. Picking from first-person mode
returns to orbit first. **Show focus guides** controls overlay visibility.

The sphere represents the sharp radius only in Subject isolation mode. Other
modes show the picked focus marker. Targets outside the viewport are reported
without drawing a misleading label at the viewport edge. A locked target's
distance entry is disabled until **Unlock focus** is used.

Guides are DOM/SVG overlays and are excluded from PNG/video exports. Picking
temporarily bypasses DoF only in the interactive preview; exporting with the
picker open retains the authored effect. The foreground gather also excludes
in-focus samples from its near-layer coverage calculation.

Preview gathers follow Fast/Balanced/Fine quality. Exports use the Fine gather.
Near-radius dilation uses two quarter-resolution buffers; no buffers are
allocated per frame. Two half-resolution color layers separate near and far
surfaces before mipmapped filtering. The sample footprint grows with blur radius
and sample spacing, suppressing repeated rail/texel patterns without filtering
the sharp focus layer. Visible background samples fill defocused foreground
edges before premultiplied compositing, so the original sharp silhouette does
not remain under the blur. Uniform foreground regions preserve their alpha.
The effect remains a screen-space approximation: it cannot
recover surfaces hidden behind an opaque foreground object. Transparent surfaces
and clouds still have a representative depth rather than full multilayer optics.

## Research

[Blender camera DOF settings](https://docs.blender.org/manual/en/5.3/render/cameras.html)
provide object focus, distance, aperture and bokeh controls. CraftBox follows the
focus-target behavior and keeps a distinct physical-lens mode.

[Unreal Engine depth of field](https://dev.epicgames.com/documentation/unreal-engine/depth-of-field-in-unreal-engine)
separates near, far and focal regions and blends transitions between them. This
informs the independent foreground/background strength and artistic sharp band.

[AMD FidelityFX Depth of Field](https://gpuopen.com/manuals/fidelityfx_sdk/techniques/depth-of-field/)
explains signed circle of confusion, near/far separation, foreground dilation and
resolution-dependent blur limits. The WebGL implementation uses these principles
with bounded fragment gathers rather than the SDK's compute implementation.

The spatial subject sphere is an explicit artistic option, not a physical lens
simulation. It is useful for isolating a building in a large orthographic map.

## Verification

Unit tests cover signed lens blur, camera-space focus, physical lens limits,
artistic focus at miniature scales, resolution scaling and style migration.
`tmp/dof-check/verify.py` renders a real GPU fixture: near/focused/far checkerboards,
same-plane spatial isolation, near silhouette spread, transparent-edge tint and
2x output scaling. It also loads Hive chunks through the actual RenderEngine,
checks focus after moving the camera, tests the cinematic fallback, and exports
a PNG. Artifacts and measurements are in `tmp/dof-check`.

The GPU fixture additionally checks the physical lens with a perspective camera
and compares exports with and without an active picker. `verify-focus-ui.py`
tests the real app's hover/commit/cancel flow, live radius updates, guide toggle
and selection started from flight mode.

The strong-blur checker fixture also limits residual fine detail and foreground
silhouette jumps. It tests both transparent surroundings and an opaque colored
backdrop, whose opacity must remain intact.

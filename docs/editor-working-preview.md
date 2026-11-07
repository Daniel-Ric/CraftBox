# Editing and export quality

The Render and Cinematic editors start in `Interactive`; `Full preview` remains available in the view toolbar. Both modes retain the original meshes, UVs and texture atlas. Capture 3D also requests the original meshes. Navigation no longer triggers a second geometry cycle, and returning from an export does not replace terrain with proxies.

Interactive mode reduces screen resolution and expensive effects while navigating. Its adaptive pixel ratio has a 0.65 floor; motion is bounded to 1.8 million pixels. At rest it increases to at most 1.5 device pixels per CSS pixel and 3.2 million pixels. It progressively averages up to twelve real, jittered subpixel samples. Camera, settings, atlas and resolution changes invalidate the accumulated image. Animated atmospheres invalidate it each frame to avoid trails. This is raster antialiasing, not a path tracer or a cosmetic grain overlay.

Ambient occlusion and shadow updates return when navigation settles. Shadow sampling stays enabled while navigating, using cached maps; the shader configuration, zero-MSAA editor targets and 1024-pixel sun shadow target stay stable across movement and idle. Newly enabled shadow lights receive an initial map before updates can be frozen. Their editor budgets remain bounded; depth of field and temporal motion blur are reserved for Full preview and final rendering. Geometry uploads use the existing frame queue. Cloud-clearance preparation runs separately from the editor ready state; final rendering waits for it.

Scene page requests still accept legacy `previewStep: 4` for compatibility, but it now returns the same original geometry as `0`. It no longer samples distant height columns. Other values are rejected. Full-detail source geometry and textures are loaded once, progressively; sufficiently large scenes still need adequate RAM/GPU memory. This change does not promise unlimited scenes or eliminate initial source meshing costs.

Final stills, panoramas and video use the existing export-quality pipeline. Interactive accumulated history is disabled for export and reset on return. Export does not reload or downgrade the source meshes.

## Export tasks

File exports share an application-level task view in the console/status strip. A new task opens a narrow centered dialog with its current phase, elapsed time and measured progress where available. Unknown totals use an indeterminate bar. Run in background hides the dialog; the status-strip task button reopens active and recent exports. Completed tasks can be cleared. Cancellation is shown only for operations with a real interruption path.

Model and world jobs carry an operation ID through the desktop bridge. Progress is accepted only for the matching job, and cancellation checks the ID under the native application mutex. Same-family jobs are also reserved in the originating window before the save dialog opens. This prevents another window or export dialog from adopting or cancelling an existing job.

The interaction follows the JetBrains [Progress indicators](https://plugins.jetbrains.com/docs/intellij/progress-bar.html) and [Processes](https://www.jetbrains.com/help/clion/working-with-processes.html) guidance. Visual components, colors and controls remain CraftBox's existing dialog and status-strip components.

## Empty editor

The project empty state uses one vertically aligned action group: primary New project, outlined Open project, and subdued Open world. Each uses the existing Button component and a matching icon, including keyboard focus styling.

Verification artifacts for this change are recorded in `tmp/hostname-prototype/viewport-export-*`. Native regression coverage compares a thin distant bridge in full and legacy-preview pages; the old implementation failed this comparison. Browser fixtures exercise the actual React components and WebGL renderer. These controlled checks do not substitute for a measured FPS comparison on a user's complete captured world.

The HDR movement regression and its red/green GPU checks are documented in `hdr-viewport-movement-fix-20261105.md`. Full preview and export have a separate pipeline cache key from the resting interactive budget.

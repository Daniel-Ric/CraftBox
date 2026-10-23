# Render-Editor

Open Render-Editor from the sidebar, then choose a captured map or open a
Bedrock world archive/folder. Custom blocks and their pack textures are retained;
actors, player models, pets and holograms are excluded from map renders.

Both editors share the Bildgestaltung inspector: Looks, Licht, Farbe, Details
and Ausgabe. Lebendige Insel, Blütengarten, Warmer Film, Winterabend and Klar &
neutral coordinate lighting, HDR environment and color grading as starting
points. Saturation, vibrance, contrast, temperature, tint, highlights, shadows,
vignette and edge-limited sharpening have sliders and direct numeric inputs.
Disable Farbkorrektur aktiv for a before/after comparison. Grading follows ACES
tone mapping in display-referred sRGB and preserves image alpha. These controls
apply to the preview and all PNG/GIF/MP4/WebM exports. The cinematic camera
inspector has Kamera, Sequenz, Pfad and Export tabs; opening Bildgestaltung
replaces that inspector rather than reducing the preview with two sidebars.

The editor offers twelve lighting looks: Tageslicht, Weiches Licht, Tropisch,
Abendsonne, Sonnenuntergang, Blaue Stunde, Mondnacht, Winterlicht, Dramatisch,
Architektur, Studio and Neutral. A look changes lighting and background without
replacing the chosen output size, image format or transparent-background choice.

Four photographed Poly Haven CC0 HDR panoramas are bundled for offline use.
Choose Skybox, rotate it, adjust Horizontkorrektur if needed, and tune
Himmelshelligkeit separately from Himmelslicht. Own `.hdr` equirectangular 2:1
panoramas can be selected locally (64 MiB file limit, 16 megapixel image limit).
The orthographic map uses a perspective sky backplate that follows the complete
model-camera orientation and its zoom. The default horizon correction is zero;
there is no fixed compensation for the isometric camera pitch. Perspective
cinematic previews use the actual lens FOV for their panorama rays.
Panning leaves the infinitely distant sky in place. Sky rotation stays
synchronized with the environment lighting. With Sonne
an Skybox ausrichten enabled, the shadow-casting light follows the brightest
upper-hemisphere direction in the panorama. Disable it to set sun angles manually.

Schnell, Ausgewogen and Fein configure AO sample count, denoising, multisampling,
AO resolution and shadow resolution together. Shadow resolution can also be
overridden separately. AO is suspended during camera movement; alpha cutouts
are respected by its dedicated depth/normal pass. Translucent surfaces do not
occlude the AO pass. Four isolated per-cell mip levels use linear-light,
alpha-weighted averages and four-tap anisotropic sampling. Their separate gutters
prevent distant texture shimmer without mixing adjacent blocks. Minified
cutouts use filtered MSAA coverage. Larger custom textures keep their original
atlas coordinates. Shadows use deterministic tent-filtered PCF with receiver
plane depth correction instead of per-pixel randomized taps. Shadow frusta fit
the world bounds in light space. Preview resolution follows display DPI; exports
retain the explicitly selected pixel dimensions.

PNG export retains the preview framing and supports up to 3840 pixels on the
longer edge. Transparent PNGs retain sky lighting and directional shadows;
the sky backplate and screen-space AO are disabled to preserve alpha.

Asset sources and hashes: `ui/desktop/frontend/public/skies/sources.json`.
Graphics verification: `tools/verify-render-editor.py --pages <recorded-pages.json>
--url <production-preview-url> --output <directory> --skies` exercises all bundled
skies, custom HDR import, all quality modes, 4K PNG and transparent PNG export.

## Cinematic-Editor

Both editors now provide **Szenenlichter** in the light inspector. Add point,
spot or area lights, select them in the list or scene, and place them on a
surface or move them with the translation gizmo. Position/target/size use the
world bounding-sphere radius. Power is luminous flux in lumens; point and spot
lights use inverse-square falloff. Up to 16 authored lights and four shadow
lights are supported. Area lights illuminate one side and currently do not cast
shadows. Lights persist in studio projects and render styles, and changing a
lighting look retains authored lights.

The Details inspector includes depth of field with focus picking, f-stop,
focus distance in blocks and world scale (blocks per metre). Increasing the
scale makes the world behave like a miniature. Perspective focal length follows
the actual lens FOV; orthographic views use the explicit virtual focal length.
Visible transparent terrain participates in the dedicated focus depth pass.

The Render editor's **360°-Panorama** command exports a full-sphere 2:1 PNG at
1024, 2048 or 4096 pixels wide from the current camera position and orientation.
It stitches overscanned HDR cube faces before spherical bloom, output conversion
and grading. Depth of field and vignette are omitted for this spherical projection.
Progress and cancellation are available; the regular viewport is restored afterward.

The separate sidebar workspace offers a perspective program monitor and a 3D
scene view with the camera spline, keyframe positions and target sight lines.
Choose Orbit, Dolly-in or Kranfahrt, or move the scene camera and use Ansicht
übernehmen to capture a view. Coordinates are relative to the world's bounding
sphere; 1 unit is one world radius, centered on its bounds.

The timeline uses independent scalar channels for position XYZ, target XYZ,
FOV, unwrapped roll, exposure, focus distance and aperture. Each channel has its
own keys and mute state. Add, duplicate, delete or move keys without deleting
crossed keys. The curve editor shows value or velocity graphs and supports
weighted Bezier handles, free/aligned/mirrored tangents, automatic tangents,
linear interpolation and holds. Version-1 projects migrate by recovering their
original cubic controls, preserving the existing flight. Zoom, frame snapping,
timecode, frame stepping, looping and playback speed are available. Space
plays/pauses; arrows step frames; Ctrl+Z undoes and Ctrl+Shift+Z redoes. History
retains 50 edits. Focus and aperture edits in the camera panel author independent
keys at the playhead; camera click-focus also authors a focus key.

The Shots inspector enables a named camera library with separate animation and
lens settings. Duplicate cameras, edit their source animation, assign cameras to
shots, split at the playhead, trim ranges and set source-in/speed. The shot lane
selects program time. The monitor distinguishes source-camera editing from the
program's shot sequence; playback and exports evaluate exact cut boundaries.
Gaps hold the preceding shot's final source pose, and source animation holds at
its end. Cameras used by shots cannot be deleted until reassigned. Camera/source
edits and shot edits participate in the same undo history and project document.

The Ziel inspector manages named target objects with their own XYZ animation in
program time. Place a target on a terrain surface, edit its position at the
playhead, or edit its independent curves using the existing graph editor. Each
camera can follow a target with its own offset and keyed influence. Angular
blending avoids collapsing the look direction for opposing targets. Optional
focus binding uses the target's camera-space depth in world blocks; aperture and
DOF enable remain camera settings. Targets appear as selectable editor guides
in scene inspection and are excluded from depth passes, thumbnails and exports.
Shared targets keep program time when a shot remaps the camera's source time.

The Motion inspector provides shutter angle (0–720°), phase (−720–720°), and
2–64 stratified temporal samples per output frame. A 180° shutter integrates half
a frame interval; −90° phase centers that exposure. Integration is clipped at
shot cuts and sequence boundaries. A GPU half-float accumulator combines
premultiplied scene-linear radiance before bloom, output conversion and grading;
exposure changes are included in the sample weights. Optional live preview uses
up to eight samples. PNG, GIF, MP4 and WebM use export quality independently of
that preview switch; dialog thumbnails use up to eight samples. Map annotations
are composited once after integration with central-frame depth/projection.

Projects autosave per world; media-bearing cinematic drafts use IndexedDB with
a small local-storage reference. Audio is embedded in versioned project files,
so saving a project also retains its source media. Bildgestaltung opens shared
lighting and output controls. HDR
reflection intensity, surface roughness and bloom are configurable; translucent
surfaces use an IOR of 1.333 and a clearcoat reflection layer. Cutouts use MSAA
alpha coverage. Shadow normal bias is scaled to the world shadow texel footprint,
and filter radius scales with resolution to keep softness consistent.

MP4 exports H.264 frame by frame using WebCodecs, up to a 3840-pixel long edge,
subject to installed codec support. GIF uses per-frame 256-color palettes,
320–960 pixels and 10–25 fps. GIF and MP4 sample exact sequence time, so slow
rendering does not skip frames. WebM also renders frame by frame using a VP9/VP8
WebCodecs encoder and Mediabunny muxing, with the same resolution/time/shutter
settings and cancellation path. Frames are captured before asynchronous encoding
and released afterwards.
The Audio inspector imports local mono/stereo material, displays source waveforms,
and supports multiple mix tracks, clip move/trim, source-in/out, playback rate,
gain, stereo pan, fades, mute and solo. Native AudioContext scheduling drives the
playhead during audio playback. MP4 includes an AAC mix and WebM an Opus mix when
audible clips exist, subject to codec support. Output uses an integer 48 kHz
sample clock and the same gain/pan/fade model as preview. GIF and PNG are silent.
Markers support names, comments, colors and ranges, frame snapping, direct lane
editing, previous/next navigation and undo. Press M to add a marker.

Native Save dialogs select destinations; cancellation
leaves no output. Source and render settings are locked during exports. Files
are capped below the 256 MiB bridge payload limit. The bundled encoders work
offline; no FFmpeg installation or remote rendering service is required.

`tools/verify-cinematic-editor.py` exercises editing, undo, the 3D path view,
lighting controls, animated GIF decoding, H.264 MP4 playback metadata, desktop
layout and light/dark theme screens with recorded scene pages.
`tools/verify-cinematic-tracks.py` exercises independent axes, direct Bezier
handle dragging, velocity graphs, focus pulls, separate cameras, program cuts,
project saving and an actual short animated GIF using the same real WebGL terrain.
`tools/verify-cinematic-tracking.py` verifies cinematic click focus, target surface
placement, independent target curves, tracking/focus settings and project saving.
`tools/verify-cinematic-audio.py` checks real audio import/playback, saved media
and markers, IndexedDB reload, and independently decoded AAC/Opus timing, gain,
pan and source ranges. `tools/verify-cinematic-light-exports.py` compares actual
lit/unlit motion exports and cancels during a 64-sample frame.
`tools/verify-cinematic-render-audit.py` compares GPU pixels at a camera cut and
checks static sharp/temporal map-overlay parity.
`tools/verify-cinematic-motion.py` compares actual sharp/blurred GIFs, checks
brightness and edge energy, preview and saving. The video-motion verifier uses
direct WebCodecs decoding for pixel checks because HTMLVideo drawing can be black
under headless SwiftShader despite valid encoded frames. Use a fresh strict port
for each browser run: Windows server-helper children can otherwise leave an older
preview instance listening on a reused port.

The desktop theme uses neutral graphite panels and blue selection/action states,
with equivalent light and dark palettes, system UI typography, compact controls
and visible focus. Palette reference: Adobe Spectrum's gray/surface hierarchy
at https://spectrum.adobe.com/foundations/color/grays.

Incremental Windows builds must first run `go run build.go --windows-resources
amd64` to embed `assets/craft-box.ico` and the manifest, including Wails' required
icon group ID 3. Then run the normal `go build -tags gui,production` command.
`go run tools/verify-windows-resources.go <exe>` checks the built artifact.

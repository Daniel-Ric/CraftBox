# Studio expansion

The full requested scope is retained here. Completion requires implementation,
project round trips, relevant automated tests, and visual/runtime verification.
Existing world rendering, resource packs, lighting and exports must be preserved.

## Work packages

- [x] Shared placeable point, spot and area lights: object list, transforms,
  target, color, power, falloff, size, shadows, viewport manipulation,
  persistence and inclusion in still/video exports.
- [x] Render depth of field: click to focus, focus distance, aperture, preview
  and exported-image parity.
- [x] Render 360 panorama: actual equirectangular 2:1 projection, full sphere,
  stable orientation, quality settings, export restoration.
- [x] Independent animation tracks: per-property key times, selection,
  add/delete/move, compatibility with existing sequences, undo/redo.
- [x] Curve editor: value curves, editable tangents, interpolation, timeline
  synchronization and predictable overshoot handling.
- [x] Focus pulls: animated distance/aperture with focus-target selection.
- [x] Multiple shots and cameras: named camera library, shot ranges/cuts,
  ordering, preview and deterministic export.
- [x] Target tracking: selectable scene target, offset and keyed tracking weight.
- [x] Motion blur: shutter angle and temporal samples in deterministic export,
  including cut boundaries.
- [x] Audio and markers: local audio import, waveform, offset/gain/mute,
  synchronized playback, video audio muxing, named timeline markers.

## Architecture and visual direction

Keep the established graphite/blue studio theme and existing typography. Use
compact object lists, inspectors and timeline lanes. Object selection should be
visible in the viewport; editor guides must never enter exported media. Share
scene settings between the editors, while animation and shots belong to the
cinematic document. Old documents must still load with explicit defaults.

Use world-relative coordinates consistently with the existing cinematic camera
(one unit = world bounding sphere radius). Validate imported data rather than
silently allowing NaN, invalid IDs, unbounded lists or unusable transforms.

## Reference behavior

- Maxon Redshift lights: point/spot/area types, power/color, spot distribution,
  area dimensions: https://help.maxon.net/c4d/2025/en-us/Subsystems/Default/Content/html/Redshift_C4D_Light.html
- Adobe animation graphs: property/value and speed graphs, direction handles:
  https://helpx.adobe.com/ca/after-effects/desktop/animate-in-after-effects/speed-between-keyframes/speed.html
- Chunky cameras: focus and equirectangular panoramic projection:
  https://chunky-dev.github.io/docs/reference/user_interface/chunky/render_controls/camera/

## Verification record

2026-10-21, first implementation increment:

- Shared lights implemented in `StudioLightControls`, `StudioLightRig` and both
  editor canvases. Point/spot inverse-square falloff, area dimensions and one-time
  LTC initialization, selection, numeric transforms, surface placement, translation
  gizmo, targets, enabled state, duplicate/delete, shadow limits and project/style
  serialization. Guides are excluded from depth passes and exports/thumbnails.
- Depth of field implemented as an HDR disk gather with signed thin-lens CoC,
  focus picking in camera-space depth, f-stop, scale and orthographic focal length.
  Perspective sensor width follows actual aspect. A separate depth pass includes
  visible transparent surfaces, preserving the existing AO transparency policy.
- Panorama implemented: six 92-degree HDR faces, floating-point premultiplied-alpha
  reconstruction, global spherical bloom, output conversion and grading. Export
  dialog offers 1024/2048/4096 by half-height PNG, progress and cancellation.
  Camera/pass state is restored in finally; planar DOF/vignette are omitted.
- Browser evidence under `tmp/studio-lights`: render/cinematic light object actions
  and click focus passed with recorded 48-chunk terrain, no JS/WebGL errors.
  Actual 1024x512 panorama output and viewport size restoration passed. The initial
  face-local bloom produced strong sky seams; diagnosis found up to 55 RGB levels
  difference across the sun-facing face. The revised global HDR pipeline removed
  those visible discontinuities in the inspected image. Higher resolutions,
  cancellation and motion/video exports still need broader visual/runtime coverage
  before these packages are marked fully verified. A later transparent 1024x512
  panorama passed alpha-range and viewport-restoration checks. Saving the actual
  studio document retained all three authored lights and the picked DOF settings.
  Raw framebuffer alpha is explicitly unpremultiplied before HDR reconstruction
  and re-premultiplied only for final canvas output.
- `animation-tracks.ts` now supplies independent scalar channels, weighted Bezier
  time/value sampling, endpoint holds, cuts and collision-safe key movement. Four
  tests pass. This is a foundation, not yet connected to the cinematic UI/document.
- Latest full frontend suite: **152 tests in 50 files passed** (2026-10-21).
  Production TypeScript/Vite build passed after the HDR redesign and alpha fix;
  the final post-format build is recorded in the turn's tool output. Existing
  bundle-size advisories and jsdom canvas notices remain unrelated to this work.

## Next integration work

2026-10-21, animation/camera integration increment:

- Independent channels connected to the timeline, version-2 serialization and
  history. Exact migration of legacy cubic paths; weighted time/value curves,
  velocity view, direct handle/key dragging and numeric tangent controls. Sampled
  key insertion uses De Casteljau subdivision (out-of-range physical keys are
  bounded); extreme aligned handles shorten while retaining their slope.
- Focus/aperture keys drive the DOF pass for preview and deterministic exports.
  Camera click-focus writes a key at the playhead. Numeric focus pulls passed the
  real browser check; the new cinematic click-focus control still needs its own
  browser regression coverage, alongside broader export coverage.
- Named cameras own independent source animation/lens settings. Shots provide
  ordered non-overlapping ranges, exact cuts, source-in/rate, trim/split/delete,
  program/source monitor modes and a clickable shot lane. Gap/end holds are
  explicit. Root fallback is static so a short program can reference longer source
  cameras without breaking reload. In-use cameras cannot be removed.
- Reviewer findings fixed: stale sibling coordinates, track-aware duration limits,
  multi-turn roll reload, physical overshoot insertion, extreme linked handles,
  and source/program duration mismatch. Dedicated regression tests pass.
- `tools/verify-cinematic-tracks.py` passed with recorded terrain: independent
  axes, channel insertion, actual tangent drag, velocity graph, numeric focus
  pull, separate camera edits, exact-boundary program selection, project save and
  an actual six-frame GIF with differing camera images. Artifacts are under
  `tmp/cinematic-tracks`. No JavaScript/WebGL errors occurred.
- Latest complete frontend suite: **175 tests in 54 files passed**. TypeScript /
  Vite production build passed. An earlier suite run concurrent with software
  WebGL hit two five-second UI timeouts; the isolated full rerun passed unchanged.

Audio acceptance requires synchronized playback and actual muxed video audio;
the verification record below covers both.

2026-10-22, tracking and shutter integration increment:

- Named authored target objects with independent XYZ keys, program-time sampling,
  curve-editor reuse, surface picking, selection guides, camera-specific offsets,
  angular influence blending and keyed tracking strength. Focus binding converts
  normalized scene coordinates to camera-space depth in blocks. Parser validates
  IDs/references/limits; deleting a target removes its constraints across cameras.
  Existing terrain pages exclude entities; actor replay is not implied.
- Browser `verify-cinematic-tracking.py` passed cinematic click-focus key creation,
  target placement, target curves, tracking/focus binding and saved studio data.
  Artifacts: `tmp/cinematic-tracking`. Model/UI tests also cover independent target
  motion, source/program time mapping, opposing/coincident aim directions, undo,
  invalid imports and generic influence-key insertion.
- Shutter angle/phase/sample settings and GPU HDR temporal integration implemented.
  Deterministic sample windows stop at cuts, brightness is normalized, premultiplied
  alpha is preserved, and bloom/output/grade run after integration. An initial
  cloned render-target texture caused black output; actual source binding fixed it.
  The strengthened real GIF comparison passed: brightness retained, edge energy
  approximately 111 sharp versus 64 blurred. Artifacts: `tmp/cinematic-motion`.
- WebM migrated from real-time MediaRecorder to deterministic WebCodecs VP9/VP8
  plus Mediabunny. Initialize the encoder before rendering and pass owned
  VideoFrame snapshots. Unit tests verify timestamp/order, frame cleanup and
  cancellation. PNG/thumbnail shutter rendering is independent of preview toggle.
  Map annotations run after temporal integration using central-frame depth.
- Actual WebM output passed 720x406 / 0.25-second metadata and direct decoded pixel
  brightness checks with preview blur disabled. Headless HTMLVideo.drawImage
  reported black despite valid pixels; an independent WebCodecs sample decoder
  confirmed three valid images and the verifier now checks pixels directly.
  Transparent cinematic PNG verification passed with intermediate alpha coverage
  and viewport restoration, no JavaScript/WebGL errors. Artifacts are under
  `tmp/cinematic-video-motion`. Extended cut/image,
  4K, high-sample cancellation and map-overlay parity checks were subsequently
  completed in the final export audit below.
- Latest complete frontend suite: **187 tests in 57 files passed**; production
  TypeScript/Vite build passed after formatting and the WebM migration.
- Reference behavior: Maxon target expression
  https://help.maxon.net/c4d/en-us/Content/html/TTARGETEXPRESSION.html ; Adobe focus
  linkage https://helpx.adobe.com/be_en/after-effects/desktop/work-with-layers/camera-layer/cameras-lights-points-interest.html ;
  Adobe shutter angle/phase/samples
  https://helpx.adobe.com/after-effects/desktop/animate-in-after-effects/assorted-animation-tools/assorted-animation-tools.html .

2026-10-22, audio and marker integration increment:

- Local audio import with cached mono/stereo PCM, source min/max waveforms,
  reusable material library, multiple mix tracks, clip move/trim, source ranges,
  rate, gain, pan, equal-power fades, mute/solo and a pre-clamp peak meter.
  Native AudioContext scheduling drives the playback clock and schedules loops
  ahead. Video export mixes integer 48 kHz samples and muxes AAC into MP4 or
  Opus into WebM using the shared deterministic Mediabunny encoder adapter.
- Named/color-coded point and range markers, comments, frame snapping,
  keyboard/lane editing, adjacent-marker navigation and project undo/redo.
  Audio is embedded in saved documents. Media drafts use IndexedDB rather than
  local-storage payloads; content signatures avoid rescanning base64 every frame.
- `tools/verify-cinematic-audio.py` passed real import/waveform controls, native
  playback source offset/rate/timing, a saved document containing media/marker,
  full-page reload from IndexedDB, and actual MP4/WebM exports. Independent
  decoders checked 720x406 images, nonblack pixels, silent leading offset,
  approximately 0.1004 RMS left signal, silent right channel, and source-in plus
  the expected 440-to-880 Hz transition. Codecs: AAC/Opus. Artifacts:
  `tmp/cinematic-audio` including movies and decoded metrics.
- Review identified retained buffers after duplicate import and import completion
  after unmount. Both fixed, with UI regression tests verified failing without
  the fixes and passing with them. Existing material-array identity is preserved.
- Full frontend suite: **197 tests in 59 files passed**. Production TypeScript /
  Vite build passed after correcting two test-only query option types.
- References: Adobe waveform editing
  https://helpx.adobe.com/ca/premiere/desktop/add-audio-effects/adjust-volume-and-levels/live-waveform-editing.html ;
  audio mixing concepts
  https://helpx.adobe.com/ie/premiere/desktop/add-audio-effects/basic-audio-editing/audio-editing-concepts.html ;
  marker behavior
  https://helpx.adobe.com/premiere/desktop/add-text-images/insert-images-and-graphics/create-markers-in-the-effect-controls-panel.html .

2026-10-22, extended export verification:

- Actual 4096x2048 panorama exported and visually inspected. A separate 4096-wide
  cancellation saved no file and restored viewport dimensions; subsequent export
  and studio saving still worked. Artifacts: `tmp/panorama-4k`.
- Real PNG comparisons with studio lights on/off and DOF on/off confirmed both
  contributions (mean RGB differences approximately 2.08 and 0.55 respectively).
  Transparent DOF PNG retained intermediate alpha coverage; a subsequent
  transparent 1024x512 panorama passed. Artifacts: `tmp/render-still-parity`.
- Actual motion-blurred cinematic GIFs with a studio light enabled/disabled
  differed by approximately 7.43 mean RGB levels. Video rendering now yields
  between temporal samples and propagates AbortSignal through both canvas handles.
  A real 64-sample video was cancelled after eight samples of its first frame,
  restored the editor and saved no movie. Iterator-finally state restoration has
  dedicated tests and a review found no further important issue. Artifacts:
  `tmp/cinematic-light-exports`.
- Latest full frontend suite: **199 tests in 60 files passed**. TypeScript/Vite
  production build passed after the cancellation refactor.
- Post-refactor WebM decoding and transparent cinematic PNG verification passed
  again (`tmp/cinematic-video-motion`), including viewport restoration.
- `tools/verify-cinematic-render-audit.py` compared actual GPU-rendered images
  at an exact camera cut and through temporal map compositing. Eight samples at
  a 720-degree shutter all stayed on the new shot's side of the cut. The temporal
  output differed from the correct sharp camera by 0.67 mean RGB levels versus
  32.23 between cameras. Map overlays changed the image by 11.80 levels; static
  sharp/temporal map parity differed by only 1.00. Artifacts, inspected images and
  metrics: `tmp/cinematic-render-audit`. No JavaScript errors occurred.

## Final requirement audit — 2026-10-22

All ten requested work packages retain their original scope. Primary reference
behavior is linked above; the implementation is adapted to the existing terrain
renderer and studio document rather than requiring an external DCC application.

| Requirement | Implementation and automated evidence | Runtime/export evidence | Result |
| --- | --- | --- | --- |
| Placeable lights in both editors | `StudioLightControls`, shared `StudioLightRig`, parser and controls tests; transforms, targets, color/power, dimensions, shadows, persistence | Both-editor object operations; real still and motion GIF on/off comparisons; saved studio lights | Complete |
| Render click-focus DOF | `DepthOfFieldControls`, thin-lens and camera-depth tests, separate transparency-aware depth pass | Terrain click-focus, saved settings, DOF on/off PNG comparison, transparent alpha coverage | Complete |
| Render 360 panorama | Cube/full-sphere projection tests, HDR stitching and global postprocessing | Inspected 4096x2048 PNG; transparent panorama; cancellation with no save and restored viewport | Complete |
| Independent cinematic tracks | `cinematic-tracks` migration, channel isolation, insertion, timing, collisions and undo tests | Independent axis edits, per-channel timeline and saved project | Complete |
| Curve editor | `animation-curves` sampler/handle/velocity and extreme-handle tests | Actual tangent dragging, value/velocity views and saved curves | Complete |
| Focus pulls | Independent focus/aperture sampling tests; cinematic lens controls and target focus binding | Numeric animation, cinematic click-focus keys, saved focus binding and rendered DOF path | Complete |
| Multiple cameras and shots | `cinematic-shots` source/rate, exact cuts, gap holds, parser/round-trip tests | Separate camera edits, shot GIF, exact-cut image isolation at wide shutter | Complete |
| Target tracking | `cinematic-tracking` animated targets, program/source clocks, angular blending, references and influence keys | Target surface placement, target curves, focus/aim binding and saved project | Complete |
| Motion blur | Shutter/sample/cut tests; premultiplied HDR integration; abort/iterator cleanup tests | Sharp/blur GIF edge comparison; decoded WebM; transparent PNG; actual mid-frame cancellation; map-overlay pixel parity | Complete |
| Audio and markers | Mix/trim/rate/pan/fade/mute/solo and marker tests; embedded media, IndexedDB drafts, import lifecycle regression tests | Native playback scheduling; media/marker save and reload; independently decoded MP4/AAC and WebM/Opus with offset, source-in, gain, pan and tone checks | Complete |

Final gates: 199 tests / 60 files passed, production TypeScript/Vite build passed,
and the lifecycle/cancellation review reported no remaining important findings.
Existing bundle-size advisories and jsdom canvas notices are unchanged.

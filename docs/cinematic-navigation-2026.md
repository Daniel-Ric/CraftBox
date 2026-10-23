# Cinematic navigation revision — 2026-10-20

The Cinematic header now exposes Light, Atmosphere and Color directly, alongside the complete Image Design inspector and the Camera/Sequence workspace. These use the existing shared RenderEditor settings, including all cloud/fog libraries, HDR environments, grading, details, output and saved render styles.

**Freie Kamera** enters focus-scoped navigation. Click the viewport, move with WASD, move vertically with Q/E, and right-drag to look around. Shift accelerates, Alt slows movement; the speed selector provides 0.1×–4×. Escape returns to orbit. Blur, mode changes and disposal clear movement state. Preview and exports disable navigation. Position changes move the target by the same offset, while mouse look changes orientation around the camera position.

**Neuer Keyframe aus Ansicht** captures the current camera at the playhead. **Keyframe aktualisieren** applies the view to the selected keyframe and retains its time. Both use sequence undo/redo. View capture preserves lens, relative exposure and normalized roll. A reviewer found that an interpolated roll greater than 180° could make saved projects unreadable; the regression reproduces the failure and verifies that 185° saves/reopens as −175°.

Cloud modeling is selectable in both editors: Natural applies volume-envelope warping and multiple noise scales to the puffy silhouette; Bodies retains the preceding compact lobe model; Classic restores the earlier procedural field. The natural model's structure moves with wind instead of sliding independently. Shape settings survive render style saves and imports.

## Validation

- Full frontend suite: **98 tests in 32 files**, `npm.cmd test -- --maxWorkers=1`.
- Production TypeScript/Vite and native builds passed.
- `tools/verify-cinematic-navigation.py`: actual 48-chunk terrain, atmosphere/color/light access, forward movement with translated target, mouse look without camera-position changes, selected-keyframe capture, no continued movement after blur, Escape, compact layouts.
- `tools/verify-cinematic-editor.py --atmosphere`: timeline editing and real GIF, MP4 (720×406, 0.25s), WebM (637×358) decoding with no browser/shader errors.
- `tools/verify-atmosphere-views.py`: seven actual GPU views, changed seed, all three cloud models produce different outputs. Natural horizon image visually inspected.
- Independent follow-up review verified the normalized-roll correction; no important remaining issue in that correction.

Evidence: `tmp/cinematic-navigation/`, `tmp/atmosphere-v2/views/`.

## Installed build

- Executable: `D:/CraftBoxs/craft-box.exe`
- Version: `1.59.0-studio-camera-navigation.20261020-dev`
- Backup: `D:/CraftBoxs/craft-box-before-navigation-20261020-113638.exe`
- Build, staged and installed SHA-256: `6E15741D7C4F3BE95A59091B3DE7B812A86A24ECF52D5C072596F1F6E3AEC7EB`
- Original app icon retained; icon group 3, nine images and manifest verified.

Restart CraftBox to load this revision. No running user application was terminated.

The later REA/Cinema4D-style lighting request remains open and is tracked separately in [render-lighting-research.md](render-lighting-research.md). The REA skill is installed, but native analysis readiness and the requested emitter/GI/cone implementation are not part of this delivered camera revision.

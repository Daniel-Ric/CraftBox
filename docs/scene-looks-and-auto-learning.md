# Scene library and reference-based Auto — 2026-10-20

The shared Render/Cinematic inspector now offers 60 complete scene looks: 20 existing looks and 40 additions. The library groups them into cloud worlds, nature/coast, fantasy/history, winter/mountains, desert/savanna, city/night, interiors/studio and film/neutral. Search matches descriptions and tags as well as names. All looks are also available in the new-project dialog.

Applying a look starts from render defaults, then applies its complete reference configuration. This avoids inherited fog, cloud or studio-light settings from a previously selected look. Output size, aspect, transparency and pitch lock remain unchanged.

## Auto selection

This supersedes the former independent 120-direction lighting search described in earlier versions of `auto-art-direction.md`.

Auto still measures the actual mesh, texture atlas and emitter metadata locally. It ranks the scene library using authored feature exemplars, model form, vegetation/water/bright/warm/dark surface proportions, color diversity, user intent and optional time/atmosphere constraints. These are visual proxies, not semantic biome recognition.

The winning scene supplies the lighting and grading. Auto preserves its sun direction, energy and color treatment; bounded adjustments cover atmosphere dimensions, deterministic distribution seeds, bright-material protection, dark-material exposure and emitter budget. Explicit time overrides intentionally replace the reference's solar rig. The report names the selected scene and explains all resulting settings.

This is reference-based calibration and nearest-example preference learning, not neural-network training on rendered images. It neither evaluates rendered candidate images nor guarantees subjective perfection.

## Personal feedback

Select a scene look, then choose **Diesen Look für Auto merken**. CraftBox stores the look ID and eight normalized model features in localStorage (`craftbox-direction-examples`, schema version 1), capped at 64 examples. No model files or images leave the computer. Similar feature profiles increase that look's ranking. Explicit atmosphere constraints still take precedence.

Remembering another look for essentially identical features replaces the earlier preference, including when its look ID differs. Invalid stored data is ignored; storage failures produce a visible message. The memory belongs to the local application profile and persists across editor changes/restarts.

## Verification

- 117 frontend tests pass, including all 40 new profiles' schema roundtrips, deterministic reference selection, preservation of reference lighting/grading, model-scale adaptation, local learning and corrected preferences.
- Production TypeScript/Vite build and desktop integration tests pass.
- `tools/verify-scene-learning.py` exercises the production UI with the real 48-chunk terrain fixture, including both editors, all 60 library entries and remembered/corrected look selection.
- `tools/verify-scene-previews.py` captured and visually checked four representative new looks (peach clouds, clear winter, lagoon and warm interior), with no shader/browser errors. Evidence is under `tmp/scene-learning/`; these are samples, not an exhaustive visual rating of every look.

The profiles configure existing rendering features. They do not add weather simulation, rain geometry or new emissive blocks. The cloud looks reuse the existing continuous cloud-bank renderer.

## Delivery

Version `1.59.0-studio-scene-library.20261020-dev` installed at `D:\CraftBoxs\craft-box.exe`. SHA256 `86F9A97C279E1F091D6502B8DEB85AAC09296330644E0502D2BEF96C317BFDA8`. Previous binary retained as `craft-box-before-scene-library-20261020-200154.exe`. The existing icon group (nine images) and application manifest were verified in the Windows executable.

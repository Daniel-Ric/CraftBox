# Editor navigation performance

Both editors use RenderEngine. Previously its motion budget disabled GTAO only;
the full device pixel ratio, high cloud ray-march budget, high depth-of-field
sample count and up to eight cinematic shutter passes continued during movement.
Timeline playback was not marked as movement at all.

The shared viewport now uses a transient movement budget:

- Orbit/pan/zoom, free flight, timeline scrubbing and playback use reduced preview
  pixel density. Slow frame sequences adapt it further with hysteresis, without
  reallocating buffers every frame. Full density returns after the existing
  180ms settling interval. Project/output settings are never changed.
- Clouds retain their appearance with fewer march/shadow samples and lower
  preview resolution while moving. Depth of field retains its focus settings
  with fewer preview samples. Preview shutter accumulation uses two samples
  while moving, then returns to the configured preview budget when paused.
- Map views retain full resolution. Hidden canvases do not perform these updates.
- Still, panorama and video exports force their own output size/density and full
  export settings. The movement budget cannot override export cloud quality,
  depth-of-field quality or temporal/spatial samples. Geometry is unchanged.

## Evidence

`tools/verify-navigation-performance.mjs` exercises actual RenderEngine instances
for both editors in headless Edge using stored Hive geometry. It verifies exact
PNG equality before/after navigation and complete restoration of preview budgets.
The fixture is in `ui/desktop/frontend/perf/navigation-fixture.ts`, outside the
production entry graph. Results: `tools/light-audit/navigation-performance.json`.

At 480x300 logical pixels / DPR2, moving preview work dropped from 576,000 to
104,040 pixels (81.94%), DoF samples from 96/64 to 32/16, and cloud march steps
from 112 to 40. Both export comparisons were pixel-identical. These are workload
checks, not hardware FPS claims: the test uses SwiftShader and records transition
costs, so reported frame timings must not be treated as a steady-state benchmark.

Regression tests cover motion/idle restoration, adaptation on slow GPUs, cloud
export isolation, full shutter sample counts and video export started during
navigation. TypeScript and the full frontend suite are checked before packaging.

Verification: 102 frontend files / 345 tests passed in the complete run, followed
by targeted runs covering the added video-export guards and 60Hz recovery (26
tests in affected modules). TypeScript, production Vite build and desktop Go
tests pass. Independent static review found no concrete regression.

Packaged as `1.63.1-smooth-preview-dev`, `builds/Craft-Box.exe` (85,549,568 bytes),
SHA256 `8F4DBF84C544D2A13E5A1F51083A8C9C5991AA1E23FA70E982EE36D2B4DD18E2`.
Windows icon resources and application manifest were verified.

The final GPU repeat ran with HMR disabled and an explicitly observed page job;
it rejects page replacement, lost WebGL contexts and uniform/empty exports. Both
editors passed, with 153/175 distinct sampled output colors and pixel-identical
exports before/after navigation. PNG evidence is stored alongside the JSON report.

Deployment: direct replacement at `D:\bedrock-tools\Craft-Box.exe` was denied,
including after the running CraftBox processes exited. The installed file still
has SHA256 `5C561450F6570D83A6D1DFE043885F2D7D6F3AD23803539702CB99826226EBF6`.
`tools/Install-CraftBox.ps1` is updated for the new verified build and must be run
locally to complete installation; it was not executed as a bypass by the agent.

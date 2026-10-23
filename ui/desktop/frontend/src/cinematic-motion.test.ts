import { expect, it } from "vitest";
import { createSequence } from "./cinematic";
import { enableCameras, splitShot } from "./cinematic-shots";
import { temporalSamples } from "./cinematic-motion";

it("uses shutter angle/phase and stratified subframes measured at the export frame rate", () => {
  const s = createSequence();
  s.motionBlur = {
    enabled: true,
    angle: 180,
    phase: -90,
    samples: 8,
    preview: false,
  };
  const samples = temporalSamples(s, 2, 24);
  expect(samples).toHaveLength(8);
  expect(samples.reduce((sum, s) => sum + s.weight, 0)).toBeCloseTo(1);
  expect(samples[0].time).toBeCloseTo(2 - 1 / 96 + 1 / 768);
  expect(samples.at(-1)!.time).toBeCloseTo(2 + 1 / 96 - 1 / 768);
  expect(temporalSamples(s, 2, 12)[0].time).toBeLessThan(samples[0].time);
});
it("clips shutter integration at camera cuts, including the frame exactly on the cut", () => {
  let s = enableCameras(createSequence());
  s = splitShot(s, s.shots![0].id, 6);
  s.motionBlur = {
    enabled: true,
    angle: 720,
    phase: -360,
    samples: 16,
    preview: false,
  };
  expect(temporalSamples(s, 6, 24).every((s) => s.time >= 6)).toBe(true);
  expect(temporalSamples(s, 6 - 1 / 24, 24).every((s) => s.time < 6)).toBe(
    true,
  );
  expect(temporalSamples(s, 0, 24).every((s) => s.time >= 0)).toBe(true);
  expect(temporalSamples(s, 12, 24).every((s) => s.time <= 12)).toBe(true);
});
it("does a single exact render when disabled or shutter angle is zero", () => {
  const s = createSequence();
  expect(temporalSamples(s, 3, 24)).toEqual([{ time: 3, weight: 1 }]);
  s.motionBlur = {
    enabled: true,
    angle: 0,
    phase: 0,
    samples: 8,
    preview: false,
  };
  expect(temporalSamples(s, 3, 24)).toEqual([{ time: 3, weight: 1 }]);
});

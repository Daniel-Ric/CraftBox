import { expect, it } from "vitest";
import {
  analyseModel,
  directModel,
  type ModelAnalysis,
} from "../src/auto-direction";
import {
  planCloudComposition,
  estimateCloudOcclusion,
} from "../src/cloud-direction";
import { renderDefaults, renderPresets } from "../src/render-settings";
it("detects a near cloud on the same perspective sightline as a distant landmark", () => {
  const view = {
    position: [0, 0, 0],
    target: [0, 0, 1],
    projection: "perspective" as const,
  };
  expect(
    estimateCloudOcclusion([100, 0, 1000], [10, 0, 100], [5, 10, 5], view),
  ).toBeGreaterThan(0.5);
  expect(
    estimateCloudOcclusion([100, 0, 1000], [10, 0, 100], [5, 10, 5], {
      ...view,
      projection: "orthographic",
    }),
  ).toBe(0);
  expect(
    estimateCloudOcclusion([10, 0, 100], [100, 0, 1000], [5, 10, 5], view),
  ).toBe(0);
});
function island() {
  const samples = [];
  for (let z = -12; z <= 12; z++)
    for (let x = -12; x <= 12; x++) {
      if (x * x + z * z > 144) continue;
      samples.push({
        position: [
          x * 16,
          35 + Math.sin(x * 0.3) * 5 + Math.cos(z * 0.2) * 4,
          z * 16,
        ],
        normal: [0, 1, 0],
        color: [0.2, 0.55, 0.24],
        area: 256,
        mode: "solid",
      });
    }
  samples.push({
    position: [0, 130, 0],
    normal: [0, 1, 0],
    color: [0.8, 0.8, 0.7],
    area: 256,
    mode: "solid",
  });
  return analyseModel(samples, {
    lights: 6,
    triangles: 100000,
    bounds: [-220, -160, -220, 220, 150, 220],
  });
}
it("measures the upper visible terrain independently of buried depth", () => {
  const a = island();
  expect(a.surface!.ground).toBeGreaterThan(20);
  expect(a.surface!.ground).toBeLessThan(50);
  expect(a.surface!.coverage).toBeLessThan(0.85);
});
it("keeps a curated cloud body instead of replacing it with a 48 block sheet", () => {
  const a = island();
  const d = directModel(a, { intent: "auto", atmosphere: "clouds" });
  expect(d.settings.atmosphere.cloudThickness).toBeGreaterThanOrEqual(90);
  expect(d.cloudComposition!.candidates).toBeGreaterThan(8);
  expect(d.settings.atmosphere.cloudLayers.length).toBeGreaterThan(0);
});
it("frames the terrain while preserving color and deterministic placements", () => {
  const a = island();
  const reference = {
    ...renderDefaults.atmosphere,
    ...renderPresets.cloudGolden.atmosphere,
    cloudColor: "#abcd12",
  };
  const p = planCloudComposition(
    a,
    reference,
    { position: [800, 500, 800], target: [0, 40, 0] },
    "cloudGolden",
  );
  expect(p.atmosphere.cloudColor).toBe("#abcd12");
  expect(p.atmosphere.cloudSeed).toBe(reference.cloudSeed);
  expect(p.atmosphere.cloudReference).toBe("world");
  expect(p.atmosphere.cloudHeight).toBeGreaterThan(-100);
  expect(p.atmosphere.cloudOpening.strength).toBeGreaterThan(0.8);
  expect(p.atmosphere.cloudLayers.length).toBeLessThanOrEqual(3);
  expect(p).toEqual(
    planCloudComposition(
      a,
      reference,
      { position: [800, 500, 800], target: [0, 40, 0] },
      "cloudGolden",
    ),
  );
  expect(p.summary.score).toBeLessThanOrEqual(p.summary.baselineScore);
});
it("responds to the camera without changing geometry evidence or the camera", () => {
  const a = island(),
    before = structuredClone(a);
  const reference = {
    ...renderDefaults.atmosphere,
    ...renderPresets.cloudGolden.atmosphere,
  };
  const front = planCloudComposition(
    a,
    reference,
    { position: [800, 400, 0], target: [0, 40, 0] },
    "cloudGolden",
  );
  const back = planCloudComposition(
    a,
    reference,
    { position: [-800, 400, 0], target: [0, 40, 0] },
    "cloudGolden",
  );
  expect(front.atmosphere.cloudLayers).not.toEqual(back.atmosphere.cloudLayers);
  expect(a).toEqual(before);
});
it("respects clear atmosphere and falls back safely without terrain evidence", () => {
  const a = island();
  expect(
    directModel(a, { intent: "auto", atmosphere: "clear" }).settings.atmosphere
      .clouds,
  ).toBe(false);
  const legacy = { ...a, surface: undefined } as ModelAnalysis;
  const reference = {
    ...renderDefaults.atmosphere,
    ...renderPresets.cloudGolden.atmosphere,
  };
  const p = planCloudComposition(legacy, reference, undefined, "cloudGolden");
  expect(p.summary.mode).toBe("reference");
  expect(p.atmosphere.cloudThickness).toBe(reference.cloudThickness);
  expect(p.atmosphere.cloudLayers).toEqual([]);
});

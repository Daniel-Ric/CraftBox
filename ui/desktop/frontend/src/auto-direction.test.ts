import { expect, it } from "vitest";
import {
  analyseModel,
  directModel,
  type SurfaceSample,
} from "./auto-direction";
import { renderDefaults, renderPresets } from "./render-settings";
import { decodeRenderStyle, encodeRenderStyle } from "./render-style-library";
import { applyTimeOfDay } from "./render-daytime";

const model = (scale = 1, winter = false): SurfaceSample[] =>
  Array.from({ length: 512 }, (_, i) => ({
    position: [
      ((i % 16) - 8) * scale,
      (i % 13) * scale,
      (Math.floor(i / 16) - 16) * scale,
    ],
    normal: i % 3 ? [0, 1, 0] : [1, 0, 0],
    color: winter ? [0.91, 0.94, 0.97] : [0.16, 0.55, 0.22],
    area: scale * scale,
    mode: winter ? "solid" : "mask",
  }));

it("produces deterministic complete portable settings from measured model features", () => {
  const a = analyseModel(model(), { lights: 4, triangles: 2048 });
  const result = directModel(a, { intent: "auto" });
  expect(result).toEqual(directModel(a, { intent: "auto" }));
  expect(Object.keys(result.settings).sort()).toEqual(
    Object.keys(renderDefaults).sort(),
  );
  expect(Object.keys(result.decisions).sort()).toEqual(
    Object.keys(renderDefaults).sort(),
  );
  expect(Object.keys(result.settings.atmosphere).sort()).toEqual(
    Object.keys(renderDefaults.atmosphere).sort(),
  );
  expect(
    decodeRenderStyle(encodeRenderStyle("Auto", result.settings)).settings,
  ).toEqual(result.settings);
  expect(a.vegetation).toBeGreaterThan(0.8);
  expect(a.palette.length).toBeGreaterThan(0);
  expect(a.palette[0]).toMatch(/^#[0-9a-f]{6}$/);
  expect(a.confidence).toBeGreaterThan(0);
  expect(result.settings.quality).toBe("fine");
});

it("adapts atmosphere scale and material/color decisions rather than choosing a fixed preset", () => {
  const green = directModel(
    analyseModel(model(), { lights: 0, triangles: 2048 }),
    { intent: "vivid" },
  );
  const snow = directModel(
    analyseModel(model(12, true), { lights: 40, triangles: 50000 }),
    { intent: "cinematic", hour: 18.5, atmosphere: "clouds" },
  );
  expect(snow.settings.atmosphere.cloudSize).toBeGreaterThan(
    green.settings.atmosphere.cloudSize,
  );
  expect(snow.settings.highlights).toBeLessThan(green.settings.highlights);
  expect(snow.settings.timeOfDay).toBe(18.5);
  expect(snow.settings.atmosphere.cloudModel).toBe("banks");
  expect(snow.settings.blockLightPower).not.toBe(
    green.settings.blockLightPower,
  );
  expect(
    snow.settings.atmosphere.cloudHeight +
      snow.settings.atmosphere.cloudThickness,
  ).toBeLessThanOrEqual(snow.analysis.size[1] * 0.55);
});
it("anchors automatic art direction to a curated scene and preserves its lighting and grading", () => {
  const result = directModel(
    analyseModel(model(20, true), { lights: 3, triangles: 50000 }),
    { intent: "cinematic", atmosphere: "clouds" },
  );
  expect(result.reference.id).toBeTruthy();
  const reference = renderPresets[result.reference.id];
  expect(result.settings.azimuth).toBe(reference.azimuth);
  expect(result.settings.sun).toBe(reference.sun);
  expect(result.settings.saturation).toBe(reference.saturation);
  expect(result.reference.alternatives.length).toBeGreaterThan(0);
});

it("rejects empty and malformed model evidence", () => {
  expect(() => analyseModel([], { lights: 0, triangles: 0 })).toThrow();
  expect(() =>
    analyseModel([{ ...model()[0], area: NaN }], { lights: 0, triangles: 1 }),
  ).toThrow();
});
it("keeps oblique geometry normals unit length for physically bounded illumination", () => {
  const a = analyseModel(
    model().map((s) => ({ ...s, normal: [Math.SQRT1_2, 0, Math.SQRT1_2] })),
    { lights: 0, triangles: 512 },
  );
  for (const n of a.normals)
    expect(Math.hypot(...n.direction)).toBeCloseTo(1, 5);
});

it("couples sun/sky/color with user time including midnight and preserves unrelated settings", () => {
  const noon = applyTimeOfDay(renderDefaults, 12),
    night = applyTimeOfDay(renderDefaults, 0),
    dawn = applyTimeOfDay(renderDefaults, 6.5);
  expect(noon.sun).toBeGreaterThan(night.sun);
  expect(night.sky).toBe("night");
  expect(dawn.warmth).not.toBe(noon.warmth);
  expect(night.timeLinked).toBe(true);
  expect(night.sunLinked).toBe(false);
  expect(night.atmosphere).toEqual(renderDefaults.atmosphere);
  expect(() => applyTimeOfDay(renderDefaults, NaN)).toThrow();
});

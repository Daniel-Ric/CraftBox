import { expect, it } from "vitest";
it("clears previous atmosphere and film lights when switching to a clear look", () => {
  const before = applyRenderPreset(
    { ...renderDefaults, studioSpot: true },
    "cloudGolden",
  );
  const after = applyRenderPreset(before, "crisp");
  expect(after.atmosphere.clouds).toBe(false);
  expect(after.atmosphere.fog).toBe(false);
  expect(after.studioSpot).toBe(false);
  expect(after).toEqual(applyRenderPreset(renderDefaults, "crisp"));
});
import {
  applyRenderPreset,
  renderDefaults,
  renderDimensions,
  renderPresets,
  renderQuality,
  renderLookDescriptions,
} from "./render-settings";
it("offers reference looks with distinct color grades and preserves export framing", () => {
  const s = { ...renderDefaults, aspect: 1, edge: 3840, transparent: true };
  for (const id of ["island", "garden", "film", "winterevening", "crisp"]) {
    const look = applyRenderPreset(s, id);
    expect(look.edge).toBe(3840);
    expect(look.aspect).toBe(1);
    expect(look.transparent).toBe(true);
    expect(renderLookDescriptions[id]).toBeTruthy();
    expect(look.saturation).toBeGreaterThan(0);
  }
  expect(renderPresets.island.saturation).toBeGreaterThan(
    renderPresets.film.saturation!,
  );
});
it("keeps 4K and portrait renders within device limits", () => {
  expect(renderDimensions(3840, 16 / 9)).toEqual({ width: 3840, height: 2160 });
  expect(renderDimensions(1920, 3 / 4)).toEqual({ width: 1440, height: 1920 });
  expect(() => renderDimensions(3840, 16 / 9, 2048)).toThrow();
  expect(() => renderDimensions(1920, NaN)).toThrow();
});
it("offers distinct lighting looks without replacing output framing", () => {
  expect(Object.keys(renderPresets).length).toBeGreaterThanOrEqual(10);
  const current = {
    ...renderDefaults,
    edge: 3840,
    aspect: 1,
    transparent: true,
  };
  for (const id of Object.keys(renderPresets)) {
    const applied = applyRenderPreset(current, id);
    expect([applied.edge, applied.aspect, applied.transparent]).toEqual([
      3840,
      1,
      true,
    ]);
    expect(applied.environment).toBeGreaterThanOrEqual(0);
    expect(applied.ao).toBeLessThanOrEqual(1);
  }
  expect(renderPresets.night.sky).toBe("night");
  expect(renderQuality.fast.samples).toBeLessThan(renderQuality.fine.samples);
});

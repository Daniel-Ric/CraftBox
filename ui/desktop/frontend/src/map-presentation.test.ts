import { expect, it } from "vitest";
import { mapDefaults, mapRenderSettings } from "./map-presentation";
import { applyRenderPreset, renderDefaults } from "./render-settings";
import { createCloudLayer } from "./cloud-layers";
it("renders a readable 2D map without inheriting clouds, fog, bloom or cinematic lights", () => {
  const scene = applyRenderPreset(renderDefaults, "cloudGolden");
  scene.atmosphere = {
    ...scene.atmosphere,
    cloudLayers: [createCloudLayer("volume")],
  };
  const map = mapRenderSettings(scene, {
    ...mapDefaults,
    enabled: true,
    shading: false,
    transparent: true,
  });
  expect(map.atmosphere.clouds).toBe(false);
  expect(map.atmosphere.fog).toBe(false);
  expect(map.atmosphere.cloudLayers).toEqual([]);
  expect(map.bloom).toBe(0);
  expect(map.blockLights).toBe(false);
  expect(map.studioSpot).toBe(false);
  expect(map.sky).toBe("none");
  expect(map.ao).toBe(0);
  expect(map.transparent).toBe(true);
  expect(scene.atmosphere.clouds).toBe(true);
  expect(scene.atmosphere.cloudLayers).toHaveLength(1);
});

import { expect, it } from "vitest";
import { Box3, Vector3 } from "three";
import { createCloudLayer, keyCloudValue } from "./cloud-layers";
import {
  cloudRenderBudget,
  prepareCloudLayers,
  cloudOpeningMask,
  atmosphereAnimating,
} from "./cloud-render-config";
import { atmosphereDefaults } from "./atmosphere-settings";
it("excludes empty volumes after sampling their animation without losing later visibility", () => {
  const layer = createCloudLayer("volume"),
    bounds = new Box3(new Vector3(-10, -10, -10), new Vector3(10, 10, 10));
  for (const patch of [{ density: 0 }, { coverage: 0 }, { dissolve: 1 }])
    expect(
      prepareCloudLayers([{ ...layer, ...patch }], bounds, 0),
    ).toHaveLength(0);
  let animated = keyCloudValue(layer, "dissolve", 0, 1, 24);
  animated = keyCloudValue(animated, "dissolve", 1, 0, 24);
  expect(prepareCloudLayers([animated], bounds, 0)).toHaveLength(0);
  expect(prepareCloudLayers([animated], bounds, 1)).toHaveLength(1);
});
it("uses independent preview/export budgets without downgrading saved settings", () => {
  const s = {
    ...atmosphereDefaults,
    cloudPreviewQuality: "low" as const,
    cloudExportQuality: "ultra" as const,
  };
  expect(cloudRenderBudget(s, "fine", false)).toEqual({
    steps: 40,
    resolution: 0.35,
    shadowSteps: 3,
  });
  expect(cloudRenderBudget(s, "fast", true)).toEqual({
    steps: 192,
    resolution: 1,
    shadowSteps: 8,
  });
  expect(s.cloudPreviewQuality).toBe("low");
});
it("transforms local volumes around the scene center with rotated world bounds", () => {
  const l = {
    ...createCloudLayer("volume"),
    position: [1, 0, 0] as [number, number, number],
    size: [2, 1, 0.5] as [number, number, number],
    rotation: [0, 90, 0] as [number, number, number],
  };
  const bounds = new Box3(new Vector3(-10, -10, -10), new Vector3(10, 10, 10));
  const [p] = prepareCloudLayers([l], bounds, 0);
  const center = new Vector3(Math.sqrt(300), 0, 0);
  expect(center.clone().applyMatrix4(p.worldToLocal).length()).toBeLessThan(
    1e-8,
  );
  expect(p.bounds.max.z - p.bounds.min.z).toBeCloseTo(Math.sqrt(300) * 2);
  expect(p.bounds.max.x - p.bounds.min.x).toBeCloseTo(Math.sqrt(300) * 0.5);
});
it("opens a feathered elliptic region above its floor while retaining lower cloud banks", () => {
  const o = {
    strength: 1,
    center: [0, 0] as [number, number],
    radius: [1, 0.6] as [number, number],
    floor: -0.5,
    softness: 0.2,
    irregularity: 0,
  };
  expect(cloudOpeningMask([0, 0, 0], o)).toBe(0);
  expect(cloudOpeningMask([0, -1, 0], o)).toBe(1);
  expect(cloudOpeningMask([2, 0, 0], o)).toBe(1);
  expect(cloudOpeningMask([0.9, 0, 0], o)).toBeGreaterThan(0);
});
it("keeps authored-layer wind and curves repainting without a legacy cloud layer", () => {
  const s = {
    ...atmosphereDefaults,
    clouds: false,
    fog: false,
    windSpeed: 0,
    animate: true,
    cloudLayers: [{ ...createCloudLayer("volume"), windSpeed: 2 }],
  };
  expect(atmosphereAnimating(s)).toBe(true);
  s.cloudLayers[0].windSpeed = 0;
  expect(atmosphereAnimating(s)).toBe(false);
  s.cloudLayers[0].tracks = [
    {
      channel: "position.x",
      keys: [
        {
          id: "a",
          time: 0,
          value: 0,
          interpolation: "linear",
          incoming: [0, 0],
          outgoing: [0, 0],
        },
        {
          id: "b",
          time: 2,
          value: 1,
          interpolation: "linear",
          incoming: [0, 0],
          outgoing: [0, 0],
        },
      ],
    },
  ];
  expect(atmosphereAnimating(s)).toBe(true);
  expect(atmosphereAnimating({ ...s, animate: false })).toBe(false);
  expect(
    atmosphereAnimating({
      ...s,
      cloudLayers: s.cloudLayers.map((l) => ({ ...l, enabled: false })),
    }),
  ).toBe(false);
});

import { expect, it } from "vitest";
import { atmosphereDefaults, normalizeAtmosphere } from "./atmosphere-settings";
import {
  createCloudLayer,
  keyCloudValue,
  sampleCloudLayer,
  unwrapCloudRotation,
} from "./cloud-layers";

const layer = () => ({
  id: "foreground",
  name: "Foreground",
  kind: "volume",
  enabled: true,
  position: [-0.8, -0.4, 0.7],
  size: [1.2, 0.6, 0.8],
  rotation: [0, 30, 0],
  color: "#eef8ff",
  density: 1.3,
  coverage: 0.7,
  detail: 0.6,
  erosion: 0.35,
  turbulence: 0.4,
  seed: 51,
  softness: 0.3,
  shadows: 0.8,
  windSpeed: 0,
  windDirection: 90,
  tracks: [],
});
it("migrates existing atmosphere profiles without adding authored cloud objects", () => {
  const migrated = normalizeAtmosphere({
    clouds: true,
    cloudModel: "banks",
    cloudHeight: -40,
  });
  expect(migrated).toMatchObject({
    clouds: true,
    cloudModel: "banks",
    cloudHeight: -40,
    cloudLayers: [],
    cloudOpening: { strength: 0 },
    cloudPreviewQuality: "auto",
    cloudExportQuality: "high",
  });
});
it("round trips independent cloud objects including transforms and bounded animation", () => {
  const authored = {
    ...layer(),
    tracks: [
      {
        channel: "density",
        keys: [
          {
            id: "appear",
            time: 2,
            value: 0,
            interpolation: "linear",
            incoming: [0, 0],
            outgoing: [0, 0],
          },
          {
            id: "solid",
            time: 4,
            value: 3,
            interpolation: "bezier",
            incoming: [-0.4, 0],
            outgoing: [0.4, 0],
          },
        ],
      },
    ],
  };
  const parsed = normalizeAtmosphere(
    JSON.parse(
      JSON.stringify({ ...atmosphereDefaults, cloudLayers: [authored] }),
    ),
  );
  expect((parsed as any).cloudLayers).toEqual([authored]);
  expect((parsed as any).cloudLayers[0]).not.toBe(authored);
});
it("rejects corrupt cloud objects instead of silently importing unusable GPU data", () => {
  for (const change of [
    { size: [0, 1, 1] },
    { position: [NaN, 0, 0] },
    { rotation: [0, Infinity, 0] },
    { kind: "unknown" },
    { density: -1 },
    { color: "red" },
    {
      tracks: [
        { channel: "density", keys: [{ id: "bad", time: -1, value: 1 }] },
      ],
    },
  ]) {
    expect(() =>
      normalizeAtmosphere({ cloudLayers: [{ ...layer(), ...change }] } as any),
    ).toThrow();
  }
  expect(() =>
    normalizeAtmosphere({ cloudLayers: [layer(), layer()] } as any),
  ).toThrow();
  expect(() =>
    normalizeAtmosphere({
      cloudLayers: Array.from({ length: 9 }, (_, i) => ({
        ...layer(),
        id: String(i),
      })),
    } as any),
  ).toThrow();
});
it("animates each cloud property independently, bounds overshoot and keeps exact holds", () => {
  const base = createCloudLayer("volume");
  let animated = keyCloudValue(base, "position.x", 0, -1, 24);
  animated = keyCloudValue(animated, "position.x", 2, 1, 24);
  animated = keyCloudValue(animated, "density", 0, 0, 24);
  animated = keyCloudValue(animated, "density", 2, 4, 24);
  expect(sampleCloudLayer(animated, 1).position[0]).toBeCloseTo(0);
  expect(sampleCloudLayer(animated, 1).density).toBeCloseTo(2);
  expect(sampleCloudLayer(animated, 1).size).toEqual(base.size);
  const original = JSON.stringify(animated);
  const sampled = sampleCloudLayer(animated, 100);
  sampled.position[0] = 20;
  expect(JSON.stringify(animated)).toBe(original);
  const overshoot = {
    ...animated,
    tracks: animated.tracks.map((t) =>
      t.channel === "density"
        ? {
            ...t,
            keys: t.keys.map((k, i) => ({
              ...k,
              outgoing: [0.5, 40] as [number, number],
              incoming: [-0.5, 40] as [number, number],
              interpolation: i === 0 ? ("bezier" as const) : k.interpolation,
            })),
          }
        : t,
    ),
  };
  expect(sampleCloudLayer(overshoot, 1).density).toBe(4);
});
it("retains multi-turn rotation and the nearest Euler branch when manipulating a cloud", () => {
  expect(unwrapCloudRotation([0, 5, 0], [0, 720, 0])).toEqual([0, 725, 0]);
  expect(unwrapCloudRotation([0, -179, 0], [0, 179, 0])).toEqual([0, 181, 0]);
  expect(unwrapCloudRotation([-60, -70, -50], [120, 250, 130])).toEqual([
    120, 250, 130,
  ]);
});

import { expect, it } from "vitest";
import {
  emissionAttributes,
  selectEmitters,
  decodeSceneLights,
  LightingRig,
} from "./render-lighting";
import { Box3, DirectionalLight, PerspectiveCamera, Vector3 } from "three";
import { renderDefaults } from "./render-settings";
it("suppresses artificial block glow in daylight while retaining night illumination", () => {
  const rig = new LightingRig(),
    camera = new PerspectiveCamera(),
    bounds = new Box3(new Vector3(-5, -5, -5), new Vector3(5, 5, 5));
  rig.setChunk(
    "0,0",
    [{ position: [0, 1, 0], level: 15, color: [1, 1, 1] }],
    new Vector3(),
  );
  rig.update(
    { ...renderDefaults, sun: 0.18, ambient: 0.2, environment: 0.16 },
    camera,
    bounds,
  );
  const night = rig.points[0].intensity,
    nightEmission = rig.emission.value;
  rig.update({ ...renderDefaults, sun: 3.5, elevation: 55 }, camera, bounds);
  expect(rig.points[0].intensity).toBeLessThan(night * 0.05);
  expect(rig.emission.value).toBeLessThan(nightEmission * 0.05);
  const linkedSun = new DirectionalLight(0xffffff, 3);
  linkedSun.position.set(100, 1, 0);
  rig.update(
    { ...renderDefaults, sunLinked: true, elevation: 55 },
    camera,
    bounds,
    linkedSun,
  );
  expect(rig.points[0].intensity).toBeGreaterThan(night * 0.9);
  rig.dispose();
});
it("invalidates shadow depth when light range or spot projection changes", () => {
  const rig = new LightingRig(),
    camera = new PerspectiveCamera(),
    bounds = new Box3(new Vector3(-5, -5, -5), new Vector3(5, 5, 5));
  rig.setChunk(
    "0,0",
    [{ position: [0, 1, 0], level: 15, color: [1, 1, 1] }],
    new Vector3(),
  );
  rig.update(renderDefaults, camera, bounds);
  expect(rig.update(renderDefaults, camera, bounds)).toBe(false);
  expect(
    rig.update({ ...renderDefaults, blockLightRange: 2 }, camera, bounds),
  ).toBe(true);
  rig.update({ ...renderDefaults, studioSpot: true }, camera, bounds);
  expect(
    rig.update(
      { ...renderDefaults, studioSpot: true, spotAngle: 60 },
      camera,
      bounds,
    ),
  ).toBe(true);
  rig.dispose();
});
it("lights only emitter surfaces and selects stable nearby sources without mutating scene data", () => {
  const lights = decodeSceneLights([
    { position: [0.5, 0.5, 0.5], level: 15, color: [1, 0.5, 0.2] },
    { position: [20.5, 0.5, 0.5], level: 8, color: [0.2, 0.5, 1] },
  ]);
  const positions = new Float32Array([0, 1, 0, 0, 1, 1, 1, 1, 1]);
  const normals = new Int8Array([0, 127, 0, 0, 127, 0, 0, 127, 0]);
  expect([
    ...emissionAttributes(
      positions,
      normals,
      new Uint32Array([0, 1, 2]),
      lights,
    ),
  ]).toEqual([1, 1, 1]);
  expect([
    ...emissionAttributes(positions, normals, new Uint32Array([0, 1, 2]), []),
  ]).toEqual([0, 0, 0]);
  expect(selectEmitters(lights, [0, 0, 0], 1)[0].position).toEqual([
    0.5, 0.5, 0.5,
  ]);
  expect(lights).toHaveLength(2);
  expect(() =>
    decodeSceneLights([{ position: [NaN, 0, 0], level: 15, color: [1, 1, 1] }]),
  ).toThrow();
  expect(() =>
    decodeSceneLights([{ position: [0, 0, 0], level: 99, color: [1, 1, 1] }]),
  ).toThrow();
  expect(decodeSceneLights(undefined)).toEqual([]);
});

import { describe, expect, it, vi } from "vitest";
import { RectAreaLightUniformsLib } from "three/addons/lights/RectAreaLightUniformsLib.js";
import { Box3, Vector3, PointLight, SpotLight, RectAreaLight } from "three";
import {
  createStudioLight,
  parseStudioLights,
  StudioLightRig,
} from "./studio-lights";
import { decodeRenderStyle, encodeRenderStyle } from "./render-style-library";
import { renderDefaults, applyRenderPreset } from "./render-settings";

describe("studio light objects", () => {
  it("initializes global area-light lookup textures once across rigs", () => {
    const initialize = vi.spyOn(RectAreaLightUniformsLib, "init");
    const rig = new StudioLightRig(),
      other = new StudioLightRig();
    const bounds = new Box3(new Vector3(-1, -1, -1), new Vector3(1, 1, 1));
    rig.update([createStudioLight("area"), createStudioLight("area")], bounds);
    other.update([createStudioLight("area")], bounds);
    expect(initialize).toHaveBeenCalledTimes(1);
    rig.dispose();
    other.dispose();
    initialize.mockRestore();
  });
  it("loads old projects and round trips independent light objects", () => {
    expect(parseStudioLights(undefined)).toEqual([]);
    const a = createStudioLight("spot"),
      b = createStudioLight("area");
    expect(a.id).not.toBe(b.id);
    const settings = { ...renderDefaults, studioLights: [a, b] };
    const loaded = decodeRenderStyle(
      encodeRenderStyle("Lights", settings),
    ).settings;
    expect(loaded.studioLights).toEqual([a, b]);
    expect(applyRenderPreset(settings, "daylight").studioLights).toEqual([
      a,
      b,
    ]);
  });
  it("rejects malformed transforms, duplicate ids and excessive shadow lights", () => {
    const a = createStudioLight("point");
    expect(() =>
      parseStudioLights([{ ...a, position: [NaN, 1, 0] }]),
    ).toThrow();
    expect(() => parseStudioLights([{ ...a, color: "red" }])).toThrow();
    expect(() => parseStudioLights([a, a])).toThrow();
    expect(() => parseStudioLights([{ ...a, power: -1 }])).toThrow();
    expect(() =>
      parseStudioLights(
        Array.from({ length: 5 }, () => ({
          ...createStudioLight("point"),
          shadows: true,
        })),
      ),
    ).toThrow();
  });
  it("uses world-relative placement, inverse-square lights, spot targets and area size", () => {
    const rig = new StudioLightRig();
    const point = createStudioLight("point"),
      spot = createStudioLight("spot"),
      area = createStudioLight("area");
    point.position = [1, 0, 0];
    const bounds = new Box3(new Vector3(-2, -2, -2), new Vector3(2, 2, 2));
    rig.update([point, spot, area], bounds);
    const p = rig.lights.get(point.id) as PointLight;
    expect(p).toBeInstanceOf(PointLight);
    expect(p.position.x).toBeCloseTo(Math.sqrt(12));
    expect(p.decay).toBe(2);
    expect(p.intensity).toBeCloseTo(point.power / (4 * Math.PI));
    const s = rig.lights.get(spot.id) as SpotLight;
    expect(s).toBeInstanceOf(SpotLight);
    expect(s.target.position.toArray()).toEqual([0, 0, 0]);
    expect(rig.lights.get(area.id)).toBeInstanceOf(RectAreaLight);
    expect((rig.lights.get(area.id) as RectAreaLight).width).toBeCloseTo(
      area.width * Math.sqrt(12),
    );
    rig.dispose();
  });
  it("reuses unchanged GPU lights and releases removed lights and shadow resources", () => {
    const rig = new StudioLightRig(),
      a = createStudioLight("point");
    const bounds = new Box3(new Vector3(-1, -1, -1), new Vector3(1, 1, 1));
    expect(rig.update([a], bounds)).toBe(true);
    const original = rig.lights.get(a.id);
    expect(rig.update([a], bounds)).toBe(false);
    expect(rig.lights.get(a.id)).toBe(original);
    rig.update([{ ...a, enabled: false }], bounds);
    expect(original?.visible).toBe(false);
    rig.update([], bounds);
    expect(rig.lights.size).toBe(0);
    expect(rig.group.children.length).toBe(0);
    rig.dispose();
  });
});

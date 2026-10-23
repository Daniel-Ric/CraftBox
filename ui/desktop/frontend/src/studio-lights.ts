import * as THREE from "three";
import { RectAreaLightUniformsLib } from "three/addons/lights/RectAreaLightUniformsLib.js";

export type LightVector = [number, number, number];
export type StudioLight = {
  id: string;
  name: string;
  type: "point" | "spot" | "area";
  enabled: boolean;
  color: string;

  power: number;

  position: LightVector;
  target: LightVector;
  range: number;
  angle: number;
  penumbra: number;
  width: number;
  height: number;
  shadows: boolean;
};

export function createStudioLight(type: StudioLight["type"]): StudioLight {
  return {
    id: crypto.randomUUID(),
    name: { point: "Point light", spot: "Spot light", area: "Area light" }[
      type
    ],
    type,
    enabled: true,
    color: "#fff0da",
    power: type === "area" ? 4000 : 2000,
    position: [0.7, 0.8, 0.7],
    target: [0, 0, 0],
    range: 4,
    angle: 35,
    penumbra: 0.4,
    width: 0.5,
    height: 0.5,
    shadows: false,
  };
}

export function parseStudioLights(raw: unknown): StudioLight[] {
  if (raw === undefined) return [];
  if (!Array.isArray(raw) || raw.length > 16)
    throw new Error("You can add up to 16 scene lights.");
  const ids = new Set<string>();
  let shadows = 0;
  return raw.map((value) => {
    if (!value || typeof value !== "object")
      throw new Error("Invalid scene light.");
    const l = value as StudioLight;
    const vector = (v: unknown): v is LightVector =>
      Array.isArray(v) &&
      v.length === 3 &&
      v.every((n) => Number.isFinite(n) && Math.abs(n) <= 100);
    const range = (n: number, min: number, max: number) =>
      Number.isFinite(n) && n >= min && n <= max;
    if (
      typeof l.id !== "string" ||
      !l.id ||
      l.id.length > 100 ||
      ids.has(l.id) ||
      typeof l.name !== "string" ||
      !l.name.trim() ||
      l.name.length > 100 ||
      !["point", "spot", "area"].includes(l.type) ||
      typeof l.enabled !== "boolean" ||
      typeof l.shadows !== "boolean" ||
      !/^#[a-f\d]{6}$/i.test(l.color) ||
      !vector(l.position) ||
      !vector(l.target) ||
      !range(l.power, 0, 1000000) ||
      !range(l.range, 0, 100) ||
      !range(l.angle, 1, 89) ||
      !range(l.penumbra, 0, 1) ||
      !range(l.width, 0.001, 100) ||
      !range(l.height, 0.001, 100) ||
      (l.type === "area" && l.shadows)
    )
      throw new Error("Invalid scene light: " + (l.name ?? ""));
    if (l.shadows && ++shadows > 4)
      throw new Error("Up to four scene lights can cast shadows.");
    ids.add(l.id);
    return { ...l, position: [...l.position], target: [...l.target] };
  });
}

type LocalLight = THREE.PointLight | THREE.SpotLight | THREE.RectAreaLight;
let areaUniformsInitialized = false;
export class StudioLightRig {
  group = new THREE.Group();
  lights = new Map<string, LocalLight>();
  private signature = "";

  update(objects: StudioLight[], bounds: THREE.Box3): boolean {
    const signature = JSON.stringify([
      objects,
      bounds.min.toArray(),
      bounds.max.toArray(),
    ]);
    if (signature === this.signature) return false;
    this.signature = signature;
    const radius = bounds.isEmpty()
      ? 1
      : Math.max(1, bounds.getBoundingSphere(new THREE.Sphere()).radius);
    const center = bounds.isEmpty()
      ? new THREE.Vector3()
      : bounds.getCenter(new THREE.Vector3());
    for (const [id, light] of this.lights) {
      const data = objects.find((o) => o.id === id);
      if (!data || data.type !== light.userData.studioType)
        this.remove(id, light);
    }
    for (const o of objects) {
      let light = this.lights.get(o.id);
      if (!light) {
        if (o.type === "area" && !areaUniformsInitialized) {
          RectAreaLightUniformsLib.init();
          areaUniformsInitialized = true;
        }
        light =
          o.type === "point"
            ? new THREE.PointLight()
            : o.type === "spot"
              ? new THREE.SpotLight()
              : new THREE.RectAreaLight();
        light.userData.studioType = o.type;
        this.lights.set(o.id, light);
        this.group.add(light);
        if (light instanceof THREE.SpotLight) this.group.add(light.target);
      }
      light.name = o.name;
      light.visible = o.enabled;
      light.color.set(o.color);
      light.position.fromArray(o.position).multiplyScalar(radius).add(center);
      const target = new THREE.Vector3()
        .fromArray(o.target)
        .multiplyScalar(radius)
        .add(center);
      if (light instanceof THREE.RectAreaLight) {
        light.width = o.width * radius;
        light.height = o.height * radius;
        light.intensity = o.power / (Math.PI * light.width * light.height);
        light.lookAt(target);
      } else {
        light.power = o.power;
        light.distance = o.range * radius;
        light.decay = 2;
        light.castShadow = o.shadows && o.enabled;
        light.shadow.mapSize.set(1024, 1024);
        light.shadow.bias = -0.00005;
        light.shadow.normalBias = radius * 0.0005;
        light.shadow.camera.near = Math.max(0.01, radius * 0.001);
        light.shadow.camera.far =
          o.range > 0
            ? Math.max(light.shadow.camera.near + 1, o.range * radius)
            : radius * 200;
        light.shadow.camera.updateProjectionMatrix();
        if (light instanceof THREE.SpotLight) {
          light.target.position.copy(target);
          light.angle = THREE.MathUtils.degToRad(o.angle);
          light.penumbra = o.penumbra;
        }
      }
    }
    this.group.updateMatrixWorld(true);
    return true;
  }
  private remove(id: string, light: LocalLight) {
    if (light instanceof THREE.SpotLight) this.group.remove(light.target);
    if (!(light instanceof THREE.RectAreaLight)) light.shadow.dispose();
    light.removeFromParent();
    this.lights.delete(id);
  }
  dispose() {
    for (const [id, light] of this.lights) this.remove(id, light);
    this.signature = "";
  }
}

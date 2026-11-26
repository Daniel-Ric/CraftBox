import * as THREE from "three";
import { RectAreaLightUniformsLib } from "three/addons/lights/RectAreaLightUniformsLib.js";

export type LightVector = [number, number, number];
type LightPose = Pick<StudioLight, "position" | "target" | "color" | "power" | "angle" | "penumbra">;
type LightKey = LightPose & { time: number; ease?: "linear" | "smooth" | "hold" };
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
  gobo?: string;
  keys?: LightKey[];
  receiver?: string;
  receiverMode?: "include" | "exclude";
};

export function receiverBlocks(value?: string): string[] {
  return [...new Set(value?.split(/[,\s]+/).filter(Boolean) ?? [])];
}

export function validLightReceivers(value: string) {
  const blocks = receiverBlocks(value);
  return blocks.length > 0 && blocks.length <= 8 && blocks.every((block) => block.length <= 120 && /^[a-z0-9_.-]+:[a-z0-9_./-]+$/.test(block));
}

export function keyStudioLight(light: StudioLight, time: number): StudioLight {
  const previous = light.keys?.find((key) => key.time === time);
  if ((light.keys?.length ?? 0) >= 2000 && !previous) return light;
  const { position, target, color, power, angle, penumbra } = light;
  return {
    ...light,
    keys: [...(light.keys ?? []).filter((key) => key.time !== time), {
      time, position, target, color, power, angle, penumbra, ease: previous?.ease,
    }].sort((a, b) => a.time - b.time),
  };
}

export function sampleStudioLights(lights: StudioLight[], time?: number): StudioLight[] {
  if (time === undefined || !lights.some((light) => light.keys?.length)) return lights;
  return lights.map((light) => {
    const keys = light.keys;
    if (!keys?.length) return light;
    const next = keys.findIndex((key) => key.time >= time);
    const b = keys[next < 0 ? keys.length - 1 : next];
    const a = keys[Math.max(0, next < 0 ? keys.length - 1 : next - 1)];
    const weight = a === b ? 0 : (time - a.time) / (b.time - a.time);
    const t = a.ease === "hold" && weight < 1 ? 0 : a.ease === "smooth" ? weight * weight * (3 - 2 * weight) : weight;
    const mix = THREE.MathUtils.lerp;
    const vector = (from: LightVector, to: LightVector): LightVector => [
      mix(from[0], to[0], t), mix(from[1], to[1], t), mix(from[2], to[2], t),
    ];
    return {
      ...light,
      position: vector(a.position, b.position),
      target: vector(a.target, b.target),
      color: `#${new THREE.Color(a.color).lerp(new THREE.Color(b.color), t).getHexString()}`,
      power: mix(a.power, b.power, t),
      angle: mix(a.angle, b.angle, t),
      penumbra: mix(a.penumbra, b.penumbra, t),
    };
  });
}

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
    const pose = (value: LightPose) =>
      value && typeof value.color === "string" && /^#[a-f\d]{6}$/i.test(value.color) &&
      vector(value.position) && vector(value.target) && range(value.power, 0, 1000000) &&
      range(value.angle, 1, 89) && range(value.penumbra, 0, 1);
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
      !pose(l) ||
      !range(l.range, 0, 100) ||
      !range(l.width, 0.001, 100) ||
      !range(l.height, 0.001, 100) ||
      (l.receiver !== undefined && (typeof l.receiver !== "string" || l.receiver.length > 1000 || !validLightReceivers(l.receiver))) ||
      (l.receiverMode !== undefined && !["include", "exclude"].includes(l.receiverMode)) ||
      (l.gobo !== undefined && (l.type !== "spot" || typeof l.gobo !== "string" ||
        l.gobo.length > 1800000 || !/^data:image\/png;base64,[A-Za-z0-9+/]+={0,2}$/.test(l.gobo))) ||
      (l.keys !== undefined && (!Array.isArray(l.keys) || l.keys.length > 2000 ||
        l.keys.some((key, index, keys) => !pose(key) || !range(key.time, 0, 86400) ||
          (key.ease !== undefined && !["linear", "smooth", "hold"].includes(key.ease)) ||
          (index > 0 && keys[index - 1].time >= key.time)))) ||
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
  private gobos: (string | undefined)[] = [];
  private textures = new Map<THREE.SpotLight, Promise<void>>();

  private updateGobo(light: THREE.SpotLight, source?: string) {
    if (light.userData.gobo === source) return;
    light.userData.gobo = source;
    light.map?.dispose();
    light.map = null;
    this.textures.delete(light);
    if (!source) return;
    let ready: () => void = () => {};
    const pending = new Promise<void>((resolve) => { ready = resolve; });
    const texture = new THREE.TextureLoader().load(source, () => {
      ready();
      if (light.map === texture) this.onTextureReady?.();
    }, undefined, () => {
      ready();
      if (light.map === texture) {
        light.map = null;
        texture.dispose();
        this.onTextureReady?.();
      }
    });
    texture.colorSpace = THREE.SRGBColorSpace;
    light.map = texture;
    this.textures.set(light, pending);
  }

  onTextureReady?: () => void;

  waitForTextures() { return Promise.all(this.textures.values()); }

  update(objects: StudioLight[], bounds: THREE.Box3): boolean {
    const signature = JSON.stringify([
      objects.map(({ keys, gobo, ...light }) => light),
      bounds.min.toArray(),
      bounds.max.toArray(),
    ]);
    if (signature === this.signature && objects.every((light, index) => light.gobo === this.gobos[index])) return false;
    this.signature = signature;
    this.gobos = objects.map((light) => light.gobo);
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
          this.updateGobo(light, o.gobo);
        }
      }
    }
    this.group.updateMatrixWorld(true);
    return true;
  }
  private remove(id: string, light: LocalLight) {
    if (light instanceof THREE.SpotLight) {
      this.group.remove(light.target);
      this.textures.delete(light);
      light.map?.dispose();
      light.map = null;
    }
    if (!(light instanceof THREE.RectAreaLight)) light.shadow.dispose();
    light.removeFromParent();
    this.lights.delete(id);
  }
  dispose() {
    for (const [id, light] of this.lights) this.remove(id, light);
    this.signature = "";
    this.gobos = [];
  }
}

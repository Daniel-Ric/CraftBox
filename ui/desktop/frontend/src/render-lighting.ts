import * as THREE from "three";
import type { SceneLight } from "./bridge";
import type { RenderSettings } from "./render-settings";

const skyDirections = [
  [0, 0], [0.65, 0], [-0.65, 0], [0, 0.65], [0, -0.65],
] as const;

export function sampleIndirectLight(
  point: THREE.Vector3,
  blockLevel: number,
  occupied: (x: number, y: number, z: number) => boolean,
  emitters: readonly SceneLight[],
) {
  let sky = 0;
  for (const [dx, dz] of skyDirections) {
    let distance = 1;
    while (distance <= 20 && !occupied(
      Math.floor(point.x + dx * distance),
      Math.floor(point.y + distance),
      Math.floor(point.z + dz * distance),
    )) distance++;
    sky += distance > 20 ? 0.2 : 0.018 * Math.min(8, distance - 1);
  }
  const block = [0, 0, 0];
  if (blockLevel <= 0) return { sky, block };
  let total = 0;
  for (const { position, level, color } of emitters) {
    const distance = (position[0] - point.x) ** 2 +
      (position[1] - point.y) ** 2 + (position[2] - point.z) ** 2;
    if (distance > 196) continue;
    const weight = (level / 15) ** 2 * (1 - distance / 196) / Math.max(2, distance);
    total += weight;
    for (let axis = 0; axis < 3; axis++) block[axis] += color[axis] * weight;
  }
  const strength = 0.8 ** (15 - blockLevel);
  for (let axis = 0; axis < 3; axis++)
    block[axis] = strength * (total > 0 ? block[axis] / total : 1);
  return { sky, block };
}

export function installIndirectLight(
  material: THREE.Material,
  power: { value: number },
  skyColor: { value: THREE.Color },
  blockPower: { value: number },
) {
  const previous = material.onBeforeCompile;
  const key = material.customProgramCacheKey();
  material.onBeforeCompile = (shader, renderer) => {
    previous.call(material, shader, renderer);
    shader.uniforms.indirectLightPower = power;
    shader.uniforms.indirectSkyColor = skyColor;
    shader.uniforms.indirectBlockPower = blockPower;
    shader.vertexShader = shader.vertexShader
      .replace("#include <common>", "#include <common>\nattribute vec4 indirectProbe; varying vec4 vIndirectProbe;")
      .replace("#include <begin_vertex>", "#include <begin_vertex>\nvIndirectProbe = indirectProbe;");
    shader.fragmentShader = shader.fragmentShader
      .replace("#include <common>", "#include <common>\nvarying vec4 vIndirectProbe; uniform float indirectLightPower, indirectBlockPower; uniform vec3 indirectSkyColor;")
      .replace("#include <opaque_fragment>", `
        vec3 bounce = vIndirectProbe.rgb * indirectBlockPower + indirectSkyColor * vIndirectProbe.a;
        outgoingLight += diffuseColor.rgb * bounce * indirectLightPower;
        #include <opaque_fragment>`);
  };
  material.customProgramCacheKey = () => key + "-indirect-light";
}

export function decodeSceneLights(raw: unknown): SceneLight[] {
  if (raw === undefined) return [];
  if (!Array.isArray(raw) || raw.length > 131072)
    throw new Error("Invalid block light list");
  return raw.map((l) => {
    if (
      !l ||
      !Number.isInteger(l.level) ||
      l.level < 1 ||
      l.level > 15 ||
      !Array.isArray(l.position) ||
      l.position.length !== 3 ||
      !l.position.every(
        (n: unknown) =>
          typeof n === "number" && Number.isFinite(n) && Math.abs(n) < 40000000,
      ) ||
      !Array.isArray(l.color) ||
      l.color.length !== 3 ||
      !l.color.every(
        (n: unknown) =>
          typeof n === "number" && Number.isFinite(n) && n >= 0 && n <= 1,
      )
    )
      throw new Error("Invalid block light data");
    return {
      position: [...l.position],
      level: l.level,
      color: [...l.color],
    } as SceneLight;
  });
}
export function selectEmitters(
  lights: readonly SceneLight[],
  focus: readonly number[],
  budget: number,
) {
  const distance = (l: SceneLight) =>
    l.position.reduce((d, p, i) => d + (p - focus[i]) ** 2, 0);
  return lights
    .map((l, i) => ({
      l,
      i,
      score: (l.level * l.level) / Math.max(4, distance(l)),
    }))
    .sort((a, b) => b.score - a.score || a.i - b.i)
    .slice(0, budget)
    .map((v) => v.l);
}
export function emissionAttributes(
  positions: Float32Array,
  normals: Int8Array,
  indices: Uint32Array|Uint16Array,
  lights: SceneLight[],
) {
  const result = new Uint8Array(positions.length / 3);
	if(lights.length===0)return result;
  const cells = new Map(
    lights.map((l) => [l.position.map(Math.floor).join(","), Math.round(l.level*17)]),
  );
  for (let t = 0; t < indices.length; t += 3) {
    const a = indices[t] * 3,
      b = indices[t + 1] * 3,
      c = indices[t + 2] * 3;
    const key = [0, 1, 2]
      .map((i) =>
        Math.floor(
          (positions[a + i] + positions[b + i] + positions[c + i]) / 3 -
            (normals[a + i] / 127) * 0.001,
        ),
      )
      .join(",");
    const level = cells.get(key) ?? 0;
    for (let k = 0; k < 3; k++)
      result[indices[t + k]] = Math.max(result[indices[t + k]], level);
  }
  return result;
}
export function installEmission(
  material: THREE.Material,
  power: { value: number },
  shadowOnly = false,
) {
  const previous = material.onBeforeCompile,
    cache = material.customProgramCacheKey();
  material.onBeforeCompile = (shader, renderer) => {
    previous.call(material, shader, renderer);
    shader.uniforms.blockEmissionPower = power;
    shader.vertexShader = shader.vertexShader
      .replace(
        "#include <common>",
        "#include <common>\nattribute float blockEmission; varying float vBlockEmission;",
      )
      .replace(
        "#include <begin_vertex>",
        "#include <begin_vertex>\nvBlockEmission=blockEmission;",
      );
    shader.fragmentShader = shader.fragmentShader.replace(
      "#include <common>",
      "#include <common>\nvarying float vBlockEmission; uniform float blockEmissionPower;",
    );
    if (shadowOnly)
      shader.fragmentShader = shader.fragmentShader.replace(
        "#include <alphatest_fragment>",
        "#include <alphatest_fragment>\nif(vBlockEmission>0.)discard;",
      );
    else
      shader.fragmentShader = shader.fragmentShader.replace(
        "#include <emissivemap_fragment>",
        "#include <emissivemap_fragment>\ntotalEmissiveRadiance+=diffuseColor.rgb*vBlockEmission*blockEmissionPower;",
      );
  };
  material.customProgramCacheKey = () => cache + "-block-emission-v1";
  material.needsUpdate = true;
}



export function installBlockLight(
  material: THREE.Material,
  power: { value: number },
) {
  const previous = material.onBeforeCompile,
    cache = material.customProgramCacheKey();
  material.onBeforeCompile = (shader, renderer) => {
    previous.call(material, shader, renderer);
    shader.uniforms.blockSurfacePower = power;
    shader.vertexShader = shader.vertexShader
      .replace(
        "#include <common>",
        "#include <common>\nattribute float blockLight; varying float vBlockLight;",
      )
      .replace(
        "#include <begin_vertex>",
        "#include <begin_vertex>\nvBlockLight=blockLight;",
      );
    shader.fragmentShader = shader.fragmentShader
      .replace(
        "#include <common>",
        "#include <common>\nvarying float vBlockLight; uniform float blockSurfacePower;",
      )
      .replace(
        "#include <opaque_fragment>",
        `
        float blockBrightness = vBlockLight > 0.0 ? pow(0.8, 15.0 - vBlockLight) : 0.0;
        outgoingLight = max(outgoingLight, diffuseColor.rgb * blockBrightness * blockSurfacePower);
        #include <opaque_fragment>`,
      );
  };
  material.customProgramCacheKey = () => cache + "-block-surface-v1";
  material.needsUpdate = true;
}

export class LightingRig {
  readonly group = new THREE.Group();
  readonly points = Array.from(
    { length: 24 },
    () => new THREE.PointLight(0xffffff, 0, 16, 2),
  );
  readonly spot = new THREE.SpotLight(0xffe3b5, 0, 0, Math.PI / 6, 0.5, 2);
  readonly emission = { value: 1.5 };
  readonly surfacePower = { value: 1 };
  readonly depth = new THREE.MeshDepthMaterial({
    depthPacking: THREE.BasicDepthPacking,
    alphaTest: 0.5,
    side: THREE.DoubleSide,
  });
  private chunks = new Map<string, SceneLight[]>();
  private bakedChunks = new Set<string>();
  private dirty = true;
  private all: SceneLight[] = [];
  private legacy: SceneLight[] = [];
  private selected: SceneLight[] = [];
  private shadowed = new Set<SceneLight>();
  private selectionPosition = new THREE.Vector3(Infinity, Infinity, Infinity);
  private selectionBudget = 0;
  private selectionShadowBudget = 0;
  constructor() {
    this.group.name = "CraftBox lighting";
    this.group.add(...this.points, this.spot, this.spot.target);
    for (const p of this.points) {
      p.shadow.mapSize.set(512, 512);
      p.shadow.camera.near = 0.05;
      p.shadow.bias = -0.001;
      p.shadow.normalBias = 0.025;
    }
    this.spot.shadow.mapSize.set(1024, 1024);
    this.spot.shadow.camera.near = 0.1;
    this.spot.shadow.bias = -0.0003;
    this.spot.shadow.normalBias = 0.025;
    installEmission(this.depth, this.emission, true);
  }
  setChunk(
    id: string,
    lights: SceneLight[],
    offset: THREE.Vector3,
    baked = false,
  ) {
    if (baked) this.bakedChunks.add(id);
    else this.bakedChunks.delete(id);
    this.chunks.set(
      id,
      lights.map((l) => ({
        ...l,
        position: [
          l.position[0] + offset.x,
          l.position[1] + offset.y,
          l.position[2] + offset.z,
        ],
      })),
    );
    this.dirty = true;
  }
  clear() {
    this.chunks.clear();
    this.bakedChunks.clear();
    this.all = [];
    this.legacy = [];
    this.selected = [];
    this.shadowed.clear();
    this.dirty = true;
  }
  update(
    settings: RenderSettings,
    camera: THREE.Camera,
    bounds: THREE.Box3,
    sun?: THREE.DirectionalLight,
    focusPoint = camera.position,
  ) {
    const sunHeight = sun
      ? sun.position.clone().sub(sun.target.position).normalize().y
      : Math.sin(THREE.MathUtils.degToRad(settings.elevation));
    const daylight = settings.blockDaylightAdaptation
      ? THREE.MathUtils.smoothstep(
          (sun?.intensity ?? settings.sun) * Math.max(0, sunHeight),
          0.4,
          1.8,
        )
      : 0;
    const artificialPower = 1 - daylight;
    const usePointLights =
      settings.blockLights && settings.blockLightPower > 0 && artificialPower > 0;
    const focus = focusPoint
      .clone()
      .divideScalar(2)
      .floor()
      .multiplyScalar(2)
      .addScalar(1);
    const budget = Math.round(settings.lightBudget);
    const shadowBudget = Math.min(budget, Math.round(settings.blockLightShadows));
    const reselect =
      this.dirty ||
      !this.selectionPosition.equals(focus) ||
      this.selectionBudget !== budget ||
      this.selectionShadowBudget !== shadowBudget;
    if (usePointLights && this.dirty) {
      const chunks = [...this.chunks].sort(([a], [b]) => a.localeCompare(b));
      this.all = chunks.flatMap(([, lights]) => lights);
      this.legacy = chunks
        .filter(([id]) => !this.bakedChunks.has(id))
        .flatMap(([, lights]) => lights);
      this.dirty = false;
    }
    if (usePointLights && reselect) {
      const position = focus.toArray();
      const shadowed = shadowBudget
        ? selectEmitters(this.all, position, shadowBudget)
        : [];
      this.shadowed = new Set(shadowed);
      this.selected = [
        ...shadowed,
        ...selectEmitters(
          this.legacy.filter((light) => !this.shadowed.has(light)),
          position,
          budget - shadowed.length,
        ),
      ];
      this.selectionPosition.copy(focus);
      this.selectionBudget = budget;
      this.selectionShadowBudget = shadowBudget;
    }

    this.surfacePower.value = settings.blockLights
      ? settings.blockLightPower
      : 0;
    this.emission.value = settings.blockLights
      ? settings.emissionPower * (0.25 + 0.75 * artificialPower)
      : 0;
    const picked = usePointLights ? this.selected : [];
    let changed = false;
    this.points.forEach((p, i) => {
      const l = picked[i],
        visible = !!l,
        cast = visible && this.shadowed.has(l);
      if (p.visible !== visible) changed = true;
      p.visible = visible;
      if (p.castShadow !== cast) {
        p.castShadow = cast;
        changed = true;
      }
      if (!l) {
        p.intensity = 0;
        return;
      }
      const pos = new THREE.Vector3(...l.position);
      if (!p.position.equals(pos)) {
        p.position.copy(pos);
        changed = true;
      }
      const range = (l.level + 1) * settings.blockLightRange;
      if (p.distance !== range) changed = true;
      p.color.setRGB(...l.color, THREE.SRGBColorSpace);
      p.intensity =
        45 * settings.blockLightPower * (l.level / 15) ** 2 * artificialPower;
      p.distance = range;
      p.shadow.camera.far = p.distance;
      p.shadow.radius = settings.softness;
    });
    const center = bounds.isEmpty()
        ? new THREE.Vector3()
        : bounds.getCenter(new THREE.Vector3()),
      scale = bounds.isEmpty()
        ? 10
        : Math.max(2, bounds.getBoundingSphere(new THREE.Sphere()).radius);
    const pos = new THREE.Vector3(
        settings.spotX,
        settings.spotY,
        settings.spotZ,
      )
        .multiplyScalar(scale)
        .add(center),
      target = new THREE.Vector3(
        settings.spotTargetX,
        settings.spotTargetY,
        settings.spotTargetZ,
      )
        .multiplyScalar(scale)
        .add(center);
    if (pos.distanceToSquared(target) < 0.01) target.y -= 1;
    if (
      !this.spot.position.equals(pos) ||
      !this.spot.target.position.equals(target)
    ) {
      changed = true;
      this.spot.position.copy(pos);
      this.spot.target.position.copy(target);
    }
    const spotAngle = THREE.MathUtils.degToRad(settings.spotAngle);
    if (this.spot.angle !== spotAngle || this.spot.distance !== scale * 12)
      changed = true;
    this.spot.intensity = settings.studioSpot
      ? settings.spotPower * scale * scale * 0.04
      : 0;
    this.spot.color.set(settings.spotColor);
    this.spot.angle = spotAngle;
    this.spot.penumbra = settings.spotSoftness;
    this.spot.distance = scale * 12;
    this.spot.shadow.camera.far = this.spot.distance;
    this.spot.shadow.radius = 1 + settings.spotSoftness * 3;
    if (this.spot.castShadow !== settings.studioSpot) {
      changed = true;
      this.spot.castShadow = settings.studioSpot;
    }
    this.spot.visible = settings.studioSpot;
    return changed;
  }
  dispose() {
    for (const p of [...this.points, this.spot]) p.shadow.map?.dispose();
    this.depth.dispose();
    this.chunks.clear();
  }
}

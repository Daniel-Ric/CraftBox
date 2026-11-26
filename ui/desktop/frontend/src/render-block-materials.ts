import * as THREE from "three";
import type { PackedPart } from "./bridge";
import {
  receiverBlocks,
  type StudioLight,
  type StudioLightRig,
} from "./studio-lights";

export type BlockMaterial = {
  block: string;
  color: string;
  tint: boolean;
  roughness: number;
  metalness: number;
  emission: number;
};
export const blockMaterialLimit = 16;
export const blockName = /^[a-z0-9_.-]+:[a-z0-9_./-]+$/;

export function parseBlockMaterials(raw: unknown): BlockMaterial[] {
  if (raw === undefined) return [];
  if (!Array.isArray(raw) || raw.length > blockMaterialLimit)
    throw new Error("Invalid block materials.");
  const names = new Set<string>();
  return raw.map((value: BlockMaterial) => {
    if (
      !value ||
      typeof value.block !== "string" ||
      value.block.length > 120 ||
      !blockName.test(value.block) ||
      names.has(value.block) ||
      typeof value.color !== "string" ||
      !/^#[a-f0-9]{6}$/i.test(value.color) ||
      typeof value.tint !== "boolean" ||
      !Number.isFinite(value.roughness) ||
      value.roughness < 0 ||
      value.roughness > 1 ||
      !Number.isFinite(value.metalness) ||
      value.metalness < 0 ||
      value.metalness > 1 ||
      !Number.isFinite(value.emission) ||
      value.emission < 0 ||
      value.emission > 25
    )
      throw new Error("Invalid block materials.");
    names.add(value.block);
    return value;
  });
}

export class RenderBlockMaterials {
  private names = new Map<string, number>();
  readonly blockNames = new Map<number, string>();
  private clay = { value: new THREE.Vector4() };
  private properties = {
    value: Array.from(
      { length: blockMaterialLimit },
      () => new THREE.Vector4(),
    ),
  };
  private materialCount = { value: 0 };
  private colors = {
    value: Array.from(
      { length: blockMaterialLimit },
      () => new THREE.Vector4(),
    ),
  };
  private receivers = new Map<
    THREE.Light,
    { ids: number[]; exclude: boolean }
  >();
  private linkedLights = {
    value: Array.from({ length: 16 }, () => new THREE.Vector3()),
  };
  private firstReceivers = {
    value: Array.from({ length: 16 }, () => new THREE.Vector4()),
  };
  private secondReceivers = {
    value: Array.from({ length: 16 }, () => new THREE.Vector4()),
  };
  private linkCount = { value: 0 };
  private installed = new WeakSet<THREE.Material>();

  private id(name: string) {
    let id = this.names.get(name);
    if (id === undefined) {
      id = this.names.size + 1;
      this.names.set(name, id);
      this.blockNames.set(id, name);
    }
    return id;
  }

  attribute(runs: PackedPart["blockRuns"], count: number) {
    const ids = new Float32Array(count);
    let start = 0;
    for (const run of runs ?? []) {
      ids.fill(this.id(run.name), start, run.end);
      start = run.end;
    }
    return new THREE.BufferAttribute(ids, 1);
  }

  update(clay: boolean, clayColor: string, materials: BlockMaterial[]) {
    const color = new THREE.Color(clayColor);
    this.clay.value.set(color.r, color.g, color.b, Number(clay));
    this.materialCount.value = Math.min(materials.length, blockMaterialLimit);
    this.properties.value.forEach((p, i) => {
      const material = materials[i];
      if (!material) {
        p.set(0, 0, 0, 0);
        return;
      }
      p.set(
        this.id(material.block),
        material.roughness,
        material.metalness,
        material.emission,
      );
      color.set(material.color);
      this.colors.value[i].set(
        color.r,
        color.g,
        color.b,
        Number(material.tint),
      );
    });
  }

  link(lights: StudioLight[], rig: StudioLightRig) {
    this.receivers.clear();
    for (const light of lights) {
      if (!light.enabled || !light.receiver) continue;
      const object = rig.lights.get(light.id);
      if (object)
        this.receivers.set(object, {
          ids: receiverBlocks(light.receiver).map((name) => this.id(name)),
          exclude: light.receiverMode === "exclude",
        });
    }
  }

  prepareLighting(scene: THREE.Scene, camera: THREE.Camera) {
    this.linkCount.value = 0;
    if (!this.receivers.size) return;
    const lights: (THREE.PointLight | THREE.SpotLight | THREE.RectAreaLight)[] =
      [];
    scene.traverseVisible((item) => {
      if (
        (item instanceof THREE.PointLight ||
          item instanceof THREE.SpotLight ||
          item instanceof THREE.RectAreaLight) &&
        item.layers.test(camera.layers)
      )
        lights.push(item);
    });
    const order = (light: (typeof lights)[number]) =>
      Number(light.castShadow) * 2 +
      Number(light instanceof THREE.SpotLight && !!light.map);
    lights.sort((a, b) => order(b) - order(a));
    const indices = [0, 0, 0];
    let index = 0;
    for (const light of lights) {
      const type =
        light instanceof THREE.PointLight
          ? 0
          : light instanceof THREE.SpotLight
            ? 1
            : 2;
      const position = indices[type]++;
      const receiver = this.receivers.get(light);
      if (!receiver) continue;
      this.linkedLights.value[index].set(
        type,
        position,
        Number(receiver.exclude),
      );
      const [a = -1, b = -1, c = -1, d = -1, e = -1, f = -1, g = -1, h = -1] =
        receiver.ids;
      this.firstReceivers.value[index].set(a, b, c, d);
      this.secondReceivers.value[index].set(e, f, g, h);
      index++;
    }
    this.linkCount.value = index;
  }

  install(material: THREE.Material) {
    if (this.installed.has(material)) return;
    this.installed.add(material);
    const previous = material.onBeforeCompile;
    const cacheKey = material.customProgramCacheKey.bind(material);
    const linkLighting = (source: string) =>
      source
        .replace(
          /get(Point|Spot)LightInfo\( (\w+), geometryPosition, directLight \);/g,
          (match, type) =>
            `${match}\n directLight.color *= studioReceiver(${type === "Point" ? 0 : 1}.0, float(UNROLLED_LOOP_INDEX));`,
        )
        .replace(
          "rectAreaLight = rectAreaLights[ i ];",
          "$&\n rectAreaLight.color *= studioReceiver(2.0, float(UNROLLED_LOOP_INDEX));",
        );
    material.customProgramCacheKey = () =>
      `${cacheKey()}|studio-block-materials`;
    material.onBeforeCompile = (shader, renderer) => {
      previous.call(material, shader, renderer);
      shader.uniforms.studioClay = this.clay;
      shader.uniforms.studioProperties = this.properties;
      shader.uniforms.studioMaterialCount = this.materialCount;
      shader.uniforms.studioColors = this.colors;
      shader.uniforms.studioLinkedLights = this.linkedLights;
      shader.uniforms.studioFirstReceivers = this.firstReceivers;
      shader.uniforms.studioSecondReceivers = this.secondReceivers;
      shader.uniforms.studioLinkCount = this.linkCount;
      shader.vertexShader =
        `attribute float studioBlock; varying float vStudioBlock;\n${shader.vertexShader}`.replace(
          "#include <begin_vertex>",
          "$&\n vStudioBlock = studioBlock;",
        );
      shader.fragmentShader = `varying float vStudioBlock;
        uniform vec4 studioClay;
        uniform vec4 studioProperties[${blockMaterialLimit}];
        uniform int studioMaterialCount;
        uniform vec4 studioColors[${blockMaterialLimit}];
        uniform vec3 studioLinkedLights[16];
        uniform vec4 studioFirstReceivers[16], studioSecondReceivers[16];
        uniform int studioLinkCount;
        float studioReceiver(float type, float index) {
          for (int i = 0; i < studioLinkCount; i++) {
            vec3 link = studioLinkedLights[i];
            if (link.x == type && link.y == index) {
              vec4 block = vec4(vStudioBlock);
              bool matches = any(lessThan(abs(studioFirstReceivers[i] - block), vec4(0.5))) ||
                any(lessThan(abs(studioSecondReceivers[i] - block), vec4(0.5)));
              return float(matches != (link.z > 0.5));
            }
          }
          return 1.0;
        }\n${shader.fragmentShader}`
        .replace(
          "#include <alphamap_fragment>",
          `$&
          vec4 studioParameters = vec4(0.0);
          if (studioClay.a > 0.5) { diffuseColor.rgb = studioClay.rgb; }
          else if (vStudioBlock > 0.5) {
            for (int i = 0; i < studioMaterialCount; i++) {
              if (abs(studioProperties[i].x - vStudioBlock) >= 0.5) continue;
              studioParameters = studioProperties[i];
              if (studioColors[i].a > 0.5) diffuseColor.rgb *= studioColors[i].rgb;
              break;
            }
          }`,
        )
        .replace(
          "#include <roughnessmap_fragment>",
          "$&\n if(studioClay.a > 0.5) roughnessFactor = 0.82; else if(studioParameters.x > 0.0) roughnessFactor = studioParameters.y;",
        )
        .replace(
          "#include <metalnessmap_fragment>",
          "$&\n if(studioClay.a > 0.5) metalnessFactor = 0.0; else if(studioParameters.x > 0.0) metalnessFactor = studioParameters.z;",
        )
        .replace(
          "#include <lights_fragment_end>",
          "$&\n if(studioClay.a > 0.5) totalEmissiveRadiance = vec3(0.0); else totalEmissiveRadiance += diffuseColor.rgb * studioParameters.w;",
        );
      shader.fragmentShader = linkLighting(
        shader.fragmentShader.replace(
          "#include <lights_fragment_begin>",
          THREE.ShaderChunk.lights_fragment_begin,
        ),
      );
    };
  }
}

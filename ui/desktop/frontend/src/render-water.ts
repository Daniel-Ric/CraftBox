import * as THREE from "three";
import { Reflector } from "three/addons/objects/Reflector.js";
import type { RenderSettings } from "./render-settings";

export function waterLevels(
  positions: Float32Array,
  indices: Uint32Array,
  normals?: Int8Array,
) {
  const levels = new Map<number, number>();
  for (let i = 0; i + 2 < indices.length; i += 3) {
    const a = indices[i] * 3,
      b = indices[i + 1] * 3,
      c = indices[i + 2] * 3;
    const height = positions[a + 1];
    if (
      (normals && normals[a + 1] < 100) ||
      Math.abs(height - positions[b + 1]) > 0.01 ||
      Math.abs(height - positions[c + 1]) > 0.01
    )
      continue;
    const area =
      Math.abs(
        (positions[b] - positions[a]) * (positions[c + 2] - positions[a + 2]) -
          (positions[c] - positions[a]) * (positions[b + 2] - positions[a + 2]),
      ) / 2;
    if (area > 0) {
      const level = Math.round(height * 16) / 16;
      levels.set(level, (levels.get(level) ?? 0) + area);
    }
  }
  return levels;
}

export class WaterSurface {
  private chunkLevels = new Map<string, Map<number, number>>();
  private areas = new Map<number, number>();
  private reflector?: Reflector;
  material = new THREE.MeshPhysicalMaterial({
    roughness: 0.2,
    metalness: 0,
    ior: 1.333,
    clearcoat: 1,
    clearcoatRoughness: 0.15,
    transparent: true,
    depthWrite: false,
    side: THREE.DoubleSide,
  });
  uniforms = {
    waterDepth: { value: null as THREE.DepthTexture | null },
    waterResolution: { value: new THREE.Vector2(1, 1) },
    waterInverseProjection: { value: new THREE.Matrix4() },
    waterPerspective: { value: false },
    waterOpacity: { value: 0.35 },
    waterVisibility: { value: 12 },
    waterWaves: { value: 0.15 },
    waterTint: { value: new THREE.Color("#ffffff") },
    waterShallowColor: { value: new THREE.Color() },
    waterDeepColor: { value: new THREE.Color() },
    waterDeepLightColor: { value: new THREE.Color() },
    waterColorStrength: { value: 0.65 },
    waterColorDepth: { value: 18 },
    waterTextureStrength: { value: 0.35 },
    waterWaveScale: { value: 1 },
    waterTransparency: { value: 0 },
    waterDeepLight: { value: 0.08 },
    waterShoreSoftness: { value: 1.5 },
    waterSceneMap: { value: null as THREE.Texture | null },
    waterSceneMatrix: { value: new THREE.Matrix4() },
    waterSceneLevel: { value: 0 },
    waterSceneStrength: { value: 0 },
    waterReflectionDetailDistance: { value: 240 },
  };
  setChunk(key: string, levels: Map<number, number>) {
    for (const [height, area] of this.chunkLevels.get(key) ?? []) {
      const remaining = (this.areas.get(height) ?? 0) - area;
      if (remaining > 0.0001) this.areas.set(height, remaining);
      else this.areas.delete(height);
    }
    if (levels.size) this.chunkLevels.set(key, levels);
    else this.chunkLevels.delete(key);
    for (const [height, area] of levels)
      this.areas.set(height, (this.areas.get(height) ?? 0) + area);
  }
  clear() {
    this.chunkLevels.clear();
    this.areas.clear();
    this.uniforms.waterSceneStrength.value = 0;
  }
  dispose() {
    this.reflector?.geometry.dispose();
    this.reflector?.dispose();
  }
  reflect(
    renderer: THREE.WebGLRenderer,
    scene: THREE.Scene,
    camera: THREE.Camera,
    strength: number,
    scale: number,
    guides: THREE.Object3D[],
  ) {
    this.uniforms.waterSceneStrength.value = 0;
    if (!this.material.visible || strength <= 0 || scale <= 0) return;
    let level = 0,
      largest = 16;
    for (const [height, area] of this.areas)
      if (
        height < camera.position.y - 0.05 &&
        (area > largest || (area === largest && height > level))
      ) {
        level = height;
        largest = area;
      }
    if (largest <= 16) return;
    if (!this.reflector) {
      this.reflector = new Reflector(new THREE.PlaneGeometry(), { multisample: 0 });
      const texture = this.reflector.getRenderTarget().texture;
      texture.generateMipmaps = true;
      texture.minFilter = THREE.LinearMipmapLinearFilter;
    }
    const reflector = this.reflector;
    const material = reflector.material;
    if (!(material instanceof THREE.ShaderMaterial)) return;
    const size = renderer
      .getDrawingBufferSize(new THREE.Vector2())
      .multiplyScalar(scale);
    const limit = Math.min(1536, renderer.capabilities.maxTextureSize);
    size.multiplyScalar(Math.min(1, limit / Math.max(size.x, size.y)));
    const target = reflector.getRenderTarget();
    const width = Math.max(1, Math.round(size.x)),
      height = Math.max(1, Math.round(size.y));
    if (target.width !== width || target.height !== height)
      target.setSize(width, height);
    reflector.position.y = level;
    reflector.rotation.x = -Math.PI / 2;
    reflector.updateMatrixWorld(true);
    camera.updateMatrixWorld();
    const visible = guides.map((guide) => guide.visible);
    const background = scene.background;
    const alpha = renderer.getClearAlpha();
    const renderTarget = renderer.getRenderTarget();
    const xr = renderer.xr.enabled;
    const shadowUpdate = renderer.shadowMap.autoUpdate;
    try {
      guides.forEach((guide) => {
        guide.visible = false;
      });
      scene.background = null;
      renderer.setClearAlpha(0);
      this.material.visible = false;
      reflector.onBeforeRender(
        renderer,
        scene,
        camera,
        reflector.geometry,
        material,
        null!,
      );
      this.uniforms.waterSceneMap.value = target.texture;
      this.uniforms.waterSceneMatrix.value
        .copy(material.uniforms.textureMatrix.value)
        .multiply(new THREE.Matrix4().copy(reflector.matrixWorld).invert());
      this.uniforms.waterSceneLevel.value = level;
      this.uniforms.waterSceneStrength.value = Math.min(1, strength);
    } finally {
      this.material.visible = true;
      scene.background = background;
      renderer.setClearAlpha(alpha);
      renderer.setRenderTarget(renderTarget);
      renderer.xr.enabled = xr;
      renderer.shadowMap.autoUpdate = shadowUpdate;
      guides.forEach((guide, index) => {
        guide.visible = visible[index];
      });
    }
  }
  install() {
    const previous = this.material.onBeforeCompile;
    const key = this.material.customProgramCacheKey();
    this.material.onBeforeCompile = (shader, renderer) => {
      previous.call(this.material, shader, renderer);
      Object.assign(shader.uniforms, this.uniforms);
      shader.vertexShader = shader.vertexShader
        .replace(
          "#include <common>",
          "#include <common>\nvarying vec3 waterPosition; varying float waterTop;",
        )
        .replace(
          "#include <project_vertex>",
          "#include <project_vertex>\nwaterPosition=(modelMatrix*vec4(transformed,1.0)).xyz;waterTop=abs(normal.y);",
        );
      shader.fragmentShader = shader.fragmentShader
        .replace(
          "#include <common>",
          `#include <common>
          uniform sampler2D waterDepth; uniform vec2 waterResolution;
          uniform sampler2D waterSceneMap; uniform mat4 waterSceneMatrix;
          uniform float waterSceneLevel,waterSceneStrength,waterReflectionDetailDistance;
          uniform mat4 waterInverseProjection; uniform bool waterPerspective;
          uniform float waterOpacity,waterVisibility,waterWaves;
          uniform vec3 waterTint,waterShallowColor,waterDeepColor,waterDeepLightColor;
          uniform float waterColorStrength,waterColorDepth,waterTextureStrength,waterWaveScale,waterTransparency,waterDeepLight,waterShoreSoftness;
          varying vec3 waterPosition; varying float waterTop;`,
        )
        .replace(
          "#include <color_fragment>",
          `#include <color_fragment>
          vec2 waterUv=gl_FragCoord.xy/waterResolution;
          float bedDepth=texture2D(waterDepth,waterUv).r;
          vec4 bed=waterInverseProjection*vec4(waterUv*2.0-1.0,bedDepth*2.0-1.0,1.0);
          float thickness=bedDepth>=0.999999?waterVisibility*8.0:max(0.0,-bed.z/bed.w-vViewPosition.z);
                                                                                   
          if(waterPerspective)thickness*=length(vViewPosition)/max(0.001,vViewPosition.z);
          float absorption=1.0-exp(-thickness/max(0.25,waterVisibility));
          float shore=smoothstep(0.0,waterShoreSoftness,thickness);
          diffuseColor.a=clamp((waterOpacity*shore+(1.0-waterOpacity*shore)*absorption)*(1.0-waterTransparency),0.0,1.0);
          float waterDepthColor=1.0-exp(-thickness/max(0.25,waterColorDepth));
          vec3 waterMean=diffuseColor.rgb;
          #ifdef USE_MAP
          if(vAtlasTile.z>0.5)waterMean=terrainLevel(4.0,vec2(0.5)).rgb;
          #endif
          vec3 waterTexture=max(vec3(0.0),mix(waterMean,diffuseColor.rgb,waterTextureStrength));
          float waterDetail=clamp(dot(waterTexture,vec3(.2126,.7152,.0722))/max(.001,dot(waterMean,vec3(.2126,.7152,.0722))),.35,2.5);
          vec3 waterPalette=mix(waterShallowColor,waterDeepColor,waterDepthColor)*waterDetail;
          diffuseColor.rgb=mix(waterTexture,waterPalette,waterColorStrength)*waterTint;`,
        )
        .replace(
          "#include <emissivemap_fragment>",
          `#include <emissivemap_fragment>
          totalEmissiveRadiance+=waterDeepLightColor*waterDeepLight*waterDepthColor;`,
        )
        .replace(
          "#include <normal_fragment_maps>",
          `#include <normal_fragment_maps>
          vec2 wavePosition=waterPosition.xz/waterWaveScale;
          vec3 ripple=vec3(sin(wavePosition.x*0.7+wavePosition.y*0.3),0.0,cos(wavePosition.y*0.8-wavePosition.x*0.2));
          float rippleFilter=1.0-smoothstep(1.0,4.0,max(length(dFdx(wavePosition)),length(dFdy(wavePosition))));
          normal=normalize(normal+mat3(viewMatrix)*ripple*waterWaves*0.12*step(0.9,waterTop)*rippleFilter);`,
        )
        .replace(
          "#include <opaque_fragment>",
          `if (waterSceneStrength > 0.0) {
            vec4 projection = waterSceneMatrix * vec4(waterPosition, 1.0);
            vec2 uv = projection.xy / projection.w + ripple.xz * waterWaves * 0.008 * rippleFilter;
            vec2 reflectionDx = dFdx(uv), reflectionDy = dFdy(uv);
            if (waterTop > 0.9 && abs(waterPosition.y - waterSceneLevel) < 0.04 &&
                projection.w > 0.0 && all(greaterThan(uv, vec2(0.0))) && all(lessThan(uv, vec2(1.0)))) {
              float blur = roughnessFactor * roughnessFactor * 4.0
                + 2.0 * smoothstep(0.15, 1.0, vViewPosition.z / waterReflectionDetailDistance);
              float filterWidth = exp2(blur);
              vec4 reflection = textureGrad(waterSceneMap, uv, reflectionDx * filterWidth, reflectionDy * filterWidth);
              float facing = abs(dot(normal, normalize(vViewPosition)));
              float fresnel = 0.025 + 0.975 * pow(1.0 - clamp(facing, 0.0, 1.0), 5.0);
              float edge = smoothstep(0.0, 0.025, min(min(uv.x, uv.y), min(1.0 - uv.x, 1.0 - uv.y)));
              float weight = fresnel * (1.0 - roughnessFactor * 0.8) * waterSceneStrength * edge;
              outgoingLight = outgoingLight * (1.0 - reflection.a * weight) + reflection.rgb * weight;
              diffuseColor.a = max(diffuseColor.a, reflection.a * weight * (1.0 - waterTransparency));
            }
          }
          #include <opaque_fragment>`,
        );
    };
    this.material.customProgramCacheKey = () => key + "-depth-water-v4";
    this.material.needsUpdate = true;
  }
  configure(
    settings: RenderSettings,
    camera: THREE.Camera,
    depth: THREE.DepthTexture,
    width: number,
    height: number,
  ) {
    this.uniforms.waterDepth.value = depth;
    this.uniforms.waterResolution.value.set(width, height);
    this.uniforms.waterInverseProjection.value.copy(
      camera.projectionMatrixInverse,
    );
    this.uniforms.waterPerspective.value =
      camera instanceof THREE.PerspectiveCamera;
    this.uniforms.waterOpacity.value = settings.waterOpacity;
    this.uniforms.waterVisibility.value = settings.waterVisibility;
    this.uniforms.waterWaves.value = settings.waterWaves;
    this.uniforms.waterTint.value.set(settings.waterTint);
    for (const k of [
      "waterColorStrength",
      "waterColorDepth",
      "waterTextureStrength",
      "waterWaveScale",
      "waterTransparency",
      "waterDeepLight",
      "waterShoreSoftness",
      "waterReflectionDetailDistance",
    ] as const)
      this.uniforms[k].value = settings[k];
    for (const k of [
      "waterShallowColor",
      "waterDeepColor",
      "waterDeepLightColor",
    ] as const)
      this.uniforms[k].value.set(settings[k]);
    this.material.visible = settings.waterTransparency < 1;
    this.material.roughness = settings.waterRoughness;
    this.material.envMapIntensity = settings.waterReflections;
  }
}

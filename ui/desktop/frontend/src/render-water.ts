import * as THREE from "three";
import type { RenderSettings } from "./render-settings";

                                                                             
                                                                            
export class WaterSurface {
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
  };
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
        );
    };
    this.material.customProgramCacheKey = () => key + "-depth-water-v2";
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

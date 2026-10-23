import * as THREE from "three";
import { FullScreenQuad, type Pass } from "three/addons/postprocessing/Pass.js";
import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
import { ShaderPass } from "three/addons/postprocessing/ShaderPass.js";
import { UnrealBloomPass } from "three/addons/postprocessing/UnrealBloomPass.js";
import { OutputPass } from "three/addons/postprocessing/OutputPass.js";
import { ColorGradePass } from "./render-color-grade";
import type { RenderSettings } from "./render-settings";

const vertexShader = `varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`;

export class TemporalAccumulator {
  target = new THREE.WebGLRenderTarget(1, 1, {
    type: THREE.HalfFloatType,
    depthBuffer: false,
  });
  material = new THREE.ShaderMaterial({
    uniforms: {
      source: { value: null },
      weight: { value: 1 },
      exposure: { value: 1 },
    },
    vertexShader,
    fragmentShader: `uniform sampler2D source;uniform float weight,exposure;varying vec2 vUv;void main(){vec4 c=texture2D(source,vUv);gl_FragColor=vec4(c.rgb*exposure,c.a)*weight;}`,
    depthTest: false,
    depthWrite: false,
    transparent: true,
    blending: THREE.CustomBlending,
    blendEquation: THREE.AddEquation,
    blendSrc: THREE.OneFactor,
    blendDst: THREE.OneFactor,
    blendEquationAlpha: THREE.AddEquation,
    blendSrcAlpha: THREE.OneFactor,
    blendDstAlpha: THREE.OneFactor,
    toneMapped: false,
  });
  quad = new FullScreenQuad(this.material);
  composer: EffectComposer;
  source: ShaderPass;
  bloom = new UnrealBloomPass(new THREE.Vector2(1, 1), 0, 0.4, 1.2);
  output = new OutputPass();
  grade = new ColorGradePass();
  alpha: ShaderPass;
  readonly renderer: THREE.WebGLRenderer;
  constructor(renderer: THREE.WebGLRenderer, overlay?: Pass) {
    this.renderer = renderer;
    this.composer = new EffectComposer(
      renderer,
      new THREE.WebGLRenderTarget(1, 1, {
        type: THREE.HalfFloatType,
        depthBuffer: false,
      }),
    );
    this.composer.setPixelRatio(1);
    this.source = new ShaderPass({
      uniforms: { source: { value: this.target.texture } },
      vertexShader,
      fragmentShader: `uniform sampler2D source;varying vec2 vUv;void main(){vec4 c=texture2D(source,vUv);gl_FragColor=vec4(c.a>0.000001?c.rgb/c.a:vec3(0.0),c.a);}`,
    });


    this.source.uniforms.source.value = this.target.texture;
    this.alpha = new ShaderPass({
      uniforms: {
        tDiffuse: { value: null },
        premultiply: {
          value: renderer.getContext().getContextAttributes()
            ?.premultipliedAlpha
            ? 1
            : 0,
        },
      },
      vertexShader,
      fragmentShader: `uniform sampler2D tDiffuse;uniform float premultiply;varying vec2 vUv;void main(){vec4 c=texture2D(tDiffuse,vUv);gl_FragColor=vec4(c.rgb*mix(1.0,c.a,premultiply),c.a);}`,
    });
    for (const pass of [
      this.source,
      this.bloom,
      this.output,
      this.grade,
      ...(overlay ? [overlay] : []),
      this.alpha,
    ])
      this.composer.addPass(pass);
  }
  begin(width: number, height: number) {
    if (this.target.width !== width || this.target.height !== height) {
      this.target.setSize(width, height);
      this.composer.setSize(width, height);
    }
    const target = this.renderer.getRenderTarget(),
      color = this.renderer.getClearColor(new THREE.Color()),
      alpha = this.renderer.getClearAlpha();
    try {
      this.renderer.setRenderTarget(this.target);
      this.renderer.setClearColor(0, 0);
      this.renderer.clear();
    } finally {
      this.renderer.setClearColor(color, alpha);
      this.renderer.setRenderTarget(target);
    }
  }
  add(texture: THREE.Texture, weight: number, exposure: number) {
    const target = this.renderer.getRenderTarget(),
      clear = this.renderer.autoClear;
    try {
      this.renderer.autoClear = false;
      this.renderer.setRenderTarget(this.target);
      this.material.uniforms.source.value = texture;
      this.material.uniforms.weight.value = weight;
      this.material.uniforms.exposure.value = exposure;
      this.quad.render(this.renderer);
    } finally {
      this.renderer.autoClear = clear;
      this.renderer.setRenderTarget(target);
    }
  }
  finish(settings: RenderSettings) {
    this.bloom.enabled = settings.bloom > 0 && !settings.transparent;
    this.bloom.strength = settings.bloom;
    this.grade.configure(settings);
    this.grade.setSize(this.target.width, this.target.height);
    this.composer.render();
  }
  dispose() {
    this.target.dispose();
    this.material.dispose();
    this.quad.dispose();
    for (const p of [
      this.source,
      this.bloom,
      this.output,
      this.grade,
      this.alpha,
    ])
      p.dispose();
    this.composer.dispose();
  }
}

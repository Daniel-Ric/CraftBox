import * as THREE from "three";
import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
import { ShaderPass } from "three/addons/postprocessing/ShaderPass.js";
import { OutputPass } from "three/addons/postprocessing/OutputPass.js";
import { ColorGradePass } from "./render-color-grade";
import type { PixelImage } from "./panorama";
import type { RenderSettings } from "./render-settings";


export function renderPanoramaPost(
  renderer: THREE.WebGLRenderer,
  image: PixelImage,
  settings: RenderSettings,
): string {
  const pixels = image.width * image.height;
  const base = new Uint16Array(pixels * 4),
    bright = new Uint16Array(pixels * 4);
  for (let i = 0; i < pixels; i++) {
    const at = i * 4;
    const maximum = Math.max(
      image.data[at],
      image.data[at + 1],
      image.data[at + 2],
    );
    const threshold = THREE.MathUtils.smoothstep(maximum, 0.9, 1.3);
    for (let c = 0; c < 4; c++) {
      const value = Number.isFinite(image.data[at + c])
        ? Math.max(0, Math.min(65504, image.data[at + c]))
        : 0;
      base[at + c] = THREE.DataUtils.toHalfFloat(value);
      bright[at + c] = THREE.DataUtils.toHalfFloat(
        c === 3 ? 1 : value * threshold * image.data[at + 3],
      );
    }
  }
  const source = new THREE.DataTexture(
    base,
    image.width,
    image.height,
    THREE.RGBAFormat,
    THREE.HalfFloatType,
  );
  const highlights = new THREE.DataTexture(
    bright,
    image.width,
    image.height,
    THREE.RGBAFormat,
    THREE.HalfFloatType,
  );
  for (const texture of [source, highlights]) {
    texture.flipY = true;
    texture.wrapS = THREE.RepeatWrapping;
    texture.magFilter = THREE.LinearFilter;
    texture.minFilter = THREE.LinearFilter;
    texture.colorSpace = THREE.LinearSRGBColorSpace;
    texture.needsUpdate = true;
  }
  highlights.generateMipmaps = true;
  highlights.minFilter = THREE.LinearMipmapLinearFilter;
  const target = new THREE.WebGLRenderTarget(image.width, image.height, {
    type: THREE.HalfFloatType,
  });
  target.texture.wrapS = THREE.RepeatWrapping;
  const composer = new EffectComposer(renderer, target);
  composer.setPixelRatio(1);
  composer.setSize(image.width, image.height);
  composer.renderTarget2.texture.wrapS = THREE.RepeatWrapping;
  const sphereBloom = new ShaderPass({
    uniforms: {
      source: { value: source },
      highlights: { value: highlights },
      width: { value: image.width },
      strength: { value: settings.transparent ? 0 : settings.bloom },
    },
    vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
    fragmentShader: `uniform sampler2D source,highlights;uniform float width,strength;varying vec2 vUv;
      const float PI=3.14159265359;
      vec3 direction(vec2 uv){float lon=(uv.x-.5)*2.0*PI,lat=(uv.y-.5)*PI;return vec3(sin(lon)*cos(lat),sin(lat),-cos(lon)*cos(lat));}
      vec2 sphereUV(vec3 d){return vec2(atan(d.x,-d.z)/(2.0*PI)+.5,asin(clamp(d.y,-1.0,1.0))/PI+.5);}
      void main(){
        vec4 original=texture2D(source,vUv);vec3 bloom=vec3(0.0);
        if(strength>0.0){
          vec3 n=direction(vUv),right=normalize(cross(n,abs(n.y)>.99?vec3(1,0,0):vec3(0,1,0))),up=cross(right,n);
          for(int level=0;level<5;level++){
            float radius=.003*pow(2.3,float(level));
            float lod=max(0.0,log2(width*radius/(2.0*PI)));
            vec3 layer=textureLod(highlights,vUv,lod).rgb;
            for(int tap=0;tap<8;tap++){
              float angle=float(tap)*PI*.25;

              vec3 ray=n*cos(radius)+(right*cos(angle)+up*sin(angle))*sin(radius);
              layer+=textureLod(highlights,sphereUV(ray),lod).rgb;
            }
            bloom+=layer/9.0*.2;
          }
        }
        gl_FragColor=vec4(original.rgb+bloom*strength,original.a);
      }`,
  });
  const output = new OutputPass(),
    grade = new ColorGradePass();
  const framebufferAlpha = new ShaderPass({
    uniforms: {
      tDiffuse: { value: null },
      premultiply: {
        value: renderer.getContext().getContextAttributes()?.premultipliedAlpha
          ? 1
          : 0,
      },
    },
    vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
    fragmentShader: `uniform sampler2D tDiffuse;uniform float premultiply;varying vec2 vUv;void main(){vec4 c=texture2D(tDiffuse,vUv);gl_FragColor=vec4(c.rgb*mix(1.0,c.a,premultiply),c.a);}`,
  });
  grade.configure({ ...settings, vignette: 0 });
  composer.addPass(sphereBloom);
  composer.addPass(output);
  composer.addPass(grade);
  composer.addPass(framebufferAlpha);
  try {
    renderer.setSize(image.width, image.height, false);
    composer.render();
    return renderer.domElement.toDataURL("image/png");
  } finally {
    sphereBloom.dispose();
    output.dispose();
    grade.dispose();
    framebufferAlpha.dispose();
    composer.dispose();
    source.dispose();
    highlights.dispose();
  }
}

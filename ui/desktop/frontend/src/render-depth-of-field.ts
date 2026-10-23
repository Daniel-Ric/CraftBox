import {
  Camera,
  OrthographicCamera,
  PerspectiveCamera,
  Texture,
  Vector2,
  Vector3,
} from "three";
import { ShaderPass } from "three/addons/postprocessing/ShaderPass.js";
import type { RenderSettings } from "./render-settings";


export function circleOfConfusion(
  depth: number,
  o: {
    focus: number;
    focalLength: number;
    fStop: number;
    sensorWidth: number;
    width: number;
  },
) {
  const focal = o.focalLength / 1000;
  if (
    ![depth, o.focus, focal, o.fStop, o.sensorWidth, o.width].every(
      Number.isFinite,
    ) ||
    depth <= 0 ||
    o.focus <= focal ||
    o.fStop <= 0 ||
    o.sensorWidth <= 0
  )
    return 0;
  return (
    ((((focal * focal * (depth - o.focus)) /
      (o.fStop * depth * (o.focus - focal))) *
      1000) /
      o.sensorWidth) *
    o.width
  );
}
export function focusDepth(point: Vector3, camera: Camera) {
  camera.updateMatrixWorld(true);
  return Math.max(
    0.001,
    -point.clone().applyMatrix4(camera.matrixWorldInverse).z,
  );
}


export class DepthOfFieldPass extends ShaderPass {
  constructor() {
    super({
      uniforms: {
        tDiffuse: { value: null },
        depthMap: { value: null },
        cloudDepthMap: { value: null },
        cloudDepthEnabled: { value: 0 },
        texel: { value: new Vector2(1, 1) },
        nearClip: { value: 0.1 },
        farClip: { value: 50000 },
        perspective: { value: 1 },
        focus: { value: 100 },
        focal: { value: 0.05 },
        fStop: { value: 2.8 },
        worldScale: { value: 1 },
        maxRadius: { value: 16 },
        imageWidth: { value: 1920 },
        sensorWidth: { value: 36 },
      },
      vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
      fragmentShader: `#include <packing>
        uniform sampler2D tDiffuse,depthMap,cloudDepthMap;
        uniform float cloudDepthEnabled;
        uniform vec2 texel;
        uniform float nearClip,farClip,perspective,focus,focal,fStop,worldScale,maxRadius,imageWidth,sensorWidth;
        varying vec2 vUv;
        float depthAt(vec2 uv){
          float d=texture2D(depthMap,uv).x;
          float z=-(perspective>0.5 ? perspectiveDepthToViewZ(d,nearClip,farClip) : orthographicDepthToViewZ(d,nearClip,farClip));
          if(cloudDepthEnabled>.5){vec2 cloud=texture2D(cloudDepthMap,uv).rg;
            if(cloud.r>.001&&cloud.r<z&&cloud.g>.005)z=1./mix(1./max(z,.001),1./cloud.r,clamp(cloud.g,0.,1.));
          }
          return z/worldScale;
        }
        float radiusAt(float z){
          return clamp(focal*focal*(z-focus)/(fStop*max(z,0.0001)*max(focus-focal,0.0001))*1000.0/sensorWidth*imageWidth*0.5,-maxRadius,maxRadius);
        }
        void main(){
          vec4 source=texture2D(tDiffuse,vUv);
          float centerDepth=depthAt(vUv),centerRadius=radiusAt(centerDepth);

          vec3 sum=source.rgb;float alpha=source.a,weight=1.0;


          for(int i=0;i<48;i++){
            float index=float(i)+0.5;
            float radius=sqrt(index/48.0)*maxRadius;
            float angle=index*2.3999632297;
            vec2 uv=vUv+vec2(cos(angle),sin(angle))*radius*texel;
            if(any(lessThan(uv,vec2(0.0)))||any(greaterThan(uv,vec2(1.0))))continue;
            float z=depthAt(uv),r=radiusAt(z);
            float coverage=1.0-smoothstep(max(abs(centerRadius),abs(r))-0.75,max(abs(centerRadius),abs(r))+0.75,radius);
            if(z>centerDepth && centerRadius>=-0.5)coverage*=smoothstep(0.5,2.0,abs(centerRadius));
            float w=coverage/max(1.0,abs(r)*abs(r)*0.04);
            vec4 sampleColor=texture2D(tDiffuse,uv);
            sum+=sampleColor.rgb*w;alpha+=sampleColor.a*w;weight+=w;
          }
          gl_FragColor=vec4(sum/weight,alpha/weight);
        }`,
    });
    this.enabled = false;
  }
  configure(
    s: RenderSettings,
    camera: PerspectiveCamera | OrthographicCamera,
    depth: Texture,
    map = false,
    cloudDepth?: Texture,
  ) {
    this.enabled = s.dofEnabled && !map;
    if (!this.enabled) return;
    const u = this.uniforms;
    u.depthMap.value = depth;
    u.cloudDepthEnabled.value = cloudDepth ? 1 : 0;
    u.cloudDepthMap.value = cloudDepth ?? depth;
    u.nearClip.value = camera.near;
    u.farClip.value = camera.far;
    u.perspective.value = camera instanceof PerspectiveCamera ? 1 : 0;
    u.focus.value = s.dofDistance / s.dofWorldScale;

    u.focal.value =
      camera instanceof PerspectiveCamera
        ? 0.012 / Math.tan((camera.getEffectiveFOV() * Math.PI) / 360)
        : s.dofFocalLength / 1000;
    u.sensorWidth.value =
      camera instanceof PerspectiveCamera ? 24 * camera.aspect : 36;
    u.focus.value = Math.max(
      u.focal.value + 0.0001,
      s.dofDistance / s.dofWorldScale,
    );
    u.fStop.value = s.dofAperture;
    u.worldScale.value = s.dofWorldScale;
    u.maxRadius.value = s.dofMaxBlur;
  }
  setSize(width: number, height: number) {
    this.uniforms.texel.value.set(1 / width, 1 / height);
    this.uniforms.imageWidth.value = width;
  }
}

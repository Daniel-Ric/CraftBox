import * as THREE from "three";
import type { LightingRig } from "./render-lighting";



export const localVolumeShader = `
uniform float localLightPower;
#if VOLUME_POINTS > 0
uniform samplerCubeShadow localShadowA,localShadowB;
uniform vec4 localPosition[2],localRadiance[2];
uniform vec2 localClip[2];
#endif
#if VOLUME_SPOT > 0
uniform sampler2DShadow localSpotShadow;
uniform mat4 localSpotMatrix;
uniform vec4 localSpotPosition,localSpotRadiance,localSpotDirection;
uniform vec2 localSpotCone;
#endif
vec3 localIllumination(vec3 p,vec3 viewDir){
 vec3 result=vec3(0.);
 #if VOLUME_POINTS > 0
 for(int i=0;i<VOLUME_POINTS;i++){
  vec3 v=p-localPosition[i].xyz;float distanceToLight=length(v),faceZ=max(abs(v.x),max(abs(v.y),abs(v.z)));
  float nearClip=localClip[i].x,farClip=localClip[i].y;
  if(faceZ<=nearClip||faceZ>=farClip||distanceToLight<.01)continue;
  float depth=farClip*(faceZ-nearClip)/(faceZ*(farClip-nearClip))-.001;
  float visible=i==0?texture(localShadowA,vec4(v/distanceToLight,depth)):texture(localShadowB,vec4(v/distanceToLight,depth));
  float cutoff=pow(clamp(1.-pow(distanceToLight/localPosition[i].w,4.),0.,1.),2.);
  result+=localRadiance[i].rgb*localRadiance[i].w*visible*cutoff/max(1.,distanceToLight*distanceToLight)*(.3+.7*pow(max(0.,dot(viewDir,v/distanceToLight)),4.));
 }
 #endif
 #if VOLUME_SPOT > 0
 vec3 v=p-localSpotPosition.xyz;float distanceToLight=length(v);
 if(distanceToLight>.01){
  float cone=smoothstep(localSpotCone.x,max(localSpotCone.x+.00001,localSpotCone.y),dot(normalize(v),localSpotDirection.xyz));
  vec4 projected=localSpotMatrix*vec4(p,1.);vec3 q=projected.xyz/max(.00001,projected.w);
  if(projected.w>0.&&q.x>0.&&q.x<1.&&q.y>0.&&q.y<1.&&q.z>0.&&q.z<1.){
   float visible=texture(localSpotShadow,vec3(q.xy,q.z-.0005));
   float cutoff=pow(clamp(1.-pow(distanceToLight/localSpotPosition.w,4.),0.,1.),2.);
   result+=localSpotRadiance.rgb*localSpotRadiance.w*cone*visible*cutoff/max(1.,distanceToLight*distanceToLight)*(.3+.7*pow(max(0.,dot(viewDir,normalize(v))),4.));
  }
 }
 #endif
 return result*localLightPower;
}
`;

export class VolumeLighting {
  rig?: LightingRig;
  power = 0.6;
  readonly uniforms = {
    localLightPower: { value: 0.6 },
    localShadowA: { value: null as THREE.Texture | null },
    localShadowB: { value: null as THREE.Texture | null },
    localPosition: { value: [new THREE.Vector4(), new THREE.Vector4()] },
    localRadiance: { value: [new THREE.Vector4(), new THREE.Vector4()] },
    localClip: { value: [new THREE.Vector2(), new THREE.Vector2()] },
    localSpotShadow: { value: null as THREE.Texture | null },
    localSpotMatrix: { value: new THREE.Matrix4() },
    localSpotPosition: { value: new THREE.Vector4() },
    localSpotRadiance: { value: new THREE.Vector4() },
    localSpotDirection: { value: new THREE.Vector4() },
    localSpotCone: { value: new THREE.Vector2() },
  };
  sync(material: THREE.ShaderMaterial) {
    const points =
      this.power > 0
        ? (this.rig?.points
            .filter(
              (p) =>
                p.visible &&
                p.intensity > 0 &&
                p.castShadow &&
                p.shadow.map?.depthTexture,
            )
            .slice(0, 2) ?? [])
        : [];
    const spot =
      this.power > 0 &&
      this.rig?.spot.visible &&
      this.rig.spot.intensity > 0 &&
      this.rig.spot.shadow.map?.depthTexture
        ? this.rig.spot
        : undefined;
    if (
      material.defines.VOLUME_POINTS !== points.length ||
      material.defines.VOLUME_SPOT !== Number(!!spot)
    ) {
      material.defines = {
        ...material.defines,
        VOLUME_POINTS: points.length,
        VOLUME_SPOT: Number(!!spot),
      };
      material.needsUpdate = true;
    }
    const u = this.uniforms;
    u.localLightPower.value = this.power;
    for (let i = 0; i < points.length; i++) {
      const p = points[i];
      u.localPosition.value[i].set(
        p.position.x,
        p.position.y,
        p.position.z,
        p.distance,
      );
      u.localRadiance.value[i].set(
        p.color.r,
        p.color.g,
        p.color.b,
        p.intensity,
      );
      u.localClip.value[i].set(p.shadow.camera.near, p.shadow.camera.far);
    }
    if (points.length) {
      u.localShadowA.value = points[0].shadow.map!.depthTexture;
      u.localShadowB.value = (points[1] ?? points[0]).shadow.map!.depthTexture;
    }
    if (spot) {
      u.localSpotShadow.value = spot.shadow.map!.depthTexture;
      u.localSpotMatrix.value.copy(spot.shadow.matrix);
      u.localSpotPosition.value.set(
        spot.position.x,
        spot.position.y,
        spot.position.z,
        spot.distance,
      );
      u.localSpotRadiance.value.set(
        spot.color.r,
        spot.color.g,
        spot.color.b,
        spot.intensity,
      );
      const d = spot.target.position.clone().sub(spot.position).normalize();
      u.localSpotDirection.value.set(d.x, d.y, d.z, 0);
      u.localSpotCone.value.set(
        Math.cos(spot.angle),
        Math.cos(spot.angle * (1 - spot.penumbra)),
      );
    }
  }
}

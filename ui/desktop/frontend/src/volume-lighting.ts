import * as THREE from "three";
import type { LightingRig } from "./render-lighting";
import type { StudioLightRig } from "./studio-lights";

const spotSlots = [0, 1] as const;
const spotDeclarations = spotSlots.map((i) => `
#if VOLUME_SPOT > ${i}
uniform mat4 localSpotMatrix${i};
uniform vec4 localSpotPosition${i},localSpotRadiance${i},localSpotDirection${i};
uniform vec2 localSpotCone${i};
#if VOLUME_SPOT_SHADOW_${i} > 0
uniform sampler2DShadow localSpotShadow${i};
#endif
#if VOLUME_SPOT_GOBO_${i} > 0
uniform sampler2D localSpotGobo${i};
#endif
#endif
`).join("\n");

const spotContributions = spotSlots.map((i) => `
#if VOLUME_SPOT > ${i}
{
 vec3 v=p-localSpotPosition${i}.xyz;float distanceToLight=length(v);
 if(distanceToLight>.01){
  float cone=smoothstep(localSpotCone${i}.x,max(localSpotCone${i}.x+.00001,localSpotCone${i}.y),dot(normalize(v),localSpotDirection${i}.xyz));
  vec4 projected=localSpotMatrix${i}*vec4(p,1.);vec3 q=projected.xyz/max(.00001,projected.w);
  if(cone>0.&&projected.w>0.&&q.x>0.&&q.x<1.&&q.y>0.&&q.y<1.&&q.z>0.&&q.z<1.){
   vec3 mask=vec3(1.0);
   #if VOLUME_SPOT_SHADOW_${i} > 0
   mask*=texture(localSpotShadow${i},vec3(q.xy,q.z-.0005));
   #endif
   #if VOLUME_SPOT_GOBO_${i} > 0
   mask*=texture(localSpotGobo${i},q.xy).rgb;
   #endif
   float reach=localSpotPosition${i}.w;
   float cutoff=reach>0.?pow(clamp(1.-pow(distanceToLight/reach,4.),0.,1.),2.):1.;
   result+=localSpotRadiance${i}.rgb*localSpotRadiance${i}.w*cone*mask*cutoff/max(1.,distanceToLight*distanceToLight)*(.3+.7*pow(max(0.,dot(viewDir,normalize(v))),4.));
  }
 }
}
#endif
`).join("\n");

export const localVolumeShader = `
uniform float localLightPower;
#if VOLUME_POINTS > 0
uniform samplerCubeShadow localShadowA,localShadowB;
uniform vec4 localPosition[2],localRadiance[2];
uniform vec2 localClip[2];
#endif
${spotDeclarations}
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
 ${spotContributions}
 return result*localLightPower;
}
`;

export class VolumeLighting {
  rig?: LightingRig;
  studio?: StudioLightRig;
  power = 0.6;
  private readonly spots = spotSlots.map(() => ({
    Shadow: { value: null as THREE.Texture | null },
    Gobo: { value: null as THREE.Texture | null },
    Matrix: { value: new THREE.Matrix4() },
    Position: { value: new THREE.Vector4() },
    Radiance: { value: new THREE.Vector4() },
    Direction: { value: new THREE.Vector4() },
    Cone: { value: new THREE.Vector2() },
  }));
  readonly uniforms = {
    localLightPower: { value: 0.6 },
    localShadowA: { value: null as THREE.Texture | null },
    localShadowB: { value: null as THREE.Texture | null },
    localPosition: { value: [new THREE.Vector4(), new THREE.Vector4()] },
    localRadiance: { value: [new THREE.Vector4(), new THREE.Vector4()] },
    localClip: { value: [new THREE.Vector2(), new THREE.Vector2()] },
    ...Object.fromEntries(this.spots.flatMap((spot, i) => Object.entries(spot).map(([name, uniform]) => [`localSpot${name}${i}`, uniform]))),
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
    const spots = this.power > 0 ? [this.rig?.spot, ...(this.studio?.lights.values() ?? [])]
      .filter((light): light is THREE.SpotLight => light instanceof THREE.SpotLight && light.visible && light.intensity > 0)
      .sort((a, b) => Number(!!b.map) - Number(!!a.map) || b.intensity - a.intensity).slice(0, this.spots.length) : [];
    const spotDefines = Object.fromEntries(spotSlots.flatMap((i) => [
      [`VOLUME_SPOT_SHADOW_${i}`, Number(!!(spots[i]?.castShadow && spots[i]?.shadow.map?.depthTexture))],
      [`VOLUME_SPOT_GOBO_${i}`, Number(!!spots[i]?.map?.image)],
    ]));
    if (material.defines.VOLUME_POINTS !== points.length || material.defines.VOLUME_SPOT !== spots.length ||
      Object.entries(spotDefines).some(([key, value]) => material.defines[key] !== value)) {
      material.defines = {
        ...material.defines,
        VOLUME_POINTS: points.length,
        VOLUME_SPOT: spots.length,
        ...spotDefines,
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
    spots.forEach((spot, i) => {
      const uniforms = this.spots[i];
      uniforms.Shadow.value = spot.shadow.map?.depthTexture ?? null;
      uniforms.Gobo.value = spot.map;
      if (!spot.castShadow) spot.shadow.updateMatrices(spot);
      uniforms.Matrix.value.copy(spot.shadow.matrix);
      uniforms.Position.value.set(
        spot.position.x,
        spot.position.y,
        spot.position.z,
        spot.distance,
      );
      uniforms.Radiance.value.set(
        spot.color.r,
        spot.color.g,
        spot.color.b,
        spot.intensity,
      );
      const d = spot.target.position.clone().sub(spot.position).normalize();
      uniforms.Direction.value.set(d.x, d.y, d.z, 0);
      uniforms.Cone.value.set(
        Math.cos(spot.angle),
        Math.cos(spot.angle * (1 - spot.penumbra)),
      );
    });
  }
}

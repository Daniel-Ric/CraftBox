import * as THREE from "three";
import type { CloudClearance } from "./cloud-clearance";
import { FullScreenQuad, Pass } from "three/addons/postprocessing/Pass.js";
import {
  normalizeAtmosphere,
  atmosphereHeight,
  cloudTypes,
  fogTypes,
} from "./atmosphere-settings";
import type { RenderSettings } from "./render-settings";
import { VolumeLighting, localVolumeShader } from "./volume-lighting";
import { authoredCloudField } from "./cloud-layer-shader";
import { prepareCloudLayers, cloudRenderBudget } from "./cloud-render-config";



function volumeNoise() {
  const size = 64,
    data = new Uint8Array(size * size * size * 4);
  const hash = (x: number, y: number, z: number) => {
    let n =
      Math.imul(x, 374761393) ^
      Math.imul(y, 668265263) ^
      Math.imul(z, 2147483647);
    n = Math.imul(n ^ (n >>> 13), 1274126177);
    return ((n ^ (n >>> 16)) >>> 0) / 4294967295;
  };
  const noise = (x: number, y: number, z: number, grid: number) => {
    const ix = Math.floor(x),
      iy = Math.floor(y),
      iz = Math.floor(z);
    const smooth = (t: number) => t * t * (3 - 2 * t),
      fx = smooth(x - ix),
      fy = smooth(y - iy),
      fz = smooth(z - iz);
    let v = 0;
    for (let dz = 0; dz < 2; dz++)
      for (let dy = 0; dy < 2; dy++)
        for (let dx = 0; dx < 2; dx++)
          v +=
            hash((ix + dx) % grid, (iy + dy) % grid, (iz + dz) % grid) *
            (dx ? fx : 1 - fx) *
            (dy ? fy : 1 - fy) *
            (dz ? fz : 1 - fz);
    return v;
  };
  for (let z = 0; z < size; z++)
    for (let y = 0; y < size; y++)
      for (let x = 0; x < size; x++) {
        let n = 0;
        for (let octave = 0; octave < 4; octave++) {
          const grid = 4 << octave;
          n +=
            noise(
              (x / size) * grid,
              (y / size) * grid,
              (z / size) * grid,
              grid,
            ) * Math.pow(0.5, octave);
        }
        const wx = (x / size) * 6,
          wy = (y / size) * 6,
          wz = (z / size) * 6,
          ix = Math.floor(wx),
          iy = Math.floor(wy),
          iz = Math.floor(wz);
        let nearest = 10;
        for (let dz = -1; dz <= 1; dz++)
          for (let dy = -1; dy <= 1; dy++)
            for (let dx = -1; dx <= 1; dx++) {
              const cx = ix + dx,
                cy = iy + dy,
                cz = iz + dz,
                hx = (cx + 6) % 6,
                hy = (cy + 6) % 6,
                hz = (cz + 6) % 6;
              const px = cx + hash(hx, hy, hz),
                py = cy + hash(hx + 31, hy + 17, hz + 13),
                pz = cz + hash(hx + 71, hy + 29, hz + 47);
              nearest = Math.min(
                nearest,
                Math.hypot(px - wx, py - wy, pz - wz),
              );
            }
        const at = (x + size * (y + size * z)) * 4;
        data[at] = Math.round((n / 1.875) * 255);
        data[at + 1] = Math.round(Math.max(0, 1 - nearest / 1.25) * 255);
        data[at + 2] = Math.round(
          noise((x / size) * 16, (y / size) * 16, (z / size) * 16, 16) * 255,
        );
        data[at + 3] = 255;
      }
  const texture = new THREE.Data3DTexture(data, size, size, size);
  texture.format = THREE.RGBAFormat;
  texture.type = THREE.UnsignedByteType;
  texture.minFilter = THREE.LinearMipmapLinearFilter;
  texture.magFilter = THREE.LinearFilter;
  texture.generateMipmaps = true;
  texture.wrapS = texture.wrapT = texture.wrapR = THREE.RepeatWrapping;
  texture.unpackAlignment = 1;
  texture.needsUpdate = true;
  return texture;
}

const cloudField = `
precision highp sampler3D;
uniform sampler3D aNoise;
uniform sampler2D aClearance;
uniform vec4 aClearanceRegion,aAvoidance;
uniform vec4 aCloud;
uniform vec4 aWeather;
uniform vec2 aWind;
uniform vec3 aStructure;
uniform vec4 aSeed,aDistribution;
uniform vec4 aRegion;
float cloudLod=0.;
${authoredCloudField}
float weatherHash(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453);}
float weatherNoise(vec2 p){vec2 i=floor(p),f=fract(p);f=f*f*(3.-2.*f);return mix(mix(weatherHash(i),weatherHash(i+vec2(1.,0.)),f.x),mix(weatherHash(i+vec2(0.,1.)),weatherHash(i+1.),f.x),f.y);}
bool puffyCloud(){return aRegion.w<1.5&&(aWeather.z<.5||(aWeather.z>1.5&&aWeather.z<2.5)||(aWeather.z>3.5&&aWeather.z<5.5)||aWeather.z>6.5);}
bool finiteCloud(){return puffyCloud()||aRegion.w>2.5;}
float cloudBody(vec3 p){
  bool bank=aRegion.w>2.5;
  float size=aCloud.z;
  if(!bank&&aWeather.z>4.5&&aWeather.z<5.5)size*=.58;
  float spacing=bank?1.15:1.4;
  vec2 cellPos=(p.xz-aWind)/ (size*spacing)+aSeed.xy,cell=floor(cellPos);
  float body=0.;
  for(int z=-1;z<=1;z++)for(int x=-1;x<=1;x++){
    vec2 id=cell+vec2(float(x),float(z));
    float r=weatherHash(id+vec2(13.7,39.1)),r2=weatherHash(id+vec2(71.3,4.8)),r3=weatherHash(id+vec2(7.9,85.2));
    float clustered=aCloud.w+(weatherNoise(id*.29+19.)-.5)*aDistribution.x*.65;
    float presence=smoothstep(r-.12,r+.12,clustered);
    if(presence<.001)continue;
    vec2 center=id+(bank?vec2(.4+.2*r2,.4+.2*r3):vec2(.2+.6*r2,.2+.6*r3));
    vec2 delta=(cellPos-center)*size*spacing;
    float ry=aCloud.y*(.24+.15*r),rx=size*(.32+.16*r2),rz=size*(.32+.16*r3);
    if(!bank&&aWeather.z>3.5&&aWeather.z<4.5){rx*=1.25;rz*=1.25;ry*=.75;}
    float cy=aCloud.x+aCloud.y*(.35+.24*r2);
    if(!bank&&aWeather.z>1.5&&aWeather.z<2.5){ry=aCloud.y*(.35+.08*r);cy=aCloud.x+aCloud.y*(.46+.08*r2);rx*=.92;rz*=.92;}
    if(bank){rx=size*(.5+.12*r2);rz=size*(.5+.12*r3);ry=aCloud.y*(.18+.12*r);cy=aCloud.x+aCloud.y*(.3+.22*r2);}
    if(bank&&(abs(delta.x)>size*1.12||abs(delta.y)>size*1.12))continue;
    vec3 d=vec3(delta.x,(p.y-cy),delta.y);
    vec3 noisePos=(p-vec3(aWind.x,0.,aWind.y)+vec3(aSeed.x,31.,aSeed.y))/size;
    if(aRegion.w<.5||bank){


      vec3 warp=vec3(textureLod(aNoise,noisePos*.36,cloudLod).r,textureLod(aNoise,noisePos*.36+vec3(.37,.11,.71),cloudLod).r,textureLod(aNoise,noisePos*.36+vec3(.13,.67,.29),cloudLod).r)-.5;
      d+=warp*vec3(rx,ry,rz)*mix(.5,1.5,aStructure.z);
    }
    if(abs(d.x)>rx*1.7||abs(d.z)>rz*1.7)continue;
    float radius=length(d/vec3(rx,ry,rz));
    float crown=length((d-vec3(rx*.45,ry*.34,rz*.12))/vec3(rx*.68,ry*.78,rz*.72));
    float shoulder=length((d+vec3(rx*.48,ry*.05,rz*.3))/vec3(rx*.73,ry*.68,rz*.68));
    float turret=length((d-vec3(-rx*.18,ry*.52,rz*.35))/vec3(rx*.48,ry*.65,rz*.48));
    float flank=length((d-vec3(rx*.55,-ry*.15,-rz*.32))/vec3(rx*.52,ry*.52,rz*.55));
    if(aWeather.z>6.5)crown=length((d-vec3(0.,ry*.55,0.))/vec3(rx*1.25,ry*.5,rz*1.25));
    float surface=min(min(radius,min(crown,shoulder)),min(turret,flank));
    float coarse=textureLod(aNoise,noisePos*.43,cloudLod).g;
    float fine=textureLod(aNoise,noisePos*mix(.8,2.5,aStructure.x),cloudLod).b;
    if(aRegion.w<.5||bank){
      float billows=textureLod(aNoise,noisePos*.95+vec3(.21,.39,.67),cloudLod).g;
      surface+=(coarse-.5)*.95+(billows-.5)*.55+(fine-.5)*aStructure.y*.7;
    }else surface+=(coarse-.5)*mix(.2,.5,aStructure.z)+(fine-.5)*aStructure.y*.28;
    float feather=mix(.27,.12,aDistribution.y);
    float density=1.-smoothstep(.65-feather,1.+feather,surface);
    body=bank?1.-(1.-body)*(1.-density*presence):max(body,density*presence);
  }
  return body;
}
float rawCloudShape(vec3 p) {
  if(aWeather.y<.5 || aWeather.x<=0. || aCloud.w<=0.)return 0.;
  float h=(p.y-aCloud.x)/aCloud.y;
  if(h<=0. || h>=1.)return 0.;
  if(aRegion.w>2.5){float edge=1.-smoothstep(aRegion.z*.65,aRegion.z,length(p.xz-aRegion.xy));return cloudBody(p)*aWeather.x*edge*smoothstep(0.,.06,h)*(1.-smoothstep(.94,1.,h));}
  if(puffyCloud()){
    float edge=1.-smoothstep(aRegion.z*.65,aRegion.z,length(p.xz-aRegion.xy));
    return cloudBody(p)*aWeather.x*edge*smoothstep(0.,.05,h)*(1.-smoothstep(.95,1.,h));
  }
  vec2 xz=(p.xz-aWind)/aCloud.z+aSeed.xy;
  float angle=aSeed.x*.037,c=cos(angle),s=sin(angle);xz=mat2(c,-s,s,c)*xz;
  float weather=weatherNoise(xz*.34)*.65+weatherNoise(xz*.79+17.)*.35;
  vec3 q=vec3(xz.x*.25,h*.4,xz.y*.25);
  if(aWeather.z>2.5&&aWeather.z<3.5)q*=vec3(.18,1.4,1.3);
  if(aWeather.z>4.5&&aWeather.z<5.5)q*=vec3(1.8,1.2,1.8);
  if(aWeather.z>3.5&&aWeather.z<4.5)q*=vec3(.7,.65,.7);
  if(aWeather.z>6.5)q.xz*=mix(1.4,.45,smoothstep(.4,.9,h));
  if(aWeather.z>5.5&&aWeather.z<6.5)q*=vec3(.5,1.,1.4);
  float warp=textureLod(aNoise,q*.31+vec3(.2,.7,.3),cloudLod).r-.5;
  q+=vec3(warp,-warp*.8,warp*.7)*aStructure.z;
  vec2 base=textureLod(aNoise,q*.71,cloudLod).rg,secondary=textureLod(aNoise,q*1.37+vec3(.14,.3,.8),cloudLod).rg;
  float rounded=mix(base.r,base.g,aDistribution.y*.7)*.7+secondary.g*.3;
  float threshold=mix(.79,.18,aCloud.w)+(weather-.5)*aDistribution.x*.65;
  float shape=smoothstep(threshold-.04,threshold+.19,rounded);
  float bottom=.035+weatherNoise(xz*.67+31.)*.17,top=.65+weatherNoise(xz*.53-11.)*.33;
  float vertical=smoothstep(bottom,bottom+.2,h)*(1.-smoothstep(top-.28,top,h));
  if(aWeather.z>.5&&aWeather.z<1.5){shape=mix(.35,1.,shape)*smoothstep(0.,.2,aCloud.w);vertical=smoothstep(.02,.22,h)*(1.-smoothstep(.68,.98,h));}
  if((aWeather.z>1.5&&aWeather.z<2.5)||aWeather.z>6.5){vertical=smoothstep(bottom,bottom+.1,h)*(1.-smoothstep(.75,1.,h));shape=pow(shape,.7);}
  if(aWeather.z>2.5&&aWeather.z<3.5){shape=pow(shape,1.7);vertical*=.4;}
  if(aWeather.z>5.5&&aWeather.z<6.5){vertical*=smoothstep(.05,.4,h)*(1.-smoothstep(.55,.9,h));shape=pow(shape,.7);}
  float detail=textureLod(aNoise,q*mix(1.4,3.9,aStructure.x)+vec3(.23,.53,.81),cloudLod).b;
  return max(0.,shape*vertical-(1.-detail)*aStructure.y*.24)*aWeather.x;
}
float cloudShape(vec3 p) {
  return rawCloudShape(p)*cloudMask(p);
}
float cloudMask(vec3 p) {
  float mask=openingMask(p);
  if(aAvoidance.x<.5||aAvoidance.y<=0.)return mask;
  vec2 uv=(p.xz-aClearanceRegion.xy)/aClearanceRegion.z;
  if(any(lessThan(uv,vec2(0.)))||any(greaterThan(uv,vec2(1.))))return mask;
  float distance=textureLod(aClearance,uv,0.).r*aAvoidance.w;
  float influence=(1.-smoothstep(0.,aAvoidance.z,distance))*smoothstep(aClearanceRegion.w-aAvoidance.z*.7,aClearanceRegion.w+aAvoidance.z*.7,p.y);
  return mask*(1.-influence*aAvoidance.y);
}
vec4 cloudMedium(vec3 p){
  float base=cloudShape(p)*6./aCloud.y;
  return vec4(cloudBaseColor*base,base)+authoredMedium(p,false);
}
float cloudExtinction(vec3 p){return cloudMedium(p).a;}
float cloudGroundExtinction(vec3 p){return cloudShape(p)*6./aCloud.y*aWeather.w+authoredMedium(p,true).a;
}
`;

const vertex = `out vec2 vUv;void main(){vUv=uv;gl_Position=vec4(position.xy,0.,1.);}`;
export class AtmospherePass extends Pass {
  private clearance?: CloudClearance;
  private emptyClearance = new THREE.DataTexture(
    new Uint8Array([255]),
    1,
    1,
    THREE.RedFormat,
  );
  readonly localLighting = new VolumeLighting();
  readonly noise = volumeNoise();
  readonly uniforms = {
    ...this.localLighting.uniforms,
    aNoise: { value: this.noise },
    aClearance: { value: this.emptyClearance },
    aClearanceRegion: { value: new THREE.Vector4() },
    aAvoidance: { value: new THREE.Vector4() },
    aCloud: { value: new THREE.Vector4() },
    aWeather: { value: new THREE.Vector4() },
    aWind: { value: new THREE.Vector2() },
    aStructure: { value: new THREE.Vector3() },
    aSeed: { value: new THREE.Vector4() },
    aDistribution: { value: new THREE.Vector4() },
    aRegion: { value: new THREE.Vector4() },
    aLayerCount: { value: 0 },
    aLayerTransform: {
      value: Array.from({ length: 8 }, () => new THREE.Matrix4()),
    },
    aLayerMin: { value: Array.from({ length: 8 }, () => new THREE.Vector3()) },
    aLayerMax: { value: Array.from({ length: 8 }, () => new THREE.Vector3()) },
    aLayerParameters: {
      value: Array.from({ length: 8 }, () => new THREE.Vector4()),
    },
    aLayerForm: { value: Array.from({ length: 8 }, () => new THREE.Vector4()) },
    aLayerStyle: {
      value: Array.from({ length: 8 }, () => new THREE.Vector4()),
    },
    aLayerTint: { value: Array.from({ length: 8 }, () => new THREE.Vector4()) },
    aLayerScale: {
      value: Array.from({ length: 8 }, () => new THREE.Vector4()),
    },
    aOpening: { value: new THREE.Vector4() },
    aOpeningShape: { value: new THREE.Vector4() },
    aScene: { value: new THREE.Vector4(0, 0, 0, 10) },
    cloudBaseColor: { value: new THREE.Color() },
    cloudLighting: { value: new THREE.Vector2(0.45, 0.35) },
    shadowSteps: { value: 6 },
    cameraView: { value: new THREE.Matrix4() },
    absorption: { value: 1 },
    sceneDepth: { value: null as THREE.Texture | null },
    inverseProjection: { value: new THREE.Matrix4() },
    cameraWorld: { value: new THREE.Matrix4() },
    fogColor: { value: new THREE.Color() },
    cloudColor: { value: new THREE.Color() },
    sunColor: { value: new THREE.Color() },
    sunDirection: { value: new THREE.Vector3(1, 1, 1).normalize() },
    fog: { value: new THREE.Vector4() },
    fogDetails: { value: new THREE.Vector4() },
    fogRegion: { value: new THREE.Vector3() },
    fogVariation: { value: 0 },
    maxDistance: { value: 2000 },
    steps: { value: 64 },
    ambient: { value: 0.6 },
    sunPower: { value: 1 },
  };
  private target = new THREE.WebGLRenderTarget(1, 1, {
    type: THREE.HalfFloatType,
    depthBuffer: false,
    count: 2,
  });
  private material = new THREE.ShaderMaterial({
    defines: { VOLUME_POINTS: 0, VOLUME_SPOT: 0 },
    glslVersion: THREE.GLSL3,
    depthTest: false,
    depthWrite: false,
    uniforms: this.uniforms,
    vertexShader: vertex,
    fragmentShader:
      cloudField +
      localVolumeShader +
      `
    uniform sampler2D sceneDepth;
    uniform mat4 inverseProjection,cameraWorld;
    uniform vec3 fogColor,cloudColor,sunColor,sunDirection;
    uniform vec4 fog;
    uniform vec4 fogDetails;
    uniform vec3 fogRegion;
    uniform float fogVariation,maxDistance,ambient,sunPower,absorption;
    uniform int steps,shadowSteps;
    uniform vec2 cloudLighting;
    uniform mat4 cameraView;
    in vec2 vUv;layout(location=0) out vec4 volumeOut;layout(location=1) out vec4 cloudDepthOut;
    vec3 unproject(float depth){vec4 p=inverseProjection*vec4(vUv*2.-1.,depth*2.-1.,1.);return (cameraWorld*vec4(p.xyz/p.w,1.)).xyz;}
    vec2 slab(vec3 o,vec3 d,float lo,float hi,float limit){
      if(abs(d.y)<.00001)return o.y>=lo&&o.y<=hi?vec2(0.,limit):vec2(limit,0.);
      vec2 t=(vec2(lo,hi)-o.y)/d.y;return vec2(max(0.,min(t.x,t.y)),min(limit,max(t.x,t.y)));
    }
    vec2 regionRay(vec3 o,vec3 d,float loY,float hiY,float limit,vec3 region){
      vec3 lo=vec3(region.x-region.z,loY,region.y-region.z),hi=vec3(region.x+region.z,hiY,region.y+region.z);
      vec3 safeDir=vec3(abs(d.x)<.00001?.00001:d.x,abs(d.y)<.00001?.00001:d.y,abs(d.z)<.00001?.00001:d.z);
      vec3 t0=(lo-o)/safeDir,t1=(hi-o)/safeDir,entry=min(t0,t1),exitT=max(t0,t1);
      return vec2(max(0.,max(entry.x,max(entry.y,entry.z))),min(limit,min(exitT.x,min(exitT.y,exitT.z))));
    }
    vec2 layerRay(vec3 o,vec3 d,vec3 lo,vec3 hi,float limit){
      float entry=0.,exitT=limit;
      for(int k=0;k<3;k++){
        if(abs(d[k])<.00001){if(o[k]<lo[k]||o[k]>hi[k])return vec2(limit);}
        else{float a=(lo[k]-o[k])/d[k],b=(hi[k]-o[k])/d[k];entry=max(entry,min(a,b));exitT=min(exitT,max(a,b));}
      }
      return exitT>entry?vec2(entry,exitT):vec2(limit);
    }
    void main(){
      float depth=texture(sceneDepth,vUv).r;
      vec3 o=unproject(0.),farPoint=unproject(1.);
      vec3 d=normalize(farPoint-o);
      float limit=depth<.999999?min(distance(unproject(depth),o),maxDistance):maxDistance;
      vec2 cf=aWeather.y>.5?slab(o,d,aCloud.x,aCloud.x+aCloud.y,limit):vec2(limit,0.);
      if(aWeather.y>.5&&finiteCloud())cf=regionRay(o,d,aCloud.x,aCloud.x+aCloud.y,limit,aRegion.xyz);
      bool bankFog=aDistribution.z>.5&&aDistribution.z<2.5;
      vec2 ff=fog.w>.5?(bankFog?regionRay(o,d,fog.y-fog.z*4.,fog.y+fog.z*5.,limit,fogRegion):slab(o,d,-12000.,fog.y+fog.z*5.,limit)):vec2(limit,0.);
      if(cf.y<=cf.x)cf=vec2(limit);
      if(ff.y<=ff.x)ff=vec2(limit);
      float bounds[20];bounds[0]=cf.x;bounds[1]=cf.y;bounds[2]=ff.x;bounds[3]=ff.y;
      vec2 intervals[8];
      for(int i=0;i<8;i++){intervals[i]=i<aLayerCount?layerRay(o,d,aLayerMin[i],aLayerMax[i],limit):vec2(limit);bounds[4+i*2]=intervals[i].x;bounds[5+i*2]=intervals[i].y;}
      for(int a=0;a<19;a++)for(int b=0;b<19;b++)if(bounds[b]>bounds[b+1]){float tmp=bounds[b];bounds[b]=bounds[b+1];bounds[b+1]=tmp;}
      float occupied=0.,cloudOccupied=0.;
      for(int k=0;k<19;k++){
        float mid=(bounds[k]+bounds[k+1])*.5;
        bool cloudSegment=mid>=cf.x&&mid<=cf.y;
        for(int i=0;i<8;i++)if(i<aLayerCount&&mid>=intervals[i].x&&mid<=intervals[i].y)cloudSegment=true;
        if(cloudSegment)cloudOccupied+=bounds[k+1]-bounds[k];
        if(cloudSegment||(mid>=ff.x&&mid<=ff.y))occupied+=bounds[k+1]-bounds[k];
      }
      vec3 scattering=vec3(0.);float transmittance=1.,cloudMoment=0.,cloudWeight=0.;
      float mu=dot(d,sunDirection);
      float phase=.45+.55*pow(max(0.,mu),8.);


      for(int segment=0;segment<19;segment++){
      float start=bounds[segment],end=bounds[segment+1],mid=(start+end)*.5;
      if(end-start<.0001)continue;
      bool inCloud=mid>=cf.x&&mid<=cf.y&&cf.y>cf.x;
      for(int k=0;k<8;k++)if(k<aLayerCount&&mid>=intervals[k].x&&mid<=intervals[k].y&&intervals[k].y>intervals[k].x)inCloud=true;
      bool inFog=mid>=ff.x&&mid<=ff.y&&ff.y>ff.x;
      if(!inCloud&&!inFog)continue;
      int segmentSteps=max(inCloud?12:8,int(float(steps)*(end-start)/max(.001,inCloud?cloudOccupied:occupied)));
      for(int i=0;i<192;i++){
        if(i>=segmentSteps||transmittance<.008)break;
        float u0=float(i)/float(segmentSteps),u1=float(i+1)/float(segmentSteps);
        float ds=(end-start)*(u1*u1-u0*u0);
        float t=start+(end-start)*(u0*u0+u1*u1)*.5;
        cloudLod=max(0.,log2(max(1.,ds/aCloud.z*12.)));
        vec3 p=o+d*t;
        float fade=1.-smoothstep(maxDistance*.55,maxDistance,t);
        float outsideDistance=max(0.,max(aCloud.x-o.y,o.y-aCloud.x-aCloud.y));
        float outsideBlend=smoothstep(0.,max(2.,aCloud.y*.3),outsideDistance);
        float horizon=aRegion.w>2.5?1.:mix(1.,pow(smoothstep(0.,.12,abs(d.y)),3.),outsideBlend);
        vec4 medium=cloudMedium(p);
        float cd=medium.a*fade*(aLayerCount>0?1.:horizon);
        vec3 tint=medium.a>.000001?medium.rgb/medium.a:cloudColor;
        float fd=0.;
        if(fog.w>.5){vec3 fq=(p-vec3(aWind.x,0.,aWind.y))/fogDetails.x+vec3(aSeed.z,0.,aSeed.w);
          float warp=texture(aNoise,fq*.37).g-.5;fq+=warp*aDistribution.w;
          float n=texture(aNoise,fq*.73).r*.6+texture(aNoise,fq*1.91+17.).g*.4;
          float localHeight=fog.y+(weatherNoise(fq.xz*.7)-.5)*fog.z*aDistribution.w*2.;
          float modulation=mix(1.,smoothstep(.25,.7,n)*1.8,fogVariation);
          if(aDistribution.z>.5&&aDistribution.z<1.5)modulation=mix(.03,pow(max(0.,n*1.8),2.),fogVariation);
          if(aDistribution.z>1.5&&aDistribution.z<2.5)modulation*=.6+.4*sin(p.y/fog.z*5.+n*3.);
          if(aDistribution.z>2.5&&aDistribution.z<3.5)modulation=1.;
          if(aDistribution.z>3.5)modulation*=smoothstep(.25,.7,texture(aNoise,fq*vec3(1.,.18,1.)).g)*2.;
          float heightFalloff=exp(-(bankFog?abs(p.y-localHeight):max(0.,p.y-localHeight))/fog.z);
          float regionFade=bankFog?1.-smoothstep(fogRegion.z*.45,fogRegion.z,length(p.xz-fogRegion.xy)):1.;
          fd=fog.x/180.*heightFalloff*regionFade*modulation*fade*smoothstep(fogDetails.y,fogDetails.y+max(1.,fogDetails.y*.2),t);}
        float density=cd+fd;
        if(density<.000001)continue;
        float shadow=0.;
        if(cd>0.)for(int j=0;j<8;j++){
          if(j>=shadowSteps)break;
          float u=float(j)/float(shadowSteps),v=float(j+1)/float(shadowSteps),reach=max(max(aCloud.y,aCloud.z*.75),aLayerCount>0?aScene.w:0.);
          float ds=(v*v-u*u)*reach;
          shadow+=cloudExtinction(p+sunDirection*(u*u+v*v)*.5*reach)*ds;
        }
        float transmission=exp(-shadow*absorption);
        vec3 skyFill=vec3(.56,.69,.9)*(.14+ambient*.22);
        float rim=cloudLighting.x*pow(max(0.,mu),12.)*(1.-transmission)*exp(-shadow*absorption*.25);
        float multiple=cloudLighting.y*(1.-exp(-shadow*.5))*exp(-shadow*absorption*.12);
        vec3 cloudLight=tint*(skyFill+sunColor*sunPower*(transmission*(.55+phase*.65)+.16*exp(-shadow*absorption*.32)+rim+multiple));
        float g=fogDetails.w;
        float fogPhase=(1.-g*g)/pow(max(.05,1.+g*g-2.*g*mu),1.5);
        vec3 fogLight=fogColor*(vec3(.48+ambient*.24)+sunColor*sunPower*min(4.,fogPhase)*fogDetails.z*.3);
        vec3 light=(cloudLight*cd+fogLight*fd)/density;
        light+=localIllumination(p,d)*mix(tint,fogColor,fd/density);
        float extinction=exp(-density*ds);
        float weight=transmittance*(1.-extinction)*cd/density;
        cloudMoment+=max(.001,-(cameraView*vec4(p,1.)).z)*weight;cloudWeight+=weight;
        scattering+=transmittance*(1.-extinction)*light;
        transmittance*=extinction;
      }
      }
      volumeOut=vec4(scattering,transmittance);
      cloudDepthOut=vec4(cloudWeight>.00001?cloudMoment/cloudWeight:0.,cloudWeight,0.,1.);
    }`,
  });
  private composite = new THREE.ShaderMaterial({
    depthTest: false,
    depthWrite: false,
    uniforms: {
      beauty: { value: null as THREE.Texture | null },
      volume: { value: this.target.texture },
      sceneDepth: { value: null as THREE.Texture | null },
      texel: { value: new THREE.Vector2() },
    },
    vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=vec4(position.xy,0.,1.);}`,
    fragmentShader: `uniform sampler2D beauty,volume,sceneDepth;uniform vec2 texel;varying vec2 vUv;
    void main(){vec4 base=texture2D(beauty,vUv);float depth=texture2D(sceneDepth,vUv).r;vec4 sum=vec4(0.);float weights=0.;
      for(int y=0;y<2;y++)for(int x=0;x<2;x++){vec2 sampleUv=(floor(vUv/texel-.5)+vec2(float(x),float(y))+.5)*texel;
        vec2 f=fract(vUv/texel-.5);vec2 w=mix(1.-f,f,vec2(float(x),float(y)));
        float weight=w.x*w.y*exp(-abs(texture2D(sceneDepth,sampleUv).r-depth)*12000.)+.00001;
        sum+=texture2D(volume,sampleUv)*weight;weights+=weight;}
      vec4 v=sum/weights;gl_FragColor=vec4(base.rgb*v.a+v.rgb,1.-(1.-base.a)*v.a);}`,
  });
  private quad = new FullScreenQuad(this.material);
  private width = 1;
  private height = 1;
  private resolution = 0.5;
  constructor() {
    super();
    this.enabled = false;
  }
  configure(
    settings: RenderSettings,
    camera: THREE.Camera,
    depth: THREE.Texture,
    light: THREE.DirectionalLight,
    time: number,
    span: number,
    bounds?: THREE.Box3,
    context: { exporting?: boolean; timeline?: boolean } = {},
  ) {
    const s = normalizeAtmosphere(settings.atmosphere),
      u = this.uniforms;
    u.aAvoidance.value.set(
      this.clearance ? 1 : 0,
      s.cloudAvoidance,
      s.cloudClearance,
      this.clearance?.maxDistance ?? 1,
    );
    this.enabled =
      (s.fog && s.fogDensity > 0) ||
      (s.clouds && s.cloudDensity > 0 && s.coverage > 0) ||
      s.cloudLayers.some((l) => l.enabled);
    const heights =
      bounds && !bounds.isEmpty()
        ? { min: bounds.min.y, max: bounds.max.y }
        : undefined;
    const regionCenter =
      bounds && !bounds.isEmpty()
        ? bounds.getCenter(new THREE.Vector3())
        : new THREE.Vector3();
    const regionSize =
      bounds && !bounds.isEmpty()
        ? bounds.getSize(new THREE.Vector3())
        : new THREE.Vector3(span, 0, span);
    u.aRegion.value.set(
      regionCenter.x,
      regionCenter.z,
      Math.max(
        s.cloudSize * 1.5,
        Math.hypot(regionSize.x, regionSize.z) * 0.5 * s.cloudExtent,
      ),
      s.cloudModel === "banks"
        ? 3
        : s.cloudModel === "classic"
          ? 2
          : s.cloudModel === "bodies"
            ? 1
            : 0,
    );
    u.fogRegion.value.set(
      regionCenter.x,
      regionCenter.z,
      Math.max(s.fogSize * 1.5, Math.hypot(regionSize.x, regionSize.z) * 1.2),
    );
    u.aCloud.value.set(
      atmosphereHeight(s.cloudHeight, s.cloudReference, heights),
      s.cloudThickness,
      s.cloudSize,
      s.coverage,
    );
    u.aWeather.value.set(
      s.cloudDensity,
      s.clouds && s.cloudDensity > 0 && s.coverage > 0 ? 1 : 0,
      Object.keys(cloudTypes).indexOf(s.cloudType),
      s.cloudShadows,
    );
    const seconds = s.time + (s.animate ? time : 0),
      angle = (s.windDirection * Math.PI) / 180;
    u.aWind.value.set(
      Math.sin(angle) * seconds * s.windSpeed,
      Math.cos(angle) * seconds * s.windSpeed,
    );
    u.aStructure.value.set(s.detail, s.erosion, s.turbulence);
    const box = bounds ?? new THREE.Box3(),
      sceneRadius = box.isEmpty()
        ? 10
        : Math.max(2, box.getBoundingSphere(new THREE.Sphere()).radius);
    const animationTime = s.time + (context.timeline || s.animate ? time : 0);
    const layers = prepareCloudLayers(s.cloudLayers, box, animationTime);
    u.aLayerCount.value = layers.length;
    layers.forEach((p, i) => {
      const l = p.layer,
        angle = (l.windDirection * Math.PI) / 180;
      u.aLayerTransform.value[i].copy(p.worldToLocal);
      u.aLayerMin.value[i].copy(p.bounds.min);
      u.aLayerMax.value[i].copy(p.bounds.max);
      u.aLayerParameters.value[i].set(
        l.density,
        l.coverage,
        l.dissolve ?? 0,
        l.kind === "volume" ? 1 : 0,
      );
      u.aLayerForm.value[i].set(l.detail, l.erosion, l.turbulence, l.seed);
      u.aLayerStyle.value[i].set(l.softness, l.shadows, 0, 0);
      const color = new THREE.Color(l.color);
      u.aLayerTint.value[i].set(color.r, color.g, color.b, 1);
      u.aLayerScale.value[i].set(
        6 / p.height,
        (Math.sin(angle) * seconds * l.windSpeed) / Math.max(1, p.radius),
        (Math.cos(angle) * seconds * l.windSpeed) / Math.max(1, p.radius),
        0,
      );
    });
    const opening = s.cloudOpening;
    u.aOpening.value.set(
      opening.center[0],
      opening.center[1],
      opening.floor,
      opening.strength,
    );
    u.aOpeningShape.value.set(
      opening.radius[0],
      opening.radius[1],
      opening.softness,
      opening.irregularity,
    );
    u.aScene.value.set(
      regionCenter.x,
      regionCenter.y,
      regionCenter.z,
      sceneRadius,
    );
    u.cloudBaseColor.value.set(s.cloudColor);
    u.cloudLighting.value.set(s.cloudSilverLining, s.cloudMultipleScattering);
    const seed = (n: number) => {
      const x = Math.sin(n * 12.9898) * 43758.5453,
        y = Math.sin(n * 78.233 + 31) * 19642.349;
      return [(x - Math.floor(x)) * 500, (y - Math.floor(y)) * 500];
    };
    const cs = seed(s.cloudSeed),
      fs = seed(s.fogSeed);
    u.aSeed.value.set(cs[0], cs[1], fs[0], fs[1]);
    u.aDistribution.value.set(
      s.cloudClumping,
      s.cloudRoundness,
      Object.keys(fogTypes).indexOf(s.fogType),
      s.fogTurbulence,
    );
    u.absorption.value = s.lightAbsorption;
    u.fog.value.set(
      s.fogDensity,
      atmosphereHeight(s.fogHeight, s.fogReference, heights),
      s.fogFalloff,
      s.fog && s.fogDensity > 0 ? 1 : 0,
    );
    u.fogVariation.value = s.fogVariation;
    u.fogDetails.value.set(
      s.fogSize,
      s.fogStart,
      s.fogScattering,
      s.fogAnisotropy,
    );
    u.fogColor.value.set(s.fogColor);
    u.cloudColor.value.set(s.cloudColor);
    u.sunColor.value.copy(light.color);
    u.sunDirection.value
      .copy(light.position)
      .sub(light.target.position)
      .normalize();
    u.sunPower.value = Math.min(2, light.intensity / 3);
    u.ambient.value = Math.min(2, settings.ambient);
    u.maxDistance.value = s.distance;
    const budget = cloudRenderBudget(s, settings.quality, !!context.exporting);
    u.steps.value = budget.steps;
    u.shadowSteps.value = budget.shadowSteps;
    camera.updateWorldMatrix(true, false);
    u.inverseProjection.value.copy(camera.projectionMatrixInverse);
    u.cameraWorld.value.copy(camera.matrixWorld);
    u.cameraView.value.copy(camera.matrixWorldInverse);
    u.sceneDepth.value = depth;
    this.composite.uniforms.sceneDepth.value = depth;
    const resolution = budget.resolution;
    if (resolution !== this.resolution) {
      this.resolution = resolution;
      this.setSize(this.width, this.height);
    }
  }
  installCloudShadows(material: THREE.Material) {
    if (!(material instanceof THREE.MeshStandardMaterial)) return;
    const original = material.onBeforeCompile,
      cache = material.customProgramCacheKey();
    material.onBeforeCompile = (shader, renderer) => {
      original.call(material, shader, renderer);
      for (const key of [
        "aNoise",
        "aClearance",
        "aClearanceRegion",
        "aAvoidance",
        "aCloud",
        "aWeather",
        "aWind",
        "aStructure",
        "aSeed",
        "aDistribution",
        "aRegion",
        "aLayerCount",
        "aLayerTransform",
        "aLayerMin",
        "aLayerMax",
        "aLayerParameters",
        "aLayerForm",
        "aLayerStyle",
        "aLayerTint",
        "aLayerScale",
        "aOpening",
        "aOpeningShape",
        "aScene",
        "cloudBaseColor",
      ] as const)
        shader.uniforms[key] = this.uniforms[key];
      shader.vertexShader = shader.vertexShader
        .replace(
          "#include <common>",
          "#include <common>\nvarying vec3 atmosphereWorld;",
        )
        .replace(
          "#include <begin_vertex>",
          "#include <begin_vertex>\natmosphereWorld=(modelMatrix*vec4(transformed,1.)).xyz;",
        );
      shader.fragmentShader = shader.fragmentShader
        .replace(
          "#include <common>",
          "#include <common>\nvarying vec3 atmosphereWorld;\n" + cloudField,
        )
        .replace(
          "#include <lights_fragment_end>",
          `#include <lights_fragment_end>
        if((aWeather.y>.5&&aWeather.w>0.)||aLayerCount>0){
          vec3 sunDir=normalize(vec3(0.,1.,0.));
          #if NUM_DIR_LIGHTS > 0
          sunDir=inverseTransformDirection(directionalLights[0].direction,viewMatrix);
          #endif
          float cloudShadow=0.;
          if(sunDir.y>.01){
            if(aWeather.y>.5){float travel=(aCloud.x+aCloud.y*.4-atmosphereWorld.y)/sunDir.y;if(travel>0.)cloudShadow+=cloudShape(atmosphereWorld+sunDir*travel)*aWeather.w*2.;}
            for(int i=0;i<8;i++){if(i>=aLayerCount)break;float travel=((aLayerMin[i].y+aLayerMax[i].y)*.5-atmosphereWorld.y)/sunDir.y;if(travel>0.)cloudShadow+=layerDensity(i,atmosphereWorld+sunDir*travel)*(aLayerMax[i].y-aLayerMin[i].y)*aLayerStyle[i].y*cloudMask(atmosphereWorld+sunDir*travel);}
          }
          float lightTransmission=exp(-cloudShadow*2.);
          reflectedLight.directDiffuse*=lightTransmission;reflectedLight.directSpecular*=lightTransmission;
        }`,
        );
    };
    material.customProgramCacheKey = () => cache + "-cloud-shadows-v2";
  }
  setSize(width: number, height: number) {
    this.width = width;
    this.height = height;
    this.target.setSize(
      Math.max(1, Math.ceil(width * this.resolution)),
      Math.max(1, Math.ceil(height * this.resolution)),
    );
    this.composite.uniforms.texel.value.set(
      1 / this.target.width,
      1 / this.target.height,
    );
  }
  get cloudDepthTexture() {
    return this.target.textures[1];
  }
  setClearance(field?: CloudClearance) {
    this.clearance?.texture.dispose();
    this.clearance = field;
    this.emptyClearance.needsUpdate = true;
    this.uniforms.aClearance.value = field?.texture ?? this.emptyClearance;
    if (field)
      this.uniforms.aClearanceRegion.value.set(
        field.min.x,
        field.min.y,
        field.extent,
        field.bottom,
      );
  }
  render(
    renderer: THREE.WebGLRenderer,
    write: THREE.WebGLRenderTarget,
    read: THREE.WebGLRenderTarget,
  ) {
    this.quad.material = this.material;
    this.localLighting.sync(this.material);
    renderer.setRenderTarget(this.target);
    this.quad.render(renderer);
    this.composite.uniforms.beauty.value = read.texture;
    this.quad.material = this.composite;
    renderer.setRenderTarget(this.renderToScreen ? null : write);
    this.quad.render(renderer);
  }
  dispose() {
    this.clearance?.texture.dispose();
    this.emptyClearance.dispose();
    this.noise.dispose();
    this.target.dispose();
    this.material.dispose();
    this.composite.dispose();
    this.quad.dispose();
  }
}

import {
  Camera,
  OrthographicCamera,
  PerspectiveCamera,
  Texture,
  Vector2,
  Vector3,
  Matrix4,
  ShaderMaterial,
  WebGLRenderTarget,
  WebGLRenderer,
  HalfFloatType,
  NearestFilter,
  LinearFilter,
  LinearMipmapLinearFilter,
} from "three";
import { FullScreenQuad } from "three/addons/postprocessing/Pass.js";
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
  private height = 1080;
  private radius = 16;
  private nearX = new WebGLRenderTarget(1, 1, {
    type: HalfFloatType,
    minFilter: NearestFilter,
    magFilter: NearestFilter,
    depthBuffer: false,
  });
  private nearY = this.nearX.clone();
  private nearColor = new WebGLRenderTarget(1, 1, {
    type: HalfFloatType,
    minFilter: LinearMipmapLinearFilter,
    magFilter: LinearFilter,
    depthBuffer: false,
    generateMipmaps: true,
  });
  private farColor = this.nearColor.clone();
  private prefilter: ShaderMaterial;
  private prefilterQuad: FullScreenQuad;
  private dilation: ShaderMaterial;
  private dilationQuad: FullScreenQuad;
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
        nearStrength: { value: 2 },
        farStrength: { value: 1 },
        focusRange: { value: 8 },
        transition: { value: 80 },
        mode: { value: 1 },
        targetView: { value: new Vector3() },
        projectionInverse: { value: new Matrix4() },
        nearField: { value: null },
        dilationInput: { value: null },
        dilationAxis: { value: new Vector2() },
        dilationStage: { value: 0 },
        samples: { value: new Vector2(64, 32) },
        nearColor: { value: null },
        farColor: { value: null },
        prefilterLayer: { value: 0 },
      },
      vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
      fragmentShader: `${depthFunctions}
        uniform sampler2D nearField;
        uniform sampler2D nearColor,farColor;
        uniform vec2 samples;
        vec4 gatherColor(vec2 uv,float radius,float count,bool nearLayer){
          vec4 source=texture2D(tDiffuse,uv);
          // Filter each depth layer before sparse gathering to suppress repeated rail/texel patterns.
          float lod=max(0.0,log2(max(radius,1.0)/sqrt(count)*.886227));
          vec4 filtered=nearLayer?textureLod(nearColor,uv,lod):textureLod(farColor,uv,lod);
          vec3 color=filtered.a>.00001?filtered.rgb/filtered.a:source.rgb;
          return vec4(mix(source.rgb,color,smoothstep(1.0,3.0,radius))*source.a,source.a);
        }
        void main(){
          vec4 source=texture2D(tDiffuse,vUv);
          float z=depthAt(vUv),r=radiusAt(vUv,z);
          // Adaptive far gather: small circles use the full sample budget too.
          float farRadius=max(r,0.0);
          vec4 farSum=vec4(0.0);float farWeight=0.0;
          if(farRadius>.5){
            for(int i=0;i<64;i++){
              if(float(i)>=samples.y)break;
              float n=float(i)+.5,dist=sqrt(n/samples.y)*farRadius;
              vec2 uv=clamp(vUv+vec2(cos(n*2.39996323),sin(n*2.39996323))*dist*texel,texel*.5,1.0-texel*.5);
              float sz=depthAt(uv),sr=radiusAt(uv,sz);
              // Keep sharp and foreground silhouettes out of background bokeh.
              float w=step(0.0,sr)*smoothstep(dist-.75,dist+.75,max(sr,0.0));
              vec4 c=gatherColor(uv,farRadius,samples.y,false);
              farSum+=c*w;farWeight+=w;
            }
          }
          vec4 base=vec4(source.rgb*source.a,source.a);
          if(farWeight>.001)base=mix(base,farSum/farWeight,smoothstep(.5,1.5,farRadius));
          // Conservative near-radius dilation finds foreground spreading across edges.
          float searchRadius=max(-r,texture2D(nearField,vUv).r);
          vec4 nearSum=vec4(0.0);float nearWeight=0.0;
          vec4 backgroundSum=vec4(0.0);float backgroundWeight=0.0;
          if(searchRadius>.5){
            for(int i=0;i<96;i++){
              if(float(i)>=samples.x)break;
              float n=float(i)+.5,dist=sqrt(n/samples.x)*searchRadius;
              vec2 uv=vUv+vec2(cos(n*2.39996323),sin(n*2.39996323))*dist*texel;
              if(any(lessThan(uv,vec2(0.0)))||any(greaterThan(uv,vec2(1.0))))continue;
              float sr=-radiusAt(uv,depthAt(uv));
              if(sr<=.5){
                // Reconstruct the surface behind a defocused foreground edge.
                // Keeping the sharp foreground as its own backdrop leaves hard silhouettes.
                if(r<-.5){
                  float bw=1.0/max(dist*dist,1.0);
                  vec4 background=sr<-.5?gatherColor(uv,-sr,samples.x,false):texture2D(tDiffuse,uv);
                  if(sr>=-.5)background.rgb*=background.a;
                  backgroundSum+=background*bw;backgroundWeight+=bw;
                }
                continue;
              }
              float w=smoothstep(dist-.75,dist+.75,sr)/max(sr*sr,1.0);
              vec4 c=gatherColor(uv,searchRadius,samples.x,true);
              nearSum+=c*w;nearWeight+=w;
            }
            float coverage=clamp(nearWeight*searchRadius*searchRadius/samples.x,0.0,1.0);
            if(nearWeight>.000001){
              vec4 nearLayer=nearSum/nearWeight;
              if(r<-.5){
                if(backgroundWeight>.000001)base=backgroundSum/backgroundWeight;
                else {base=nearLayer;coverage=0.0;}
              }
              base=nearLayer*coverage+base*(1.0-nearLayer.a*coverage);
            }
          }
          gl_FragColor=vec4(base.rgb/max(base.a,.00001),base.a);
        }`,
    });
    this.dilation = new ShaderMaterial({
      uniforms: this.uniforms,
      vertexShader: this.material.vertexShader,
      fragmentShader: `${depthFunctions}
        uniform sampler2D dilationInput;
        uniform vec2 dilationAxis;
        uniform float dilationStage;
        void main(){
          float radius=0.0;
          for(int i=-16;i<=16;i++){
            vec2 uv=clamp(vUv+dilationAxis*(float(i)/16.0)*(maxRadius+2.0)*texel,texel*.5,1.0-texel*.5);
            radius=max(radius,dilationStage<.5?max(0.0,-radiusAt(uv,depthAt(uv))):texture2D(dilationInput,uv).r);
          }
          gl_FragColor=vec4(radius,0.0,0.0,1.0);
        }`,
      depthTest: false,
      depthWrite: false,
    });
    this.dilationQuad = new FullScreenQuad(this.dilation);
    this.prefilter = new ShaderMaterial({
      uniforms: this.uniforms,
      vertexShader: this.material.vertexShader,
      depthTest: false,
      depthWrite: false,
      fragmentShader: `${depthFunctions}
        uniform float prefilterLayer;
        void main(){
          vec4 sum=vec4(0.0);
          for(int y=0;y<2;y++)for(int x=0;x<2;x++){
            vec2 uv=clamp(vUv+(vec2(float(x),float(y))-.5)*texel,texel*.5,1.0-texel*.5);
            float r=radiusAt(uv,depthAt(uv));float weight=prefilterLayer<.5?step(.5,-r):step(.5,r);
            vec4 c=texture2D(tDiffuse,uv);sum+=vec4(c.rgb*c.a,c.a)*weight;
          }
          gl_FragColor=sum*.25;
        }`,
    });
    this.prefilterQuad = new FullScreenQuad(this.prefilter);
    this.enabled = false;
  }
  override render(
    renderer: WebGLRenderer,
    writeBuffer: WebGLRenderTarget,
    readBuffer: WebGLRenderTarget,
    deltaTime?: number,
    maskActive?: boolean,
  ) {
    this.uniforms.tDiffuse.value = readBuffer.texture;
    this.uniforms.prefilterLayer.value = 0;
    renderer.setRenderTarget(this.nearColor);
    this.prefilterQuad.render(renderer);
    this.uniforms.prefilterLayer.value = 1;
    renderer.setRenderTarget(this.farColor);
    this.prefilterQuad.render(renderer);
    this.uniforms.nearColor.value = this.nearColor.texture;
    this.uniforms.farColor.value = this.farColor.texture;
    this.uniforms.dilationStage.value = 0;


    this.uniforms.dilationInput.value = this.uniforms.depthMap.value;
    this.uniforms.dilationAxis.value.set(1, 0);
    renderer.setRenderTarget(this.nearX);
    this.dilationQuad.render(renderer);
    this.uniforms.dilationStage.value = 1;
    this.uniforms.dilationAxis.value.set(0, 1);
    this.uniforms.dilationInput.value = this.nearX.texture;
    renderer.setRenderTarget(this.nearY);
    this.dilationQuad.render(renderer);
    this.uniforms.nearField.value = this.nearY.texture;
    super.render(
      renderer,
      writeBuffer,
      readBuffer,
      deltaTime ?? 0,
      maskActive ?? false,
    );
  }
  configure(
    s: RenderSettings,
    camera: PerspectiveCamera | OrthographicCamera,
    depth: Texture,
    map = false,
    cloudDepth?: Texture,
    target?: Vector3,
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
    u.focal.value =
      camera instanceof PerspectiveCamera
        ? 0.012 / Math.tan((camera.getEffectiveFOV() * Math.PI) / 360)
        : s.dofFocalLength / 1000;
    u.sensorWidth.value =
      camera instanceof PerspectiveCamera ? 24 * camera.aspect : 36;
    u.focus.value =
      s.dofMode === "lens"
        ? Math.max(u.focal.value + 0.0001, s.dofDistance / s.dofWorldScale)
        : Math.max(0.000001, s.dofDistance / s.dofWorldScale);
    u.fStop.value = s.dofAperture;
    u.worldScale.value = s.dofWorldScale;
    this.radius = s.dofMaxBlur;
    u.maxRadius.value = (this.radius * this.height) / 1080;
    u.nearStrength.value = s.dofNearStrength;
    u.farStrength.value = s.dofFarStrength;
    u.focusRange.value = s.dofFocusRange;
    u.transition.value = s.dofTransition;
    u.samples.value.set(
      s.quality === "fine" ? 96 : s.quality === "fast" ? 32 : 64,
      s.quality === "fine" ? 64 : s.quality === "fast" ? 16 : 32,
    );
    u.mode.value =
      s.dofMode === "lens" ? 0 : s.dofMode === "subject" && target ? 2 : 1;
    camera.updateMatrixWorld(true);
    u.projectionInverse.value.copy(camera.projectionMatrixInverse);
    u.targetView.value
      .copy(target ?? new Vector3())
      .applyMatrix4(camera.matrixWorldInverse);
  }
  override setSize(width: number, height: number) {
    this.height = height;
    this.uniforms.texel.value.set(1 / width, 1 / height);
    this.uniforms.imageWidth.value = width;
    this.uniforms.maxRadius.value = (this.radius * height) / 1080;
    this.nearX.setSize(
      Math.max(1, Math.ceil(width / 4)),
      Math.max(1, Math.ceil(height / 4)),
    );
    this.nearY.setSize(this.nearX.width, this.nearX.height);
    this.nearColor.setSize(
      Math.max(1, Math.ceil(width / 2)),
      Math.max(1, Math.ceil(height / 2)),
    );
    this.farColor.setSize(this.nearColor.width, this.nearColor.height);
  }
  override dispose() {
    super.dispose();
    this.nearX.dispose();
    this.nearY.dispose();
    this.dilation.dispose();
    this.dilationQuad.dispose();
    this.nearColor.dispose();
    this.farColor.dispose();
    this.prefilter.dispose();
    this.prefilterQuad.dispose();
  }
}

const depthFunctions = `#include <packing>
        uniform sampler2D tDiffuse,depthMap,cloudDepthMap;
        uniform float cloudDepthEnabled;
        uniform vec2 texel;
        uniform float nearClip,farClip,perspective,focus,focal,fStop,worldScale,maxRadius,imageWidth,sensorWidth;
        uniform float nearStrength,farStrength,focusRange,transition,mode;
        uniform mat4 projectionInverse;
        uniform vec3 targetView;
        varying vec2 vUv;
        float depthAt(vec2 uv){
          float d=texture2D(depthMap,uv).x;
          float z=-(perspective>0.5 ? perspectiveDepthToViewZ(d,nearClip,farClip) : orthographicDepthToViewZ(d,nearClip,farClip));
          if(cloudDepthEnabled>.5){vec2 cloud=texture2D(cloudDepthMap,uv).rg;
            if(cloud.r>.001&&cloud.r<z&&cloud.g>.005)z=1./mix(1./max(z,.001),1./cloud.r,clamp(cloud.g,0.,1.));
          }
          return z/worldScale;
        }
        float radiusAt(vec2 uv,float z){
          float delta=(z-focus)*worldScale;
          float strength=delta<0.0?nearStrength:farStrength;
          float lens=focal*focal*(z-focus)/(fStop*max(z,.0001)*max(focus-focal,.0001))*1000.0/sensorWidth*imageWidth*.5;
          if(mode<.5)return clamp(lens*strength,-maxRadius,maxRadius);
          float distanceFromFocus=abs(delta);
          if(mode>1.5){
            float d=texture2D(depthMap,uv).r;
            vec4 view=projectionInverse*vec4(uv*2.0-1.0,d*2.0-1.0,1.0);
            vec3 position=view.xyz/view.w;
            if(perspective>.5)position*=z*worldScale/max(-position.z,.00001);
            else position.z=-z*worldScale;
            distanceFromFocus=length(position-targetView);
          }
          float blur=smoothstep(0.0,max(transition,.1),max(0.0,distanceFromFocus-focusRange))*strength;
          return (delta<0.0?-1.0:1.0)*min(maxRadius,maxRadius*blur);
        }`;

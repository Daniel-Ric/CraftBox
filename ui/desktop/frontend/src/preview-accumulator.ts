import * as THREE from "three";
import {Pass,FullScreenQuad} from "three/addons/postprocessing/Pass.js";

// Incremental temporal AA. This averages real subpixel samples; it neither
// alters geometry nor paints artificial grain over the scene.
export class PreviewAccumulator extends Pass {
  samples=0;
  maxSamples=12;
  private targets=[0,1].map(()=>new THREE.WebGLRenderTarget(1,1,{type:THREE.HalfFloatType,depthBuffer:false}));
  private index=0;
  private material=new THREE.ShaderMaterial({
    uniforms:{current:{value:null},history:{value:null},count:{value:0}},
    vertexShader:"varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}",
    fragmentShader:"uniform sampler2D current,history;uniform float count;varying vec2 vUv;void main(){vec4 c=texture2D(current,vUv);gl_FragColor=count<0.5?c:mix(c,texture2D(history,vUv),count/(count+1.0));}",depthTest:false,depthWrite:false,toneMapped:false,
  });
  private copy=new THREE.ShaderMaterial({uniforms:{source:{value:null}},vertexShader:"varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}",fragmentShader:"uniform sampler2D source;varying vec2 vUv;void main(){gl_FragColor=texture2D(source,vUv);}",depthTest:false,depthWrite:false,toneMapped:false});
  private quad=new FullScreenQuad(this.material);
  reset(){this.samples=0}
  setSize(width:number,height:number){for(const t of this.targets)t.setSize(width,height);this.reset()}
  render(renderer:THREE.WebGLRenderer,write:THREE.WebGLRenderTarget,read:THREE.WebGLRenderTarget){
    const previous=renderer.getRenderTarget(),next=1-this.index;
    try{
      this.material.uniforms.current.value=read.texture;this.material.uniforms.history.value=this.targets[this.index].texture;this.material.uniforms.count.value=this.samples;
      renderer.setRenderTarget(this.targets[next]);this.quad.material=this.material;this.quad.render(renderer);
      this.copy.uniforms.source.value=this.targets[next].texture;this.quad.material=this.copy;
      renderer.setRenderTarget(this.renderToScreen?null:write);this.quad.render(renderer);
      this.index=next;this.samples=Math.min(this.maxSamples,this.samples+1);
    }finally{renderer.setRenderTarget(previous)}
  }
  dispose(){for(const t of this.targets)t.dispose();this.material.dispose();this.copy.dispose();this.quad.dispose()}
}
export function previewJitter(index:number):[number,number]{
  const halton=(n:number,base:number)=>{let value=0,f=1;while(n>0){f/=base;value+=f*(n%base);n=Math.floor(n/base)}return value};
  return [halton(index+1,2)-.5,halton(index+1,3)-.5];
}

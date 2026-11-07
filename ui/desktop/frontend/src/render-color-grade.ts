import { Vector2 } from "three";
import { ShaderPass } from "three/addons/postprocessing/ShaderPass.js";
import type { RenderSettings } from "./render-settings";

export class ColorGradePass extends ShaderPass {
  constructor() {
    super({
      uniforms: {
        tDiffuse: { value: null },
        texel: { value: new Vector2(1, 1) },
        saturation: { value: 1 },
        vibrance: { value: 0 },
        contrast: { value: 1 },
        temperature: { value: 0 },
        tint: { value: 0 },
        highlights: { value: 0 },
        shadows: { value: 0 },
        sharpen: { value: 0 },
        detailProtection: { value: 0.65 },
        vignette: { value: 0 },
      },
      vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
      fragmentShader: `uniform sampler2D tDiffuse;uniform vec2 texel;uniform float saturation,vibrance,contrast,temperature,tint,highlights,shadows,sharpen,vignette,detailProtection;varying vec2 vUv;
   void main(){
    vec4 source=texture2D(tDiffuse,vUv);vec3 c=source.rgb;
    vec4 a=texture2D(tDiffuse,vUv+vec2(texel.x,0.0)),b=texture2D(tDiffuse,vUv-vec2(texel.x,0.0)),d=texture2D(tDiffuse,vUv+vec2(0.0,texel.y)),e=texture2D(tDiffuse,vUv-vec2(0.0,texel.y));
    float sum=a.a+b.a+d.a+e.a;
    vec3 blurred=(a.rgb*a.a+b.rgb*b.a+d.rgb*d.a+e.rgb*e.a)/max(sum,0.0001);
    vec3 lo=min(c,min(min(a.a>0.0?a.rgb:c,b.a>0.0?b.rgb:c),min(d.a>0.0?d.rgb:c,e.a>0.0?e.rgb:c)));
    vec3 hi=max(c,max(max(a.a>0.0?a.rgb:c,b.a>0.0?b.rgb:c),max(d.a>0.0?d.rgb:c,e.a>0.0?e.rgb:c)));
    float fineContrast=abs(dot(c-blurred,vec3(.2126,.7152,.0722)));
    float sharpGain=sharpen*(1.0-detailProtection*smoothstep(.025,.16,fineContrast));
    if(sum>0.0)c=clamp(c+(c-blurred)*sharpGain,lo,hi);
    c*=vec3(1.0+temperature*0.13+tint*0.04,1.0-tint*0.08,1.0-temperature*0.13+tint*0.04);
    float y=dot(c,vec3(0.2126,0.7152,0.0722));
    c+=vec3(highlights*smoothstep(0.5,1.0,y)*0.22+shadows*(1.0-smoothstep(0.0,0.65,y))*0.16);
    c=(c-0.5)*contrast+0.5;
    y=dot(c,vec3(0.2126,0.7152,0.0722));
    float chroma=max(c.r,max(c.g,c.b))-min(c.r,min(c.g,c.b));
    c=mix(vec3(y),c,saturation*(1.0+vibrance*(1.0-clamp(chroma,0.0,1.0))));
    float radius=length((vUv-0.5)*vec2(1.0,0.8));
    c*=1.0-vignette*smoothstep(0.2,0.65,radius)*0.5;
    gl_FragColor=vec4(clamp(c,0.0,1.0),source.a);
   }`,
    });
  }
  configure(s: RenderSettings) {
    this.enabled = s.grading;
    for (const name of [
      "saturation",
      "vibrance",
      "contrast",
      "temperature",
      "tint",
      "highlights",
      "shadows",
      "sharpen",
      "detailProtection",
      "vignette",
    ] as const)
      this.uniforms[name].value = s[name];
  }
  setSize(width: number, height: number) {
    this.uniforms.texel.value.set(1 / width, 1 / height);
  }
}

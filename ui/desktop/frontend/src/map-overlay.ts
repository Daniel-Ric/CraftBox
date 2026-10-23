import * as THREE from "three";
import { FullScreenQuad, Pass } from "three/addons/postprocessing/Pass.js";
import type { MapPresentation } from "./map-presentation";
export class MapOverlayPass extends Pass {
  private canvas = document.createElement("canvas");
  private annotations = new THREE.CanvasTexture(this.canvas);
  private markerKey = "";
  private material = new THREE.ShaderMaterial({
    depthTest: false,
    depthWrite: false,
    uniforms: {
      annotations: { value: this.annotations },
      beauty: { value: null as THREE.Texture | null },
      depth: { value: null as THREE.Texture | null },
      inverseProjection: { value: new THREE.Matrix4() },
      world: { value: new THREE.Matrix4() },
      options: { value: new THREE.Vector4() },
      height: { value: new THREE.Vector4() },
    },
    vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=vec4(position.xy,0.,1.);}`,
    fragmentShader: `varying vec2 vUv;uniform sampler2D beauty,depth,annotations;uniform mat4 inverseProjection,world;uniform vec4 options,height;
 void main(){vec4 color=texture2D(beauty,vUv);float d=texture2D(depth,vUv).r;if(d<.999999){vec4 q=inverseProjection*vec4(vUv*2.-1.,d*2.-1.,1.);vec3 p=(world*vec4(q.xyz/q.w,1.)).xyz;
 float h=clamp((p.y-height.x)/max(1.,height.y-height.x),0.,1.);vec3 low=mix(vec3(.22,.48,.48),vec3(.58,.67,.34),smoothstep(0.,.5,h));vec3 high=mix(vec3(.67,.53,.35),vec3(.95),smoothstep(.65,1.,h));vec3 tint=mix(low,high,smoothstep(.4,.8,h));color.rgb=mix(color.rgb,tint,height.z*.7);
 vec2 g=abs(fract(p.xz/options.x+.5)-.5)/max(fwidth(p.xz/options.x),vec2(.0001));float line=1.-clamp(min(g.x,g.y),0.,1.);color.rgb=mix(color.rgb,vec3(.22,.36,.48),line*options.y);
 float v=abs(fract(p.y/options.z+.5)-.5)/max(fwidth(p.y/options.z),.015);float contour=1.-clamp(v,0.,1.);color.rgb=mix(color.rgb,vec3(.2,.16,.1),contour*options.w*.5);
 }vec4 marker=texture2D(annotations,vUv);float alpha=marker.a+color.a*(1.-marker.a);vec3 rgb=(marker.rgb*marker.a+color.rgb*color.a*(1.-marker.a))/max(alpha,.00001);gl_FragColor=vec4(rgb,alpha);}`,
  });
  private quad = new FullScreenQuad(this.material);
  constructor() {
    super();
    this.enabled = false;
  }
  configure(
    s: MapPresentation | undefined,
    camera: THREE.Camera,
    depth: THREE.Texture,
    bounds: THREE.Box3,
    origin?: THREE.Vector3,
    pixelRatio = 1,
  ) {
    this.enabled =
      !!s?.enabled &&
      (s.grid || s.contours || s.heightTint > 0 || s.markers.length > 0);
    if (!s) return;
    const u = this.material.uniforms;
    u.options.value.set(
      s.gridSize,
      s.grid ? s.gridOpacity : 0,
      s.contourStep,
      s.contours ? 1 : 0,
    );
    u.height.value.set(bounds.min.y, bounds.max.y, s.heightTint, 0);
    u.depth.value = depth;
    u.inverseProjection.value.copy(camera.projectionMatrixInverse);
    u.world.value.copy(camera.matrixWorld);
    this.drawMarkers(s, camera, bounds, origin, pixelRatio);
  }
  private drawMarkers(
    s: MapPresentation,
    camera: THREE.Camera,
    bounds: THREE.Box3,
    origin?: THREE.Vector3,
    pixelRatio = 1,
  ) {
    const key = JSON.stringify([
      s.markers,
      camera.matrixWorld.elements,
      camera.projectionMatrix.elements,
      this.canvas.width,
      this.canvas.height,
      origin?.x,
      origin?.z,
      pixelRatio,
    ]);
    if (key === this.markerKey) return;
    this.markerKey = key;
    const ctx = this.canvas.getContext("2d");
    if (!ctx) return;
    ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    const point = new THREE.Vector3();
    for (const marker of s.markers) {
      point
        .set(
          marker.x - (origin?.x ?? 0),
          bounds.max.y + 1,
          marker.z - (origin?.z ?? 0),
        )
        .project(camera);
      if (point.z < -1 || point.z > 1) continue;
      const x = (point.x * 0.5 + 0.5) * this.canvas.width,
        y = (-point.y * 0.5 + 0.5) * this.canvas.height,
        scale = marker.scale * Math.max(this.canvas.height / 900, pixelRatio);
      const h = 34 * scale,
        pad = 12 * scale;
      ctx.font = `600 ${14 * scale}px "Segoe UI",sans-serif`;
      const text = [marker.icon, marker.label].filter(Boolean).join("  "),
        width = Math.max(h, ctx.measureText(text).width + pad * 2 + 10 * scale);
      ctx.fillStyle = marker.background;
      ctx.beginPath();
      ctx.roundRect(x - width / 2, y - h - 7 * scale, width, h, 8 * scale);
      ctx.fill();
      ctx.beginPath();
      ctx.moveTo(x - 5 * scale, y - 8 * scale);
      ctx.lineTo(x, y);
      ctx.lineTo(x + 5 * scale, y - 8 * scale);
      ctx.fill();
      ctx.fillStyle = marker.color;
      ctx.beginPath();
      ctx.arc(
        x - width / 2 + pad,
        y - h / 2 - 7 * scale,
        4 * scale,
        0,
        Math.PI * 2,
      );
      ctx.fill();
      ctx.fillStyle = "#ffffff";
      ctx.textAlign = "left";
      ctx.textBaseline = "middle";
      ctx.fillText(
        text,
        x - width / 2 + pad + 12 * scale,
        y - h / 2 - 7 * scale,
      );
    }
    this.annotations.needsUpdate = true;
  }
  setSize(width: number, height: number) {
    this.canvas.width = Math.max(1, Math.round(width));
    this.canvas.height = Math.max(1, Math.round(height));
    this.markerKey = "";
  }
  render(
    renderer: THREE.WebGLRenderer,
    write: THREE.WebGLRenderTarget,
    read: THREE.WebGLRenderTarget,
  ) {
    this.material.uniforms.beauty.value = read.texture;
    renderer.setRenderTarget(this.renderToScreen ? null : write);
    this.quad.render(renderer);
  }
  dispose() {
    this.annotations.dispose();
    this.material.dispose();
    this.quad.dispose();
  }
}

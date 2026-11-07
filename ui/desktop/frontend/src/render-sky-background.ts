import * as THREE from "three";
import { FullScreenQuad, Pass } from "three/addons/postprocessing/Pass.js";

export function skyViewRotation(
  camera: THREE.Camera,
  rotation: number,
  elevation: number,
) {
  camera.updateWorldMatrix(true, false);
  const orientation = camera.getWorldQuaternion(new THREE.Quaternion());
  return new THREE.Matrix3().setFromMatrix4(
    new THREE.Matrix4()
      .makeRotationY((-rotation * Math.PI) / 180)
      .multiply(new THREE.Matrix4().makeRotationX((elevation * Math.PI) / 180))
      .multiply(new THREE.Matrix4().makeRotationFromQuaternion(orientation)),
  );
}

export class SkyBackgroundPass extends Pass {
  private material = new THREE.ShaderMaterial({
    depthTest: false,
    depthWrite: false,
    uniforms: {
      panorama: { value: null },
      viewRotation: { value: new THREE.Matrix3() },
      moonViewRotation: { value: new THREE.Matrix3() },
      aspect: { value: 1 },
      intensity: { value: 1 },
      night: { value: 0 },
      moonDirection: { value: new THREE.Vector3(0, 1, 0) },
      tanHalfFov: { value: Math.tan((65 * Math.PI) / 360) },
    },
    vertexShader: `varying vec2 vUv;void main(){vUv=uv;gl_Position=vec4(position.xy,1.,1.);}`,
    fragmentShader: `uniform sampler2D panorama;uniform mat3 viewRotation,moonViewRotation;uniform float aspect,intensity,tanHalfFov,night;uniform vec3 moonDirection;varying vec2 vUv;
      void main(){vec3 ray=normalize(viewRotation*vec3((vUv.x*2.-1.)*aspect*tanHalfFov,(vUv.y*2.-1.)*tanHalfFov,-1.));
      vec2 uv=vec2(atan(ray.z,ray.x)*.159154943+.5,asin(clamp(ray.y,-1.,1.))*.318309886+.5);
      vec3 color=texture2D(panorama,uv).rgb*intensity;
      if(night>.5){
        vec3 moonRay=normalize(moonViewRotation*vec3((vUv.x*2.-1.)*aspect*tanHalfFov,(vUv.y*2.-1.)*tanHalfFov,-1.));
        float angle=acos(clamp(dot(moonRay,moonDirection),-1.,1.));
        float disc=1.-smoothstep(.03,.032,angle);
        vec3 tangent=normalize(cross(moonDirection,vec3(0.,1.,.001))),up=cross(tangent,moonDirection);
        vec2 q=vec2(dot(moonRay,tangent),dot(moonRay,up))/.031;
        float sphere=sqrt(max(0.,1.-dot(q,q)));
        float craters=.88+.07*sin(q.x*19.+sin(q.y*13.))+.05*sin(q.y*27.+q.x*7.);
        float phase=smoothstep(-.3,.15,q.x*.35+sphere*.85);
        vec3 moon=vec3(.65,.76,.94)*craters*(.3+.7*sphere)*phase;
        color=mix(color,moon*max(.5,intensity*3.),disc);
        color+=vec3(.035,.055,.095)*exp(-angle*angle/.0015)*max(0.,1.-disc);
      }
      gl_FragColor=vec4(color,1.);}`,
  });
  private quad = new FullScreenQuad(this.material);
  constructor() {
    super();
    this.needsSwap = false;
    this.enabled = false;
  }
  configure(
    texture: THREE.Texture | undefined,
    camera: THREE.OrthographicCamera | THREE.PerspectiveCamera,
    rotation: number,
    elevation: number,
    intensity: number,
    moonDirection?: THREE.Vector3,
  ) {
    this.enabled = Boolean(texture);
    this.material.uniforms.panorama.value = texture ?? null;
    this.material.uniforms.viewRotation.value.copy(
      skyViewRotation(camera, rotation, elevation),
    );
    this.material.uniforms.tanHalfFov.value =
      Math.tan(
        ((camera instanceof THREE.PerspectiveCamera ? camera.fov : 65) *
          Math.PI) /
          360,
      ) /
      (camera instanceof THREE.PerspectiveCamera
        ? Math.max(0.01, camera.zoom)
        : 1);
    this.material.uniforms.intensity.value = intensity;
    this.material.uniforms.night.value = moonDirection ? 1 : 0;
    this.material.uniforms.moonViewRotation.value.copy(
      skyViewRotation(camera, 0, 0),
    );
    if (moonDirection)
      this.material.uniforms.moonDirection.value
        .copy(moonDirection)
        .normalize();
  }
  setSize(width: number, height: number) {
    this.material.uniforms.aspect.value = width / height;
  }
  render(
    renderer: THREE.WebGLRenderer,
    _write: THREE.WebGLRenderTarget,
    read: THREE.WebGLRenderTarget,
  ) {
    const auto = renderer.autoClear;
    renderer.autoClear = false;
    try {
      renderer.setRenderTarget(read);
      renderer.clear();
      this.quad.render(renderer);
    } finally {
      renderer.autoClear = auto;
    }
  }
  dispose() {
    this.material.dispose();
    this.quad.dispose();
  }
}

import * as THREE from "three";
import { Pass } from "three/addons/postprocessing/Pass.js";

export class TerrainNormalsPass extends Pass {
  target = new THREE.WebGLRenderTarget(1, 1, {
    type: THREE.HalfFloatType,
    minFilter: THREE.NearestFilter,
    magFilter: THREE.NearestFilter,
    depthTexture: new THREE.DepthTexture(1, 1, THREE.UnsignedIntType),
  });
  private replacements = new Map<THREE.Material, THREE.ShaderMaterial>();
  private scene: THREE.Scene;
  private camera: THREE.Camera;
  private terrain: THREE.Group;
  excluded: THREE.Object3D[] = [];
  includeTransparent = false;
  constructor(scene: THREE.Scene, camera: THREE.Camera, terrain: THREE.Group) {
    super();
    this.scene = scene;
    this.camera = camera;
    this.terrain = terrain;
    this.needsSwap = false;
  }
  setSize(width: number, height: number) {
    this.target.setSize(width, height);
  }
  setCamera(camera: THREE.Camera) {
    this.camera = camera;
  }
  render(renderer: THREE.WebGLRenderer) {
    const target = renderer.getRenderTarget(),
      background = this.scene.background;
    const color = renderer.getClearColor(new THREE.Color()),
      alpha = renderer.getClearAlpha();
    const shadows = renderer.shadowMap.enabled,
      autoClear = renderer.autoClear;
    const restore: {
      mesh: THREE.Mesh;
      material: THREE.Material | THREE.Material[];
      visible: boolean;
    }[] = [];
    const excludedVisibility = this.excluded.map((object) => ({
      object,
      visible: object.visible,
    }));
    try {
      for (const { object } of excludedVisibility) object.visible = false;
      this.terrain.traverse((item) => {
        if (!(item instanceof THREE.Mesh)) return;
        restore.push({
          mesh: item,
          material: item.material,
          visible: item.visible,
        });
        const source = item.material as THREE.MeshStandardMaterial;
        if (!source.visible) {
          item.visible = false;
          return;
        }
        if (source.transparent && !this.includeTransparent) {
          item.visible = false;
          return;
        }
        let material = this.replacements.get(source);
        if (!material) {
          material = new THREE.ShaderMaterial({
            side: source.side,
            uniforms: {
              atlas: { value: source.map ?? null },
              cutoff: { value: source.alphaTest },
            },
            vertexShader: `attribute float foliage; varying float plant; varying vec2 atlasUv; varying vec3 viewNormal;
              void main(){plant=foliage;atlasUv=uv;viewNormal=normalize(normalMatrix*normal);gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
            fragmentShader: `uniform sampler2D atlas;uniform float cutoff;varying float plant;varying vec2 atlasUv;varying vec3 viewNormal;
              void main(){if(cutoff>0.0&&texture2D(atlas,atlasUv).a<cutoff)discard;
              vec3 n=normalize(viewNormal)*(gl_FrontFacing?1.0:-1.0);vec3 canopy=normalize(mat3(viewMatrix)*vec3(0.0,1.0,0.0));
              if(plant>0.5)n=plant>1.5?normalize(n*.55+canopy*.85):canopy;gl_FragColor=vec4(n*.5+.5,1.0);}`,
          });
          this.replacements.set(source, material);
        }
        material.uniforms.atlas.value = source.map ?? null;
        material.uniforms.cutoff.value =
          this.includeTransparent && source.transparent
            ? Math.max(0.01, source.alphaTest)
            : source.alphaTest;
        item.material = material;
      });
      this.scene.background = null;
      renderer.shadowMap.enabled = false;
      renderer.autoClear = false;
      renderer.setRenderTarget(this.target);
      renderer.setClearColor(0x7777ff, 1);
      renderer.clear();
      renderer.render(this.scene, this.camera);
    } finally {
      for (const { object, visible } of excludedVisibility)
        object.visible = visible;
      for (const item of restore) {
        item.mesh.material = item.material;
        item.mesh.visible = item.visible;
      }
      this.scene.background = background;
      renderer.shadowMap.enabled = shadows;
      renderer.autoClear = autoClear;
      renderer.setRenderTarget(target);
      renderer.setClearColor(color, alpha);
    }
  }
  dispose() {
    this.target.dispose();
    for (const material of this.replacements.values()) material.dispose();
    this.replacements.clear();
  }
}

import { expect, it } from "vitest";
import * as THREE from "three";
import { TerrainNormalsPass } from "./render-normals";
it.each([false, true])(
  "honors foliage alpha and configurable transparent depth, restoring materials after errors (transparent=%s)",
  (includeTransparent) => {
    const scene = new THREE.Scene(),
      terrain = new THREE.Group();
    scene.add(terrain);
    const atlas = new THREE.Texture();
    const original = new THREE.MeshStandardMaterial({
      map: atlas,
      alphaTest: 0.5,
    });
    const leaf = new THREE.Mesh(new THREE.PlaneGeometry(), original);
    const glass = new THREE.Mesh(
      new THREE.PlaneGeometry(),
      new THREE.MeshStandardMaterial({ transparent: true }),
    );
    terrain.add(leaf, glass);
    scene.background = new THREE.Color("red");
    const pass = new TerrainNormalsPass(
      scene,
      new THREE.OrthographicCamera(),
      terrain,
    );
    pass.includeTransparent = includeTransparent;
    const renderer = {
      shadowMap: { enabled: true },
      autoClear: true,
      getRenderTarget: () => null,
      setRenderTarget: () => {},
      getClearColor: (c: THREE.Color) => c.set("blue"),
      getClearAlpha: () => 1,
      setClearColor: () => {},
      clear: () => {},
      render: () => {
        const material = leaf.material as unknown as THREE.ShaderMaterial;
        expect(material.uniforms.atlas.value).toBe(atlas);
        expect(material.uniforms.cutoff.value).toBe(0.5);
        expect(glass.visible).toBe(includeTransparent);
        expect(scene.background).toBe(null);
        throw new Error("test render failure");
      },
    };
    expect(() =>
      pass.render(renderer as unknown as THREE.WebGLRenderer),
    ).toThrow("test render failure");
    expect(leaf.material).toBe(original);
    expect(glass.visible).toBe(true);
    expect(scene.background).toEqual(new THREE.Color("red"));
    expect(renderer.shadowMap.enabled).toBe(true);
    pass.dispose();
  },
);

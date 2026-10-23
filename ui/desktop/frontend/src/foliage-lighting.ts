import type { Material } from "three";


export function foliageAttributes(positions: Float32Array) {
  const flags = new Float32Array(positions.length / 3),
    eps = 1e-4;
  for (let v = 0; v + 3 < flags.length; v += 4) {
    const at = v * 3;
    const same = (a: number, b: number) => Math.abs(a - b) < eps;
    const vertical =
      same(positions[at + 1], positions[at + 4]) &&
      same(positions[at + 7], positions[at + 10]) &&
      !same(positions[at + 1], positions[at + 7]);
    const edges =
      same(positions[at], positions[at + 9]) &&
      same(positions[at + 2], positions[at + 11]) &&
      same(positions[at + 3], positions[at + 6]) &&
      same(positions[at + 5], positions[at + 8]);
    const diagonal =
      !same(positions[at], positions[at + 3]) &&
      !same(positions[at + 2], positions[at + 5]);
    if (vertical && edges && diagonal) flags.fill(1, v, v + 4);


    for (let axis = 0; axis < 3; axis++) {
      const plane = positions[at + axis],
        fraction = plane - Math.floor(plane);
      if (
        (same(fraction, 1 / 64) || same(fraction, 63 / 64)) &&
        [1, 2, 3].every((i) => same(positions[at + i * 3 + axis], plane))
      )
        flags.fill(2, v, v + 4);
    }
  }
  return flags;
}
export function installFoliageLighting(material: Material) {
  const original = material.onBeforeCompile,
    cache = material.customProgramCacheKey.bind(material);
  material.onBeforeCompile = function (shader, renderer) {
    original.call(this, shader, renderer);
    shader.vertexShader = shader.vertexShader
      .replace(
        "#include <common>",
        "#include <common>\nattribute float foliage; varying float vFoliage;",
      )
      .replace(
        "#include <begin_vertex>",
        "#include <begin_vertex>\nvFoliage=foliage;",
      );
    shader.fragmentShader = shader.fragmentShader
      .replace(
        "#include <common>",
        "#include <common>\nvarying float vFoliage;",
      )
      .replace(
        "#include <normal_fragment_begin>",
        "#include <normal_fragment_begin>\nif(vFoliage>0.5){vec3 canopy=normalize(mat3(viewMatrix)*vec3(0.0,1.0,0.0));normal=vFoliage>1.5?normalize(normal*.55+canopy*.85):canopy;nonPerturbedNormal=normal;}",
      );
  };
  const previous = cache();
  material.customProgramCacheKey = () => previous + "-canopy-vines-v2";
}

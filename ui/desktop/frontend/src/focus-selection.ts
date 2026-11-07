import {
  Camera,
  Object3D,
  Raycaster,
  Vector2,
  Mesh,
  MeshBasicMaterial,
} from "three";

export function pickFocusSurface(
  terrain: Object3D,
  camera: Camera,
  pointer: Vector2,
) {
  camera.updateMatrixWorld(true);
  terrain.updateMatrixWorld(true);
  const ray = new Raycaster();
  ray.setFromCamera(pointer, camera);
  return ray.intersectObject(terrain, true).find((hit) => {
    for (
      let object: Object3D | null = hit.object;
      object;
      object = object.parent
    )
      if (!object.visible) return false;
    const mesh = hit.object as Mesh;
    const material = (
      Array.isArray(mesh.material)
        ? mesh.material[hit.face?.materialIndex ?? 0]
        : mesh.material
    ) as MeshBasicMaterial;
    if (!material?.visible || material.opacity <= 0.001) return false;
    const cutoff = Math.max(
      material.alphaTest,
      material.transparent ? 0.01 : 0,
    );
    const image = material.map?.image as
      { data?: ArrayLike<number>; width: number; height: number } | undefined;
    if (
      cutoff > 0 &&
      hit.uv &&
      image?.data &&
      image.data.length >= image.width * image.height * 4
    ) {
      const uv = hit.uv.clone();
      material.map!.updateMatrix();
      material.map!.transformUv(uv);
      const x = Math.max(
          0,
          Math.min(image.width - 1, Math.floor(uv.x * image.width)),
        ),
        y = Math.max(
          0,
          Math.min(image.height - 1, Math.floor(uv.y * image.height)),
        );
      if (image.data[(y * image.width + x) * 4 + 3] / 255 < cutoff)
        return false;
    }
    return true;
  });
}

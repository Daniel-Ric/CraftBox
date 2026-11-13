import * as T from "three";
import { mergeGeometries } from "three/addons/utils/BufferGeometryUtils.js";

function singleMaterial(mesh: T.Mesh): T.Material | undefined {
  const materials = mesh.material;
  if (!Array.isArray(materials)) return materials;
  const used = mesh.geometry.groups.length
    ? mesh.geometry.groups.map((g) => materials[g.materialIndex ?? 0])
    : materials;
  const first = used[0];
  return first && used.every((m) => m === first) ? first : undefined;
}
export function compactPreviewModel(root: T.Group) {
  const parents: T.Object3D[] = [];
  root.traverse((node) => {
    if (!(node instanceof T.Mesh)) parents.push(node);
  });
  for (const parent of parents) {
    const groups = new Map<string, T.Mesh[]>();
    for (const child of parent.children) {
      if (
        !(child instanceof T.Mesh) ||
        child.children.length ||
        child.userData.previewOnly ||
        Object.keys(child.geometry.morphAttributes).length
      )
        continue;
      const material = singleMaterial(child);
      if (!material || material.transparent) continue;
      child.material = material;
      const key = [
        material.uuid,
        child.name,
        child.userData.layer,
        child.userData.authoredCulling,
        child.visible,
        child.renderOrder,
        child.layers.mask,
        child.castShadow,
        child.receiveShadow,
        child.frustumCulled,
      ].join("|");
      const list = groups.get(key) ?? [];
      list.push(child);
      groups.set(key, list);
    }
    const chunks: T.Mesh[][] = [];
    for (const group of groups.values()) {
      let chunk: T.Mesh[] = [],
        vertices = 0;
      for (const mesh of group) {
        const count = mesh.geometry.attributes.position.count;
        if (chunk.length && vertices + count > 32768) {
          chunks.push(chunk);
          chunk = [];
          vertices = 0;
        }
        chunk.push(mesh);
        vertices += count;
      }
      if (chunk.length) chunks.push(chunk);
    }
    for (const meshes of chunks) {
      if (meshes.length < 2) continue;
      const geometries = meshes.map((mesh) => {
        mesh.updateMatrix();
        return mesh.geometry.clone().applyMatrix4(mesh.matrix);
      });
      const merged = mergeGeometries(geometries, false);
      geometries.forEach((g) => g.dispose());
      if (!merged) continue;
      const first = meshes[0],
        batch = new T.Mesh(merged, first.material);
      batch.name = first.name;
      batch.userData = { ...first.userData };
      batch.visible = first.visible;
      batch.renderOrder = first.renderOrder;
      batch.layers.mask = first.layers.mask;
      batch.castShadow = first.castShadow;
      batch.receiveShadow = first.receiveShadow;
      batch.frustumCulled = first.frustumCulled;
      parent.add(batch);
      for (const mesh of meshes) {
        parent.remove(mesh);
        mesh.geometry.dispose();
      }
    }
  }
}

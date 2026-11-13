import * as T from "three";
import type { OrbitControls } from "three/addons/controls/OrbitControls.js";

export function visibleSelectionBounds(meshes: Iterable<T.Mesh>) {
  const bounds = new T.Box3();
  for (const mesh of meshes) {
    let visible = true;
    for (let node: T.Object3D | null = mesh; node; node = node.parent) if (!node.visible) { visible = false; break; }
    if (!visible || mesh.userData.previewOnly) continue;
    const materials = Array.isArray(mesh.material) ? mesh.material : [mesh.material];
    if (materials.every(material => !material.visible || material.opacity <= 0)) continue;
    mesh.geometry.computeBoundingBox();
    if (mesh.geometry.boundingBox) bounds.union(mesh.geometry.boundingBox.clone().applyMatrix4(mesh.matrixWorld));
  }
  return bounds;
}

export function fitAssetCamera(camera: T.PerspectiveCamera, controls: OrbitControls, box: T.Box3) {
  const center = box.getCenter(new T.Vector3()), size = box.getSize(new T.Vector3());
  const radius = Math.max(size.x, size.y, size.z, 0.1);
  const damping = controls.enableDamping;
  controls.enableDamping = false;
  controls.update();
  controls.target.copy(center);
  camera.position.copy(center).add(new T.Vector3(radius * 1.1, radius * 0.5, -radius * 2));
  camera.near = Math.max(0.001, radius / 1000);
  camera.far = radius * 100;
  camera.updateProjectionMatrix();
  controls.update();
  controls.enableDamping = damping;
  return radius;
}

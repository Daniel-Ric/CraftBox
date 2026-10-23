import * as THREE from "three";



export function modelFrame(bounds: THREE.Box3, fov: number, aspect: number) {
  if (bounds.isEmpty()) return null;
  const center = bounds.getCenter(new THREE.Vector3());
  const radius = Math.max(bounds.getSize(new THREE.Vector3()).length() / 2, 1);
  const vertical = THREE.MathUtils.degToRad(fov / 2);
  const horizontal = Math.atan(Math.tan(vertical) * Math.max(aspect, 0.01));
  const distance = (radius * 1.08) / Math.sin(Math.min(vertical, horizontal));
  return { center, radius, distance, far: Math.max(4000, distance * 15) };
}

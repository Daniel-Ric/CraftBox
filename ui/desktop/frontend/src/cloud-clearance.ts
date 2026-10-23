import * as THREE from "three";



export async function buildCloudClearance(
  terrain: THREE.Object3D,
  cancelled = () => false,
) {
  if (cancelled()) return;
  terrain.updateWorldMatrix(true, true);
  const bounds = new THREE.Box3().setFromObject(terrain);
  if (bounds.isEmpty()) return;
  const size = 128,
    cell =
      (Math.max(
        0.25,
        bounds.max.x - bounds.min.x,
        bounds.max.z - bounds.min.z,
      ) +
        256) /
      (size - 4);
  const center = bounds.getCenter(new THREE.Vector3());
  const min = new THREE.Vector2(
    center.x - (size * cell) / 2,
    center.z - (size * cell) / 2,
  );
  const distance = new Float32Array(size * size).fill(10000);
  const meshes: THREE.Mesh[] = [];
  terrain.traverse((o) => {
    const mesh = o as THREE.Mesh;
    if (mesh.isMesh && mesh.geometry.getAttribute("position"))
      meshes.push(mesh);
  });
  const a = new THREE.Vector3(),
    b = new THREE.Vector3(),
    c = new THREE.Vector3();
  let count = 0;
  for (const mesh of meshes) {
    const positions = mesh.geometry.getAttribute("position"),
      indices = mesh.geometry.index;
    const end = indices ? indices.count : positions.count;
    for (let i = 0; i + 2 < end; i += 3) {
      a.fromBufferAttribute(
        positions,
        indices ? indices.getX(i) : i,
      ).applyMatrix4(mesh.matrixWorld);
      b.fromBufferAttribute(
        positions,
        indices ? indices.getX(i + 1) : i + 1,
      ).applyMatrix4(mesh.matrixWorld);
      c.fromBufferAttribute(
        positions,
        indices ? indices.getX(i + 2) : i + 2,
      ).applyMatrix4(mesh.matrixWorld);
      const ax = (a.x - min.x) / cell,
        az = (a.z - min.y) / cell,
        bx = (b.x - min.x) / cell,
        bz = (b.z - min.y) / cell,
        cx = (c.x - min.x) / cell,
        cz = (c.z - min.y) / cell;
      const x0 = Math.max(0, Math.floor(Math.min(ax, bx, cx))),
        x1 = Math.min(size - 1, Math.floor(Math.max(ax, bx, cx)));
      const z0 = Math.max(0, Math.floor(Math.min(az, bz, cz))),
        z1 = Math.min(size - 1, Math.floor(Math.max(az, bz, cz)));
      const cross = (
        x: number,
        z: number,
        px: number,
        pz: number,
        qx: number,
        qz: number,
      ) => (x - px) * (qz - pz) - (z - pz) * (qx - px);
      const tolerance =
        Math.max(
          Math.hypot(bx - ax, bz - az),
          Math.hypot(cx - bx, cz - bz),
          Math.hypot(ax - cx, az - cz),
        ) * 0.72;
      for (let z = z0; z <= z1; z++)
        for (let x = x0; x <= x1; x++) {
          if (x1 - x0 <= 1 && z1 - z0 <= 1) {
            distance[x + z * size] = 0;
            continue;
          }
          const u = cross(x + 0.5, z + 0.5, ax, az, bx, bz),
            v = cross(x + 0.5, z + 0.5, bx, bz, cx, cz),
            w = cross(x + 0.5, z + 0.5, cx, cz, ax, az);
          if (
            (u >= -tolerance && v >= -tolerance && w >= -tolerance) ||
            (u <= tolerance && v <= tolerance && w <= tolerance)
          )
            distance[x + z * size] = 0;
        }
      if (++count % 16384 === 0) {
        await new Promise((resolve) => setTimeout(resolve, 0));
        if (cancelled()) return;
      }
    }
  }
  const relax = (
    x: number,
    z: number,
    dx: number,
    dz: number,
    cost: number,
  ) => {
    if (x + dx >= 0 && x + dx < size && z + dz >= 0 && z + dz < size)
      distance[x + z * size] = Math.min(
        distance[x + z * size],
        distance[x + dx + (z + dz) * size] + cost,
      );
  };
  for (let z = 0; z < size; z++)
    for (let x = 0; x < size; x++) {
      relax(x, z, -1, 0, 1);
      relax(x, z, 0, -1, 1);
      relax(x, z, -1, -1, Math.SQRT2);
      relax(x, z, 1, -1, Math.SQRT2);
    }
  for (let z = size - 1; z >= 0; z--)
    for (let x = size - 1; x >= 0; x--) {
      relax(x, z, 1, 0, 1);
      relax(x, z, 0, 1, 1);
      relax(x, z, 1, 1, Math.SQRT2);
      relax(x, z, -1, 1, Math.SQRT2);
    }
  const maxDistance = cell * size;
  const data = Uint8Array.from(distance, (d) =>
    Math.round(Math.min(1, d / size) * 255),
  );
  const texture = new THREE.DataTexture(
    data,
    size,
    size,
    THREE.RedFormat,
    THREE.UnsignedByteType,
  );
  texture.minFilter = texture.magFilter = THREE.LinearFilter;
  texture.needsUpdate = true;
  return {
    texture,
    min,
    extent: size * cell,
    bottom: bounds.min.y,
    maxDistance,
    distanceAt(x: number, z: number) {
      const ix = Math.floor((x - min.x) / cell),
        iz = Math.floor((z - min.y) / cell);
      return ix < 0 || iz < 0 || ix >= size || iz >= size
        ? Infinity
        : distance[ix + iz * size] * cell;
    },
  };
}
export type CloudClearance = NonNullable<
  Awaited<ReturnType<typeof buildCloudClearance>>
>;

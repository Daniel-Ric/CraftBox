

export function billboardAttributes(positions: Float32Array<ArrayBuffer>) {
  if (positions.length % 12 !== 0) throw new Error("Invalid hologram");
  const centers = new Float32Array(positions.length);
  const offsets = new Float32Array(positions.length / 3 * 2);
  for (let start = 0; start < positions.length; start += 12) {
    const cx = (positions[start] + positions[start + 6]) / 2;
    const cy = (positions[start + 1] + positions[start + 7]) / 2;
    const cz = (positions[start + 2] + positions[start + 8]) / 2;
    const dx = positions[start + 3] - positions[start];
    const dz = positions[start + 5] - positions[start + 2];
    const width = Math.hypot(dx, dz);
    for (let i = 0; i < 4; i++) {
      const p = start + i * 3;
      centers.set([cx, cy, cz], p);
      offsets[p / 3 * 2] = width ? ((positions[p] - cx) * dx + (positions[p + 2] - cz) * dz) / width : 0;
      offsets[p / 3 * 2 + 1] = positions[p + 1] - cy;
    }
  }
  return { centers, offsets };
}

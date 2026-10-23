import type { AtlasPatch, PackedChunk, PackedPart, ScenePart } from "./bridge";

export type DecodedPart = {
  mode: ScenePart["mode"];
  positions: Float32Array<ArrayBuffer>;
  normals: Int8Array<ArrayBuffer>;
  uvs: Float32Array<ArrayBuffer>;
  indices: Uint32Array<ArrayBuffer>;
  bounds: [number, number, number, number, number, number];
};
export type DecodedChunk = Omit<PackedChunk, "parts"> & {
  parts: DecodedPart[];
};
export type DecodedPatch = Omit<AtlasPatch, "data"> & {
  pixels: Uint8Array<ArrayBuffer>;
};
export type DecodedBatch = { chunks: DecodedChunk[]; patches: DecodedPatch[] };

function decodeBase64(data: string): ArrayBuffer {
  const text = atob(data),
    bytes = new Uint8Array(text.length);
  for (let i = 0; i < text.length; i++) bytes[i] = text.charCodeAt(i);
  return bytes.buffer;
}
export function decodePart(part: PackedPart): DecodedPart {
  if (
    !Number.isSafeInteger(part.vertices) ||
    !Number.isSafeInteger(part.indices) ||
    part.vertices <= 0 ||
    part.vertices > 4_000_000 ||
    part.indices < 0 ||
    part.indices % 3 !== 0
  )
    throw new Error("Invalid 3D buffer size");
  const buffer = decodeBase64(part.data),
    normalOffset = part.vertices * 12,
    uvOffset = (normalOffset + part.vertices * 3 + 3) & ~3,
    indexOffset = uvOffset + part.vertices * 8;
  if (buffer.byteLength !== indexOffset + part.indices * 4)
    throw new Error("Incomplete 3D buffer");
  const positions = new Float32Array(buffer, 0, part.vertices * 3),
    normals = new Int8Array(buffer, normalOffset, part.vertices * 3),
    uvs = new Float32Array(buffer, uvOffset, part.vertices * 2),
    indices = new Uint32Array(buffer, indexOffset, part.indices);
  const bounds: [number, number, number, number, number, number] = [
    Infinity,
    Infinity,
    Infinity,
    -Infinity,
    -Infinity,
    -Infinity,
  ];
  for (let i = 0; i < positions.length; i++) {
    const v = positions[i];
    if (!Number.isFinite(v)) throw new Error("Invalid 3D coordinate");
    const axis = i % 3;
    bounds[axis] = Math.min(bounds[axis], v);
    bounds[axis + 3] = Math.max(bounds[axis + 3], v);
  }
  for (const index of indices)
    if (index >= part.vertices) throw new Error("Invalid 3D index");
  for (const uv of uvs)
    if (!Number.isFinite(uv)) throw new Error("Invalid texture coordinate");
  return { mode: part.mode, positions, normals, uvs, indices, bounds };
}
export function decodePatch(
  patch: AtlasPatch,
  atlasSize: number,
): DecodedPatch {
  if (
    !Number.isInteger(patch.x) ||
    !Number.isInteger(patch.y) ||
    !Number.isInteger(patch.size) ||
    patch.size < 1 ||
    patch.size > 128 ||
    patch.x < 0 ||
    patch.y < 0 ||
    patch.x + patch.size > atlasSize ||
    patch.y + patch.size > atlasSize
  )
    throw new Error("Invalid texture region");
  const pixels = new Uint8Array(decodeBase64(patch.data));
  if (pixels.length !== patch.size * patch.size * 4)
    throw new Error("Incomplete texture tile");
  return { x: patch.x, y: patch.y, size: patch.size, pixels };
}

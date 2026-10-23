import type { ChunkBounds } from "./bridge";

export function selectedChunks(
  start: { x: number; z: number },
  end: { x: number; z: number },
): ChunkBounds {
  return {
    minX: Math.floor(Math.min(start.x, end.x) / 16),
    minZ: Math.floor(Math.min(start.z, end.z) / 16),
    maxX: Math.floor(Math.max(start.x, end.x) / 16),
    maxZ: Math.floor(Math.max(start.z, end.z) / 16),
  };
}

export function worldAtPointer(
  x: number,
  y: number,
  width: number,
  height: number,
  view: { x: number; z: number; scale: number },
) {
  return {
    x: view.x + (x - width / 2) / view.scale,
    z: view.z + (y - height / 2) / view.scale,
  };
}

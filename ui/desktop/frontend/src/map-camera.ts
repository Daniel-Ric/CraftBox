import type { Tile } from "./bridge";
export function fitTiles(tiles: Iterable<Tile>, width: number, height: number) {
  const list = [...tiles];
  const centre = centerOnTiles(list);
  let minX = Infinity,
    maxX = -Infinity,
    minZ = Infinity,
    maxZ = -Infinity;
  for (const tile of list) {
    minX = Math.min(minX, tile.x);
    maxX = Math.max(maxX, tile.x);
    minZ = Math.min(minZ, tile.z);
    maxZ = Math.max(maxZ, tile.z);
  }
  const spanX = list.length ? (maxX - minX + 1) * 16 : 16;
  const spanZ = list.length ? (maxZ - minZ + 1) * 16 : 16;
  return {
    ...centre,
    scale: Math.min(
      2,
      Math.max(0.001, (width - 64) / spanX),
      Math.max(0.001, (height - 64) / spanZ),
    ),
  };
}
export function centerOnTiles(tiles: Iterable<Tile>) {
  let minX = Infinity,
    maxX = -Infinity,
    minZ = Infinity,
    maxZ = -Infinity;
  for (const tile of tiles) {
    minX = Math.min(minX, tile.x);
    maxX = Math.max(maxX, tile.x);
    minZ = Math.min(minZ, tile.z);
    maxZ = Math.max(maxZ, tile.z);
  }
  return Number.isFinite(minX)
    ? { x: (minX + maxX + 1) * 8, z: (minZ + maxZ + 1) * 8 }
    : { x: 0, z: 0 };
}

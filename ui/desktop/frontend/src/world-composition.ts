import type { ChunkBounds, WorldPlacement } from "./bridge";
export type Placement = WorldPlacement & { id: string; color: string };
export function placedBounds(
  p: WorldPlacement,
  source: ChunkBounds,
): [number, number, number, number] {
  const b = p.bounds ?? source;
  return [
    b.minX * 16 + p.offset[0],
    b.minZ * 16 + p.offset[2],
    (b.maxX + 1) * 16 + p.offset[0],
    (b.maxZ + 1) * 16 + p.offset[2],
  ];
}
export function arrangePlacements(
  parts: Placement[],
  bounds: Record<string, ChunkBounds>,
  gap: number,
  direction: "east" | "south" = "east",
) {
  let cursor = 0;
  return parts.map((p) => {
    const b = p.bounds ??
        bounds[p.worldId] ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 },
      width = (b.maxX - b.minX + 1) * 16,
      depth = (b.maxZ - b.minZ + 1) * 16;
    const offset: [number, number, number] = [
      direction === "east" ? cursor - b.minX * 16 : -b.minX * 16,
      p.offset[1],
      direction === "south" ? cursor - b.minZ * 16 : -b.minZ * 16,
    ];
    cursor += (direction === "east" ? width : depth) + Math.max(0, gap);
    return { ...p, offset };
  });
}

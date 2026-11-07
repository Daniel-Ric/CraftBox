import type { Tile, ChunkBounds } from "./bridge";
export type MapView = {
  x: number;
  z: number;
  scale: number;
  rotation?: number;
};
export type MapLayer = "surface" | "height" | "biome";
export type MapStyle = {
  elevationMin?: number;
  elevationMax?: number;
  layer: MapLayer;
  detail: "simple" | "textures";
  relief: boolean;
  contours: boolean;
  contourStep: number;
  grid: boolean;
  coverage: boolean;
  recent: boolean;
  saturation: number;
  heightTint: number;
  rotation: number;
  gridSize: number;
  gridOpacity: number;
};
export function normalizeMapStyle(raw: unknown): MapStyle {
  const out = { ...mapStyleDefaults };
  if (!raw || typeof raw !== "object") return out;
  const value = raw as Record<string, unknown>;
  if (["surface", "height", "biome"].includes(String(value.layer)))
    out.layer = value.layer as MapLayer;
  if (value.detail === "simple" || value.detail === "textures")
    out.detail = value.detail;
  for (const key of [
    "relief",
    "contours",
    "grid",
    "coverage",
    "recent",
  ] as const)
    if (typeof value[key] === "boolean") out[key] = value[key] as boolean;
  for (const [key, min, max] of [
    ["contourStep", 1, 128],
    ["rotation", 0, 360],
    ["gridSize", 1, 4096],
    ["gridOpacity", 0, 1],
    ["saturation", 0, 2],
    ["heightTint", 0, 1],
  ] as const) {
    const v = value[key];
    if (typeof v === "number" && Number.isFinite(v))
      out[key] = Math.max(min, Math.min(max, v));
  }
  return out;
}
export function normalizeSlice(raw: unknown) {
  return typeof raw === "number" &&
    Number.isInteger(raw) &&
    raw >= -32768 &&
    raw <= 32767
    ? raw
    : undefined;
}
export const mapStyleDefaults: MapStyle = {
  layer: "surface",
  detail: "textures",
  relief: true,
  contours: false,
  contourStep: 16,
  grid: false,
  coverage: false,
  recent: true,
  saturation: 1,
  heightTint: 0,
  rotation: 0,
  gridSize: 16,
  gridOpacity: 0.3,
};
export function pointAt(
  x: number,
  y: number,
  w: number,
  h: number,
  v: MapView,
) {
  const a = ((v.rotation ?? 0) * Math.PI) / 180,
    dx = (x - w / 2) / v.scale,
    dz = (y - h / 2) / v.scale;
  return {
    x: v.x + dx * Math.cos(a) + dz * Math.sin(a),
    z: v.z - dx * Math.sin(a) + dz * Math.cos(a),
  };
}
export function screenAt(
  x: number,
  z: number,
  w: number,
  h: number,
  v: MapView,
) {
  const a = ((v.rotation ?? 0) * Math.PI) / 180,
    dx = x - v.x,
    dz = z - v.z;
  return {
    x: w / 2 + (dx * Math.cos(a) - dz * Math.sin(a)) * v.scale,
    y: h / 2 + (dx * Math.sin(a) + dz * Math.cos(a)) * v.scale,
  };
}
export function zoomAt(
  v: MapView,
  f: number,
  x: number,
  y: number,
  w: number,
  h: number,
) {
  const p = pointAt(x, y, w, h, v),
    next = { ...v, scale: Math.max(0.001, Math.min(64, v.scale * f)) },
    q = pointAt(x, y, w, h, next);
  return { ...next, x: next.x + p.x - q.x, z: next.z + p.z - q.z };
}
export function normalBounds(b: ChunkBounds): ChunkBounds {
  return {
    minX: Math.floor(Math.min(b.minX, b.maxX)),
    maxX: Math.floor(Math.max(b.minX, b.maxX)),
    minZ: Math.floor(Math.min(b.minZ, b.maxZ)),
    maxZ: Math.floor(Math.max(b.minZ, b.maxZ)),
  };
}
export function mapCoverage(
  tiles: Iterable<{ x: number; z: number }>,
  b: ChunkBounds,
) {
  let captured = 0;
  const seen = new Set<string>();
  for (const t of tiles)
    if (t.x >= b.minX && t.x <= b.maxX && t.z >= b.minZ && t.z <= b.maxZ)
      seen.add(`${t.x},${t.z}`);
  captured = seen.size;
  const total = (b.maxX - b.minX + 1) * (b.maxZ - b.minZ + 1);
  return { captured, total, missing: Math.max(0, total - captured) };
}
export function visibleTile(
  t: { x: number; z: number },
  v: MapView,
  w: number,
  h: number,
) {
  const p = screenAt(t.x * 16 + 8, t.z * 16 + 8, w, h, v),
    pad = 16 * v.scale;
  return p.x >= -pad && p.x <= w + pad && p.y >= -pad && p.y <= h + pad;
}
export function niceScale(scale: number) {
  const value = 100 / scale,
    p = 10 ** Math.floor(Math.log10(value));
  return ([5, 2, 1].find((x) => x * p <= value) ?? 1) * p;
}
export function terrainBounds(
  tiles: Iterable<{ x: number; z: number }>,
): ChunkBounds | undefined {
  let minX = Infinity,
    minZ = Infinity,
    maxX = -Infinity,
    maxZ = -Infinity;
  for (const t of tiles) {
    minX = Math.min(minX, t.x);
    minZ = Math.min(minZ, t.z);
    maxX = Math.max(maxX, t.x);
    maxZ = Math.max(maxZ, t.z);
  }
  return Number.isFinite(minX) ? { minX, minZ, maxX, maxZ } : undefined;
}
export function biomeColor(id: number): [number, number, number] {
  const hue = (id * 137.508 + 95) % 360,
    c = 0.52,
    x = c * (1 - Math.abs(((hue / 60) % 2) - 1)),
    m = 0.28;
  const rgb =
    hue < 60
      ? [c, x, 0]
      : hue < 120
        ? [x, c, 0]
        : hue < 180
          ? [0, c, x]
          : hue < 240
            ? [0, x, c]
            : hue < 300
              ? [x, 0, c]
              : [c, 0, x];
  return rgb.map((v) => Math.round((v + m) * 255)) as [number, number, number];
}
export function heightColor(
  y: number,
  min = -64,
  max = 320,
): [number, number, number] {
  const t = Math.max(0, Math.min(1, (y - min) / Math.max(1, max - min)));
  return [
    Math.round(48 + 190 * t),
    Math.round(110 + 100 * t),
    Math.round(105 + 100 * t),
  ];
}
export function metadataAt(tiles: Map<string, Tile>, x: number, z: number) {
  const cx = Math.floor(x / 16),
    cz = Math.floor(z / 16),
    t = tiles.get(`${cx},${cz}`),
    i =
      (((Math.floor(z) % 16) + 16) % 16) * 16 +
      (((Math.floor(x) % 16) + 16) % 16);
  return {
    height: t?.heights?.[i],
    biome: t?.biomes?.[i],
    biomeName:
      t?.biomes?.[i] === undefined
        ? undefined
        : t.biomeNames?.[String(t.biomes[i])],
    block: t?.blocks?.[i],
    tile: t,
  };
}

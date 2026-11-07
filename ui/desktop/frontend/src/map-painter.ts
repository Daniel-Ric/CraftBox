import { styleMapPixels } from "./map-tile-style";
export { styleMapPixels } from "./map-tile-style";
import type { Tile, ChunkBounds } from "./bridge";
import type { MapMarker } from "./map-presentation";
import {
  screenAt,
  visibleTile,
  niceScale,
  type MapStyle,
  type MapView,
} from "./map-workbench";
export class MapImages {
  images = new Map<
    string,
    { source: string; image: HTMLImageElement; revision: number }
  >();
  styled = new Map<
    string,
    { key: string; source: string; canvas: HTMLCanvasElement }
  >();
  serial = 0;
  private spatialTiles?: Map<string, Tile>;
  private spatialGroups = new Map<number, Map<string, Tile[]>>();
  private signatures = new WeakMap<
    Tile[],
    { style: string; serial: number; tiles: Map<string, Tile>; key: string }
  >();
  visibleGroups(
    tiles: Map<string, Tile>,
    v: MapView,
    w: number,
    h: number,
    span: number,
  ) {
    if (this.spatialTiles !== tiles) {
      this.spatialTiles = tiles;
      this.spatialGroups.clear();
    }
    let groups = this.spatialGroups.get(span);
    if (!groups) {
      groups = new Map();
      this.spatialGroups.set(span, groups);
      for (const t of tiles.values()) {
        const k = `${Math.floor(t.x / span)},${Math.floor(t.z / span)}`;
        let group = groups.get(k);
        if (!group) groups.set(k, (group = []));
        group.push(t);
      }
    }
    const a = ((v.rotation ?? 0) * Math.PI) / 180,
      unit = span * 16;
    const rx =
      (Math.abs(w * Math.cos(a)) + Math.abs(h * Math.sin(a))) / v.scale / 2 +
      16;
    const rz =
      (Math.abs(w * Math.sin(a)) + Math.abs(h * Math.cos(a))) / v.scale / 2 +
      16;
    const minX = Math.floor((v.x - rx) / unit),
      maxX = Math.floor((v.x + rx) / unit),
      minZ = Math.floor((v.z - rz) / unit),
      maxZ = Math.floor((v.z + rz) / unit);
    const out = new Map<string, Tile[]>();
    if ((maxX - minX + 1) * (maxZ - minZ + 1) < groups.size * 2) {
      for (let x = minX; x <= maxX; x++)
        for (let z = minZ; z <= maxZ; z++) {
          const k = `${x},${z}`,
            g = groups.get(k);
          if (g) out.set(k, g);
        }
    } else
      for (const [k, g] of groups) {
        const [x, z] = k.split(",").map(Number);
        if (x >= minX && x <= maxX && z >= minZ && z <= maxZ) out.set(k, g);
      }
    return out;
  }
  groupSignature(list: Tile[], tiles: Map<string, Tile>, style: string) {
    const cached = this.signatures.get(list);
    if (
      cached?.style === style &&
      cached.serial === this.serial &&
      cached.tiles === tiles
    )
      return cached.key;
    const key = JSON.stringify([
      style,
      ...list.map((t) => [
        t.x,
        t.z,
        t.version,
        this.images.get(`${t.x},${t.z}`)?.revision,
        tiles.get(`${t.x - 1},${t.z}`)?.version,
        tiles.get(`${t.x},${t.z - 1}`)?.version,
      ]),
    ]);
    this.signatures.set(list, { style, serial: this.serial, tiles, key });
    return key;
  }
  overview = new Map<
    string,
    { key: string; canvas: HTMLCanvasElement; next: number }
  >();
  dispose() {
    for (const item of this.images.values()) item.image.onload = null;
    this.images.clear();
    this.styled.clear();
    this.overview.clear();
    this.spatialTiles = undefined;
    this.spatialGroups.clear();
    this.signatures = new WeakMap();
  }
  sync(tiles: Map<string, Tile>, onLoad: () => void) {
    for (const [k, t] of tiles) {
      if (this.images.get(k)?.source === t.image) continue;
      const image = new Image();
      image.onload = () => {
        const entry = this.images.get(k);
        if (entry?.image !== image) return;
        entry.revision = ++this.serial;
        this.styled.delete(k);
        onLoad();
      };
      image.src = t.image;
      this.images.set(k, { source: t.image, image, revision: ++this.serial });
    }
    for (const k of this.images.keys())
      if (!tiles.has(k)) {
        this.images.delete(k);
        this.styled.delete(k);
      }
  }
  tile(t: Tile, tiles: Map<string, Tile>, s: MapStyle) {
    const key = `${t.x},${t.z}`,
      img = this.images.get(key)?.image;
    if (!img?.complete || !img.naturalWidth) return;
    const neighbours = [
      [-1, 0],
      [1, 0],
      [0, -1],
      [0, 1],
    ].map(([x, z]) => tiles.get(`${t.x + x},${t.z + z}`)?.version ?? 0);
    const range =
      s.layer === "height" || (s.layer === "surface" && s.heightTint > 0);
    const signature = JSON.stringify([
      t.version,
      s.detail,
      s.layer,
      s.relief,
      s.contours,
      s.contourStep,
      s.saturation,
      s.heightTint,
      range ? s.elevationMin : 0,
      range ? s.elevationMax : 0,
      neighbours,
    ]);
    const cached = this.styled.get(key);
    if (cached?.key === signature && cached.source === t.image)
      return cached.canvas;
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height =
      s.detail === "simple" ? 16 : img.naturalWidth;
    const ctx = canvas.getContext("2d", { willReadFrequently: true });
    if (!ctx) return;
    ctx.imageSmoothingEnabled = true;
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    const data = ctx.getImageData(0, 0, canvas.width, canvas.height);
    styleMapPixels(data, t, tiles, s);
    ctx.putImageData(data, 0, 0);
    if (this.styled.size > 512)
      this.styled.delete(this.styled.keys().next().value!);
    this.styled.set(key, { key: signature, source: t.image, canvas });
    return canvas;
  }
}
export type MapDraw = {
  frameBudgetMs?: number;
  clear?: boolean;
  metadataTiles?: Map<string, Tile>;
  tiles: Map<string, Tile>;
  view: MapView;
  style: MapStyle;
  width: number;
  height: number;
  selection?: ChunkBounds;
  markers: MapMarker[];
  measure: { x: number; z: number }[];
  player?: [number, number, number];
  spawn?: [number, number, number];
  coverage?: [number, number][];
  recent?: Map<string, number>;
  annotations: boolean;
  now?: number;
};
export function paintMap(
  ctx: CanvasRenderingContext2D,
  cache: MapImages,
  o: MapDraw,
) {
  const budget = o.frameBudgetMs ?? Infinity;
  let preparedMs = 0;
  let pending = false;
  const { width: w, height: h, view: v, style: s, tiles } = o;
  if (o.clear !== false) ctx.clearRect(0, 0, w, h);
  ctx.save();
  ctx.translate(w / 2, h / 2);
  ctx.rotate(((v.rotation ?? 0) * Math.PI) / 180);
  ctx.scale(v.scale, v.scale);
  ctx.translate(-v.x, -v.z);
  ctx.imageSmoothingEnabled = v.scale < 1;
  const visible =
    v.scale >= 0.6 || (s.recent && !!o.recent?.size)
      ? [...cache.visibleGroups(tiles, v, w, h, 8).values()]
          .flat()
          .filter((t) => visibleTile(t, v, w, h))
      : [];
  if (v.scale < 0.6) {
                                                                                
    const span = Math.max(8, 2 ** Math.ceil(Math.log2(64 / (16 * v.scale))));
    const density = Math.min(1, 2 ** Math.ceil(Math.log2(v.scale * 2)));
    const overviewStyle = { ...s, detail: "simple" as const };
    const groups = cache.visibleGroups(tiles, v, w, h, span);
    const styleKey = JSON.stringify(overviewStyle);
    for (const [key, list] of groups) {
      const [gx, gz] = key.split(",").map(Number),
        cacheKey = `${span}:${density}:${key}`,
        sig = cache.groupSignature(list, o.metadataTiles ?? tiles, styleKey);
      let group = cache.overview.get(cacheKey);
      if (group?.key !== sig) {
        const c = document.createElement("canvas");
        c.width = c.height = Math.max(1, Math.ceil(span * 16 * density));
        group = { key: sig, canvas: c, next: 0 };
        cache.overview.set(cacheKey, group);
      }
      if (group.next < list.length) {
        const started = performance.now();
        const c = group.canvas;
        const g = c.getContext("2d");
        if (!g) continue;
        g.imageSmoothingEnabled = true;
        for (; group.next < list.length; group.next++) {
          if (preparedMs + performance.now() - started >= budget) {
            pending = true;
            break;
          }
          const t = list[group.next];
          const tile = cache.tile(t, o.metadataTiles ?? tiles, overviewStyle);
          if (tile)
            g.drawImage(
              tile,
              (t.x - gx * span) * 16 * density,
              (t.z - gz * span) * 16 * density,
              16 * density,
              16 * density,
            );
        }
        preparedMs += performance.now() - started;
      }
      ctx.drawImage(
        group.canvas,
        gx * span * 16,
        gz * span * 16,
        span * 16,
        span * 16,
      );
    }
                                                                               
    let bytes = 0;
    const retained = new Set(
      [...groups.keys()].map((k) => `${span}:${density}:${k}`),
    );
    for (const group of cache.overview.values())
      bytes += group.canvas.width * group.canvas.height * 4;
    for (const [k, group] of cache.overview) {
      if (bytes <= 32 * 1024 * 1024) break;
      if (!retained.has(k)) {
        bytes -= group.canvas.width * group.canvas.height * 4;
        cache.overview.delete(k);
      }
    }
  } else
    for (const t of visible) {
      const image = cache.tile(t, o.metadataTiles ?? tiles, s);
      if (image) ctx.drawImage(image, t.x * 16, t.z * 16, 16, 16);
    }
  if (o.annotations) {
    if (o.recent)
      for (const [key, time] of o.recent)
        if ((o.now ?? Date.now()) - time >= 2500) o.recent.delete(key);
    if (s.coverage) {
      const known = new Set(
        (o.coverage ?? [...tiles.values()].map((t) => [t.x, t.z])).map((p) =>
          p.join(","),
        ),
      );
      const r = Math.hypot(w, h) / v.scale / 2,
        minX = Math.floor((v.x - r) / 16),
        maxX = Math.ceil((v.x + r) / 16),
        minZ = Math.floor((v.z - r) / 16),
        maxZ = Math.ceil((v.z + r) / 16);
      if ((maxX - minX) * (maxZ - minZ) < 4096)
        for (let x = minX; x <= maxX; x++)
          for (let z = minZ; z <= maxZ; z++) {
            ctx.strokeStyle = known.has(`${x},${z}`) ? "#74b88a" : "#d89a6a";
            ctx.globalAlpha = known.has(`${x},${z}`) ? 0.35 : 0.2;
            ctx.lineWidth = 1 / v.scale;
            ctx.strokeRect(x * 16, z * 16, 16, 16);
            if (!known.has(`${x},${z}`)) {
              ctx.beginPath();
              ctx.moveTo(x * 16, z * 16);
              ctx.lineTo(x * 16 + 16, z * 16 + 16);
              ctx.stroke();
            }
          }
      ctx.globalAlpha = 1;
    }
    if (s.recent && o.recent)
      for (const t of visible) {
        const age =
          (o.now ?? Date.now()) - (o.recent.get(`${t.x},${t.z}`) ?? 0);
        if (age >= 0 && age < 2500) {
          ctx.fillStyle = `rgba(60,180,225,${0.22 * (1 - age / 2500)})`;
          ctx.fillRect(t.x * 16, t.z * 16, 16, 16);
        }
      }
    if (s.grid) {
      const step = Math.max(
          s.gridSize,
          2 ** Math.ceil(Math.log2(36 / v.scale)),
        ),
        r = Math.hypot(w, h) / v.scale / 2;
      ctx.beginPath();
      for (
        let x = Math.floor((v.x - r) / step) * step;
        x < v.x + r;
        x += step
      ) {
        ctx.moveTo(x, v.z - r);
        ctx.lineTo(x, v.z + r);
      }
      for (
        let z = Math.floor((v.z - r) / step) * step;
        z < v.z + r;
        z += step
      ) {
        ctx.moveTo(v.x - r, z);
        ctx.lineTo(v.x + r, z);
      }
      ctx.strokeStyle = `rgba(160,175,185,${s.gridOpacity})`;
      ctx.lineWidth = 1 / v.scale;
      ctx.stroke();
    }
    if (o.selection) {
      const b = o.selection,
        x = b.minX * 16,
        z = b.minZ * 16,
        sw = (b.maxX - b.minX + 1) * 16,
        sh = (b.maxZ - b.minZ + 1) * 16;
      ctx.fillStyle = "rgba(75,164,218,.17)";
      ctx.fillRect(x, z, sw, sh);
      ctx.strokeStyle = "#62b6e8";
      ctx.lineWidth = 2 / v.scale;
      ctx.strokeRect(x, z, sw, sh);
      for (const [p, q] of [
        [x, z],
        [x + sw, z],
        [x, z + sh],
        [x + sw, z + sh],
      ]) {
        ctx.fillStyle = "#b9def4";
        ctx.fillRect(
          p - 4 / v.scale,
          q - 4 / v.scale,
          8 / v.scale,
          8 / v.scale,
        );
      }
    }
  }
  ctx.restore();
  if (!o.annotations) return pending;
  const text = (label: string, x: number, y: number, color = "#eee") => {
    ctx.font = "12px Segoe UI, sans-serif";
    ctx.lineWidth = 4;
    ctx.strokeStyle = "#202020";
    ctx.strokeText(label, x, y);
    ctx.fillStyle = color;
    ctx.fillText(label, x, y);
  };
  for (const m of o.markers) {
    const p = screenAt(m.x, m.z, w, h, v);
    ctx.save();
    ctx.translate(p.x, p.y);
    ctx.scale(
      Math.max(0.25, Math.min(4, m.scale)),
      Math.max(0.25, Math.min(4, m.scale)),
    );
    ctx.font = "12px Segoe UI, sans-serif";
    const label = `${m.icon ? m.icon + " " : ""}${m.label}`,
      labelWidth = ctx.measureText(label).width;
    ctx.fillStyle = m.background;
    ctx.fillRect(7, -12, labelWidth + 10, 21);
    ctx.beginPath();
    ctx.arc(0, 0, 5, 0, Math.PI * 2);
    ctx.fillStyle = m.color;
    ctx.fill();
    text(label, 12, 3, m.color);
    ctx.restore();
  }
  if (o.measure.length) {
    ctx.strokeStyle = "#f1cf78";
    ctx.lineWidth = 2;
    ctx.beginPath();
    o.measure.forEach((p, i) => {
      const q = screenAt(p.x, p.z, w, h, v);
      if (i) ctx.lineTo(q.x, q.y);
      else ctx.moveTo(q.x, q.y);
    });
    ctx.stroke();
    for (const p of o.measure) {
      const q = screenAt(p.x, p.z, w, h, v);
      ctx.beginPath();
      ctx.arc(q.x, q.y, 4, 0, Math.PI * 2);
      ctx.fillStyle = "#f1cf78";
      ctx.fill();
    }
  }
  if (o.player) {
    const p = screenAt(o.player[0], o.player[2], w, h, v);
    ctx.beginPath();
    ctx.arc(p.x, p.y, 6, 0, Math.PI * 2);
    ctx.fillStyle = "#43d1aa";
    ctx.fill();
    ctx.strokeStyle = "#fff";
    ctx.lineWidth = 2;
    ctx.stroke();
  }
  if (o.spawn) {
    const p = screenAt(o.spawn[0], o.spawn[2], w, h, v);
    text("⚑", p.x, p.y, "#f1cf78");
  }
  const units = niceScale(v.scale),
    sx = 18,
    sy = h - 55;
  ctx.strokeStyle = "#eee";
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(sx, sy - 4);
  ctx.lineTo(sx, sy);
  ctx.lineTo(sx + units * v.scale, sy);
  ctx.lineTo(sx + units * v.scale, sy - 4);
  ctx.stroke();
  text(`${units.toLocaleString("en-US")} blocks`, sx, sy - 9);
  const a = ((v.rotation ?? 0) * Math.PI) / 180,
    nx = w - 26,
    ny = 94;
  ctx.strokeStyle = "#eee";
  ctx.beginPath();
  ctx.moveTo(nx - Math.sin(a) * 9, ny + Math.cos(a) * 9);
  ctx.lineTo(nx + Math.sin(a) * 9, ny - Math.cos(a) * 9);
  ctx.stroke();
  text("N", nx + Math.sin(a) * 17 - 4, ny - Math.cos(a) * 17 + 4);
  return pending;
}

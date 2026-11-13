import type { BlockbenchModel } from "./blockbench-project";

type Rect = { x: number; y: number; width: number; height: number };
type Point = { x: number; y: number };
type Layout = { width: number; height: number; positions: Point[] };
type Data = Record<string, any>;
type Patch = Rect & { masks: Rect[]; move: ((dx: number, dy: number, height: number) => void)[] };
const contains = (a: Rect, b: Rect) => a.x <= b.x && a.y <= b.y && a.x + a.width >= b.x + b.width && a.y + a.height >= b.y + b.height;
const overlaps = (a: Rect, b: Rect) => a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;

export function packTextureRegions(regions: Rect[], originalWidth: number, originalHeight: number): Layout {
  const original = { width: originalWidth, height: originalHeight, positions: regions.map(({ x, y }) => ({ x, y })) };
  if (!regions.length || regions.length > 2048) return original;
  const padding = 1, area = regions.reduce((sum, r) => sum + (r.width + padding * 2) * (r.height + padding * 2), 0);
  const minWidth = Math.max(...regions.map(r => r.width + 2)), minHeight = Math.max(...regions.map(r => r.height + 2));
  const widths = [...new Set([minWidth, originalWidth, ...[0.8, 1, 1.25, 1.6].map(scale => Math.max(minWidth, Math.ceil(Math.sqrt(area) * scale)))])].filter(width => width <= 8192);
  let best = original, score = originalWidth * originalHeight;
  const ordered = regions.map((region, index) => ({ region, index })).sort((a, b) => b.region.width * b.region.height - a.region.width * a.region.height);
  for (const width of widths) {
    for (let attempt = 0; attempt < 4; attempt++) {
      const height = Math.min(8192, Math.max(minHeight, Math.ceil(area / width * 1.25 * 1.5 ** attempt)));
      let free: Rect[] = [{ x: 0, y: 0, width, height }], usedWidth = 0, usedHeight = 0;
      const positions: Point[] = [];
      for (const { region, index } of ordered) {
        const w = region.width + 2, h = region.height + 2;
        const candidates = free.filter(rect => rect.width >= w && rect.height >= h).sort((a, b) => Math.min(a.width - w, a.height - h) - Math.min(b.width - w, b.height - h) || a.width * a.height - b.width * b.height);
        if (!candidates.length) break;
        const placed = { x: candidates[0].x, y: candidates[0].y, width: w, height: h };
        positions[index] = { x: placed.x + 1, y: placed.y + 1 };
        usedWidth = Math.max(usedWidth, placed.x + w); usedHeight = Math.max(usedHeight, placed.y + h);
        free = free.flatMap(rect => {
          if (!overlaps(rect, placed)) return [rect];
          const next: Rect[] = [];
          if (placed.x > rect.x) next.push({ ...rect, width: placed.x - rect.x });
          if (placed.x + w < rect.x + rect.width) next.push({ ...rect, x: placed.x + w, width: rect.x + rect.width - placed.x - w });
          if (placed.y > rect.y) next.push({ ...rect, height: placed.y - rect.y });
          if (placed.y + h < rect.y + rect.height) next.push({ ...rect, y: placed.y + h, height: rect.y + rect.height - placed.y - h });
          return next;
        });
        free = free.filter((rect, i) => !free.some((other, j) => j !== i && contains(other, rect) && (other.width * other.height > rect.width * rect.height || j < i)));
      }
      if (positions.filter(Boolean).length !== regions.length) continue;
      const nextArea = usedWidth * usedHeight;
      const nextScore = nextArea * (1 + Math.abs(Math.log2(usedWidth / usedHeight)) * 0.08);
      if (nextArea < originalWidth * originalHeight && Math.max(usedWidth, usedHeight) <= Math.max(originalWidth, originalHeight) && nextScore < score) { best = { width: usedWidth, height: usedHeight, positions }; score = nextScore; }
      break;
    }
  }
  return best;
}

export async function compactModelTextures(model: BlockbenchModel): Promise<BlockbenchModel> {
  const copy = structuredClone(model), raw = copy.geometry as Data;
  const modern = raw["minecraft:geometry"];
  const key = copy.geometryId ?? Object.keys(raw).find(key => key.startsWith("geometry."));
  const geometry = (Array.isArray(modern) ? modern.find(g => !copy.geometryId || g.description?.identifier === copy.geometryId) : raw.bones ? raw : key ? raw[key] : undefined) as Data | undefined;
  if (!geometry) return model;
  const description = geometry.description ?? geometry;
  const width = description.texture_width ?? geometry.texturewidth ?? raw.texturewidth ?? 64;
  const height = description.texture_height ?? geometry.textureheight ?? raw.textureheight ?? 64;
  const patches: Patch[] = [];
  let unsupported = false;
  const rectangle = (u: number, v: number, w: number, h: number): Rect => {
    const x = Math.min(width - 1, Math.floor(Math.min(u, u + w))), y = Math.min(height - 1, Math.floor(Math.min(v, v + h)));
    return { x, y, width: Math.max(1, Math.ceil(Math.max(u, u + w)) - x), height: Math.max(1, Math.ceil(Math.max(v, v + h)) - y) };
  };
  const add = (masks: Rect[], move: Patch["move"][number]) => {
    masks = masks.filter(rect => rect.width > 0 && rect.height > 0);
    if (!masks.length) return;
    const x = Math.min(...masks.map(rect => rect.x)), y = Math.min(...masks.map(rect => rect.y));
    const w = Math.max(...masks.map(rect => rect.x + rect.width)) - x, h = Math.max(...masks.map(rect => rect.y + rect.height)) - y;
    if (![x, y, w, h].every(Number.isFinite) || x < 0 || y < 0 || x + w > width || y + h > height) { unsupported = true; return; }
    patches.push({ x, y, width: w, height: h, masks, move: [move] });
  };
  for (const bone of geometry.bones ?? []) {
    for (const cube of bone.cubes ?? []) {
      const [w, h, d] = cube.size;
      if (Array.isArray(cube.uv) || !cube.uv) {
        const [u, v] = cube.uv ?? [0, 0];
        const rects = [[u + d + w, v + d, d, h], [u, v + d, d, h], [u + d, v, w, d], [u + d + w, v, w, d], [u + d + w + d, v + d, w, h], [u + d, v + d, w, h]];
        add(rects.filter(([, , w, h]) => w !== 0 && h !== 0).map(([x, y, w, h]) => rectangle(x, y, w, h)), (dx, dy) => { cube.uv = [u + dx, v + dy]; });
      } else for (const [face, definition] of Object.entries(cube.uv) as [string, Data][]) {
        const [u, v] = definition.uv ?? [0, 0];
        const [rw, rh] = definition.uv_size ?? (/up|down/.test(face) ? [Math.abs(w), Math.abs(d)] : /east|west/.test(face) ? [Math.abs(d), Math.abs(h)] : [Math.abs(w), Math.abs(h)]);
        add([rectangle(u, v, rw, rh)], (dx, dy) => { definition.uv = [u + dx, v + dy]; });
      }
    }
    if (bone.poly_mesh) {
      const mesh = bone.poly_mesh;
      if (!Array.isArray(mesh.polys)) { unsupported = true; continue; }
      const sourceUV = mesh.uvs;
      mesh.uvs = [];
      for (const polygon of mesh.polys) {
        const points = polygon.map((corner: number[]) => { const uv = sourceUV[corner[2]]; return mesh.normalized_uvs ? [uv[0] * width, (1 - uv[1]) * height] : [uv[0], height - uv[1]]; }) as number[][];
        const minU = Math.min(...points.map(p => p[0])), minV = Math.min(...points.map(p => p[1]));
        const maxU = Math.max(...points.map(p => p[0])), maxV = Math.max(...points.map(p => p[1]));
        const indices = polygon.map((corner: number[], i: number) => { corner[2] = mesh.uvs.length; mesh.uvs.push([points[i][0], height - points[i][1]]); return corner[2]; });
        add([rectangle(minU, minV, maxU - minU, maxV - minV)], (dx, dy, newHeight) => { indices.forEach((index: number, i: number) => { mesh.uvs[index] = [points[i][0] + dx, newHeight - points[i][1] - dy]; }); });
      }
      mesh.normalized_uvs = false;
    }
  }
  if (!patches.length || unsupported) return model;
  const merged: Patch[] = [];
  for (const patch of patches.sort((a, b) => b.width * b.height - a.width * a.height)) {
    const parent = merged.find(region => contains(region, patch));
    if (parent) { parent.masks.push(...patch.masks); parent.move.push(...patch.move); } else merged.push(patch);
  }
  const layout = packTextureRegions(merged, width, height);
  const repacked = layout.width !== width || layout.height !== height || merged.some((patch, index) => layout.positions[index].x !== patch.x || layout.positions[index].y !== patch.y);
  const converted = new Map<string, string>();
  for (const [alias, source] of Object.entries(copy.textures)) {
    if (!source) continue;
    let cropped = converted.get(source);
    if (!cropped) {
      const image = new Image(); image.src = source; await image.decode();
      if (image.naturalWidth % width || image.naturalHeight % height) return model;
      const sx = image.naturalWidth / width, sy = image.naturalHeight / height;
      const canvas = document.createElement("canvas");
      canvas.width = layout.width * sx; canvas.height = layout.height * sy;
      if (canvas.width > 16384 || canvas.height > 16384 || canvas.width * canvas.height > 16_777_216) return model;
      const context = canvas.getContext("2d"); if (!context) throw Error("Texture extraction is unavailable.");
      context.imageSmoothingEnabled = false;
      merged.forEach((patch, index) => {
        const destination = layout.positions[index];
        for (const mask of patch.masks) {
          const x = (destination.x + mask.x - patch.x) * sx, y = (destination.y + mask.y - patch.y) * sy;
          context.clearRect(x, y, mask.width * sx, mask.height * sy);
          context.drawImage(image, mask.x * sx, mask.y * sy, mask.width * sx, mask.height * sy, x, y, mask.width * sx, mask.height * sy);
        }
        if (repacked && destination.x > 0 && destination.y > 0 && destination.x + patch.width < layout.width && destination.y + patch.height < layout.height) {
          const x = destination.x * sx, y = destination.y * sy, w = patch.width * sx, h = patch.height * sy;
          context.drawImage(canvas, x, y, w, 1, x, y - sy, w, sy);
          context.drawImage(canvas, x, y + h - 1, w, 1, x, y + h, w, sy);
          context.drawImage(canvas, x, y, 1, h, x - sx, y, sx, h);
          context.drawImage(canvas, x + w - 1, y, 1, h, x + w, y, sx, h);
        }
      });
      cropped = canvas.toDataURL("image/png"); converted.set(source, cropped);
    }
    copy.textures[alias] = cropped;
    copy.textureNames = { ...copy.textureNames, [alias]: `${copy.name}_${alias === "*" ? "texture" : alias}.png`.replace(/[<>:"/\\|?*]/g, "_") };
  }
  merged.forEach((patch, index) => patch.move.forEach(move => move(layout.positions[index].x - patch.x, layout.positions[index].y - patch.y, layout.height)));
  if (geometry.description) { geometry.description.texture_width = layout.width; geometry.description.texture_height = layout.height; }
  else { geometry.texturewidth = layout.width; geometry.textureheight = layout.height; }
  if (copy.sources) {
    delete copy.sources["craftbox:original-textures"];
    if (copy.object?.geometryFile) copy.sources[copy.object.geometryFile] = structuredClone(copy.geometry);
  }
  return copy;
}

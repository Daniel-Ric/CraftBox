import type { ModelAnalysis, SurfaceSample } from "./auto-direction";
import {
  normalizeAtmosphere,
  type AtmosphereSettings,
} from "./atmosphere-settings";
import type { CloudLayer } from "./cloud-layers";

export type AnalysisView = {
  projection?: "perspective" | "orthographic";
  position: number[];
  target: number[];
  up?: number[];
};
export type TerrainSurface = {
  resolution: number;
  bounds: number[]; // minX, minZ, maxX, maxZ
  heights: (number | null)[];
  ground: number;
  upper: number;
  coverage: number;
  center: number[];
  extent: number[];
};
export type CloudComposition = {
  mode: "reference" | "terrain";
  template: string;
  candidates: number;
  score: number;
  baselineScore: number;
  ground?: number;
  protectedCells?: number;
  message: string;
};
const clamp = (v: number, lo: number, hi: number) =>
  Math.max(lo, Math.min(hi, v));
const length = (v: number[]) => Math.hypot(...v);
const normalize = (v: number[]) => {
  const n = length(v);
  return v.map((x) => x / Math.max(n, 1e-6));
};
const dot = (a: number[], b: number[]) =>
  a.reduce((sum, x, i) => sum + x * b[i], 0);
type Sightline = { origin: number[]; delta: number[]; length: number };
function sightline(point: number[], view: AnalysisView): Sightline | undefined {
  const forward = normalize(view.target.map((x, i) => x - view.position[i]));
  const depth = dot(
    point.map((x, i) => x - view.position[i]),
    forward,
  );
  if (depth <= 0) return;
  const origin =
    view.projection === "orthographic"
      ? point.map((x, i) => x - forward[i] * depth)
      : view.position;
  const delta = point.map((x, i) => x - origin[i]);
  return { origin, delta, length: length(delta) };
}
function rayOcclusion(
  ray: Sightline | undefined,
  center: number[],
  halfSize: number[],
) {
  if (!ray) return 0;
  let aa = 0,
    bb = 0,
    cc = -1;
  for (let i = 0; i < 3; i++) {
    const size = Math.max(1, halfSize[i]),
      o = (ray.origin[i] - center[i]) / size,
      d = ray.delta[i] / size;
    aa += d * d;
    bb += o * d;
    cc += o * o;
  }
  const discriminant = bb * bb - aa * cc;
  if (aa < 1e-12 || discriminant <= 0) return 0;
  const root = Math.sqrt(discriminant),
    near = Math.max(0, (-bb - root) / aa),
    far = Math.min(1, (-bb + root) / aa);
  if (far <= near) return 0;
  return (
    1 -
    Math.exp((-(far - near) * ray.length) / Math.max(1, Math.min(...halfSize)))
  );
}
/** Ray/ellipsoid obstruction correctly handles near-cloud/far-landmark
 * alignment for both perspective and orthographic cameras. */
export function estimateCloudOcclusion(
  point: number[],
  center: number[],
  halfSize: number[],
  view: AnalysisView,
) {
  return rayOcclusion(sightline(point, view), center, halfSize);
}

/** Upper horizontal surfaces, not the depth of the captured underground box. */
export function buildSurfaceProfile(
  samples: readonly SurfaceSample[],
  bounds: number[],
): TerrainSurface | undefined {
  const resolution = 24,
    heights: (number | null)[] = Array(resolution * resolution).fill(null);
  const width = Math.max(1, bounds[3] - bounds[0]),
    depth = Math.max(1, bounds[5] - bounds[2]);
  for (const s of samples) {
    if (s.normal[1] < 0.35) continue;
    const x = clamp(
      Math.floor(((s.position[0] - bounds[0]) / width) * resolution),
      0,
      resolution - 1,
    );
    const z = clamp(
      Math.floor(((s.position[2] - bounds[2]) / depth) * resolution),
      0,
      resolution - 1,
    );
    const at = x + z * resolution;
    heights[at] = Math.max(heights[at] ?? -Infinity, s.position[1]);
  }
  const occupied = heights.flatMap((y, i) =>
    y === null ? [] : [{ y, x: i % resolution, z: Math.floor(i / resolution) }],
  );
  if (occupied.length < 12) return;
  const sorted = occupied.map((c) => c.y).sort((a, b) => a - b);
  const quantile = (q: number) => sorted[Math.floor((sorted.length - 1) * q)];
  const loX = Math.min(...occupied.map((c) => c.x)),
    hiX = Math.max(...occupied.map((c) => c.x));
  const loZ = Math.min(...occupied.map((c) => c.z)),
    hiZ = Math.max(...occupied.map((c) => c.z));
  return {
    resolution,
    heights,
    bounds: [bounds[0], bounds[2], bounds[3], bounds[5]],
    ground: quantile(0.25),
    upper: quantile(0.85),
    coverage: occupied.length / heights.length,
    center: [
      bounds[0] + (((loX + hiX + 1) * 0.5) / resolution) * width,
      bounds[2] + (((loZ + hiZ + 1) * 0.5) / resolution) * depth,
    ],
    extent: [
      (((hiX - loX + 1) * 0.5) / resolution) * width,
      (((hiZ - loZ + 1) * 0.5) / resolution) * depth,
    ],
  };
}

/** Candidate scores estimate geometric obstruction and framing, not rendered
 * image quality. Keep the curated recipe as a candidate and safe fallback. */
export function planCloudComposition(
  a: ModelAnalysis,
  reference: AtmosphereSettings,
  view: AnalysisView | undefined,
  template: string,
): { atmosphere: AtmosphereSettings; summary: CloudComposition } {
  const original = structuredClone(reference),
    surface = a.surface;
  const fallback = {
    atmosphere: original,
    summary: {
      mode: "reference" as const,
      template,
      candidates: 1,
      score: 0,
      baselineScore: 0,
      message:
        "Curated cloud recipe preserved; insufficient upper-surface evidence for placement.",
    },
  };
  if (!reference.clouds || !surface) return fallback;
  const span = Math.max(a.size[0], a.size[2]),
    radius = Math.max(2, length(a.size) * 0.5);
  const scale = clamp(Math.sqrt(span / 512), 0.8, 1.8);
  const thickness = clamp(reference.cloudThickness * scale, 90, 240);
  const cloudSize = clamp(reference.cloudSize * scale, 80, 360);
  const center = a.center,
    target = view?.target ?? [
      surface.center[0],
      surface.ground,
      surface.center[1],
    ];
  const camera = view?.position ?? [
    target[0] + span,
    target[1] + span * 0.7,
    target[2] + span,
  ];
  const actualView = { ...view, position: camera, target };
  const cameraSide = normalize([
    camera[0] - surface.center[0],
    0,
    camera[2] - surface.center[1],
  ]);
  const cells = surface.heights.flatMap((y, i) =>
    y === null
      ? []
      : [
          {
            p: [
              surface.bounds[0] +
                (((i % surface.resolution) + 0.5) / surface.resolution) *
                  (surface.bounds[2] - surface.bounds[0]),
              y,
              surface.bounds[1] +
                ((Math.floor(i / surface.resolution) + 0.5) /
                  surface.resolution) *
                  (surface.bounds[3] - surface.bounds[1]),
            ],
            important: y > surface.upper + 4 ? 2.5 : 1,
          },
        ],
  );
  const projected = cells.map((c) => ({
    ...c,
    ray: sightline(c.p, actualView),
  }));
  const candidate = structuredClone(reference);
  Object.assign(candidate, {
    cloudModel: "banks",
    cloudReference: "world",
    cloudHeight: clamp(surface.ground - thickness * 0.65, -2048, 4096),
    cloudThickness: thickness,
    cloudSize,
    cloudExtent: clamp(reference.cloudExtent, 1.6, 2.5),
    cloudAvoidance: Math.max(0.9, reference.cloudAvoidance),
    cloudClearance: clamp(cloudSize * 0.12, 12, 40),
    cloudSeed: reference.cloudSeed,
    cloudPreviewQuality: "high",
    cloudExportQuality:
      reference.cloudExportQuality === "ultra" ? "ultra" : "high",
  });
  // Protect the build above the ground line, while retaining cloud depth below
  // it. Both base clouds and authored volumes use this same opening mask.
  candidate.cloudOpening = {
    strength: 1,
    center: [
      (surface.center[0] - center[0]) / radius,
      (surface.center[1] - center[2]) / radius,
    ],
    radius: [
      clamp((surface.extent[0] * 0.94) / radius, 0.08, 3),
      clamp((surface.extent[1] * 0.94) / radius, 0.08, 3),
    ],
    floor: (surface.ground - 12 - center[1]) / radius,
    softness: clamp(12 / radius, 0.025, 0.08),
    irregularity: 0.13,
  };
  if (candidate.fog) {
    candidate.fogReference = "world";
    candidate.fogHeight = clamp(surface.ground - 20, -2048, 4096);
    candidate.fogDensity = Math.min(0.075, candidate.fogDensity);
    candidate.fogFalloff = Math.min(30, candidate.fogFalloff);
  }
  const score = (layers: CloudLayer[], field: AtmosphereSettings) => {
    let obstruction = 0,
      weight = 0;
    const volumes = layers.map((layer) => {
      const p = layer.position.map((x, i) => center[i] + x * radius),
        s = layer.size.map((x) => x * radius * 0.5);
      // A conservative horizontal radius also covers the rotated billows.
      const horizontal = Math.max(s[0], s[2]);
      return { p, s: [horizontal, s[1], horizontal] };
    });
    for (const c of projected) {
      for (const { p, s } of volumes) {
        const q = c.p.map((x, i) => (x - p[i]) / Math.max(1, s[i]));
        const collision = Math.max(0, 1 - length(q));
        const foreground = rayOcclusion(c.ray, p, s) * 0.6;
        obstruction += (collision * 2 + foreground) * c.important;
      }
      weight += c.important;
    }
    const fieldBottom =
      field.cloudReference === "bottom"
        ? center[1] - a.size[1] * 0.5 + field.cloudHeight
        : field.cloudReference === "top"
          ? center[1] + a.size[1] * 0.5 + field.cloudHeight
          : field.cloudReference === "center"
            ? center[1] + field.cloudHeight
            : field.cloudHeight;
    const cloudTop = fieldBottom + field.cloudThickness * 0.8;
    const framingGap =
      Math.abs(cloudTop - surface.ground) / Math.max(60, thickness);
    return (
      obstruction / Math.max(1, weight) +
      // With equally clear sightlines, place more of the cloud mass behind
      // the build and keep the foreground perimeter readable.
      volumes.reduce(
        (sum, { p }) =>
          sum +
          Math.max(
            0,
            dot(
              [p[0] - surface.center[0], 0, p[2] - surface.center[1]],
              cameraSide,
            ),
          ) /
            radius,
        0,
      ) *
        0.04 +
      framingGap * 0.18 +
      (layers.length ? 0 : 0.22)
    );
  };
  const baselineScore = score(reference.cloudLayers, reference);
  let best = { atmosphere: original, score: baselineScore };
  let evaluated = 1;
  const worldWidth = clamp(Math.max(cloudSize * 1.6, span * 0.3), 96, 540);
  // Explore phase and width/height profiles. The camera controls which side
  // should frame the build rather than obscure its foreground landmarks.
  for (let phase = 0; phase < 12; phase++)
    for (const style of [0, 1]) {
      const layers: CloudLayer[] = [-75, 0, 75].map((arc, i) => {
        const angle = ((phase * 30 + arc) * Math.PI) / 180;
        const x =
          surface.center[0] +
          Math.cos(angle) * (surface.extent[0] + worldWidth * 0.28);
        const z =
          surface.center[1] +
          Math.sin(angle) * (surface.extent[1] + worldWidth * 0.28);
        const y =
          surface.ground -
          thickness * (style ? 0.32 : 0.48) +
          (i === 1 ? thickness * 0.08 : 0);
        const w = worldWidth * (i === 1 ? 1.12 : 0.9),
          h = thickness * (style ? 0.9 : 0.7);
        return {
          id: `aad-framing-${i}`,
          name: ["Cloud flank left", "Cloud horizon", "Cloud flank right"][i],
          kind: "volume",
          enabled: true,
          position: [
            (x - center[0]) / radius,
            (y - center[1]) / radius,
            (z - center[2]) / radius,
          ],
          size: [w / radius, h / radius, (w * 0.8) / radius],
          rotation: [0, (phase * 30 + arc + 30) % 360, 0],
          color: "#ffffff",
          density: clamp(reference.cloudDensity * 0.8, 0.65, 1.3),
          coverage: 0.78,
          dissolve: 0,
          detail: reference.detail,
          erosion: Math.min(0.38, reference.erosion),
          turbulence: reference.turbulence,
          seed: (a.seed + i * 197 + phase * 31) % 100000,
          softness: 0.45,
          shadows: 0.7,
          windSpeed: reference.windSpeed * 0.3,
          windDirection: reference.windDirection,
          tracks: [],
        };
      });
      const value = score(layers, candidate);
      evaluated++;
      if (value < best.score)
        best = {
          atmosphere: { ...structuredClone(candidate), cloudLayers: layers },
          score: value,
        };
    }
  // If a fitted layout would be worse by this proxy, preserve the reference.
  if (best.atmosphere === original)
    return {
      ...fallback,
      summary: {
        ...fallback.summary,
        candidates: evaluated,
        score: baselineScore,
        baselineScore,
        message:
          "Curated cloud recipe retained: fitted candidates would obstruct more of the build.",
      },
    };
  return {
    atmosphere: normalizeAtmosphere(best.atmosphere),
    summary: {
      mode: "terrain",
      template,
      candidates: evaluated,
      score: best.score,
      baselineScore,
      ground: surface.ground,
      protectedCells: cells.length,
      message:
        "Curated cloud bodies fitted around the upper terrain, with a clear build opening and camera-aware flanks.",
    },
  };
}

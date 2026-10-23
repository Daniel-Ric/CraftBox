import {
  renderDefaults,
  renderPresets,
  applyRenderPreset,
  type RenderSettings,
} from "./render-settings";
import { lookCatalog } from "./look-catalog";
import {
  directionFeatures,
  featureDistance,
  loadDirectionExamples,
} from "./direction-learning";
import { normalizeAtmosphere } from "./atmosphere-settings";
import { applyTimeOfDay } from "./render-daytime";
export type SurfaceSample = {
  position: number[];
  normal: number[];
  color: number[];
  area: number;
  mode: string;
};
export type ModelAnalysis = {
  size: number[];
  center: number[];
  samples: number;
  triangles: number;
  lights: number;
  vegetation: number;
  water: number;
  brightness: number;
  whiteness: number;
  warmth: number;
  saturation: number;
  darkness: number;
  colorDiversity: number;
  verticality: number;
  symmetry: number;
  footprint: number;
  relief: number;
  detail: number;
  confidence: number;
  dominant: string;
  palette: string[];
  scores: Record<string, number>;
  normals: { direction: number[]; weight: number }[];
  seed: number;
};
export type DirectionOptions = {
  intent: "auto" | "vivid" | "cinematic" | "neutral";
  hour?: number;
  atmosphere?: "auto" | "clouds" | "clear";
  resolution?: number;
};
export type DirectionResult = {
  settings: RenderSettings;
  analysis: ModelAnalysis;
  title: string;
  reasons: string[];
  decisions: Record<keyof RenderSettings, string>;
  atmosphereDecisions: Record<string, string>;
  candidates: number;
  reference: {
    id: string;
    name: string;
    score: number;
    learned: boolean;
    alternatives: { id: string; name: string; score: number }[];
  };
};
const clamp = (x: number, a = 0, b = 1) => Math.max(a, Math.min(b, x));
const hex = (rgb: number[]) =>
  "#" +
  rgb
    .map((x) =>
      Math.round(clamp(x) * 255)
        .toString(16)
        .padStart(2, "0"),
    )
    .join("");
const labels: Record<string, string> = {
  garden: "Vibrant nature",
  winter: "Bright winter architecture",
  desert: "Warm landscape",
  fantasy: "Fantasy island",
  architecture: "Architecture",
  dramatic: "Dark scene",
};

export function analyseModel(
  samples: SurfaceSample[],
  evidence: { lights: number; triangles: number; bounds?: number[] },
): ModelAnalysis {
  const valid = samples.filter(
    (s) =>
      s.area > 0 &&
      Number.isFinite(s.area) &&
      s.position.length === 3 &&
      s.normal.length === 3 &&
      s.color.length === 3 &&
      [...s.position, ...s.normal, ...s.color].every(Number.isFinite),
  );
  if (!valid.length)
    throw new Error("No model surfaces available for analysis.");
  const bounds = evidence.bounds?.slice() ?? [
    Infinity,
    Infinity,
    Infinity,
    -Infinity,
    -Infinity,
    -Infinity,
  ];
  let area = 0,
    vegetation = 0,
    water = 0,
    brightness = 0,
    whiteness = 0,
    warmth = 0,
    saturation = 0,
    darkness = 0,
    vertical = 0;
  const colors = new Map<string, { weight: number; rgb: number[] }>(),
    normals = new Map<string, { direction: number[]; weight: number }>();
  for (const s of valid) {
    const w = s.area;
    area += w;
    if (!evidence.bounds)
      for (let axis = 0; axis < 3; axis++) {
        bounds[axis] = Math.min(bounds[axis], s.position[axis]);
        bounds[axis + 3] = Math.max(bounds[axis + 3], s.position[axis]);
      }
    const [r, g, b] = s.color.map((v) => clamp(v)),
      hi = Math.max(r, g, b),
      lo = Math.min(r, g, b),
      sat = hi > 0 ? (hi - lo) / hi : 0,
      lum = 0.2126 * r + 0.7152 * g + 0.0722 * b;
    brightness += lum * w;
    saturation += sat * w;
    darkness += (1 - clamp(lum / 0.25)) * w;
    whiteness += clamp((lum - 0.65) / 0.3) * (1 - sat) * w;
    const green = clamp((g - Math.max(r, b)) / 0.22);
    vegetation +=
      Math.max(green, s.mode === "mask" ? green * 0.3 + 0.4 : 0) * w;
    water += (s.mode === "blend" ? 1 : clamp((b - r - 0.08) / 0.3) * 0.3) * w;
    warmth += clamp((r - b) / 0.4) * w;
    vertical += (1 - Math.abs(s.normal[1])) * w;
    const key = [r, g, b].map((v) => Math.min(7, Math.floor(v * 8))).join(",");
    const bin = colors.get(key) ?? { weight: 0, rgb: [0, 0, 0] };
    bin.weight += w;
    bin.rgb[0] += r * w;
    bin.rgb[1] += g * w;
    bin.rgb[2] += b * w;
    colors.set(key, bin);
    const n = s.normal.map((v) => Math.round(v * 8) / 8),
      len = Math.hypot(...n),
      nk = n.join(",");
    if (len > 1e-6) {
      const nb = normals.get(nk) ?? {
        direction: n.map((v) => v / len),
        weight: 0,
      };
      nb.weight += w;
      normals.set(nk, nb);
    }
  }
  for (const bin of colors.values())
    bin.rgb = bin.rgb.map((v) => v / bin.weight);
  const size = [0, 1, 2].map((i) => Math.max(1, bounds[i + 3] - bounds[i])),
    center = [0, 1, 2].map((i) => (bounds[i + 3] + bounds[i]) / 2);
  const grid = new Float64Array(32 * 32),
    heights = new Float64Array(24);
  for (const s of valid) {
    const x = Math.floor(clamp((s.position[0] - bounds[0]) / size[0]) * 31),
      z = Math.floor(clamp((s.position[2] - bounds[2]) / size[2]) * 31);
    grid[x + z * 32] += s.area;
    heights[Math.floor(clamp((s.position[1] - bounds[1]) / size[1]) * 23)] +=
      s.area;
  }
  let occupied = 0,
    symmetry = 0,
    entropy = 0;
  for (let z = 0; z < 32; z++)
    for (let x = 0; x < 32; x++) {
      const v = grid[x + z * 32];
      if (v > area / 100000) occupied++;
      symmetry += Math.min(v, grid[31 - x + z * 32]);
    }
  for (const bin of colors.values()) {
    const p = bin.weight / area;
    entropy -= p * Math.log2(p);
  }
  const norm = (v: number) => clamp(v / area),
    veg = norm(vegetation),
    snow = norm(whiteness),
    wet = norm(water),
    warm = norm(warmth),
    dark = norm(darkness),
    div = clamp(entropy / 6);
  const verticality = norm(vertical),
    relief = clamp(size[1] / Math.sqrt(size[0] * size[2]), 0, 3),
    footprint = occupied / 1024,
    detail = clamp(
      Math.log10(
        1 + evidence.triangles / Math.max(1, Math.sqrt(size[0] * size[2])),
      ) / 4,
    );
  const scores = {
    garden: veg * 1.6 + wet * 0.2,
    winter: snow * 1.8,
    desert: warm * 1.1 + Math.max(0, 0.3 - veg),
    fantasy: div * 0.55 + relief * 0.24 + wet * 0.4,
    architecture:
      verticality * 0.65 + (symmetry / area) * 0.25 + (1 - veg) * 0.15,
    dramatic: dark * 1.6,
  };
  const dominant = Object.keys(scores).sort(
    (a, b) =>
      scores[b as keyof typeof scores] - scores[a as keyof typeof scores],
  )[0];
  const sorted = [...colors.values()].sort((a, b) => b.weight - a.weight);
  const seed =
    Math.round(
      size[0] * 13 +
        size[1] * 31 +
        size[2] * 17 +
        (brightness / area) * 1000 +
        entropy * 100,
    ) % 999999;
  return {
    size,
    center,
    samples: valid.length,
    triangles: evidence.triangles,
    lights: Math.max(0, evidence.lights),
    vegetation: veg,
    water: wet,
    brightness: norm(brightness),
    whiteness: snow,
    warmth: warm,
    saturation: norm(saturation),
    darkness: dark,
    colorDiversity: div,
    verticality,
    symmetry: clamp(symmetry / area),
    footprint,
    relief,
    detail,
    confidence: clamp(Math.sqrt(valid.length / 2048)) * 0.9,
    dominant,
    palette: sorted.slice(0, 6).map((b) => hex(b.rgb)),
    scores,
    normals: [...normals.values()].map((n) => ({
      ...n,
      weight: n.weight / area,
    })),
    seed,
  };
}



const referenceFeatures: Record<string, number[]> = {
  clouds: [0.2, 0.08, 0.5, 0.2, 0.08, 0.6, 0.4, 0.4],
  nature: [0.65, 0.22, 0.1, 0.12, 0.06, 0.25, 0.5, 0.25],
  fantasy: [0.3, 0.08, 0.28, 0.22, 0.1, 0.65, 0.6, 0.5],
  winter: [0.06, 0.05, 0.8, 0.03, 0.04, 0.4, 0.2, 0.45],
  desert: [0.07, 0.04, 0.12, 0.65, 0.1, 0.4, 0.3, 0.25],
  city: [0.06, 0.1, 0.2, 0.15, 0.24, 0.82, 0.25, 0.5],
  interior: [0.06, 0.04, 0.12, 0.26, 0.4, 0.7, 0.3, 0.3],
  film: [0.2, 0.12, 0.3, 0.2, 0.15, 0.45, 0.35, 0.3],
};
export function directModel(
  a: ModelAnalysis,
  o: DirectionOptions,
): DirectionResult {
  const features = directionFeatures(a),
    examples = loadDirectionExamples(),
    span = Math.max(a.size[0], a.size[2]);
  const island =
    a.footprint < 0.87 &&
    Math.min(a.size[0], a.size[2]) / span > 0.6 &&
    a.relief > 0.15;
  const ranked = lookCatalog
    .map((look) => {
      const preset = renderPresets[look.id],
        clouds = !!preset.atmosphere?.clouds;
      if (
        (o.atmosphere === "clouds" && !clouds) ||
        (o.atmosphere === "clear" && clouds)
      )
        return { ...look, score: -Infinity, learned: false };
      const proto = [...referenceFeatures[look.category]];
      if (look.id === "cloudGolden") {
        proto[2] = 0.4;
        proto[3] = 0.24;
        proto[6] = 0.45;
      }
      if (look.id === "cloudPearl") proto[2] = 0.85;
      if (look.id === "cloudFairytale") {
        proto[0] = 0.4;
        proto[6] = 0.65;
      }
      let score = -featureDistance(features, proto);
      if (o.atmosphere === "auto" || !o.atmosphere)
        score += clouds ? (island ? 0.4 : -0.25) : 0;
      if (o.intent === "cinematic")
        score +=
          (preset.sky === "sunset" || preset.sky === "dawn" ? 0.12 : 0) +
          (preset.contrast ?? 1) * 0.03;
      if (o.intent === "vivid")
        score +=
          ((preset.saturation ?? 1) - 1) * 0.8 + (preset.vibrance ?? 0) * 0.15;
      if (o.intent === "neutral")
        score += [
          "crisp",
          "neutral",
          "studio",
          "showcaseLight",
          "cityDay",
        ].includes(look.id)
          ? 1
          : -0.6;
      const candidateNight = preset.sky === "night" || (preset.sun ?? 3) < 0.7;
      if (o.hour !== undefined) {
        const t = preset.timeOfDay ?? (candidateNight ? 22 : 12),
          d = Math.abs(t - o.hour);
        score -= Math.min(d, 24 - d) * 0.025;
      } else if (candidateNight && a.darkness < 0.3 && a.lights < 6)
        score -= 0.35;
      if (look.id === "cloudGolden") score += 0.035;
      const bonus = examples
        .filter((e) => e.lookId === look.id)
        .reduce(
          (best, e) =>
            Math.max(
              best,
              2 * Math.exp(-featureDistance(features, e.features) * 5),
            ),
          0,
        );
      score += bonus;
      return { ...look, score, learned: bonus > 0.1 };
    })
    .filter((l) => Number.isFinite(l.score))
    .sort((x, y) => y.score - x.score || x.id.localeCompare(y.id));
  const reference = ranked[0];
  if (!reference) throw new Error("No suitable reference scene found.");
  let settings = applyRenderPreset(
    { ...renderDefaults, edge: o.resolution ?? 3840 },
    reference.id,
  );
  if (o.hour !== undefined) settings = applyTimeOfDay(settings, o.hour);
  const atmosphere = { ...settings.atmosphere };
  const scale = clamp(span / 512, 0.6, 4);
  for (const key of [
    "cloudSize",
    "cloudThickness",
    "cloudHeight",
    "cloudClearance",
    "fogHeight",
    "fogFalloff",
    "fogSize",
  ] as const)
    atmosphere[key] *= scale;
  if (atmosphere.clouds) {
    atmosphere.cloudThickness = clamp(atmosphere.cloudThickness, 2, 512);
    atmosphere.cloudHeight = Math.min(
      atmosphere.cloudHeight,
      a.size[1] * 0.55 - atmosphere.cloudThickness,
    );
  }
  atmosphere.cloudSeed = a.seed;
  atmosphere.fogSeed = (a.seed + 137) % 999999;
  settings = {
    ...settings,
    quality: "fine",
    atmosphere: normalizeAtmosphere(atmosphere),
    blockLightPower: clamp(
      settings.blockLightPower + Math.min(1, a.lights / 60) * 0.15,
      0,
      8,
    ),
    lightBudget: a.lights > 16 ? 24 : a.lights > 8 ? 16 : a.lights > 4 ? 8 : 4,
    blockLightShadows: a.lights > 1 ? 2 : a.lights > 0 ? 1 : 0,
    highlights: Math.min(settings.highlights, -0.1 - a.whiteness * 0.25),
    exposure: clamp(
      settings.exposure + Math.max(0, 0.25 - a.brightness) * 0.25,
      0.3,
      1.8,
    ),
  };
  const decisions = {} as DirectionResult["decisions"];
  for (const key of Object.keys(settings) as (keyof RenderSettings)[])
    decisions[key] = "Matched reference: " + reference.name + ".";
  for (const key of [
    "blockLightPower",
    "blockLightShadows",
    "lightBudget",
  ] as const)
    decisions[key] =
      a.lights + " captured light sources; budget adjusted to the model.";
  decisions.atmosphere =
    "Reference atmosphere scaled by " +
    scale.toFixed(2) +
    " adjusted to world dimensions; lighting mood preserved.";
  decisions.exposure =
    "Reference exposure with limited correction for dark surfaces.";
  decisions.highlights =
    "Reference highlights with extra protection for bright materials.";
  decisions.edge = "Output size from advanced preferences.";
  if (o.hour !== undefined)
    for (const key of [
      "timeOfDay",
      "timeLinked",
      "azimuth",
      "elevation",
      "sun",
      "sky",
      "warmth",
      "ambient",
      "environment",
      "skyRotation",
      "skyBrightness",
      "background",
      "temperature",
    ] as const)
      decisions[key] = "Specified time of day " + o.hour.toFixed(2) + " h.";
  const atmosphereDecisions = Object.fromEntries(
    Object.keys(settings.atmosphere).map((k) => [k, decisions.atmosphere]),
  );
  return {
    settings,
    analysis: a,
    title: reference.name,
    candidates: ranked.length,
    decisions,
    atmosphereDecisions,
    reference: {
      id: reference.id,
      name: reference.name,
      score: reference.score,
      learned: reference.learned,
      alternatives: ranked
        .slice(1, 4)
        .map((l) => ({ id: l.id, name: l.name, score: l.score })),
    },
    reasons: [
      a.samples.toLocaleString("en-US") +
        " area-weighted surface samples.",
      "Model character: " + labels[a.dominant] + ".",
      ranked.length +
        " Scene looks evaluated for shape, materials, palette and creative intent.",
      reference.learned
        ? "A saved local reference influenced the selection."
        : "Calibrated against curated library scenes.",
      "Reference sun direction, light energy and color treatment are preserved; scale and material protection are adjusted as needed.",
    ],
  };
}

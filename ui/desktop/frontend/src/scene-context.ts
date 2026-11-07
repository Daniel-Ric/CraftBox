export type SceneTheme =
  | "winter"
  | "autumn"
  | "spring"
  | "summer"
  | "halloween"
  | "easter"
  | "christmas";
export const sceneThemeLabels: Record<SceneTheme, string> = {
  winter: "Winter",
  autumn: "Autumn",
  spring: "Spring",
  summer: "Summer",
  halloween: "Halloween",
  easter: "Easter",
  christmas: "Christmas",
};
type Visual = {
  warmFoliage?: number;
  warmth: number;
  whiteness: number;
  vegetation: number;
  water: number;
  darkness: number;
  saturation: number;
  brightness: number;
  colorDiversity: number;
};
export type SceneContext = {
  themes: Record<SceneTheme, number>;
  theme: SceneTheme | "unknown";
  strength: number;
  evidence: string[];
  cautions: string[];
  hasBlockEvidence: boolean;
};
const clamp = (x: number) =>
  Math.max(0, Math.min(1, Number.isFinite(x) ? x : 0));
export function detectSceneContext(
  v: Visual,
  raw: Record<string, number> = {},
  sourceName = "",
): SceneContext {
  const n = (key: string) =>
      Math.max(0, Number.isFinite(raw[key]) ? raw[key] : 0),
    total = n("total");
  const present = (key: string, min = 3) =>
    clamp(n(key) / Math.max(min, Math.min(24, Math.sqrt(total) * 0.4)));
  const snow =
    n("surface") > 0
      ? clamp((n("snowSurface") + n("iceSurface") * 0.8) / n("surface"))
      : clamp(((n("snow") + n("ice") * 0.8) / Math.max(1, total)) * 3.5);
  const foliage = present("foliage", 20),
    flowers = present("flowers", 12),
    pumpkins = present("pumpkins"),
    lanterns = present("jackolanterns"),
    webs = present("webs"),
    skulls = present("skulls"),
    eggs = present("eggs"),
    rabbits = present("rabbits"),
    gifts = present("gifts"),
    festive = present("festive");
  // A title is supporting evidence only: server release names can be stale.
  const title = sourceName.toLowerCase(),
    hint = (pattern: RegExp) => (pattern.test(title) ? 0.08 : 0);
  const spookyFamilies = [Math.max(pumpkins, lanterns), webs, skulls].filter(
    (x) => x >= 0.3,
  ).length;
  const easterFamilies = [eggs, rabbits, flowers].filter(
    (x) => x >= 0.3,
  ).length;
  const halloween =
    spookyFamilies >= 3
      ? clamp(
          0.2 +
            pumpkins * 0.12 +
            lanterns * 0.24 +
            webs * 0.22 +
            skulls * 0.2 +
            hint(/halloween|haunted|spooky/),
        )
      : 0;
  const easter =
    easterFamilies >= 2 && (eggs > 0.3 || rabbits > 0.3)
      ? clamp(
          0.18 +
            eggs * 0.36 +
            rabbits * 0.28 +
            flowers * 0.18 +
            hint(/easter|ostern|spring festival/),
        )
      : 0;
  const christmas =
    gifts > 0.3 && (snow > 0.15 || festive > 0.3)
      ? clamp(
          0.2 +
            gifts * 0.35 +
            festive * 0.2 +
            snow * 0.2 +
            hint(/christmas|weihnacht|xmas/),
        )
      : 0;
  const themes: Record<SceneTheme, number> = {
    winter: clamp(snow * 0.9 + present("ice") * 0.1),
    autumn: clamp(
      foliage * clamp((v.warmFoliage ?? 0) * 5) * 0.75 +
        present("autumn") * 0.4,
    ),
    spring: clamp(flowers * 0.52 + foliage * 0.18 + (1 - v.warmth) * 0.1),
    summer: clamp(foliage * 0.22 + v.water * 0.28 + v.vegetation * 0.2),
    halloween,
    easter,
    christmas,
  };
  const ranked = (Object.entries(themes) as [SceneTheme, number][]).sort(
    (a, b) => b[1] - a[1],
  );
  const [theme, strength] = ranked[0];
  const evidence: string[] = [];
  for (const [key, label] of Object.entries({
    snow: "Snow blocks",
    ice: "Ice blocks",
    foliage: "Foliage",
    flowers: "Flowers",
    pumpkins: "Pumpkins",
    jackolanterns: "Jack-o-lanterns",
    webs: "Cobwebs",
    skulls: "Skulls",
    eggs: "Egg decorations",
    rabbits: "Rabbit decorations",
    gifts: "Gift decorations",
    festive: "Festive decorations",
  }))
    if (n(key) > 0)
      evidence.push(`${label}: ${Math.round(n(key)).toLocaleString("en-US")}`);
  const cautions = [
    "Color and server titles alone do not establish a season or event.",
  ];
  if (!total)
    cautions.push(
      "Block metadata unavailable; only visual styling can be assessed.",
    );
  if (v.warmth > 0.5 && halloween < 0.65)
    cautions.push(
      "Warm/orange surfaces are treated as a palette, not Halloween evidence.",
    );
  if (v.whiteness > 0.5 && themes.winter < 0.6)
    cautions.push("Bright materials alone are not evidence of snow or winter.");
  return {
    themes,
    theme: strength >= 0.6 ? theme : "unknown",
    strength: clamp(strength),
    evidence,
    cautions,
    hasBlockEvidence: total > 0,
  };
}

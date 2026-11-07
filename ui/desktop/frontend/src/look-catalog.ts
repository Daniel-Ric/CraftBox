import {
  renderPresetLabels,
  renderLookDescriptions,
  renderPresets,
} from "./render-settings";
import { sceneLooks, type SceneLook, type LookCategory } from "./scene-looks";
const legacyCategory: Record<string, LookCategory> = {
  cloudIsland: "clouds",
  cloudGolden: "clouds",
  cloudSoft: "clouds",
  island: "nature",
  garden: "nature",
  tropical: "nature",
  winter: "winter",
  winterevening: "winter",
  night: "city",
  bluehour: "city",
  architecture: "city",
  studio: "interior",
  neutral: "film",
};
const featured = [
  "cloudGolden",
  "cloudIsland",
  "cloudSoft",
  "island",
  "garden",
  "film",
  "winterevening",
  "crisp",
];
const skySwatches: Record<string, string> = {
  night: "#253b62",
  sunset: "#d7a7ab",
  dawn: "#bbc6db",
  daylight: "#91bbd3",
};
export const lookCatalog: SceneLook[] = [
  ...featured,
  ...Object.keys(renderPresetLabels).filter((id) => !featured.includes(id)),
].map(
  (id) =>
    sceneLooks.find((l) => l.id === id) ?? {
      id,
      name: renderPresetLabels[id],
      description: renderLookDescriptions[id] ?? "Neutral lighting and detail.",
      category: legacyCategory[id] ?? "film",
      recommended: featured.includes(id),
      tags: [],
      settings: renderPresets[id] ?? {},
      palette: [
        skySwatches[renderPresets[id]?.sky ?? "daylight"] ??
          renderPresets[id]?.background ??
          "#91bbd3",
        renderPresets[id]?.warmth ?? "#fff1dc",
        renderPresets[id]?.atmosphere?.fogColor ?? "#c7d2dc",
        legacyCategory[id] === "winter"
          ? "#dce8ef"
          : legacyCategory[id] === "nature"
            ? "#69874a"
            : "#75878c",
      ],
    },
);
export function filterLooks(category: string, query: string) {
  const words = query
    .trim()
    .toLocaleLowerCase("en")
    .split(/\s+/)
    .filter(Boolean);
  return lookCatalog.filter(
    (l) =>
      (category === "all" || category === "recommended"
        ? category !== "recommended" || l.recommended
        : l.category === category) &&
      words.every((q) =>
        [l.name, l.description, ...l.tags, l.category, l.theme ?? ""]
          .join(" ")
          .toLocaleLowerCase("en")
          .includes(q),
      ),
  );
}

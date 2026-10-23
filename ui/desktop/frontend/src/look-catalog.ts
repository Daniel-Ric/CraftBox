import { renderPresetLabels, renderLookDescriptions } from "./render-settings";
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
export const lookCatalog: SceneLook[] = [
  ...featured,
  ...Object.keys(renderPresetLabels).filter((id) => !featured.includes(id)),
].map(
  (id) =>
    sceneLooks.find((l) => l.id === id) ?? {
      id,
      name: renderPresetLabels[id],
      description:
        renderLookDescriptions[id] ??
        "Coordinated lighting and neutral detail as a starting point for your own scene styling.",
      category: legacyCategory[id] ?? "film",
      recommended: featured.includes(id),
      tags: [],
      settings: {},
      palette: ["#8bbad6", "#75878c", "#b8b4a4", "#e3d6c0"],
    },
);
export function filterLooks(category: string, query: string) {
  const q = query.trim().toLocaleLowerCase("en");
  return lookCatalog.filter(
    (l) =>
      (category === "all" || category === "recommended"
        ? category !== "recommended" || q || l.recommended
        : l.category === category) &&
      (!q ||
        [l.name, l.description, ...l.tags]
          .join(" ")
          .toLocaleLowerCase("en")
          .includes(q)),
  );
}

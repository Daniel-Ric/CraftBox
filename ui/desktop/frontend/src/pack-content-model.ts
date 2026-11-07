import type { PackObject } from "./bridge";

export const packObjectTypes = [
  ["all", "All content"],
  ["models", "3D models"],
  ["entity", "Entities"],
  ["item-model", "Wearables and held models"],
  ["block", "Blocks"],
  ["geometry", "Unassigned geometries"],
  ["item", "Item icons"],
  ["texture", "Textures"],
  ["ui", "Interface images"],
] as const;
export const packObjectKind = (kind: string) =>
  ({
    entity: "Entity",
    "item-model": "Wearable / held model",
    block: "Block",
    geometry: "Geometry",
    item: "Item icon",
    texture: "Texture",
    ui: "Interface image",
  })[kind] || kind;
export const isPackModel = (kind: string) =>
  ["entity", "item-model", "block", "geometry"].includes(kind);
export function packCategoryMatches(object: PackObject, category: string) {
  return (
    category === "all" ||
    (category === "models"
      ? isPackModel(object.kind)
      : object.kind === category)
  );
}
export function readablePackName(value: string) {
  const text = value
    .replace(/([a-z\d])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
  return text ? text[0].toUpperCase() + text.slice(1) : "Unnamed object";
}
export function packObjectLabel(object: PackObject) {
  const original = object.name.replace(/\\/g, "/").trim();
  const colon = original.indexOf(":");
  const namespace =
    colon >= 0 && !original.slice(0, colon).includes("/")
      ? original.slice(0, colon)
      : "";
  const path = namespace ? original.slice(colon + 1) : original;
  const parts = path
    .split("/")
    .map((s) => s.trim())
    .filter(Boolean);
  let leaf = parts.pop() || path;
  if (object.kind === "geometry" && /^geometry\./i.test(leaf)) {
    leaf =
      object.geometryFile.replace(/\\/g, "/").split("/").pop() ||
      leaf.slice(9).replace(/\./g, " ");
  }
  leaf = leaf.replace(/(?:\.geo)?\.json$|\.(?:png|tga|jpe?g|webp)$/i, "");
  if (object.kind === "geometry") {
    const variant = object.geometryId.split(".").pop() || "";
    const words = (value: string) =>
      value.toLowerCase().replace(/[^a-z0-9]/g, "");
    if (variant && !words(leaf).includes(words(variant)))
      leaf += " · " + readablePackName(variant);
  }
  return {
    name: readablePackName(leaf),
    context: [namespace, ...parts.map(readablePackName)]
      .filter(Boolean)
      .join(" / "),
  };
}
const normalize = (value: string) =>
  value
    .toLocaleLowerCase("en-US")
    .replace(/[\\/_.:\s-]+/g, " ")
    .trim();
export function packObjectMatches(object: PackObject, query: string) {
  const label = packObjectLabel(object);
  const haystack = normalize(
    `${label.name} ${object.name} ${object.geometryId} ${object.geometryFile} ${object.definition} ${packObjectKind(object.kind)}`,
  );
  return normalize(query)
    .split(" ")
    .every((word) => haystack.includes(word));
}
const collator = new Intl.Collator("en-US", {
  numeric: true,
  sensitivity: "base",
});
export function sortPackObjects(a: PackObject, b: PackObject) {
  const al = packObjectLabel(a),
    bl = packObjectLabel(b);
  return (
    collator.compare(al.name, bl.name) ||
    collator.compare(al.context, bl.context) ||
    collator.compare(a.kind, b.kind) ||
    collator.compare(a.id, b.id)
  );
}

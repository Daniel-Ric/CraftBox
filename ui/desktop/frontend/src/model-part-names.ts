import type { ModelPart } from "./model-parts";

export function readablePartName(value: string) {
  const tail = value.split(/[/:]/).filter(Boolean).at(-1) ?? value;
  const words = tail.replace(/^geometry\./i, "").replace(/([a-z])([A-Z])/g, "$1 $2").replace(/([a-zA-Z])(\d+)/g, "$1 $2").replace(/[_.-]+/g, " ").replace(/\s+/g, " ").trim();
  return words.replace(/\b[a-z]/g, letter => letter.toUpperCase()) || "Model part";
}

export function nameModelParts(parts: ModelPart[], animations: Record<string, unknown> = {}, context = "Model") {
  const named = parts.map(part => {
    const sourceName = part.sourceName ?? part.name;
    const bones = new Set(part.bones.map(bone => bone.toLowerCase()));
    const families = new Map<string, number>();
    for (const [id, value] of Object.entries(animations)) {
      if (!value || typeof value !== "object") continue;
      const animated = Object.keys((value as { bones?: Record<string, unknown> }).bones ?? {});
      const matches = animated.filter(bone => bones.has(bone.toLowerCase())).length;
      const tokens = id.split(".");
      if (!matches || tokens.length < 3) continue;
      const family = tokens.at(-2)!;
      if (/^(animation|animations|entity|shared|common|global|default)$/i.test(family)) continue;
      families.set(family, (families.get(family) ?? 0) + matches / animated.length);
    }
    const ranked = [...families].sort((a, b) => b[1] - a[1]);
    const family = ranked.length && (ranked.length === 1 || ranked[0][1] > ranked[1][1] * 1.5) ? ranked[0][0] : undefined;
    const generic = /^(root|scene|model|mesh|main|all|board|body|human|pet|mount)\d*$/i.test(sourceName);
    const contextual = generic && (parts.length === 1 || /^(board|mesh|model)$/i.test(sourceName));
    const familyLabel = family && (generic || readablePartName(sourceName) === readablePartName(family));
    const label = familyLabel && part.detection !== "element" && part.detection !== "bone" ? family : contextual ? context : sourceName;
    return { ...part, sourceName, name: readablePartName(label) };
  });
  const counts = new Map<string, number>(), used = new Map<string, number>();
  for (const part of named) counts.set(part.name, (counts.get(part.name) ?? 0) + 1);
  return named.map(part => {
    if ((counts.get(part.name) ?? 0) < 2) return part;
    const number = (used.get(part.name) ?? 0) + 1;
    used.set(part.name, number);
    return { ...part, name: `${part.name} ${number}` };
  });
}

export function selectionProjectName(parts: ModelPart[], context: string) {
  if (!parts.length) return readablePartName(context);
  const names = [...new Set(parts.map(part => part.name))];
  const label = names.slice(0, 3).join(" + ") + (names.length > 3 ? ` + ${names.length - 3} more` : "");
  return label.slice(0, 120).trim();
}

import type { ModelAnalysis } from "./auto-direction";
import { renderPresetLabels } from "./render-settings";
const key = "craftbox-direction-examples";
export type DirectionExample = {
  lookId: string;
  features: number[];
  updated: number;
};
export function directionFeatures(a: ModelAnalysis) {
  return [
    a.vegetation,
    a.water,
    a.whiteness,
    a.warmth,
    a.darkness,
    a.verticality,
    a.colorDiversity,
    Math.min(1, a.relief / 2),
  ];
}
export function featureDistance(a: number[], b: number[]) {
  const weights = [1.4, 1.1, 1.4, 0.8, 1, 0.6, 0.6, 0.35];
  return a.reduce((sum, x, i) => sum + (x - b[i]) ** 2 * weights[i], 0) / 7.25;
}
export function loadDirectionExamples(): DirectionExample[] {
  try {
    const raw = JSON.parse(localStorage.getItem(key) || "null");
    if (raw?.version !== 1 || !Array.isArray(raw.examples)) return [];
    return raw.examples
      .filter(
        (e: DirectionExample) =>
          e &&
          Object.hasOwn(renderPresetLabels, e.lookId) &&
          Array.isArray(e.features) &&
          e.features.length === 8 &&
          e.features.every((x) => Number.isFinite(x) && x >= 0 && x <= 1) &&
          Number.isFinite(e.updated),
      )
      .slice(0, 64);
  } catch {
    return [];
  }
}
export function rememberDirectionExample(
  analysis: ModelAnalysis,
  lookId: string,
) {
  if (!Object.hasOwn(renderPresetLabels, lookId))
    throw new Error("This look is not in the scene library.");
  const features = directionFeatures(analysis),
    examples = loadDirectionExamples().filter(
      (e) => featureDistance(features, e.features) > 0.00001,
    );
  examples.unshift({ lookId, features, updated: Date.now() });
  try {
    localStorage.setItem(
      key,
      JSON.stringify({ version: 1, examples: examples.slice(0, 64) }),
    );
  } catch {
    throw new Error(
      "Could not save the auto-styling reference to your local profile.",
    );
  }
}

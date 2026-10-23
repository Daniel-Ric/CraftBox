import { expect, it } from "vitest";
import { sceneLooks, lookCategories } from "./scene-looks";
import {
  applyRenderPreset,
  renderDefaults,
  renderPresets,
} from "./render-settings";
import { encodeRenderStyle, decodeRenderStyle } from "./render-style-library";
it("provides forty distinct scene looks with portable complete configurations", () => {
  expect(sceneLooks.length).toBeGreaterThanOrEqual(40);
  expect(new Set(sceneLooks.map((x) => x.id)).size).toBe(sceneLooks.length);
  const signatures = new Set<string>();
  for (const look of sceneLooks) {
    expect(lookCategories[look.category]).toBeTruthy();
    expect(look.description.length).toBeGreaterThan(25);
    const applied = applyRenderPreset(
      { ...renderDefaults, edge: 2560, aspect: 1, transparent: true },
      look.id,
    );
    expect(renderPresets[look.id]).toBeDefined();
    expect([applied.edge, applied.aspect, applied.transparent]).toEqual([
      2560,
      1,
      true,
    ]);
    expect(
      decodeRenderStyle(encodeRenderStyle(look.name, applied)).settings,
    ).toEqual(applied);
    signatures.add(JSON.stringify(applied));
  }
  expect(signatures.size).toBe(sceneLooks.length);
});

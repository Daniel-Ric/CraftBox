import { beforeEach, expect, it } from "vitest";
import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { afterEach } from "vitest";
import { lookCatalog, filterLooks } from "../src/look-catalog";
import { renderDefaults, applyRenderPreset } from "../src/render-settings";
import { detectSceneContext } from "../src/scene-context";
import LookLibrary from "../src/LookLibrary";
import { analyseModel, directModel } from "../src/auto-direction";
import AtmosphereControls from "../src/AtmosphereControls";
import {
  atmosphereDefaults,
  normalizeAtmosphere,
} from "../src/atmosphere-settings";
beforeEach(() => localStorage.clear());
afterEach(cleanup);
const visual = {
  warmth: 0.9,
  whiteness: 0.1,
  vegetation: 0.15,
  water: 0.05,
  darkness: 0.1,
  saturation: 0.6,
  brightness: 0.5,
  colorDiversity: 0.5,
};
it("offers at least 100 distinct complete looks", () => {
  expect(lookCatalog.length).toBeGreaterThanOrEqual(100);
  expect(new Set(lookCatalog.map((l) => l.id)).size).toBe(lookCatalog.length);
  for (const look of lookCatalog) {
    const s = applyRenderPreset(renderDefaults, look.id);
    expect(Number.isFinite(s.exposure)).toBe(true);
    expect(s.atmosphere).toBeTruthy();
  }
});
it("never infers Halloween from orange alone, an old server title, or a pumpkin farm", () => {
  for (const signals of [
    {},
    { total: 3000, pumpkins: 300 },
    { total: 3000, pumpkins: 300, foliage: 100 },
  ]) {
    const c = detectSceneContext(visual, signals, "Halloween Update Release");
    expect(c.themes.halloween).toBeLessThan(0.5);
  }
});
it("distinguishes snowy terrain from white architecture and combines event evidence", () => {
  expect(
    detectSceneContext(
      { ...visual, whiteness: 0.9 },
      { total: 2000 },
      "Marble palace",
    ).themes.winter,
  ).toBeLessThan(0.5);
  expect(
    detectSceneContext(visual, { total: 2000, snow: 800, ice: 80 }, "").themes
      .winter,
  ).toBeGreaterThan(0.6);
  expect(
    detectSceneContext(
      visual,
      { total: 3000, pumpkins: 30, jackolanterns: 15, webs: 30, skulls: 8 },
      "Haunted courtyard",
    ).themes.halloween,
  ).toBeGreaterThan(0.7);
  expect(
    detectSceneContext(
      visual,
      { total: 3000, eggs: 30, flowers: 120, rabbits: 12 },
      "Spring festival",
    ).themes.easter,
  ).toBeGreaterThan(0.7);
});
it("combines category and multi-word search instead of silently bypassing filters", () => {
  const all = filterLooks("winter", "winter clear");
  expect(all.length).toBeGreaterThan(0);
  expect(all.every((l) => l.category === "winter")).toBe(true);
  expect(filterLooks("recommended", "winter").every((l) => l.recommended)).toBe(
    true,
  );
});
it("uses evidence gates in actual auto selection and honours manual overrides", () => {
  const a = analyseModel(
    [
      {
        position: [0, 0, 0],
        normal: [0, 1, 0],
        color: [1, 0.45, 0.05],
        area: 100,
        mode: "solid",
      },
    ],
    { lights: 5, triangles: 100 },
  );
  const auto = directModel(a, { intent: "auto" });
  expect(lookCatalog.find((l) => l.id === auto.reference.id)?.theme).not.toBe(
    "halloween",
  );
  const manual = directModel(a, { intent: "auto", theme: "halloween" });
  expect(lookCatalog.find((l) => l.id === manual.reference.id)?.theme).toBe(
    "halloween",
  );
  expect(
    directModel(a, { intent: "neutral", theme: "none", hour: 12 }).settings
      .timeOfDay,
  ).toBe(12);
});
it("edits global cloud tint and preserves it through settings normalization", () => {
  const changes: any[] = [];
  render(
    <AtmosphereControls
      value={{ ...atmosphereDefaults }}
      onChange={(v) => changes.push(v)}
      disabled={false}
    />,
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Cloud color: Soft pink" }),
  );
  expect(changes.at(-1).cloudColor).toBe("#f2cfdd");
  expect(normalizeAtmosphere(changes.at(-1)).cloudColor).toBe("#f2cfdd");
});
it("preserves manual cloud colour unless automatic tint is explicitly enabled", () => {
  const a = analyseModel(
    [
      {
        position: [0, 0, 0],
        normal: [0, 1, 0],
        color: [0.7, 0.8, 0.9],
        area: 100,
        mode: "solid",
      },
    ],
    { lights: 0, triangles: 100, signals: { total: 100, snow: 70, ice: 10 } },
  );
  expect(
    directModel(a, { intent: "auto", cloudColor: "#ab12cd" }).settings
      .atmosphere.cloudColor,
  ).toBe("#ab12cd");
  const automatic = directModel(a, {
    intent: "auto",
    cloudColor: "#ab12cd",
    autoCloudColor: true,
  });
  expect(automatic.settings.atmosphere.cloudColor).not.toBe("#ab12cd");
  expect(automatic.settings.atmosphere.cloudColor).toMatch(/^#[a-f0-9]{6}$/i);
});
it("keeps snow coverage independent of underground depth and supports theme with clouds", () => {
  const shallow = detectSceneContext(
    visual,
    { total: 300, surface: 256, snowSurface: 256, snow: 256 },
    "",
  );
  const deep = detectSceneContext(
    visual,
    { total: 100000, surface: 256, snowSurface: 256, snow: 256 },
    "",
  );
  expect(deep.themes.winter).toBe(shallow.themes.winter);
  expect(deep.themes.winter).toBeGreaterThan(0.8);
  const a = analyseModel(
    [
      {
        position: [0, 0, 0],
        normal: [0, 1, 0],
        color: [1, 0.5, 0],
        area: 100,
        mode: "solid",
      },
    ],
    { lights: 0, triangles: 100 },
  );
  for (const theme of ["halloween", "easter", "winter"] as const) {
    const result = directModel(a, {
      intent: "auto",
      theme,
      atmosphere: "clouds",
    });
    expect(result.settings.atmosphere.clouds).toBe(true);
    expect(lookCatalog.find((l) => l.id === result.reference.id)?.theme).toBe(
      theme,
    );
  }
});
it("browses before applying and supports favourites", async () => {
  const chosen: string[] = [];
  render(
    <LookLibrary
      selected="crisp"
      disabled={false}
      onSelect={(id) => chosen.push(id)}
    />,
  );
  fireEvent.click(screen.getByRole("button", { name: "Open look library" }));
  await screen.findByRole("dialog", { name: "Scene looks" });
  fireEvent.click(screen.getByRole("button", { name: /^All scenes/ }));
  fireEvent.change(screen.getByRole("searchbox", { name: "Browse library" }), {
    target: { value: "Halloween" },
  });
  fireEvent.click(
    screen.getAllByRole("button", { name: /Inspect Halloween/ })[0],
  );
  expect(chosen).toEqual([]);
  fireEvent.click(
    screen.getByRole("button", { name: "Add look to favourites" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Apply look" }));
  expect(chosen.length).toBe(1);
});

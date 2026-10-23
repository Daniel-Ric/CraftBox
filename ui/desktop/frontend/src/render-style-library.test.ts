import { expect, it } from "vitest";
import { renderDefaults, renderPresets } from "./render-settings";
import {
  encodeRenderStyle,
  decodeRenderStyle,
  saveRenderStyle,
  loadRenderStyles,
} from "./render-style-library";
it("transfers a complete render style between worlds without keeping mutable references", async () => {
  localStorage.clear();
  const settings = {
    ...renderDefaults,
    exposure: 1.2,
    blockLightPower: 2.5,
    blockLightRange: 1.4,
    blockLightShadows: 2,
    lightBudget: 24,
    studioSpot: true,
    spotColor: "#ffaa66",
    spotY: -0.4,
    volumeLightPower: 1.2,
    sky: "sunset",
    atmosphere: {
      ...renderDefaults.atmosphere,
      clouds: true,
      cloudReference: "bottom" as const,
      cloudHeight: -32,
      cloudSeed: 901,
    },
  };
  const encoded = encodeRenderStyle("Insel am Evening", settings);
  const restored = decodeRenderStyle(encoded);
  expect(restored.name).toBe("Insel am Evening");
  expect(restored.settings).toEqual(settings);
  await saveRenderStyle(restored);
  settings.atmosphere.cloudHeight = 200;
  expect((await loadRenderStyles())[0].settings.atmosphere.cloudHeight).toBe(
    -32,
  );
  expect(() =>
    decodeRenderStyle(
      '{"format":"craftbox-render-style","version":1,"name":"Bad","settings":{"exposure":"bright"}}',
    ),
  ).toThrow();
  expect(() =>
    decodeRenderStyle(
      JSON.stringify({
        format: "craftbox-render-style",
        version: 1,
        name: "Bad shadow",
        settings: { shadow: -1 },
      }),
    ),
  ).toThrow();
});
it("preserves bundled looks and embedded custom HDR data", () => {
  for (const [name, settings] of Object.entries(renderPresets)) {
    const complete = { ...renderDefaults, ...settings };
    expect(
      decodeRenderStyle(encodeRenderStyle(name, complete)).settings,
    ).toEqual(complete);
  }
  const asset = { name: "studio.hdr", data: btoa("#?RADIANCE\nHDR fixture") };
  const style = decodeRenderStyle(
    encodeRenderStyle(
      "Eigener Sky",
      { ...renderDefaults, sky: "custom" },
      asset,
    ),
  );
  expect(style.customSky).toEqual(asset);
  expect(() =>
    decodeRenderStyle(
      encodeRenderStyle("Missing HDR", { ...renderDefaults, sky: "custom" }),
    ),
  ).toThrow();
});

import { expect, it } from "vitest";
import { createNightSky } from "./night-sky";
it("keeps the night sky dark while retaining visible stars and faint diffuse illumination", () => {
  const sky = createNightSky();
  const pixels = sky.background.image.data as Float32Array;
  const fill = sky.lighting.image.data as Float32Array;
  let bright = 0,
    mean = 0,
    maxFill = 0;
  for (let i = 0; i < pixels.length; i += 4) {
    mean += pixels[i] + pixels[i + 1] + pixels[i + 2];
    if (pixels[i] > 0.2) bright++;
    maxFill = Math.max(maxFill, fill[i], fill[i + 1], fill[i + 2]);
  }
  expect(mean / (pixels.length / 4) / 3).toBeLessThan(0.015);
  expect(bright).toBeGreaterThan(300);
  expect(maxFill).toBeLessThan(0.04);
  sky.background.dispose();
  sky.lighting.dispose();
});

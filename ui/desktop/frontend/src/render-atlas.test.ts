import { it, expect } from "vitest";
import {
  AtlasMipmaps,
  atlasTileAttributes,
  filteredAlphaTestChunk,
} from "./render-atlas";
it("uses filtered opacity as MSAA coverage rather than discarding sparse mipmapped cutouts", () => {
  expect(filteredAlphaTestChunk).toContain("filteredCoverage");
  expect(filteredAlphaTestChunk).toContain("ALPHA_TO_COVERAGE");
  expect(filteredAlphaTestChunk).toContain("vAtlasTile.z");
});
it("averages a checkerboard in linear light without mixing neighboring tiles", () => {
  const mips = new AtlasMipmaps(36);
  const pixels = new Uint8Array(18 * 18 * 4);
  for (let y = 0; y < 18; y++)
    for (let x = 0; x < 18; x++) {
      const i = (y * 18 + x) * 4;
      pixels[i] = (x + y) % 2 ? 255 : 0;
      pixels[i + 3] = 255;
    }
  mips.update([{ x: 0, y: 0, size: 18, pixels }]);
  const last = mips.textures[3].image;
  const i = (1 * last.width + 1) * 4;
  expect(last.data![i]).toBeGreaterThan(180);
  expect(last.data![i]).toBeLessThan(195);
  expect(last.data![i + 1]).toBe(0);
  expect(last.data![i + 3]).toBe(255);
  mips.dispose();
});
it("keeps transparent color out of lower mip colors", () => {
  const mips = new AtlasMipmaps(36),
    pixels = new Uint8Array(18 * 18 * 4);
  for (let y = 0; y < 18; y++)
    for (let x = 0; x < 18; x++) {
      const i = (y * 18 + x) * 4;
      if ((x + y) % 2) {
        pixels[i] = 255;
        pixels[i + 3] = 255;
      } else pixels[i + 2] = 255;
    }
  mips.update([{ x: 0, y: 0, size: 18, pixels }]);
  const last = mips.textures[3].image,
    i = (last.width + 1) * 4;
  expect(last.data![i]).toBe(255);
  expect(last.data![i + 2]).toBe(0);
  expect(last.data![i + 3]).toBeCloseTo(128, 0);
  mips.dispose();
});
it("uses isolated mip tiles only for faces inside a single standard texture cell", () => {
  const uv = new Float32Array([1, 1, 17, 1, 17, 17, 1, 17].map((v) => v / 36));
  expect([...atlasTileAttributes(uv, 36)]).toEqual([
    0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1,
  ]);
  uv[2] = 35 / 36;
  expect(atlasTileAttributes(uv, 36)[2]).toBe(0);
});

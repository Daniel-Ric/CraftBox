import { it, expect } from "vitest";
import { encodeGIF, exportDimensions } from "./cinematic-export";
it("preserves GIF duration at 12 fps despite centisecond timing", () => {
  const data = new Uint8ClampedArray(4 * 4 * 4).fill(255);
  const gif = encodeGIF(
    Array.from({ length: 12 }, () => ({ data, width: 4, height: 4 })),
    12,
  );
  let duration = 0;
  for (let i = 0; i < gif.length - 7; i++)
    if (gif[i] === 0x21 && gif[i + 1] === 0xf9 && gif[i + 2] === 4)
      duration += gif[i + 4] + gif[i + 5] * 256;
  expect(duration).toBe(100);
});
it("encodes an animated GIF with a real GIF89a signature and trailer", () => {
  const rgba = new Uint8ClampedArray(4 * 4 * 4).fill(255);
  const gif = encodeGIF(
    [
      { data: rgba, width: 4, height: 4 },
      { data: rgba, width: 4, height: 4 },
    ],
    10,
  );
  expect(new TextDecoder().decode(gif.slice(0, 6))).toBe("GIF89a");
  expect(gif[gif.length - 1]).toBe(0x3b);
});
it("fits even video dimensions to the chosen aspect ratio", () => {
  expect(exportDimensions(1920, 16 / 9)).toEqual({ width: 1920, height: 1080 });
  expect(exportDimensions(640, 3 / 4)).toEqual({ width: 480, height: 640 });
});

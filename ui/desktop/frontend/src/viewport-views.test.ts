import { expect, it } from "vitest";
import { perspectives, polarLimits } from "./viewport-views";
it("locks the current inclination while leaving full rotation when unlocked", () => {
  expect(polarLimits(true, 1.1)).toEqual([1.1, 1.1]);
  expect(polarLimits(false, 1.1)).toEqual([0, Math.PI]);
  expect(perspectives.find((v) => v.id === "bottom")?.tilt).toBeLessThan(-89);
  expect(new Set(perspectives.map((v) => `${v.angle}:${v.tilt}`)).size).toBe(
    perspectives.length,
  );
});

import { expect, it } from "vitest";
import { billboardAttributes } from "./scene-billboard";

it("keeps rotated hologram dimensions and center when facing the camera", () => {
  const { centers, offsets } = billboardAttributes(new Float32Array([2, 4, 1, 2, 4, 5, 2, 6, 5, 2, 6, 1]));
  expect([...centers]).toEqual([2,5,3,2,5,3,2,5,3,2,5,3]);
  expect([...offsets]).toEqual([-2,-1,2,-1,2,1,-2,1]);
  expect(() => billboardAttributes(new Float32Array(3))).toThrow();
});

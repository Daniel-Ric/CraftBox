import { expect, it } from "vitest";
import { centerOnTiles, fitTiles } from "./map-camera";
it("centers on real terrain far from origin before the first player position arrives", () => {
  const view = centerOnTiles([
    { x: 128, z: -32, image: "" },
    { x: 129, z: -31, image: "" },
  ]);
  expect(view).toEqual({ x: 2064, z: -496 });
});
it("fits separated terrain into the viewport", () => {
  const view = fitTiles(
    [
      { x: 0, z: 0, image: "" },
      { x: 100, z: 40, image: "" },
    ],
    800,
    400,
  );
  expect(view.x).toBe(808);
  expect(view.z).toBe(328);
  expect(view.scale * 1616).toBeLessThanOrEqual(736);
});

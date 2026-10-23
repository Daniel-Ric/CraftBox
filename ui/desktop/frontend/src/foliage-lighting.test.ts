import { expect, it } from "vitest";
import { foliageAttributes } from "./foliage-lighting";
it("recognizes native vine sheets at 1/64 offsets without relighting solid cube faces", () => {
  const p = new Float32Array([
    0,
    0,
    1 / 64,
    1,
    0,
    1 / 64,
    1,
    1,
    1 / 64,
    0,
    1,
    1 / 64,
    63 / 64,
    1,
    0,
    63 / 64,
    0,
    0,
    63 / 64,
    0,
    1,
    63 / 64,
    1,
    1,
    0,
    0,
    0,
    1,
    0,
    0,
    1,
    1,
    0,
    0,
    1,
    0,
  ]);
  expect([...foliageAttributes(p)]).toEqual([
    2, 2, 2, 2, 2, 2, 2, 2, 0, 0, 0, 0,
  ]);
});
it("recognizes crossed plant planes and preserves cube and sloped surfaces", () => {
  const positions = new Float32Array([
    0, 0, 0, 1, 0, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0,
    0, 0, 1, 0, 1, 1, 1, 0, 0, 1, 1,
  ]);
  expect([...foliageAttributes(positions)]).toEqual([
    1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
  ]);
});

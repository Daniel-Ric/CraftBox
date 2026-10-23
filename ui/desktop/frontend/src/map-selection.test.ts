import { describe, it, expect } from "vitest";
import { selectedChunks, worldAtPointer } from "./map-selection";

describe("world region selection", () => {
  it("includes negative and reversed chunk boundaries", () => {
    expect(selectedChunks({ x: 16, z: 32 }, { x: -1, z: -17 })).toEqual({
      minX: -1,
      minZ: -2,
      maxX: 1,
      maxZ: 2,
    });
    expect(selectedChunks({ x: -16, z: -16 }, { x: -16, z: -16 })).toEqual({
      minX: -1,
      minZ: -1,
      maxX: -1,
      maxZ: -1,
    });
  });
  it("selects world coordinates after zoom and pan", () => {
    expect(
      worldAtPointer(220, 140, 400, 200, { x: 100, z: -100, scale: 2 }),
    ).toEqual({ x: 110, z: -80 });
  });
});

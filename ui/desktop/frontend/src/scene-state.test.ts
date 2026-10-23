import { describe, expect, it } from "vitest";
import { mergeChunks, sceneStats, SceneLedger } from "./scene-state";
import type { SceneChunk } from "./bridge";

const chunk = (x: number): SceneChunk => ({
  x,
  z: 0,
  missingTextures: 0,
  approximateBlocks: 0,
  parts: [
    {
      mode: "opaque",
      positions: [],
      normals: [],
      uvs: [],
      indices: [0, 1, 2, 0, 2, 3],
    },
  ],
});
describe("retained 3D chunks", () => {
  it("keeps replacement totals and enforces the memory budget before publishing", () => {
    const ledger = new SceneLedger();
    const item = {
      x: 0,
      z: 0,
      missingTextures: 2,
      approximateBlocks: 3,
      parts: [
        { indices: { length: 6 }, positions: { buffer: { byteLength: 100 } } },
      ],
    };
    ledger.update(item);
    ledger.update(item);
    expect(ledger.snapshot()).toEqual({
      chunks: 1,
      triangles: 2,
      missing: 2,
      approximate: 3,
    });
    expect(ledger.bytes).toBe(100);
    expect(() =>
      ledger.update({
        ...item,
        x: 1,
        parts: [
          {
            indices: { length: 3 },
            positions: { buffer: { byteLength: 256 * 1024 * 1024 } },
          },
        ],
      }),
    ).toThrow();
    expect(ledger.snapshot().chunks).toBe(1);
    ledger.update({
      ...item,
      parts: [],
      missingTextures: 0,
      approximateBlocks: 0,
    });
    expect(ledger.snapshot().triangles).toBe(0);
    expect(ledger.bytes).toBe(0);
    ledger.clear();
    expect(ledger.snapshot().chunks).toBe(0);
  });
  it("replaces changed chunks, including emptied ones, without duplicating geometry", () => {
    const first = mergeChunks(new Map(), [chunk(0), chunk(1)], true);
    const next = mergeChunks(first, [{ ...chunk(1), parts: [] }], false);
    expect(sceneStats(next).triangles).toBe(2);
    expect(first.get("1,0")?.parts.length).toBe(1);
    expect(next.size).toBe(2);
  });
  it("drops the previous world's geometry on reset", () => {
    const previous = mergeChunks(new Map(), [chunk(0)], true);
    expect([...mergeChunks(previous, [chunk(9)], true).keys()]).toEqual([
      "9,0",
    ]);
  });
});

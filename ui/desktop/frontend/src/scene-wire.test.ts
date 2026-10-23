import { describe, it, expect } from "vitest";
import { decodePart, decodePatch } from "./scene-wire";
import { FrameQueue } from "./scene-queue";

describe("compact 3D transport", () => {
  it("reuses one buffer with exact float vertices, UVs and compact normalized normals", () => {
    const buffer = new ArrayBuffer(84);
    new Float32Array(buffer, 0, 9).set([0, 1, 0, 1, 1, 0, 0, 2, 0]);
    new Int8Array(buffer, 36, 9).set([0, 0, -127, 0, 0, -127, 0, 0, -127]);
    new Float32Array(buffer, 48, 6).set([0, 0, 1, 0, 0, 1]);
    new Uint32Array(buffer, 72, 3).set([0, 1, 2]);
    const data = btoa(String.fromCharCode(...new Uint8Array(buffer)));
    const part = decodePart({ mode: "opaque", vertices: 3, indices: 3, data });
    expect([...part.positions]).toEqual([0, 1, 0, 1, 1, 0, 0, 2, 0]);
    expect([...part.normals]).toEqual([0, 0, -127, 0, 0, -127, 0, 0, -127]);
    expect(part.positions.buffer).toBe(part.indices.buffer);
    expect(part.bounds).toEqual([0, 1, 0, 1, 2, 0]);
    expect(() =>
      decodePart({ mode: "opaque", vertices: 4, indices: 3, data }),
    ).toThrow();
  });
  it("rejects atlas patches outside their texture", () => {
    expect(() =>
      decodePatch({ x: 2047, y: 0, size: 18, data: "" }, 2048),
    ).toThrow();
    expect([
      ...decodePatch(
        { x: 0, y: 0, size: 1, data: btoa("\x01\x02\x03\xff") },
        2048,
      ).pixels,
    ]).toEqual([1, 2, 3, 255]);
  });
});

describe("frame-budgeted geometry uploads", () => {
  it("yields between batches and only acknowledges committed chunks", async () => {
    let clock = 0;
    const committed: number[] = [];
    const queue = new FrameQueue<number>(
      (value) => {
        committed.push(value);
        clock += 3;
      },
      () => clock,
      4,
      2,
    );
    let complete = false;
    const done = queue.append([1, 2, 3, 4]).then(() => {
      complete = true;
    });
    expect(queue.flush()).toBe(2);
    await Promise.resolve();
    expect(committed).toEqual([1, 2]);
    expect(complete).toBe(false);
    queue.flush();
    await done;
    expect(committed).toEqual([1, 2, 3, 4]);
    expect(complete).toBe(true);
  });
  it("discards pending GPU work when the view closes", async () => {
    const committed: number[] = [];
    const queue = new FrameQueue<number>((value) => committed.push(value));
    const done = queue.append([1]);
    queue.cancel();
    await expect(done).rejects.toThrow();
    expect(queue.flush()).toBe(0);
    expect(committed).toEqual([]);
  });
});

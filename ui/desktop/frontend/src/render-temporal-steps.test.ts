import { expect, it, vi } from "vitest";
import { renderTemporalSteps } from "./render-temporal-steps";

it("yields between temporal samples and closes the render transaction on cancellation", async () => {
  const controller = new AbortController(),
    events: string[] = [];
  function* samples() {
    try {
      for (let i = 0; i < 64; i++) {
        events.push(`sample:${i}`);
        yield;
      }
    } finally {
      events.push("restore");
    }
  }
  const yielding = vi.fn(async () => {
    events.push("yield");
    controller.abort();
  });
  await expect(
    renderTemporalSteps(samples(), controller.signal, yielding),
  ).rejects.toThrow();
  expect(events).toEqual(["sample:0", "yield", "restore"]);
  expect(yielding).toHaveBeenCalledTimes(1);
});

it("finishes every sample in order and rejects before drawing an already cancelled frame", async () => {
  const draw = vi.fn(),
    restore = vi.fn(),
    yielding = vi.fn(async () => {});
  function* samples() {
    try {
      for (let i = 0; i < 3; i++) {
        draw(i);
        yield;
      }
    } finally {
      restore();
    }
  }
  await renderTemporalSteps(samples(), undefined, yielding);
  expect(draw.mock.calls.flat()).toEqual([0, 1, 2]);
  expect(restore).toHaveBeenCalledOnce();
  draw.mockClear();
  await expect(
    renderTemporalSteps(samples(), AbortSignal.abort(), yielding),
  ).rejects.toThrow();
  expect(draw).not.toHaveBeenCalled();
});

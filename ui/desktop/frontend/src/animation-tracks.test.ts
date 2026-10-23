import { expect, it } from "vitest";
import {
  sampleTrack,
  moveTrackKey,
  upsertTrackKey,
  type AnimationTrack,
} from "./animation-tracks";

const track = (): AnimationTrack => ({
  channel: "fov",
  keys: [
    {
      id: "a",
      time: 0,
      value: 20,
      interpolation: "linear",
      incoming: [-1, 0],
      outgoing: [1, 0],
    },
    {
      id: "b",
      time: 4,
      value: 60,
      interpolation: "linear",
      incoming: [-1, 0],
      outgoing: [1, 0],
    },
  ],
});
it("samples independent channels on their own key times with endpoint holds", () => {
  const lens = track(),
    exposure = {
      ...track(),
      channel: "exposure" as const,
      keys: track().keys.map((k) => ({
        ...k,
        time: k.time + 2,
        value: k.value / 40,
      })),
    };
  expect(sampleTrack(lens, 1, 45)).toBe(30);
  expect(sampleTrack(exposure, 1, 1)).toBe(0.5);
  expect(sampleTrack(lens, 10, 45)).toBe(60);
  expect(sampleTrack({ ...lens, keys: [] }, 1, 45)).toBe(45);
});
it("solves weighted Bezier time handles, supports value overshoot and exact cuts", () => {
  const curve = track();
  curve.keys[0].interpolation = "bezier";
  curve.keys[0].outgoing = [0.4, 80];
  curve.keys[1].incoming = [-0.4, 20];
  expect(sampleTrack(curve, 2, 45)).toBeGreaterThan(60);
  curve.keys[0].interpolation = "hold";
  expect(sampleTrack(curve, 3.99, 45)).toBe(20);
  expect(sampleTrack(curve, 4, 45)).toBe(60);
});
it("moves keys without deleting crossed keys or colliding with another frame", () => {
  const original = track();
  const moved = moveTrackKey(original, "a", 5, 24, 10);
  expect(moved.keys.map((k) => k.id)).toEqual(["b", "a"]);
  expect(original.keys[0].time).toBe(0);
  expect(moveTrackKey(original, "a", 4.001, 24, 10)).toBe(original);
  const inserted = upsertTrackKey(
    original,
    { ...original.keys[0], id: "new", time: 1.02 },
    24,
  );
  expect(inserted.keys).toHaveLength(3);
  expect(inserted.keys[1].time).toBe(1);
});
it("keeps snapped keys inside a duration that falls between frames", () => {
  const moved = moveTrackKey(track(), "b", 0.12, 24, 0.12);
  expect(moved.keys.find((key) => key.id === "b")!.time).toBeLessThanOrEqual(
    0.12,
  );
});

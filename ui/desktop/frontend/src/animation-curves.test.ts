import { expect, it } from "vitest";
import { sampleTrack, parseAnimationTracks, type AnimationTrack } from "./animation-tracks";
import { graphPath, sampleVelocity, setCurveTangent, autoCurveKey, splitCurveAt } from "./animation-curves";

const curve = (): AnimationTrack => ({ channel: "fov", keys: [
  { id: "a", time: 0, value: 20, interpolation: "bezier", incoming: [0, 0], outgoing: [1, 0] },
  { id: "b", time: 4, value: 60, interpolation: "bezier", incoming: [-1, 0], outgoing: [1, 0] },
] });
it("draws exactly the weighted Bezier controls used by the sampler", () => {
  const path = graphPath(curve(), (t) => t * 10, (v) => v);
  expect(path).toBe("M0,20 C10,20 30,60 40,60");
  const at = sampleTrack(curve(), 2, 0);
  expect(at).toBeCloseTo(40);
  expect(sampleVelocity(curve(), 2)).toBeCloseTo((sampleTrack(curve(), 2.00001, 0) - sampleTrack(curve(), 1.99999, 0)) / 0.00002, 4);
});
it("links tangent slopes without silently changing the opposite handle length", () => {
  const key = { ...curve().keys[0], incoming: [-2, -3] as [number, number] };
  const aligned = setCurveTangent(key, "outgoing", [1, 4], "aligned");
  expect(aligned.incoming).toEqual([-2, -8]);
  expect(setCurveTangent(key, "outgoing", [1, 4], "mirrored").incoming).toEqual([-1, -4]);
  expect(setCurveTangent(key, "outgoing", [1, 4], "free").incoming).toEqual([-2, -3]);
});
it("automatic tangents stay flat at reversals and speed stays zero through held cuts", () => {
  const track = curve(); track.keys.push({ ...track.keys[0], id: "c", time: 8, value: 20 });
  expect(autoCurveKey(track, "b").keys[1].outgoing[1]).toBe(0);
  track.keys[0].interpolation = "hold";
  expect(sampleVelocity(track, 3.999)).toBe(0);
  expect(graphPath(track, (t) => t, (v) => v)).toContain("L4,20 L4,60");
});
it("keeps inserted keys and extreme linked handles within reloadable document limits", () => {
  const track = curve(); track.keys[0].outgoing[1] = 1000;
  const split = splitCurveAt(track, 2, "split");
  expect(split.keys[1].value).toBe(120);
  expect(() => parseAnimationTracks([split], 4)).not.toThrow();
  const linked = setCurveTangent(track.keys[0], "incoming", [-0.00001, 100], "aligned");
  expect(Math.abs(linked.outgoing[1])).toBeLessThanOrEqual(1000000);
  expect(linked.outgoing[1] / linked.outgoing[0]).toBeCloseTo(-10000000);
  expect(() => parseAnimationTracks([{ ...track, keys: [linked, track.keys[1]] }], 4)).not.toThrow();
});

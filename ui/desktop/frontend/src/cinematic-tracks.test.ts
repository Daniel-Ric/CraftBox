import { expect, it } from "vitest";
import { createSequence, sampleSequence, parseSequence } from "./cinematic";
import { independentSequence, editPoseKey, addChannelKey, changeSequenceFPS, insertPoseKey, removePoseKey, minimumSequenceDuration, duplicateChannelKey } from "./cinematic-tracks";
import { moveTrackKey } from "./animation-tracks";

it("migrates the original curved flight without changing sampled positions, lens or angular motion", () => {
  const legacy = createSequence();
  legacy.duration = 9;
  legacy.keys = [0, 1, 4, 9].map((time, i) => ({ ...legacy.keys[0], id: String(i), time, position: [i * 0.8, i % 2, i * -0.4], target: [i * 0.1, 0.4, -0.3], fov: [25, 70, 35, 55][i], roll: [170, -170, -150, 175][i] }));
  const migrated = independentSequence(legacy);
  expect(migrated.version).toBe(2);
  for (let t = 0; t <= 9; t += 0.037) {
    const a = sampleSequence(legacy, t), b = sampleSequence(migrated, t);
    a.position.forEach((v, axis) => expect(b.position[axis]).toBeCloseTo(v, 7));
    a.target.forEach((v, axis) => expect(b.target[axis]).toBeCloseTo(v, 7));
    expect(b.fov).toBeCloseTo(a.fov, 7);
    expect(Math.sin(b.roll * Math.PI / 180)).toBeCloseTo(Math.sin(a.roll * Math.PI / 180), 7);
  }
});
it("moves only the selected channel and keeps sibling key times and values intact", () => {
  const s = independentSequence(createSequence());
  const position = structuredClone(s.tracks!.find((t) => t.channel === "position.x"));
  s.tracks = s.tracks!.map((t) => t.channel === "fov" ? moveTrackKey(t, t.keys[1].id, 3, 24, 12) : t);
  expect(s.tracks.find((t) => t.channel === "position.x")).toEqual(position);
  expect(sampleSequence(s, 3).fov).toBeCloseTo(40);
  expect(parseSequence(JSON.stringify(s))).toEqual(s);
});
it("keeps full-pose edits in sync without overwriting independently authored lens keys", () => {
  let s = independentSequence(createSequence());
  s = addChannelKey(s, "fov", 2, 80);
  s = editPoseKey(s, s.keys[0].id, { position: [5, 2, 1] });
  expect(sampleSequence(s, 0).position).toEqual([5, 2, 1]);
  expect(sampleSequence(s, 2).fov).toBeCloseTo(80);
});
it("retains subframe keys when changing FPS instead of silently deleting collided keys", () => {
  let s = independentSequence({ ...createSequence(), fps: 60 });
  s = addChannelKey(s, "fov", 1 / 60, 70);
  const changed = changeSequenceFPS(s, 24);
  expect(changed.tracks!.find((t) => t.channel === "fov")!.keys).toHaveLength(4);
  expect(parseSequence(JSON.stringify(changed)).fps).toBe(24);
});
it("rejects invalid imported channel keys, duplicate channels and tangent data", () => {
  const s = independentSequence(createSequence());
  s.tracks![0].keys[0].outgoing = [NaN, 1];
  expect(() => parseSequence(JSON.stringify(s))).toThrow();
  const duplicate = independentSequence(createSequence());
  duplicate.tracks!.push(duplicate.tracks![0]);
  expect(() => parseSequence(JSON.stringify(duplicate))).toThrow();
});
it("samples focus pulls and aperture separately from camera movement", () => {
  let s = independentSequence(createSequence(), { enabled: true, distance: 100, aperture: 2.8 });
  s = addChannelKey(s, "focusDistance", 0, 100);
  s = addChannelKey(s, "focusDistance", 6, 500);
  s = addChannelKey(s, "aperture", 0, 1.4);
  s = addChannelKey(s, "aperture", 12, 8);
  const middle = sampleSequence(s, 3);
  expect(middle.depthOfField).toBe(true);
  expect(middle.focusDistance).toBeGreaterThan(100);
  expect(middle.focusDistance).toBeLessThan(500);
  expect(middle.aperture).toBeGreaterThan(1.4);
  expect(sampleSequence(parseSequence(JSON.stringify(s)), 3)).toEqual(middle);
});
it("duplicates and deletes complete poses in the actual channel data", () => {
  const original = independentSequence(createSequence());
  const copied = insertPoseKey(original, { ...original.keys[0], id: "copy", time: 2, fov: 80 });
  expect(sampleSequence(copied, 2).fov).toBe(80);
  const removed = removePoseKey(copied, "copy");
  expect(removed.tracks!.every((track) => track.keys.every((key) => key.time !== 2))).toBe(true);
});
it("keeps value edits from deleting subframe neighbors and blocks conflicting whole-pose moves atomically", () => {
  let s = independentSequence({ ...createSequence(), fps: 60 });
  s = addChannelKey(s, "fov", 1 / 60, 80);
  s = changeSequenceFPS(s, 24);
  const edited = editPoseKey(s, s.keys[0].id, { fov: 30 });
  expect(edited.tracks!.find((t) => t.channel === "fov")!.keys).toHaveLength(4);
  const blocked = addChannelKey(edited, "fov", 1 / 24, 60);
  expect(editPoseKey(blocked, blocked.keys[0].id, { time: 1 / 24 })).toBe(blocked);
});
it("inserts a sampled Bezier key without changing the authored curve", () => {
  const original = independentSequence(createSequence());
  const inserted = addChannelKey(original, "fov", 2);
  for (let t = 0; t <= 12; t += 0.039) expect(sampleSequence(inserted, t).fov).toBeCloseTo(sampleSequence(original, t).fov, 6);
});
it("includes independent keys in the minimum duration and reloads multi-turn pose rolls", () => {
  const s = independentSequence(createSequence());
  s.keys = [s.keys[0]];
  expect(minimumSequenceDuration(s)).toBe(12);
  const inserted = insertPoseKey(s, { ...s.keys[0], id: "spin", time: 2, roll: 540 });
  expect(parseSequence(JSON.stringify(inserted)).keys.find((k) => k.id === "spin")!.roll).toBe(540);
});
it("duplicates into a free frame, including backwards from the sequence end, and preserves occupied keys", () => {
  const original = independentSequence(createSequence());
  const track = original.tracks!.find((t) => t.channel === "fov")!;
  const copy = duplicateChannelKey(original, "fov", track.keys.at(-1)!.id);
  const keys = copy.tracks!.find((t) => t.channel === "fov")!.keys;
  expect(keys).toHaveLength(4);
  for (const key of track.keys) expect(keys.find((k) => k.id === key.id)).toEqual(key);
  expect(keys.at(-2)!.time).toBeCloseTo(12 - 1 / 24);
});

import { expect, it } from "vitest";
import { moveKeyframe } from "./cinematic";
it("does not delete other keyframes when a drag crosses an occupied frame", () => {
  const s = createSequence();
  expect(moveKeyframe(s.keys, "start", 6, 24)).toEqual(s.keys);
  const moved = moveKeyframe(s.keys, "start", 8, 24);
  expect(moved).toHaveLength(3);
  expect(moved.find((k) => k.id === "middle")?.time).toBe(6);
});
import {
  createSequence,
  sampleSequence,
  upsertKeyframe,
  parseSequence,
  timecode,
} from "./cinematic";
it("interpolates camera and lens, clamps time and reaches exact endpoints", () => {
  const sequence = createSequence();
  expect(sampleSequence(sequence, -10)).toEqual(sampleSequence(sequence, 0));
  expect(sampleSequence(sequence, 100)).toEqual(
    sampleSequence(sequence, sequence.duration),
  );
  const a = sampleSequence(sequence, 0),
    b = sampleSequence(sequence, sequence.duration);
  expect(a.position).not.toEqual(b.position);
  expect(
    sampleSequence(sequence, sequence.duration / 2).position.every(
      Number.isFinite,
    ),
  ).toBe(true);
});
it("replaces keyframes on the same frame and keeps chronological order", () => {
  const s = createSequence();
  const key = { ...s.keys[0], id: "new", time: 4, fov: 30 };
  const keys = upsertKeyframe(
    upsertKeyframe(s.keys, key, 24),
    { ...key, id: "replace", time: 4.001, fov: 70 },
    24,
  );
  expect(keys.filter((k) => Math.abs(k.time - 4) < 0.1)).toHaveLength(1);
  expect(keys.map((k) => k.time)).toEqual(
    [...keys.map((k) => k.time)].sort((a, b) => a - b),
  );
});
it("round trips projects and rejects corrupt or unbounded camera data", () => {
  const s = createSequence();
  expect(parseSequence(JSON.stringify(s))).toEqual(s);
  expect(() => parseSequence('{"duration":0}')).toThrow();
  s.keys[0].position[0] = Infinity;
  expect(() => parseSequence(JSON.stringify(s))).toThrow();
});
it("formats frame accurate timecode", () => {
  expect(timecode(61.5, 24)).toBe("00:01:01:12");
});
it("round trips a 50 fps project without discarding its timeline", () => {
  const sequence = { ...createSequence(), fps: 50, name: "Edited film" };
  expect(parseSequence(JSON.stringify(sequence))).toEqual(sequence);
});
it("flies through intermediate smooth keys with matching nonzero velocity at uneven times", () => {
  const sequence = createSequence();
  sequence.duration = 9;
  sequence.keys = [0, 1, 4, 9].map((time, i) => ({
    ...sequence.keys[0],
    id: String(i),
    time,
    position: [time, 0, 0],
    target: [time * 0.5, 0, -1],
    fov: 30 + time,
    roll: time,
    exposure: 1 + time * 0.1,
  }));
  const epsilon = 0.0001;
  for (const time of [1, 4]) {
    const before = sampleSequence(sequence, time - epsilon),
      key = sampleSequence(sequence, time),
      after = sampleSequence(sequence, time + epsilon);
    expect(key.position).toEqual([time, 0, 0]);
    for (const field of ["position", "target"] as const) {
      const left = (key[field][0] - before[field][0]) / epsilon,
        right = (after[field][0] - key[field][0]) / epsilon;
      expect(left).toBeGreaterThan(field === "position" ? 0.8 : 0.4);
      expect(right).toBeCloseTo(left, 2);
    }
    for (const field of ["fov", "roll", "exposure"] as const) {
      const left = (key[field] - before[field]) / epsilon,
        right = (after[field] - key[field]) / epsilon;
      expect(left).toBeGreaterThan(0.05);
      expect(right).toBeCloseTo(left, 2);
    }
  }
});
it("preserves holds and linear timing when smoothing a flight", () => {
  const sequence = createSequence();
  sequence.keys[0].ease = "hold";
  sequence.keys[1].ease = "linear";
  expect(sampleSequence(sequence, 3).position).toEqual(
    sequence.keys[0].position,
  );
  const middle = sampleSequence(sequence, 9);
  expect(middle.position[0]).toBeCloseTo(
    (sequence.keys[1].position[0] + sequence.keys[2].position[0]) / 2,
  );
  expect(sampleSequence(sequence, 6).position).toEqual(
    sequence.keys[1].position,
  );
});
it("keeps a curved flight moving through a corner and eases only the flight endpoints", () => {
  const sequence = createSequence();
  sequence.duration = 2;
  sequence.keys = sequence.keys.map((key, i) => ({
    ...key,
    time: i,
    position: (
      [
        [0, 0, 0],
        [1, 0, 0],
        [1, 1, 0],
      ] as [number, number, number][]
    )[i],
  }));
  const epsilon = 0.0001,
    at = sampleSequence(sequence, 1),
    before = sampleSequence(sequence, 1 - epsilon),
    after = sampleSequence(sequence, 1 + epsilon);
  const incoming = at.position.map(
      (v, i) => (v - before.position[i]) / epsilon,
    ),
    outgoing = after.position.map((v, i) => (v - at.position[i]) / epsilon);
  expect(Math.hypot(...incoming)).toBeGreaterThan(0.5);
  incoming.forEach((v, i) => expect(outgoing[i]).toBeCloseTo(v, 2));
  expect(
    Math.hypot(...sampleSequence(sequence, epsilon).position) / epsilon,
  ).toBeLessThan(0.001);
});
it("does not lose cruising speed at a curved intermediate waypoint", () => {
  const sequence = createSequence();
  sequence.duration = 4;
  sequence.keys = [
    [0, 0, 0],
    [1, 0, 0],
    [2, 0, 0],
    [2, 1, 0],
    [2, 2, 0],
  ].map((position, index) => ({
    ...sequence.keys[0],
    id: String(index),
    time: index,
    position: position as [number, number, number],
  }));
  const epsilon = 0.00001;
  const at = sampleSequence(sequence, 2).position;
  const before = sampleSequence(sequence, 2 - epsilon).position;
  const after = sampleSequence(sequence, 2 + epsilon).position;
  const incoming = at.map((v, axis) => (v - before[axis]) / epsilon);
  const outgoing = after.map((v, axis) => (v - at[axis]) / epsilon);
  expect(Math.hypot(...incoming)).toBeCloseTo(1, 3);
  expect(Math.hypot(...outgoing)).toBeCloseTo(1, 3);
  incoming.forEach((v, axis) => expect(outgoing[axis]).toBeCloseTo(v, 3));
});
it("does not amplify a near reversal into an overshoot beyond the turning point", () => {
  const sequence = createSequence();
  sequence.duration = 2;
  sequence.keys = sequence.keys.map((key, index) => ({
    ...key,
    time: index,
    position: [
      [0, 0, 0],
      [1, 0, 0],
      [0.000001, 0, 0],
    ][index] as [number, number, number],
  }));
  for (let time = 0; time <= 2; time += 0.01) {
    const x = sampleSequence(sequence, time).position[0];
    expect(x).toBeGreaterThanOrEqual(0);
    expect(x).toBeLessThanOrEqual(1 + 1e-9);
  }
});
it("keeps lens and exposure bounded and rolls over the short angular path", () => {
  const sequence = createSequence();
  sequence.keys[0].roll = 170;
  sequence.keys[1].roll = -170;
  sequence.keys[2].roll = -150;
  sequence.keys[0].fov = 10;
  sequence.keys[1].fov = 120;
  sequence.keys[2].fov = 10;
  sequence.keys[0].exposure = 0.1;
  sequence.keys[1].exposure = 4;
  sequence.keys[2].exposure = 0.1;
  for (let time = 0; time < sequence.duration; time += 0.05) {
    const pose = sampleSequence(sequence, time);
    expect(pose.fov).toBeGreaterThanOrEqual(10);
    expect(pose.fov).toBeLessThanOrEqual(120);
    expect(pose.exposure).toBeGreaterThanOrEqual(0.1);
    expect(pose.exposure).toBeLessThanOrEqual(4);
  }
  expect(sampleSequence(sequence, 3).roll).toBeGreaterThan(170);
});
it("matches lens velocities when joining linear and smooth segments in either direction", () => {
  for (const modes of [
    ["linear", "smooth"],
    ["smooth", "linear"],
  ] as const) {
    const sequence = createSequence();
    sequence.duration = 4;
    sequence.keys = sequence.keys.map((key, i) => ({
      ...key,
      time: [0, 1, 4][i],
      fov: [10, 20, 80][i],
      roll: [0, 10, 70][i],
      exposure: [1, 1.2, 2.4][i],
      ease: modes[Math.min(i, 1)],
    }));
    const epsilon = 0.0001,
      before = sampleSequence(sequence, 1 - epsilon),
      at = sampleSequence(sequence, 1),
      after = sampleSequence(sequence, 1 + epsilon);
    for (const field of ["fov", "roll", "exposure"] as const) {
      expect((after[field] - at[field]) / epsilon).toBeCloseTo(
        (at[field] - before[field]) / epsilon,
        2,
      );
    }
  }
});

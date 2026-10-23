import { expect, it } from "vitest";
import { Vector3 } from "three";
import { createSequence, parseSequence, sampleSequence } from "./cinematic";
import { addChannelKey, independentSequence } from "./cinematic-tracks";
import { addCamera, enableCameras, splitShot } from "./cinematic-shots";
import {
  addTrackingTarget,
  editTargetPosition,
  removeTrackingTarget,
  sampleTarget,
} from "./cinematic-tracking";

it("aims at an independently animated target including offset and preserves zero-weight camera motion", () => {
  let s = addTrackingTarget(independentSequence(createSequence()), [0, 0, 0]);
  const id = s.targets![0].id;
  s = editTargetPosition(s, id, 0, [0, 0, 0]);
  s = editTargetPosition(s, id, 12, [2, 1, 0]);
  s.tracking = {
    enabled: true,
    targetId: id,
    offset: [0, 0.5, 0],
    weight: 1,
    focus: true,
  };
  const pose = sampleSequence(s, 6),
    point = sampleTarget(s.targets![0], 6);
  expect(pose.target).toEqual([point[0], point[1] + 0.5, point[2]]);
  expect(pose.focusTarget).toEqual(pose.target);
  s = addChannelKey(s, "trackingWeight", 0, 0);
  expect(sampleSequence(s, 6).target).toEqual(
    sampleSequence({ ...s, tracking: undefined }, 6).target,
  );
  expect(parseSequence(JSON.stringify(s))).toEqual(s);
});
it("uses program time for shared targets while each shot remaps camera animation time", () => {
  let s = addTrackingTarget(
    addCamera(enableCameras(independentSequence(createSequence()))),
  );
  const id = s.targets![0].id;
  s = editTargetPosition(s, id, 0, [0, 0, 0]);
  s = editTargetPosition(s, id, 12, [12, 0, 0]);
  s.cameras = s.cameras!.map((c) => ({
    ...c,
    tracking: {
      enabled: true,
      targetId: id,
      offset: [0, 0, 0],
      weight: 1,
      focus: false,
    },
  }));
  s = splitShot(s, s.shots![0].id, 6);
  s.shots![1] = {
    ...s.shots![1],
    cameraId: s.cameras![1].id,
    sourceIn: 0,
    rate: 0.5,
  };
  expect(sampleSequence(s, 8).target).toEqual(sampleTarget(s.targets![0], 8));
  expect(parseSequence(JSON.stringify(s))).toEqual(s);
});
it("blends aim in angular space without collapsing opposite targets and tolerates coincident points", () => {
  let s = addTrackingTarget(
    independentSequence(createSequence()),
    [1.8, 0.8, 3.8],
  );
  s.keys = [{ ...s.keys[0], target: [1.8, 0.8, -0.2] }];
  s.tracks = [];
  s.tracking = {
    enabled: true,
    targetId: s.targets![0].id,
    offset: [0, 0, 0],
    weight: 0.5,
    focus: false,
  };
  const p = sampleSequence(s, 0);
  expect(
    new Vector3(...p.target).distanceTo(new Vector3(...p.position)),
  ).toBeCloseTo(2);
  s.targets![0].position = [...p.position];
  expect(sampleSequence(s, 0).target.every(Number.isFinite)).toBe(true);
});
it("rejects broken tracking references and removes constraints together with a deleted target", () => {
  const s = addTrackingTarget(independentSequence(createSequence()));
  s.tracking = {
    enabled: true,
    targetId: "missing",
    offset: [0, 0, 0],
    weight: 1,
    focus: false,
  };
  expect(() => parseSequence(JSON.stringify(s))).toThrow();
  s.tracking.targetId = s.targets![0].id;
  const removed = removeTrackingTarget(s, s.targets![0].id);
  expect(removed.tracking).toBeUndefined();
  expect(() => parseSequence(JSON.stringify(removed))).not.toThrow();
});
it("inserts generic tracking keys at the actual constraint influence rather than full strength", () => {
  let s = addTrackingTarget(independentSequence(createSequence()));
  s.tracking = {
    enabled: true,
    targetId: s.targets![0].id,
    offset: [0, 0, 0],
    weight: 0.3,
    focus: false,
  };
  s = addChannelKey(s, "trackingWeight", 2);
  expect(
    s.tracks!.find((t) => t.channel === "trackingWeight")!.keys[0].value,
  ).toBe(0.3);
});

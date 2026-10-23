import { MathUtils, Quaternion, Vector3 } from "three";
import {
  parseAnimationTracks,
  sampleTrack,
  type AnimationTrack,
} from "./animation-tracks";
import { addChannelKey, independentSequence } from "./cinematic-tracks";
import type { CameraPose, Sequence, Vec3 } from "./cinematic";

export type TrackingTarget = {
  id: string;
  name: string;
  position: Vec3;
  tracks: AnimationTrack[];
};
export type CameraTracking = {
  enabled: boolean;
  targetId: string;
  offset: Vec3;
  weight: number;
  focus: boolean;
};
export const targetChannels = [
  "position.x",
  "position.y",
  "position.z",
] as const;
export function sampleTarget(target: TrackingTarget, time: number): Vec3 {
  return targetChannels.map((channel, i) => {
    const track = target.tracks.find((t) => t.channel === channel);
    return track
      ? sampleTrack(track, time, track.defaultValue ?? target.position[i])
      : target.position[i];
  }) as Vec3;
}
export function targetSequence(
  project: Sequence,
  target: TrackingTarget,
): Sequence {
  return {
    version: 2,
    name: target.name,
    duration: project.duration,
    fps: project.fps,
    keys: [
      {
        id: target.id,
        time: 0,
        position: target.position,
        target: [0, 0, 0],
        fov: 45,
        roll: 0,
        exposure: 1,
        ease: "smooth",
      },
    ],
    tracks: target.tracks,
  };
}
export function addTrackingTarget(
  project: Sequence,
  position: Vec3 = [0, 0, 0],
): Sequence {
  if ((project.targets?.length ?? 0) >= 64)
    throw new Error("You can add up to 64 tracking targets.");
  return {
    ...independentSequence(project),
    targets: [
      ...(project.targets ?? []),
      {
        id: crypto.randomUUID(),
        name: `Target ${1 + (project.targets?.length ?? 0)}`,
        position,
        tracks: [],
      },
    ],
  };
}
export function editTargetPosition(
  project: Sequence,
  id: string,
  time: number,
  position: Vec3,
): Sequence {
  return {
    ...project,
    targets: project.targets?.map((target) => {
      if (target.id !== id) return target;
      let s = targetSequence(project, target);
      for (let i = 0; i < 3; i++)
        s = addChannelKey(s, targetChannels[i], time, position[i]);
      return { ...target, tracks: s.tracks! };
    }),
  };
}
export function removeTrackingTarget(project: Sequence, id: string): Sequence {
  return {
    ...project,
    targets: project.targets?.filter((t) => t.id !== id),
    tracking: project.tracking?.targetId === id ? undefined : project.tracking,
    cameras: project.cameras?.map((c) => ({
      ...c,
      tracking: c.tracking?.targetId === id ? undefined : c.tracking,
    })),
  };
}
export function applyTracking(
  pose: CameraPose,
  sequence: Sequence,
  cameraTime: number,
  targetTime: number,
): CameraPose {
  const constraint = sequence.tracking;
  const target =
    constraint?.enabled &&
    sequence.targets?.find((t) => t.id === constraint.targetId);
  if (!constraint || !target) return pose;
  const point = sampleTarget(target, targetTime).map(
    (v, i) => v + constraint.offset[i],
  ) as Vec3;
  const track = sequence.tracks?.find((t) => t.channel === "trackingWeight");
  const weight = MathUtils.clamp(
    track
      ? sampleTrack(track, cameraTime, track.defaultValue ?? constraint.weight)
      : constraint.weight,
    0,
    1,
  );
  if (constraint.focus) pose.focusTarget = point;
  if (weight <= 0) return pose;
  const origin = new Vector3(...pose.position),
    destination = new Vector3(...point).sub(origin);
  if (destination.lengthSq() < 1e-16) return pose;
  if (weight === 1) {
    pose.target = point;
    return pose;
  }
  const base = new Vector3(...pose.target).sub(origin),
    distance = base.length();
  if (distance < 1e-8) {
    pose.target = point;
    return pose;
  }
  const length = MathUtils.lerp(distance, destination.length(), weight);
  const rotation = new Quaternion().setFromUnitVectors(
    base.clone().normalize(),
    destination.normalize(),
  );
  base
    .normalize()
    .applyQuaternion(new Quaternion().slerp(rotation, weight))
    .multiplyScalar(length)
    .add(origin);
  pose.target = base.toArray() as Vec3;
  return pose;
}
const finite = (v: unknown, min: number, max: number) =>
  typeof v === "number" && Number.isFinite(v) && v >= min && v <= max;
const vector = (v: unknown) =>
  Array.isArray(v) &&
  v.length === 3 &&
  v.every((n) => finite(n, -10000, 10000));
export function parseTrackingTargets(
  raw: unknown,
  duration: number,
): TrackingTarget[] | undefined {
  if (raw === undefined) return undefined;
  if (!Array.isArray(raw) || raw.length > 64)
    throw new Error("Invalid tracking targets.");
  const ids = new Set<string>();
  return raw.map((t) => {
    if (
      !t ||
      typeof t.id !== "string" ||
      !t.id.length ||
      ids.has(t.id) ||
      typeof t.name !== "string" ||
      t.name.length > 200 ||
      !vector(t.position)
    )
      throw new Error("Invalid tracking target.");
    ids.add(t.id);
    const tracks = parseAnimationTracks(t.tracks, duration);
    if (
      tracks.some(
        (track) =>
          !targetChannels.includes(
            track.channel as (typeof targetChannels)[number],
          ),
      )
    )
      throw new Error("Invalid target track.");
    return { id: t.id, name: t.name, position: t.position, tracks };
  });
}
export function parseCameraTracking(
  raw: unknown,
  targets: TrackingTarget[] | undefined,
): CameraTracking | undefined {
  if (raw === undefined) return undefined;
  const c = raw as CameraTracking;
  if (
    !c ||
    typeof c.enabled !== "boolean" ||
    !targets?.some((t) => t.id === c.targetId) ||
    !vector(c.offset) ||
    !finite(c.weight, 0, 1) ||
    typeof c.focus !== "boolean"
  )
    throw new Error("Invalid target tracking.");
  return {
    enabled: c.enabled,
    targetId: c.targetId,
    offset: c.offset,
    weight: c.weight,
    focus: c.focus,
  };
}

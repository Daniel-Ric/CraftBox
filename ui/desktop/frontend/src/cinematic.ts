import { MathUtils } from "three";
import {
  channelLimits,
  parseAnimationTracks,
  sampleTrack,
  type AnimationTrack,
} from "./animation-tracks";
import {
  resolveShot,
  type CinematicCamera,
  type CameraShot,
} from "./cinematic-shots";
import {
  applyTracking,
  parseTrackingTargets,
  parseCameraTracking,
  type CameraTracking,
  type TrackingTarget,
} from "./cinematic-tracking";
import { parseMotionBlur, type MotionBlur } from "./cinematic-motion";
import { parseAudioTimeline, type AudioTimeline } from "./cinematic-audio";
import { parseMarkers, type TimelineMarker } from "./cinematic-markers";
export type Vec3 = [number, number, number];
export type CameraPose = {
  position: Vec3;
  target: Vec3;
  fov: number;
  roll: number;
  exposure: number;
  focusDistance?: number;
  aperture?: number;
  depthOfField?: boolean;
  focusTarget?: Vec3;
  trackingWeight?: number;
};
export type CameraKey = CameraPose & {
  id: string;
  time: number;
  ease: "linear" | "smooth" | "hold";
};
export type Sequence = {
  version: 1 | 2;
  name: string;
  duration: number;
  fps: number;
  keys: CameraKey[];
  tracks?: AnimationTrack[];
  lens?: { enabled: boolean; distance: number; aperture: number };
  cameras?: CinematicCamera[];
  shots?: CameraShot[];
  targets?: TrackingTarget[];
  tracking?: CameraTracking;
  motionBlur?: MotionBlur;
  audio?: AudioTimeline;
  markers?: TimelineMarker[];
};
export function createSequence(): Sequence {
  return {
    version: 1,
    name: "New camera sequence",
    duration: 12,
    fps: 24,
    keys: [
      {
        id: "start",
        time: 0,
        position: [1.8, 0.8, 1.8],
        target: [0, 0, 0],
        fov: 45,
        roll: 0,
        exposure: 1,
        ease: "smooth",
      },
      {
        id: "middle",
        time: 6,
        position: [0, 0.5, 2],
        target: [0, 0, 0],
        fov: 40,
        roll: 0,
        exposure: 1,
        ease: "smooth",
      },
      {
        id: "end",
        time: 12,
        position: [-1.8, 0.8, 1.8],
        target: [0, 0, 0],
        fov: 45,
        roll: 0,
        exposure: 1,
        ease: "smooth",
      },
    ],
  };
}
export function upsertKeyframe(keys: CameraKey[], key: CameraKey, fps: number) {
  const time = Math.round(key.time * fps) / fps;
  return [
    ...keys.filter(
      (k) =>
        k.id !== key.id && Math.round(k.time * fps) !== Math.round(time * fps),
    ),
    { ...key, time },
  ].sort((a, b) => a.time - b.time);
}
export function moveKeyframe(
  keys: CameraKey[],
  id: string,
  time: number,
  fps: number,
) {
  const frame = Math.round(time * fps);
  if (keys.some((k) => k.id !== id && Math.round(k.time * fps) === frame))
    return keys;
  return keys
    .map((k) => (k.id === id ? { ...k, time: frame / fps } : k))
    .sort((a, b) => a.time - b.time);
}
export function sampleSequence(sequence: Sequence, time: number): CameraPose {
  const shot = resolveShot(sequence, time);
  if (shot)
    return sampleCameraSequence(
      {
        ...sequence,
        ...shot.camera,
        tracking: shot.camera.tracking,
        cameras: undefined,
        shots: undefined,
      },
      shot.sourceTime,
      time,
    );
  return sampleCameraSequence(sequence, time);
}
export function sampleCameraSequence(
  sequence: Sequence,
  time: number,
  targetTime = time,
): CameraPose {
  const result = sampleLegacySequence(
    sequence,
    sequence.version === 2 ? 0 : time,
  );
  if (sequence.version !== 2) return result;
  if (sequence.lens) {
    result.focusDistance = sequence.lens.distance;
    result.aperture = sequence.lens.aperture;
    result.depthOfField = sequence.lens.enabled;
  }
  if (sequence.tracking) result.trackingWeight = sequence.tracking.weight;
  for (const track of sequence.tracks ?? []) {
    const [field, axis] = track.channel.split(".");
    const limits = channelLimits[track.channel];
    if (field === "position" || field === "target") {
      const index = "xyz".indexOf(axis);
      result[field][index] = sampleTrack(
        track,
        time,
        track.defaultValue ?? result[field][index],
      );
    } else if (
      field === "fov" ||
      field === "roll" ||
      field === "exposure" ||
      field === "focusDistance" ||
      field === "aperture" ||
      field === "trackingWeight"
    ) {
      result[field] = MathUtils.clamp(
        sampleTrack(track, time, track.defaultValue ?? result[field] ?? 1),
        ...limits,
      );
    }
  }
  return applyTracking(result, sequence, time, targetTime);
}
export function sampleLegacySequence(
  sequence: Sequence,
  time: number,
): CameraPose {
  const keys = sequence.keys;
  if (keys.length === 1 || time <= keys[0].time) return pose(keys[0]);
  if (time >= keys[keys.length - 1].time) return pose(keys[keys.length - 1]);
  const i = keys.findIndex(
    (_, index) => index < keys.length - 1 && time < keys[index + 1].time,
  );
  const a = keys[i],
    b = keys[i + 1];
  const duration = b.time - a.time;
  const t = MathUtils.clamp((time - a.time) / duration, 0, 1);
  if (a.ease === "hold") return pose(a);
  const angleDelta = (from: number, to: number) =>
    MathUtils.euclideanModulo(to - from + 180, 360) - 180;


  const interpolate = (from: number, to: number, va: number, vb: number) => {
    const t2 = t * t,
      t3 = t2 * t;
    return (
      (2 * t3 - 3 * t2 + 1) * from +
      (t3 - 2 * t2 + t) * duration * va +
      (-2 * t3 + 3 * t2) * to +
      (t3 - t2) * duration * vb
    );
  };
  const vectorVelocity = (
    index: number,
    field: "position" | "target",
  ): Vec3 => {
    if (index === 0 || index === keys.length - 1) return [0, 0, 0];
    const previous = keys[index - 1],
      current = keys[index],
      next = keys[index + 1];
    if (previous.ease === "hold" || current.ease === "hold") return [0, 0, 0];
    const before = current.time - previous.time,
      after = next.time - current.time;
    const incoming = current[field].map(
      (v, axis) => (v - previous[field][axis]) / before,
    ) as Vec3;
    const outgoing = next[field].map(
      (v, axis) => (v - current[field][axis]) / after,
    ) as Vec3;
    if (previous.ease === "linear") return incoming;
    if (current.ease === "linear") return outgoing;
    const velocity = incoming.map(
      (v, axis) => (v * after + outgoing[axis] * before) / (before + after),
    ) as Vec3;




    const speed = Math.hypot(...velocity),
      incomingSpeed = Math.hypot(...incoming),
      outgoingSpeed = Math.hypot(...outgoing);
    if (speed < 1e-10 || incomingSpeed === 0 || outgoingSpeed === 0)
      return [0, 0, 0];
    const w1 = 2 * after + before,
      w2 = after + 2 * before;
    const directionCosine = MathUtils.clamp(
      incoming.reduce((dot, v, axis) => dot + v * outgoing[axis], 0) /
        (incomingSpeed * outgoingSpeed),
      -1,
      1,
    );


    const turnFactor = Math.sqrt(Math.min(1, 1 + directionCosine));
    const cruisingSpeed = Math.min(
      (w1 + w2) / (w1 / incomingSpeed + w2 / outgoingSpeed),
      3 * Math.min(incomingSpeed, outgoingSpeed),
    );
    return velocity.map(
      (v) => (v * cruisingSpeed * turnFactor) / speed,
    ) as Vec3;
  };
  const curve = (field: "position" | "target"): Vec3 => {
    if (a.ease !== "smooth")
      return a[field].map((v, axis) =>
        MathUtils.lerp(v, b[field][axis], t),
      ) as Vec3;
    const va = vectorVelocity(i, field),
      vb = vectorVelocity(i + 1, field);
    return a[field].map((v, axis) =>
      interpolate(v, b[field][axis], va[axis], vb[axis]),
    ) as Vec3;
  };
  const scalar = (field: "fov" | "roll" | "exposure") => {
    const delta = (from: number, to: number) =>
      field === "roll" ? angleDelta(from, to) : to - from;
    const end = a[field] + delta(a[field], b[field]);
    if (a.ease !== "smooth") return MathUtils.lerp(a[field], end, t);
    const velocity = (index: number) => {
      if (index === 0 || index === keys.length - 1) return 0;
      const previous = keys[index - 1],
        current = keys[index],
        next = keys[index + 1];
      if (previous.ease === "hold" || current.ease === "hold") return 0;
      const before = current.time - previous.time,
        after = next.time - current.time;
      const incoming = delta(previous[field], current[field]) / before,
        outgoing = delta(current[field], next[field]) / after;
      if (incoming * outgoing <= 0) return 0;
      if (
        previous.ease === "linear" &&
        Math.abs(incoming) <= 3 * Math.abs(outgoing)
      )
        return incoming;
      if (
        current.ease === "linear" &&
        Math.abs(outgoing) <= 3 * Math.abs(incoming)
      )
        return outgoing;

      const w1 = 2 * after + before,
        w2 = after + 2 * before;
      return (w1 + w2) / (w1 / incoming + w2 / outgoing);
    };
    return interpolate(a[field], end, velocity(i), velocity(i + 1));
  };
  return {
    position: curve("position"),
    target: curve("target"),
    fov: scalar("fov"),
    roll: scalar("roll"),
    exposure: scalar("exposure"),
  };
}
function pose(key: CameraKey): CameraPose {
  return {
    position: [...key.position],
    target: [...key.target],
    fov: key.fov,
    roll: key.roll,
    exposure: key.exposure,
  };
}
export function timecode(seconds: number, fps: number) {
  const frame = Math.max(0, Math.round(seconds * fps));
  return [
    Math.floor(frame / fps / 3600),
    Math.floor(frame / fps / 60) % 60,
    Math.floor(frame / fps) % 60,
    frame % fps,
  ]
    .map((n) => String(n).padStart(2, "0"))
    .join(":");
}
export function parseSequence(text: string): Sequence {
  const s = JSON.parse(text) as Sequence;
  const finite = (n: unknown, min: number, max: number) =>
    typeof n === "number" && Number.isFinite(n) && n >= min && n <= max;
  const vector = (v: unknown) =>
    Array.isArray(v) &&
    v.length === 3 &&
    v.every((n) => finite(n, -10000, 10000));
  if (
    ![1, 2].includes(s.version) ||
    typeof s.name !== "string" ||
    s.name.length > 200 ||
    !finite(s.duration, 0.1, 3600) ||
    ![24, 25, 30, 50, 60].includes(s.fps) ||
    !Array.isArray(s.keys) ||
    !s.keys.length ||
    s.keys.length > 2000
  )
    throw new Error("Invalid cinematic project");
  const ids = new Set<string>(),
    frames = new Set<number>();
  for (const k of s.keys) {
    const frame = Math.round(k.time * s.fps);
    if (
      typeof k.id !== "string" ||
      ids.has(k.id) ||
      (s.version === 1 && frames.has(frame)) ||
      !finite(k.time, 0, s.duration) ||
      !vector(k.position) ||
      !vector(k.target) ||
      !finite(k.fov, 10, 120) ||
      !finite(
        k.roll,
        s.version === 2 ? -1000000 : -180,
        s.version === 2 ? 1000000 : 180,
      ) ||
      !finite(k.exposure, 0.1, 4) ||
      !["linear", "smooth", "hold"].includes(k.ease)
    )
      throw new Error("Invalid camera keyframes");
    ids.add(k.id);
    frames.add(frame);
  }
  const result = { ...s, keys: [...s.keys].sort((a, b) => a.time - b.time) };
  if (s.version === 2)
    result.tracks = parseAnimationTracks(s.tracks, s.duration);
  if (
    s.lens &&
    (typeof s.lens.enabled !== "boolean" ||
      !finite(s.lens.distance, 0.001, 100000) ||
      !finite(s.lens.aperture, 0.7, 64))
  )
    throw new Error("Invalid camera lens settings.");
  const targets = parseTrackingTargets(s.targets, s.duration);
  if (targets !== undefined) result.targets = targets;
  if (s.tracking !== undefined)
    result.tracking = parseCameraTracking(s.tracking, targets);
  if (s.motionBlur !== undefined)
    result.motionBlur = parseMotionBlur(s.motionBlur);
  if (s.audio !== undefined) result.audio = parseAudioTimeline(s.audio);
  if (s.markers !== undefined) result.markers = parseMarkers(s.markers, s.duration);
  if (s.cameras !== undefined || s.shots !== undefined) {
    if (
      s.version !== 2 ||
      !Array.isArray(s.cameras) ||
      !s.cameras.length ||
      s.cameras.length > 16 ||
      !Array.isArray(s.shots) ||
      !s.shots.length ||
      s.shots.length > 1000
    )
      throw new Error("Invalid cameras or shots.");
    const cameraIds = new Set<string>();
    result.cameras = s.cameras.map((camera) => {
      if (
        !camera ||
        typeof camera.id !== "string" ||
        !camera.id.length ||
        cameraIds.has(camera.id) ||
        "cameras" in camera ||
        "shots" in camera
      )
        throw new Error("Invalid camera.");
      cameraIds.add(camera.id);
      const parsed = parseSequence(
        JSON.stringify({
          version: 2,
          name: camera.name,
          duration: camera.duration,
          fps: s.fps,
          keys: camera.keys,
          tracks: camera.tracks,
          lens: camera.lens,
        }),
      );
      return {
        id: camera.id,
        name: parsed.name,
        duration: parsed.duration,
        keys: parsed.keys,
        tracks: parsed.tracks!,
        ...(parsed.lens ? { lens: parsed.lens } : {}),
        ...(camera.tracking
          ? { tracking: parseCameraTracking(camera.tracking, targets) }
          : {}),
      };
    });
    const shotIds = new Set<string>();
    result.shots = [...s.shots].sort((a, b) => a.start - b.start);
    let end = 0;
    for (const shot of result.shots) {
      const camera = result.cameras.find((c) => c.id === shot.cameraId);
      if (
        !shot ||
        typeof shot.id !== "string" ||
        !shot.id.length ||
        shotIds.has(shot.id) ||
        typeof shot.name !== "string" ||
        shot.name.length > 200 ||
        !camera ||
        !finite(shot.start, end, s.duration) ||
        !finite(shot.end, shot.start + 1e-8, s.duration) ||
        !finite(shot.sourceIn, 0, camera.duration) ||
        !finite(shot.rate, 0.01, 100)
      )
        throw new Error("Invalid or overlapping shot.");
      shotIds.add(shot.id);
      end = shot.end;
    }
  }
  return result;
}

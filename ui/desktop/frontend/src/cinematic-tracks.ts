import {
  sampleLegacySequence,
  sampleSequence,
  upsertKeyframe,
  moveKeyframe,
  type CameraPose,
  type CameraKey,
  type Sequence,
} from "./cinematic";
import {
  channelLabels,
  upsertTrackKey,
  moveTrackKey,
  type AnimationChannel,
  type AnimationTrack,
  type ScalarKey,
} from "./animation-tracks";
import { splitCurveAt } from "./animation-curves";

export function minimumSequenceDuration(sequence: Sequence): number {
  let duration = 0.1;
  for (const key of sequence.keys) duration = Math.max(duration, key.time);
  for (const track of sequence.tracks ?? [])
    for (const key of track.keys) duration = Math.max(duration, key.time);
  return duration;
}

export function channelValue(
  pose: CameraPose,
  channel: AnimationChannel,
): number {
  const [field, axis] = channel.split(".");
  if (field === "position" || field === "target")
    return pose[field]["xyz".indexOf(axis)];
  if (channel === "focusDistance") return pose.focusDistance ?? 100;
  if (channel === "aperture") return pose.aperture ?? 2.8;
  if (channel === "trackingWeight") return pose.trackingWeight ?? 1;
  return pose[channel as "fov" | "roll" | "exposure"];
}
const unwrap = (value: number, reference: number) =>
  reference + ((((value - reference + 180) % 360) + 360) % 360) - 180;


export function independentSequence(
  sequence: Sequence,
  lens?: Sequence["lens"],
): Sequence {
  if (sequence.version === 2 && sequence.tracks) return sequence;
  const channels = Object.keys(channelLabels).filter(
    (c) => c !== "focusDistance" && c !== "aperture" && c !== "trackingWeight",
  ) as AnimationChannel[];
  const tracks: AnimationTrack[] = channels.map((channel) => {
    const keys: ScalarKey[] = sequence.keys.map((k) => ({
      id: `${k.id}:${channel}`,
      time: k.time,
      value: channelValue(k, channel),
      interpolation: k.ease === "smooth" ? "bezier" : k.ease,
      incoming: [0, 0],
      outgoing: [0, 0],
    }));
    if (channel === "roll")
      for (let i = 1; i < keys.length; i++)
        keys[i].value = unwrap(keys[i].value, keys[i - 1].value);
    for (let i = 0; i < keys.length - 1; i++) {
      const a = keys[i],
        b = keys[i + 1],
        span = b.time - a.time;
      if (a.interpolation !== "bezier") continue;
      let first = channelValue(
        sampleLegacySequence(sequence, a.time + span / 3),
        channel,
      );
      let second = channelValue(
        sampleLegacySequence(sequence, a.time + (span * 2) / 3),
        channel,
      );
      if (channel === "roll") {
        first = unwrap(first, a.value);
        second = unwrap(second, a.value);
      }
      const p = 27 * first - 8 * a.value - b.value,
        q = 27 * second - a.value - 8 * b.value;
      const c1 = (2 * p - q) / 18,
        c2 = (2 * q - p) / 18;
      a.outgoing = [span / 3, c1 - a.value];
      b.incoming = [-span / 3, c2 - b.value];
    }
    return { channel, keys };
  });
  return { ...sequence, version: 2, tracks, ...(lens ? { lens } : {}) };
}

export function addChannelKey(
  sequence: Sequence,
  channel: AnimationChannel,
  time: number,
  value?: number,
): Sequence {
  const s = independentSequence(sequence),
    pose = sampleSequence(s, time);
  const track = s.tracks!.find((t) => t.channel === channel) ?? {
    channel,
    keys: [],
  };
  const k: ScalarKey = {
    id: crypto.randomUUID(),
    time: Math.max(
      0,
      Math.min(
        Math.floor(s.duration * s.fps) / s.fps,
        Math.round(time * s.fps) / s.fps,
      ),
    ),
    value: value ?? channelValue(pose, channel),
    interpolation: "bezier",
    incoming: [-1 / s.fps, 0],
    outgoing: [1 / s.fps, 0],
  };
  const existing = track.keys.find(
    (key) => Math.round(key.time * s.fps) === Math.round(k.time * s.fps),
  );
  const next = existing
    ? {
        ...track,
        keys: track.keys.map((key) =>
          key.id === existing.id ? { ...key, value: value ?? key.value } : key,
        ),
      }
    : value === undefined && track.keys.length
      ? splitCurveAt(track, k.time, k.id)
      : upsertTrackKey(track, k, s.fps);
  return {
    ...s,
    tracks: [...s.tracks!.filter((t) => t.channel !== channel), next],
  };
}

export function duplicateChannelKey(
  sequence: Sequence,
  channel: AnimationChannel,
  id: string,
): Sequence {
  const track = sequence.tracks?.find((t) => t.channel === channel);
  const source = track?.keys.find((key) => key.id === id);
  if (!track || !source) return sequence;
  const occupied = new Set(
    track.keys.map((key) => Math.round(key.time * sequence.fps)),
  );
  const frame = Math.round(source.time * sequence.fps),
    last = Math.floor(sequence.duration * sequence.fps);
  for (let distance = 1; distance <= last; distance++) {
    for (const candidate of [frame + distance, frame - distance]) {
      if (candidate < 0 || candidate > last || occupied.has(candidate))
        continue;
      const next = {
        ...track,
        keys: [
          ...track.keys,
          {
            ...source,
            id: crypto.randomUUID(),
            time: candidate / sequence.fps,
          },
        ].sort((a, b) => a.time - b.time),
      };
      return {
        ...sequence,
        tracks: sequence.tracks!.map((t) => (t === track ? next : t)),
      };
    }
  }
  return sequence;
}

export function editPoseKey(
  sequence: Sequence,
  id: string,
  change: Partial<CameraKey>,
): Sequence {
  const s = independentSequence(sequence),
    original = s.keys.find((k) => k.id === id);
  if (!original) return s;
  const updated = { ...original, ...change };
  const keys =
    change.time === undefined
      ? s.keys.map((k) => (k.id === id ? updated : k))
      : moveKeyframe(s.keys, id, change.time, s.fps);
  if (change.time !== undefined && keys === s.keys) return s;
  const time = keys.find((k) => k.id === id)!.time;
  if (
    change.time !== undefined &&
    s.tracks!.some((track) => {
      const moving = track.keys.find(
        (key) => Math.abs(key.time - original.time) < 1e-8,
      );
      return (
        moving &&
        track.keys.some(
          (key) =>
            key.id !== moving.id &&
            Math.round(key.time * s.fps) === Math.round(time * s.fps),
        )
      );
    })
  )
    return s;
  const tracks = s.tracks!.map((track) => {
    const [field] = track.channel.split(".");
    if (change.time !== undefined) {
      const key = track.keys.find(
        (k) => Math.abs(k.time - original.time) < 1e-8,
      );
      return key ? moveTrackKey(track, key.id, time, s.fps, s.duration) : track;
    }
    if (!(field in change) && change.ease === undefined) return track;
    const existing = track.keys.find(
      (k) => Math.abs(k.time - original.time) < 1e-8,
    );
    const key: ScalarKey = existing ?? {
      id: crypto.randomUUID(),
      time: original.time,
      value: channelValue(updated, track.channel),
      interpolation: "bezier",
      incoming: [-1 / s.fps, 0],
      outgoing: [1 / s.fps, 0],
    };
    const value: ScalarKey = {
      ...key,
      value: field in change ? channelValue(updated, track.channel) : key.value,
      ...(change.ease === undefined
        ? {}
        : {
            interpolation:
              change.ease === "smooth" ? ("bezier" as const) : change.ease,
          }),
    };
    return {
      ...track,
      keys: existing
        ? track.keys.map((k) => (k.id === existing.id ? value : k))
        : [...track.keys, value].sort((a, b) => a.time - b.time),
    };
  });
  return { ...s, keys, tracks };
}

export function changeSequenceFPS(sequence: Sequence, fps: number): Sequence {
  if (![24, 25, 30, 50, 60].includes(fps))
    throw new Error("Invalid frame rate.");

  return { ...independentSequence(sequence), fps };
}
export function insertPoseKey(sequence: Sequence, key: CameraKey): Sequence {
  let s = independentSequence(sequence);
  const time = Math.min(
    Math.floor(s.duration * s.fps) / s.fps,
    Math.max(0, Math.round(key.time * s.fps) / s.fps),
  );
  const inserted = { ...key, time };
  s = { ...s, keys: upsertKeyframe(s.keys, inserted, s.fps) };
  for (const channel of Object.keys(channelLabels) as AnimationChannel[]) {
    if (
      channel === "trackingWeight" ||
      (channel === "focusDistance" && key.focusDistance === undefined) ||
      (channel === "aperture" && key.aperture === undefined)
    )
      continue;
    s = addChannelKey(s, channel, time, channelValue(key, channel));
  }
  return s;
}
export function removePoseKey(sequence: Sequence, id: string): Sequence {
  const key = sequence.keys.find((k) => k.id === id);
  if (!key || sequence.keys.length < 2) return sequence;
  return {
    ...sequence,
    keys: sequence.keys.filter((k) => k.id !== id),
    tracks: sequence.tracks?.map((t) => {
      const removed = t.keys.find((k) => Math.abs(k.time - key.time) < 1e-8);
      return removed
        ? {
            ...t,
            defaultValue: removed.value,
            keys: t.keys.filter((k) => k.id !== removed.id),
          }
        : t;
    }),
  };
}

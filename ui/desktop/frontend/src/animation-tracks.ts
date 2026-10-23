
export type AnimationChannel =
  | "position.x"
  | "position.y"
  | "position.z"
  | "target.x"
  | "target.y"
  | "target.z"
  | "fov"
  | "roll"
  | "exposure"
  | "focusDistance"
  | "aperture"
  | "trackingWeight";
export type ScalarKey = {
  id: string;
  time: number;
  value: number;
  interpolation: "hold" | "linear" | "bezier";
  incoming: [number, number];
  outgoing: [number, number];
};
export type AnimationTrack<C extends string = AnimationChannel> = {
  channel: C;
  keys: ScalarKey[];
  muted?: boolean;
  defaultValue?: number;
};
export const channelLimits: Record<AnimationChannel, [number, number]> = {
  "position.x": [-10000, 10000],
  "position.y": [-10000, 10000],
  "position.z": [-10000, 10000],
  "target.x": [-10000, 10000],
  "target.y": [-10000, 10000],
  "target.z": [-10000, 10000],
  fov: [10, 120],
  roll: [-1000000, 1000000],
  exposure: [0.1, 4],
  focusDistance: [0.001, 100000],
  aperture: [0.7, 64],
  trackingWeight: [0, 1],
};
export const channelLabels: Record<AnimationChannel, string> = {
  "position.x": "Position X",
  "position.y": "Position Y",
  "position.z": "Position Z",
  "target.x": "Look-at target X",
  "target.y": "Look-at target Y",
  "target.z": "Look-at target Z",
  fov: "Field of view",
  roll: "Camera roll",
  exposure: "Exposure",
  focusDistance: "Focus distance",
  aperture: "Aperture",
  trackingWeight: "Target tracking",
};
const cubic = (a: number, b: number, c: number, d: number, t: number) => {
  const s = 1 - t;
  return s * s * s * a + 3 * s * s * t * b + 3 * s * t * t * c + t * t * t * d;
};
export function segmentControls(a: ScalarKey, b: ScalarKey) {
  const span = b.time - a.time;
  let outTime = Math.max(0, Math.min(span, a.outgoing[0]));
  let inTime = span + Math.max(-span, Math.min(0, b.incoming[0]));
  if (outTime > inTime) outTime = inTime = (outTime + inTime) / 2;
  return {
    x1: a.time + outTime,
    y1: a.value + a.outgoing[1],
    x2: a.time + inTime,
    y2: b.value + b.incoming[1],
  };
}

export function sampleTrack(
  track: AnimationTrack<string>,
  time: number,
  fallback: number,
): number {
  const keys = track.keys;
  if (!keys.length || track.muted) return fallback;
  if (time <= keys[0].time) return keys[0].value;
  if (time >= keys[keys.length - 1].time) return keys[keys.length - 1].value;
  let lo = 0,
    hi = keys.length - 1;
  while (hi - lo > 1) {
    const middle = (lo + hi) >>> 1;
    if (keys[middle].time <= time) lo = middle;
    else hi = middle;
  }
  const a = keys[lo],
    b = keys[hi];
  if (a.interpolation === "hold") return a.value;
  const span = b.time - a.time,
    normalized = (time - a.time) / span;
  if (a.interpolation === "linear")
    return a.value + (b.value - a.value) * normalized;
  const controls = segmentControls(a, b);
  const outTime = controls.x1 - a.time,
    inTime = controls.x2 - a.time;
  let low = 0,
    high = 1;
  for (let i = 0; i < 36; i++) {
    const u = (low + high) / 2;
    if (cubic(0, outTime, inTime, span, u) < time - a.time) low = u;
    else high = u;
  }
  return cubic(
    a.value,
    a.value + a.outgoing[1],
    b.value + b.incoming[1],
    b.value,
    (low + high) / 2,
  );
}

export function parseScalarTracks<C extends string>(
  raw: unknown,
  duration: number,
  schema: Record<C, [number, number]>,
): AnimationTrack<C>[] {
  if (!Array.isArray(raw) || raw.length > Object.keys(schema).length)
    throw new Error("Invalid animation tracks.");
  const channels = new Set<string>();
  const finite = (v: unknown, min: number, max: number): v is number =>
    typeof v === "number" && Number.isFinite(v) && v >= min && v <= max;
  return raw.map((item) => {
    if (
      !item ||
      !Object.hasOwn(schema, item.channel) ||
      channels.has(item.channel) ||
      !Array.isArray(item.keys) ||
      item.keys.length > 2000 ||
      (item.muted !== undefined && typeof item.muted !== "boolean")
    )
      throw new Error("Invalid or duplicate animation track.");
    channels.add(item.channel);
    const channel = item.channel as C,
      limits = schema[channel],
      ids = new Set<string>(),
      times = new Set<number>();
    if (
      item.defaultValue !== undefined &&
      !finite(item.defaultValue, limits[0], limits[1])
    )
      throw new Error("Invalid track base value.");
    const keys: ScalarKey[] = item.keys.map((k: ScalarKey) => {
      const handle = (v: unknown): v is [number, number] =>
        Array.isArray(v) &&
        v.length === 2 &&
        finite(v[0], -3600, 3600) &&
        finite(v[1], -1000000, 1000000);
      if (
        !k ||
        typeof k.id !== "string" ||
        !k.id ||
        k.id.length > 100 ||
        ids.has(k.id) ||
        times.has(k.time) ||
        !finite(k.time, 0, duration) ||
        !finite(k.value, limits[0], limits[1]) ||
        !["hold", "linear", "bezier"].includes(k.interpolation) ||
        !handle(k.incoming) ||
        !handle(k.outgoing) ||
        k.incoming[0] > 0 ||
        k.outgoing[0] < 0
      )
        throw new Error("Invalid track keyframe.");
      ids.add(k.id);
      times.add(k.time);
      return {
        id: k.id,
        time: k.time,
        value: k.value,
        interpolation: k.interpolation,
        incoming: [...k.incoming],
        outgoing: [...k.outgoing],
      };
    });
    return {
      channel,
      keys: keys.sort((a, b) => a.time - b.time),
      ...(item.muted === undefined ? {} : { muted: item.muted }),
      ...(item.defaultValue === undefined
        ? {}
        : { defaultValue: item.defaultValue }),
    };
  });
}
export function parseAnimationTracks(
  raw: unknown,
  duration: number,
): AnimationTrack[] {
  return parseScalarTracks(raw, duration, channelLimits);
}

export function upsertTrackKey<C extends string>(
  track: AnimationTrack<C>,
  key: ScalarKey,
  fps: number,
): AnimationTrack<C> {
  if (
    !Number.isFinite(key.time) ||
    !Number.isFinite(key.value) ||
    key.time < 0 ||
    !Number.isFinite(fps) ||
    fps <= 0
  )
    throw new Error("Invalid animation keyframe.");
  const frame = Math.round(key.time * fps);
  const inserted: ScalarKey = {
    ...key,
    time: frame / fps,
    incoming: [...key.incoming],
    outgoing: [...key.outgoing],
  };
  return {
    ...track,
    keys: [
      ...track.keys.filter(
        (k) => k.id !== key.id && Math.round(k.time * fps) !== frame,
      ),
      inserted,
    ].sort((a, b) => a.time - b.time),
  };
}

export function moveTrackKey<C extends string>(
  track: AnimationTrack<C>,
  id: string,
  time: number,
  fps: number,
  duration: number,
): AnimationTrack<C> {
  if (
    ![time, fps, duration].every(Number.isFinite) ||
    fps <= 0 ||
    duration <= 0
  )
    throw new Error("Invalid keyframe time.");
  const frame = Math.min(
    Math.floor(duration * fps),
    Math.round(Math.max(0, time) * fps),
  );
  if (track.keys.some((k) => k.id !== id && Math.round(k.time * fps) === frame))
    return track;
  return {
    ...track,
    keys: track.keys
      .map((k) => (k.id === id ? { ...k, time: frame / fps } : k))
      .sort((a, b) => a.time - b.time),
  };
}

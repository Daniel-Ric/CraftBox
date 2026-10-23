import {
  channelLimits,
  segmentControls,
  sampleTrack,
  type AnimationChannel,
  type AnimationTrack,
  type ScalarKey,
} from "./animation-tracks";
export type TangentMode = "free" | "aligned" | "mirrored";
function boundedHandle(handle: [number, number]): [number, number] {
  const scale = Math.min(1, 1000000 / Math.max(1, Math.abs(handle[1])));
  return [handle[0] * scale, handle[1] * scale];
}
export function splitCurveAt<C extends string>(
  track: AnimationTrack<C>,
  time: number,
  id: string,
  bounds: [number, number] = channelLimits[track.channel as AnimationChannel],
): AnimationTrack<C> {
  if (track.keys.some((k) => Math.abs(k.time - time) < 1e-9)) return track;
  const index = track.keys.findIndex(
    (k, i) =>
      i < track.keys.length - 1 &&
      k.time < time &&
      track.keys[i + 1].time > time,
  );
  const key: ScalarKey = {
    id,
    time,
    value: sampleTrack(
      { ...track, muted: false },
      time,
      track.defaultValue ?? 0,
    ),
    interpolation: "bezier",
    incoming: [0, 0],
    outgoing: [0, 0],
  };
  if (index < 0) {
    const keys = track.keys.map((k) => ({ ...k }));
    if (keys.length && time > keys.at(-1)!.time)
      keys[keys.length - 1].outgoing = [0, 0];
    if (keys.length && time < keys[0].time) keys[0].incoming = [0, 0];
    return { ...track, keys: [...keys, key].sort((a, b) => a.time - b.time) };
  }
  const a = track.keys[index],
    b = track.keys[index + 1];
  key.interpolation = a.interpolation;
  if (a.interpolation !== "bezier")
    return {
      ...track,
      keys: [...track.keys, key].sort((a, b) => a.time - b.time),
    };
  const controls = segmentControls(a, b),
    point = (x: number, y: number): [number, number] => [x, y];
  const p0 = point(a.time, a.value),
    p1 = point(controls.x1, controls.y1),
    p2 = point(controls.x2, controls.y2),
    p3 = point(b.time, b.value);
  const mix = (
    a: [number, number],
    b: [number, number],
    t: number,
  ): [number, number] => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t];
  const parts = (t: number) => {
    const q0 = mix(p0, p1, t),
      q1 = mix(p1, p2, t),
      q2 = mix(p2, p3, t),
      r0 = mix(q0, q1, t),
      r1 = mix(q1, q2, t);
    return { q0, q2, r0, r1, s: mix(r0, r1, t) };
  };
  let lo = 0,
    hi = 1;
  for (let i = 0; i < 36; i++) {
    const t = (lo + hi) / 2;
    if (parts(t).s[0] < time) lo = t;
    else hi = t;
  }
  const { q0, q2, r0, r1, s } = parts((lo + hi) / 2);


  const limits = bounds;
  key.value = Math.max(limits[0], Math.min(limits[1], s[1]));
  key.incoming = boundedHandle([r0[0] - time, r0[1] - key.value]);
  key.outgoing = boundedHandle([r1[0] - time, r1[1] - key.value]);
  return {
    ...track,
    keys: [
      ...track.keys.map((k) =>
        k.id === a.id
          ? { ...k, outgoing: point(q0[0] - a.time, q0[1] - a.value) }
          : k.id === b.id
            ? { ...k, incoming: point(q2[0] - b.time, q2[1] - b.value) }
            : k,
      ),
      key,
    ].sort((a, b) => a.time - b.time),
  };
}
export function setCurveTangent(
  key: ScalarKey,
  side: "incoming" | "outgoing",
  handle: [number, number],
  mode: TangentMode,
): ScalarKey {
  const opposite = side === "incoming" ? "outgoing" : "incoming";
  handle = boundedHandle(handle);
  const next = { ...key, [side]: handle };
  if (mode === "mirrored") next[opposite] = [-handle[0], -handle[1]];
  else if (mode === "aligned" && Math.abs(handle[0]) > 1e-8)
    next[opposite] = boundedHandle([
      key[opposite][0],
      (key[opposite][0] * handle[1]) / handle[0],
    ]);
  return next;
}
export function autoCurveKey<C extends string>(
  track: AnimationTrack<C>,
  id: string,
): AnimationTrack<C> {
  const index = track.keys.findIndex((k) => k.id === id);
  if (index < 0) return track;
  const key = track.keys[index],
    before = track.keys[index - 1],
    after = track.keys[index + 1];
  const incoming = before
    ? (key.value - before.value) / (key.time - before.time)
    : 0;
  const outgoing = after
    ? (after.value - key.value) / (after.time - key.time)
    : 0;
  const slope = incoming * outgoing > 0 ? 2 / (1 / incoming + 1 / outgoing) : 0;
  const left = before ? (key.time - before.time) / 3 : 0,
    right = after ? (after.time - key.time) / 3 : 0;
  return {
    ...track,
    keys: track.keys.map((k) =>
      k.id === id
        ? {
            ...k,
            interpolation: "bezier",
            incoming: [-left, -left * slope],
            outgoing: [right, right * slope],
          }
        : k,
    ),
  };
}
export function graphPath(
  track: AnimationTrack<string>,
  x: (value: number) => number,
  y: (value: number) => number,
): string {
  if (!track.keys.length) return "";
  let path = `M${x(track.keys[0].time)},${y(track.keys[0].value)}`;
  for (let i = 0; i < track.keys.length - 1; i++) {
    const a = track.keys[i],
      b = track.keys[i + 1];
    if (a.interpolation === "hold")
      path += ` L${x(b.time)},${y(a.value)} L${x(b.time)},${y(b.value)}`;
    else if (a.interpolation === "linear")
      path += ` L${x(b.time)},${y(b.value)}`;
    else {
      const c = segmentControls(a, b);
      path += ` C${x(c.x1)},${y(c.y1)} ${x(c.x2)},${y(c.y2)} ${x(b.time)},${y(b.value)}`;
    }
  }
  return path;
}
export function sampleVelocity(
  track: AnimationTrack<string>,
  time: number,
): number {
  if (
    track.muted ||
    track.keys.length < 2 ||
    time < track.keys[0].time ||
    time >= track.keys.at(-1)!.time
  )
    return 0;
  const i = track.keys.findIndex(
    (k, index) =>
      index < track.keys.length - 1 &&
      time >= k.time &&
      time < track.keys[index + 1].time,
  );
  if (i < 0) return 0;
  const a = track.keys[i],
    b = track.keys[i + 1];
  if (a.interpolation === "hold") return 0;
  if (a.interpolation === "linear")
    return (b.value - a.value) / (b.time - a.time);
  const c = segmentControls(a, b);
  const cubic = (v0: number, v1: number, v2: number, v3: number, t: number) =>
    (1 - t) ** 3 * v0 +
    3 * (1 - t) ** 2 * t * v1 +
    3 * (1 - t) * t * t * v2 +
    t * t * t * v3;
  let lo = 0,
    hi = 1;
  for (let n = 0; n < 36; n++) {
    const t = (lo + hi) / 2;
    if (cubic(a.time, c.x1, c.x2, b.time, t) < time) lo = t;
    else hi = t;
  }
  const t = (lo + hi) / 2;
  const derivative = (v0: number, v1: number, v2: number, v3: number) =>
    3 * (1 - t) ** 2 * (v1 - v0) +
    6 * (1 - t) * t * (v2 - v1) +
    3 * t * t * (v3 - v2);
  const dt = derivative(a.time, c.x1, c.x2, b.time);
  return Math.abs(dt) > 1e-9
    ? derivative(a.value, c.y1, c.y2, b.value) / dt
    : 0;
}

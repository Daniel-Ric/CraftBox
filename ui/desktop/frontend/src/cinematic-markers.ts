import type { Sequence } from "./cinematic";
export type TimelineMarker = {
  id: string;
  time: number;
  duration: number;
  name: string;
  comment: string;
  color: string;
};
export function addMarker(sequence: Sequence, time: number): Sequence {
  if ((sequence.markers?.length ?? 0) >= 2000) return sequence;
  const marker: TimelineMarker = {
    id: crypto.randomUUID(),
    time:
      Math.max(
        0,
        Math.min(
          Math.floor(sequence.duration * sequence.fps),
          Math.round(time * sequence.fps),
        ),
      ) / sequence.fps,
    duration: 0,
    name: `Marker ${(sequence.markers?.length ?? 0) + 1}`,
    comment: "",
    color: "#e7b66d",
  };
  return {
    ...sequence,
    markers: [...(sequence.markers ?? []), marker].sort(
      (a, b) => a.time - b.time,
    ),
  };
}
export function editMarker(
  sequence: Sequence,
  id: string,
  change: Partial<TimelineMarker>,
): Sequence {
  return {
    ...sequence,
    markers: sequence.markers
      ?.map((m) => {
        if (m.id !== id) return m;
        const next = { ...m, ...change },
          time =
            Math.max(
              0,
              Math.min(
                Math.floor(sequence.duration * sequence.fps),
                Math.round(next.time * sequence.fps),
              ),
            ) / sequence.fps;
        return {
          ...next,
          id: m.id,
          time,
          duration: Math.max(
            0,
            Math.min(
              sequence.duration - time,
              Math.round(next.duration * sequence.fps) / sequence.fps,
            ),
          ),
        };
      })
      .sort((a, b) => a.time - b.time),
  };
}
export function adjacentMarker(
  sequence: Sequence,
  time: number,
  direction: number,
) {
  return direction > 0
    ? sequence.markers?.find((m) => m.time > time + 1e-8)
    : sequence.markers?.findLast((m) => m.time < time - 1e-8);
}
export function parseMarkers(
  raw: unknown,
  duration: number,
): TimelineMarker[] | undefined {
  if (raw === undefined) return undefined;
  if (!Array.isArray(raw) || raw.length > 2000)
    throw new Error("Invalid markers.");
  const ids = new Set<string>();
  for (const m of raw) {
    if (
      !m ||
      typeof m.id !== "string" ||
      !m.id ||
      ids.has(m.id) ||
      typeof m.name !== "string" ||
      m.name.length > 200 ||
      typeof m.comment !== "string" ||
      m.comment.length > 2000 ||
      !/^#[a-f\d]{6}$/i.test(m.color) ||
      !Number.isFinite(m.time) ||
      m.time < 0 ||
      m.time > duration ||
      !Number.isFinite(m.duration) ||
      m.duration < 0 ||
      m.time + m.duration > duration + 1e-8
    )
      throw new Error("Invalid sequence marker.");
    ids.add(m.id);
  }
  return [...raw].sort((a, b) => a.time - b.time);
}

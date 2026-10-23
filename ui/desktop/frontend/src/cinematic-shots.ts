import { independentSequence } from "./cinematic-tracks";
import type { Sequence } from "./cinematic";

export type CinematicCamera = Pick<
  Sequence,
  "name" | "duration" | "keys" | "lens" | "tracking"
> & { id: string; tracks: NonNullable<Sequence["tracks"]> };
export type CameraShot = {
  id: string;
  name: string;
  cameraId: string;
  start: number;
  end: number;
  sourceIn: number;
  rate: number;
};

export function enableCameras(sequence: Sequence): Sequence {
  if (sequence.cameras?.length) return sequence;
  const s = independentSequence(sequence),
    id = crypto.randomUUID();
  return {
    ...s,
    keys: [{ ...s.keys[0], time: 0 }],
    tracks: [],
    cameras: [
      {
        id,
        name: "Camera 1",
        duration: s.duration,
        keys: s.keys,
        tracks: s.tracks!,
        ...(s.lens ? { lens: s.lens } : {}),
        ...(s.tracking ? { tracking: s.tracking } : {}),
      },
    ],
    shots: [
      {
        id: crypto.randomUUID(),
        name: "Shot 1",
        cameraId: id,
        start: 0,
        end: s.duration,
        sourceIn: 0,
        rate: 1,
      },
    ],
  };
}

export function cameraSequence(project: Sequence, id?: string): Sequence {
  const camera =
    project.cameras?.find((c) => c.id === id) ?? project.cameras?.[0];
  return camera
    ? {
        ...project,
        keys: camera.keys,
        tracks: camera.tracks,
        lens: camera.lens,
        tracking: camera.tracking,
        cameras: undefined,
        shots: undefined,
      }
    : project;
}
export function applyCameraEdit(
  project: Sequence,
  next: Sequence,
  id?: string,
): Sequence {
  if (!project.cameras?.length) return next;
  const selected =
    project.cameras.find((c) => c.id === id) ?? project.cameras[0];


  return {
    ...project,
    name: next.name,
    duration: next.duration,
    fps: next.fps,
    keys: [{ ...next.keys[0], time: 0 }],
    tracks: [],
    cameras: project.cameras.map((c) =>
      c.id === selected.id
        ? {
            ...c,
            duration: Math.max(c.duration, next.duration),
            keys: next.keys,
            tracks: next.tracks!,
            tracking: next.tracking,
            ...(next.lens ? { lens: next.lens } : {}),
          }
        : c,
    ),
  };
}
export function addCamera(sequence: Sequence, sourceId?: string): Sequence {
  const s = enableCameras(sequence);
  if (s.cameras!.length >= 16) throw new Error("You can add up to 16 cameras.");
  const source = s.cameras!.find((c) => c.id === sourceId) ?? s.cameras![0];
  return {
    ...s,
    cameras: [
      ...s.cameras!,
      {
        ...structuredClone(source),
        id: crypto.randomUUID(),
        name: `Camera ${s.cameras!.length + 1}`,
      },
    ],
  };
}
export function removeCamera(sequence: Sequence, id: string): Sequence {
  if (sequence.shots?.some((s) => s.cameraId === id))
    throw new Error("This camera is still used by a shot.");
  return { ...sequence, cameras: sequence.cameras?.filter((c) => c.id !== id) };
}
export function resolveShot(sequence: Sequence, time: number) {
  if (!sequence.cameras?.length || !sequence.shots?.length) return undefined;
  const shots = sequence.shots;
  let shot = shots[0];
  for (const candidate of shots) {
    if (candidate.start <= time) shot = candidate;
    else break;
  }
  const camera = sequence.cameras.find((c) => c.id === shot.cameraId);
  if (!camera) return undefined;
  const sourceTime = Math.max(
    0,
    Math.min(
      camera.duration,
      shot.sourceIn +
        (Math.max(shot.start, Math.min(shot.end, time)) - shot.start) *
          shot.rate,
    ),
  );
  return { shot, camera, sourceTime };
}
export function splitShot(
  sequence: Sequence,
  id: string,
  time: number,
): Sequence {
  const shot = sequence.shots?.find((s) => s.id === id);
  const cut = Math.round(time * sequence.fps) / sequence.fps;
  if (
    !shot ||
    cut <= shot.start ||
    cut >= shot.end ||
    sequence.shots!.length >= 1000
  )
    return sequence;
  const duration = sequence.cameras!.find(
    (c) => c.id === shot.cameraId,
  )!.duration;
  return {
    ...sequence,
    shots: sequence.shots!.flatMap((s) =>
      s.id !== id
        ? [s]
        : [
            { ...s, end: cut },
            {
              ...s,
              id: crypto.randomUUID(),
              name: `${s.name.slice(0, 189)} · Schnitt`,
              start: cut,
              sourceIn: Math.min(
                duration,
                s.sourceIn + (cut - s.start) * s.rate,
              ),
            },
          ],
    ),
  };
}

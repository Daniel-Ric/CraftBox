import { resolveShot } from "./cinematic-shots";
import type { CameraPose, Sequence } from "./cinematic";
export type MotionBlur = {
  enabled: boolean;
  angle: number;
  phase: number;
  samples: number;
  preview: boolean;
};
export type TemporalSample = { time: number; weight: number };
export type MotionFrameSample = TemporalSample & { pose: CameraPose };
export const motionDefaults: MotionBlur = {
  enabled: false,
  angle: 180,
  phase: -90,
  samples: 8,
  preview: false,
};
export function temporalSamples(
  sequence: Sequence,
  time: number,
  fps: number,
  preview = false,
): TemporalSample[] {
  const motion = sequence.motionBlur;
  if (!motion?.enabled || motion.angle <= 0 || motion.samples <= 1)
    return [{ time, weight: 1 }];
  let min = 0,
    max = sequence.duration;
  const current = resolveShot(sequence, time);
  if (current && sequence.shots) {
    const index = sequence.shots.indexOf(current.shot);
    if (index > 0) min = current.shot.start;
    const next = sequence.shots[index + 1];
    if (next) max = next.start - 1e-8;
  }
  const count = preview ? Math.min(8, motion.samples) : motion.samples;
  const start = Math.max(min, Math.min(max, time + motion.phase / 360 / fps));
  const end = Math.max(
    min,
    Math.min(max, time + (motion.phase + motion.angle) / 360 / fps),
  );
  return Array.from({ length: count }, (_, i) => ({
    time: start + ((end - start) * (i + 0.5)) / count,
    weight: 1 / count,
  }));
}
export function parseMotionBlur(raw: unknown): MotionBlur | undefined {
  if (raw === undefined) return undefined;
  const m = raw as MotionBlur;
  if (
    !m ||
    typeof m.enabled !== "boolean" ||
    typeof m.preview !== "boolean" ||
    !Number.isFinite(m.angle) ||
    m.angle < 0 ||
    m.angle > 720 ||
    !Number.isFinite(m.phase) ||
    m.phase < -720 ||
    m.phase > 720 ||
    !Number.isInteger(m.samples) ||
    m.samples < 2 ||
    m.samples > 64
  )
    throw new Error("Invalid motion blur.");
  return {
    enabled: m.enabled,
    angle: m.angle,
    phase: m.phase,
    samples: m.samples,
    preview: m.preview,
  };
}

import { GIFEncoder, quantize, applyPalette } from "gifenc";
import { sampleSequence, type CameraPose, type Sequence } from "./cinematic";
import { api } from "./bridge";
import {runExportTask} from "./export-tasks";
import { temporalSamples, type MotionFrameSample } from "./cinematic-motion";
import {
  hasAudibleAudio,
  mixAudioBlock,
  type DecodedAudio,
} from "./cinematic-audio";
export function exportDimensions(edge: number, aspect: number) {
  return {
    width: Math.max(
      2,
      Math.round((aspect >= 1 ? edge : edge * aspect) / 2) * 2,
    ),
    height: Math.max(
      2,
      Math.round((aspect >= 1 ? edge / aspect : edge) / 2) * 2,
    ),
  };
}
type Pixels = { data: Uint8ClampedArray; width: number; height: number };
function writeGIFFrame(
  gif: ReturnType<typeof GIFEncoder>,
  image: Pixels,
  fps: number,
  frameIndex: number,
) {
  const palette = quantize(image.data, 256);
  gif.writeFrame(applyPalette(image.data, palette), image.width, image.height, {
    palette,
    delay:
      (Math.round(((frameIndex + 1) * 100) / fps) -
        Math.round((frameIndex * 100) / fps)) *
      10,
    repeat: 0,
  });
}
export function encodeGIF(frames: Pixels[], fps: number) {
  const gif = GIFEncoder();
  frames.forEach((frame, i) => writeGIFFrame(gif, frame, fps, i));
  gif.finish();
  return gif.bytes();
}
export async function saveCinematic(blob: Blob, name: string, kind: string) {
  if (blob.size > 190 * 1024 * 1024)
    throw new Error(
      "Export exceeds 190 MiB. Reduce the resolution or duration.",
    );
  if (window.go?.desktop.App) {
    const data = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve((reader.result as string).split(",")[1]);
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(blob);
    });
    return api.SaveCinematicFile(name, kind, data);
  }
  const url = URL.createObjectURL(blob),
    a = document.createElement("a");
  a.href = url;
  a.download = `${name}.${kind}`;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 10000);
  return "Download started";
}
async function encodeCinematic(options: {
  kind: "gif" | "mp4" | "webm" | "mov" | "mkv";
  fps?: number;
  bitrate?: number;
  includeAudio?: boolean;
  start?: number;
  end?: number;
  sequence: Sequence;
  edge: number;
  aspect: number;
  gifFps: number;
  signal: AbortSignal;
  render: (
    pose: CameraPose,
    width: number,
    height: number,
    time: number,
    samples?: MotionFrameSample[],
  ) => Promise<HTMLCanvasElement>;
  progress: (time: number, percent: number) => void;
}) {
  const { kind, sequence, signal } = options,
    { width, height } = exportDimensions(options.edge, options.aspect);
  const rangeStart = options.start ?? 0,
    rangeEnd = options.end ?? sequence.duration;
  if (
    !Number.isFinite(rangeStart) ||
    !Number.isFinite(rangeEnd) ||
    rangeStart < 0 ||
    rangeEnd > sequence.duration ||
    rangeEnd <= rangeStart
  )
    throw new Error("Choose a valid timeline range");
  const fps = kind === "gif" ? options.gifFps : (options.fps ?? sequence.fps),
    frames = Math.ceil((rangeEnd - rangeStart) * fps);
  if (!Number.isFinite(fps) || fps < 1 || fps > 120)
    throw new Error("Frame rate must be between 1 and 120");
  if (kind === "gif" && frames * width * height > 800000000)
    throw new Error(
      "GIF is too large. Reduce its size, frame rate or duration.",
    );
  const gif = kind === "gif" ? GIFEncoder() : null;
  const scratch = document.createElement("canvas");
  scratch.width = width;
  scratch.height = height;
  const ctx =
    kind === "gif"
      ? scratch.getContext("2d", { willReadFrequently: true })
      : null;
  let webm:
    | Awaited<
        ReturnType<(typeof import("./cinematic-webm"))["createWebMEncoder"]>
      >
    | undefined;
  try {
    signal.throwIfAborted();
    const withAudio =
      kind !== "gif" &&
      options.includeAudio !== false &&
      hasAudibleAudio(sequence.audio);
    let buffers = new Map<string, DecodedAudio>();
    if (withAudio) {
      const runtime = await import("./cinematic-audio-runtime");
      buffers = runtime.pcmBuffers(
        await runtime.decodeTimeline(sequence.audio!, signal),
      );
    }
    if (kind === "webm" && !options.bitrate)
      webm = await (
        await import("./cinematic-webm")
      ).createWebMEncoder(width, height, fps, withAudio);
    if (
      kind === "mp4" ||
      kind === "mov" ||
      kind === "mkv" ||
      (kind === "webm" && !!options.bitrate)
    )
      webm = await (
        await import("./cinematic-video")
      ).createCinematicVideoEncoder(
        kind,
        width,
        height,
        fps,
        withAudio,
        options.bitrate,
      );
    for (let i = 0; i < frames; i++) {
      signal.throwIfAborted();

      const time = rangeStart + i / fps,
        canvas = await options.render(
          sampleSequence(sequence, time),
          width,
          height,
          time,
          temporalSamples(sequence, time, fps).map((sample) => ({
            ...sample,
            pose: sampleSequence(sequence, sample.time),
          })),
        );
      if (gif) {
        if (!ctx) throw new Error("Could not create the GIF frame buffer");
        ctx.drawImage(canvas, 0, 0, width, height);
        writeGIFFrame(gif, ctx.getImageData(0, 0, width, height), fps, i);
        if (gif.bytesView().byteLength > 190 * 1024 * 1024)
          throw new Error("GIF exceeds 190 MiB. Reduce its size.");
      } else if (webm) {
        const frame = new VideoFrame(canvas, {
          timestamp: Math.round((i / fps) * 1000000),
          duration: Math.round(1000000 / fps),
        });
        try {
          await webm!.add(frame);
        } finally {
          frame.close();
        }
      }
      if (webm && withAudio) {
        const start = Math.round((i * 48000) / fps),
          end = Math.min(
            Math.round((rangeEnd - rangeStart) * 48000),
            Math.round(((i + 1) * 48000) / fps),
          );
        if (end > start)
          await webm.addAudio(
            mixAudioBlock(
              sequence.audio!,
              buffers,
              Math.round((options.start ?? 0) * 48000) + start,
              end - start,
            ).channels,
            start,
          );
      }
      options.progress(time, ((i + 1) / frames) * 100);

      await new Promise<void>((resolve) => setTimeout(resolve, 0));
    }
    signal.throwIfAborted();
    if (gif) {
      gif.finish();
      return new Blob([gif.bytes()], { type: "image/gif" });
    }
    if (webm) return await webm.finish();
    throw new Error("Video encoder unavailable.");
  } finally {
    await webm?.close();
  }
}

export async function exportCinematic(options:Parameters<typeof encodeCinematic>[0]){
  return runExportTask("Render "+options.kind.toUpperCase()+" video",task=>encodeCinematic({
    ...options,signal:AbortSignal.any([options.signal,task.signal]),
    progress:(time,percent)=>{options.progress(time,percent);task.update("Rendering frames…",percent,100)},
  }),{cancelable:true});
}

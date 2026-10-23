export type AudioAsset = {
  id: string;
  name: string;
  mime: string;
  data: string;
  duration: number;
  sampleRate: number;
  channels: number;
  peaks: number[][];
};
export type AudioTrack = {
  id: string;
  name: string;
  gainDb: number;
  pan: number;
  muted: boolean;
  solo: boolean;
};
export type AudioClip = {
  id: string;
  name: string;
  assetId: string;
  trackId: string;
  start: number;
  in: number;
  out: number;
  rate: number;
  gainDb: number;
  pan: number;
  fadeIn: number;
  fadeOut: number;
  muted: boolean;
};
export type AudioTimeline = {
  assets: AudioAsset[];
  tracks: AudioTrack[];
  clips: AudioClip[];
  masterGainDb: number;
};
export type DecodedAudio = { sampleRate: number; channels: Float32Array[] };
export const audioDefaults = (): AudioTimeline => ({
  assets: [],
  tracks: [
    {
      id: crypto.randomUUID(),
      name: "A1 · Music",
      gainDb: 0,
      pan: 0,
      muted: false,
      solo: false,
    },
  ],
  clips: [],
  masterGainDb: 0,
});
export const clipDuration = (clip: AudioClip) =>
  (clip.out - clip.in) / clip.rate;
export const dbGain = (db: number) => 10 ** (db / 20);
export function audibleTrack(audio: AudioTimeline, track: AudioTrack) {
  return !track.muted && (!audio.tracks.some((t) => t.solo) || track.solo);
}
export function clipEnvelope(clip: AudioClip, age: number) {
  const duration = clipDuration(clip);
  if (age < 0 || age >= duration) return 0;
  return (
    (clip.fadeIn > 0
      ? Math.sin((Math.min(1, age / clip.fadeIn) * Math.PI) / 2)
      : 1) *
    (clip.fadeOut > 0
      ? Math.sin((Math.min(1, (duration - age) / clip.fadeOut) * Math.PI) / 2)
      : 1)
  );
}
export function hasAudibleAudio(audio?: AudioTimeline) {
  return !!audio?.clips.some(
    (c) =>
      !c.muted &&
      audio.tracks.some((t) => t.id === c.trackId && audibleTrack(audio, t)),
  );
}
export function editAudioClipTiming(
  audio: AudioTimeline,
  id: string,
  mode: "move" | "in" | "out",
  delta: number,
  fps: number,
  snap: boolean,
): AudioTimeline {
  if (!Number.isFinite(delta)) return audio;
  return {
    ...audio,
    clips: audio.clips.map((clip) => {
      if (clip.id !== id) return clip;
      const asset = audio.assets.find((a) => a.id === clip.assetId)!;
      const minimum = Math.min(1 / fps, asset.duration / clip.rate),
        duration = clipDuration(clip);
      if (snap) {
        const anchor = mode === "out" ? clip.start + duration : clip.start;
        delta = Math.round((anchor + delta) * fps) / fps - anchor;
      }
      let next = { ...clip };
      if (mode === "move")
        next.start = Math.max(0, Math.min(3600, clip.start + delta));
      else if (mode === "in") {
        delta = Math.max(
          -clip.in / clip.rate,
          Math.min(duration - minimum, delta),
        );
        next.start += delta;
        next.in += delta * clip.rate;
      } else
        next.out = Math.max(
          clip.in + minimum * clip.rate,
          Math.min(asset.duration, clip.out + delta * clip.rate),
        );
      next.fadeIn = Math.min(next.fadeIn, clipDuration(next));
      next.fadeOut = Math.min(next.fadeOut, clipDuration(next));
      return next;
    }),
  };
}

export function mixAudioBlock(
  audio: AudioTimeline,
  buffers: Map<string, DecodedAudio>,
  startFrame: number,
  frames: number,
  sampleRate = 48000,
) {
  const channels = [new Float32Array(frames), new Float32Array(frames)];
  for (const clip of audio.clips) {
    const track = audio.tracks.find((t) => t.id === clip.trackId);
    if (!track || clip.muted || !audibleTrack(audio, track)) continue;
    const first = Math.max(0, Math.ceil(clip.start * sampleRate - startFrame)),
      last = Math.min(
        frames,
        Math.ceil((clip.start + clipDuration(clip)) * sampleRate - startFrame),
      );
    if (last <= first) continue;
    const buffer = buffers.get(clip.assetId);
    if (!buffer) throw new Error(`Audio data is missing: ${clip.name}`);
    const gain = dbGain(clip.gainDb + track.gainDb + audio.masterGainDb),
      pan = Math.max(-1, Math.min(1, clip.pan + track.pan));
    const sample = (channel: number, time: number) => {
      const data = buffer.channels[channel],
        position = time * buffer.sampleRate,
        i = Math.floor(position),
        f = position - i;
      return (data[i] ?? 0) * (1 - f) + (data[i + 1] ?? 0) * f;
    };
    for (let i = first; i < last; i++) {
      const age = (startFrame + i) / sampleRate - clip.start,
        time = clip.in + age * clip.rate;
      const envelope = gain * clipEnvelope(clip, age),
        left = sample(0, time) * envelope;
      if (buffer.channels.length === 1) {
        const angle = ((pan + 1) * Math.PI) / 4;
        channels[0][i] += left * Math.cos(angle);
        channels[1][i] += left * Math.sin(angle);
      } else {
        const right = sample(1, time) * envelope;
        if (pan <= 0) {
          channels[0][i] += left + right * Math.cos(((pan + 1) * Math.PI) / 2);
          channels[1][i] += right * Math.sin(((pan + 1) * Math.PI) / 2);
        } else {
          channels[0][i] += left * Math.cos((pan * Math.PI) / 2);
          channels[1][i] += right + left * Math.sin((pan * Math.PI) / 2);
        }
      }
    }
  }
  let peak = 0;
  for (const channel of channels)
    for (let i = 0; i < frames; i++) {
      peak = Math.max(peak, Math.abs(channel[i]));
      channel[i] = Math.max(-1, Math.min(1, channel[i]));
    }
  return { channels, peak };
}
const finite = (v: unknown, min: number, max: number): v is number =>
  typeof v === "number" && Number.isFinite(v) && v >= min && v <= max;
const text = (v: unknown, max = 200): v is string =>
  typeof v === "string" && v.length <= max;
function validBase64(data: string) {
  if (!data.length || data.length % 4 || /[^A-Za-z0-9+/=]/.test(data))
    return false;
  const padding = data.indexOf("=");
  return (
    padding < 0 ||
    (padding >= data.length - 2 && /^={1,2}$/.test(data.slice(padding)))
  );
}
export function parseAudioTimeline(raw: unknown): AudioTimeline | undefined {
  if (raw === undefined) return undefined;
  const a = raw as AudioTimeline;
  if (
    !a ||
    !Array.isArray(a.assets) ||
    a.assets.length > 16 ||
    !Array.isArray(a.tracks) ||
    a.tracks.length > 16 ||
    !Array.isArray(a.clips) ||
    a.clips.length > 128 ||
    !finite(a.masterGainDb, -60, 12)
  )
    throw new Error("Invalid audio track.");
  const assetIds = new Set<string>(),
    trackIds = new Set<string>(),
    clipIds = new Set<string>();
  let bytes = 0,
    decodedBytes = 0;
  for (const asset of a.assets) {
    if (
      !asset ||
      !text(asset.id) ||
      !asset.id ||
      assetIds.has(asset.id) ||
      !text(asset.name) ||
      !text(asset.mime, 100) ||
      !text(asset.data, 32 * 1024 * 1024) ||
      !validBase64(asset.data) ||
      !finite(asset.duration, 0.001, 3600) ||
      !finite(asset.sampleRate, 8000, 192000) ||
      !Number.isInteger(asset.sampleRate) ||
      ![1, 2].includes(asset.channels) ||
      !Array.isArray(asset.peaks) ||
      asset.peaks.length !== asset.channels ||
      asset.peaks.some(
        (p) =>
          !Array.isArray(p) ||
          p.length > 8192 ||
          p.length % 2 ||
          p.some((v) => !finite(v, -1, 1)),
      )
    )
      throw new Error("Invalid audio asset.");
    assetIds.add(asset.id);
    bytes += asset.data.length;
    decodedBytes += asset.duration * asset.sampleRate * asset.channels * 4;
  }
  if (bytes > 40 * 1024 * 1024 || decodedBytes > 256 * 1024 * 1024)
    throw new Error(
      "Audio exceeds the project storage limit. Shorten the source files or remove unused assets.",
    );
  for (const track of a.tracks) {
    if (
      !track ||
      !text(track.id) ||
      !track.id ||
      trackIds.has(track.id) ||
      !text(track.name) ||
      !finite(track.gainDb, -60, 12) ||
      !finite(track.pan, -1, 1) ||
      typeof track.muted !== "boolean" ||
      typeof track.solo !== "boolean"
    )
      throw new Error("Invalid audio mix track.");
    trackIds.add(track.id);
  }
  for (const clip of a.clips) {
    const asset = a.assets.find((s) => s.id === clip?.assetId);
    if (
      !clip ||
      !text(clip.id) ||
      !clip.id ||
      clipIds.has(clip.id) ||
      !text(clip.name) ||
      !asset ||
      !trackIds.has(clip.trackId) ||
      !finite(clip.start, -3600, 3600) ||
      !finite(clip.in, 0, asset.duration) ||
      !finite(clip.out, clip.in + 0.00001, asset.duration) ||
      !finite(clip.rate, 0.25, 4) ||
      !finite(clip.gainDb, -60, 12) ||
      !finite(clip.pan, -1, 1) ||
      !finite(clip.fadeIn, 0, clipDuration(clip)) ||
      !finite(clip.fadeOut, 0, clipDuration(clip)) ||
      typeof clip.muted !== "boolean"
    )
      throw new Error("Invalid audio clip.");
    clipIds.add(clip.id);
  }
  return a;
}

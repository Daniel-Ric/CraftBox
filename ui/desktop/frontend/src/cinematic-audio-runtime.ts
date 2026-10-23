import {
  audibleTrack,
  clipDuration,
  clipEnvelope,
  dbGain,
  hasAudibleAudio,
  mixAudioBlock,
  parseAudioTimeline,
  type AudioAsset,
  type AudioTimeline,
  type DecodedAudio,
} from "./cinematic-audio";

const decoded = new Map<
  string,
  { data: string; buffer: Promise<AudioBuffer> }
>();
function context() {
  if (typeof AudioContext === "undefined")
    throw new Error("This WebView does not support audio.");
  return new AudioContext({ sampleRate: 48000 });
}
export async function decodeAudioAsset(
  asset: AudioAsset,
): Promise<AudioBuffer> {
  const existing = decoded.get(asset.id);
  if (existing?.data === asset.data) return existing.buffer;
  const buffer = (async () => {
    const ctx = context();
    try {
      const bytes = Uint8Array.from(atob(asset.data), (c) => c.charCodeAt(0));
      const result = await ctx.decodeAudioData(bytes.buffer);
      if (
        Math.abs(result.duration - asset.duration) > 0.05 ||
        result.numberOfChannels !== asset.channels
      )
        throw new Error(
          `Audio does not match its metadata: ${asset.name}`,
        );
      return result;
    } finally {
      await ctx.close();
    }
  })();
  decoded.set(asset.id, { data: asset.data, buffer });
  try {
    return await buffer;
  } catch (error) {
    decoded.delete(asset.id);
    throw error;
  }
}
export function releaseAudioCache(audio?: AudioTimeline) {
  const ids = new Set(audio?.assets.map((a) => a.id) ?? []);
  for (const id of decoded.keys()) if (!ids.has(id)) decoded.delete(id);
}
export function discardAudioAsset(id: string) {
  decoded.delete(id);
}
export async function importAudioAsset(file: File): Promise<AudioAsset> {
  if (!file.size || file.size > 16 * 1024 * 1024)
    throw new Error("Audio files must not exceed 16 MiB.");
  const bytes = new Uint8Array(await file.arrayBuffer()),
    ctx = context();
  let buffer: AudioBuffer;
  try {
    buffer = await ctx.decodeAudioData(bytes.slice().buffer);
  } catch {
    throw new Error(
      "Could not decode this audio file. Use WAV, MP3, Ogg, FLAC or another format supported by this WebView.",
    );
  } finally {
    await ctx.close();
  }
  if (buffer.numberOfChannels > 2)
    throw new Error("Import mono or stereo audio.");
  const peaks: number[][] = [],
    bins = Math.min(2048, buffer.length);
  for (let ch = 0; ch < buffer.numberOfChannels; ch++) {
    const data = buffer.getChannelData(ch),
      values: number[] = [];
    for (let bin = 0; bin < bins; bin++) {
      let low = 0,
        high = 0;
      for (
        let i = Math.floor((bin * data.length) / bins);
        i < Math.floor(((bin + 1) * data.length) / bins);
        i++
      ) {
        low = Math.min(low, data[i]);
        high = Math.max(high, data[i]);
      }
      values.push(Math.max(-1, low), Math.min(1, high));
    }
    peaks.push(values);
  }
  let binary = "";
  for (let i = 0; i < bytes.length; i += 32768)
    binary += String.fromCharCode(...bytes.subarray(i, i + 32768));
  const asset: AudioAsset = {
    id: crypto.randomUUID(),
    name: file.name.slice(0, 200),
    mime: (file.type || "audio/unknown").slice(0, 100),
    data: btoa(binary),
    duration: buffer.duration,
    sampleRate: buffer.sampleRate,
    channels: buffer.numberOfChannels,
    peaks,
  };
  parseAudioTimeline({
    assets: [asset],
    tracks: [],
    clips: [],
    masterGainDb: 0,
  });
  decoded.set(asset.id, { data: asset.data, buffer: Promise.resolve(buffer) });
  return asset;
}
export async function decodeTimeline(
  audio: AudioTimeline,
  signal?: AbortSignal,
) {
  const buffers = new Map<string, AudioBuffer>();
  const needed = new Set(
    audio.clips
      .filter(
        (c) =>
          !c.muted &&
          audio.tracks.some(
            (t) => t.id === c.trackId && audibleTrack(audio, t),
          ),
      )
      .map((c) => c.assetId),
  );
  for (const asset of audio.assets)
    if (needed.has(asset.id)) {
      signal?.throwIfAborted();
      buffers.set(asset.id, await decodeAudioAsset(asset));
    }
  signal?.throwIfAborted();
  return buffers;
}
export function pcmBuffers(
  buffers: Map<string, AudioBuffer>,
): Map<string, DecodedAudio> {
  return new Map(
    [...buffers].map(([id, b]) => [
      id,
      {
        sampleRate: b.sampleRate,
        channels: Array.from({ length: b.numberOfChannels }, (_, ch) =>
          b.getChannelData(ch),
        ),
      },
    ]),
  );
}


export class CinematicAudioPlayer {
  private ctx?: AudioContext;
  private sources = new Map<AudioBufferSourceNode, () => void>();
  private audio?: AudioTimeline;
  private pcm = new Map<string, DecodedAudio>();
  private timer?: ReturnType<typeof setInterval>;
  private generation = 0;
  private anchor = 0;
  private offset = 0;
  private speed = 1;
  private length = 1;
  private looping = false;
  async play(
    audio: AudioTimeline,
    time: number,
    duration: number,
    speed: number,
    loop: boolean,
  ) {
    this.stop();
    const generation = this.generation;
    if (!hasAudibleAudio(audio)) return false;
    this.ctx ??= context();
    await this.ctx.resume();
    const buffers = await decodeTimeline(audio);
    if (generation !== this.generation) return false;
    const ctx = this.ctx;
    this.audio = audio;
    this.pcm = pcmBuffers(buffers);
    this.anchor = ctx.currentTime + 0.035;
    this.offset = time;
    this.length = duration;
    this.speed = speed;
    this.looping = loop;
    const schedule = (cycleOffset: number, cycleAnchor: number) => {
      for (const clip of audio.clips) {
        const track = audio.tracks.find((t) => t.id === clip.trackId),
          buffer = buffers.get(clip.assetId);
        if (clip.muted || !track || !audibleTrack(audio, track) || !buffer)
          continue;
        const begin = Math.max(cycleOffset, 0, clip.start),
          end = Math.min(duration, clip.start + clipDuration(clip));
        if (end <= begin) continue;
        const source = ctx.createBufferSource(),
          gain = ctx.createGain(),
          pan = ctx.createStereoPanner();
        source.buffer = buffer;
        source.playbackRate.value = clip.rate * speed;
        pan.pan.value = Math.max(-1, Math.min(1, clip.pan + track.pan));
        const when = cycleAnchor + (begin - cycleOffset) / speed,
          level = dbGain(clip.gainDb + track.gainDb + audio.masterGainDb);
        const boundaries = [
          ...new Set([
            begin,
            end,
            Math.max(begin, Math.min(end, clip.start + clip.fadeIn)),
            Math.max(
              begin,
              Math.min(end, clip.start + clipDuration(clip) - clip.fadeOut),
            ),
          ]),
        ].sort((a, b) => a - b);
        gain.gain.setValueAtTime(
          level * clipEnvelope(clip, begin - clip.start),
          when,
        );
        for (let n = 0; n < boundaries.length - 1; n++) {
          const a = boundaries[n],
            b = boundaries[n + 1];
          if (b <= a) continue;

          const curve = Float32Array.from(
            { length: 65 },
            (_, i) =>
              level *
              clipEnvelope(
                clip,
                Math.min(
                  clipDuration(clip) - 1e-9,
                  a + ((b - a) * i) / 64 - clip.start,
                ),
              ),
          );
          gain.gain.setValueCurveAtTime(
            curve,
            cycleAnchor + (a - cycleOffset) / speed,
            (b - a) / speed,
          );
        }
        source.connect(gain).connect(pan).connect(ctx.destination);
        const cleanup = () => {
          source.disconnect();
          gain.disconnect();
          pan.disconnect();
          this.sources.delete(source);
        };
        this.sources.set(source, cleanup);
        source.onended = cleanup;
        source.start(when, clip.in + (begin - clip.start) * clip.rate);
        source.stop(when + (end - begin) / speed);
      }
    };
    schedule(time, this.anchor);
    if (loop) {
      let next = this.anchor + (duration - time) / speed;
      const pump = () => {
        while (next < ctx.currentTime + 0.5) {
          schedule(0, next);
          next += duration / speed;
        }
      };
      pump();
      this.timer = setInterval(pump, 100);
    }
    return true;
  }
  time() {
    const value =
      this.offset +
      Math.max(0, (this.ctx?.currentTime ?? this.anchor) - this.anchor) *
        this.speed;
    return this.looping ? value % this.length : value;
  }
  peak() {
    return this.audio
      ? mixAudioBlock(
          this.audio,
          this.pcm,
          Math.floor(this.time() * 48000),
          128,
        ).peak
      : 0;
  }
  stop() {
    this.generation++;
    if (this.timer) clearInterval(this.timer);
    this.timer = undefined;
    for (const [source, cleanup] of this.sources) {
      source.onended = null;
      try {
        source.stop();
      } catch {

      }
      cleanup();
    }
    this.sources.clear();
    this.audio = undefined;
    this.pcm.clear();
  }
  dispose() {
    this.stop();
    void this.ctx?.close();
    this.ctx = undefined;
  }
}

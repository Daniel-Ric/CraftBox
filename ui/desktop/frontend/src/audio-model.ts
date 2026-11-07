import type { AssetFile } from "./bridge";
export type AudioEntry = {
  id: string;
  name: string;
  files: AssetFile[];
  grouped: boolean;
};
const folderOf = (path: string) =>
  path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "Sounds";
export function groupAudioFiles(
  files: AssetFile[],
  grouped: boolean,
): AudioEntry[] {
  const folders = new Map<string, AssetFile[]>();
  for (const file of files)
    if (file.kind === "sound") {
      const folder = folderOf(file.path);
      if (!folders.has(folder)) folders.set(folder, []);
      folders.get(folder)!.push(file);
    }
  const emitted = new Set<string>();
  const result: AudioEntry[] = [];
  for (const file of files) {
    const folder = folderOf(file.path);
    const members = folders.get(folder);
    if (grouped && file.kind === "sound" && members && members.length > 1) {
      if (!emitted.has(folder))
        result.push({
          id: `group:${folder}`,
          name: folder,
          files: [...members].sort((a, b) =>
            a.path.localeCompare(b.path, "en", { numeric: true }),
          ),
          grouped: true,
        });
      emitted.add(folder);
    } else
      result.push({
        id: file.path,
        name: file.path,
        files: [file],
        grouped: false,
      });
  }
  return result;
}
export type AudioClip = {
  id: string;
  path: string;
  duration: number;
  start: number;
  trimStart: number;
  trimEnd: number;
  gain: number;
  pan: number;
  mute: boolean;
  solo: boolean;
};
export function arrangeAudio<
  T extends {
    id: string;
    duration: number;
    trimStart?: number;
    trimEnd?: number;
  },
>(items: T[], mode: "sequence" | "parallel") {
  let cursor = 0;
  return items.map((item) => {
    const track = {
      gain: 0,
      pan: 0,
      mute: false,
      solo: false,
      ...item,
      start: mode === "sequence" ? cursor : 0,
      trimStart: item.trimStart ?? 0,
      trimEnd: item.trimEnd ?? item.duration,
    };
    cursor += track.trimEnd - track.trimStart;
    return track;
  });
}
export function clipWindow(
  clip: Pick<AudioClip, "start" | "trimStart" | "trimEnd">,
  position: number,
  end: number,
) {
  const from = Math.max(position, clip.start),
    to = Math.min(end, clip.start + clip.trimEnd - clip.trimStart);
  return to > from
    ? {
        when: from - position,
        offset: clip.trimStart + from - clip.start,
        duration: to - from,
      }
    : null;
}
export function audible(
  clip: Pick<AudioClip, "id" | "mute" | "solo">,
  clips: Pick<AudioClip, "id" | "solo">[],
) {
  return !clip.mute && (!clips.some((c) => c.solo) || clip.solo);
}
export function wavePeaks(samples: Float32Array, bins = 8192) {
  bins = Math.min(bins, samples.length);
  const peaks = new Float32Array(bins * 2);
  for (let i = 0; i < bins; i++) {
    let low = 0,
      high = 0;
    for (
      let j = Math.floor((i * samples.length) / bins);
      j < Math.floor(((i + 1) * samples.length) / bins);
      j++
    ) {
      low = Math.min(low, samples[j]);
      high = Math.max(high, samples[j]);
    }
    peaks[i * 2] = low;
    peaks[i * 2 + 1] = high;
  }
  return peaks;
}
export function audioChannels(data: Uint8Array) {
  const tag = (at: number, text: string) =>
    at >= 0 &&
    at + text.length <= data.length &&
    [...text].every((c, i) => data[at + i] === c.charCodeAt(0));
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
  if (tag(0, "RIFF") && tag(8, "WAVE")) {
    for (let at = 12; at + 8 <= data.length;) {
      const size = view.getUint32(at + 4, true);
      if (tag(at, "fmt ") && size >= 16 && at + 24 <= data.length)
        return view.getUint16(at + 10, true) || 32;
      at += 8 + size + (size % 2);
    }
  }
  if (tag(0, "OggS")) {
    let channels = 0,
      pages = 0;
    for (let at = 0; at + 27 <= data.length && pages++ < 50000;) {
      if (!tag(at, "OggS")) return 32;
      const n = data[at + 26],
        body = at + 27 + n;
      if (body > data.length) return 32;
      let size = 0;
      for (let i = at + 27; i < body; i++) size += data[i];
      if (body + size > data.length) return 32;
      if (data[at + 5] & 2) {
        let count = 0;
        if (
          data[body] === 1 &&
          tag(body + 1, "vorbis") &&
          body + 12 <= data.length
        )
          count = data[body + 11];
        else if (tag(body, "OpusHead") && body + 10 <= data.length)
          count = data[body + 9];
        if (!count) return 32;
        channels = Math.max(channels, count);
      }
      at = body + size;
    }
    return pages >= 50000 ? 32 : channels || 32;
  }
  let at = 0;
  if (tag(0, "ID3") && data.length >= 10)
    at =
      10 +
      ((data[6] & 127) * 2097152 +
        (data[7] & 127) * 16384 +
        (data[8] & 127) * 128 +
        (data[9] & 127)) +
      (data[5] & 16 ? 10 : 0);
  if (
    at + 4 <= data.length &&
    data[at] === 255 &&
    (data[at + 1] & 224) === 224 &&
    (data[at + 1] & 6) !== 0 &&
    (data[at + 1] & 24) !== 8
  )
    return data[at + 3] >> 6 === 3 ? 1 : 2;
  return 32;
}
export function parseTime(value: string): number | null {
  if (!/^\d+(?::\d{1,2}){0,2}(?:\.\d+)?$/.test(value.trim())) return null;
  const parts = value.trim().split(":").map(Number);
  if (parts.slice(1).some((v) => v >= 60)) return null;
  const time = parts.reduce((n, v) => n * 60 + v, 0);
  return Number.isFinite(time) ? time : null;
}
export function audioTime(value: number) {
  const v = Math.max(0, Number.isFinite(value) ? value : 0);
  return `${Math.floor(v / 60)}:${(v % 60).toFixed(2).padStart(5, "0")}`;
}

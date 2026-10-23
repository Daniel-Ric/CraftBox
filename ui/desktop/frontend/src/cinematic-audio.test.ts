import { expect, it } from "vitest";
import { createSequence, parseSequence } from "./cinematic";
import {
  mixAudioBlock,
  parseAudioTimeline,
  editAudioClipTiming,
  type AudioTimeline,
} from "./cinematic-audio";
import { addMarker, editMarker, adjacentMarker } from "./cinematic-markers";

const audio = (): AudioTimeline => ({
  assets: [
    {
      id: "a",
      name: "Tone.wav",
      mime: "audio/wav",
      data: "AA==",
      duration: 1,
      sampleRate: 48000,
      channels: 1,
      peaks: [[-1, 1]],
    },
  ],
  tracks: [
    { id: "t", name: "Musik", gainDb: 0, pan: 0, muted: false, solo: false },
  ],
  clips: [
    {
      id: "c",
      name: "Tone",
      assetId: "a",
      trackId: "t",
      start: 0,
      in: 0,
      out: 1,
      rate: 1,
      gainDb: 0,
      pan: 0,
      fadeIn: 0,
      fadeOut: 0,
      muted: false,
    },
  ],
  masterGainDb: 0,
});
it("mixes source trim, playback rate, gain and native stereo panning on the exact sample clock", () => {
  const a = audio();
  a.clips[0] = {
    ...a.clips[0],
    start: 0.25,
    in: 0.25,
    rate: 2,
    gainDb: -6.020599913,
    pan: -1,
  };
  const buffers = new Map([
    ["a", { sampleRate: 4, channels: [new Float32Array([0, 1, 0, -1])] }],
  ]);
  const block = mixAudioBlock(a, buffers, 0, 8, 8);
  expect(Array.from(block.channels[0])).toEqual(expect.arrayContaining([0]));
  expect(block.channels[0][2]).toBeCloseTo(0.5);
  expect(block.channels[0][3]).toBeCloseTo(0);
  expect(block.channels[0][4]).toBeCloseTo(-0.5);
  expect(block.channels[0][5]).toBe(0);
  expect(block.channels[1].every((v) => v === 0)).toBe(true);
});
it("applies equal-power fades, solo/mute and reports clipping before limiting output PCM", () => {
  const a = audio();
  a.clips[0].fadeIn = 0.5;
  const buffers = new Map([
    ["a", { sampleRate: 4, channels: [new Float32Array([1, 1, 1, 1])] }],
  ]);
  expect(mixAudioBlock(a, buffers, 0, 4, 4).channels[0][1]).toBeCloseTo(0.5);
  a.clips[0].fadeIn = 0;
  a.clips[0].gainDb = 12;
  const loud = mixAudioBlock(a, buffers, 0, 4, 4);
  expect(loud.peak).toBeGreaterThan(1);
  expect(loud.channels[0][0]).toBe(1);
  a.tracks[0].muted = true;
  expect(mixAudioBlock(a, new Map(), 0, 4, 4).peak).toBe(0);
  a.tracks[0].muted = false;
  a.tracks.push({ ...a.tracks[0], id: "solo", solo: true });
  expect(mixAudioBlock(a, new Map(), 0, 4, 4).peak).toBe(0);
});
it("keeps audio valid on project roundtrip and rejects orphan clips and invalid trim/fade data", () => {
  const s = createSequence();
  s.audio = audio();
  expect(parseSequence(JSON.stringify(s)).audio).toEqual(s.audio);
  const bad = audio();
  bad.clips[0].assetId = "missing";
  expect(() => parseAudioTimeline(bad)).toThrow();
  bad.clips[0].assetId = "a";
  bad.clips[0].out = 2;
  expect(() => parseAudioTimeline(bad)).toThrow();
  expect(() => mixAudioBlock(audio(), new Map(), 0, 4, 4)).toThrow();
});
it("snaps named range markers to frames, preserves neighbors and navigates in sequence order", () => {
  let s = addMarker(createSequence(), 2.011);
  s = addMarker(s, 5);
  const first = s.markers![0];
  expect(first.time).toBe(2);
  s = editMarker(s, first.id, {
    name: "Schnitt",
    color: "#66aaff",
    duration: 1,
    comment: "Auf den Takt",
  });
  expect(adjacentMarker(s, 2.1, 1)?.time).toBe(5);
  expect(adjacentMarker(s, 2.1, -1)?.time).toBe(2);
  expect(parseSequence(JSON.stringify(s)).markers).toEqual(s.markers);
  const bad = structuredClone(s);
  bad.markers![0].duration = 100;
  expect(() => parseSequence(JSON.stringify(bad))).toThrow();
});
it("moves and trims audio clips without changing source timing or allowing invalid ranges", () => {
  const a = audio();
  a.clips[0].start = 2;
  const moved = editAudioClipTiming(a, "c", "move", 1.01, 24, true);
  expect(moved.clips[0].start).toBe(3);
  expect(moved.clips[0].in).toBe(0);
  const trimmed = editAudioClipTiming(a, "c", "in", 0.25, 24, true);
  expect(trimmed.clips[0]).toMatchObject({ start: 2.25, in: 0.25, out: 1 });
  expect(() =>
    parseAudioTimeline(editAudioClipTiming(a, "c", "out", -10, 24, true)),
  ).not.toThrow();
});
it("validates realistic multi-megabyte audio payloads without overflowing the regular-expression stack", () => {
  const a = audio();
  a.assets[0].data = "AAAA".repeat(1500000);
  expect(() => parseAudioTimeline(a)).not.toThrow();
});

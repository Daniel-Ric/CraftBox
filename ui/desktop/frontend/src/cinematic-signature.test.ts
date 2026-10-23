import { expect, it } from "vitest";
import { createSequence } from "./cinematic";
import { cinematicSignature } from "./cinematic-signature";
import type { CinematicViewState } from "./studio-document";
it("tracks media content and undo while keeping large embedded bytes out of frame signatures", () => {
  const sequence = createSequence();
  sequence.audio = {
    assets: [
      {
        id: "audio",
        name: "a.wav",
        mime: "audio/wav",
        data: "AAAA".repeat(100000),
        duration: 1,
        sampleRate: 48000,
        channels: 1,
        peaks: [[0, 1]],
      },
    ],
    tracks: [],
    clips: [],
    masterGainDb: 0,
  };
  const state = { sequence, time: 0 } as CinematicViewState;
  const first = JSON.stringify(cinematicSignature(state));
  expect(first.length).toBeLessThan(3000);
  expect(JSON.stringify(cinematicSignature({ ...state, time: 1 }))).not.toBe(
    first,
  );
  const changed = {
    ...sequence,
    audio: {
      ...sequence.audio,
      assets: [{ ...sequence.audio.assets[0], data: "BBBB".repeat(100000) }],
    },
  };
  expect(
    JSON.stringify(cinematicSignature({ ...state, sequence: changed })),
  ).not.toBe(first);
  expect(JSON.stringify(cinematicSignature(state))).toBe(first);
});

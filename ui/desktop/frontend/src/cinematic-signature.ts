import type { CinematicViewState } from "./studio-document";
import type { Sequence } from "./cinematic";
const sequences = new WeakMap<Sequence, string>();
const assets = new Map<string, { data: string; hash: string }>();
function fingerprint(id: string, data: string) {
  const cached = assets.get(id);
  if (cached?.data === data) return cached.hash;
  let a = 2166136261,
    b = 5381;
  for (let i = 0; i < data.length; i++) {
    const c = data.charCodeAt(i);
    a = Math.imul(a ^ c, 16777619);
    b = Math.imul(b, 33) ^ c;
  }
  const hash = `${data.length}:${a >>> 0}:${b >>> 0}`;
  if (assets.size >= 32) assets.delete(assets.keys().next().value!);
  assets.set(id, { data, hash });
  return hash;
}

export function cinematicSignature(state?: CinematicViewState) {
  if (!state) return undefined;
  let signature = sequences.get(state.sequence);
  if (!signature) {
    const sequence = state.sequence;
    signature = JSON.stringify({
      ...sequence,
      ...(sequence.audio
        ? {
            audio: {
              ...sequence.audio,
              assets: sequence.audio.assets.map((a) => ({
                ...a,
                data: fingerprint(a.id, a.data),
              })),
            },
          }
        : {}),
    });
    sequences.set(sequence, signature);
  }
  return { ...state, sequence: signature };
}

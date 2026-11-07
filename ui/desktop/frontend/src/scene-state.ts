import type { SceneChunk } from "./bridge";

let configuredSceneBudget = 512 * 1024 * 1024;
export function configureSceneBudget(bytes?: number) {
  if (
    bytes !== undefined &&
    Number.isSafeInteger(bytes) &&
    bytes >= 128 * 1024 * 1024 &&
    bytes <= 1024 * 1024 * 1024
  )
    configuredSceneBudget = bytes;
}

type CountableChunk = {
  x: number;
  z: number;
  missingTextures: number;
  approximateBlocks: number;
  parts: {
    indices: { length: number };
    positions: { buffer: { byteLength: number } };
  }[];
};
export class SceneLedger {
  private readonly budget: number;
  constructor(budget = configuredSceneBudget) { this.budget = budget; }
  private entries = new Map<
    string,
    { triangles: number; missing: number; approximate: number; bytes: number }
  >();
  private triangles = 0;
  private missing = 0;
  private approximate = 0;
  bytes = 0;
  clear() {
    this.entries.clear();
    this.triangles = 0;
    this.missing = 0;
    this.approximate = 0;
    this.bytes = 0;
  }
  update(chunk: CountableChunk) {
    const key = `${chunk.x},${chunk.z}`,
      old = this.entries.get(key);
    const next = {
      triangles: chunk.parts.reduce((n, p) => n + p.indices.length / 3, 0),
      missing: chunk.missingTextures,
      approximate: chunk.approximateBlocks,
      bytes: chunk.parts.reduce((n, p) => n + p.positions.buffer.byteLength, 0),
    };
    if (this.bytes - (old?.bytes ?? 0) + next.bytes > this.budget)
      throw new Error("The 3D view is too large. Select a smaller region.");
    this.triangles += next.triangles - (old?.triangles ?? 0);
    this.missing += next.missing - (old?.missing ?? 0);
    this.approximate += next.approximate - (old?.approximate ?? 0);
    this.bytes += next.bytes - (old?.bytes ?? 0);
    this.entries.set(key, next);
  }
  snapshot() {
    return {
      chunks: this.entries.size,
      triangles: this.triangles,
      missing: this.missing,
      approximate: this.approximate,
    };
  }
}

export function mergeChunks(
  previous: Map<string, SceneChunk>,
  chunks: SceneChunk[],
  reset: boolean,
) {
  const next = reset ? new Map<string, SceneChunk>() : new Map(previous);
  for (const chunk of chunks) next.set(`${chunk.x},${chunk.z}`, chunk);
  return next;
}
export function sceneStats(chunks: Map<string, SceneChunk>) {
  let triangles = 0,
    missing = 0,
    approximate = 0;
  for (const chunk of chunks.values()) {
    missing += chunk.missingTextures;
    approximate += chunk.approximateBlocks;
    for (const part of chunk.parts) triangles += part.indices.length / 3;
  }
  return { chunks: chunks.size, triangles, missing, approximate };
}

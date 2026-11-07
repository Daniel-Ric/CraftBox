import { decodePart, decodePatch } from "./scene-wire";
import { decodeSceneLights } from "./render-lighting";
import type { PackedChunk, AtlasPatch } from "./bridge";

const scope = self as unknown as {
  onmessage:
    | ((
        event: MessageEvent<{
          id: number;
          chunks: PackedChunk[];
          patches: AtlasPatch[];
          atlasSize: number;
        }>,
      ) => void)
    | null;
  postMessage: (message: unknown, transfer: Transferable[]) => void;
};
scope.onmessage = ({ data }) => {
  try {
    const chunks = data.chunks.map((chunk) => ({
      ...chunk,
      lights: decodeSceneLights(chunk.lights),
      parts: chunk.parts.map(decodePart),
    }));
    const patches = data.patches.map((patch) =>
      decodePatch(patch, data.atlasSize),
    );
    const transfer: ArrayBuffer[] = [];
    for (const chunk of chunks)
      for (const part of chunk.parts) {
        transfer.push(part.positions.buffer);
        if (part.blockLight) transfer.push(part.blockLight.buffer);
      }
    for (const patch of patches) transfer.push(patch.pixels.buffer);
    scope.postMessage({ id: data.id, batch: { chunks, patches } }, transfer);
  } catch (error) {
    scope.postMessage(
      {
        id: data.id,
        error: String(error instanceof Error ? error.message : error),
      },
      [],
    );
  }
};

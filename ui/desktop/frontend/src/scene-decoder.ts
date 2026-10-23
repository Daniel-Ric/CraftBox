import type { ScenePage } from "./bridge";
import type { DecodedBatch } from "./scene-wire";

export class SceneDecoder {
  private worker: Worker | undefined;
  private failure: Error | undefined;
  private next = 0;
  private pending = new Map<
    number,
    { resolve: (batch: DecodedBatch) => void; reject: (error: Error) => void }
  >();
  constructor() {
    try {
      this.worker = new Worker(
        new URL("./scene-decoder.worker.ts", import.meta.url),
        { type: "module" },
      );
      this.worker.onmessage = ({
        data,
      }: {
        data: { id: number; batch?: DecodedBatch; error?: string };
      }) => {
        const task = this.pending.get(data.id);
        if (!task) return;
        this.pending.delete(data.id);
        if (data.error || !data.batch)
          task.reject(new Error(data.error ?? "3D data unavailable"));
        else task.resolve(data.batch);
      };
      this.worker.onerror = () =>
        this.fail(
          new Error(
            "Could not process 3D data. Reopen the 3D view.",
          ),
        );
    } catch {
      this.fail(
        new Error(
          "Could not start the 3D background worker. Reopen the 3D view.",
        ),
      );
    }
  }
  private fail(error: Error) {
    this.failure = error;
    this.worker?.terminate();
    for (const task of this.pending.values()) task.reject(error);
    this.pending.clear();
  }
  decode(page: ScenePage): Promise<DecodedBatch> {
    if (this.failure) return Promise.reject(this.failure);
    return new Promise((resolve, reject) => {
      const id = ++this.next;
      this.pending.set(id, { resolve, reject });
      try {
        this.worker!.postMessage({
          id,
          chunks: page.chunks,
          patches: page.patches,
          atlasSize: page.atlasSize,
        });
      } catch {
        this.fail(new Error("Could not transfer 3D data"));
      }
    });
  }
  dispose() {
    this.fail(new Error("3D view closed"));
  }
}

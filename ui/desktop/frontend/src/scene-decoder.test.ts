import { afterEach, expect, it, vi } from "vitest";
import { SceneDecoder } from "./scene-decoder";
import type { ScenePage } from "./bridge";

afterEach(() => vi.unstubAllGlobals());
it("rejects future decodes if the worker fails before any page arrives", async () => {
  let worker: { onerror?: () => void };
  vi.stubGlobal(
    "Worker",
    class {
      constructor() {
        worker = this;
      }
      onerror?: () => void;
      terminate() {}
      postMessage() {}
    },
  );
  const decoder = new SceneDecoder();
  worker!.onerror!();
  await expect(decoder.decode({} as ScenePage)).rejects.toThrow();
  decoder.dispose();
});
it("reports construction failures through decode without leaking an effect", async () => {
  vi.stubGlobal(
    "Worker",
    class {
      constructor() {
        throw new Error("blocked");
      }
    },
  );
  const decoder = new SceneDecoder();
  await expect(decoder.decode({} as ScenePage)).rejects.toThrow();
  decoder.dispose();
});

import { afterEach, expect, it, vi } from "vitest";
import { createSequence, type CameraPose } from "./cinematic";
import type { MotionFrameSample } from "./cinematic-motion";

const mock = vi.hoisted(() => ({ create: vi.fn() }));
vi.mock("./cinematic-webm", () => ({ createWebMEncoder: mock.create }));
import { exportCinematic } from "./cinematic-export";
afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

function setup() {
  const events: string[] = [],
    closed: number[] = [];
  vi.stubGlobal(
    "VideoFrame",
    class {
      timestamp: number;
      constructor(_canvas: unknown, options: { timestamp: number }) {
        this.timestamp = options.timestamp;
        events.push(`snapshot:${this.timestamp}`);
      }
      close() {
        closed.push(this.timestamp);
      }
    },
  );
  const sequence = createSequence();
  sequence.duration = 0.25;
  sequence.motionBlur = {
    enabled: true,
    angle: 180,
    phase: -90,
    samples: 4,
    preview: false,
  };
  const encoder = {
    add: vi.fn(async () => {}),
    finish: vi.fn(async () => new Blob(["webm"], { type: "video/webm" })),
    close: vi.fn(async () => {}),
  };
  mock.create.mockImplementation(async () => {
    events.push("encoder-ready");
    return encoder;
  });
  const canvas = document.createElement("canvas"),
    controller = new AbortController();
  const render = vi.fn(
    async (
      _pose: CameraPose,
      _width: number,
      _height: number,
      _time: number,
      _samples?: MotionFrameSample[],
    ) => {
      events.push("render");
      return canvas;
    },
  );
  const options = {
    kind: "webm" as const,
    sequence,
    edge: 720,
    aspect: 16 / 9,
    gifFps: 12,
    signal: controller.signal,
    render,
    progress: () => {},
  };
  return { events, closed, encoder, controller, options, render };
}
it("initializes the encoder before rendering and snapshots every frame at deterministic timestamps", async () => {
  const p = setup();
  expect((await exportCinematic(p.options)).type).toBe("video/webm");
  expect(p.events.slice(0, 3)).toEqual([
    "encoder-ready",
    "render",
    "snapshot:0",
  ]);
  expect(p.encoder.add.mock.calls).toHaveLength(6);
  expect(p.closed).toEqual([0, 41667, 83333, 125000, 166667, 208333]);
  expect(p.render.mock.calls[0][4]).toHaveLength(4);
  expect(p.encoder.finish).toHaveBeenCalledOnce();
  expect(p.encoder.close).toHaveBeenCalledOnce();
});
it("closes frames and encoder when cancellation interrupts a WebM export", async () => {
  const p = setup();
  p.encoder.add.mockImplementationOnce(async () => {
    p.controller.abort();
  });
  await expect(exportCinematic(p.options)).rejects.toMatchObject({
    name: "AbortError",
  });
  expect(p.closed).toEqual([0]);
  expect(p.encoder.finish).not.toHaveBeenCalled();
  expect(p.encoder.close).toHaveBeenCalledOnce();
});

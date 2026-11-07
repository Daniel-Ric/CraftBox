import { expect, it, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/react";
import AutoDirectionControls from "../src/AutoDirectionControls";
import { renderDefaults } from "../src/render-settings";
import * as T from "three";
import {
  sampleScene,
  unpackSamples,
  AnalysisCache,
} from "../src/scene-analysis";
import { analyseModel } from "../src/auto-direction";

function mesh(triangles: number, mode = "solid") {
  const geometry = new T.BufferGeometry();
  geometry.setAttribute(
    "position",
    new T.Float32BufferAttribute([0, 0, 0, 2, 0, 0, 0, 0, 2], 3),
  );
  geometry.setAttribute(
    "normal",
    new T.Float32BufferAttribute([0, 1, 0, 0, 1, 0, 0, 1, 0], 3),
  );
  geometry.setAttribute(
    "uv",
    new T.Float32BufferAttribute([0, 0, 0, 0, 0, 0], 2),
  );
  geometry.setIndex(Array.from({ length: triangles * 3 }, (_, i) => i % 3));
  return { geometry, mode, offset: [0, 0, 0] as [number, number, number] };
}
const atlas = {
  pixels: new Uint8Array([255, 128, 0, 255]),
  width: 1,
  height: 1,
};
afterEach(cleanup);
it("uses area-weighted luminance tails rather than only an average", () => {
  const surface = (color: number, area: number) => ({
    position: [0, 0, 0],
    normal: [0, 1, 0],
    color: [color, color, color],
    area,
    mode: "solid",
  });
  const a = analyseModel([surface(0.5, 100)], { triangles: 100, lights: 0 });
  const b = analyseModel([surface(0.05, 50), surface(0.95, 50)], {
    triangles: 100,
    lights: 0,
  });
  expect(a.brightness).toBeCloseTo(b.brightness);
  expect(b.luminance!.p90 - b.luminance!.p10).toBeGreaterThan(0.8);
  expect(a.luminance!.p90 - a.luminance!.p10).toBeLessThan(0.05);
});
it("shows the actual phase immediately and aborts underlying work", async () => {
  let actualSignal: AbortSignal | undefined;
  render(
    <AutoDirectionControls
      ready
      settings={renderDefaults}
      onApply={() => {
        throw Error("Cancelled work must not apply");
      }}
      analyse={(progress, signal) =>
        new Promise((_resolve, reject) => {
          actualSignal = signal;
          progress?.(0.34, {
            stage: "surfaces",
            label: "Sampling Hive material families",
            detail: "12,000 / 48,000 probes",
          });
          signal?.addEventListener(
            "abort",
            () => reject(new DOMException("Cancelled", "AbortError")),
            { once: true },
          );
        })
      }
    />,
  );
  fireEvent.click(screen.getByRole("button", { name: "Auto-style scene" }));
  expect(
    await screen.findByText("Sampling Hive material families"),
  ).toBeTruthy();
  expect(screen.getByText("12,000 / 48,000 probes")).toBeTruthy();
  expect(screen.getByRole("progressbar").getAttribute("value")).toBe("0.34");
  fireEvent.click(screen.getByRole("button", { name: "Cancel analysis" }));
  expect(actualSignal?.aborted).toBe(true);
});
it("bounds work while preserving small material strata and area weights", async () => {
  const events: any[] = [];
  const result = await sampleScene(
    [mesh(100000), mesh(2, "water"), mesh(1, "mask")],
    atlas,
    (p, status) => events.push({ p, ...status }),
    undefined,
    100,
  );
  const samples = unpackSamples(result);
  expect(result.count).toBeLessThanOrEqual(100);
  expect(samples.some((s) => s.mode === "water")).toBe(true);
  expect(samples.some((s) => s.mode === "mask")).toBe(true);
  expect(samples.reduce((sum, s) => sum + s.area, 0)).toBeCloseTo(200006, 0);
  expect(events[0].stage).toBe("prepare");
  expect(events.some((e) => e.stage === "surfaces" && e.completed > 0)).toBe(
    true,
  );
  expect(events.every((e, i) => i === 0 || e.p >= events[i - 1].p)).toBe(true);
});
it("is deterministic, uses rectangular atlases and ignores transparent samples", async () => {
  const input = [mesh(300)];
  const a = await sampleScene(input, atlas, undefined, undefined, 20);
  const b = await sampleScene(input, atlas, undefined, undefined, 20);
  expect(a.data).toEqual(b.data);
  const rectangular = mesh(1);
  rectangular.geometry.attributes.uv.array.fill(0.75);
  const image = {
    width: 1,
    height: 2,
    pixels: new Uint8Array([0, 0, 0, 0, 0, 255, 0, 255]),
  };
  expect(
    unpackSamples(await sampleScene([rectangular], image))[0].color,
  ).toEqual([0, 1, 0]);
  expect(
    (await sampleScene(input, { ...atlas, pixels: new Uint8Array(4) })).count,
  ).toBe(0);
});
it("yields and stops real computation after cancellation", async () => {
  const controller = new AbortController();
  const task = sampleScene(
    [mesh(200000)],
    atlas,
    (_p, event) => {
      if (event?.stage === "surfaces" && event.completed! > 0)
        controller.abort();
    },
    controller.signal,
  );
  await expect(task).rejects.toMatchObject({ name: "AbortError" });
});
it("reuses immutable evidence only for the same scene revision and source", () => {
  const cache = new AnalysisCache();
  const result = analyseModel(
    [
      {
        position: [0, 0, 0],
        normal: [0, 1, 0],
        color: [1, 1, 1],
        area: 1,
        mode: "solid",
      },
    ],
    { lights: 0, triangles: 1 },
  );
  cache.set(3, "Hive", result);
  const read = cache.get(3, "Hive")!;
  read.size[0] = 999;
  expect(cache.get(3, "Hive")!.size[0]).not.toBe(999);
  expect(cache.get(4, "Hive")).toBeUndefined();
  expect(cache.get(3, "Other world")).toBeUndefined();
});

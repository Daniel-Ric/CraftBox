import { expect, it, vi } from "vitest";
import { RenderEngine } from "./RenderCanvas";
import { renderDefaults } from "./render-settings";
import { Group } from "three";

it("tracks free-camera edits immediately but ignores initialization and export frames", () => {
  vi.useFakeTimers();
  try {
    const onViewDirty = vi.fn(),
      onViewChange = vi.fn(),
      pose = {
        position: [1, 2, 3],
        target: [0, 0, 0],
        fov: 45,
        roll: 0,
        exposure: 1,
      };
    const engine = {
      loading: false,
      initializing: true,
      disposed: false,
      exporting: false,
      callbacks: { current: { onViewDirty, onViewChange } },
      camera: { zoom: 2 },
      snapshot: () => pose,
    } as unknown as RenderEngine;
    RenderEngine.prototype.scheduleViewChange.call(engine);
    expect(onViewDirty).not.toHaveBeenCalled();
    Object.assign(engine, { initializing: false });
    RenderEngine.prototype.scheduleViewChange.call(engine);
    expect(onViewDirty).toHaveBeenCalledOnce();
    vi.advanceTimersByTime(150);
    expect(onViewChange).toHaveBeenCalledWith({ ...pose, zoom: 2 });
    Object.assign(engine, { exporting: true });
    RenderEngine.prototype.scheduleViewChange.call(engine);
    expect(onViewDirty).toHaveBeenCalledOnce();
  } finally {
    vi.useRealTimers();
  }
});

it("defers settings and sky work while terrain is still loading", () => {
  const engine = { loading: true, options: renderDefaults } as RenderEngine;
  const look = { ...renderDefaults, exposure: 1.4, sky: "sunset" };

  expect(() => RenderEngine.prototype.apply.call(engine, look)).not.toThrow();
  expect(engine.options).toBe(renderDefaults);
});

it("does not draw effects or partial geometry before terrain completion", () => {
  const engine = { loading: true } as RenderEngine;
  expect(() => RenderEngine.prototype.draw.call(engine)).not.toThrow();
});

it("applies the latest requested look once and waits for its sky before reporting ready", async () => {
  const latest = { ...renderDefaults, exposure: 1.4 };
  let finishSky!: () => void;
  const skyTask = new Promise<void>((resolve) => {
    finishSky = resolve;
  });
  const applied = vi.fn(),
    ready = vi.fn();
  const engine = {
    loading: true,
    terrain: new Group(),
    atmosphere: { setClearance: vi.fn() },
    disposed: false,
    options: renderDefaults,
    callbacks: { current: { settings: latest, onReady: ready } },
    apply: applied,
    light: { castShadow: false },
    fit: vi.fn(),
    setMapView: vi.fn(),
    skyTask,
  } as unknown as RenderEngine;
  const completion = RenderEngine.prototype.finishLoading.call(engine);
  await Promise.resolve();
  expect(applied).toHaveBeenCalledExactlyOnceWith(latest, undefined);
  expect(ready).not.toHaveBeenCalled();
  finishSky();
  await completion;
  expect(ready).toHaveBeenCalledOnce();
  expect(engine.light.castShadow).toBe(true);
});

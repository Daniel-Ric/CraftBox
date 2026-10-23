import { it, expect, vi } from "vitest";
import { createSequence, sampleSequence } from "./cinematic";
import { PerspectiveCamera, Vector3, Group } from "three";
import { RenderEngine } from "./RenderCanvas";
it("locks viewport fit and view commands while an export is rendering", () => {
  const engine = { exporting: true } as RenderEngine;
  expect(() => RenderEngine.prototype.fit.call(engine)).not.toThrow();
  expect(() => RenderEngine.prototype.view.call(engine, 45, 30)).not.toThrow();
});
it("restores the inspection camera and controls after producing a cinematic thumbnail", () => {
  const camera = new PerspectiveCamera(53);
  camera.position.set(4, 5, 6);
  camera.up.set(0, 1, 0);
  camera.lookAt(0, 0, 0);
  const position = camera.position.clone(),
    rotation = camera.quaternion.clone(),
    target = new Vector3(1, 2, 3),
    path = new Group();
  const engine = {
    camera,
    controls: { target: target.clone(), enabled: true },
    path,
    loading: false,
    disposed: false,
    exporting: false,
    renderer: {
      toneMappingExposure: 1,
      domElement: {
        toDataURL: () => {
          expect(engine.exporting).toBe(true);
          return "data:image/png;base64,preview";
        },
      },
    },
    callbacks: {
      current: {
        shot: { pose: { position: [1, 1, 1] }, sequence: createSequence() },
      },
    },
    drawCapture: RenderEngine.prototype.drawCapture,
    draw(this: RenderEngine, override?: unknown) {
      if (override) {
        camera.position.set(10, 20, 30);
        camera.fov = 20;
        camera.lookAt(3, 4, 5);
        this.controls.target.set(3, 4, 5);
        this.controls.enabled = false;
        this.path.visible = false;
        this.renderer.toneMappingExposure = 2;
      }
    },
  } as unknown as RenderEngine;
  expect(RenderEngine.prototype.preview.call(engine)).toBe(
    "data:image/png;base64,preview",
  );
  expect(camera.position).toEqual(position);
  expect(camera.quaternion.toArray()).toEqual(rotation.toArray());
  expect(camera.fov).toBe(53);
  expect(engine.controls.target).toEqual(target);
  expect(engine.controls.enabled).toBe(true);
  expect(path.visible).toBe(true);
  expect(engine.renderer.toneMappingExposure).toBe(1);
  expect(engine.exporting).toBe(false);
});
it("renders a cinematic capture with full shutter quality independently of the preview switch", () => {
  const sequence = createSequence();
  sequence.motionBlur = {
    enabled: true,
    angle: 180,
    phase: -90,
    samples: 16,
    preview: false,
  };
  const drawTemporal = vi.fn();
  const engine = {
    callbacks: {
      current: {
        shot: { sequence, time: 6, pose: sampleSequence(sequence, 6) },
      },
    },
    drawTemporal,
  } as unknown as RenderEngine;
  RenderEngine.prototype.drawCapture.call(engine);
  expect(drawTemporal.mock.calls[0][0]).toHaveLength(16);
  RenderEngine.prototype.drawCapture.call(engine, true);
  expect(drawTemporal.mock.calls[1][0]).toHaveLength(8);
});

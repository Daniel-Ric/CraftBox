import { expect, it } from "vitest";
import { PerspectiveCamera, Vector3 } from "three";
import { FreeCameraNavigation, snapshotRoll } from "./free-camera";
import { createSequence, parseSequence } from "./cinematic";
it("keeps a camera roll crossing 180 degrees valid when saving and reopening a sequence", () => {
  const camera = new PerspectiveCamera();
  camera.position.set(0, 0, 10);
  camera.lookAt(0, 0, 0);
  camera.rotateZ((185 * Math.PI) / 180);
  const sequence = createSequence();
  sequence.keys[0].roll = snapshotRoll(camera, new Vector3());
  expect(parseSequence(JSON.stringify(sequence)).keys[0].roll).toBeCloseTo(
    -175,
  );
});
it("flies relative to the view only while the viewport is focused and stops on blur or disable", () => {
  const element = document.createElement("canvas");
  document.body.append(element);
  element.tabIndex = 0;
  const camera = new PerspectiveCamera();
  camera.position.set(0, 0, 10);
  camera.lookAt(0, 0, 0);
  const target = new Vector3();
  const nav = new FreeCameraNavigation(
    camera,
    element,
    target,
    () => {},
    () => {},
  );
  nav.enabled = true;
  element.focus();
  element.dispatchEvent(
    new KeyboardEvent("keydown", { code: "KeyW", bubbles: true }),
  );
  nav.update(1, 2);
  expect(camera.position.z).toBeCloseTo(8);
  expect(target.z).toBeCloseTo(-2);
  element.blur();
  nav.update(1, 2);
  expect(camera.position.z).toBeCloseTo(8);
  element.focus();
  element.dispatchEvent(
    new KeyboardEvent("keydown", { code: "KeyE", bubbles: true }),
  );
  nav.update(1, 2);
  expect(camera.position.y).toBeCloseTo(2);
  nav.enabled = false;
  nav.enabled = true;
  nav.update(1, 2);
  expect(camera.position.y).toBeCloseTo(2);
  nav.dispose();
  element.remove();
});

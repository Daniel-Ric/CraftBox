import { expect, it } from "vitest";
import {
  circleOfConfusion,
  focusDepth,
  DepthOfFieldPass,
} from "./render-depth-of-field";
import { OrthographicCamera, PerspectiveCamera, Vector3, Texture } from "three";
import { renderDefaults } from "./render-settings";

it("uses signed thin-lens blur with a sharp focus plane and aperture-dependent blur", () => {
  const options = {
    focus: 2,
    focalLength: 50,
    fStop: 2,
    sensorWidth: 36,
    width: 1920,
  };
  expect(circleOfConfusion(2, options)).toBe(0);
  expect(circleOfConfusion(1, options)).toBeLessThan(0);
  expect(circleOfConfusion(4, options)).toBeGreaterThan(0);
  expect(circleOfConfusion(4, { ...options, fStop: 4 })).toBeCloseTo(
    circleOfConfusion(4, options) / 2,
  );
});
it("focuses at view-space depth, not radial distance, for both projections", () => {
  for (const camera of [
    new PerspectiveCamera(),
    new OrthographicCamera(-2, 2, 2, -2),
  ]) {
    camera.position.set(0, 0, 10);
    camera.lookAt(0, 0, 0);
    camera.updateMatrixWorld(true);
    expect(focusDepth(new Vector3(3, 0, 2), camera)).toBeCloseTo(8);
  }
});
it("uses the actual sensor aspect and prevents a focus plane inside the lens", () => {
  const pass = new DepthOfFieldPass(),
    camera = new PerspectiveCamera(45, 16 / 9);
  pass.configure(
    {
      ...renderDefaults,
      dofEnabled: true,
      dofDistance: 0.001,
      dofWorldScale: 1000,
    },
    camera,
    new Texture(),
  );
  expect(pass.uniforms.sensorWidth.value).toBeCloseTo((24 * 16) / 9);
  expect(pass.uniforms.focus.value).toBeGreaterThan(pass.uniforms.focal.value);
  pass.dispose();
});

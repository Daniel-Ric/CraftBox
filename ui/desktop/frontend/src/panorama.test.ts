import { expect, it } from "vitest";
import {
  cubeFaces,
  equirectangularDirection,
  projectCubeDirection,
  stitchPanorama,
  decodeHDRFace,
} from "./panorama";
import { PerspectiveCamera, Vector3, DataUtils } from "three";

it("maps the full sphere, poles and seam to six correctly oriented cameras", () => {
  expect(equirectangularDirection(0.5, 0.5)).toEqual([0, 0, -1]);
  expect(equirectangularDirection(0.5, 0)[1]).toBeCloseTo(1);
  expect(equirectangularDirection(0.5, 1)[1]).toBeCloseTo(-1);
  expect(equirectangularDirection(0, 0.5)[2]).toBeCloseTo(
    equirectangularDirection(1, 0.5)[2],
  );
  cubeFaces.forEach((face, index) => {
    expect(projectCubeDirection(face.direction, 92)).toEqual({
      face: index,
      u: 0.5,
      v: 0.5,
    });
  });
});
it("stitches an actual 2:1 image with all six directions and preserves alpha", async () => {
  const faces = cubeFaces.map((_, i) => ({
    width: 2,
    height: 2,
    data: new Uint8ClampedArray(
      Array.from({ length: 4 }, () => [i * 40, 20, 30, 128]).flat(),
    ),
  }));
  const output = await stitchPanorama(faces, 128);
  expect(output.width).toBe(128);
  expect(output.height).toBe(64);
  const colors = new Set<number>();
  for (let i = 0; i < output.data.length; i += 4) {
    colors.add(output.data[i]);
    expect(output.data[i + 3]).toBe(128);
  }
  expect(colors.size).toBe(6);
});
it("reconstructs matching Three.js camera rays across every cube-face boundary", () => {
  for (let y = 1; y < 20; y++)
    for (let x = 0; x < 40; x++) {
      const direction = equirectangularDirection(x / 40, y / 20),
        projected = projectCubeDirection(direction);
      const face = cubeFaces[projected.face],
        camera = new PerspectiveCamera(92, 1, 0.1, 1000);
      camera.up.fromArray(face.up);
      camera.lookAt(new Vector3().fromArray(face.direction));
      camera.updateMatrixWorld(true);
      const actual = new Vector3(projected.u * 2 - 1, 1 - projected.v * 2, 0)
        .unproject(camera)
        .sub(camera.position)
        .normalize();
      expect(
        actual.distanceTo(new Vector3().fromArray(direction)),
      ).toBeLessThan(1e-6);
    }
});
it("retains unclipped linear HDR values for one global Pan postprocess", async () => {
  const faces = cubeFaces.map(() => ({
    width: 2,
    height: 2,
    data: new Float32Array(
      Array.from({ length: 4 }, () => [16, 2, 0.5, 0.75]).flat(),
    ),
  }));
  const result = await stitchPanorama(faces, 64);
  expect(result.data).toBeInstanceOf(Float32Array);
  expect([...result.data.slice(0, 4)]).toEqual([16, 2, 0.5, 0.75]);
});
it("unpremultiplies and vertically flips raw HDR framebuffer samples exactly once", () => {
  const encoded = new Uint16Array(
    [2, 1, 0.5, 0.5, 9, 9, 9, 0, 0.25, 0.5, 1, 0.25, 1, 1, 1, 1].map(
      DataUtils.toHalfFloat,
    ),
  );
  const decoded = decodeHDRFace(encoded, 2);
  expect([...decoded.data.slice(0, 4)]).toEqual([1, 2, 4, 0.25]);
  expect([...decoded.data.slice(8, 12)]).toEqual([4, 2, 1, 0.5]);
  expect([...decoded.data.slice(12, 16)]).toEqual([0, 0, 0, 0]);
});

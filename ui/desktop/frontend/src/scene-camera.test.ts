import { describe, expect, it } from "vitest";
import * as THREE from "three";
import { modelFrame } from "./scene-camera";

describe("centered model framing", () => {
  it("uses the geometric center in all three axes", () => {
    const bounds = new THREE.Box3(
      new THREE.Vector3(-120, -64, 18),
      new THREE.Vector3(80, 224, 82),
    );
    expect(modelFrame(bounds, 55, 1.5)?.center.toArray()).toEqual([
      -20, 80, 50,
    ]);
    expect(modelFrame(new THREE.Box3(), 55, 1)).toBeNull();
  });
  it.each([0.3, 0.6, 1, 1.6, 3])(
    "keeps every corner visible while rotating at aspect %s",
    (aspect) => {
      for (const size of [
        new THREE.Vector3(900, 12, 36),
        new THREE.Vector3(20, 650, 30),
        new THREE.Vector3(250, 250, 250),
      ]) {
        const bounds = new THREE.Box3().setFromCenterAndSize(
          new THREE.Vector3(37, -45, 290),
          size,
        );
        const frame = modelFrame(bounds, 55, aspect)!;
        const camera = new THREE.PerspectiveCamera(55, aspect, 0.1, frame.far);
        for (const direction of [
          new THREE.Vector3(1, 0.9, 1),
          new THREE.Vector3(-1, 0.15, 0),
          new THREE.Vector3(0.1, 1, -0.2),
        ]) {
          camera.position
            .copy(frame.center)
            .addScaledVector(direction.normalize(), frame.distance);
          camera.lookAt(frame.center);
          camera.updateMatrixWorld();
          expect(frame.center.clone().project(camera).x).toBeCloseTo(0);
          expect(frame.center.clone().project(camera).y).toBeCloseTo(0);
          for (const x of [bounds.min.x, bounds.max.x])
            for (const y of [bounds.min.y, bounds.max.y])
              for (const z of [bounds.min.z, bounds.max.z]) {
                const p = new THREE.Vector3(x, y, z).project(camera);
                expect(Math.abs(p.x)).toBeLessThan(1);
                expect(Math.abs(p.y)).toBeLessThan(1);
                expect(p.z).toBeGreaterThan(-1);
                expect(p.z).toBeLessThan(1);
              }
        }
      }
    },
  );
});

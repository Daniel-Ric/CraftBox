import { expect, it } from "vitest";
import * as THREE from "three";
import { skyViewRotation } from "./render-sky-background";

it("aligns the Pan with the actual camera pitch without an isometric offset", () => {
  const camera = new THREE.OrthographicCamera();
  camera.position.set(
    0,
    Math.sin((35 * Math.PI) / 180) * 10,
    Math.cos((35 * Math.PI) / 180) * 10,
  );
  camera.lookAt(0, 0, 0);
  const ray = new THREE.Vector3(0, 0, -1).applyMatrix3(
    skyViewRotation(camera, 0, 0),
  );
  expect(ray.y).toBeCloseTo(-Math.sin((35 * Math.PI) / 180));
  camera.position.set(
    0,
    Math.sin((60 * Math.PI) / 180) * 10,
    Math.cos((60 * Math.PI) / 180) * 10,
  );
  camera.lookAt(0, 0, 0);
  const tilted = new THREE.Vector3(0, 0, -1).applyMatrix3(
    skyViewRotation(camera, 0, 0),
  );
  expect(tilted.y).toBeCloseTo(-Math.sin((60 * Math.PI) / 180));
  expect(tilted.distanceTo(ray)).toBeGreaterThan(0.4);
});

it("uses world orientation and ignores translation for the distant Pan", () => {
  const camera = new THREE.OrthographicCamera(),
    parent = new THREE.Group();
  parent.add(camera);
  parent.rotation.y = 0.7;
  camera.rotation.set(-0.4, 0.2, 0.15);
  const before = skyViewRotation(camera, 35, 10);
  parent.position.set(1000, -300, 50);
  expect(skyViewRotation(camera, 35, 10).elements).toEqual(before.elements);
  const actual = new THREE.Vector3(0.3, 0.2, -1)
    .normalize()
    .applyMatrix3(before);
  const expected = new THREE.Vector3(0.3, 0.2, -1)
    .normalize()
    .applyQuaternion(camera.getWorldQuaternion(new THREE.Quaternion()))
    .applyAxisAngle(new THREE.Vector3(1, 0, 0), (10 * Math.PI) / 180)
    .applyAxisAngle(new THREE.Vector3(0, 1, 0), (-35 * Math.PI) / 180);
  expect(actual.distanceTo(expected)).toBeLessThan(1e-6);
});

it("keeps an elevated sky continuous when adjacent cube cameras observe the same world ray", () => {
  const a = new THREE.PerspectiveCamera(92),
    b = new THREE.PerspectiveCamera(92);
  a.lookAt(0, 0, -1);
  b.lookAt(1, 0, 0);
  a.updateMatrixWorld(true);
  b.updateMatrixWorld(true);
  const worldRay = new THREE.Vector3(1, 0.2, -1).normalize();
  const sample = (camera: THREE.PerspectiveCamera) =>
    worldRay
      .clone()
      .applyQuaternion(camera.quaternion.clone().invert())
      .applyMatrix3(skyViewRotation(camera, 30, 15));
  expect(sample(a).distanceTo(sample(b))).toBeLessThan(1e-6);
});

import { expect, it } from "vitest";
import * as THREE from "three";
import { buildCloudClearance } from "./cloud-clearance";

it("follows separated terrain footprints and leaves their open gap and foreground free", async () => {
  const terrain = new THREE.Group();
  for (const x of [-30, 30]) {
    const mesh = new THREE.Mesh(new THREE.BoxGeometry(12, 20, 12));
    mesh.position.set(x, 10, 0);
    terrain.add(mesh);
  }
  terrain.position.set(100, -40, 50);
  const field = await buildCloudClearance(terrain);
  expect(field).toBeDefined();
  expect(field!.distanceAt(70, 50)).toBeLessThan(2);
  expect(field!.distanceAt(130, 50)).toBeLessThan(2);
  expect(field!.distanceAt(100, 50)).toBeGreaterThan(18);
  expect(field!.distanceAt(100, 100)).toBeGreaterThan(30);
  expect(field!.bottom).toBe(-40);
  field!.texture.dispose();
});

it("does not create a clearance field for empty or cancelled terrain", async () => {
  expect(await buildCloudClearance(new THREE.Group())).toBeUndefined();
  expect(
    await buildCloudClearance(new THREE.Group(), () => true),
  ).toBeUndefined();
});

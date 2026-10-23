import { expect, it } from "vitest";
import { createSequence, parseSequence, sampleSequence } from "./cinematic";
import { independentSequence, addChannelKey } from "./cinematic-tracks";
import { enableCameras, addCamera, cameraSequence, applyCameraEdit, resolveShot, splitShot, removeCamera } from "./cinematic-shots";

it("preserves the migrated flight and keeps camera edits independent through project reload", () => {
  const original = independentSequence(createSequence());
  let project = enableCameras(original);
  const first = project.cameras![0].id;
  project = addCamera(project, first);
  const second = project.cameras![1].id;
  project = applyCameraEdit(project, addChannelKey(cameraSequence(project, second), "fov", 0, 100), second);
  expect(sampleSequence(cameraSequence(project, first), 0).fov).toBe(45);
  expect(project.cameras![1].tracks!.find((t) => t.channel === "fov")!.keys[0].value).toBe(100);
  expect(parseSequence(JSON.stringify(project))).toEqual(project);
  for (let t = 0; t < 12; t += 0.13) expect(sampleSequence(project, t)).toEqual(sampleSequence(original, t));
});
it("cuts exactly at the shot boundary and maps source in/rate without interpolation across cameras", () => {
  let p = addCamera(enableCameras(independentSequence(createSequence())));
  const second = p.cameras![1].id;
  p = applyCameraEdit(p, addChannelKey(cameraSequence(p, second), "fov", 0, 100), second);
  p = splitShot(p, p.shots![0].id, 6);
  p.shots![1] = { ...p.shots![1], cameraId: second, sourceIn: 2, rate: 0.5 };
  expect(resolveShot(p, 6 - 1e-7)!.camera.id).toBe(p.cameras![0].id);
  expect(resolveShot(p, 6)!.camera.id).toBe(second);
  expect(resolveShot(p, 8)!.sourceTime).toBe(3);
  expect(sampleSequence(p, 8).fov).toBe(sampleSequence(cameraSequence(p, second), 3).fov);
  expect(() => removeCamera(p, second)).toThrow();
});
it("holds source time in timeline gaps, rejects malformed references, overlap and recursive camera data", () => {
  const p = enableCameras(independentSequence(createSequence()));
  p.shots![0].end = 4;
  expect(resolveShot(p, 8)!.sourceTime).toBe(4);
  const bad = structuredClone(p);
  bad.shots![0].cameraId = "missing";
  expect(() => parseSequence(JSON.stringify(bad))).toThrow();
  const overlap = structuredClone(p);
  overlap.shots!.push({ ...overlap.shots![0], id: "overlap", start: 2, end: 6 });
  expect(() => parseSequence(JSON.stringify(overlap))).toThrow();
  const nested = JSON.parse(JSON.stringify(p));
  nested.cameras[0].cameras = nested.cameras;
  expect(() => parseSequence(JSON.stringify(nested))).toThrow();
});
it("keeps the root fallback reloadable when the program is shorter than a source camera", () => {
  const p = enableCameras(independentSequence(createSequence()));
  p.shots![0].end = 8;
  const source = cameraSequence(p, p.cameras![0].id);
  source.keys = source.keys.map((k) => ({ ...k, time: k.time * 2 / 3 }));
  source.tracks = source.tracks!.map((t) => ({ ...t, keys: t.keys.map((k) => ({ ...k, time: k.time * 2 / 3 })) }));
  const changed = applyCameraEdit(p, { ...source, duration: 8 }, p.cameras![0].id);
  expect(changed.cameras![0].duration).toBe(12);
  expect(() => parseSequence(JSON.stringify(changed))).not.toThrow();
});

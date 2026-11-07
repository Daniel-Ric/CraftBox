import { Box3, Euler, Matrix4, Quaternion, Sphere, Vector3 } from "three";
import {
  sampleCloudLayer,
  type CloudLayer,
  type CloudOpening,
} from "./cloud-layers";
import type { AtmosphereSettings } from "./atmosphere-settings";
export function atmosphereAnimating(s: AtmosphereSettings) {
  return (
    s.animate &&
    ((s.windSpeed > 0 && (s.fog || s.clouds)) ||
      s.cloudLayers.some(
        (l) =>
          l.enabled &&
          ((l.windSpeed > 0 && l.density > 0 && l.coverage > 0) ||
            l.tracks.some((t) => !t.muted && t.keys.length > 1)),
      ))
  );
}
export function cloudRenderBudget(
  s: AtmosphereSettings,
  quality: string,
  exporting: boolean,
  interactive = false,
): { steps: number; resolution: number; shadowSteps: number } {
  if (interactive && !exporting) {
    const selected = cloudRenderBudget(s, quality, false);
    return {
      steps: Math.min(40, selected.steps),
      resolution: Math.min(0.5, selected.resolution),
      shadowSteps: Math.min(3, selected.shadowSteps),
    };
  }
  const q = exporting ? s.cloudExportQuality : s.cloudPreviewQuality;
  if (q === "low") return { steps: 40, resolution: 0.35, shadowSteps: 3 };
  if (q === "medium") return { steps: 72, resolution: 0.5, shadowSteps: 5 };
  if (q === "high") return { steps: 112, resolution: 1, shadowSteps: 6 };
  if (q === "ultra") return { steps: 192, resolution: 1, shadowSteps: 8 };
  return {
    steps: quality === "fine" ? 112 : quality === "fast" ? 40 : 72,
    resolution: quality === "fine" ? 1 : 0.5,
    shadowSteps: 6,
  };
}
export function prepareCloudLayers(
  layers: CloudLayer[],
  bounds: Box3,
  time: number,
) {
  const center = bounds.isEmpty()
      ? new Vector3()
      : bounds.getCenter(new Vector3()),
    radius = bounds.isEmpty()
      ? 10
      : Math.max(2, bounds.getBoundingSphere(new Sphere()).radius);
  return layers
    .filter((l) => l.enabled)
    .map((l) => sampleCloudLayer(l, time))
    .filter((l) => l.density > 0 && l.coverage > 0 && (l.dissolve ?? 0) < 1)
    .map((l) => {
      const position = new Vector3(...l.position)
          .multiplyScalar(radius)
          .add(center),
        size = new Vector3(...l.size).multiplyScalar(radius * 0.5);
      const rotation = new Quaternion().setFromEuler(
        new Euler(
          ...(l.rotation.map((v) => (v * Math.PI) / 180) as [
            number,
            number,
            number,
          ]),
        ),
      );
      const matrix = new Matrix4().compose(position, rotation, size);
      return {
        layer: l,
        worldToLocal: matrix.clone().invert(),
        bounds: new Box3(
          new Vector3(-1, -1, -1),
          new Vector3(1, 1, 1),
        ).applyMatrix4(matrix),
        height: l.size[1] * radius,
        radius,
      };
    });
}

export function cloudOpeningMask(p: [number, number, number], o: CloudOpening) {
  const smooth = (a: number, b: number, x: number) => {
    const t = Math.max(0, Math.min(1, (x - a) / (b - a)));
    return t * t * (3 - 2 * t);
  };
  const x = (p[0] - o.center[0]) / o.radius[0],
    z = (p[2] - o.center[1]) / o.radius[1];
  const angle = Math.atan2(z, x),
    warp =
      1 +
      o.irregularity *
        (Math.sin(angle * 3 + 0.7) * 0.12 + Math.sin(angle * 7 + 1.9) * 0.07);
  const inside =
    1 - smooth(1 - o.softness, 1 + o.softness, Math.hypot(x, z) / warp);
  return (
    1 -
    o.strength *
      inside *
      smooth(o.floor - o.softness, o.floor + o.softness, p[1])
  );
}

import type { OrthographicCamera, PerspectiveCamera } from "three";
import { Vector3, type Box3 } from "three";

export function fittedOrthoSpan(
  bounds: Box3,
  camera: OrthographicCamera,
  aspect: number,
  margin = 0.08,
  shift = 0,
) {
  if (bounds.isEmpty()) return 2;
  camera.updateMatrixWorld(true);
  const projected = bounds
    .clone()
    .applyMatrix4(camera.matrixWorldInverse)
    .getSize(new Vector3());
  return (
    (Math.max(2, projected.y, projected.x / aspect) * (1 + margin)) /
    (1 - 2 * Math.abs(shift))
  );
}

                                                                              
                                                       
export function spatialSamples(count: number): [number, number][] {
  const side = count === 16 ? 4 : count === 4 ? 2 : 1;
  return Array.from({ length: side * side }, (_, i) => [
    ((i % side) + 0.5) / side - 0.5,
    (Math.floor(i / side) + 0.5) / side - 0.5,
  ]);
}

export function withCameraJitter<T>(
  camera: OrthographicCamera | PerspectiveCamera,
  width: number,
  height: number,
  offset: [number, number],
  draw: () => T,
): T {
  const view = camera.view ? { ...camera.view } : null;
  try {
    if (view?.enabled) {
      camera.setViewOffset(
        view.fullWidth,
        view.fullHeight,
        view.offsetX + (offset[0] * view.width) / width,
        view.offsetY + (offset[1] * view.height) / height,
        view.width,
        view.height,
      );
    } else
      camera.setViewOffset(width, height, offset[0], offset[1], width, height);
    return draw();
  } finally {
    camera.view = view;
    camera.updateProjectionMatrix();
  }
}

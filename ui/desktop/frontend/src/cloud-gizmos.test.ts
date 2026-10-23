import { expect, it, vi } from "vitest";
import { Box3, Group, PerspectiveCamera, Vector3 } from "three";
import { CloudGizmos } from "./cloud-gizmos";
import { createCloudLayer } from "./cloud-layers";

it("places a cloud even when a previous transform handle remains hovered", () => {
  const camera = new PerspectiveCamera(45, 1, 1, 1000),
    canvas = document.createElement("canvas");
  camera.position.set(0, 0, 50);
  camera.updateMatrixWorld();
  canvas.getBoundingClientRect = () =>
    ({ left: 0, top: 0, width: 100, height: 100 }) as DOMRect;
  const gizmos = new CloudGizmos(
      camera,
      canvas,
      () => {},
      () => {},
    ),
    layer = createCloudLayer("volume"),
    onTransform = vi.fn();
  gizmos.transform.axis = "X";
  gizmos.update(
    [layer],
    new Box3(new Vector3(-10, -10, -10), new Vector3(10, 10, 10)),
    0,
    {
      selected: layer.id,
      placing: true,
      mode: "translate",
      onSelect: () => {},
      onTransform,
    },
    true,
    camera,
  );
  expect(
    gizmos.pick(
      { clientX: 50, clientY: 50 } as PointerEvent,
      camera,
      canvas,
      new Group(),
    ),
  ).toBe(true);
  expect(onTransform).toHaveBeenCalledOnce();
  gizmos.dispose();
});

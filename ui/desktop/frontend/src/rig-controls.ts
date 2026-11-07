import * as T from "three";
import { TransformControls } from "three/addons/controls/TransformControls.js";
import type { OrbitControls } from "three/addons/controls/OrbitControls.js";
import {
  limitedAxes,
  type RigPart,
  type createHumanRig,
} from "./human-skeleton";
import type { SkinPose } from "./asset-geometry";

export type RigEditorOptions = {
  selected: string;
  mode: "translate" | "rotate";
  space: "local" | "world";
  showSkeleton: boolean;
  onSelect: (id: string) => void;
  onPose: (pose: SkinPose) => void;
  onBegin: () => void;
  onEnd: () => void;
  onParts: (parts: RigPart[]) => void;
};

export function createRigControls(
  scene: T.Scene,
  camera: T.PerspectiveCamera,
  canvas: HTMLCanvasElement,
  orbit: OrbitControls,
  model: T.Group,
  rig: ReturnType<typeof createHumanRig>,
  get: () => { editor?: RigEditorOptions; pose: SkinPose },
) {
  const gizmo = new TransformControls(camera, canvas);
  gizmo.setSize(0.8);
  const helper = gizmo.getHelper();
  scene.add(helper);
  const proxy = new T.Object3D();
  scene.add(proxy);
  const overlay = new T.Group();
  overlay.name = "rigOverlay";
  scene.add(overlay);
  const sphere = new T.SphereGeometry(0.035, 10, 8),
    materials = new Map<string, T.MeshBasicMaterial>(),
    markers = new Map<string, T.Mesh>();
  for (const part of rig.parts.slice(0, 220)) {
    const color =
      part.kind === "target"
        ? "#47ce9b"
        : part.kind === "pole"
          ? "#ffc46e"
          : "#55a9ff";
    if (!materials.has(color))
      materials.set(
        color,
        new T.MeshBasicMaterial({
          color,
          depthTest: false,
          transparent: true,
          opacity: 0.9,
        }),
      );
    const marker = new T.Mesh(sphere, materials.get(color));
    marker.userData.rigPart = part.id;
    marker.renderOrder = 1000;
    overlay.add(marker);
    markers.set(part.id, marker);
  }
  const linePositions = new T.Float32BufferAttribute(
    new Float32Array(rig.parts.length * 6),
    3,
  );
  linePositions.setUsage(T.DynamicDrawUsage);
  const lineGeometry = new T.BufferGeometry();
  lineGeometry.setAttribute("position", linePositions);
  const lines = new T.LineSegments(
    lineGeometry,
    new T.LineBasicMaterial({
      color: 0x67a9ed,
      depthTest: false,
      transparent: true,
      opacity: 0.55,
    }),
  );
  lines.renderOrder = 999;
  overlay.add(lines);
  const ray = new T.Raycaster(),
    pointer = new T.Vector2();
  let down = [0, 0],
    dragging = false,
    previousSelected = "",
    reported = false;
  const drag = (event: { value: unknown }) => {
    dragging = !!event.value;
    orbit.enabled = !dragging;
    if (dragging) get().editor?.onBegin();
    else get().editor?.onEnd();
  };
  gizmo.addEventListener("dragging-changed", drag);
  const change = () => {
    proxy.updateMatrixWorld(true);
    const { editor, pose } = get();
    if (!editor || !dragging) return;
    const part = rig.parts.find((v) => v.id === editor.selected),
      object = rig.object(editor.selected);
    if (!part || !object || part.available === false) return;
    if (part.kind === "target" || part.kind === "pole") {
      const position = proxy.position.clone();
      if (model.parent) model.parent.worldToLocal(position);
      const field = part.kind === "target" ? "targets" : "poles";
      editor.onPose({
        ...pose,
        [field]: { ...pose[field], [part.id]: position.toArray() },
      });
      return;
    }
    object.parent?.updateWorldMatrix(true, false);
    const local = (
        object.parent?.matrixWorld.clone().invert() ?? new T.Matrix4()
      ).multiply(proxy.matrixWorld),
      position = new T.Vector3(),
      rotation = new T.Quaternion(),
      scale = new T.Vector3();
    local.decompose(position, rotation, scale);
    if (editor.mode === "translate") {
      const base =
          (object.userData.basePosition as T.Vector3) ?? new T.Vector3(),
        offset = position.sub(base);
      const max = pose.limits !== false && part.id !== "character" && part.kind!=="cosmetic" ? 2 : 128;
      offset.clampScalar(-max, max);
      editor.onPose({
        ...pose,
        positions: {
          ...pose.positions,
          [part.id]: offset.toArray() as [number, number, number],
        },
      });
    } else {
      const base = (object.userData.baseRotation as T.Euler) ?? new T.Euler(),
        angles = new T.Euler().setFromQuaternion(rotation, base.order);
      const axes = limitedAxes(
        part.id,
        [
          T.MathUtils.radToDeg(angles.x - base.x),
          T.MathUtils.radToDeg(angles.y - base.y),
          T.MathUtils.radToDeg(angles.z - base.z),
        ],
        pose.limits !== false,
      );
      const next = { ...pose, axes: { ...pose.axes, [part.id]: axes } };
      const legacy = Object.keys(pose).find(
        (key) =>
          key.toLowerCase() === part.id.toLowerCase() &&
          typeof pose[key as keyof SkinPose] === "number",
      );
      if (legacy) (next as unknown as Record<string, unknown>)[legacy] = 0;
      editor.onPose(next);
    }
  };
  gizmo.addEventListener("objectChange", change);
  const pointerDown = (e: PointerEvent) => {
    down = [e.clientX, e.clientY];
  };
  const pointerUp = (e: PointerEvent) => {
    const editor = get().editor;
    if (
      !editor ||
      e.button !== 0 ||
      dragging ||
      gizmo.axis ||
      Math.hypot(e.clientX - down[0], e.clientY - down[1]) > 4
    )
      return;
    const rect = canvas.getBoundingClientRect();
    pointer.set(
      ((e.clientX - rect.left) / rect.width) * 2 - 1,
      (-(e.clientY - rect.top) / rect.height) * 2 + 1,
    );
    ray.setFromCamera(pointer, camera);
    const handles = ray.intersectObjects(
      [...markers.values()].filter((v) => v.visible),
      false,
    );
    if (handles[0]) {
      editor.onSelect(handles[0].object.userData.rigPart);
      return;
    }
    const hits = ray.intersectObject(model, true);
    for (const hit of hits) {
      if (!(hit.object instanceof T.Mesh) || !hit.object.visible) continue;
      let node: T.Object3D | null = hit.object;
      while (node) {
        const match = rig.parts.find((p) => rig.object(p.id) === node);
        if (match) {
          editor.onSelect(match.id);
          return;
        }
        node = node.parent;
      }
    }
  };
  canvas.addEventListener("pointerdown", pointerDown);
  canvas.addEventListener("pointerup", pointerUp);
  function update() {
    const { editor, pose } = get();
    if (!editor) {
      overlay.visible = false;
      gizmo.detach();
      return;
    }
    if (!reported) {
      reported = true;
      editor.onParts(rig.parts);
    }
    overlay.visible = editor.showSkeleton;
    model.updateMatrixWorld(true);
    const selected = rig.parts.find((p) => p.id === editor.selected),
      object = rig.object(editor.selected);
    for (const part of rig.parts) {
      const marker = markers.get(part.id);
      if (!marker) continue;
      marker.position.copy(rig.worldPosition(part.id));
      marker.scale.setScalar(
        part.id === editor.selected ? 1.65 : part.kind === "target" ? 1.2 : 1,
      );
      marker.visible =
        part.available !== false &&
        (part.kind !== "pole" ||
          !!pose.targets?.[
            part.id.replace("ElbowPole", "Hand").replace("KneePole", "Foot")
          ] ||
          part.id === editor.selected);
    }
    const points: number[] = [];
    for (const part of rig.parts) {
      if (!part.parent || part.kind === "target" || part.kind === "pole")
        continue;
      const parent = rig.object(part.parent);
      if (!parent) continue;
      points.push(
        ...rig.worldPosition(part.id).toArray(),
        ...rig.worldPosition(part.parent).toArray(),
      );
    }
    linePositions.array.set(points);
    linePositions.needsUpdate = true;
    lines.geometry.setDrawRange(0, points.length / 3);
    lines.geometry.computeBoundingSphere();
    if (!object || !selected) {
      gizmo.detach();
      return;
    }
    const target = selected.kind === "target" || selected.kind === "pole";
    if (!dragging) {
      proxy.position.copy(rig.worldPosition(selected.id));
      proxy.quaternion.copy(
        target
          ? new T.Quaternion()
          : object.getWorldQuaternion(new T.Quaternion()),
      );
      proxy.scale.setScalar(1);
      proxy.updateMatrixWorld(true);
    }
    if (previousSelected !== selected.id) {
      previousSelected = selected.id;
      gizmo.attach(proxy);
    }
    gizmo.setMode(target ? "translate" : editor.mode);
    gizmo.setSpace(target ? "world" : editor.space);
  }
  return {
    update,
    dispose: () => {
      canvas.removeEventListener("pointerdown", pointerDown);
      canvas.removeEventListener("pointerup", pointerUp);
      gizmo.dispose();
      scene.remove(helper, proxy, overlay);
      sphere.dispose();
      for (const material of materials.values()) material.dispose();
      lines.geometry.dispose();
      (lines.material as T.Material).dispose();
      orbit.enabled = true;
    },
  };
}

import * as T from "three";
import { TransformControls } from "three/addons/controls/TransformControls.js";
import type { CloudEditorState } from "./CloudLayerControls";
import {
  sampleCloudLayer,
  unwrapCloudRotation,
  type CloudLayer,
  type CloudVector,
} from "./cloud-layers";
export class CloudGizmos {
  readonly group = new T.Group();
  readonly transform: TransformControls;
  private anchor = new T.Object3D();
  private boxes = new Map<string, T.Mesh<T.BoxGeometry, T.MeshBasicMaterial>>();
  private editor?: CloudEditorState;
  private center = new T.Vector3();
  private radius = 10;
  private rotationReference: CloudVector = [0, 0, 0];
  constructor(
    camera: T.Camera,
    canvas: HTMLCanvasElement,
    onDirty: () => void,
    onDrag: (value: boolean) => void,
  ) {
    this.transform = new TransformControls(camera, canvas);
    this.transform.setSize(0.75);
    this.group.add(this.anchor, this.transform.getHelper());
    this.transform.addEventListener("dragging-changed", (e) =>
      onDrag(Boolean(e.value)),
    );
    this.transform.addEventListener("objectChange", () => {
      onDirty();
      if (!this.editor?.selected || !this.transform.dragging) return;
      const vector = (v: T.Vector3, min: number, max: number) =>
        v.toArray().map((n) => T.MathUtils.clamp(n, min, max)) as CloudVector;
      if (this.editor.mode === "translate")
        this.editor.onTransform(this.editor.selected, {
          position: vector(
            this.anchor.position
              .clone()
              .sub(this.center)
              .divideScalar(this.radius),
            -100,
            100,
          ),
        });
      else if (this.editor.mode === "scale")
        this.editor.onTransform(this.editor.selected, {
          size: vector(
            this.anchor.scale.clone().divideScalar(this.radius),
            0.01,
            100,
          ),
        });
      else {
        this.rotationReference = unwrapCloudRotation(
          [
            this.anchor.rotation.x,
            this.anchor.rotation.y,
            this.anchor.rotation.z,
          ].map((n) => (n * 180) / Math.PI) as CloudVector,
          this.rotationReference,
        ).map((n) => T.MathUtils.clamp(n, -36000, 36000)) as CloudVector;
        this.editor.onTransform(this.editor.selected, {
          rotation: this.rotationReference,
        });
      }
    });
  }
  update(
    layers: CloudLayer[],
    bounds: T.Box3,
    time: number,
    editor: CloudEditorState | undefined,
    visible: boolean,
    camera: T.Camera,
  ) {
    this.editor = editor;
    this.group.visible = visible && !!editor?.selected;
    this.transform.enabled = this.group.visible && !editor?.placing;
    this.transform.camera = camera;
    if (!this.group.visible) {
      this.transform.detach();
      return;
    }
    this.center = bounds.isEmpty()
      ? new T.Vector3()
      : bounds.getCenter(this.center);
    this.radius = bounds.isEmpty()
      ? 10
      : Math.max(2, bounds.getBoundingSphere(new T.Sphere()).radius);
    for (const [id, box] of this.boxes)
      if (!layers.some((l) => l.id === id)) {
        box.geometry.dispose();
        box.material.dispose();
        box.removeFromParent();
        this.boxes.delete(id);
      }
    for (const source of layers) {
      const l = sampleCloudLayer(source, time);
      let box = this.boxes.get(l.id);
      if (!box) {
        box = new T.Mesh(
          new T.BoxGeometry(1, 1, 1),
          new T.MeshBasicMaterial({
            wireframe: true,
            transparent: true,
            depthTest: false,
            depthWrite: false,
          }),
        );
        box.renderOrder = 10000;
        box.userData.cloudId = l.id;
        this.boxes.set(l.id, box);
        this.group.add(box);
      }
      box.position
        .fromArray(l.position)
        .multiplyScalar(this.radius)
        .add(this.center);
      box.scale.fromArray(l.size).multiplyScalar(this.radius);
      box.rotation.set(
        ...(l.rotation.map((v) => (v * Math.PI) / 180) as [
          number,
          number,
          number,
        ]),
      );
      box.material.color.set(l.id === editor?.selected ? "#88caff" : "#a5bac6");
      box.material.opacity =
        l.id === editor?.selected ? (l.enabled ? 0.7 : 0.25) : 0.15;
      if (l.id === editor?.selected && !this.transform.dragging) {
        this.anchor.position.copy(box.position);
        this.anchor.quaternion.copy(box.quaternion);
        this.rotationReference = [...l.rotation];
        this.anchor.scale.copy(box.scale);
        this.transform.setMode(editor.mode);
        this.transform.setSpace(
          editor.mode === "translate" ? "world" : "local",
        );
        if (this.transform.object !== this.anchor)
          this.transform.attach(this.anchor);
      }
    }
    if (!layers.some((l) => l.id === editor?.selected)) this.transform.detach();
    this.group.updateMatrixWorld(true);
  }
  pick(
    event: PointerEvent,
    camera: T.Camera,
    canvas: HTMLCanvasElement,
    terrain: T.Object3D,
  ) {
    if (
      !this.group.visible ||
      !this.editor?.selected ||
      (!this.editor.placing && this.transform.axis)
    )
      return false;
    const rect = canvas.getBoundingClientRect(),
      ray = new T.Raycaster();
    ray.setFromCamera(
      new T.Vector2(
        ((event.clientX - rect.left) / rect.width) * 2 - 1,
        (-(event.clientY - rect.top) / rect.height) * 2 + 1,
      ),
      camera,
    );
    if (this.editor.placing) {
      const point =
        ray.intersectObject(terrain, true)[0]?.point ??
        ray.ray.intersectPlane(
          new T.Plane().setFromNormalAndCoplanarPoint(
            camera.getWorldDirection(new T.Vector3()),
            this.anchor.position,
          ),
          new T.Vector3(),
        );
      if (point)
        this.editor.onTransform(this.editor.selected, {
          position: point
            .clone()
            .sub(this.center)
            .divideScalar(this.radius)
            .toArray()
            .map((n) => T.MathUtils.clamp(n, -100, 100)) as CloudVector,
        });
      return true;
    }
    const hit = ray.intersectObjects([...this.boxes.values()])[0];
    if (hit) {
      this.editor.onSelect(hit.object.userData.cloudId);
      return true;
    }
    return false;
  }
  dispose() {
    this.transform.dispose();
    for (const box of this.boxes.values()) {
      box.geometry.dispose();
      box.material.dispose();
    }
    this.boxes.clear();
    this.group.removeFromParent();
  }
}

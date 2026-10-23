import { Camera, Euler, Vector3, Quaternion, Matrix4, MathUtils } from "three";
export function snapshotRoll(camera: Camera, target: Vector3) {
  const facing = new Quaternion().setFromRotationMatrix(
    new Matrix4().lookAt(camera.position, target, new Vector3(0, 1, 0)),
  );
  const relative = facing.invert().multiply(camera.quaternion);
  return (
    MathUtils.euclideanModulo(
      MathUtils.radToDeg(2 * Math.atan2(relative.z, relative.w)) + 180,
      360,
    ) - 180
  );
}


export class FreeCameraNavigation {
  private active = false;
  private keys = new Set<string>();
  private dragging: number | null = null;
  private previous = [0, 0];
  private direction = new Vector3();
  private euler = new Euler(0, 0, 0, "YXZ");
  private camera: Camera;
  private element: HTMLElement;
  private target: Vector3;
  private changed: () => void;
  private exit: () => void;
  constructor(
    camera: Camera,
    element: HTMLElement,
    target: Vector3,
    changed: () => void,
    exit: () => void,
  ) {
    this.camera = camera;
    this.element = element;
    this.target = target;
    this.changed = changed;
    this.exit = exit;
    element.addEventListener("keydown", this.keydown);
    element.addEventListener("keyup", this.keyup);
    element.addEventListener("blur", this.clear);
    window.addEventListener("blur", this.clear);
    element.addEventListener("pointerdown", this.down);
    element.addEventListener("pointermove", this.move);
    element.addEventListener("pointerup", this.up);
    element.addEventListener("pointercancel", this.up);
    element.addEventListener("contextmenu", this.context);
  }
  get enabled() {
    return this.active;
  }
  set enabled(value: boolean) {
    if (this.active !== value) this.clear();
    this.active = value;
  }
  private clear = () => {
    this.keys.clear();
    if (
      this.dragging !== null &&
      this.element.hasPointerCapture?.(this.dragging)
    )
      this.element.releasePointerCapture(this.dragging);
    this.dragging = null;
  };
  private keydown = (e: KeyboardEvent) => {
    if (!this.active || e.ctrlKey || e.metaKey) return;
    if (e.code === "Escape") {
      this.clear();
      this.exit();
      e.stopPropagation();
      return;
    }
    if (
      [
        "KeyW",
        "KeyA",
        "KeyS",
        "KeyD",
        "KeyQ",
        "KeyE",
        "ShiftLeft",
        "ShiftRight",
        "AltLeft",
        "AltRight",
      ].includes(e.code)
    ) {
      this.keys.add(e.code);
      e.preventDefault();
      e.stopPropagation();
    }
  };
  private keyup = (e: KeyboardEvent) => {
    this.keys.delete(e.code);
  };
  private down = (e: PointerEvent) => {
    if (!this.active) return;
    this.element.focus();
    if (e.button !== 2) return;
    e.preventDefault();
    this.dragging = e.pointerId;
    this.previous = [e.clientX, e.clientY];
    this.element.setPointerCapture(e.pointerId);
  };
  private up = (e: PointerEvent) => {
    if (e.pointerId === this.dragging) this.clear();
  };
  private context = (e: Event) => {
    if (this.active) e.preventDefault();
  };
  private move = (e: PointerEvent) => {
    if (!this.active || this.dragging !== e.pointerId) return;
    const dx = e.clientX - this.previous[0],
      dy = e.clientY - this.previous[1];
    this.previous = [e.clientX, e.clientY];
    const distance = Math.max(
      0.1,
      this.camera.position.distanceTo(this.target),
    );
    this.euler.setFromQuaternion(this.camera.quaternion, "YXZ");
    this.euler.y -= dx * 0.003;
    this.euler.x = Math.max(
      -Math.PI * 0.495,
      Math.min(Math.PI * 0.495, this.euler.x - dy * 0.003),
    );
    this.euler.z = 0;
    this.camera.quaternion.setFromEuler(this.euler);
    this.camera.getWorldDirection(this.direction);
    this.target
      .copy(this.camera.position)
      .addScaledVector(this.direction, distance);
    this.changed();
  };
  update(seconds: number, speed: number) {
    if (
      !this.active ||
      document.activeElement !== this.element ||
      !this.keys.size
    )
      return;
    const axis = (positive: string, negative: string) =>
      Number(this.keys.has(positive)) - Number(this.keys.has(negative));
    const forward = this.camera.getWorldDirection(new Vector3()),
      right = new Vector3()
        .crossVectors(forward, new Vector3(0, 1, 0))
        .normalize();
    const movement = forward
      .multiplyScalar(axis("KeyW", "KeyS"))
      .addScaledVector(right, axis("KeyD", "KeyA"))
      .add(new Vector3(0, axis("KeyE", "KeyQ"), 0));
    if (movement.lengthSq() === 0) return;
    const multiplier =
      this.keys.has("ShiftLeft") || this.keys.has("ShiftRight")
        ? 5
        : this.keys.has("AltLeft") || this.keys.has("AltRight")
          ? 0.2
          : 1;
    movement.normalize().multiplyScalar(seconds * speed * multiplier);
    this.camera.position.add(movement);
    this.target.add(movement);
    this.changed();
  }
  dispose() {
    this.clear();
    this.element.removeEventListener("keydown", this.keydown);
    this.element.removeEventListener("keyup", this.keyup);
    this.element.removeEventListener("blur", this.clear);
    window.removeEventListener("blur", this.clear);
    this.element.removeEventListener("pointerdown", this.down);
    this.element.removeEventListener("pointermove", this.move);
    this.element.removeEventListener("pointerup", this.up);
    this.element.removeEventListener("pointercancel", this.up);
    this.element.removeEventListener("contextmenu", this.context);
  }
}

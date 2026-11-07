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
  levelFlight = false;
  verticalShortcuts = false;
  sensitivity = 0.003;
  smoothing = 0;
  boost = 5;
  private velocity = new Vector3();
  private pendingLook = [0, 0];
  onSpeedScale?: (scale: number) => void;
  private wasLocked = false;
  private menuRelease = false;
  setCamera(camera: Camera) {
    this.camera = camera;
    this.clear();
  }
  async lockPointer() {
    if (!this.active || !this.element.requestPointerLock) return false;
    this.element.focus();
    try {
      await this.element.requestPointerLock();
      return document.pointerLockElement === this.element;
    } catch {
      return false;
    }
  }
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
    element.addEventListener("wheel", this.wheel, { passive: false });
    document.addEventListener("mousemove", this.lockedMove);
    document.addEventListener("pointerlockchange", this.lockChange);
    document.addEventListener("visibilitychange", this.visibility);
  }
  get enabled() {
    return this.active;
  }
  set enabled(value: boolean) {
    if (this.active !== value) this.clear();
    this.active = value;
    if (!value && document.pointerLockElement === this.element)
      document.exitPointerLock();
  }
  private clear = () => {
    this.keys.clear();
    this.velocity.set(0, 0, 0);
    this.pendingLook = [0, 0];
    this.releaseDrag();
  };
  stop() {
    this.clear();
  }
  private releaseDrag = () => {
    if (
      this.dragging !== null &&
      this.element.hasPointerCapture?.(this.dragging)
    )
      this.element.releasePointerCapture(this.dragging);
    this.dragging = null;
  };
  private keydown = (e: KeyboardEvent) => {
    if (!this.active || e.ctrlKey || e.metaKey) return;
    if (e.code === "Tab") {
      if (document.pointerLockElement === this.element) {
        this.clear();
        this.menuRelease = true;
        document.exitPointerLock();
      } else void this.lockPointer();
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    if (!this.verticalShortcuts && (e.code === "Space" || e.code === "KeyC"))
      return;
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
        "Space",
        "KeyC",
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
    if (e.pointerId === this.dragging) this.releaseDrag();
  };
  private context = (e: Event) => {
    if (this.active) e.preventDefault();
  };
  private move = (e: PointerEvent) => {
    if (!this.active || this.dragging !== e.pointerId) return;
    const dx = e.clientX - this.previous[0],
      dy = e.clientY - this.previous[1];
    this.previous = [e.clientX, e.clientY];
    this.look(dx, dy);
  };
  private wheel = (e: WheelEvent) => {
    if (!this.active || document.activeElement !== this.element) return;
    e.preventDefault();
    this.onSpeedScale?.(e.deltaY < 0 ? 1.25 : 0.8);
  };
  private lockedMove = (e: MouseEvent) => {
    if (this.active && document.pointerLockElement === this.element)
      this.look(e.movementX, e.movementY);
  };
  private lockChange = () => {
    const locked = document.pointerLockElement === this.element;
    if (this.wasLocked && !locked) {
      this.clear();
      if (this.active && !this.menuRelease) this.exit();
      this.menuRelease = false;
    }
    this.wasLocked = locked;
  };
  private visibility = () => {
    if (document.hidden) this.clear();
  };
  private look(dx: number, dy: number) {
    if (this.smoothing > 0) {
      this.pendingLook[0] += dx;
      this.pendingLook[1] += dy;
      return;
    }
    this.applyLook(dx, dy);
  }
  private applyLook(dx: number, dy: number) {
    const distance = Math.max(
      0.1,
      this.camera.position.distanceTo(this.target),
    );
    this.euler.setFromQuaternion(this.camera.quaternion, "YXZ");
    this.euler.y -= dx * this.sensitivity;
    this.euler.x = Math.max(
      -Math.PI * 0.495,
      Math.min(Math.PI * 0.495, this.euler.x - dy * this.sensitivity),
    );
    this.euler.z = 0;
    this.camera.quaternion.setFromEuler(this.euler);
    this.camera.getWorldDirection(this.direction);
    this.target
      .copy(this.camera.position)
      .addScaledVector(this.direction, distance);
    this.changed();
  }
  update(seconds: number, speed: number) {
    if (!this.active || document.activeElement !== this.element) return;
    if (this.smoothing > 0) {
      seconds = Math.max(0, Math.min(0.1, seconds));
      const fraction =
        1 - Math.exp(-seconds / Math.max(0.001, this.smoothing * 0.35));
      const dx = this.pendingLook[0] * fraction,
        dy = this.pendingLook[1] * fraction;
      this.pendingLook[0] -= dx;
      this.pendingLook[1] -= dy;
      if (Math.abs(dx) + Math.abs(dy) > 0.0001) this.applyLook(dx, dy);
    }
    const axis = (positive: string, negative: string) =>
      Number(this.keys.has(positive)) - Number(this.keys.has(negative));
    const forward = this.camera.getWorldDirection(new Vector3()),
      right = new Vector3()
        .crossVectors(forward, new Vector3(0, 1, 0))
        .normalize();
    if (this.levelFlight) {
      forward.y = 0;
      forward.normalize();
    }
    const movement = forward
      .multiplyScalar(axis("KeyW", "KeyS"))
      .addScaledVector(right, axis("KeyD", "KeyA"))
      .add(
        new Vector3(
          0,
          Number(
            this.keys.has("KeyE") ||
              (this.verticalShortcuts && this.keys.has("Space")),
          ) -
            Number(
              this.keys.has("KeyQ") ||
                (this.verticalShortcuts && this.keys.has("KeyC")),
            ),
          0,
        ),
      );
    const multiplier =
      this.keys.has("ShiftLeft") || this.keys.has("ShiftRight")
        ? this.boost
        : this.keys.has("AltLeft") || this.keys.has("AltRight")
          ? 0.2
          : 1;
    movement.normalize().multiplyScalar(speed * multiplier);
    if (this.smoothing > 0) {
      const tau = this.keys.size ? this.smoothing : this.smoothing * 0.5;
      const fraction = 1 - Math.exp(-seconds / Math.max(0.001, tau));
      const displacement = movement
        .clone()
        .multiplyScalar(seconds)
        .addScaledVector(this.velocity.clone().sub(movement), tau * fraction);
      this.velocity.lerp(movement, fraction);
      movement.copy(displacement);
    } else movement.multiplyScalar(seconds);
    if (movement.lengthSq() < 1e-12) return;
    this.camera.position.add(movement);
    this.target.add(movement);
    this.changed();
  }
  dispose() {
    this.enabled = false;
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
    this.element.removeEventListener("wheel", this.wheel);
    document.removeEventListener("mousemove", this.lockedMove);
    document.removeEventListener("pointerlockchange", this.lockChange);
    document.removeEventListener("visibilitychange", this.visibility);
  }
}

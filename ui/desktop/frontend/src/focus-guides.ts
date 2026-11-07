import { Camera, Vector3 } from "three";
import type { RenderSettings } from "./render-settings";
import "./focus-guides.css";

export function projectFocusPoint(
  point: Vector3,
  camera: Camera,
  width: number,
  height: number,
) {
  camera.updateMatrixWorld(true);
  const view = point.clone().applyMatrix4(camera.matrixWorldInverse);
  if (view.z >= 0) return undefined;
  const p = point.clone().project(camera);
  if (![p.x, p.y, p.z].every(Number.isFinite) || p.z < -1 || p.z > 1)
    return undefined;
  return {
    x: (p.x * 0.5 + 0.5) * width,
    y: (0.5 - p.y * 0.5) * height,
    depth: -view.z,
  };
}
export function focusSpherePaths(
  point: Vector3,
  radius: number,
  camera: Camera,
  width: number,
  height: number,
) {
  if (radius <= 0 || !Number.isFinite(radius)) return [];
  return [0, 1, 2].map((axis) => {
    let path = "",
      open = false;
    for (let i = 0; i <= 64; i++) {
      const a = (i / 64) * Math.PI * 2,
        p = point.clone();
      p.setComponent(
        (axis + 1) % 3,
        p.getComponent((axis + 1) % 3) + Math.cos(a) * radius,
      );
      p.setComponent(
        (axis + 2) % 3,
        p.getComponent((axis + 2) % 3) + Math.sin(a) * radius,
      );
      const screen = projectFocusPoint(p, camera, width, height);
      if (!screen) {
        open = false;
        continue;
      }
      path += `${open ? "L" : "M"}${screen.x.toFixed(1)},${screen.y.toFixed(1)} `;
      open = true;
    }
    return path;
  });
}

                                                                                        
export class FocusGuides {
  readonly element = document.createElement("div");
  private svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  private status = document.createElement("div");
  private signature = "";
  private host: HTMLElement;
  private canvas: HTMLCanvasElement;
  constructor(host: HTMLElement, canvas: HTMLCanvasElement) {
    this.host = host;
    this.canvas = canvas;
    if (
      !getComputedStyle(host).position ||
      getComputedStyle(host).position === "static"
    )
      host.style.position = "relative";
    this.element.className = "focus-guides";
    this.element.hidden = true;
    this.svg.setAttribute("aria-label", "Focus target and sharp region");
    this.svg.setAttribute("role", "img");
    this.status.className = "focus-guide-status";
    this.status.setAttribute("role", "status");
    this.status.setAttribute("aria-live", "off");
    this.element.append(this.svg, this.status);
    host.append(this.element);
  }
  update(
    camera: Camera,
    settings: RenderSettings,
    target: Vector3 | undefined,
    hover: Vector3 | undefined,
    picking: boolean,
    visible: boolean,
  ) {
    const enabled =
      visible && (picking || (settings.dofEnabled && Boolean(target)));
    this.element.hidden = !enabled;
    if (!enabled) return;
    const rect = this.canvas.getBoundingClientRect(),
      host = this.host.getBoundingClientRect(),
      width = rect.width,
      height = rect.height;
    this.element.style.left = `${rect.left - host.left}px`;
    this.element.style.top = `${rect.top - host.top}px`;
    this.element.style.width = `${width}px`;
    this.element.style.height = `${height}px`;
    this.svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
    const point = picking ? hover : target;
    const screen = point && projectFocusPoint(point, camera, width, height);
    const onScreen =
      screen &&
      screen.x >= 0 &&
      screen.x <= width &&
      screen.y >= 0 &&
      screen.y <= height;
    let markup = "";
    if (screen) {
      const color = picking ? "#ffd166" : "#52e0cb",
        x = screen.x,
        y = screen.y;
      if (settings.dofMode === "subject" && point) {
        const paths = focusSpherePaths(
          point,
          settings.dofFocusRange,
          camera,
          width,
          height,
        );
        for (const path of paths)
          markup += `<path d="${path}" stroke="${color}" class="focus-sphere"/>`;
      }
      if (onScreen) {
        markup += `<g transform="translate(${x.toFixed(1)},${y.toFixed(1)})" class="focus-reticle" stroke="${color}"><circle r="12"/><circle r="4"/><path d="M-20,0H-14M14,0H20M0,-20V-14M0,14V20"/></g>`;
        markup += `<text x="${Math.min(width - 95, Math.max(5, x + 20)).toFixed(1)}" y="${Math.max(18, y - 17).toFixed(1)}" fill="${color}" class="focus-guide-label">${picking ? "Preview" : "Focus locked"}</text>`;
      }
    }
    if (markup !== this.signature) {
      this.svg.innerHTML = markup;
      this.signature = markup;
    }
    const info = picking
      ? onScreen
        ? "Click to focus · Esc cancel"
        : "Select a visible surface · Esc cancel"
      : onScreen
        ? `${settings.dofMode === "subject" ? `Sharp radius ${settings.dofFocusRange.toFixed(1)} b · ` : ""}Focus ${screen.depth.toFixed(1)} b`
        : "Focus target outside view";
    if (this.status.textContent !== info) this.status.textContent = info;
  }
  dispose() {
    this.element.remove();
  }
}

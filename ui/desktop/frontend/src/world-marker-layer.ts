import * as THREE from "three";
import { markerBlockCenter, type MapMarker } from "./map-presentation";

function markerLabel(marker: MapMarker) {
  const text = [marker.icon, marker.label].filter(Boolean).join(" ");
  if (!text) return;
  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  const font = '600 28px "Segoe UI", sans-serif';
  ctx.font = font;
  canvas.width = Math.ceil(Math.min(ctx.measureText(text).width, 680) + 44);
  canvas.height = 52;
  ctx.beginPath();
  ctx.roundRect(1, 1, canvas.width - 2, canvas.height - 2, 11);
  ctx.fillStyle = marker.background;
  ctx.fill();
  ctx.lineWidth = 2;
  ctx.strokeStyle = marker.color;
  ctx.stroke();
  ctx.font = font;
  ctx.textBaseline = "middle";
  ctx.fillStyle = "#fff";
  ctx.fillText(text, 22, 26, canvas.width - 44);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  const label = new THREE.Sprite(
    new THREE.SpriteMaterial({
      map: texture,
      depthWrite: false,
      toneMapped: false,
    }),
  );
  const aspect = canvas.width / canvas.height;
  label.center.set(-0.65 / aspect, 0.5);
  label.position.y = 1.72;
  label.scale.set(aspect, 1, 1);
  return label;
}

export class WorldMarkerLayer {
  readonly group = new THREE.Group();
  private readonly stem = new THREE.CylinderGeometry(
    0.085,
    0.085,
    1.65,
    8,
  ).translate(0, 0.825, 0);
  private readonly head = new THREE.OctahedronGeometry(0.53)
    .scale(0.85, 1.12, 0.85)
    .translate(0, 1.72, 0);
  private readonly ray = new THREE.Raycaster(
    undefined,
    new THREE.Vector3(0, -1, 0),
  );
  private readonly bounds = new THREE.Box3();
  private readonly markersByChunk = new Map<string, MapMarker[]>();
  private readonly pinsByChunk = new Map<string, THREE.Group[]>();
  private signature = "";

  setMarkers(
    markers: MapMarker[],
    terrain: ReadonlyMap<string, THREE.Group>,
    origin?: THREE.Vector3,
  ) {
    const signature = JSON.stringify(markers);
    if (signature === this.signature) return;
    this.signature = signature;
    this.clearTerrain();
    this.markersByChunk.clear();
    for (const marker of markers) {
      if (!marker || !Number.isFinite(marker.x) || !Number.isFinite(marker.z))
        continue;
      const key = `${Math.floor(marker.x / 16)},${Math.floor(marker.z / 16)}`;
      let list = this.markersByChunk.get(key);
      if (!list) this.markersByChunk.set(key, (list = []));
      list.push(marker);
    }
    if (!origin) return;
    for (const key of this.markersByChunk.keys()) {
      const chunk = terrain.get(key);
      if (chunk) this.updateChunk(key, chunk, origin);
    }
  }

  updateChunk(key: string, terrain: THREE.Group, origin: THREE.Vector3) {
    const markers = this.markersByChunk.get(key);
    if (!markers) return;
    this.clearChunk(key);
    terrain.updateWorldMatrix(true, true);
    this.bounds.setFromObject(terrain);
    if (this.bounds.isEmpty()) return;
    const pins: THREE.Group[] = [];
    const heights = new Map<string, number>();
    for (const marker of markers) {
      const x = markerBlockCenter(marker.x) - origin.x;
      const z = markerBlockCenter(marker.z) - origin.z;
      const column = `${x},${z}`;
      let y = heights.get(column);
      if (y === undefined) {
        this.ray.ray.origin.set(x, this.bounds.max.y + 1, z);
        y =
          this.ray.intersectObjects(terrain.children, false)[0]?.point.y ??
          this.bounds.max.y;
        heights.set(column, y);
      }
      const pin = new THREE.Group();
      pin.position.set(x, y, z);
      pin.userData.markerScale = THREE.MathUtils.clamp(
        marker.scale || 1,
        0.25,
        4,
      );
      const material = new THREE.MeshLambertMaterial({
        color: marker.color,
        emissive: marker.color,
        emissiveIntensity: 0.2,
      });
      pin.add(
        new THREE.Mesh(this.stem, material),
        new THREE.Mesh(this.head, material),
      );
      const label = markerLabel(marker);
      if (label) pin.add(label);
      this.group.add(pin);
      pins.push(pin);
    }
    this.pinsByChunk.set(key, pins);
  }

  updateScale(camera: THREE.PerspectiveCamera, viewportHeight: number) {
    const unitsPerPixel =
      (2 * Math.tan(THREE.MathUtils.degToRad(camera.fov) / 2)) /
      Math.max(1, viewportHeight);
    for (const pin of this.group.children) {
      const size =
        camera.position.distanceTo(pin.position) * unitsPerPixel * 24;
      pin.scale.setScalar(
        THREE.MathUtils.clamp(size, 0.4, 24) * pin.userData.markerScale,
      );
    }
  }

  clearTerrain() {
    for (const key of this.pinsByChunk.keys()) this.clearChunk(key);
  }

  dispose() {
    this.clearTerrain();
    this.stem.dispose();
    this.head.dispose();
    this.group.removeFromParent();
  }

  private clearChunk(key: string) {
    for (const pin of this.pinsByChunk.get(key) ?? []) {
      const materials = new Set<THREE.Material>();
      for (const part of pin.children) {
        if (part instanceof THREE.Mesh) materials.add(part.material);
        if (part instanceof THREE.Sprite) {
          part.material.map?.dispose();
          materials.add(part.material);
        }
      }
      for (const material of materials) material.dispose();
      pin.removeFromParent();
    }
    this.pinsByChunk.delete(key);
  }
}

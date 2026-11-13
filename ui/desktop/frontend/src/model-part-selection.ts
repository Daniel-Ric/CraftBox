import * as T from "three";
import { elementKey, modelParts, type ModelElement, type ModelPart, type PartMode } from "./model-parts";
import { visibleSelectionBounds } from "./asset-camera";
import { nameModelParts } from "./model-part-names";

export type ModelSelectionOptions = {
  enabled: boolean;
  mode: PartMode;
  selected: string[];
  isolate?: boolean;
  animations?: Record<string, unknown>;
  name?: string;
  onParts: (parts: ModelPart[]) => void;
  onSelect: (ids: string[]) => void;
};

const imagePixels = new WeakMap<object, ImageData | null>();
function hitVisible(hit: T.Intersection<T.Object3D>) {
  let node: T.Object3D | null = hit.object;
  while (node) { if (!node.visible) return false; node = node.parent; }
  const mesh = hit.object as T.Mesh;
  const material = (Array.isArray(mesh.material) ? mesh.material[hit.face?.materialIndex ?? 0] : mesh.material) as T.MeshStandardMaterial;
  if (!material?.visible || material.opacity <= 0) return false;
  const image = material.map?.image as HTMLImageElement | undefined;
  if (!image || !hit.uv || (!material.alphaTest && !material.transparent)) return true;
  if (!imagePixels.has(image)) {
    let pixels: ImageData | null = null;
    try {
      const width = image.naturalWidth || image.width, height = image.naturalHeight || image.height;
      if (width * height <= 16_777_216) {
        const canvas = document.createElement("canvas");
        canvas.width = width; canvas.height = height;
        const context = canvas.getContext("2d", { willReadFrequently: true });
        if (context) { context.drawImage(image, 0, 0); pixels = context.getImageData(0, 0, width, height); }
      }
    } catch { pixels = null; }
    imagePixels.set(image, pixels);
  }
  const pixels = imagePixels.get(image);
  if (!pixels) return true;
  const uv = material.map!.transformUv(hit.uv.clone());
  const x = Math.max(0, Math.min(pixels.width - 1, Math.floor(uv.x * pixels.width)));
  const y = Math.max(0, Math.min(pixels.height - 1, Math.floor(uv.y * pixels.height)));
  return pixels.data[(y * pixels.width + x) * 4 + 3] / 255 >= Math.max(material.alphaTest, 0.01);
}

export function createModelPartSelection(
  root: T.Group,
  camera: T.Camera,
  canvas: HTMLCanvasElement,
  options: () => ModelSelectionOptions | undefined,
  invalidate: () => void,
) {
  const meshes: T.Mesh[] = [];
  root.traverse(node => { if (node instanceof T.Mesh && node.userData.sourceElement && !node.userData.previewOnly) meshes.push(node); });
  let restTransforms: Map<T.Object3D, T.Matrix4> | undefined;
  let ready = false;
  const catalogs = new Map<PartMode, ModelPart[]>();
  const raycaster = new T.Raycaster(), pointer = new T.Vector2();
  let mode: PartMode | undefined, parts: ModelPart[] = [], index = new Map<string, string>();
  let pointerDown: { x: number; y: number; id: number; dragged: boolean } | undefined;
  let hover: string | undefined, paintedHover: string | undefined;
  let selectedState: string[] | undefined, selectedEnabled = false;
  let lastIsolate = false;
  const overlays = new Map<T.Mesh, { node: T.Mesh; materials: T.ShaderMaterial[] }>();
  const selectedMeshes = new Set<T.Mesh>();
  const reducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
  const sync = () => {
    const current = options();
    if (!current?.enabled || current.mode === mode) return;
    if (!restTransforms) {
      root.updateWorldMatrix(true, true);
      restTransforms = new Map();
      root.traverse(node => restTransforms!.set(node, node.matrixWorld.clone()));
    }
    mode = current.mode;
    parts = catalogs.get(mode) ?? nameModelParts(modelParts(root, mode, restTransforms), current.animations, current.name);
    catalogs.set(mode, parts);
    index = new Map();
    for (const part of parts) for (const element of part.elements) {
      const key = elementKey(element);
      if (!index.has(key) || mode === "bones") index.set(key, part.id);
    }
    hover = undefined;
    selectedState = undefined;
    current.onParts(parts);
  };
  const pick = (event: PointerEvent) => {
    if (!ready) return undefined;
    sync();
    const rect = canvas.getBoundingClientRect();
    if (!rect.width || !rect.height) return undefined;
    pointer.set((event.clientX - rect.left) / rect.width * 2 - 1, -(event.clientY - rect.top) / rect.height * 2 + 1);
    root.updateMatrixWorld(true);
    camera.updateMatrixWorld(true);
    raycaster.setFromCamera(pointer, camera);
    const hit = raycaster.intersectObjects(meshes, false).find(hitVisible);
    return hit ? index.get(elementKey(hit.object.userData.sourceElement as ModelElement)) : undefined;
  };
  const down = (event: PointerEvent) => {
    if (!options()?.enabled || event.button !== 0 || !event.isPrimary) return;
    pointerDown = { x: event.clientX, y: event.clientY, id: event.pointerId, dragged: false };
    canvas.focus({ preventScroll: true });
  };
  const up = (event: PointerEvent) => {
    const start = pointerDown; pointerDown = undefined;
    const current = options();
    if (!current?.enabled || !start || start.dragged || event.pointerId !== start.id || event.button !== 0 || Math.hypot(event.clientX - start.x, event.clientY - start.y) > 5) return;
    const id = pick(event);
    if (!id) return;
    const selected = new Set(current.selected);
    if (selected.has(id)) selected.delete(id); else selected.add(id);
    current.onSelect([...selected]);
    invalidate();
  };
  const move = (event: PointerEvent) => {
    if (pointerDown && event.pointerId === pointerDown.id && Math.hypot(event.clientX - pointerDown.x, event.clientY - pointerDown.y) > 5) pointerDown.dragged = true;
    if (!options()?.enabled || event.buttons) return;
    const next = pick(event);
    if (hover !== next) { hover = next; invalidate(); }
    canvas.style.cursor = next ? "pointer" : "grab";
  };
  const leave = () => { hover = undefined; canvas.style.cursor = ""; invalidate(); };
  const cancel = () => { pointerDown = undefined; leave(); };
  const key = (event: KeyboardEvent) => {
    const current = options();
    if (!current?.enabled) return;
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "a") {
      event.preventDefault(); event.stopPropagation(); current.onSelect(parts.map(part => part.id)); invalidate();
    } else if (event.key === "Escape" && current.selected.length) {
      event.preventDefault(); event.stopPropagation(); current.onSelect([]); invalidate();
    }
  };
  const overlay = (mesh: T.Mesh) => {
    const sources = (Array.isArray(mesh.material) ? mesh.material : [mesh.material]) as T.MeshStandardMaterial[];
    const materials = sources.map(source => new T.ShaderMaterial({
      uniforms: { color: { value: new T.Color(0x58c7ff) }, opacity: { value: 0.8 }, resolution: { value: new T.Vector2(1, 1) }, map: { value: source.map ?? null }, hasMap: { value: !!source.map }, mapTransform: { value: source.map?.matrix ?? new T.Matrix3() }, alphaCutoff: { value: source.alphaTest || 0.01 } },
      vertexShader: "uniform vec2 resolution; uniform mat3 mapTransform; varying vec2 vUv; void main(){vUv=(mapTransform*vec3(uv,1.0)).xy;vec4 p=projectionMatrix*modelViewMatrix*vec4(position,1.0);vec3 n=normalMatrix*normal;vec2 direction=(projectionMatrix*vec4(n,0.0)).xy;float len=length(direction);if(len>0.00001)p.xy+=direction/len*1.6*2.0/resolution*p.w;gl_Position=p;}",
      fragmentShader: "uniform vec3 color; uniform float opacity; uniform sampler2D map; uniform bool hasMap; uniform float alphaCutoff; varying vec2 vUv; void main(){if(hasMap&&texture2D(map,vUv).a<alphaCutoff)discard;gl_FragColor=vec4(color,opacity);}",
      side: T.BackSide, transparent: true, depthWrite: false, toneMapped: false,
    }));
    const node = new T.Mesh(mesh.geometry, Array.isArray(mesh.material) ? materials : materials[0]);
    node.userData.previewOnly = true;
    node.raycast = () => {};
    node.renderOrder = 20;
    mesh.add(node);
    return { node, materials };
  };
  canvas.tabIndex = 0;
  canvas.addEventListener("pointerdown", down);
  canvas.addEventListener("pointerup", up);
  canvas.addEventListener("pointermove", move);
  canvas.addEventListener("pointerleave", leave);
  canvas.addEventListener("pointercancel", cancel);
  canvas.addEventListener("keydown", key);
  return {
    update(time: number) {
      ready = true;
      sync();
      const current = options();
      const refit = lastIsolate !== !!current?.isolate || (!!current?.isolate && selectedState !== current?.selected);
      lastIsolate = !!current?.isolate;
      if (selectedState !== current?.selected || selectedEnabled !== !!current?.enabled || paintedHover !== hover) {
        selectedState = current?.selected;
        selectedEnabled = !!current?.enabled;
        paintedHover = hover;
        const selected = new Set(current?.selected);
        const elements = new Set(parts.filter(part => selected.has(part.id) || part.id === hover).flatMap(part => part.elements.map(elementKey)));
        const selectedElements = new Set(parts.filter(part => selected.has(part.id)).flatMap(part => part.elements.map(elementKey)));
        selectedMeshes.clear();
        for (const mesh of meshes) {
          if (selectedElements.has(elementKey(mesh.userData.sourceElement))) selectedMeshes.add(mesh);
          const show = !!current?.enabled && elements.has(elementKey(mesh.userData.sourceElement));
          if (show && !overlays.has(mesh)) overlays.set(mesh, overlay(mesh));
          const item = overlays.get(mesh);
          if (item && !show) {
            item.node.removeFromParent();
            item.materials.forEach(material => material.dispose());
            overlays.delete(mesh);
          }
        }
      }
      if (current?.isolate && selectedMeshes.size)
        for (const mesh of meshes) if (!selectedMeshes.has(mesh)) mesh.visible = false;
      const opacity = reducedMotion ? 0.7 : 0.62 + Math.sin(time * 3.5) * 0.14;
      for (const [mesh, item] of overlays) for (const material of item.materials) {
        material.uniforms.opacity.value = selectedMeshes.has(mesh) ? opacity : 0.25;
        material.uniforms.resolution.value.set(Math.max(1, canvas.clientWidth), Math.max(1, canvas.clientHeight));
      }
      return refit;
    },
    bounds() {
      root.updateWorldMatrix(true, true);
      return visibleSelectionBounds(selectedMeshes);
    },
    continuous: () => !reducedMotion && !!options()?.enabled && !!options()?.selected.length,
    dispose() {
      canvas.removeEventListener("pointerdown", down); canvas.removeEventListener("pointerup", up);
      canvas.removeEventListener("pointermove", move); canvas.removeEventListener("pointerleave", leave);
      canvas.removeEventListener("pointercancel", cancel); canvas.removeEventListener("keydown", key);
      canvas.style.cursor = "";
      for (const { node, materials } of overlays.values()) { node.removeFromParent(); materials.forEach(material => material.dispose()); }
      overlays.clear();
    },
  };
}

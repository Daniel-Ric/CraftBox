import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { TransformControls } from "three/examples/jsm/controls/TransformControls.js";
import { SceneDecoder } from "./scene-decoder";
import { createPreviewLoop } from "./preview-loop";
import { useDocumentScope } from "./document-scope";
import EditorLoadingOverlay from "./EditorLoadingOverlay";
import {
  structures,
  normalized,
  type Box,
  type Vec3,
  type StructureDocument,
  type Difference,
} from "./structure-api";

type Props = {
  doc: StructureDocument;
  selection: Box;
  corner: 0 | 1;
  layer: number;
  layerOnly: boolean;
  adjacent: boolean;
  top: boolean;
  ghost?: { doc: StructureDocument; anchor: Vec3 };
  differences?: Difference[];
  onSelect: (p: Vec3, adjacent: Vec3) => void;
  onResize: (corner: 0 | 1, p: Vec3) => void;
  onView: (view: Record<string, unknown>) => void;
  onGhostAnchor?: (p: Vec3) => void;
};
export default function StructureViewport(p: Props) {
  const scope = useDocumentScope();
  const active = useRef(scope.active);
  active.current = scope.active;
  const invalidate = useRef<(() => void) | undefined>(undefined);
  const root = useRef<HTMLDivElement>(null),
    latest = useRef(p);
  latest.current = p;
  const viewerID = useRef(crypto.randomUUID());
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [progress, setProgress] = useState({ loaded: 0, total: 0 });
  const [reload, setReload] = useState(0);
  const controller = useRef<
    { refresh: () => void; stop: () => void } | undefined
  >(undefined);
  const cameraMemory = useRef<
    { id: string; position: number[]; target: number[] } | undefined
  >(undefined);
  useEffect(() => {
    const host = root.current;
    if (!host) return;
    setLoading(true);
    setProgress({loaded: 0, total: 0});
    let disposed = false,
      loadingStopped = false;
    const resources = new Set<
      THREE.BufferGeometry | THREE.Material | THREE.Texture
    >();
    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
    } catch (e) {
      setStatus(String(e));
      setLoading(false);
      return;
    }
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setClearColor(
      new THREE.Color(getComputedStyle(host).backgroundColor),
    );
    renderer.localClippingEnabled = true;
    host.appendChild(renderer.domElement);
    const scene = new THREE.Scene(),
      camera = new THREE.PerspectiveCamera(45, 1, 0.1, 20000),
      controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.screenSpacePanning = true;
    const center = new THREE.Vector3(...p.doc.size).multiplyScalar(0.5),
      distance = Math.max(...p.doc.size, 12) * 1.1;
    camera.position
      .copy(center)
      .add(new THREE.Vector3(distance, distance * 0.8, distance));
    controls.target.copy(center);
    const saved = (
      cameraMemory.current?.id === p.doc.id
        ? cameraMemory.current
        : p.doc.view.camera
    ) as { position?: number[]; target?: number[] } | undefined;
    if (
      saved?.position?.length === 3 &&
      saved?.target?.length === 3 &&
      [...saved.position, ...saved.target].every(Number.isFinite)
    ) {
      camera.position.fromArray(saved.position);
      controls.target.fromArray(saved.target);
    }
    scene.add(new THREE.HemisphereLight(0xeaf4ff, 0x74806a, 2.3));
    const light = new THREE.DirectionalLight(0xfff6e2, 2.2);
    light.position.set(100, 170, 70);
    scene.add(light);
    const gridSize = Math.min(4096, Math.max(p.doc.size[0], p.doc.size[2], 16));
    const grid = new THREE.GridHelper(
      gridSize,
      Math.min(gridSize, 128),
      0x46525d,
      0x2c343b,
    );
    grid.position.set(p.doc.size[0] / 2, -0.02, p.doc.size[2] / 2);
    scene.add(grid);
    const loop = createPreviewLoop({
      draw: () => {
        const changed = controls.update();
        renderer.render(scene, camera);
        return changed;
      },
      visible: () =>
        !disposed &&
        active.current &&
        !document.hidden &&
        host.clientWidth > 0 &&
        host.clientHeight > 0 &&
        !host.closest("[hidden]"),
      continuous: () => false,
    });
    const requestDraw = () => loop.invalidate();
    invalidate.current = requestDraw;
    controls.addEventListener("change", requestDraw);
    document.addEventListener("visibilitychange", requestDraw);
    renderer.domElement.addEventListener("webglcontextrestored", requestDraw);
    const updateTheme = () => {
      const style = getComputedStyle(host);
      renderer.setClearColor(new THREE.Color(style.backgroundColor));
      const gridMaterials = Array.isArray(grid.material)
        ? grid.material
        : [grid.material];
      for (const material of gridMaterials) {
        material.vertexColors = false;
        material.color.set(
          style.getPropertyValue("--border").trim() || "#484848",
        );
      }
      requestDraw();
    };
    updateTheme();
    const themeObserver = new MutationObserver(updateTheme);
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "style", "data-theme"],
    });
    resources.add(grid.geometry);
    for (const m of Array.isArray(grid.material)
      ? grid.material
      : [grid.material])
      resources.add(m);
    const world = new THREE.Group(),
      ghost = new THREE.Group(),
      overlay = new THREE.Group();
    scene.add(world, ghost, overlay);
    const selection = new THREE.Box3Helper(new THREE.Box3(), 0x52a9ff);
    (selection.material as THREE.Material).depthTest = false;
    selection.renderOrder = 20;
    scene.add(selection);
    resources.add(selection.geometry);
    resources.add(selection.material as THREE.Material);
    const marker = new THREE.Object3D();
    scene.add(marker);
    const gizmo = new TransformControls(camera, renderer.domElement);
    gizmo.setSize(0.65);
    gizmo.setTranslationSnap(1);
    scene.add(gizmo.getHelper());
    gizmo.attach(marker);
    gizmo.addEventListener("change", requestDraw);
    gizmo.addEventListener("dragging-changed", (e) => {
      controls.enabled = !e.value;
    });
    gizmo.addEventListener("mouseUp", () => {
      const v = marker.position;
      if (latest.current.ghost) {
        latest.current.onGhostAnchor?.(v.toArray().map(Math.round) as Vec3);
        return;
      }
      const size = latest.current.doc.size;
      const q = [v.x, v.y, v.z].map((n, i) =>
        Math.max(0, Math.min(size[i] - 1, Math.round(n))),
      ) as Vec3;
      latest.current.onResize(latest.current.corner, q);
    });
    const materials: THREE.MeshStandardMaterial[] = [];
    const decoder = new SceneDecoder();
    let lastTop = false;
    const refresh = () => {
      const q = latest.current,
        b = normalized(q.selection);
      selection.box.set(
        new THREE.Vector3(...b.min),
        new THREE.Vector3(...b.max).addScalar(1),
      );
      if (!gizmo.dragging && q.ghost) marker.position.fromArray(q.ghost.anchor);
      else if (!gizmo.dragging)
        marker.position.fromArray(
          q.corner === 0 ? q.selection.min : q.selection.max,
        );
      gizmo.enabled = true;
      const targetY = q.layerOnly ? q.layer : 1e9;
      for (const material of materials) {
        material.clippingPlanes = q.layerOnly
          ? [
              new THREE.Plane(new THREE.Vector3(0, -1, 0), targetY + 1.001),
              new THREE.Plane(new THREE.Vector3(0, 1, 0), -targetY + 0.001),
            ]
          : [];
      }
      if (q.ghost) {
        ghost.position.set(
          q.ghost.anchor[0] - q.ghost.doc.anchor[0],
          q.ghost.anchor[1] - q.ghost.doc.anchor[1],
          q.ghost.anchor[2] - q.ghost.doc.anchor[2],
        );
      }
      if (q.top !== lastTop) {
        const target = controls.target.clone(),
          dist = camera.position.distanceTo(target);
        camera.position
          .copy(target)
          .add(
            q.top
              ? new THREE.Vector3(0.001, dist, 0.001)
              : new THREE.Vector3(dist * 0.65, dist * 0.55, dist * 0.65),
          );
        camera.up.set(0, 1, 0);
        lastTop = q.top;
      }
      overlay.traverse((o) => {
        if (o instanceof THREE.LineSegments) {
          o.geometry.dispose();
          (o.material as THREE.Material).dispose();
        }
      });
      overlay.clear();
      const colors: Record<string, number> = {
        missing: 0x36cf8b,
        extra: 0xe6a84a,
        wrong: 0xed6677,
        state: 0xb38aff,
        unknown: 0x7b8895,
      };
      for (const d of (q.differences || []).slice(0, 3000)) {
        if (q.layerOnly && d.position[1] !== q.layer) continue;
        const box = new THREE.Box3Helper(
          new THREE.Box3(
            new THREE.Vector3(...d.position),
            new THREE.Vector3(...d.position).addScalar(1),
          ),
          colors[d.kind] || 0xffffff,
        );
        (box.material as THREE.Material).depthTest = false;
        box.renderOrder = 21;
        overlay.add(box);
      }
      requestDraw();
    };
    controller.current = {
      refresh,
      stop: () => {
        loadingStopped = true;
        setLoading(false);
        void structures.CancelStructurePreview("", viewerID.current);
        setStatus("Preview paused.");
      },
    };
    refresh();
    let start = [0, 0];
    const down = (e: PointerEvent) => {
      start = [e.clientX, e.clientY];
    };
    const click = (e: PointerEvent) => {
      if (
        gizmo.dragging ||
        gizmo.axis ||
        Math.hypot(e.clientX - start[0], e.clientY - start[1]) > 4 ||
        e.button !== 0
      )
        return;
      const rect = renderer.domElement.getBoundingClientRect();
      const ray = new THREE.Raycaster();
      ray.setFromCamera(
        new THREE.Vector2(
          ((e.clientX - rect.left) / rect.width) * 2 - 1,
          (-(e.clientY - rect.top) / rect.height) * 2 + 1,
        ),
        camera,
      );
      const hits = ray
        .intersectObject(world, true)
        .filter(
          (hit) =>
            !latest.current.layerOnly ||
            (hit.point.y >= latest.current.layer - 0.001 &&
              hit.point.y <= latest.current.layer + 1.001),
        );
      const hit = hits[0];
      let point: THREE.Vector3,
        normal = new THREE.Vector3(0, 1, 0);
      if (hit) {
        point = hit.point.clone();
        if (hit.face)
          normal
            .copy(hit.face.normal)
            .transformDirection(hit.object.matrixWorld);
      } else {
        const plane = new THREE.Plane(
          new THREE.Vector3(0, 1, 0),
          -latest.current.layer,
        );
        point = new THREE.Vector3();
        if (!ray.ray.intersectPlane(plane, point)) return;
      }
      const inside = point
          .clone()
          .addScaledVector(normal, hit ? -0.002 : 0.002),
        outside = point.clone().addScaledVector(normal, 0.002);
      latest.current.onSelect(
        inside.toArray().map(Math.floor) as Vec3,
        outside.toArray().map(Math.floor) as Vec3,
      );
    };
    renderer.domElement.addEventListener("pointerdown", down);
    renderer.domElement.addEventListener("pointerup", click);
    const saveView = () =>
      latest.current.onView({
        camera: {
          position: camera.position.toArray(),
          target: controls.target.toArray(),
        },
      });
    controls.addEventListener("end", saveView);
    const resize = new ResizeObserver(() => {
      const w = host.clientWidth,
        h = host.clientHeight;
      if (w && h) {
        renderer.setSize(w, h);
        camera.aspect = w / h;
        camera.updateProjectionMatrix();
        requestDraw();
      }
    });
    resize.observe(host);
    async function load(
      id: string,
      parent: THREE.Group,
      isGhost: boolean,
      previewY?: number,
    ) {
      let cursor: [number, number] | null = null,
        revision = 0,
        target = 0,
        scope = 0;
      let pixels: Uint8Array | undefined,
        texture: THREE.DataTexture | undefined;
      let pages = 0;
      do {
        if (disposed || loadingStopped) return;
        const page = await structures.GetStructureScenePage(id, {
          ...(previewY !== undefined ? { minY: previewY, maxY: previewY } : {}),
          generation,
          viewer: viewerID.current,
          limit: 12,
          cursor,
          after: 0,
          target,
          scope,
          atlasRevision: revision,
          focus: [
            Math.floor(p.doc.size[0] / 32),
            Math.floor(p.doc.size[2] / 32),
          ],
        });
        if (disposed || loadingStopped) return;
        const batch = await decoder.decode(page);
        if (disposed || loadingStopped) return;
        if (!texture) {
          pixels = new Uint8Array(page.atlasSize * page.atlasSize * 4);
          texture = new THREE.DataTexture(
            pixels,
            page.atlasSize,
            page.atlasSize,
          );
          texture.colorSpace = THREE.SRGBColorSpace;
          texture.flipY = false;
          texture.magFilter = texture.minFilter = THREE.NearestFilter;
          resources.add(texture);
        }
        for (const patch of batch.patches)
          for (let y = 0; y < patch.size; y++)
            pixels!.set(
              patch.pixels.subarray(
                y * patch.size * 4,
                (y + 1) * patch.size * 4,
              ),
              ((patch.y + y) * page.atlasSize + patch.x) * 4,
            );
        texture.needsUpdate = true;
        for (const chunk of batch.chunks)
          for (const part of chunk.parts) {
            const g = new THREE.BufferGeometry();
            g.setAttribute(
              "position",
              new THREE.BufferAttribute(part.positions, 3),
            );
            g.setAttribute(
              "normal",
              new THREE.BufferAttribute(part.normals, 3, true),
            );
            g.setAttribute("uv", new THREE.BufferAttribute(part.uvs, 2));
            g.setIndex(new THREE.BufferAttribute(part.indices, 1));
            g.computeBoundingSphere();
            const blend = ["blend", "water", "unlit-blend"].includes(part.mode);
            const material = new THREE.MeshStandardMaterial({
              map: texture,
              roughness: 1,
              side: THREE.DoubleSide,
              alphaTest: blend ? 0.01 : 0.45,
              transparent: isGhost || blend,
              opacity: isGhost ? 0.4 : blend ? 0.72 : 1,
              depthWrite: !isGhost && !blend,
              color: isGhost ? 0x87c6ff : 0xffffff,
            });
            resources.add(g);
            resources.add(material);
            if (!isGhost) materials.push(material);
            const mesh = new THREE.Mesh(g, material);
            mesh.position.set(chunk.x * 16, 0, chunk.z * 16);
            parent.add(mesh);
          }
        target = page.sequence;
        scope = page.scope;
        revision = page.atlasRevision;
        cursor = page.next;
        setProgress({loaded: page.complete ? page.total : Math.min((pages + 1) * 12, page.total), total: page.total});
        setStatus(
          page.complete
            ? ""
            : `Loading 3D · ${Math.min(++pages * 12, page.total)} / ${page.total} chunks`,
        );
        refresh();
        if (page.complete) break;
        await new Promise<void>((resolve) => window.setTimeout(resolve, 0));
      } while (cursor);
    }
    const generation = crypto.randomUUID();
    const neighbors = new THREE.Group();
    scene.add(neighbors);
    void load(p.doc.id, world, false, p.layerOnly ? p.layer : undefined)
      .then(async () => {
        if (p.layerOnly && p.adjacent) {
          if (p.layer > 0) await load(p.doc.id, neighbors, true, p.layer - 1);
          if (p.layer + 1 < p.doc.size[1])
            await load(p.doc.id, neighbors, true, p.layer + 1);
        }
      })
      .then(() => (p.ghost ? load(p.ghost.doc.id, ghost, true) : undefined))
      .then(() => new Promise<void>(resolve => requestAnimationFrame(() => resolve())))
      .then(() => { if (!disposed) setLoading(false); })
      .catch((e) => {
        if (!disposed && !loadingStopped) { setStatus(String(e)); setLoading(false); }
      });
    requestDraw();
    return () => {
      disposed = true;
      cameraMemory.current = {
        id: p.doc.id,
        position: camera.position.toArray(),
        target: controls.target.toArray(),
      };
      loop.dispose();
      invalidate.current = undefined;
      controls.removeEventListener("change", requestDraw);
      document.removeEventListener("visibilitychange", requestDraw);
      renderer.domElement.removeEventListener("webglcontextrestored", requestDraw);
      controller.current = undefined;
      resize.disconnect();
      themeObserver.disconnect();
      decoder.dispose();
      gizmo.dispose();
      controls.dispose();
      overlay.traverse((o) => {
        if (o instanceof THREE.LineSegments) {
          o.geometry.dispose();
          (o.material as THREE.Material).dispose();
        }
      });
      for (const r of resources) r.dispose();
      renderer.dispose();
      renderer.domElement.remove();
    };
  }, [
    p.doc.id,
    p.doc.revision,
    p.ghost?.doc.id,
    p.ghost?.doc.revision,
    p.layerOnly,
    p.layerOnly ? p.layer : undefined,
    p.layerOnly ? p.adjacent : undefined,
    reload,
  ]);
  useEffect(() => {
    invalidate.current?.();
  }, [scope.active]);
  useEffect(
    () => controller.current?.refresh(),
    [
      p.selection,
      p.corner,
      p.layer,
      p.layerOnly,
      p.adjacent,
      p.top,
      p.ghost?.anchor,
      p.differences,
    ],
  );
  return (
    <div
      className="structure-viewport"
      ref={root}
      aria-label="Structure 3D editor"
    >
      {loading && <EditorLoadingOverlay label="Loading structure" loaded={progress.loaded} total={progress.total} onCancel={() => controller.current?.stop()} />}
      {status && !loading && (
        <div className="structure-viewport-status" role="status">
          {status}
          <button onClick={() => controller.current?.stop()}>
            Stop preview
          </button>
          <button onClick={() => setReload((n) => n + 1)}>Reload</button>
        </div>
      )}
      <div className="structure-viewport-hint">
        Drag to orbit · Right-drag to pan · Scroll to zoom · Click a block to
        edit selection
      </div>
    </div>
  );
}

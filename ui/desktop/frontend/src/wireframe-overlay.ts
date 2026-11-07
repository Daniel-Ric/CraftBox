import * as T from "three";
import { LineSegments2 } from "three/addons/lines/LineSegments2.js";
import { LineSegmentsGeometry } from "three/addons/lines/LineSegmentsGeometry.js";
import { LineMaterial } from "three/addons/lines/LineMaterial.js";
import type { WireframeSettings, WireframeStats } from "./wireframe-settings";

type Position = T.BufferAttribute | T.InterleavedBufferAttribute;
type Entry = {
  line: LineSegments2;
  hidden: LineSegments2;
  points: T.Points;
  signature: string;
  indices: number[];
  vertexIndices: number[];
  version: number;
  position: Position;
  index: T.BufferAttribute | null;
  restore: () => void;
};
const maxSegments = 125000,
  maxSourceIndices = 750000,
  maxVertices = 50000;
export function createWireframeOverlay(root: T.Object3D) {
  const material = new LineMaterial({
    color: 0x58bdff,
    linewidth: 1.25,
    transparent: true,
    opacity: 0.85,
    depthWrite: false,
    toneMapped: false,
  });
  const hiddenMaterial = new LineMaterial({
    color: 0x63839a,
    linewidth: 1.25,
    transparent: true,
    opacity: 0.25,
    depthWrite: false,
    depthTest: true,
    depthFunc: T.GreaterDepth,
    toneMapped: false,
    dashed: true,
    dashSize: 0.12,
    gapSize: 0.08,
  });
  const vertexMaterial = new T.PointsMaterial({
    color: 0xffd166,
    size: 4,
    sizeAttenuation: false,
    depthWrite: false,
    transparent: true,
    opacity: 0.9,
    toneMapped: false,
  });
  const entries = new Map<T.Mesh, Entry>(),
    skipped = new Map<T.Mesh, { signature: string; count: number }>();
  let hideSurface = false,
    stats: WireframeStats = { meshes: 0, segments: 0, vertices: 0, skipped: 0 };
  function disposeEntry(entry: Entry) {
    entry.restore();
    entry.line.removeFromParent();
    entry.hidden.removeFromParent();
    entry.points.removeFromParent();
    entry.line.geometry.dispose();
    entry.points.geometry.dispose();
  }
  function bindSurface(source: T.Mesh, entry: Entry) {
    const before = source.onBeforeRender,
      after = source.onAfterRender;
    const saved = new Map<
      T.Material,
      { colorWrite: boolean; depthWrite: boolean }
    >();
    const restoreMaterial = (m: T.Material) => {
      const old = saved.get(m);
      if (old) {
        m.colorWrite = old.colorWrite;
        m.depthWrite = old.depthWrite;
        saved.delete(m);
      }
    };
    const beforeHook: T.Object3D["onBeforeRender"] = function (...args) {
      before.apply(source, args);
      if (hideSurface && entry.line.visible) {
        const m = args[4];
        saved.set(m, { colorWrite: m.colorWrite, depthWrite: m.depthWrite });
        m.colorWrite = false;
        m.depthWrite = true;
      }
    };
    const afterHook: T.Object3D["onAfterRender"] = function (...args) {
      restoreMaterial(args[4]);
      after.apply(source, args);
    };
    source.onBeforeRender = beforeHook;
    source.onAfterRender = afterHook;
    entry.restore = () => {
      for (const m of saved.keys()) restoreMaterial(m);
      if (source.onBeforeRender === beforeHook) source.onBeforeRender = before;
      if (source.onAfterRender === afterHook) source.onAfterRender = after;
    };
  }
  function markPreview(object: T.Object3D, name: string, order: number) {
    object.name = name;
    object.userData.previewOnly = true;
    object.raycast = () => {};
    object.renderOrder = order;
    object.frustumCulled = false;
  }
  function updatePositions(entry: Entry, position: Position) {
    const start = entry.line.geometry.getAttribute(
        "instanceStart",
      ) as T.InterleavedBufferAttribute,
      end = entry.line.geometry.getAttribute(
        "instanceEnd",
      ) as T.InterleavedBufferAttribute;
    for (let i = 0; i < entry.indices.length; i++) {
      const index = entry.indices[i];
      if (index >= 0)
        (i % 2 ? end : start).setXYZ(
          Math.floor(i / 2),
          position.getX(index),
          position.getY(index),
          position.getZ(index),
        );
    }
    start.data.needsUpdate = true;
    entry.line.geometry.computeBoundingBox();
    entry.line.geometry.computeBoundingSphere();
    const distanceStart = entry.line.geometry.getAttribute(
      "instanceDistanceStart",
    ) as T.InterleavedBufferAttribute | undefined;
    if (!distanceStart) {
      entry.hidden.computeLineDistances();
    } else {
      const distanceEnd = entry.line.geometry.getAttribute(
        "instanceDistanceEnd",
      ) as T.InterleavedBufferAttribute;
      let distance = 0;
      for (let i = 0; i < start.count; i++) {
        distanceStart.setX(i, distance);
        distance += Math.hypot(
          end.getX(i) - start.getX(i),
          end.getY(i) - start.getY(i),
          end.getZ(i) - start.getZ(i),
        );
        distanceEnd.setX(i, distance);
      }
      distanceStart.data.needsUpdate = true;
    }
    const points = entry.points.geometry.getAttribute(
      "position",
    ) as T.BufferAttribute;
    entry.vertexIndices.forEach((index, i) =>
      points.setXYZ(
        i,
        position.getX(index),
        position.getY(index),
        position.getZ(index),
      ),
    );
    points.needsUpdate = true;
    entry.points.geometry.computeBoundingSphere();
  }
  return {
    stats: () => ({ ...stats }),
    update(
      enabled: boolean,
      settings: WireframeSettings = {},
      width = 1,
      height = 1,
    ) {
      hideSurface = enabled && !!settings.hideSurface;
      for (const entry of entries.values()) {
        entry.line.visible = enabled;
        entry.hidden.visible = enabled && !!settings.xray;
        entry.points.visible = enabled && !!settings.vertices;
      }
      if (!enabled) {
        stats = { meshes: 0, segments: 0, vertices: 0, skipped: 0 };
        return;
      }
      material.color.set(settings.color ?? "#58bdff");
      material.linewidth = T.MathUtils.clamp(settings.width ?? 1.25, 0.5, 6);
      material.opacity = T.MathUtils.clamp(settings.opacity ?? 0.85, 0.05, 1);
      material.depthTest = true;
      material.resolution.set(width, height);
      hiddenMaterial.color.set(settings.hiddenColor ?? "#63839a");
      hiddenMaterial.linewidth = material.linewidth;
      hiddenMaterial.opacity = T.MathUtils.clamp(
        settings.hiddenOpacity ?? 0.25,
        0.05,
        1,
      );
      hiddenMaterial.resolution.set(width, height);
      hiddenMaterial.dashScale = T.MathUtils.clamp(
        settings.dashScale ?? 1,
        0.25,
        8,
      );
      const dashed = settings.dashedHidden !== false;
      if (hiddenMaterial.dashed !== dashed) {
        hiddenMaterial.dashed = dashed;
        hiddenMaterial.needsUpdate = true;
      }
      vertexMaterial.color.set(settings.vertexColor ?? "#ffd166");
      vertexMaterial.size = T.MathUtils.clamp(settings.vertexSize ?? 4, 1, 10);
      vertexMaterial.depthTest = !settings.xray;
      const sources: T.Mesh[] = [];
      root.traverse((node) => {
        if (node instanceof T.Mesh && !node.userData.previewOnly)
          sources.push(node);
      });
      let segments = 0,
        vertices = 0,
        meshes = 0,
        omitted = 0;
      for (const source of sources) {
        const geometry = source.geometry,
          position = geometry.getAttribute("position");
        if (!position) continue;
        const version =
          position instanceof T.InterleavedBufferAttribute
            ? position.data.version
            : position.version;
        const threshold = T.MathUtils.clamp(
          settings.thresholdAngle ?? 15,
          0,
          180,
        );
        const signature = `${geometry.uuid}:${position.count}:${geometry.index?.version}:${settings.mode ?? "edges"}:${threshold}`;
        let entry = entries.get(source);
        if (
          !entry ||
          entry.signature !== signature ||
          entry.index !== geometry.index
        ) {
          const skip = skipped.get(source);
          if (
            (skip?.signature === signature &&
              segments + skip.count > maxSegments) ||
            (geometry.index?.count ?? position.count) > maxSourceIndices
          ) {
            if (entry) {
              entry.line.visible = false;
              entry.hidden.visible = false;
              entry.points.visible = false;
            }
            omitted++;
            continue;
          }
          const edges =
            settings.mode === "triangles"
              ? new T.WireframeGeometry(geometry)
              : new T.EdgesGeometry(geometry, threshold);
          const values = edges.getAttribute("position").array,
            count = values.length / 6;
          if (segments + count > maxSegments) {
            edges.dispose();
            skipped.set(source, { signature, count });
            if (entry) {
              entry.line.visible = false;
              entry.hidden.visible = false;
              entry.points.visible = false;
            }
            omitted++;
            continue;
          }
          skipped.delete(source);
          const wide = new LineSegmentsGeometry().setPositions(
            values as Float32Array,
          );
          (
            wide.getAttribute("instanceStart") as T.InterleavedBufferAttribute
          ).data.setUsage(T.DynamicDrawUsage);
          const vertexMap = new Map<string, number>();
          for (let i = 0; i < position.count; i++)
            vertexMap.set(
              `${position.getX(i)},${position.getY(i)},${position.getZ(i)}`,
              i,
            );
          const indices = Array.from(
            { length: values.length / 3 },
            (_, i) =>
              vertexMap.get(
                `${values[i * 3]},${values[i * 3 + 1]},${values[i * 3 + 2]}`,
              ) ?? -1,
          );
          const vertexIndices = [
            ...new Set(indices.filter((i) => i >= 0)),
          ].slice(0, maxVertices);
          edges.dispose();
          if (entry) disposeEntry(entry);
          const line = new LineSegments2(wide, material),
            hidden = new LineSegments2(wide, hiddenMaterial),
            pointGeometry = new T.BufferGeometry().setAttribute(
              "position",
              new T.BufferAttribute(
                new Float32Array(vertexIndices.length * 3),
                3,
              ),
            ),
            points = new T.Points(pointGeometry, vertexMaterial);
          markPreview(line, "Preview wireframe", 20);
          markPreview(hidden, "Preview hidden wireframe", 19);
          markPreview(points, "Preview wireframe vertices", 21);
          source.add(line, hidden, points);
          entry = {
            line,
            hidden,
            points,
            signature,
            indices,
            vertexIndices,
            version: -1,
            position,
            index: geometry.index,
            restore: () => {},
          };
          entries.set(source, entry);
          bindSurface(source, entry);
        }
        if (entry.version !== version || entry.position !== position) {
          updatePositions(entry, position);
          entry.version = version;
          entry.position = position;
        }
        segments += entry.line.geometry.instanceCount;
        const visible = segments <= maxSegments;
        entry.line.visible = visible;
        entry.hidden.visible = visible && !!settings.xray;
        entry.points.visible = visible && !!settings.vertices;
        meshes++;
        if (settings.vertices) vertices += entry.vertexIndices.length;
      }
      const active = new Set(sources);
      for (const source of skipped.keys())
        if (!active.has(source)) skipped.delete(source);
      for (const [source, entry] of entries)
        if (!active.has(source)) {
          disposeEntry(entry);
          entries.delete(source);
        }
      stats = { meshes, segments, vertices, skipped: omitted };
    },
    dispose() {
      hideSurface = false;
      for (const entry of entries.values()) disposeEntry(entry);
      entries.clear();
      skipped.clear();
      material.dispose();
      hiddenMaterial.dispose();
      vertexMaterial.dispose();
    },
  };
}

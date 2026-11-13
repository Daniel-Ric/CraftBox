import * as T from "three";
import { OBB } from "three/addons/math/OBB.js";
import type { ModelPart, ModelElement, PartMode } from "./model-parts";

type Unit = { name: string; members: T.Mesh[]; node?: T.Object3D };
type Shape = { mesh: T.Mesh; box: T.Box3; obb: OBB; extent: number };

export function modelParts(root: T.Group, mode: PartMode, rest?: ReadonlyMap<T.Mesh, T.Matrix4>): ModelPart[] {
  root.updateMatrixWorld(true);
  const shapes = new Map<T.Mesh, Shape>(), groups: T.Group[] = [];
  root.traverse(node => {
    if (node instanceof T.Group && node.userData.baseRotation) groups.push(node);
    if (!(node instanceof T.Mesh) || !node.userData.sourceElement || node.userData.previewOnly) return;
    node.geometry.computeBoundingBox();
    const local = node.geometry.boundingBox;
    if (!local || local.isEmpty()) return;
    const size = local.getSize(new T.Vector3());
    if ([size.x, size.y, size.z].filter(value => value > 1e-7).length < 2) return;
    const matrix = rest?.get(node) ?? node.matrixWorld;
    const box = local.clone().applyMatrix4(matrix);
    shapes.set(node, { mesh: node, box, obb: new OBB().fromBox3(local).applyMatrix4(matrix), extent: box.getSize(new T.Vector3()).length() });
  });
  const members = (node: T.Object3D) => {
    const list: T.Mesh[] = [];
    node.traverse(child => { if (child instanceof T.Mesh && shapes.has(child)) list.push(child); });
    return list;
  };
  const part = (name: string, meshes: T.Mesh[], detection: ModelPart["detection"]): ModelPart => {
    const elements = meshes.map(mesh => mesh.userData.sourceElement as ModelElement);
    const first = elements[0];
    return { id: `${mode}:${name}:${JSON.stringify([first.bone, first.cube])}`, name, sourceName: name, detection, elements, bones: [...new Set(elements.map(element => element.bone))] };
  };
  if (mode === "elements") return [...shapes.values()].map(({ mesh }) => {
    const element = mesh.userData.sourceElement as ModelElement;
    return part(`${element.bone} · ${element.cube < 0 ? "Mesh" : `Cube ${element.cube + 1}`}`, [mesh], "element");
  });
  if (mode === "bones") return groups.flatMap(group => {
    const list = members(group);
    return list.length ? [part(group.name, list, "bone")] : [];
  });
  if (!shapes.size) return [];
  const scene = [...shapes.values()].reduce((box, shape) => box.union(shape.box), new T.Box3());
  const extents = [...shapes.values()].map(shape => shape.extent).sort((a, b) => a - b);
  const tolerance = Math.max(scene.getSize(new T.Vector3()).length() * 1e-6, extents[Math.floor(extents.length / 2)] * 0.004, 1e-7);
  for (const shape of shapes.values()) shape.obb.halfSize.addScalar(tolerance);
  const connected = (units: Unit[]) => {
    const parents = units.map((_, index) => index);
    const find = (index: number): number => parents[index] === index ? index : (parents[index] = find(parents[index]));
    const sorted = units.flatMap((unit, index) => unit.members.map(mesh => ({ index, shape: shapes.get(mesh)! }))).sort((a, b) => a.shape.box.min.x - b.shape.box.min.x);
    let remaining = units.length;
    scan: for (let i = 0; i < sorted.length; i++) for (let j = i + 1; j < sorted.length && sorted[j].shape.box.min.x <= sorted[i].shape.box.max.x + tolerance * 2; j++) {
      const left = find(sorted[i].index), right = find(sorted[j].index);
      if (left === right) continue;
      if (!sorted[i].shape.obb.intersectsOBB(sorted[j].shape.obb)) continue;
      parents[right] = left;
      if (--remaining <= 1) break scan;
    }
    const components = new Map<number, Unit[]>();
    units.forEach((unit, index) => {
      const key = find(index), component = components.get(key) ?? [];
      component.push(unit); components.set(key, component);
    });
    return [...components.values()];
  };
  if (mode === "islands") return connected([...shapes.values()].map(({ mesh }) => ({ name: mesh.name, members: [mesh] }))).map(component => {
    const dominant = [...component].sort((a, b) => b.members.length - a.members.length)[0];
    return part(dominant.name, component.flatMap(unit => unit.members), "geometry");
  });
  const boneGroups = new Set<T.Object3D>(groups);
  const partition = (node: T.Object3D): ModelPart[] => {
    const branches: Unit[] = [], own: T.Mesh[] = [];
    const visit = (child: T.Object3D) => {
      if (boneGroups.has(child)) {
        const list = members(child);
        if (list.length) branches.push({ name: child.name, members: list, node: child });
      } else if (child instanceof T.Mesh && shapes.has(child)) own.push(child);
      else for (const nested of child.children) visit(nested);
    };
    node.children.forEach(visit);
    if (!own.length && branches.length === 1) return partition(branches[0].node!);
    const units = mode === "hierarchy"
      ? [...(own.length ? [{ name: node.name || "Model", members: own }] : []), ...branches]
      : [...own.map(mesh => ({ name: node.name || mesh.name, members: [mesh] })), ...branches];
    if (!units.length) return [];
    const components = mode === "hierarchy" ? units.map(unit => [unit]) : connected(units);
    return components.map(component => {
      const dominant = [...component].sort((a, b) => b.members.length - a.members.length)[0];
      return part(component.length === units.length ? node.name || dominant.name : dominant.name, component.flatMap(unit => unit.members), branches.length ? "hierarchy" : "geometry");
    });
  };
  return partition(root);
}

import * as T from "three";
import type { BlockbenchModel } from "./blockbench-project";
import { bedrockModel, releaseModel } from "./asset-geometry";
import { selectModelMotion } from "./model-motion-selection";

export type PartMode = "parts" | "hierarchy" | "islands" | "bones" | "elements";
export type ModelElement = { bone: string; cube: number };
export type ModelPart = { id: string; name: string; sourceName?: string; detection?: "hierarchy" | "geometry" | "bone" | "element"; elements: ModelElement[]; bones: string[] };
export const elementKey = (element: ModelElement) => JSON.stringify([element.bone, element.cube]);

export { modelParts } from "./model-part-detection";

type Data = Record<string, any>;
function centerGeometry(geometry: Data) {
  const model = bedrockModel({ "minecraft:geometry": [geometry] }, new T.MeshBasicMaterial());
  try {
    const bounds = new T.Box3().setFromObject(model);
    if (bounds.isEmpty()) return undefined;
    const center = bounds.getCenter(new T.Vector3());
    const delta = [center.x, -bounds.min.y, -center.z];
    const move = (value: unknown) => Array.isArray(value) && value.length === 3 ? value.map((n, axis) => Number(n) + delta[axis]) : value;
    for (const bone of geometry.bones as Data[]) {
      bone.pivot = move(bone.pivot ?? [0, 0, 0]);
      for (const cube of bone.cubes ?? []) {
        cube.origin = move(cube.origin);
        if (cube.pivot) cube.pivot = move(cube.pivot);
      }
      if (bone.poly_mesh) bone.poly_mesh.positions = bone.poly_mesh.positions.map(move);
      for (const [name, locator] of Object.entries(bone.locators ?? {})) {
        if (Array.isArray(locator)) bone.locators[name] = move(locator);
        else if (locator && typeof locator === "object") {
          const definition = locator as Data;
          definition.offset = move(definition.offset ?? [0, 0, 0]);
        }
      }
    }
    if (geometry.description) {
      const size = bounds.getSize(new T.Vector3());
      geometry.description.visible_bounds_offset = [0, size.y / 32, 0];
      geometry.description.visible_bounds_width = Math.max(size.x, size.z) / 16 + 1;
      geometry.description.visible_bounds_height = size.y / 16 + 1;
    }
    const height = bounds.max.y - bounds.min.y;
    const target = new T.Vector3(0, height / 2, 0);
    const distance = Math.max(24, bounds.getSize(new T.Vector3()).length() * 1.7);
    const position = new T.Vector3(-1, 0.65, -1).normalize().multiplyScalar(distance).add(target);
    return { position: position.toArray(), target: target.toArray() };
  } finally { releaseModel(model); }
}

function definition(raw: unknown, identifier?: string): Data {
  const input = raw as Data;
  const modern = input?.["minecraft:geometry"];
  if (Array.isArray(modern)) {
    const selected = identifier ? modern.find(value => value.description?.identifier === identifier) : modern.length === 1 ? modern[0] : undefined;
    if (selected) return selected;
  }
  if (Array.isArray(input?.bones)) return input;
  const key = identifier || Object.keys(input ?? {}).find(key => key.startsWith("geometry."));
  if (key && input[key]) return input[key];
  throw Error("Select one authored geometry before extracting model parts.");
}

export function extractModelSelection(model: BlockbenchModel, parts: ModelPart[], name: string): BlockbenchModel {
  if (!parts.length) throw Error("Select at least one model part to extract.");
  const copy = structuredClone(model);
  const geometry = definition(copy.geometry, copy.geometryId);
  const bones = geometry.bones as Data[];
  const selected = new Set(parts.flatMap(part => part.elements.map(elementKey)));
  const available = new Set(bones.flatMap(bone => [
    ...(bone.cubes ?? []).map((_: unknown, cube: number) => elementKey({ bone: bone.name, cube })),
    ...(bone.poly_mesh ? [elementKey({ bone: bone.name, cube: -1 })] : []),
  ]));
  if ([...selected].some(key => !available.has(key))) throw Error("The selection no longer matches this model. Select the parts again.");
  const retained = new Set<string>();
  const byName = new Map(bones.map(bone => [String(bone.name), bone]));
  let elements = 0;
  for (const bone of bones) {
    const cubes = (bone.cubes ?? []) as Data[];
    bone.cubes = cubes.filter((_, index) => selected.has(elementKey({ bone: bone.name, cube: index })));
    elements += bone.cubes.length;
    if (bone.poly_mesh && selected.has(elementKey({ bone: bone.name, cube: -1 }))) elements++;
    else delete bone.poly_mesh;
    if (!bone.cubes.length && !bone.poly_mesh) continue;
    let ancestor: Data | undefined = bone;
    const visited = new Set<string>();
    while (ancestor) {
      if (visited.has(ancestor.name)) throw Error("Model contains a parent cycle.");
      visited.add(ancestor.name);
      retained.add(ancestor.name);
      ancestor = byName.get(ancestor.parent);
    }
  }
  if (!elements) throw Error("The selection no longer matches this model. Select the parts again.");
  geometry.bones = bones.filter(bone => retained.has(bone.name));
  copy.previewCamera = centerGeometry(geometry);
  const animationBones = new Set([...retained].map(name => name.toLowerCase()));
  const modern = (copy.geometry as Data)["minecraft:geometry"];
  if (Array.isArray(modern)) (copy.geometry as Data)["minecraft:geometry"] = [geometry];
  copy.name = name.trim() || "Extracted model";
  selectModelMotion(copy, animationBones);
  for (const [path, value] of Object.entries(copy.sources ?? {})) {
    if (!value || typeof value !== "object") continue;
    if (path === copy.object?.geometryFile) copy.sources![path] = structuredClone(copy.geometry);
  }
  return copy;
}

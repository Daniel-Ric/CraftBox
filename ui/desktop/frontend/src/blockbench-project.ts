import type { MotionRuntime, PackObject } from "./bridge";
import { createBlockbenchMotion } from "./blockbench-motion";

type Data = Record<string, any>;
type Vector = [number, number, number];
type Outline = {
  uuid: string;
  isOpen: boolean;
  children: (Outline | string)[];
};
export type BlockbenchModel = {
  previewCamera?: { position: number[]; target: number[] };
  name: string;
  geometry: unknown;
  geometryId?: string;
  textures: Record<string, string>;
  textureNames?: Record<string, string>;
  object?: PackObject;
  motion?: MotionRuntime;
  animations?: Record<string, unknown>;
  sources?: Record<string, unknown>;
  warnings?: string[];
};
export type BlockbenchProjectInput = {
  name: string;
  models: BlockbenchModel[];
  format?: "auto" | "entity" | "block" | "free";
};

const object = (value: unknown): Data =>
  value && typeof value === "object" && !Array.isArray(value)
    ? (value as Data)
    : {};
const list = (value: unknown): any[] => (Array.isArray(value) ? value : []);
function vector(value: unknown, fallback: Vector = [0, 0, 0]): Vector {
  if (value === undefined) return [...fallback];
  if (
    !Array.isArray(value) ||
    value.length !== 3 ||
    !value.every(Number.isFinite)
  )
    throw Error("Model contains an invalid coordinate vector.");
  return [...value] as Vector;
}
const point = (value: unknown): Vector => {
  const v = vector(value);
  return [-v[0], v[1], v[2]];
};
const rotation = (value: unknown): Vector => {
  const v = vector(value);
  return [-v[0], -v[1], v[2]];
};
const id = () => crypto.randomUUID();
const faces = ["north", "east", "south", "west", "up", "down"];
function selectedGeometry(model: BlockbenchModel): Data {
  const raw = object(model.geometry);
  const modern = list(raw["minecraft:geometry"]);
  if (modern.length) {
    const match = model.geometryId
      ? modern.find(
          (geometry) => geometry.description?.identifier === model.geometryId,
        )
      : modern.length === 1
        ? modern[0]
        : undefined;
    if (!match)
      throw Error(
        "Select one authored geometry before exporting a Blockbench project.",
      );
    return object(match);
  }
  const keys = Object.keys(raw).filter((key) => key.startsWith("geometry."));
  const key = model.geometryId ?? (keys.length === 1 ? keys[0] : undefined);
  if (key && raw[key]) {
    const legacy = object(raw[key]);
    return {
      ...legacy,
      description: {
        identifier: key,
        texture_width: legacy.texturewidth ?? raw.texturewidth ?? 64,
        texture_height: legacy.textureheight ?? raw.textureheight ?? 64,
        visible_bounds_width: legacy.visible_bounds_width,
        visible_bounds_height: legacy.visible_bounds_height,
        visible_bounds_offset: legacy.visible_bounds_offset,
      },
    };
  }
  if (Array.isArray(raw.bones)) return raw;
  throw Error("The selected asset has no authored Bedrock geometry.");
}
function pngSize(source: string): [number, number] {
  if (!source.startsWith("data:image/png;base64,"))
    throw Error("Blockbench textures must be embedded PNG images.");
  const start = source.indexOf(",") + 1;
  const bytes = atob(source.slice(start, start + 44));
  if (bytes.length < 24 || bytes.slice(1, 4) !== "PNG")
    throw Error("Invalid model texture.");
  const read = (at: number) =>
    bytes.charCodeAt(at) * 0x1000000 +
    (bytes.charCodeAt(at + 1) << 16) +
    (bytes.charCodeAt(at + 2) << 8) +
    bytes.charCodeAt(at + 3);
  const size: [number, number] = [read(16), read(20)];
  if (size.some((n) => n < 1 || n > 16384))
    throw Error("Model texture dimensions exceed 16384 pixels.");
  return size;
}
function matches(pattern: string, name: string) {
  return new RegExp(
    "^" +
      pattern
        .split("*")
        .map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
        .join(".*") +
      "$",
    "i",
  ).test(name);
}
export function createBlockbenchProject(input: BlockbenchProjectInput) {
  if (!input.models.length || input.models.length > 64)
    throw Error("Select between 1 and 64 source models.");
  const warnings = new Set<string>();
  const project: Data = {
    meta: { format_version: "5.0", model_format: "bedrock", box_uv: true },
    name: input.name,
    model_identifier: "",
    resolution: { width: 64, height: 64 },
    elements: [],
    groups: [],
    outliner: [],
    textures: [],
    animations: [],
    animation_controllers: [],
    animation_variable_placeholders: "",
    export_options: {
      craftbox: {
        version: 2,
        motion: { animations: 0, controllers: 0, states: 0, keyframes: 0 },
        sources: structuredClone(input.models),
        warnings: [],
      },
    },
  };
  const textureKeys = new Map<string, number>();
  const usedTextures = new Set<number>();
  let hasMeshes = false;
  for (const model of input.models) {
    for (const warning of model.warnings ?? []) warnings.add(warning);
    const geometry = selectedGeometry(model);
    const description = object(geometry.description);
    const width = description.texture_width ?? 64,
      height = description.texture_height ?? 64;
    if (
      ![width, height].every((n) => Number.isFinite(n) && n > 0 && n <= 16384)
    )
      throw Error("Invalid model UV resolution.");
    if (!project.model_identifier) {
      project.model_identifier = String(
        description.identifier ?? model.name,
      ).replace(/^geometry\./, "");
      project.resolution = { width, height };
      if (description.visible_bounds_width !== undefined)
        project.visible_box = [
          description.visible_bounds_width,
          description.visible_bounds_height ?? 1,
          description.visible_bounds_offset?.[1] ?? 0,
        ];
      if (geometry.item_display_transforms)
        project.display = structuredClone(geometry.item_display_transforms);
    }
    const textureIndex = new Map<string, number>();
    for (const [alias, source] of Object.entries(model.textures)) {
      if (!source) continue;
      const mode =
        model.object?.materialModes?.[alias] ?? model.object?.material ?? "";
      const key = `${width}:${height}:${mode}:${source}`;
      let index = textureKeys.get(key);
      if (index === undefined) {
        const [pixelWidth, pixelHeight] = pngSize(source);
        index = Number(project.textures.length);
        project.textures.push({
          uuid: id(),
          id: String(index),
          name: (
            model.textureNames?.[alias]?.split(/[\\/]/).pop() ??
            `${model.name}_${alias === "*" ? "texture" : alias}.png`
          ).replace(/\.(?:jpe?g|tga)$/i, ".png"),
          source,
          internal: true,
          saved: false,
          width: pixelWidth,
          height: pixelHeight,
          uv_width: width,
          uv_height: height,
          mode: "bitmap",
          visible: true,
          render_mode: /emissive/.test(mode) ? "emissive" : "default",
          render_sides: /one_sided|single_sided/.test(mode)
            ? "front"
            : "double",
        });
        textureKeys.set(key, index);
      }
      textureIndex.set(alias, index);
    }
    const fallbackTexture =
      textureIndex.get("*") ??
      textureIndex.get("default") ??
      textureIndex.values().next().value;
    const textureFor = (alias: string) => {
      const selected = textureIndex.get(alias);
      if (selected !== undefined) {
        usedTextures.add(selected);
        return selected;
      }
      if (alias !== "*" && !textureIndex.has("*"))
        warnings.add(
          `Texture material ${alias} has no exact binding in ${model.name}; the default texture is shown.`,
        );
      if (fallbackTexture !== undefined) usedTextures.add(fallbackTexture);
      return fallbackTexture ?? false;
    };
    if (!textureIndex.size)
      warnings.add(`No texture was supplied for ${model.name}.`);
    const bones = list(geometry.bones);
    if (bones.length > 10000) throw Error("Model has more than 10000 bones.");
    const groups = new Map<string, Data>(),
      outlines = new Map<string, Outline>();
    const boneNames = new Set<string>();
    for (const bone of bones) {
      if (typeof bone.name !== "string" || !bone.name)
        throw Error("Model contains an unnamed bone.");
      const key = bone.name.toLowerCase();
      if (boneNames.has(key)) throw Error(`Duplicate bone name: ${bone.name}`);
      boneNames.add(key);
      let visible = bone.neverRender !== true;
      for (const rule of model.object?.visibility ?? [])
        if (matches(rule.pattern, bone.name))
          visible = rule.visible && bone.neverRender !== true;
      const group = {
        name: bone.name,
        uuid: id(),
        origin: point(bone.pivot),
        rotation: rotation(bone.rotation),
        color: project.groups.length % 8,
        visibility: visible,
        export: true,
        mirror_uv: bone.mirror === true,
        reset: bone.reset === true,
        bedrock_binding: bone.binding ?? "",
        material: bone.material ?? "",
      };
      groups.set(key, group);
      outlines.set(key, { uuid: group.uuid, isOpen: true, children: [] });
      project.groups.push(group);
    }
    let roots: (Outline | string)[] = project.outliner;
    if (input.models.length > 1) {
      const uuid = id();
      project.groups.push({
        name: model.name,
        uuid,
        origin: [0, 0, 0],
        rotation: [0, 0, 0],
        visibility: true,
        export: true,
      });
      const wrapper: Outline = { uuid, isOpen: true, children: [] };
      project.outliner.push(wrapper);
      roots = wrapper.children;
    }
    const byName = new Map(
      bones.map((bone) => [bone.name.toLowerCase(), bone]),
    );
    const checked = new Set<string>();
    for (const bone of bones) {
      const visited = new Set<string>();
      let current: any = bone;
      while (current && !checked.has(current.name.toLowerCase())) {
        const key = current.name.toLowerCase();
        if (visited.has(key))
          throw Error(`Cyclic bone hierarchy at ${bone.name}`);
        visited.add(key);
        current = byName.get(String(current.parent ?? "").toLowerCase());
      }
      visited.forEach((key) => checked.add(key));
      const outline = outlines.get(bone.name.toLowerCase())!;
      const parent = outlines.get(String(bone.parent ?? "").toLowerCase());
      if (bone.parent && !parent)
        warnings.add(
          `Parent bone ${bone.parent} is missing; ${bone.name} remains at the model root.`,
        );
      (parent?.children ?? roots).push(outline);
      const add = (element: Data) => {
        if (project.elements.length >= 50000)
          throw Error("Model has more than 50000 editable elements.");
        element.uuid = id();
        element.export = true;
        project.elements.push(element);
        outline.children.push(element.uuid);
      };
      for (const [cubeIndex, rawCube] of list(bone.cubes).entries()) {
        const cube = object(rawCube),
          origin = vector(cube.origin),
          size = vector(cube.size);
        const from: Vector = [-(origin[0] + size[0]), origin[1], origin[2]];
        const faceData: Data = {};
        const box = Array.isArray(cube.uv) || cube.uv === undefined;
        for (const face of faces) {
          const definition = object(object(cube.uv)[face]);
          if (!box && !object(cube.uv)[face]) {
            faceData[face] = { uv: [0, 0, 0, 0], texture: null };
            continue;
          }
          const uv = definition.uv ?? [0, 0];
          const dimensions =
            face === "up" || face === "down"
              ? [Math.abs(size[0]), Math.abs(size[2])]
              : face === "east" || face === "west"
                ? [Math.abs(size[2]), Math.abs(size[1])]
                : [Math.abs(size[0]), Math.abs(size[1])];
          const span = definition.uv_size ?? dimensions;
          if (![...uv, ...span].every(Number.isFinite))
            throw Error("Invalid cube UV coordinates.");
          let rectangle = [uv[0], uv[1], uv[0] + span[0], uv[1] + span[1]];
          if (face === "up" || face === "down")
            rectangle = [
              rectangle[2],
              rectangle[3],
              rectangle[0],
              rectangle[1],
            ];
          const alias = definition.material_instance ?? "*";
          faceData[face] = {
            uv: rectangle,
            texture: textureFor(alias),
            rotation: definition.uv_rotation ?? 0,
            material_name: alias,
          };
        }
        add({
          type: "cube",
          name: cube.name ?? `${bone.name} ${cubeIndex + 1}`,
          from,
          to: from.map((n, i) => n + size[i]),
          origin: point(cube.pivot),
          rotation: rotation(cube.rotation),
          inflate: cube.inflate ?? bone.inflate ?? 0,
          mirror_uv: cube.mirror ?? bone.mirror ?? false,
          box_uv: box,
          uv_offset: box ? (cube.uv ?? [0, 0]) : [0, 0],
          autouv: 0,
          color: groups.get(bone.name.toLowerCase())!.color,
          visibility: true,
          faces: faceData,
        });
      }
      for (const [name, rawLocator] of Object.entries(object(bone.locators))) {
        const locator = object(rawLocator),
          position = point(
            Array.isArray(rawLocator) ? rawLocator : locator.offset,
          );
        add({
          type:
            name.startsWith("_null_") && Array.isArray(rawLocator)
              ? "null_object"
              : "locator",
          name: name.startsWith("_null_") ? name.slice(6) : name,
          position,
          from: position,
          rotation: rotation(locator.rotation),
          ignore_inherited_scale: locator.ignore_inherited_scale === true,
          visibility: true,
        });
      }
      if (bone.poly_mesh) {
        hasMeshes = true;
        const poly = object(bone.poly_mesh),
          vertices: Data = {},
          meshFaces: Data = {};
        const positions = list(poly.positions),
          uvs = list(poly.uvs);
        if (positions.length > 200000)
          throw Error("Polygon mesh exceeds 200000 vertices.");
        let polygons = poly.polys;
        if (typeof polygons === "string") {
          const step =
            polygons === "tri_list" ? 3 : polygons === "quad_list" ? 4 : 0;
          if (!step) throw Error(`Unsupported polygon layout: ${polygons}`);
          polygons = Array.from(
            { length: Math.floor(positions.length / step) },
            (_, face) =>
              Array.from({ length: step }, (_, vertex) => {
                const index = face * step + vertex;
                return [index, index, index];
              }),
          );
        }
        for (const [faceIndex, polygon] of list(polygons).entries()) {
          if (list(polygon).length < 3)
            throw Error("Polygon mesh has an invalid face.");
          const keys: string[] = [],
            uv: Data = {};
          for (const corner of [...polygon].reverse()) {
            const positionIndex = corner[0],
              uvIndex = corner[2];
            if (!positions[positionIndex] || !uvs[uvIndex])
              throw Error("Polygon mesh has an invalid vertex reference.");
            const key = `v${positionIndex}`;
            vertices[key] = point(positions[positionIndex]);
            keys.push(key);
            const source = uvs[uvIndex];
            uv[key] = poly.normalized_uvs
              ? [source[0] * width, (1 - source[1]) * height]
              : [source[0], height - source[1]];
          }
          meshFaces[`f${faceIndex}`] = {
            vertices: keys,
            uv,
            texture: textureFor("*"),
          };
        }
        add({
          type: "mesh",
          name: `${bone.name} mesh`,
          origin: [0, 0, 0],
          rotation: [0, 0, 0],
          vertices,
          faces: meshFaces,
          visibility: true,
        });
      }
      for (const tm of list(bone.texture_meshes)) {
        const position = vector(tm.position),
          pivot = vector(tm.local_pivot);
        add({
          type: "texture_mesh",
          name: tm.texture ?? `${bone.name} texture mesh`,
          texture_name: tm.texture ?? "",
          texture: textureIndex.has(tm.texture)
            ? project.textures[textureIndex.get(tm.texture)!].uuid
            : fallbackTexture === undefined
              ? null
              : project.textures[fallbackTexture].uuid,
          origin: [
            -position[0],
            -position[1] + vector(bone.pivot)[1],
            position[2],
          ],
          local_pivot: [pivot[0], pivot[1], -pivot[2]],
          rotation: rotation(tm.rotation),
          scale: vector(tm.scale, [1, 1, 1]),
          visibility: true,
        });
      }
    }
    const motion = createBlockbenchMotion(model, groups, warnings);
    project.animations.push(...motion.animations);
    project.animation_controllers.push(...motion.controllers);
    project.animation_variable_placeholders = [
      ...new Set([
        ...project.animation_variable_placeholders.split("\n").filter(Boolean),
        ...motion.placeholders,
      ]),
    ].join("\n");
    const report = project.export_options.craftbox.motion;
    report.animations += motion.animations.length;
    report.controllers += motion.controllers.length;
    report.states += motion.states;
    report.keyframes += motion.keyframes;
    if (model.object?.transformation)
      warnings.add(
        "Block component transformations are retained in the original source data and are not baked into the authored geometry.",
      );
  }
  if (!project.groups.length && !project.elements.length)
    throw Error("The authored geometry contains no editable model nodes.");
  const target = input.format ?? "auto";
  const block =
    target === "block" ||
    (target === "auto" && input.models[0].object?.kind === "block");
  if (target === "entity" && (usedTextures.size > 1 || hasMeshes))
    throw Error(
      "Bedrock Entity projects use one active texture and cube geometry. Choose Generic Model to preserve multiple materials or polygon meshes.",
    );
  if (block && hasMeshes)
    throw Error(
      "Bedrock Block projects use cube geometry. Choose Generic Model to edit polygon meshes.",
    );
  project.meta.model_format = block
    ? "bedrock_block"
    : target === "free" || usedTextures.size > 1 || hasMeshes
      ? "free"
      : "bedrock";
  if (project.meta.model_format === "bedrock") {
    const active = usedTextures.values().next().value;
    if (active !== undefined) project.textures[active].use_as_default = true;
  }
  if (project.meta.model_format === "free" && usedTextures.size > 1)
    warnings.add(
      "Generic Model format preserves separate material textures. Convert a copy to Bedrock in Blockbench when a single-texture Minecraft export is needed.",
    );
  if (
    block &&
    (project.animations.length || project.animation_controllers.length)
  )
    warnings.add(
      "Bedrock Block has no Animate workspace in Blockbench. Animation data is retained; choose Generic Model to edit its animation tracks.",
    );
  if (
    project.meta.model_format === "free" &&
    project.animation_controllers.length
  )
    warnings.add(
      "Generic Model supports animation tracks but hides the controller editor. Controller data is retained; use Bedrock Entity to edit state diagrams when its texture and geometry constraints are met.",
    );
  project.meta.box_uv = project.elements.every(
    (element: Data) => element.type !== "cube" || element.box_uv,
  );
  project.export_options.craftbox.warnings = [...warnings];
  const camera = input.models.length === 1 ? input.models[0].previewCamera : undefined;
  if (camera) {
    const view = { ...camera, orthographic: false, zoom: 0.5, angle: null };
    project.editor_state = { mode: "edit", selected_elements: [], selected_groups: [], previews: { main: view, media: view, no_aa_media: view } };
  }
  const motion = project.export_options.craftbox.motion;
  if (
    motion.animations > 4096 ||
    motion.controllers > 1024 ||
    motion.states > 16384 ||
    motion.keyframes > 500000
  )
    throw Error("Project exceeds editable animation graph limits.");
  if (JSON.stringify(project).length > 72 * 1024 * 1024)
    throw Error("Blockbench project exceeds 72 MiB.");
  return { project, warnings: [...warnings] };
}

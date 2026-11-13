import * as T from "three";
import { molang, previewContext, type MolangContext } from "./asset-molang";
import { limitedAxes } from "./human-skeleton";

const planePixels = new WeakMap<object, ImageData | null>();
// Some exported models keep a transparent placeholder on the back of a plane.
// Only share the other face when the placeholder is provably fully transparent;
// authored, visible front/back UVs must remain independent.
function planeUV(
  uv: Cube["uv"],
  size: V3,
  materials: T.Material[],
  aliases: string[],
  width: number,
  height: number,
  mode?: string,
) {
  if (!uv || Array.isArray(uv) || size.filter((n) => n === 0).length !== 1)
    return uv;
  const pair = [
    ["east", "west"],
    ["up", "down"],
    ["north", "south"],
  ][size.indexOf(0)];
  const empty = (face: string): boolean | undefined => {
    const def = uv[face];
    if (!def?.uv_size) return undefined;
    const material = materials[
      Math.max(0, aliases.indexOf(def.material_instance ?? "*"))
    ] as T.MeshStandardMaterial;
    if (
      mode !== undefined
        ? mode === "opaque" || /one_sided|single_sided/.test(mode)
        : (!material?.transparent && !material?.alphaTest) ||
          material?.side !== T.DoubleSide
    )
      return undefined;
    const map = material?.map,
      image = map?.image as
        (CanvasImageSource & { width: number; height: number }) | undefined;
    if (
      !image ||
      map!.offset.lengthSq() ||
      map!.repeat.x !== 1 ||
      map!.repeat.y !== 1 ||
      map!.rotation
    )
      return undefined;
    if (!planePixels.has(image)) {
      let pixels: ImageData | null = null;
      try {
        if (
          image.width * image.height <= 4_194_304 &&
          typeof document !== "undefined"
        ) {
          const canvas = document.createElement("canvas");
          canvas.width = image.width;
          canvas.height = image.height;
          const ctx = canvas.getContext("2d", { willReadFrequently: true });
          if (ctx) {
            ctx.drawImage(image, 0, 0);
            pixels = ctx.getImageData(0, 0, canvas.width, canvas.height);
          }
        }
      } catch {
        /* An unreadable image cannot prove a transparent face. */
      }
      planePixels.set(image, pixels);
    }
    const pixels = planePixels.get(image);
    if (!pixels) return undefined;
    const [u, v] = def.uv,
      [w, h] = def.uv_size;
    if (
      ![u, v, w, h, width, height].every(Number.isFinite) ||
      width <= 0 ||
      height <= 0
    )
      return undefined;
    const x0 = Math.floor((Math.min(u, u + w) * pixels.width) / width),
      x1 = Math.ceil((Math.max(u, u + w) * pixels.width) / width);
    const y0 = Math.floor((Math.min(v, v + h) * pixels.height) / height),
      y1 = Math.ceil((Math.max(v, v + h) * pixels.height) / height);
    if (
      x0 < 0 ||
      y0 < 0 ||
      x1 > pixels.width ||
      y1 > pixels.height ||
      x1 <= x0 ||
      y1 <= y0
    )
      return undefined;
    for (let y = y0; y < y1; y++)
      for (let x = x0; x < x1; x++)
        if (pixels.data[(y * pixels.width + x) * 4 + 3] !== 0) return false;
    return true;
  };
  const states = pair.map(empty);
  const blank =
    states[0] === true && states[1] === false
      ? 0
      : states[1] === true && states[0] === false
        ? 1
        : -1;
  if (blank < 0) return uv;
  const result = { ...uv };
  delete result[pair[blank]];
  return result;
}

type V3 = [number, number, number];
// Bedrock's authored model frame differs from the Three.js/Blockbench frame.
// Apply this at import, not as a negative scale on the preview/export root.
export function bedrockVector(v: readonly number[]) {
  return new T.Vector3(-v[0], v[1], v[2]);
}
function bedrockRotation(v: readonly number[]) {
  return new T.Euler(
    -T.MathUtils.degToRad(v[0]),
    -T.MathUtils.degToRad(v[1]),
    T.MathUtils.degToRad(v[2]),
    "ZYX",
  );
}
function reverseWinding(geometry: T.BufferGeometry) {
  if (!geometry.index)
    geometry.setIndex(
      Array.from({ length: geometry.attributes.position.count }, (_, i) => i),
    );
  const index = geometry.index!;
  for (let i = 0; i < index.count; i += 3) {
    const b = index.getX(i + 1);
    index.setX(i + 1, index.getX(i + 2));
    index.setX(i + 2, b);
  }
  index.needsUpdate = true;
}
type UVFace = {
  uv: [number, number];
  uv_size?: [number, number];
  uv_rotation?: number;
  material_instance?: string;
};
type Cube = {
  origin: V3;
  size: V3;
  uv?: [number, number] | Record<string, UVFace>;
  inflate?: number;
  pivot?: V3;
  rotation?: V3;
  mirror?: boolean;
};
type Bone = {
  name: string;
  binding?: string;
  locators?: Record<string, V3 | { offset?: V3; rotation?: V3 }>;
  parent?: string;
  pivot?: V3;
  rotation?: V3;
  cubes?: Cube[];
  mirror?: boolean;
  inflate?: number;
  neverRender?: boolean;
  poly_mesh?: {
    normalized_uvs?: boolean;
    positions: V3[];
    normals: V3[];
    uvs: [number, number][];
    polys: number[][][] | string;
  };
};
export type SkinPose = {
  head: number;
  leftArm: number;
  rightArm: number;
  leftLeg: number;
  rightLeg: number;
  axes?: Record<string, V3>;
  positions?: Record<string, V3>;
  targets?: Record<string, V3>;
  poles?: Record<string, V3>;
  limits?: boolean;
  cosmetics?: Record<string, CosmeticBinding>;
};
export type CosmeticBinding = {
  binding?: string;
  followPosition?: boolean;
  followRotation?: boolean;
  followScale?: boolean;
  influence?: number;
  visible?: boolean;
};
export const neutralPose: SkinPose = {
  head: 0,
  leftArm: 0,
  rightArm: 0,
  leftLeg: 0,
  rightLeg: 0,
};

function boxUV(
  g: T.BoxGeometry,
  uv: Cube["uv"],
  size: V3,
  width: number,
  height: number,
  mirror = false,
  aliases: string[] = ["*"],
) {
  const [w, h, d] = size;
  const [u, v] = Array.isArray(uv) ? uv : [0, 0];
  const rects =
    Array.isArray(uv) || !uv
      ? [
          [u + d + w, v + d, d, h],
          [u, v + d, d, h],
          [u + d, v, w, d],
          [u + d + w, v, w, d],
          [u + d + w + d, v + d, w, h],
          [u + d, v + d, w, h],
        ]
      : [];
  const box = Array.isArray(uv) || !uv;
  // A box-UV plane has one shared texture, not a second strip from the
  // unfolded cuboid layout. Emit one quad so transparent materials are not
  // drawn twice and its back uses the same UVs. Use effective dimensions:
  // inflation can turn a nominally flat cube into a real volume.
  const flatAxes = [g.parameters.width, g.parameters.height, g.parameters.depth]
    .map((n, axis) => (n === 0 ? axis : -1))
    .filter((axis) => axis >= 0);
  const planarBoxFace =
    box && flatAxes.length === 1 ? [1, 2, 5][flatAxes[0]] : undefined;
  const faces = ["west", "east", "up", "down", "south", "north"];
  const out = g.attributes.uv;
  const original = g.index!,
    visibleIndices: number[] = [];
  g.clearGroups();
  for (let face = 0; face < 6; face++) {
    if (planarBoxFace !== undefined && face !== planarBoxFace) continue;
    if (flatAxes.length === 1 && Math.floor(face / 2) !== flatAxes[0]) continue;
    const faceDefinition =
      !Array.isArray(uv) && uv ? uv[faces[face]] : undefined;
    if (uv && !Array.isArray(uv) && !faceDefinition) continue;
    let rect = rects[mirror && face < 2 ? 1 - face : face];
    if (!rect) {
      const value = (uv as Record<string, UVFace>)[faces[face]];
      if (value)
        rect = [
          ...value.uv,
          ...(value.uv_size ??
            (face < 2 ? [d, h] : face < 4 ? [w, d] : [w, h])),
        ];
      else rect = [0, 0, 0, 0];
    }
    let [x, y, rw, rh] = rect;
    if (mirror && box) {
      x += rw;
      rw = -rw;
    }
    const rotation = (((faceDefinition?.uv_rotation ?? 0) % 360) + 360) % 360;
    const points = [
      [0, 0],
      [1, 0],
      [0, 1],
      [1, 1],
    ].map(([s, t]) => {
      for (let step = 0; step < rotation / 90; step++) [s, t] = [1 - t, s];
      if (face < 2 || face > 3) s = 1 - s;
      else if (face === 2 || !box) t = 1 - t;
      return [x + s * rw, y + t * rh];
    });
    const alias = faceDefinition?.material_instance ?? "*";
    g.addGroup(visibleIndices.length, 6, Math.max(0, aliases.indexOf(alias)));
    for (let i = face * 6; i < face * 6 + 6; i++)
      visibleIndices.push(original.getX(i));
    points.forEach(([px, py], i) =>
      out.setXY(face * 4 + i, px / width, 1 - py / height),
    );
  }
  out.needsUpdate = true;
  g.setIndex(visibleIndices);
}

export function bedrockModel(
  raw: unknown,
  material: T.Material | Record<string, T.Material>,
  boneMaterials?: { pattern: string; material: string }[] | null,
): T.Group {
  const aliases =
      material instanceof T.Material ? ["*"] : Object.keys(material),
    materials =
      material instanceof T.Material ? [material] : Object.values(material);
  const input = raw as Record<string, unknown>;
  let definition: Record<string, unknown> | undefined;
  const geometries = input["minecraft:geometry"];
  if (Array.isArray(geometries))
    definition = geometries[0] as Record<string, unknown>;
  if (!definition)
    definition = Object.entries(input).find(
      ([key, value]) =>
        key.startsWith("geometry.") && value && typeof value === "object",
    )?.[1] as Record<string, unknown> | undefined;
  if (!definition || !Array.isArray(definition.bones))
    throw new Error(
      "This file does not contain supported Bedrock cube geometry.",
    );
  const description = (definition.description ?? definition) as Record<
    string,
    number
  >;
  const width = description.texture_width ?? description.texturewidth ?? 64,
    height = description.texture_height ?? description.textureheight ?? 64;
  if (width < 1 || height < 1 || width > 8192 || height > 8192)
    throw new Error("Invalid geometry texture size.");
  const bones = definition.bones as Bone[];
  if (bones.length > 512) throw new Error("Model exceeds 512 bones.");
  let cubes = 0;
  const root = new T.Group(),
    groups = new Map<string, T.Group>(),
    pivots = new Map<string, T.Vector3>();
  for (const bone of bones) {
    if (groups.has(bone.name))
      throw new Error("Model contains duplicate bone names.");
    const g = new T.Group();
    g.name = bone.name;
    groups.set(bone.name, g);
    pivots.set(bone.name, bedrockVector(bone.pivot ?? [0, 0, 0]));
  }
  for (const bone of bones) {
    const g = groups.get(bone.name)!;
    const pivot = pivots.get(bone.name)!;
    const visited = new Set([bone.name]);
    let parentName = bone.parent;
    while (parentName) {
      if (visited.has(parentName))
        throw new Error("Model contains a parent cycle.");
      visited.add(parentName);
      parentName = bones.find((b) => b.name === parentName)?.parent;
    }
    const parent = bone.parent ? groups.get(bone.parent) : undefined;
    g.position.copy(pivot);
    if (parent) {
      g.position.sub(pivots.get(bone.parent!)!);
      parent.add(g);
    } else root.add(g);
    if (bone.rotation) g.rotation.copy(bedrockRotation(bone.rotation));
    g.userData.baseRotation = g.rotation.clone();
    g.userData.basePosition = g.position.clone();
    g.userData.binding = bone.binding;
    for (const [name, value] of Object.entries(bone.locators ?? {})) {
      const locator = new T.Object3D();
      locator.name = "locator:" + name;
      const offset = Array.isArray(value) ? value : (value.offset ?? [0, 0, 0]);
      locator.position.copy(bedrockVector(offset)).sub(pivot);
      if (!Array.isArray(value) && value.rotation)
        locator.rotation.copy(bedrockRotation(value.rotation));
      g.add(locator);
    }
    for (const cube of bone.neverRender ? [] : (bone.cubes ?? [])) {
      if (++cubes > 12000) throw new Error("Model exceeds 12,000 cubes.");
      if (
        !cube.size ||
        !cube.origin ||
        [...cube.size, ...cube.origin].some(
          (n) => !Number.isFinite(n) || Math.abs(n) > 100000,
        )
      )
        throw new Error("Invalid cube coordinates.");
      const inflate = cube.inflate ?? bone.inflate ?? 0;
      const effectiveSize = cube.size.map((n) =>
        n < 0
          ? -Math.max(0, Math.abs(n) + inflate * 2)
          : Math.max(0, n + inflate * 2),
      ) as V3;
      const geometry = new T.BoxGeometry(
        ...(effectiveSize.map(Math.abs) as V3),
      );
      const mode = [...(boneMaterials ?? [])].reverse().find((rule) =>
        new RegExp(
          "^" +
            rule.pattern
              .split("*")
              .map((v) => v.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
              .join(".*") +
            "$",
          "i",
        ).test(bone.name),
      )?.material;
      const uv = planeUV(
        cube.uv,
        effectiveSize,
        materials,
        aliases,
        width,
        height,
        mode,
      );
      boxUV(
        geometry,
        uv,
        cube.size,
        width,
        height,
        cube.mirror ?? bone.mirror,
        aliases,
      );
      // Reflect the Bedrock coordinate system while preserving signed-size
      // winding. Odd-negative cubes are authored inward-facing shells (often
      // used around textured letters), not outward opaque enclosing boxes.
      const signs = effectiveSize.map((n) => (n < 0 ? -1 : 1));
      geometry.scale(-signs[0], signs[1], signs[2]);
      reverseWinding(geometry);
      if (signs[0] * signs[1] * signs[2] < 0) {
        const normals = geometry.getAttribute("normal");
        for (let i = 0; i < normals.count; i++)
          normals.setXYZ(
            i,
            -normals.getX(i),
            -normals.getY(i),
            -normals.getZ(i),
          );
        normals.needsUpdate = true;
      }
      const mesh = new T.Mesh(
        geometry,
        materials.length === 1 ? materials[0] : materials,
      );
      const pairs = [
        ["east", "west"],
        ["up", "down"],
        ["north", "south"],
      ];
      const inwardShell = signs[0] * signs[1] * signs[2] < 0;
      if (
        inwardShell ||
        (uv &&
          !Array.isArray(uv) &&
          [
            geometry.parameters.width,
            geometry.parameters.height,
            geometry.parameters.depth,
          ].some(
            (n, axis) =>
              n === 0 &&
              pairs[axis].every(
                (face) => !!(uv as Record<string, UVFace>)[face],
              ),
          ))
      ) {
        const sided = materials.map((m) => {
          const clone = m.clone();
          clone.side = T.FrontSide;
          return clone;
        });
        mesh.material = sided.length === 1 ? sided[0] : sided;
        mesh.userData.authoredCulling = true;
      }
      mesh.name = bone.name;
      mesh.position
        .copy(bedrockVector(cube.origin))
        .add(bedrockVector(cube.size).multiplyScalar(0.5))
        .sub(pivot);
      mesh.userData.layer =
        bone.name.toLowerCase().includes("layer") ||
        [
          "hat",
          "jacket",
          "leftSleeve",
          "rightSleeve",
          "leftPants",
          "rightPants",
        ].includes(bone.name);
      if (cube.rotation) {
        const rotationPivot = cube.pivot
          ? bedrockVector(cube.pivot).sub(pivot)
          : mesh.position.clone();
        const group = new T.Group();
        group.position.copy(rotationPivot);
        mesh.position.sub(rotationPivot);
        group.rotation.copy(bedrockRotation(cube.rotation));
        group.add(mesh);
        g.add(group);
      } else g.add(mesh);
    }
    if (bone.poly_mesh && !bone.neverRender) {
      const poly = bone.poly_mesh;
      if (typeof poly.polys === "string")
        throw new Error("This legacy polygon list layout is not supported.");
      const positions: number[] = [],
        normals: number[] = [],
        uvs: number[] = [];
      if (poly.positions.length > 200000 || poly.polys.length > 60000)
        throw new Error("Polygon geometry exceeds the preview limit.");
      for (const polygon of poly.polys) {
        if (polygon.length < 3 || polygon.length > 8)
          throw new Error("Invalid polygon.");
        for (let tri = 1; tri < polygon.length - 1; tri++) {
          for (const vertex of [polygon[0], polygon[tri], polygon[tri + 1]]) {
            const point = poly.positions[vertex[0]],
              normal = poly.normals[vertex[1]],
              uv = poly.uvs[vertex[2]];
            if (
              !point ||
              !normal ||
              !uv ||
              [...point, ...normal, ...uv].some((v) => !Number.isFinite(v))
            )
              throw new Error("Invalid polygon vertex reference.");
            positions.push(
              -point[0] - pivot.x,
              point[1] - pivot.y,
              point[2] - pivot.z,
            );
            normals.push(-normal[0], normal[1], normal[2]);
            uvs.push(
              poly.normalized_uvs ? uv[0] : uv[0] / width,
              poly.normalized_uvs ? uv[1] : uv[1] / height,
            );
          }
        }
      }
      const geometry = new T.BufferGeometry();
      geometry.setAttribute(
        "position",
        new T.Float32BufferAttribute(positions, 3),
      );
      geometry.setAttribute("normal", new T.Float32BufferAttribute(normals, 3));
      geometry.setAttribute("uv", new T.Float32BufferAttribute(uvs, 2));
      geometry.setIndex(
        Array.from({ length: positions.length / 3 }, (_, i) => i),
      );
      reverseWinding(geometry);
      const mesh = new T.Mesh(geometry, materials[0]);
      mesh.name = bone.name;
      mesh.userData.layer = [
        "hat",
        "jacket",
        "leftsleeve",
        "rightsleeve",
        "leftpants",
        "rightpants",
      ].includes(bone.name.toLowerCase());
      g.add(mesh);
    }
  }
  return root;
}

export function classicSkin(
  material: T.Material,
  slim: boolean,
  legacy: boolean,
): T.Group {
  const arm = slim ? 3 : 4;
  const armY = slim ? 11.5 : 12;
  const armPivotY = slim ? 21.5 : 22;
  const bones: Bone[] = [
    {
      name: "head",
      pivot: [0, 24, 0],
      cubes: [{ origin: [-4, 24, -4], size: [8, 8, 8], uv: [0, 0] }],
    },
    {
      name: "body",
      pivot: [0, 24, 0],
      cubes: [{ origin: [-4, 12, -2], size: [8, 12, 4], uv: [16, 16] }],
    },
    {
      name: "rightArm",
      pivot: [-5, armPivotY, 0],
      cubes: [
        { origin: [-4 - arm, armY, -2], size: [arm, 12, 4], uv: [40, 16] },
      ],
    },
    {
      name: "leftArm",
      pivot: [5, armPivotY, 0],
      cubes: [
        {
          origin: [4, armY, -2],
          size: [arm, 12, 4],
          uv: legacy ? [40, 16] : [32, 48],
          mirror: legacy,
        },
      ],
    },
    {
      name: "rightLeg",
      pivot: [-2, 12, 0],
      cubes: [{ origin: [-4, 0, -2], size: [4, 12, 4], uv: [0, 16] }],
    },
    {
      name: "leftLeg",
      pivot: [2, 12, 0],
      cubes: [
        {
          origin: [0, 0, -2],
          size: [4, 12, 4],
          uv: legacy ? [0, 16] : [16, 48],
          mirror: legacy,
        },
      ],
    },
    {
      name: "headLayer",
      parent: "head",
      pivot: [0, 24, 0],
      cubes: [
        { origin: [-4, 24, -4], size: [8, 8, 8], inflate: 0.5, uv: [32, 0] },
      ],
    },
  ];
  if (!legacy)
    bones.push(
      {
        name: "bodyLayer",
        parent: "body",
        pivot: [0, 24, 0],
        cubes: [
          {
            origin: [-4, 12, -2],
            size: [8, 12, 4],
            inflate: 0.25,
            uv: [16, 32],
          },
        ],
      },
      {
        name: "rightArmLayer",
        parent: "rightArm",
        pivot: [-5, armPivotY, 0],
        cubes: [
          {
            origin: [-4 - arm, armY, -2],
            size: [arm, 12, 4],
            inflate: 0.25,
            uv: [40, 32],
          },
        ],
      },
      {
        name: "leftArmLayer",
        parent: "leftArm",
        pivot: [5, armPivotY, 0],
        cubes: [
          {
            origin: [4, armY, -2],
            size: [arm, 12, 4],
            inflate: 0.25,
            uv: [48, 48],
          },
        ],
      },
      {
        name: "rightLegLayer",
        parent: "rightLeg",
        pivot: [-2, 12, 0],
        cubes: [
          { origin: [-4, 0, -2], size: [4, 12, 4], inflate: 0.25, uv: [0, 32] },
        ],
      },
      {
        name: "leftLegLayer",
        parent: "leftLeg",
        pivot: [2, 12, 0],
        cubes: [
          { origin: [0, 0, -2], size: [4, 12, 4], inflate: 0.25, uv: [0, 48] },
        ],
      },
    );
  return bedrockModel(
    {
      "minecraft:geometry": [
        {
          description: { texture_width: 64, texture_height: legacy ? 32 : 64 },
          bones,
        },
      ],
    },
    material,
  );
}

const humanPoseKeys = {
  head: "head",
  leftarm: "leftArm",
  rightarm: "rightArm",
  leftleg: "leftLeg",
  rightleg: "rightLeg",
} as const;
export function poseSkin(
  root: T.Group,
  pose: SkinPose,
  time: number,
  walking = false,
) {
  root.traverse((g) => {
    if (!(g instanceof T.Group) || !g.userData.baseRotation) return;
    const name = g.name;
    const base = g.userData.baseRotation as T.Euler;
    g.rotation.copy(base);
    if (g.userData.basePosition) g.position.copy(g.userData.basePosition);
    g.scale.setScalar(1);
    if (g.userData.cosmetic && !g.userData.editableCosmetic) return;
    const position = pose.positions?.[name];
    if (position) {
      g.position.x += position[0];
      g.position.y += position[1];
      g.position.z += position[2];
    }
    const axes = pose.axes?.[name];
    if (axes) {
      const value = limitedAxes(name, axes, pose.limits !== false);
      g.rotation.x += T.MathUtils.degToRad(value[0]);
      g.rotation.y += T.MathUtils.degToRad(value[1]);
      g.rotation.z += T.MathUtils.degToRad(value[2]);
    }
    const control =
      humanPoseKeys[name.toLowerCase() as keyof typeof humanPoseKeys];
    if (!control || g.userData.cosmetic) return;
    const swing =
      walking && control !== "head"
        ? Math.sin(time * 5) *
          (control === "leftArm" || control === "rightLeg" ? 0.65 : -0.65)
        : 0;
    g.rotation.x += T.MathUtils.degToRad(pose[control]) + swing;
  });
}

export function selectSkinGeometry(raw: unknown, identifier?: string): unknown {
  const input = raw as Record<string, any>;
  const definitions: { id: string; value: any }[] = [];
  if (Array.isArray(input["minecraft:geometry"]))
    for (const value of input["minecraft:geometry"])
      definitions.push({ id: value.description?.identifier ?? "", value });
  for (const [id, value] of Object.entries(input))
    if (id.startsWith("geometry.") && value && typeof value === "object")
      definitions.push({ id, value });
  const chosen =
    definitions.find(
      (d) => d.id === identifier || d.id.split(":")[0] === identifier,
    ) ??
    (!identifier
      ? definitions.find(
          (d) =>
            !/cape/i.test(d.id) &&
            d.value.bones?.some((b: Bone) => b.name === "head"),
        )
      : undefined);
  if (!chosen) return undefined;
  const resolve = (entry: typeof chosen, seen: Set<string>): any => {
    if (seen.has(entry.id))
      throw new Error("Skin geometry inheritance contains a cycle.");
    seen.add(entry.id);
    let bones: Bone[] = [];
    let inherited: Record<string, any> = {};
    for (const parent of entry.id.split(":").slice(1)) {
      const base = definitions.find((d) => d.id.split(":")[0] === parent);
      if (base) {
        inherited = { ...inherited, ...resolve(base, seen) };
        bones = inherited.bones ?? [];
      }
    }
    const merged = new Map(bones.map((b) => [b.name, b]));
    for (const bone of entry.value.bones ?? [])
      merged.set(bone.name, { ...merged.get(bone.name), ...bone });
    seen.delete(entry.id);
    return {
      ...inherited,
      ...entry.value,
      ...(inherited.description || entry.value.description
        ? {
            description: {
              ...inherited.description,
              ...entry.value.description,
            },
          }
        : {}),
      bones: [...merged.values()],
    };
  };
  return { "minecraft:geometry": [resolve(chosen, new Set())] };
}

export function bakeModel(root: T.Group): T.Group {
  root.updateMatrixWorld(true);
  const out = new T.Group();
  root.traverse((object) => {
    if (
      !(object instanceof T.Mesh) ||
      !object.visible ||
      object.userData.previewOnly
    )
      return;
    let parent = object.parent;
    while (parent) {
      if (!parent.visible) return;
      parent = parent.parent;
    }
    const geometry: T.BufferGeometry = object.geometry.clone();
    geometry.applyMatrix4(object.matrixWorld);
    if (object.matrixWorld.determinant() < 0) reverseWinding(geometry);
    if (geometry.index && geometry.groups.length) {
      const original = geometry.index,
        indices: number[] = [],
        groups = geometry.groups.map((group) => ({ ...group }));
      geometry.clearGroups();
      for (const group of groups) {
        const start = indices.length;
        for (let i = group.start; i < group.start + group.count; i++)
          indices.push(original.getX(i));
        geometry.addGroup(start, group.count, group.materialIndex);
      }
      geometry.setIndex(indices);
    }
    const material = Array.isArray(object.material)
      ? object.material.map((m) => m.clone())
      : object.material.clone();
    const mesh = new T.Mesh(geometry, material);
    mesh.name = object.name;
    out.add(mesh);
  });
  return out;
}

export function releaseModel(root: T.Object3D) {
  root.traverse((o) => {
    if (o instanceof T.Mesh) {
      o.geometry.dispose();
      for (const m of Array.isArray(o.material) ? o.material : [o.material])
        m.dispose();
    }
  });
}

const animationBones = new WeakMap<T.Group, Map<string, T.Object3D>>();
const animationFrames = new WeakMap<
  object,
  { time: number; value: unknown }[]
>();
function animationBone(root: T.Group, name: string) {
  let bones = animationBones.get(root);
  if (!bones) {
    bones = new Map();
    root.traverse((node) => {
      if (!bones!.has(node.name)) bones!.set(node.name, node);
    });
    animationBones.set(root, bones);
  }
  const cached = bones.get(name);
  for (let node = cached; node; node = node.parent ?? undefined)
    if (node === root) return cached;
  const found = root.getObjectByName(name);
  if (found) bones.set(name, found);
  else bones.delete(name);
  return found;
}
function vectorInput(value: unknown): boolean {
  if (typeof value === "number" || typeof value === "string") return true;
  if (Array.isArray(value))
    return (
      value.length === 3 &&
      value.every((v) => typeof v === "number" || typeof v === "string")
    );
  if (value && typeof value === "object") {
    const v = value as { post?: unknown; pre?: unknown };
    return vectorInput(v.post ?? v.pre);
  }
  return false;
}
export function poseAnimation(
  root: T.Group,
  raw: unknown,
  name: string,
  time: number,
  context: MolangContext = previewContext(time),
  weight = 1,
  additive = false,
) {
  const animations = (
    raw as {
      animations?: Record<
        string,
        {
          animation_length?: number;
          loop?: boolean | string;
          bones?: Record<string, Record<string, unknown>>;
        }
      >;
    }
  )?.animations;
  const animation = animations?.[name];
  if (!animation) return;
  const length = Number(animation.animation_length) || 1;
  const t =
    animation.loop === true
      ? time % Math.max(0.01, length)
      : Math.min(Math.max(0, time), length);
  function vector(value: unknown): V3 | undefined {
    if (typeof value === "number" || typeof value === "string") {
      const n = molang(value, context);
      return [n, n, n];
    }
    if (
      Array.isArray(value) &&
      value.length === 3 &&
      value.every((v) => typeof v === "number" || typeof v === "string")
    )
      return value.map((v) => molang(v, context)) as V3;
    if (value && typeof value === "object") {
      const v = value as { post?: unknown; pre?: unknown };
      return vector(v.post ?? v.pre);
    }
    return undefined;
  }
  function sample(value: unknown): V3 | undefined {
    const fixed = vector(value);
    if (fixed) return fixed;
    if (!value || typeof value !== "object") return;
    let keys = animationFrames.get(value);
    if (!keys) {
      keys = Object.entries(value)
        .map(([key, value]) => ({ time: Number(key), value }))
        .filter((v) => Number.isFinite(v.time) && vectorInput(v.value))
        .sort((a, b) => a.time - b.time);
      animationFrames.set(value, keys);
    }
    if (!keys.length) return;
    let lo = 0,
      hi = keys.length;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (keys[mid].time <= t) lo = mid + 1;
      else hi = mid;
    }
    const before = keys[Math.max(0, lo - 1)],
      after = keys[Math.min(lo, keys.length - 1)],
      alpha =
        after.time === before.time
          ? 0
          : (t - before.time) / (after.time - before.time);
    const a = vector(before.value)!,
      b = vector(after.value)!;
    return a.map((v, i) => v + (b[i] - v) * alpha) as V3;
  }
  for (const [bone, channels] of Object.entries(animation.bones ?? {})) {
    const object = animationBone(root, bone);
    if (!object) continue;
    const base = object.userData.baseRotation as T.Euler | undefined;
    if (base && !additive) object.rotation.copy(base);
    const position = object.userData.basePosition as T.Vector3 | undefined;
    if (position && !additive) object.position.copy(position);
    if (!additive) object.scale.setScalar(1);
    for (const [channel, value] of Object.entries(channels)) {
      const sampled = sample(value);
      if (!sampled) continue;
      if (channel === "rotation") {
        object.rotation.x -= T.MathUtils.degToRad(sampled[0]) * weight;
        object.rotation.y -= T.MathUtils.degToRad(sampled[1]) * weight;
        object.rotation.z += T.MathUtils.degToRad(sampled[2]) * weight;
      } else if (channel === "position")
        object.position.add(bedrockVector(sampled).multiplyScalar(weight));
      else if (channel === "scale")
        object.scale.multiply(
          new T.Vector3(...sampled)
            .subScalar(1)
            .multiplyScalar(weight)
            .addScalar(1),
        );
    }
  }
}

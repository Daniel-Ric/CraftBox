import * as T from "three";
import type { SkinPose } from "./asset-geometry";

type Vec3 = [number, number, number];
export type RigPart = {
  id: string;
  label: string;
  parent?: string;
  kind: "bone" | "joint" | "target" | "pole" | "cosmetic";
  limits: [Vec3, Vec3];
  available?: boolean;
};
type Limb = {
  name: string;
  upper: T.Group;
  elbow: T.Group;
  wrist: T.Group;
  length: number;
  target: string;
  pole: string;
  native: boolean;
  upperLength: number;
  lowerLength: number;
  upperDirection: T.Vector3;
  lowerDirection: T.Vector3;
  ikSupported: boolean;
};
type Deform = {
  cosmeticKey?:string;
  mesh: T.Mesh;
  toLimb: T.Matrix4;
  toMesh: T.Matrix4;
  positions: Float32Array;
  normals: Float32Array;
  limb: Limb;
  key: string;
};
const radians = T.MathUtils.degToRad;
const normalized = (name: string) => name.toLowerCase().replace(/[^a-z]/g, "");
const full: [Vec3, Vec3] = [
  [-180, -180, -180],
  [180, 180, 180],
];
export function jointLimits(name: string): [Vec3, Vec3] {
  if (/elbow/i.test(name))
    return [
      [0, -12, -12],
      [155, 12, 12],
    ];
  if (/knee/i.test(name))
    return [
      [-155, -8, -8],
      [0, 8, 8],
    ];
  if (/wrist/i.test(name))
    return [
      [-80, -90, -40],
      [80, 90, 40],
    ];
  if (/ankle/i.test(name))
    return [
      [-45, -35, -25],
      [65, 35, 25],
    ];
  if (/^head$/i.test(name))
    return [
      [-60, -85, -45],
      [60, 85, 45],
    ];
  if (/^body$/i.test(name))
    return [
      [-45, -60, -35],
      [45, 60, 35],
    ];
  if (/^(left|right)Leg$/i.test(name))
    return [
      [-45, -60, -60],
      [120, 60, 60],
    ];
  return full;
}
export function limitedAxes(name: string, axes: Vec3, enabled = true): Vec3 {
  const limits = jointLimits(name);
  return axes.map((v, i) =>
    enabled
      ? T.MathUtils.clamp(v, limits[0][i], limits[1][i])
      : T.MathUtils.clamp(v, -360, 360),
  ) as Vec3;
}

export function solveTwoBone(
  origin: T.Vector3,
  target: T.Vector3,
  pole: T.Vector3,
  upper: number,
  lower: number,
) {
  const direction = target.clone().sub(origin),
    raw = direction.length();
  if (raw < 1e-8) direction.set(0, -1, 0);
  else direction.divideScalar(raw);
  const distance = T.MathUtils.clamp(
    raw,
    Math.abs(upper - lower) + 1e-5,
    upper + lower - 1e-5,
  );
  const bend = pole
    .clone()
    .sub(origin)
    .addScaledVector(direction, -pole.clone().sub(origin).dot(direction));
  if (bend.lengthSq() < 1e-8) {
    bend.crossVectors(direction, new T.Vector3(0, 0, 1));
    if (bend.lengthSq() < 1e-8)
      bend.crossVectors(direction, new T.Vector3(1, 0, 0));
  }
  bend.normalize();
  const along =
    (upper * upper - lower * lower + distance * distance) / (2 * distance);
  return {
    elbow: origin
      .clone()
      .addScaledVector(direction, along)
      .addScaledVector(
        bend,
        Math.sqrt(Math.max(0, upper * upper - along * along)),
      ),
    end: origin.clone().addScaledVector(direction, distance),
    clamped: Math.abs(distance - raw) > 1e-4,
  };
}

type Vertex = { p: T.Vector3; n: T.Vector3; uv: T.Vector2; y: number };
function subdivide(
  geometry: T.BufferGeometry,
  toLimb: T.Matrix4,
  cuts: number[],
): T.BufferGeometry {
  const position = geometry.getAttribute("position"),
    normal = geometry.getAttribute("normal"),
    uv = geometry.getAttribute("uv"),
    index = geometry.index;
  const outP: number[] = [],
    outN: number[] = [],
    outUV: number[] = [],
    groups: { start: number; count: number; materialIndex: number }[] = [];
  const mix = (a: Vertex, b: Vertex, t: number): Vertex => ({
    p: a.p.clone().lerp(b.p, t),
    n: a.n.clone().lerp(b.n, t).normalize(),
    uv: a.uv.clone().lerp(b.uv, t),
    y: a.y + (b.y - a.y) * t,
  });
  function clip(poly: Vertex[], cut: number, sign: number) {
    const result: Vertex[] = [];
    for (let i = 0; i < poly.length; i++) {
      const a = poly[i],
        b = poly[(i + 1) % poly.length],
        inside = (a.y - cut) * sign >= -1e-6,
        next = (b.y - cut) * sign >= -1e-6;
      if (inside) result.push(a);
      if (inside !== next && Math.abs(b.y - a.y) > 1e-8)
        result.push(mix(a, b, (cut - a.y) / (b.y - a.y)));
    }
    return result;
  }
  const count = index?.count ?? position.count;
  for (let i = 0; i < count; i += 3) {
    const triangle: Vertex[] = [];
    for (let k = 0; k < 3; k++) {
      const at = index ? index.getX(i + k) : i + k,
        p = new T.Vector3().fromBufferAttribute(position, at);
      triangle.push({
        p,
        n: normal
          ? new T.Vector3().fromBufferAttribute(normal, at)
          : new T.Vector3(0, 0, 1),
        uv: uv ? new T.Vector2(uv.getX(at), uv.getY(at)) : new T.Vector2(),
        y: p.clone().applyMatrix4(toLimb).y,
      });
    }
    let polygons = [triangle];
    for (const cut of cuts) {
      const next: Vertex[][] = [];
      for (const poly of polygons) {
        const low = Math.min(...poly.map((v) => v.y)),
          high = Math.max(...poly.map((v) => v.y));
        if (low < cut - 1e-5 && high > cut + 1e-5) {
          for (const sign of [-1, 1]) {
            const piece = clip(poly, cut, sign);
            if (piece.length >= 3) next.push(piece);
          }
        } else next.push(poly);
      }
      polygons = next;
    }
    const materialIndex =
      geometry.groups.find((g) => i >= g.start && i < g.start + g.count)
        ?.materialIndex ?? 0;
    for (const poly of polygons)
      for (let k = 1; k < poly.length - 1; k++) {
        const start = outP.length / 3;
        for (const v of [poly[0], poly[k], poly[k + 1]]) {
          outP.push(v.p.x, v.p.y, v.p.z);
          outN.push(v.n.x, v.n.y, v.n.z);
          outUV.push(v.uv.x, v.uv.y);
        }
        const last = groups.at(-1);
        if (last && last.materialIndex === materialIndex) last.count += 3;
        else groups.push({ start, count: 3, materialIndex });
      }
  }
  const result = new T.BufferGeometry();
  result.setAttribute("position", new T.Float32BufferAttribute(outP, 3));
  result.setAttribute("normal", new T.Float32BufferAttribute(outN, 3));
  result.setAttribute("uv", new T.Float32BufferAttribute(outUV, 2));
  for (const g of groups) result.addGroup(g.start, g.count, g.materialIndex);
  return result;
}

export function createHumanRig(model: T.Group) {
  const objects = new Map<string, T.Object3D>(),
    parts: RigPart[] = [],
    limbs: Limb[] = [],
    deforms: Deform[] = [];
  model.name = "character";
  model.userData.baseRotation = model.rotation.clone();
  model.userData.basePosition = model.position.clone();
  const add = (
    id: string,
    label: string,
    object: T.Object3D,
    kind: RigPart["kind"],
    parent?: string,
  ) => {
    objects.set(id, object);
    const previous = parts.find((p) => p.id === id);
    if (previous) Object.assign(previous, { label, kind, parent });
    else parts.push({ id, label, kind, parent, limits: jointLimits(id) });
  };
  add("character", "Character", model, "bone");
  model.updateMatrixWorld(true);
  const boneGroups: T.Group[] = [];
  model.traverse((node) => {
    if (
      node instanceof T.Group &&
      node !== model &&
      node.userData.baseRotation &&
      !node.userData.cosmetic
    )
      boneGroups.push(node);
  });
  const body = boneGroups.find((g) => normalized(g.name) === "body");
  let waist = boneGroups.find((g) => normalized(g.name) === "waist");
  if (!waist && body) {
    waist = new T.Group();
    waist.name = "waist";
    waist.position.set(0, 12, 0);
    waist.userData.baseRotation = waist.rotation.clone();
    waist.userData.basePosition = waist.position.clone();
    model.add(waist);
    boneGroups.push(waist);
  }
  const parentTo = (parent: T.Group, child: T.Group) => {
    parent.attach(child);
    child.userData.basePosition = child.position.clone();
    child.userData.baseRotation = child.rotation.clone();
  };
  if (body && waist) {
    if (body.parent === model || normalized(body.parent?.name ?? "") === "root")
      parentTo(waist, body);
    for (const group of boneGroups) {
      if (
        (group.parent !== model &&
          normalized(group.parent?.name ?? "") !== "root") ||
        group === waist ||
        group === body
      )
        continue;
      const name = normalized(group.name);
      if (["head", "leftarm", "rightarm"].includes(name)) parentTo(body, group);
      if (["leftleg", "rightleg"].includes(name)) parentTo(waist, group);
    }
  }
  model.updateMatrixWorld(true);
  for (const group of boneGroups) {
    if (objects.has(group.name)) continue;
    objects.set(group.name, group);
    parts.push({
      id: group.name,
      label: group.name.replace(/([a-z])([A-Z])/g, "$1 $2"),
      kind: "bone",
      parent: group.parent === model ? "character" : group.parent?.name,
      limits: jointLimits(group.name),
    });
  }
  const find = (name: string) =>
    boneGroups.find((g) => normalized(g.name) === normalized(name));
  let vertices = 0;
  for (const name of ["leftArm", "rightArm", "leftLeg", "rightLeg"]) {
    const upper = find(name);
    if (!upper) continue;
    const isLeg = name.endsWith("Leg"),
      side = name.startsWith("left") ? "left" : "right";
    const meshes: T.Mesh[] = [];
    upper.traverse((node) => {
      if (node instanceof T.Mesh && !node.userData.cosmetic) meshes.push(node);
    });
    if (!meshes.length) continue;
    const toUpper = upper.matrixWorld.clone().invert(),
      bounds = new T.Box3();
    for (const mesh of meshes) {
      const transform = toUpper.clone().multiply(mesh.matrixWorld),
        p = mesh.geometry.getAttribute("position");
      for (let i = 0; i < p.count; i++)
        bounds.expandByPoint(
          new T.Vector3().fromBufferAttribute(p, i).applyMatrix4(transform),
        );
    }
    const length = T.MathUtils.clamp(-bounds.min.y, 4, 32),
      elbowName = side + (isLeg ? "Knee" : "Elbow"),
      wristName = side + (isLeg ? "Ankle" : "Wrist");
    const nativeElbow = boneGroups.find(
      (g) =>
        g !== upper &&
        upper.getObjectById(g.id) &&
        new RegExp(side + ".*(elbow|forearm|lowerarm|knee|lowerleg)", "i").test(
          normalized(g.name),
        ),
    );
    const nativeWrist = boneGroups.find(
      (g) =>
        g !== upper &&
        upper.getObjectById(g.id) &&
        new RegExp(side + ".*(wrist|hand|ankle|foot)", "i").test(
          normalized(g.name),
        ),
    );
    const elbow = nativeElbow ?? new T.Group(),
      wrist = nativeWrist ?? new T.Group();
    if (!nativeElbow) {
      elbow.name = elbowName;
      elbow.position.set(0, -length * 0.5, 0);
      upper.add(elbow);
    }
    if (!nativeWrist) {
      wrist.name = wristName;
      wrist.position.set(0, -length * 0.35, 0);
      elbow.add(wrist);
    }
    for (const joint of [elbow, wrist]) {
      if (!joint.userData.baseRotation) {
        joint.userData.baseRotation = joint.rotation.clone();
        joint.userData.basePosition = joint.position.clone();
        joint.userData.virtualRig = true;
      }
    }
    add(
      elbowName,
      side + " " + (isLeg ? "knee" : "elbow"),
      elbow,
      "joint",
      upper.name,
    );
    add(
      wristName,
      side + " " + (isLeg ? "ankle" : "wrist"),
      wrist,
      "joint",
      elbowName,
    );
    const target = side + (isLeg ? "Foot" : "Hand"),
      pole = side + (isLeg ? "KneePole" : "ElbowPole");
    const targetObject = new T.Object3D(),
      poleObject = new T.Object3D();
    targetObject.name = target;
    poleObject.name = pole;
    add(
      target,
      side + " " + (isLeg ? "foot" : "hand") + " target",
      targetObject,
      "target",
    );
    add(
      pole,
      side + (isLeg ? " knee bend direction" : " elbow bend direction"),
      poleObject,
      "pole",
    );
    const limb: Limb = {
      name,
      upper,
      elbow,
      wrist,
      length,
      target,
      pole,
      native: !!nativeElbow,
      upperLength: 0,
      lowerLength: 0,
      upperDirection: new T.Vector3(),
      lowerDirection: new T.Vector3(),
      ikSupported:
        !nativeElbow || (elbow.parent === upper && wrist.parent === elbow),
    };
    upper.updateWorldMatrix(true, true);
    limb.upperDirection.copy(
      upper.worldToLocal(elbow.getWorldPosition(new T.Vector3())),
    );
    limb.upperLength = limb.upperDirection.length();
    limb.upperDirection.normalize();
    limb.lowerDirection.copy(
      elbow.worldToLocal(wrist.getWorldPosition(new T.Vector3())),
    );
    limb.lowerLength = limb.lowerDirection.length();
    limb.lowerDirection.normalize();
    for (const part of parts)
      if (part.id === target || part.id === pole) {
        part.available = limb.ikSupported;
        if (!limb.ikSupported) part.label += " (FK only)";
      }
    limbs.push(limb);
    if (nativeElbow) continue;
    for (const mesh of meshes) {
      const toLimb = toUpper.clone().multiply(mesh.matrixWorld);
      if (
        mesh.geometry.getAttribute("position").count > 30000 ||
        vertices > 180000
      )
        continue;
      const geometry = subdivide(mesh.geometry, toLimb, [
        -length * 0.5 - 0.3,
        -length * 0.5,
        -length * 0.5 + 0.3,
        -length * 0.85 - 0.2,
        -length * 0.85,
        -length * 0.85 + 0.2,
      ]);
      vertices += geometry.getAttribute("position").count;
      mesh.geometry.dispose();
      mesh.geometry = geometry;
      deforms.push({
        mesh,
        toLimb,
        toMesh: toLimb.clone().invert(),
        positions: new Float32Array(geometry.getAttribute("position").array),
        normals: new Float32Array(geometry.getAttribute("normal").array),
        limb,
        key: "",
      });
    }
  }
  function coordinates(object: T.Object3D) {
    const point = object.getWorldPosition(new T.Vector3());
    return model.parent ? model.parent.worldToLocal(point) : point;
  }
  function registerCosmetic(cosmetic: T.Group,cosmeticKey?:string) {
    model.updateMatrixWorld(true);
    for (const limb of limbs) {
      if (limb.native) continue;
      let upper = cosmetic.getObjectByName(limb.upper.name);
      if (!upper)
        cosmetic.traverse((node) => {
          if (!upper && normalized(node.name) === normalized(limb.name))
            upper = node;
        });
      if (!upper) continue;
      upper.traverse((node) => {
        if (
          !(node instanceof T.Mesh) ||
          node.geometry.getAttribute("position").count > 30000 ||
          vertices > 180000
        )
          return;
        const toLimb = limb.upper.matrixWorld
            .clone()
            .invert()
            .multiply(node.matrixWorld),
          length = limb.length;
        const geometry = subdivide(node.geometry, toLimb, [
          -length * 0.5 - 0.3,
          -length * 0.5,
          -length * 0.5 + 0.3,
          -length * 0.85 - 0.2,
          -length * 0.85,
          -length * 0.85 + 0.2,
        ]);
        vertices += geometry.getAttribute("position").count;
        node.geometry.dispose();
        node.geometry = geometry;
        deforms.push({
          cosmeticKey,
          mesh: node,
          toLimb,
          toMesh: toLimb.clone().invert(),
          positions: new Float32Array(geometry.getAttribute("position").array),
          normals: new Float32Array(geometry.getAttribute("normal").array),
          limb,
          key: "",
        });
      });
    }
  }
  function update(pose: SkinPose) {
    model.updateMatrixWorld(true);
    for (const limb of limbs) {
      const { upper, elbow, wrist, length, target, pole } = limb;
      for (const joint of [upper, elbow, wrist]) {
        if (joint === upper) continue;
        const side = limb.name.startsWith("left") ? "left" : "right",
          leg = limb.name.endsWith("Leg");
        const id =
          joint === elbow
            ? side + (leg ? "Knee" : "Elbow")
            : joint === wrist
              ? side + (leg ? "Ankle" : "Wrist")
              : joint.name;
        const angles = pose.axes?.[id] ?? pose.axes?.[joint.name];
        if (angles) {
          const value = limitedAxes(id, angles, pose.limits !== false);
          const base = joint.userData.baseRotation as T.Euler;
          joint.rotation.set(
            base.x + radians(value[0]),
            base.y + radians(value[1]),
            base.z + radians(value[2]),
            base.order,
          );
        }
      }
      upper.updateWorldMatrix(true, true);
      if (pose.targets?.[target] && limb.ikSupported) {
        const convert = (point: Vec3) => {
          const world = new T.Vector3(...point);
          if (model.parent) model.parent.localToWorld(world);
          return upper.parent!.worldToLocal(world);
        };
        const goal = convert(pose.targets[target]),
          origin = upper.position.clone();
        const defaultPole = coordinates(upper).add(
          new T.Vector3(
            0,
            -length * 0.4,
            limb.name.endsWith("Leg") ? -length : length,
          ),
        );
        const polePosition = pose.poles?.[pole]
          ? convert(pose.poles[pole])
          : convert(defaultPole.toArray() as Vec3);
        const solved = solveTwoBone(
          origin,
          goal,
          polePosition,
          limb.upperLength,
          limb.lowerLength,
        );
        const q = new T.Quaternion().setFromUnitVectors(
          limb.upperDirection,
          solved.elbow.clone().sub(origin).normalize(),
        );
        upper.quaternion.copy(q);
        const lowerQ = new T.Quaternion().setFromUnitVectors(
          limb.lowerDirection,
          solved.end.clone().sub(solved.elbow).normalize(),
        );
        elbow.quaternion.copy(q.clone().invert().multiply(lowerQ));
        if (limb.name.endsWith("Leg")) {
          upper.updateWorldMatrix(true, true);
          const offset = wrist.quaternion.clone(),
            world = model.getWorldQuaternion(new T.Quaternion()),
            euler = new T.Euler().setFromQuaternion(world, "YXZ");
          const flat = new T.Quaternion().setFromEuler(
            new T.Euler(0, euler.y, 0, "YXZ"),
          );
          wrist.quaternion.copy(
            elbow
              .getWorldQuaternion(new T.Quaternion())
              .invert()
              .multiply(flat)
              .multiply(offset),
          );
        }
      }
      upper.updateWorldMatrix(true, true);
      const targetObject = objects.get(target)!,
        poleObject = objects.get(pole)!;
      targetObject.position.copy(
        pose.targets?.[target]
          ? new T.Vector3(...pose.targets[target])
          : coordinates(wrist),
      );
      poleObject.position.copy(
        pose.poles?.[pole]
          ? new T.Vector3(...pose.poles[pole])
          : coordinates(elbow).add(
              new T.Vector3(0, 0, limb.name.endsWith("Leg") ? -6 : 6),
            ),
      );
      targetObject.updateMatrixWorld(true);
      poleObject.updateMatrixWorld(true);
    }
    for (const item of deforms) {
      const { mesh, limb, toLimb, toMesh } = item;
      const settings=item.cosmeticKey?pose.cosmetics?.[item.cosmeticKey]:undefined;
      const influence=settings&&settings.binding&&settings.binding!=="auto"?0:T.MathUtils.clamp(settings?.influence??1,0,1);
      const elbowRotation=new T.Quaternion().slerp(limb.elbow.quaternion,settings?.followRotation===false?0:influence),wristRotation=new T.Quaternion().slerp(limb.wrist.quaternion,settings?.followRotation===false?0:influence);
      const key = [
        influence,
        ...elbowRotation.toArray(),
        ...wristRotation.toArray(),
        ...limb.elbow.position.toArray(),
        ...limb.wrist.position.toArray(),
      ].join(",");
      if (key === item.key) continue;
      item.key = key;
      const elbowPoint = new T.Vector3(0, -limb.length * 0.5, 0),
        wristPoint = new T.Vector3(0, -limb.length * 0.85, 0);
      const elbowOffset = limb.elbow.position
          .clone()
          .sub(limb.elbow.userData.basePosition).multiplyScalar(settings?.followPosition===false?0:influence),
        wristOffset = limb.wrist.position
          .clone()
          .sub(limb.wrist.userData.basePosition).multiplyScalar(settings?.followPosition===false?0:influence);
      const p = mesh.geometry.getAttribute("position"),
        n = mesh.geometry.getAttribute("normal");
      for (let i = 0; i < p.count; i++) {
        const point = new T.Vector3()
            .fromArray(item.positions, i * 3)
            .applyMatrix4(toLimb),
          normal = new T.Vector3()
            .fromArray(item.normals, i * 3)
            .transformDirection(toLimb);
        const ew = T.MathUtils.smoothstep(
            -point.y,
            limb.length * 0.5 - 0.3,
            limb.length * 0.5 + 0.3,
          ),
          ww = T.MathUtils.smoothstep(
            -point.y,
            limb.length * 0.85 - 0.2,
            limb.length * 0.85 + 0.2,
          );
        point.lerp(
          point
            .clone()
            .sub(wristPoint)
            .applyQuaternion(wristRotation)
            .add(wristPoint)
            .add(wristOffset),
          ww,
        );
        normal
          .lerp(normal.clone().applyQuaternion(wristRotation), ww)
          .normalize();
        point.lerp(
          point
            .clone()
            .sub(elbowPoint)
            .applyQuaternion(elbowRotation)
            .add(elbowPoint)
            .add(elbowOffset),
          ew,
        );
        normal
          .lerp(normal.clone().applyQuaternion(elbowRotation), ew)
          .normalize();
        point.applyMatrix4(toMesh);
        normal.transformDirection(toMesh);
        p.setXYZ(i, point.x, point.y, point.z);
        n.setXYZ(i, normal.x, normal.y, normal.z);
      }
      p.needsUpdate = n.needsUpdate = true;
      mesh.geometry.computeBoundingSphere();
      mesh.geometry.computeBoundingBox();
    }
  }
  return {
    parts,
    update,
    register: add,
    registerCosmetic,
    object: (id: string) => objects.get(id),
    position: (id: string) => {
      const object = objects.get(id);
      if (!object) return new T.Vector3();
      const part = parts.find((p) => p.id === id);
      return part?.kind === "target" || part?.kind === "pole"
        ? object.position.clone()
        : coordinates(object);
    },
    worldPosition: (id: string) => {
      const point = objects.get(id)?.position.clone() ?? new T.Vector3();
      const part = parts.find((p) => p.id === id);
      return part?.kind === "target" || part?.kind === "pole"
        ? model.parent
          ? model.parent.localToWorld(point)
          : point
        : (objects.get(id)?.getWorldPosition(new T.Vector3()) ?? point);
    },
  };
}

export function mirrorPose(pose: SkinPose): SkinPose {
  const swap = (name: string) =>
    name.replace(/^(left|right)/i, (value) =>
      value.toLowerCase() === "left"
        ? value[0] === "L"
          ? "Right"
          : "right"
        : value[0] === "R"
          ? "Left"
          : "left",
    );
  const mirror = (
    record: Record<string, Vec3> | undefined,
    rotation: boolean,
  ) =>
    Object.fromEntries(
      Object.entries(record ?? {}).map(([name, v]) => [
        swap(name),
        rotation ? [v[0], -v[1], -v[2]] : [-v[0], v[1], v[2]],
      ]),
    ) as Record<string, Vec3>;
  return {
    ...pose,
    leftArm: pose.rightArm,
    rightArm: pose.leftArm,
    leftLeg: pose.rightLeg,
    rightLeg: pose.leftLeg,
    axes: mirror(pose.axes, true),
    positions: mirror(pose.positions, false),
    targets: mirror(pose.targets, false),
    poles: mirror(pose.poles, false),
  };
}

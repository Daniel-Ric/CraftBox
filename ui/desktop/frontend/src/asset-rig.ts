import * as T from "three";
import type { CosmeticBinding, SkinPose } from "./asset-geometry";

type BindPoses = Map<T.Object3D, Map<T.Object3D, T.Matrix4>>;
export function captureBoneBindings(root: T.Group) {
  root.updateMatrixWorld(true);
  const spaces = new Set<T.Object3D>(),
    targets = new Set<T.Object3D>();
  const named = root.userData.cosmeticTargets as
    Map<string, T.Object3D> | undefined;
  for (const target of named?.values() ?? []) targets.add(target);
  root.traverse((node) => {
    const target = node.userData.followBone as T.Object3D | undefined;
    if (target) {
      targets.add(target);
      const space = node.userData.followSpace ?? root;
      spaces.add(space);
      node.userData.followBind = target.matrixWorld.clone().invert();
      if (node.userData.socketRoot)
        node.userData.socketBind = new T.Matrix4()
          .copy(space.matrixWorld)
          .invert()
          .multiply(node.userData.socketRoot.matrixWorld);
    }
    if (node.userData.editableCosmetic)
      spaces.add(node.userData.followSpace ?? root);
  });
  const poses: BindPoses = new Map();
  for (const space of spaces) {
    const inverse = space.matrixWorld.clone().invert(),
      entries = new Map<T.Object3D, T.Matrix4>();
    for (const target of targets)
      entries.set(target, inverse.clone().multiply(target.matrixWorld));
    poses.set(space, entries);
  }
  root.userData.cosmeticBindPoses = poses;
}
function blendedDelta(
  bind: T.Matrix4,
  current: T.Matrix4,
  settings: CosmeticBinding,
) {
  const weight = T.MathUtils.clamp(settings.influence ?? 1, 0, 1);
  if (
    weight === 1 &&
    settings.followPosition !== false &&
    settings.followRotation !== false &&
    settings.followScale !== false
  )
    return current.clone().multiply(bind.clone().invert());
  const p0 = new T.Vector3(),
    p1 = new T.Vector3(),
    q0 = new T.Quaternion(),
    q1 = new T.Quaternion(),
    s0 = new T.Vector3(),
    s1 = new T.Vector3();
  bind.decompose(p0, q0, s0);
  current.decompose(p1, q1, s1);
  const position =
    settings.followPosition === false
      ? p0.clone()
      : p0.clone().lerp(p1, weight);
  const rotation =
    settings.followRotation === false
      ? new T.Quaternion()
      : new T.Quaternion().slerp(q1.multiply(q0.invert()), weight);
  const scale =
    settings.followScale === false
      ? new T.Vector3(1, 1, 1)
      : new T.Vector3(1, 1, 1).lerp(s1.divide(s0), weight);
  return new T.Matrix4()
    .compose(position, rotation, scale)
    .multiply(new T.Matrix4().makeTranslation(-p0.x, -p0.y, -p0.z));
}
export function followBones(
  root: T.Group,
  pose: Pick<SkinPose, "cosmetics"> = {},
) {
  root.updateMatrixWorld(true);
  const frames = new Map<
    T.Object3D,
    {
      source: T.Matrix4;
      target: T.Object3D;
      space: T.Object3D;
      settings: CosmeticBinding;
      socket: boolean;
      rest: boolean;
    }
  >();
  const named = root.userData.cosmeticTargets as
    Map<string, T.Object3D> | undefined;
  const poses = root.userData.cosmeticBindPoses as BindPoses | undefined;
  root.traverse((node) => {
    const key = node.userData.cosmeticKey as string | undefined,
      settings = key ? (pose.cosmetics?.[key] ?? {}) : {},
      mode = settings.binding ?? "auto";
    if (node.userData.editableCosmetic)
      node.visible = settings.visible !== false;
    const explicit = mode !== "auto" && mode !== "none";
    const socket = !!node.userData.socketRoot;
    const rest = socket && mode !== "auto";
    if (explicit && !node.userData.editableCosmetic && !socket) return;
    if (mode === "none" && !socket) return;
    const target =
      explicit && !rest
        ? (named?.get(mode) ?? named?.get(mode.toLowerCase()))
        : (node.userData.followBone as T.Object3D | undefined);
    if (target && node.parent)
      frames.set(node, {
        source: node.matrixWorld.clone(),
        target,
        space: node.userData.followSpace ?? root,
        settings,
        socket,
        rest,
      });
  });
  const ordered = [...frames].sort(
    ([a], [b]) =>
      Number(!!a.userData.editableCosmetic) -
      Number(!!b.userData.editableCosmetic),
  );
  for (const [
    node,
    { source, target, space, settings, socket, rest },
  ] of ordered) {
    const inverse = space.matrixWorld.clone().invert();
    const current = inverse.clone().multiply(target.matrixWorld);
    let bind = poses?.get(space)?.get(target);
    if (!bind) {
      bind = current.clone();
      if (!root.userData.cosmeticBindPoses)
        root.userData.cosmeticBindPoses = new Map();
      const all = root.userData.cosmeticBindPoses as BindPoses;
      if (!all.has(space)) all.set(space, new Map());
      all.get(space)!.set(target, bind);
    }
    const baseline = socket
      ? space.matrixWorld
          .clone()
          .multiply(bind)
          .multiply(
            (
              (node.userData.socketBind as T.Matrix4 | undefined) ??
              new T.Matrix4()
            )
              .clone()
              .invert(),
          )
          .multiply(inverse)
          .multiply(source)
      : source;
    const destination = rest
      ? baseline
      : space.matrixWorld
          .clone()
          .multiply(blendedDelta(bind, current, settings))
          .multiply(inverse)
          .multiply(baseline);
    node.parent!.updateWorldMatrix(true, false);
    const local = node
      .parent!.matrixWorld.clone()
      .invert()
      .multiply(destination);
    local.decompose(node.position, node.quaternion, node.scale);
    node.updateWorldMatrix(false, true);
  }
}

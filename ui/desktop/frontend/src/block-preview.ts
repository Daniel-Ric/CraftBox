import * as T from "three";
const number = (v: unknown, fallback: number) =>
  typeof v === "number" && Number.isFinite(v) && Math.abs(v) <= 100000
    ? v
    : fallback;
function vector(raw: unknown, fallback: [number, number, number]): T.Vector3 {
  const value =
    raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  return new T.Vector3(
    ...(fallback.map((n, i) =>
      number(Array.isArray(raw) ? raw[i] : value[["x", "y", "z"][i]], n),
    ) as [number, number, number]),
  );
}
export function blockTransformMatrix(transform: Record<string, unknown>) {
  const t = transform;
  const rotation = vector(t.rotation, [
    number(t.RX, 0) * 90,
    number(t.RY, 0) * 90,
    number(t.RZ, 0) * 90,
  ]).multiplyScalar(Math.PI / 180);
  const translation = vector(t.translation, [
    number(t.TX, 0),
    number(t.TY, 0),
    number(t.TZ, 0),
  ]).multiplyScalar(16);
  const scale = vector(t.scale, [
    number(t.SX, 1),
    number(t.SY, 1),
    number(t.SZ, 1),
  ]);
  const rp = vector(t.rotation_pivot, [0, 0, 0])
    .multiplyScalar(16)
    .add(new T.Vector3(0, 8, 0));
  const sp = vector(t.scale_pivot, [0, 0, 0])
    .multiplyScalar(16)
    .add(new T.Vector3(0, 8, 0));
  const shift = (v: T.Vector3) =>
    new T.Matrix4().makeTranslation(v.x, v.y, v.z);
  const matrix = shift(translation)
    .multiply(shift(rp))
    .multiply(
      new T.Matrix4().makeRotationFromEuler(
        new T.Euler(rotation.x, rotation.y, rotation.z, "ZYX"),
      ),
    )
    .multiply(shift(rp.clone().negate()))
    .multiply(shift(sp))
    .multiply(new T.Matrix4().makeScale(scale.x, scale.y, scale.z))
    .multiply(shift(sp.clone().negate()));
  const basis = new T.Matrix4().makeScale(-1, 1, 1);
  return basis.clone().multiply(matrix).multiply(basis);
}
export function transformBlockModel(
  model: T.Group,
  transform?: Record<string, unknown>,
) {
  if (!transform || !Object.keys(transform).length) return model;
  const root = new T.Group();
  root.name = "Block transformation";
  root.matrixAutoUpdate = false;
  root.matrix.copy(blockTransformMatrix(transform));
  root.add(model);
  return root;
}

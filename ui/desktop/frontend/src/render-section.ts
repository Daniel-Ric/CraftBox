import { Plane, Vector3 } from "three";

export type RenderSection = {
  mode: "off" | "slice" | "box";
  axis: 0 | 1 | 2;
  flip: boolean;
  position?: number;
  min: [number, number, number];
  max: [number, number, number];
};

export const sectionDefaults: RenderSection = {
  mode: "off",
  axis: 1,
  flip: false,
  min: [-64, -64, -64],
  max: [64, 64, 64],
};

export function parseRenderSection(value: unknown): RenderSection {
  if (value === undefined) return sectionDefaults;
  const section = value as RenderSection;
  const vector = (v: unknown) =>
    Array.isArray(v) &&
    v.length === 3 &&
    v.every(
      (coordinate) =>
        Number.isInteger(coordinate) && Math.abs(coordinate) <= 30000000,
    );
  if (
    !section ||
    !["off", "slice", "box"].includes(section.mode) ||
    ![0, 1, 2].includes(section.axis) ||
    typeof section.flip !== "boolean" ||
    !vector(section.min) ||
    !vector(section.max) ||
    (section.position !== undefined &&
      (!Number.isInteger(section.position) ||
        Math.abs(section.position) > 30000000)) ||
    section.min.some((coordinate, axis) => coordinate >= section.max[axis])
  )
    throw new Error("Invalid section view.");
  return section;
}

export function sectionPlanes(
  section: RenderSection,
  origin: Vector3,
): Plane[] {
  const plane = (axis: number, boundary: number, sign: number) =>
    new Plane(
      new Vector3().setComponent(axis, sign),
      sign * (origin.getComponent(axis) - boundary),
    );
  if (section.mode === "off") return [];
  if (section.mode === "slice")
    return [
      plane(
        section.axis,
        section.position ?? section.max[section.axis],
        section.flip ? 1 : -1,
      ),
    ];
  return section.min.flatMap((lower, axis) => [
    plane(axis, lower, 1),
    plane(axis, section.max[axis], -1),
  ]);
}

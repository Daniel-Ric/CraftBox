export const perspectives = [
  { id: "iso", label: "Isometric · Front right", angle: 45, tilt: 35.264 },
  {
    id: "iso-left",
    label: "Isometric · Front left",
    angle: -45,
    tilt: 35.264,
  },
  {
    id: "iso-back",
    label: "Isometric · Back right",
    angle: 135,
    tilt: 35.264,
  },
  {
    id: "iso-back-left",
    label: "Isometric · Back left",
    angle: 225,
    tilt: 35.264,
  },
  { id: "front", label: "Front view", angle: 0, tilt: 0 },
  { id: "back", label: "Back view", angle: 180, tilt: 0 },
  { id: "left", label: "Left view", angle: -90, tilt: 0 },
  { id: "right", label: "Right view", angle: 90, tilt: 0 },
  { id: "top", label: "Top view", angle: 0, tilt: 89.5 },
  { id: "bottom", label: "Bottom view", angle: 0, tilt: -89.5 },
];
export function polarLimits(locked: boolean, angle: number): [number, number] {
  return locked ? [angle, angle] : [0, Math.PI];
}

import type { RenderSettings } from "./render-settings";
import { decodeRenderStyle, encodeRenderStyle } from "./render-style-library";
import { parseSequence, type Sequence, type CameraPose } from "./cinematic";
import { mapDefaults, type MapPresentation } from "./map-presentation";
export type CinematicViewState = {
  sequence: Sequence;
  time: number;
  selected: string;
  preview: boolean;
  showPath: boolean;
  zoom: number;
  snap: boolean;
  guides: boolean;
  loop: boolean;
  speed: number;
  navigation: "orbit" | "fly";
  navigationSpeed: number;
  cameraId?: string;
  program?: boolean;
};
export type StudioDocument = {
  format: "craftbox-studio";
  version: 1;
  name: string;
  editor: "render" | "cinematic";
  settings: RenderSettings;
  look?: string;
  mapView: MapPresentation;
  view?: CameraPose & { zoom?: number };
  cinematic?: CinematicViewState;
  customSky?: { name: string; data: string };
};
export type StudioSession = {
  name: string;
  dirty: boolean;
  busy: boolean;
  save: () => Promise<boolean>;
};
export function parseStudioDocument(text: string): StudioDocument {
  if (text.length > 96 * 1024 * 1024)
    throw new Error("The project is too large.");
  const d = JSON.parse(text);
  if (
    d?.format !== "craftbox-studio" ||
    d.version !== 1 ||
    !["render", "cinematic"].includes(d.editor) ||
    typeof d.name !== "string" ||
    !d.settings
  )
    throw new Error("Not a supported CraftBox project document.");
  const style = decodeRenderStyle(
    encodeRenderStyle(d.name, d.settings, d.customSky),
  );
  const map = { ...mapDefaults, ...d.mapView };
  for (const [key, min, max] of [
    ["rotation", 0, 360],
    ["zoom", 0.1, 20],
    ["gridSize", 1, 256],
    ["gridOpacity", 0, 1],
    ["contourStep", 1, 128],
    ["heightTint", 0, 1],
    ["saturation", 0, 2],
  ] as const) {
    if (!Number.isFinite(map[key]) || map[key] < min || map[key] > max)
      throw new Error("Invalid map setting: " + key);
  }
  for (const key of [
    "enabled",
    "grid",
    "contours",
    "shading",
    "transparent",
  ] as const)
    if (typeof map[key] !== "boolean")
      throw new Error("Invalid map option.");
  if (!/^#[a-f\d]{6}$/i.test(map.background))
    throw new Error("Invalid map background.");
  map.transparent = true;
  if (!Array.isArray(map.markers) || map.markers.length > 512)
    throw new Error("Too many or invalid map markers.");
  for (const marker of map.markers) {
    if (
      !marker ||
      typeof marker.id !== "string" ||
      typeof marker.label !== "string" ||
      marker.label.length > 100 ||
      typeof marker.icon !== "string" ||
      marker.icon.length > 8 ||
      ![marker.x, marker.z, marker.scale].every(Number.isFinite) ||
      Math.abs(marker.x) > 3e7 ||
      Math.abs(marker.z) > 3e7 ||
      marker.scale < 0.5 ||
      marker.scale > 3 ||
      !/^#[a-f\d]{6}$/i.test(marker.color) ||
      !/^#[a-f\d]{6}$/i.test(marker.background)
    )
      throw new Error("Invalid map marker.");
  }
  if (d.view) {
    const v = d.view;
    if (
      !Array.isArray(v.position) ||
      !Array.isArray(v.target) ||
      v.position.length !== 3 ||
      v.target.length !== 3 ||
      ![
        ...v.position,
        ...v.target,
        v.fov,
        v.roll,
        v.exposure,
        v.zoom ?? 1,
      ].every(Number.isFinite)
    )
      throw new Error("Invalid saved camera.");
  }
  if (d.cinematic) {
    d.cinematic.sequence = parseSequence(JSON.stringify(d.cinematic.sequence));
    if (
      !Number.isFinite(d.cinematic.time) ||
      d.cinematic.time < 0 ||
      d.cinematic.time > d.cinematic.sequence.duration
    )
      throw new Error("Invalid sequence position.");
  }
  return {
    ...d,
    name: d.name.slice(0, 180),
    settings: style.settings,
    mapView: map,
  };
}
export function restoreStudioSky(sky?: StudioDocument["customSky"]) {
  if (!sky) return;
  const raw = atob(sky.data);
  return new File([Uint8Array.from(raw, (c) => c.charCodeAt(0))], sky.name);
}
export async function captureStudioSky(file?: File) {
  if (!file) return;
  const bytes = new Uint8Array(await file.arrayBuffer());
  let text = "";
  for (let i = 0; i < bytes.length; i += 32768)
    text += String.fromCharCode(...bytes.subarray(i, i + 32768));
  return { name: file.name, data: btoa(text) };
}

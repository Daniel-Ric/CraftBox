import {
  renderDefaults,
  renderDimensions,
  type RenderSettings,
} from "./render-settings";
import { normalizeAtmosphere, atmosphereDefaults } from "./atmosphere-settings";
import { lightingRanges } from "./lighting-settings";
import { parseStudioLights } from "./studio-lights";
export type RenderStyle = {
  format: "craftbox-render-style";
  version: 1;
  id: string;
  name: string;
  settings: RenderSettings;
  customSky?: { name: string; data: string };
};
const key = "craftbox-render-styles";
export function encodeRenderStyle(
  name: string,
  settings: RenderSettings,
  customSky?: RenderStyle["customSky"],
) {
  return JSON.stringify(
    {
      format: "craftbox-render-style",
      version: 1,
      id: crypto.randomUUID(),
      name: name.trim() || "Render preset",
      settings,
      customSky,
    },
    null,
    2,
  );
}
export function decodeRenderStyle(text: string): RenderStyle {
  if (text.length > 96 * 1024 * 1024)
    throw new Error("The preset exceeds 96 MiB.");
  const raw = JSON.parse(text);
  if (
    raw?.format !== "craftbox-render-style" ||
    raw.version !== 1 ||
    typeof raw.name !== "string" ||
    !raw.settings ||
    typeof raw.settings !== "object"
  )
    throw new Error("Not a valid CraftBox render preset.");
  const settings = { ...renderDefaults } as Record<string, unknown>;

  if (raw.settings.dofMode === undefined) {
    settings.dofMode = "lens";
    settings.dofNearStrength = 1;
    settings.dofFarStrength = 1;
  }
  for (const [k, def] of Object.entries(renderDefaults)) {
    if (
      k === "atmosphere" ||
      k === "studioLights" ||
      raw.settings[k] === undefined
    )
      continue;
    const v = raw.settings[k];
    if (
      typeof v !== typeof def ||
      (typeof v === "number" &&
        (!Number.isFinite(v) ||
          Math.abs(v) > (k.startsWith("dofTarget") ? 30000000 : 100000)))
    )
      throw new Error("Invalid setting value: " + k);
    settings[k] = v;
  }
  if (!["fast", "balanced", "fine"].includes(settings.quality as string))
    throw new Error("Invalid render quality.");
  if (!["lens", "depth", "subject"].includes(settings.dofMode as string))
    throw new Error("Invalid focus mode.");
  if (![1, 4, 16].includes(settings.exportSamples as number))
    throw new Error("Invalid image sample count.");
  if (
    ![4, 8, 12, 16, 24].includes(settings.lightBudget as number) ||
    ![0, 1, 2].includes(settings.blockLightShadows as number)
  )
    throw new Error("Invalid block light budget.");
  const domains: Record<string, [number, number]> = {
    dofDistance: [0.001, 100000],
    dofAperture: [0.7, 64],
    dofFocalLength: [8, 300],
    dofWorldScale: [0.01, 1000],
    dofMaxBlur: [1, 80],
    dofNearStrength: [0, 5],
    dofFarStrength: [0, 5],
    dofFocusRange: [0, 100000],
    dofTransition: [0.1, 100000],
    ...lightingRanges,
    timeOfDay: [0, 24],
    azimuth: [0, 360],
    elevation: [5, 85],
    sun: [0, 6],
    ambient: [0.1, 3],
    exposure: [0.3, 1.8],
    ao: [0, 1.5],
    skyRotation: [0, 360],
    skyElevation: [-30, 30],
    skyBrightness: [0.05, 2],
    environment: [0, 3],
    softness: [0.5, 5],
    aoRadius: [0.25, 4],
    reflections: [0, 3],
    roughness: [0.1, 1],
    waterOpacity: [0, 1],
    waterVisibility: [0.25, 100],
    waterRoughness: [0.04, 1],
    waterReflections: [0, 3],
    waterWaves: [0, 1],
    waterColorStrength: [0, 1],
    waterColorDepth: [0.25, 100],
    waterTextureStrength: [0, 2],
    waterWaveScale: [0.1, 8],
    waterTransparency: [0, 1],
    waterDeepLight: [0, 2],
    waterShoreSoftness: [0.01, 8],
    frameMargin: [0.02, 0.4],
    frameVerticalShift: [-0.3, 0.3],
    detailProtection: [0, 1],
    bloom: [0, 1],
    saturation: [0, 1.8],
    vibrance: [-0.5, 0.6],
    contrast: [0.7, 1.4],
    temperature: [-1, 1],
    tint: [-1, 1],
    highlights: [-1, 1],
    shadows: [-1, 1],
    sharpen: [0, 0.8],
    vignette: [0, 0.6],
  };
  for (const [k, [min, max]] of Object.entries(domains))
    if ((settings[k] as number) < min || (settings[k] as number) > max)
      throw new Error("Setting value out of range: " + k);
  if (![1024, 2048, 4096].includes(settings.shadow as number))
    throw new Error("Invalid shadow resolution.");
  for (const k of [
    "background",
    "warmth",
    "spotColor",
    "waterTint",
    "waterShallowColor",
    "waterDeepColor",
    "waterDeepLightColor",
  ])
    if (!/^#[0-9a-f]{6}$/i.test(settings[k] as string))
      throw new Error("Invalid color.");
  renderDimensions(settings.edge as number, settings.aspect as number);
  const atmosphere = raw.settings.atmosphere;
  if (atmosphere !== undefined) {
    if (!atmosphere || typeof atmosphere !== "object")
      throw new Error("Invalid atmosphere.");
    for (const [k, def] of Object.entries(atmosphereDefaults))
      if (atmosphere[k] !== undefined && typeof atmosphere[k] !== typeof def)
        throw new Error("Invalid atmosphere value: " + k);
  }
  settings.atmosphere = normalizeAtmosphere(atmosphere);
  settings.studioLights = parseStudioLights(raw.settings.studioLights);
  const style: RenderStyle = {
    format: "craftbox-render-style",
    version: 1,
    id: typeof raw.id === "string" ? raw.id : crypto.randomUUID(),
    name: raw.name.trim().slice(0, 120) || "Render preset",
    settings: settings as RenderSettings,
  };
  if (raw.customSky) {
    if (
      typeof raw.customSky.name !== "string" ||
      typeof raw.customSky.data !== "string" ||
      raw.customSky.data.length > 4 * Math.ceil((64 * 1024 * 1024) / 3) ||
      raw.customSky.data.length % 4 !== 0 ||
      !/^[A-Za-z0-9+/]*={0,2}$/.test(raw.customSky.data)
    )
      throw new Error("Invalid HDR file in preset.");
    style.customSky = {
      name: raw.customSky.name.slice(0, 180),
      data: raw.customSky.data,
    };
  }
  if (style.settings.sky === "custom" && !style.customSky)
    throw new Error("The custom HDR skybox is missing from the preset.");
  return style;
}
async function database() {
  return new Promise<IDBDatabase>((resolve, reject) => {
    const request = indexedDB.open(key, 1);
    request.onupgradeneeded = () =>
      request.result.createObjectStore("styles", { keyPath: "id" });
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}
export async function loadRenderStyles(): Promise<RenderStyle[]> {
  if (typeof indexedDB === "undefined") {
    try {
      return JSON.parse(localStorage.getItem(key) || "[]").map((s: unknown) =>
        decodeRenderStyle(JSON.stringify(s)),
      );
    } catch {
      return [];
    }
  }
  const db = await database();
  try {
    return await new Promise((resolve, reject) => {
      const r = db.transaction("styles").objectStore("styles").getAll();
      r.onsuccess = () => {
        try {
          resolve(r.result.map((s) => decodeRenderStyle(JSON.stringify(s))));
        } catch (e) {
          reject(e);
        }
      };
      r.onerror = () => reject(r.error);
    });
  } finally {
    db.close();
  }
}
export async function saveRenderStyle(style: RenderStyle) {
  const copy = decodeRenderStyle(JSON.stringify(style));
  if (typeof indexedDB === "undefined") {
    const styles = await loadRenderStyles();
    localStorage.setItem(
      key,
      JSON.stringify([...styles.filter((s) => s.id !== copy.id), copy]),
    );
    return;
  }
  const db = await database();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction("styles", "readwrite");
      tx.objectStore("styles").put(copy);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error || new Error("Save canceled."));
    });
  } finally {
    db.close();
  }
}
export async function deleteRenderStyle(id: string) {
  if (typeof indexedDB === "undefined") {
    localStorage.setItem(
      key,
      JSON.stringify((await loadRenderStyles()).filter((s) => s.id !== id)),
    );
    return;
  }
  const db = await database();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction("styles", "readwrite");
      tx.objectStore("styles").delete(id);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

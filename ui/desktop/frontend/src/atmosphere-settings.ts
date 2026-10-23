import {
  parseCloudLayers,
  parseCloudOpening,
  cloudOpeningDefaults,
  type CloudLayer,
  type CloudOpening,
} from "./cloud-layers";
export type CloudType =
  | "cumulus"
  | "stratus"
  | "storm"
  | "cirrus"
  | "stratocumulus"
  | "altocumulus"
  | "lenticular"
  | "anvil";
export type HeightReference = "world" | "bottom" | "center" | "top";
export type FogType = "mist" | "banks" | "valley" | "haze" | "smoke";
export type AtmosphereSettings = {
  fog: boolean;
  fogColor: string;
  fogDensity: number;
  fogHeight: number;
  fogFalloff: number;
  fogVariation: number;
  fogSize: number;
  fogStart: number;
  fogScattering: number;
  fogAnisotropy: number;
  clouds: boolean;
  cloudType: CloudType;
  cloudModel: "natural" | "bodies" | "classic" | "banks";
  cloudColor: string;
  cloudDensity: number;
  cloudHeight: number;
  cloudThickness: number;
  cloudSize: number;
  coverage: number;
  cloudShadows: number;
  windSpeed: number;
  windDirection: number;
  animate: boolean;
  time: number;
  detail: number;
  erosion: number;
  turbulence: number;
  cloudReference: HeightReference;
  fogReference: HeightReference;
  cloudSeed: number;
  fogSeed: number;
  cloudClumping: number;
  cloudRoundness: number;
  fogType: FogType;
  fogTurbulence: number;
  distance: number;
  lightAbsorption: number;
  cloudExtent: number;
  cloudAvoidance: number;
  cloudClearance: number;
  cloudLayers: CloudLayer[];
  cloudOpening: CloudOpening;
  cloudPreviewQuality: "auto" | "low" | "medium" | "high";
  cloudExportQuality: "medium" | "high" | "ultra";
  cloudSilverLining: number;
  cloudMultipleScattering: number;
};
export const atmosphereDefaults: AtmosphereSettings = {
  fog: false,
  fogColor: "#b9cbd7",
  fogDensity: 0.35,
  fogHeight: 48,
  fogFalloff: 32,
  fogVariation: 0.45,
  fogSize: 180,
  fogStart: 0,
  fogScattering: 0.6,
  fogAnisotropy: 0.5,
  clouds: false,
  cloudType: "cumulus",
  cloudModel: "natural",
  cloudColor: "#ffffff",
  cloudDensity: 1,
  cloudHeight: 160,
  cloudThickness: 64,
  cloudSize: 96,
  coverage: 0.5,
  cloudShadows: 0.65,
  windSpeed: 2,
  windDirection: 90,
  animate: true,
  time: 0,
  detail: 0.5,
  erosion: 0.4,
  turbulence: 0.3,
  cloudReference: "world",
  fogReference: "world",
  cloudSeed: 42,
  fogSeed: 137,
  cloudClumping: 0.55,
  cloudRoundness: 0.7,
  fogType: "mist",
  fogTurbulence: 0.4,
  distance: 6000,
  lightAbsorption: 1,
  cloudExtent: 1.8,
  cloudAvoidance: 0.95,
  cloudClearance: 18,
  cloudLayers: [],
  cloudOpening: cloudOpeningDefaults,
  cloudPreviewQuality: "auto",
  cloudExportQuality: "high",
  cloudSilverLining: 0.45,
  cloudMultipleScattering: 0.35,
};
export const cloudTypes: Record<CloudType, string> = {
  cumulus: "Cumulus · billowing clouds",
  stratus: "Stratus · continuous layer",
  storm: "Cumulonimbus · dark storm clouds",
  cirrus: "Cirrus · delicate high clouds",
  stratocumulus: "Stratocumulus · scattered cloud fields",
  altocumulus: "Altocumulus · small cloud towers",
  lenticular: "Lenticularis · soft lens clouds",
  anvil: "Anvil · towering storm clouds",
};
export const fogTypes: Record<FogType, string> = {
  mist: "Haze · soft and fine",
  banks: "Fog banks · irregular wisps",
  valley: "Valley fog · low and layered",
  haze: "Atmospheric haze · uniform",
  smoke: "Dramatic fog · turbulent columns",
};
export function atmosphereHeight(
  value: number,
  reference: HeightReference,
  bounds?: { min: number; max: number },
) {
  if (!bounds || reference === "world") return value;
  return (
    value +
    (reference === "bottom"
      ? bounds.min
      : reference === "top"
        ? bounds.max
        : (bounds.min + bounds.max) / 2)
  );
}
export function normalizeAtmosphere(
  input?: Partial<AtmosphereSettings>,
): AtmosphereSettings {
  const s = { ...atmosphereDefaults, ...input };
  const limits = {
    fogDensity: [0, 3],
    fogHeight: [-2048, 4096],
    fogFalloff: [1, 512],
    fogVariation: [0, 1],
    fogSize: [8, 1024],
    fogStart: [0, 2048],
    fogScattering: [0, 2],
    fogAnisotropy: [0, 0.9],
    cloudDensity: [0, 4],
    cloudHeight: [-2048, 4096],
    cloudThickness: [2, 512],
    cloudSize: [8, 1024],
    coverage: [0, 1],
    cloudShadows: [0, 1],
    windSpeed: [0, 40],
    windDirection: [0, 360],
    time: [0, 3600],
    detail: [0, 1],
    erosion: [0, 1],
    turbulence: [0, 1],
    cloudSeed: [0, 999999],
    fogSeed: [0, 999999],
    cloudClumping: [0, 1],
    cloudRoundness: [0, 1],
    fogTurbulence: [0, 1],
    distance: [512, 30000],
    lightAbsorption: [0, 3],
    cloudExtent: [0.5, 8],
    cloudAvoidance: [0, 1],
    cloudClearance: [1, 128],
    cloudSilverLining: [0, 2],
    cloudMultipleScattering: [0, 1],
  };
  for (const [key, [min, max]] of Object.entries(limits)) {
    const k = key as keyof typeof limits;
    s[k] = Number.isFinite(s[k])
      ? Math.min(max, Math.max(min, s[k]))
      : atmosphereDefaults[k];
  }
  for (const key of ["fogColor", "cloudColor"] as const)
    if (!/^#[0-9a-f]{6}$/i.test(s[key])) s[key] = atmosphereDefaults[key];
  if (!Object.hasOwn(cloudTypes, s.cloudType)) s.cloudType = "cumulus";
  if (!["natural", "bodies", "classic", "banks"].includes(s.cloudModel))
    s.cloudModel = "natural";
  if (!Object.hasOwn(fogTypes, s.fogType)) s.fogType = "mist";
  for (const key of ["cloudReference", "fogReference"] as const)
    if (!["world", "bottom", "center", "top"].includes(s[key]))
      s[key] = "world";
  s.cloudLayers = parseCloudLayers(s.cloudLayers);
  s.cloudOpening = parseCloudOpening(s.cloudOpening);
  if (!["auto", "low", "medium", "high"].includes(s.cloudPreviewQuality))
    s.cloudPreviewQuality = "auto";
  if (!["medium", "high", "ultra"].includes(s.cloudExportQuality))
    s.cloudExportQuality = "high";
  return s;
}

import {
  atmosphereDefaults,
  type AtmosphereSettings,
} from "./atmosphere-settings";
import { lightingDefaults, type LightingSettings } from "./lighting-settings";
import { sceneLooks } from "./scene-looks";
import type { StudioLight } from "./studio-lights";
export type RenderSettings = LightingSettings & {
  dofEnabled: boolean;
  dofDistance: number;
  dofAperture: number;
  dofFocalLength: number;
  dofWorldScale: number;
  dofMaxBlur: number;
  studioLights: StudioLight[];
  atmosphere: AtmosphereSettings;
  pitchLocked: boolean;
  timeOfDay: number;
  timeLinked: boolean;
  azimuth: number;
  elevation: number;
  sun: number;
  ambient: number;
  exposure: number;
  ao: number;
  shadow: number;
  background: string;
  transparent: boolean;
  edge: number;
  aspect: number;
  sky: string;
  skyRotation: number;
  skyBrightness: number;
  environment: number;
  warmth: string;
  softness: number;
  aoRadius: number;
  quality: "fast" | "balanced" | "fine";
  sunLinked: boolean;
  skyElevation: number;
  reflections: number;
  roughness: number;
  bloom: number;
  saturation: number;
  vibrance: number;
  contrast: number;
  temperature: number;
  tint: number;
  highlights: number;
  shadows: number;
  sharpen: number;
  vignette: number;
  grading: boolean;
};
export const renderDefaults: RenderSettings = {
  dofEnabled: false,
  dofDistance: 100,
  dofAperture: 2.8,
  dofFocalLength: 50,
  dofWorldScale: 1,
  dofMaxBlur: 16,
  studioLights: [],
  ...lightingDefaults,
  atmosphere: { ...atmosphereDefaults },
  pitchLocked: false,
  timeOfDay: 12,
  timeLinked: false,
  azimuth: 135,
  elevation: 38,
  sun: 3,
  ambient: 0.65,
  exposure: 1,
  ao: 0.45,
  shadow: 2048,
  background: "#dce6e9",
  transparent: false,
  edge: 1920,
  aspect: 16 / 9,
  sky: "daylight",
  skyRotation: 0,
  skyBrightness: 1,
  environment: 0.85,
  warmth: "#fff1dc",
  softness: 2.5,
  aoRadius: 1.5,
  quality: "balanced",
  sunLinked: true,
  skyElevation: 0,
  reflections: 1,
  roughness: 0.85,
  bloom: 0.12,
  saturation: 1.08,
  vibrance: 0.12,
  contrast: 1.04,
  temperature: 0,
  tint: 0,
  highlights: -0.08,
  shadows: 0,
  sharpen: 0.12,
  vignette: 0,
  grading: true,
};
export const renderPresetLabels: Record<string, string> = {
  ...Object.fromEntries(sceneLooks.map((l) => [l.id, l.name])),
  cloudIsland: "Cloud island · Daylight",
  cloudGolden: "Cloud island · Golden backlight",
  cloudSoft: "Cloud island · Soft haze",
  island: "Vibrant island",
  garden: "Flower garden",
  film: "Warm film",
  winterevening: "Winter evening",
  crisp: "Clear & neutral",
  daylight: "Daylight",
  soft: "Soft light",
  tropical: "Tropical",
  golden: "Evening sun",
  sunset: "Sunset",
  bluehour: "Blue hour",
  night: "Moonlit night",
  winter: "Winter light",
  dramatic: "Dramatic",
  architecture: "Architecture",
  studio: "Studio",
  neutral: "Neutral",
};
export const renderLookDescriptions: Record<string, string> = {
  ...Object.fromEntries(sceneLooks.map((l) => [l.id, l.description])),
  cloudIsland:
    "Volumetric clouds around the island base, clear colors and cool shadows.",
  cloudGolden:
    "Warm backlighting, deep cloud shadows and golden rim light.",
  cloudSoft: "An airy island look with scattered clouds and delicate fog wisps.",
  island: "Rich turquoise and green, clear sunlight and open shadows.",
  garden: "Vibrant flowers, natural greens and sculpted depth.",
  film: "Warm lighting, soft highlights and restrained colors.",
  winterevening: "Cool shadows, warm sunlight and subtle bloom.",
  crisp: "Neutral colors and clean detail as a starting point.",
};
const look = (settings: Partial<RenderSettings>): Partial<RenderSettings> => ({
  quality: "fine",
  timeLinked: false,
  azimuth: 135,
  elevation: 45,
  sun: 3,
  ambient: 1.1,
  exposure: 1,
  ao: 0.45,
  aoRadius: 1.5,
  sky: "daylight",
  skyRotation: 0,
  skyBrightness: 1,
  environment: 0.85,
  warmth: "#fff1dc",
  softness: 2.5,
  sunLinked: true,
  saturation: 1.08,
  vibrance: 0.12,
  contrast: 1.04,
  temperature: 0,
  tint: 0,
  highlights: -0.08,
  shadows: 0,
  sharpen: 0.12,
  vignette: 0,
  grading: true,
  ...settings,
});
export const renderPresets: Record<string, Partial<RenderSettings>> = {
  ...Object.fromEntries(sceneLooks.map((l) => [l.id, l.settings])),
  cloudIsland: look({
    sun: 3.2,
    ambient: 0.65,
    sunLinked: false,
    elevation: 42,
    azimuth: 135,
    sky: "daylight",
    warmth: "#fff4e5",
    saturation: 1.16,
    vibrance: 0.2,
    contrast: 1.08,
    bloom: 0.08,
    shadows: 0.08,
    quality: "fine",
    atmosphere: {
      ...atmosphereDefaults,
      clouds: true,
      cloudModel: "banks",
      cloudType: "cumulus",
      cloudReference: "bottom",
      cloudHeight: -24,
      cloudThickness: 120,
      cloudSize: 130,
      coverage: 0.62,
      cloudDensity: 1.15,
      cloudExtent: 1.9,
      erosion: 0.4,
      turbulence: 0.65,
      detail: 0.7,
      lightAbsorption: 1.35,
      fog: true,
      fogReference: "bottom",
      fogHeight: 8,
      fogDensity: 0.12,
      fogFalloff: 22,
      fogColor: "#b7d9ec",
    },
  }),
  cloudGolden: look({
    timeOfDay: 17.25,
    sun: 3.6,
    ambient: 0.55,
    sunLinked: false,
    elevation: 24,
    azimuth: 225,
    sky: "sunset",
    warmth: "#ffd29a",
    saturation: 1.12,
    vibrance: 0.16,
    contrast: 1.1,
    bloom: 0.16,
    temperature: 0.08,
    highlights: -0.2,
    shadows: 0.06,
    quality: "fine",
    atmosphere: {
      ...atmosphereDefaults,
      clouds: true,
      cloudModel: "banks",
      cloudType: "cumulus",
      cloudReference: "bottom",
      cloudHeight: -28,
      cloudThickness: 150,
      cloudSize: 145,
      coverage: 0.68,
      cloudDensity: 1.25,
      cloudExtent: 2.2,
      erosion: 0.36,
      turbulence: 0.6,
      lightAbsorption: 1.7,
      fog: true,
      fogReference: "bottom",
      fogHeight: 6,
      fogDensity: 0.16,
      fogFalloff: 28,
      fogColor: "#e6cdd5",
      fogScattering: 0.9,
    },
  }),
  cloudSoft: look({
    timeOfDay: 9,
    sun: 2.5,
    ambient: 0.85,
    sunLinked: false,
    elevation: 48,
    azimuth: 110,
    sky: "daylight",
    warmth: "#fff6ed",
    saturation: 1.1,
    vibrance: 0.18,
    contrast: 1.04,
    bloom: 0.1,
    quality: "fine",
    atmosphere: {
      ...atmosphereDefaults,
      clouds: true,
      cloudModel: "banks",
      cloudReference: "bottom",
      cloudHeight: -32,
      cloudThickness: 110,
      cloudSize: 120,
      coverage: 0.43,
      cloudDensity: 0.8,
      cloudExtent: 1.7,
      turbulence: 0.7,
      erosion: 0.55,
      fog: true,
      fogReference: "bottom",
      fogHeight: 18,
      fogDensity: 0.2,
      fogFalloff: 30,
      fogVariation: 0.85,
      fogType: "banks",
      fogColor: "#c0dce8",
    },
  }),
  island: look({
    sun: 3.2,
    ambient: 0.7,
    environment: 0.9,
    sunLinked: false,
    azimuth: 135,
    elevation: 48,
    saturation: 1.22,
    vibrance: 0.3,
    contrast: 1.1,
    highlights: -0.16,
    shadows: 0.12,
    sharpen: 0.2,
    bloom: 0.08,
    ao: 0.55,
  }),
  garden: look({
    sun: 2.9,
    ambient: 0.7,
    environment: 0.85,
    sunLinked: false,
    elevation: 40,
    saturation: 1.15,
    vibrance: 0.32,
    contrast: 1.13,
    shadows: 0.06,
    highlights: -0.12,
    sharpen: 0.18,
    ao: 0.6,
    bloom: 0.1,
  }),
  film: look({
    timeOfDay: 17.5,
    sun: 2.4,
    ambient: 0.7,
    environment: 0.7,
    sky: "dawn",
    sunLinked: false,
    azimuth: 225,
    elevation: 28,
    warmth: "#ffe2ba",
    temperature: 0.18,
    saturation: 0.98,
    vibrance: 0.12,
    contrast: 1.1,
    highlights: -0.3,
    shadows: 0.1,
    sharpen: 0.1,
    vignette: 0.12,
    bloom: 0.16,
  }),
  winterevening: look({
    timeOfDay: 17,
    sun: 2.6,
    ambient: 0.5,
    environment: 0.65,
    sky: "sunset",
    sunLinked: false,
    azimuth: 230,
    elevation: 22,
    warmth: "#ffd6a0",
    temperature: 0.08,
    saturation: 1.12,
    vibrance: 0.2,
    contrast: 1.12,
    highlights: -0.2,
    shadows: 0.04,
    sharpen: 0.14,
    bloom: 0.24,
    vignette: 0.1,
    ao: 0.6,
  }),
  crisp: look({
    sun: 2.8,
    ambient: 0.8,
    environment: 0.85,
    sunLinked: false,
    elevation: 45,
    warmth: "#fff9f1",
    saturation: 1,
    vibrance: 0.06,
    contrast: 1.04,
    sharpen: 0.16,
    bloom: 0.04,
  }),
  daylight: look({}),
  soft: look({
    sun: 1.1,
    ambient: 1.5,
    environment: 1.05,
    softness: 4,
    sky: "dawn",
    warmth: "#fff7ef",
    ao: 0.35,
  }),
  tropical: look({
    elevation: 65,
    sun: 3.6,
    ambient: 1.25,
    environment: 1,
    exposure: 1.05,
    warmth: "#fff4d9",
    ao: 0.4,
  }),
  golden: look({
    timeOfDay: 17,
    azimuth: 225,
    elevation: 22,
    sun: 3.2,
    ambient: 1.15,
    environment: 0.8,
    sky: "dawn",
    warmth: "#ffd4a1",
    softness: 3,
    exposure: 1.05,
  }),
  sunset: look({
    timeOfDay: 18.5,
    azimuth: 250,
    elevation: 12,
    sun: 2.4,
    ambient: 0.95,
    sky: "sunset",
    environment: 0.75,
    warmth: "#ffa977",
    softness: 4,
  }),
  bluehour: look({
    timeOfDay: 19.25,
    azimuth: 260,
    elevation: 18,
    sun: 0.65,
    ambient: 0.85,
    sky: "sunset",
    skyBrightness: 0.45,
    environment: 0.55,
    warmth: "#bacfff",
    exposure: 1.1,
    ao: 0.35,
  }),
  night: look({
    timeOfDay: 22,
    azimuth: 60,
    elevation: 50,
    sun: 0.8,
    ambient: 0.55,
    sky: "night",
    skyBrightness: 0.45,
    environment: 0.5,
    warmth: "#a8c7ff",
    exposure: 1.15,
    ao: 0.3,
  }),
  winter: look({
    timeOfDay: 10,
    elevation: 30,
    sun: 2,
    ambient: 1.4,
    sky: "dawn",
    warmth: "#e6efff",
    environment: 1,
    softness: 3,
    exposure: 1.05,
  }),
  dramatic: look({
    timeOfDay: 17,
    azimuth: 210,
    elevation: 25,
    sun: 4.2,
    ambient: 0.65,
    environment: 0.4,
    sky: "sunset",
    warmth: "#ffd4af",
    ao: 0.6,
    softness: 1.5,
  }),
  architecture: look({
    timeOfDay: 11,
    azimuth: 135,
    elevation: 50,
    sun: 2.5,
    ambient: 1.5,
    environment: 1,
    ao: 0.5,
    softness: 2,
    sky: "daylight",
  }),
  studio: look({
    azimuth: 45,
    elevation: 60,
    sun: 2.3,
    ambient: 1.8,
    environment: 0.5,
    ao: 0.35,
    sky: "none",
    warmth: "#fffaf2",
    background: "#e4e6e6",
    softness: 4,
  }),
  neutral: look({
    azimuth: 135,
    elevation: 55,
    sun: 1.5,
    ambient: 2,
    environment: 0.65,
    ao: 0.25,
    sky: "none",
    warmth: "#ffffff",
    background: "#dce6e9",
  }),
};
export const renderQuality = {
  fast: {
    label: "Fast",
    samples: 8,
    denoise: 8,
    aoScale: 0.5,
    shadow: 1024,
    aa: 2,
  },
  balanced: {
    label: "Balanced",
    samples: 16,
    denoise: 16,
    aoScale: 0.75,
    shadow: 2048,
    aa: 4,
  },
  fine: {
    label: "Fine",
    samples: 32,
    denoise: 24,
    aoScale: 1,
    shadow: 4096,
    aa: 8,
  },
};
export function applyRenderPreset(
  current: RenderSettings,
  id: string,
): RenderSettings {
  const preset = renderPresets[id];
  if (!preset) return current;
  return {
    ...renderDefaults,
    ...preset,
    edge: current.edge,
    aspect: current.aspect,
    transparent: current.transparent,
    pitchLocked: current.pitchLocked,
    studioLights: current.studioLights,
    dofEnabled: current.dofEnabled,
    dofDistance: current.dofDistance,
    dofAperture: current.dofAperture,
    dofFocalLength: current.dofFocalLength,
    dofWorldScale: current.dofWorldScale,
    dofMaxBlur: current.dofMaxBlur,
    atmosphere: { ...atmosphereDefaults, ...preset.atmosphere },
  };
}
export function renderDimensions(edge: number, aspect: number, limit = 4096) {
  if (
    ![edge, aspect, limit].every(Number.isFinite) ||
    edge < 64 ||
    edge > limit ||
    aspect < 0.25 ||
    aspect > 4
  )
    throw new Error("Invalid image resolution");
  const width = Math.round(aspect >= 1 ? edge : edge * aspect);
  const height = Math.round(aspect >= 1 ? edge / aspect : edge);
  if (width > limit || height > limit)
    throw new Error("This resolution exceeds the graphics limit");
  return { width, height };
}

import type { RenderSettings } from "./render-settings";
export type MapMarker = {
  id: string;
  x: number;
  z: number;
  label: string;
  icon: string;
  color: string;
  background: string;
  scale: number;
};
export type MapPresentation = {
  enabled: boolean;
  rotation: number;
  zoom: number;
  grid: boolean;
  gridSize: number;
  gridOpacity: number;
  contours: boolean;
  contourStep: number;
  heightTint: number;
  shading: boolean;
  saturation: number;
  background: string;
  transparent: boolean;
  markers: MapMarker[];
};
export const mapDefaults: MapPresentation = {
  enabled: false,
  rotation: 0,
  zoom: 1,
  grid: false,
  gridSize: 16,
  gridOpacity: 0.35,
  contours: false,
  contourStep: 16,
  heightTint: 0,
  shading: true,
  saturation: 1,
  background: "#e9ecef",
  transparent: true,
  markers: [],
};
export function mapRenderSettings(
  base: RenderSettings,
  map: MapPresentation,
): RenderSettings {
  if (!map.enabled) return base;
  return {
    ...base,
    sky: "none",
    sunLinked: false,
    background: map.background,
    transparent: true,
    azimuth: 315,
    elevation: 65,
    sun: map.shading ? 2 : 0,
    ambient: map.shading ? 1.5 : 3,
    environment: 0,
    ao: map.shading ? 0.2 : 0,
    exposure: 1,
    grading: true,
    saturation: map.saturation,
    vibrance: 0,
    contrast: 1.02,
    temperature: 0,
    tint: 0,
    highlights: -0.12,
    shadows: 0.08,
    vignette: 0,
    bloom: 0,
    blockLights: false,
    studioSpot: false,
    studioLights: [],
    emissionPower: 0,
    volumeLightPower: 0,
    atmosphere: {
      ...base.atmosphere,
      clouds: false,
      fog: false,
      cloudLayers: [],
    },
  };
}

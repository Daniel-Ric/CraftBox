import type { RenderSettings } from "./render-settings";
const smooth = (a: number, b: number, x: number) => {
  const t = Math.max(0, Math.min(1, (x - a) / (b - a)));
  return t * t * (3 - 2 * t);
};
export function formatDaytime(hour: number) {
  const minutes = Math.round(hour * 60) % 1440;
  return `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`;
}

export function applyTimeOfDay(
  settings: RenderSettings,
  hour: number,
): RenderSettings {
  if (!Number.isFinite(hour) || hour < 0 || hour > 24)
    throw new Error("Invalid time of day");
  const angle = ((hour - 6) / 24) * Math.PI * 2;
  const solarElevation = (Math.asin(Math.sin(angle) * 0.94) * 180) / Math.PI;
  const day = smooth(-8, 20, solarElevation),
    gold = 1 - smooth(8, 38, solarElevation),
    night = solarElevation < -6;
  return {
    ...settings,
    timeOfDay: hour,
    timeLinked: true,
    sunLinked: false,
    azimuth: (90 + (hour - 6) * 15 + (night ? 180 : 0) + 720) % 360,
    elevation: Math.max(
      5,
      Math.min(85, night ? -solarElevation : solarElevation),
    ),
    sun: night ? 0.18 : 3.3 * day,
    ambient: 0.2 + day * 0.65,
    environment: 0.16 + day * 0.75,
    sky: night
      ? "night"
      : solarElevation < 12
        ? "sunset"
        : solarElevation < 28
          ? "dawn"
          : "daylight",
    skyRotation: (90 + (hour - 6) * 15 + 720) % 360,
    skyBrightness: night ? 0.28 : 0.6 + day * 0.4,
    warmth: night
      ? "#b6ceff"
      : gold > 0.65
        ? "#ffcb94"
        : gold > 0.25
          ? "#ffe2bc"
          : "#fff2df",
    background: night ? "#17283f" : gold > 0.65 ? "#ddc1af" : "#c6dce8",
    temperature: night ? -0.12 : gold * 0.12,
  };
}

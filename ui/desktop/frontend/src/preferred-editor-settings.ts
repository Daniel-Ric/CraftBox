import { loadWorkspacePreferences } from "./workspace-preferences";
import { renderDefaults } from "./render-settings";
export function preferredRenderSettings() {
  const p = loadWorkspacePreferences();
  const [w, h] = p.outputAspect.split(":").map(Number);
  return {
    ...renderDefaults,
    quality: p.renderQuality,
    edge: p.outputEdge,
    aspect: w / h,
    transparent: p.transparentOutput,
  };
}
export function preferredFlightSettings() {
  const p = loadWorkspacePreferences();
  return {
    speed: p.flightSpeed,
    fov: p.flightFov,
    sensitivity: p.flightSensitivity,
    smoothing: p.flightSmoothing,
    level: p.flightLevel,
  };
}

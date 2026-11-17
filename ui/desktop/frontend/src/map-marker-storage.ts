import type { CapturePoint } from "./bridge";
import type { MapMarker } from "./map-presentation";

export function readAtlasSettings(key: string): Record<string, unknown> {
  try {
    const data = JSON.parse(
      localStorage.getItem(`CraftBox-atlas:${key}`) ?? "{}",
    );
    return data && typeof data === "object" && !Array.isArray(data) ? data : {};
  } catch {
    return {};
  }
}

export function capturePointMarker(point: CapturePoint): MapMarker {
  return {
    id: point.id,
    x: point.position[0],
    z: point.position[2],
    label: point.name,
    icon: point.icon,
    color: "#f1cf78",
    background: "#202020",
    scale: 1,
  };
}

export function readWorldMapMarkers(
  worldId: string,
  capturePoints: readonly CapturePoint[],
) {
  const saved = readAtlasSettings(worldId);
  const markers: MapMarker[] = Array.isArray(saved.markers)
    ? saved.markers.slice(0, 512)
    : [];
  const seen = new Set(
    Array.isArray(saved.captureImports) ? saved.captureImports : [],
  );
  for (const marker of markers) seen.add(marker?.id);
  for (const point of capturePoints) {
    if (markers.length >= 512) break;
    if (seen.has(point.id)) continue;
    seen.add(point.id);
    markers.push(capturePointMarker(point));
  }
  return markers;
}

import StudioInput from "@/StudioInput";
import { mapDefaults, type MapPresentation } from "./map-presentation";
import MapMarkerControls from "./MapMarkerControls";
import "./map-controls.css";
export default function MapPresentationControls({
  value,
  onChange,
  onPlace,
  placing,
  onCancel,
}: {
  value: MapPresentation;
  onChange: (s: MapPresentation) => void;
  onPlace: (id?: string) => void;
  placing: boolean;
  onCancel?: () => void;
}) {
  const set = <K extends keyof MapPresentation>(
    key: K,
    v: MapPresentation[K],
  ) => onChange({ ...value, [key]: v });
  const number = (
    label: string,
    key:
      | "rotation"
      | "zoom"
      | "gridSize"
      | "gridOpacity"
      | "contourStep"
      | "heightTint"
      | "saturation",
    min: number,
    max: number,
    step: number,
  ) => (
    <label className="render-range">
      {label}
      <StudioInput
        type="number"
        aria-label={label}
        value={value[key]}
        min={min}
        max={max}
        increment={step}
        onChange={(e) => {
          if (e.target.value && Number.isFinite(+e.target.value))
            set(key, Math.min(max, Math.max(min, +e.target.value)));
        }}
      />
    </label>
  );
  const check = (
    label: string,
    key: "grid" | "contours" | "shading" | "transparent",
  ) => (
    <label className="render-check">
      <StudioInput
        type="checkbox"
        checked={value[key]}
        onChange={(e) => set(key, e.target.checked)}
      />
      {label}
    </label>
  );
  return (
    <>
      <header className="inspector-heading">
        <div>
          <strong>2D map styling</strong>
          <span>Orthographic top view</span>
        </div>
      </header>
      <div
        className="inspector-content map-presentation-controls"
        style={{ overflow: "auto", padding: 14 }}
      >
        <fieldset>
          <legend>Framing and orientation</legend>
          {number("Map rotation", "rotation", 0, 360, 1)}
          {number("Map zoom", "zoom", 0.1, 20, 0.1)}
          <button onClick={() => onChange({ ...value, rotation: 0, zoom: 1 })}>
            Align north
          </button>
          <p>Drag to pan the map. Scroll to zoom.</p>
        </fieldset>
        <fieldset>
          <legend>Terrain appearance</legend>
          {check("Terrain shading", "shading")}
          {number("Map saturation", "saturation", 0, 2, 0.05)}
          {number("Elevation tint strength", "heightTint", 0, 1, 0.05)}
          {check("Show contour lines", "contours")}
          {number("Contour interval", "contourStep", 1, 128, 1)}
        </fieldset>
        <fieldset>
          <legend>Map grid</legend>
          {check("Show grid", "grid")}
          {number("Grid spacing in blocks", "gridSize", 1, 256, 1)}
          {number("Grid opacity", "gridOpacity", 0, 1, 0.05)}
        </fieldset>
        <MapMarkerControls
          markers={value.markers}
          onChange={(markers) => set("markers", markers)}
          onPlace={onPlace}
          placing={placing}
          onCancel={onCancel}
        />
        <p>Transparent map background.</p>
        <button
          onClick={() =>
            onChange({ ...mapDefaults, enabled: true, markers: value.markers })
          }
        >
          Reset map styling
        </button>
      </div>
    </>
  );
}

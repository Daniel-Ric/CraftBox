import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { MapMarker } from "./map-presentation";
export default function MapMarkerControls({
  markers,
  onChange,
  onPlace,
  placing,
  onCancel,
}: {
  markers: MapMarker[];
  onChange: (markers: MapMarker[]) => void;
  onPlace: (id?: string) => void;
  placing: boolean;
  onCancel?: () => void;
}) {
  const [selected, setSelected] = useState("");
  const current = markers.find((m) => m.id === selected) ?? markers.at(-1);
  const update = (patch: Partial<MapMarker>) => {
    if (current)
      onChange(
        markers.map((m) => (m.id === current.id ? { ...m, ...patch } : m)),
      );
  };
  return (
    <fieldset>
      <legend>Map markers</legend>
      <Button
        variant="outline"
        disabled={markers.length >= 512}
        onClick={() => onPlace()}
      >
        Place marker on map
      </Button>
      {placing && (
        <>
          <p role="status">Click the desired position on the map.</p>
          <Button variant="ghost" onClick={onCancel}>
            Cancel placement
          </Button>
        </>
      )}
      {markers.length > 0 && (
        <>
          <label>
            Select marker
            <select
              aria-label="Select marker"
              value={current?.id ?? ""}
              onChange={(e) => setSelected(e.target.value)}
            >
              {markers.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.label || m.icon || "Marker"}
                </option>
              ))}
            </select>
          </label>
          {current && (
            <>
              <label>
                Label
                <input
                  aria-label="Marker label"
                  value={current.label}
                  maxLength={100}
                  onChange={(e) => update({ label: e.target.value })}
                />
              </label>
              <label>
                Symbol
                <input
                  aria-label="Marker symbol"
                  value={current.icon}
                  maxLength={8}
                  onChange={(e) => update({ icon: e.target.value })}
                />
              </label>
              <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                {["📍", "🏠", "⚔", "★", "⛳", "🛒"].map((icon) => (
                  <button
                    key={icon}
                    aria-label={`Symbol ${icon}`}
                    onClick={() => update({ icon })}
                  >
                    {icon}
                  </button>
                ))}
              </div>
              <label className="render-color">
                Marker color
                <input
                  aria-label="Marker color"
                  type="color"
                  value={current.color}
                  onChange={(e) => update({ color: e.target.value })}
                />
              </label>
              <label className="render-color">
                Marker background
                <input
                  aria-label="Marker background"
                  type="color"
                  value={current.background}
                  onChange={(e) => update({ background: e.target.value })}
                />
              </label>
              <label className="render-range">
                Marker size
                <input
                  aria-label="Marker size"
                  type="number"
                  value={current.scale}
                  min={0.5}
                  max={3}
                  step={0.1}
                  onChange={(e) => {
                    if (e.target.value)
                      update({
                        scale: Math.min(3, Math.max(0.5, +e.target.value)),
                      });
                  }}
                />
              </label>
              <div style={{ display: "flex", gap: 8 }}>
                {(["x", "z"] as const).map((axis) => (
                  <label key={axis}>
                    World-{axis.toUpperCase()}
                    <input
                      aria-label={`Marker ${axis.toUpperCase()}`}
                      type="number"
                      value={current[axis]}
                      onChange={(e) => {
                        if (e.target.value && Number.isFinite(+e.target.value))
                          update({ [axis]: +e.target.value });
                      }}
                    />
                  </label>
                ))}
              </div>
              <Button variant="outline" onClick={() => onPlace(current.id)}>
                Reposition
              </Button>
              <Button
                variant="ghost"
                onClick={() =>
                  onChange(markers.filter((m) => m.id !== current.id))
                }
              >
                Delete marker
              </Button>
            </>
          )}
        </>
      )}
      <p>Markers are included in PNG exports and saved in the project.</p>
    </fieldset>
  );
}

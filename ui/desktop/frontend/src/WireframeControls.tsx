import type { WireframeSettings, WireframeStats } from "./wireframe-settings";
export default function WireframeControls(p: {
  enabled: boolean;
  settings: WireframeSettings;
  stats?: WireframeStats;
  onToggle: (value: boolean) => void;
  onChange: (value: WireframeSettings) => void;
}) {
  const patch = (value: WireframeSettings) =>
    p.onChange({ ...p.settings, ...value });
  const range = (
    label: string,
    key:
      | "width"
      | "opacity"
      | "thresholdAngle"
      | "hiddenOpacity"
      | "vertexSize"
      | "dashScale",
    min: number,
    max: number,
    step: number,
    fallback: number,
    format: (n: number) => string = (n) => String(n),
  ) => (
    <label>
      <span>
        {label}
        <output>{format(p.settings[key] ?? fallback)}</output>
      </span>
      <input
        aria-label={`Wireframe ${label.toLowerCase()}`}
        type="range"
        min={min}
        max={max}
        step={step}
        value={p.settings[key] ?? fallback}
        onChange={(e) => patch({ [key]: Number(e.target.value) })}
      />
    </label>
  );
  const color = (
    label: string,
    key: "color" | "hiddenColor" | "vertexColor",
    fallback: string,
  ) => (
    <label>
      {label}
      <input
        aria-label={`Wireframe ${label.toLowerCase()}`}
        type="color"
        value={p.settings[key] ?? fallback}
        onChange={(e) => patch({ [key]: e.target.value })}
      />
    </label>
  );
  return (
    <div className="asset-wireframe-controls">
      <label className="asset-check">
        <input
          type="checkbox"
          checked={p.enabled}
          onChange={(e) => p.onToggle(e.target.checked)}
        />
        Wireframe overlay
      </label>
      {p.enabled && (
        <div className="asset-wireframe-options">
          <div
            className="asset-wireframe-presets"
            aria-label="Wireframe presets"
          >
            <button
              type="button"
              onClick={() =>
                patch({
                  hideSurface: false,
                  mode: "edges",
                  xray: false,
                  vertices: false,
                })
              }
            >
              Overlay
            </button>
            <button
              type="button"
              onClick={() =>
                patch({ hideSurface: true, mode: "edges", xray: true })
              }
            >
              Wire only
            </button>
            <button
              type="button"
              onClick={() =>
                patch({
                  hideSurface: true,
                  mode: "triangles",
                  xray: false,
                  vertices: true,
                })
              }
            >
              Topology
            </button>
          </div>
          <label className="asset-check">
            <input
              type="checkbox"
              checked={!!p.settings.hideSurface}
              onChange={(e) => patch({ hideSurface: e.target.checked })}
            />
            Hide model surfaces · wireframe only
          </label>
          <label>
            Edges
            <select
              aria-label="Wireframe style"
              value={p.settings.mode ?? "edges"}
              onChange={(e) =>
                patch({ mode: e.target.value as "edges" | "triangles" })
              }
            >
              <option value="edges">Model edges</option>
              <option value="triangles">Triangles</option>
            </select>
          </label>
          {p.settings.mode !== "triangles" &&
            range(
              "Edge angle",
              "thresholdAngle",
              0,
              180,
              1,
              15,
              (n) => `${n}°`,
            )}
          {color("Color", "color", "#58bdff")}
          {range("Line width", "width", 0.5, 6, 0.25, 1.25, (n) => `${n} px`)}
          {range(
            "Opacity",
            "opacity",
            0.05,
            1,
            0.05,
            0.85,
            (n) => `${Math.round(n * 100)}%`,
          )}
          <label className="asset-check">
            <input
              type="checkbox"
              checked={!!p.settings.xray}
              onChange={(e) => patch({ xray: e.target.checked })}
            />
            X-ray · show hidden edges
          </label>
          {p.settings.xray && (
            <div className="asset-wireframe-section">
              {color("Hidden edge color", "hiddenColor", "#63839a")}
              {range(
                "Hidden opacity",
                "hiddenOpacity",
                0.05,
                1,
                0.05,
                0.25,
                (n) => `${Math.round(n * 100)}%`,
              )}
              <label className="asset-check">
                <input
                  type="checkbox"
                  checked={p.settings.dashedHidden !== false}
                  onChange={(e) => patch({ dashedHidden: e.target.checked })}
                />
                Dashed hidden edges
              </label>
              {p.settings.dashedHidden !== false &&
                range(
                  "Dash density",
                  "dashScale",
                  0.25,
                  8,
                  0.25,
                  1,
                  (n) => `${n}×`,
                )}
            </div>
          )}
          <label className="asset-check">
            <input
              type="checkbox"
              checked={!!p.settings.vertices}
              onChange={(e) => patch({ vertices: e.target.checked })}
            />
            Show vertices
          </label>
          {p.settings.vertices && (
            <div className="asset-wireframe-section">
              {color("Vertex color", "vertexColor", "#ffd166")}
              {range(
                "Vertex size",
                "vertexSize",
                1,
                10,
                0.5,
                4,
                (n) => `${n} px`,
              )}
            </div>
          )}
          {p.stats && (
            <p className="asset-wireframe-stats">
              {p.stats.meshes.toLocaleString()} meshes ·{" "}
              {p.stats.segments.toLocaleString()} edges
              {p.settings.vertices
                ? ` · ${p.stats.vertices.toLocaleString()} vertices`
                : ""}
              {p.stats.skipped > 0
                ? ` · ${p.stats.skipped} meshes exceed the preview budget`
                : ""}
            </p>
          )}
          <button
            className="asset-wireframe-reset"
            type="button"
            onClick={() => p.onChange({})}
          >
            Reset wireframe settings
          </button>
          <small className="asset-wireframe-note">
            Preview settings preserve the model textures and exported materials.
          </small>
        </div>
      )}
    </div>
  );
}

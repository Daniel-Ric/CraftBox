import {
  atmosphereDefaults,
  cloudTypes,
  fogTypes,
  type AtmosphereSettings,
} from "./atmosphere-settings";
import { useState } from "react";
import AtmosphereLibraryDialog from "./AtmosphereLibraryDialog";
import AtmosphereHeightControl from "./AtmosphereHeightControl";
import CloudLayerControls, {
  type CloudEditorState,
  type CloudAnimationContext,
} from "./CloudLayerControls";
import "./atmosphere-controls.css";
export default function AtmosphereControls({
  value,
  onChange,
  disabled,
  cloudEditor,
  onCloudPlace,
  onCloudMode,
  animation,
}: {
  value: AtmosphereSettings;
  onChange: (value: AtmosphereSettings) => void;
  disabled: boolean;
  cloudEditor?: CloudEditorState;
  onCloudPlace?: () => void;
  onCloudMode?: (mode: CloudEditorState["mode"]) => void;
  animation?: CloudAnimationContext;
}) {
  const [library, setLibrary] = useState<"cloud" | "fog" | null>(null);
  const set = <K extends keyof AtmosphereSettings>(
    key: K,
    v: AtmosphereSettings[K],
  ) => onChange({ ...value, [key]: v });
  const number = (
    label: string,
    key: keyof AtmosphereSettings,
    min: number,
    max: number,
    step = 1,
    unit = "",
  ) =>
    key === "cloudHeight" || key === "fogHeight" ? (
      <AtmosphereHeightControl
        label={label}
        value={value[key]}
        reference={
          value[key === "cloudHeight" ? "cloudReference" : "fogReference"]
        }
        onValue={(n) => set(key, n)}
        onReference={(r) =>
          set(key === "cloudHeight" ? "cloudReference" : "fogReference", r)
        }
      />
    ) : (
      <label className="render-control">
        {label}
        {unit && <small>{unit}</small>}
        <input
          aria-label={label}
          type="number"
          min={min}
          max={max}
          step={step}
          value={value[key] as number}
          onChange={(e) => {
            const n = e.currentTarget.valueAsNumber;
            if (Number.isFinite(n)) set(key, Math.min(max, Math.max(min, n)));
          }}
        />
      </label>
    );
  const range = (label: string, key: keyof AtmosphereSettings, max = 1) => (
    <label className="render-control">
      {label}
      <output>{Number(value[key]).toFixed(2)}</output>
      <input
        aria-label={label}
        type="range"
        min={0}
        max={max}
        step={0.01}
        value={value[key] as number}
        onChange={(e) => set(key, +e.target.value)}
      />
    </label>
  );
  const color = (label: string, key: "fogColor" | "cloudColor") => (
    <label className="render-color">
      {label}
      <input
        aria-label={label}
        type="color"
        value={value[key]}
        onChange={(e) => set(key, e.target.value)}
      />
    </label>
  );
  return (
    <>
      <CloudLayerControls
        value={value}
        onChange={onChange}
        disabled={disabled}
        editor={cloudEditor}
        onPlace={onCloudPlace}
        onMode={onCloudMode}
        animation={animation}
      />
      <fieldset
        disabled={disabled}
        data-inspector="atmosphere"
        className="atmosphere-library-launchers"
      >
        <legend>Atmosphere libraries</legend>
        <button
          aria-label="Open cloud library"
          onClick={() => setLibrary("cloud")}
        >
          Cloud library · 14 presets
        </button>
        <button
          aria-label="Open fog library"
          onClick={() => setLibrary("fog")}
        >
          Fog library · 12 presets
        </button>
      </fieldset>
      <fieldset disabled={disabled} data-inspector="atmosphere">
        <legend>Height fog</legend>
        <label className="render-check">
          <input
            type="checkbox"
            checked={value.fog}
            onChange={(e) => set("fog", e.target.checked)}
          />
          Volumetric fog
        </label>
        <label className="render-control">
          Fog type
          <select
            aria-label="Fog type"
            value={value.fogType}
            onChange={(e) =>
              set("fogType", e.target.value as AtmosphereSettings["fogType"])
            }
          >
            {Object.entries(fogTypes).map(([id, label]) => (
              <option value={id} key={id}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <p>
          Light-filled haze with volumetric wisps. Buildings and terrain occlude fog behind them.
        </p>
        {color("Fog color", "fogColor")}
        {range("Fog density", "fogDensity", 3)}
        {number("Fog height", "fogHeight", -512, 2048, 1, "blocks · world Y")}
        {number("Height falloff", "fogFalloff", 1, 512, 1, "blocks")}
        {range("Fog wisps", "fogVariation")}
        {range("Fog turbulence", "fogTurbulence")}
        {number("Fog seed", "fogSeed", 0, 999999)}
        <button
          onClick={() => set("fogSeed", Math.floor(Math.random() * 999999))}
        >
          Randomize fog
        </button>
        {number("Wisp size", "fogSize", 8, 1024, 1, "blocks")}
        {number("Clear near range", "fogStart", 0, 2048, 1, "blocks")}
        {range("Fog light scattering", "fogScattering", 2)}
        {range("Fog backlighting", "fogAnisotropy", 0.9)}
      </fieldset>
      <fieldset disabled={disabled} data-inspector="atmosphere">
        <legend>Global cloud base layer</legend>
        <label className="render-check">
          <input
            type="checkbox"
            checked={value.clouds}
            onChange={(e) => set("clouds", e.target.checked)}
          />
          Volumetric clouds
        </label>
        <label className="render-control">
          Cloud type
          <select
            aria-label="Cloud type"
            value={value.cloudType}
            onChange={(e) =>
              set(
                "cloudType",
                e.target.value as AtmosphereSettings["cloudType"],
              )
            }
          >
            {Object.entries(cloudTypes).map(([id, label]) => (
              <option value={id} key={id}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label className="render-control">
          Cloud model
          <select
            aria-label="Cloud model"
            value={value.cloudModel}
            onChange={(e) =>
              set(
                "cloudModel",
                e.target.value as AtmosphereSettings["cloudModel"],
              )
            }
          >
            <option value="banks">Sea of clouds · volumetric cloud banks</option>
            <option value="natural">Natural · turbulent clouds</option>
            <option value="bodies">Solid · compact clouds</option>
            <option value="classic">Classic · previous rendering</option>
          </select>
        </label>
        {color("Cloud color", "cloudColor")}
        {range("Cloud density", "cloudDensity", 4)}
        {range("Cloud cover", "coverage")}
        {number(
          "Cloud height",
          "cloudHeight",
          -512,
          2048,
          1,
          "blocks · bottom edge in world Y",
        )}
        {number("Cloud thickness", "cloudThickness", 2, 512, 1, "blocks")}
        {number("Cloud size", "cloudSize", 8, 1024, 1, "blocks")}
        {range("Cloud shadows on terrain", "cloudShadows")}
        {range("Keep clouds clear of the model", "cloudAvoidance")}
        {number(
          "Soft model clearance",
          "cloudClearance",
          1,
          128,
          1,
          "blocks · foreground clouds are preserved",
        )}
        {range("Structure detail", "detail")}
        {range("Edge erosion", "erosion")}
        {range("Distortion", "turbulence")}
        {range("Cloud clustering", "cloudClumping")}
        {range("Cloud roundness", "cloudRoundness")}
        {number(
          "Cloud extent around model",
          "cloudExtent",
          0.5,
          8,
          0.1,
          "factor · higher values extend farther",
        )}
        {range("Light absorption", "lightAbsorption", 3)}
        {number("Cloud seed", "cloudSeed", 0, 999999)}
        <button
          onClick={() => set("cloudSeed", Math.floor(Math.random() * 999999))}
        >
          Randomize clouds
        </button>
      </fieldset>
      <fieldset disabled={disabled} data-inspector="atmosphere">
        <legend>Wind &amp; time</legend>
        {number(
          "Wind speed",
          "windSpeed",
          0,
          40,
          0.1,
          "blocks / second",
        )}
        {number("Wind direction", "windDirection", 0, 360, 1, "degrees")}
        {number(
          "Atmosphere range",
          "distance",
          512,
          30000,
          128,
          "blocks · with a soft distance fade",
        )}
        <label className="render-check">
          <input
            type="checkbox"
            checked={value.animate}
            onChange={(e) => set("animate", e.target.checked)}
          />
          Animate atmosphere
        </label>
        {number(
          "Atmosphere time",
          "time",
          0,
          3600,
          0.1,
          "seconds · start offset",
        )}
        <p>
          Heights use the selected model reference point or world Y coordinate. Sequence time drives wind and wisps frame by frame during playback and export. Fine quality increases volumetric detail.
        </p>
        <button onClick={() => onChange({ ...atmosphereDefaults })}>
          Reset atmosphere
        </button>
      </fieldset>
      <AtmosphereLibraryDialog
        kind={library}
        onClose={() => setLibrary(null)}
        onApply={(p) => onChange({ ...value, ...p.settings })}
      />
    </>
  );
}

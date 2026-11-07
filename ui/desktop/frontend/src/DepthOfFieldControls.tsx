import StudioInput from "@/StudioInput";
import type { RenderSettings } from "./render-settings";

export default function DepthOfFieldControls({
  value,
  onChange,
  disabled,
  picking,
  onPick,
  guides = true,
  onGuides,
}: {
  value: RenderSettings;
  onChange: (value: RenderSettings) => void;
  disabled: boolean;
  picking: boolean;
  onPick: () => void;
  guides?: boolean;
  onGuides?: (value: boolean) => void;
}) {
  const number = (
    label: string,
    key:
      | "dofDistance"
      | "dofAperture"
      | "dofFocalLength"
      | "dofWorldScale"
      | "dofNearStrength"
      | "dofFarStrength"
      | "dofFocusRange"
      | "dofTransition"
      | "dofMaxBlur",

    min: number,
    max: number,
    step: number,
  ) => (
    <label className="render-control">
      {label}
      <StudioInput
        aria-label={label}
        type="number"
        min={min}
        max={max}
        increment={step}
        disabled={key === "dofDistance" && value.dofTargetLocked}
        value={Number(value[key].toFixed(4))}
        onChange={(e) => {
          if (e.target.value && Number.isFinite(+e.target.value))
            onChange({
              ...value,
              [key]: Math.max(min, Math.min(max, +e.target.value)),
              ...(key === "dofDistance" ? { dofTargetLocked: false } : {}),
            });
        }}
      />
    </label>
  );
  return (
    <fieldset disabled={disabled} data-inspector="detail">
      <legend>Depth of field</legend>
      <label className="render-check">
        <StudioInput
          type="checkbox"
          aria-label="Enable depth of field"
          checked={value.dofEnabled}
          onChange={(e) => onChange({ ...value, dofEnabled: e.target.checked })}
        />
        Enable depth of field
      </label>
      <button
        type="button"
        className="inspector-next"
        aria-pressed={picking}
        onClick={onPick}
      >
        {picking
          ? "Cancel selection"
          : value.dofTargetLocked
            ? "Change subject"
            : "Pick subject"}
      </button>
      {picking && (
        <p role="status">Hover to preview · Click to focus · Esc to cancel</p>
      )}
      {value.dofTargetLocked && (
        <p className="focus-target-summary">
          Focus locked · X {value.dofTargetX.toFixed(1)} · Y{" "}
          {value.dofTargetY.toFixed(1)} · Z {value.dofTargetZ.toFixed(1)}
        </p>
      )}
      {onGuides && (
        <label className="render-check">
          <StudioInput
            type="checkbox"
            aria-label="Show focus guides"
            checked={guides}
            onChange={(e) => onGuides(e.target.checked)}
          />
          Show focus guides
        </label>
      )}
      <label className="render-control">
        Focus mode
        <select
          aria-label="Focus mode"
          value={value.dofMode}
          onChange={(e) =>
            onChange({
              ...value,
              dofMode: e.target.value as RenderSettings["dofMode"],
            })
          }
        >
          <option value="depth">Depth plane</option>
          <option value="subject">Subject isolation</option>
          <option value="lens">Physical lens</option>
        </select>
      </label>
      {value.dofTargetLocked && (
        <button
          type="button"
          className="inspector-next"
          onClick={() => onChange({ ...value, dofTargetLocked: false })}
        >
          Unlock focus
        </button>
      )}
      {value.dofMode === "subject" && !value.dofTargetLocked && (
        <p role="status">Pick a subject to isolate.</p>
      )}
      {number("Focus distance (blocks)", "dofDistance", 0.001, 100000, 0.1)}
      {number("Foreground blur", "dofNearStrength", 0, 5, 0.1)}
      {number("Background blur", "dofFarStrength", 0, 5, 0.1)}
      {value.dofMode !== "lens" && (
        <>
          {number("Sharp radius (blocks)", "dofFocusRange", 0, 100000, 1)}
          {number("Focus transition (blocks)", "dofTransition", 0.1, 100000, 1)}
        </>
      )}
      {value.dofMode === "lens" && (
        <>
          {number("Aperture (f/)", "dofAperture", 0.7, 64, 0.1)}
          {number(
            "Orthographic focal length (mm)",
            "dofFocalLength",
            8,
            300,
            1,
          )}
          <p className="inspector-intro">
            Lower f-number: more blur. Focal length follows field of view.
          </p>
          {number(
            "Scene scale (blocks per meter)",
            "dofWorldScale",
            0.01,
            1000,
            1,
          )}
          <p className="inspector-intro">
            1: Minecraft scale. Higher: miniature effect.
          </p>
        </>
      )}
      {number("Blur limit (px at 1080p)", "dofMaxBlur", 1, 80, 1)}
    </fieldset>
  );
}

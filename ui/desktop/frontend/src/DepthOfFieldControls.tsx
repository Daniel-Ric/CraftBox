import type { RenderSettings } from "./render-settings";

export default function DepthOfFieldControls({
  value,
  onChange,
  disabled,
  picking,
  onPick,
}: {
  value: RenderSettings;
  onChange: (value: RenderSettings) => void;
  disabled: boolean;
  picking: boolean;
  onPick: () => void;
}) {
  const number = (
    label: string,
    key:
      | "dofDistance"
      | "dofAperture"
      | "dofFocalLength"
      | "dofWorldScale"
      | "dofMaxBlur",
    min: number,
    max: number,
    step: number,
  ) => (
    <label className="render-control">
      {label}
      <input
        aria-label={label}
        type="number"
        min={min}
        max={max}
        step={step}
        value={Number(value[key].toFixed(4))}
        onChange={(e) => {
          if (e.target.value && Number.isFinite(+e.target.value))
            onChange({
              ...value,
              [key]: Math.max(min, Math.min(max, +e.target.value)),
            });
        }}
      />
    </label>
  );
  return (
    <fieldset disabled={disabled} data-inspector="detail">
      <legend>Depth of field</legend>
      <label className="render-check">
        <input
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
        {picking ? "Cancel focus picking" : "Click to set focus"}
      </button>
      {picking && (
        <p role="status">Click the subject you want in focus.</p>
      )}
      {number("Focus distance (blocks)", "dofDistance", 0.001, 100000, 0.1)}
      {number("Aperture (f/)", "dofAperture", 0.7, 64, 0.1)}
      {number("Orthographic focal length (mm)", "dofFocalLength", 8, 300, 1)}
      <p className="inspector-intro">
        Lower f-numbers create more blur. For perspective cameras, focal length follows the field of view.
      </p>
      {number(
        "Scene scale (blocks per meter)",
        "dofWorldScale",
        0.01,
        1000,
        1,
      )}
      <p className="inspector-intro">
        1 matches Minecraft scale. Higher values create a miniature effect with a shallower depth of field.
      </p>
      {number("Maximum blur radius (pixels)", "dofMaxBlur", 1, 40, 1)}
    </fieldset>
  );
}

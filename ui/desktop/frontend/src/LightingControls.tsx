import StudioInput from "@/StudioInput";
import type { RenderSettings } from "./render-settings";
import type { LightingSettings } from "./lighting-settings";
export default function LightingControls({
  value,
  onChange,
  disabled,
}: {
  value: RenderSettings;
  onChange: (value: RenderSettings) => void;
  disabled: boolean;
}) {
  const set = <K extends keyof LightingSettings>(
    key: K,
    v: LightingSettings[K],
  ) => onChange({ ...value, [key]: v });
  const slider = (
    label: string,
    key: keyof LightingSettings,
    min: number,
    max: number,
    step: number,
  ) => (
    <label className="render-control">
      <span>
        {label}
        <b>{Number(value[key]).toFixed(step < 1 ? 2 : 0)}</b>
      </span>
      <StudioInput
        aria-label={label}
        type="range"
        min={min}
        max={max}
        step={step}
        value={Number(value[key])}
        onChange={(e) => set(key, +e.target.value as never)}
      />
    </label>
  );
  return (
    <>
      <fieldset data-inspector="light" disabled={disabled}>
        <legend>Emissive blocks</legend>
        <label className="render-check">
          <StudioInput
            type="checkbox"
            checked={value.blockLights}
            onChange={(e) => set("blockLights", e.target.checked)}
          />
          Minecraft block lighting
        </label>
        <label className="render-check">
          <StudioInput
            type="checkbox"
            checked={value.blockDaylightAdaptation}
            onChange={(e) => set("blockDaylightAdaptation", e.target.checked)}
          />
          Adapt to daylight
        </label>
        <p className="inspector-intro">
          Sunlight dims block glow. Disable for constant glow.
        </p>
        <p className="inspector-intro">
          Texture-colored lights with distance falloff.
        </p>
        {slider("Block light intensity", "blockLightPower", 0, 8, 0.1)}
        {slider("Block light range", "blockLightRange", 0.25, 4, 0.05)}
        {slider("Surface emission", "emissionPower", 0, 8, 0.1)}
        <label className="render-control">
          Lights per view
          <select
            aria-label="Block light budget"
            value={value.lightBudget}
            onChange={(e) => set("lightBudget", +e.target.value)}
          >
            {[4, 8, 12, 16, 24].map((n) => (
              <option key={n}>{n}</option>
            ))}
          </select>
        </label>
        <label className="render-control">
          Shadow-casting block lights
          <select
            aria-label="Block light shadows"
            value={value.blockLightShadows}
            onChange={(e) => set("blockLightShadows", +e.target.value)}
          >
            {[0, 1, 2].map((n) => (
              <option key={n}>{n}</option>
            ))}
          </select>
        </label>
        {slider("Local light in fog", "volumeLightPower", 0, 4, 0.05)}
        <p className="inspector-intro">
          Nearby lights first. More lights cost performance.
        </p>
      </fieldset>
      <fieldset data-inspector="light" disabled={disabled}>
        <legend>Cinematic light · Spot</legend>
        <label className="render-check">
          <StudioInput
            type="checkbox"
            checked={value.studioSpot}
            onChange={(e) => set("studioSpot", e.target.checked)}
          />
          Enable cinematic light
        </label>
        <label className="render-color">
          Cinematic light color
          <StudioInput
            aria-label="Cinematic light color"
            type="color"
            value={value.spotColor}
            onChange={(e) => set("spotColor", e.target.value)}
          />
        </label>
        {slider("Cinematic light intensity", "spotPower", 0, 10000, 10)}
        {slider("Beam angle", "spotAngle", 5, 80, 1)}
        {slider("Beam edge", "spotSoftness", 0, 1, 0.05)}
        {(["Position", "Look-at target"] as const).map((label, i) => (
          <div className="render-control" key={label}>
            <span>{label} · relative to world size</span>
            <div className="render-light-vector">
              {(["X", "Y", "Z"] as const).map((axis) => {
                const key =
                  `spot${i ? "Target" : ""}${axis}` as keyof LightingSettings;
                return (
                  <label key={axis}>
                    {axis}
                    <StudioInput
                      aria-label={`Cinematic light ${label} ${axis}`}
                      type="number"
                      min={-5}
                      max={5}
                      increment={0.05}
                      value={Number(value[key])}
                      onChange={(e) => {
                        if (e.target.value !== "")
                          set(
                            key,
                            Math.max(-5, Math.min(5, +e.target.value)) as never,
                          );
                      }}
                    />
                  </label>
                );
              })}
            </div>
          </div>
        ))}
        <button
          className="inspector-next"
          onClick={() =>
            onChange({
              ...value,
              studioSpot: true,
              atmosphere: {
                ...value.atmosphere,
                fog: true,
                fogDensity: 0.2,
                fogHeight: 0,
                fogReference: "center",
                fogFalloff: 120,
              },
            })
          }
        >
          Fog for visible light beams
        </button>
      </fieldset>
    </>
  );
}

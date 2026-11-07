import StudioInput from "@/StudioInput";
import type { RenderSettings } from "./render-settings";
import { applyTimeOfDay, formatDaytime } from "./render-daytime";
export default function DaytimeControl({
  value,
  onChange,
  disabled,
}: {
  value: RenderSettings;
  onChange: (s: RenderSettings) => void;
  disabled: boolean;
}) {
  return (
    <fieldset
      data-inspector="light"
      disabled={disabled}
      className="daytime-controls"
    >
      <legend>Time of day</legend>
      <label className="render-control">
        <span>
          Local solar time{" "}
          <b>{value.timeLinked ? "Linked" : "Manual lighting"}</b>
        </span>
        <StudioInput
          aria-label="Time of day"
          type="time"
          required
          step={60}
          value={formatDaytime(value.timeOfDay)}
          onChange={(e) => {
            if (e.target.value) {
              const [h, m] = e.target.value.split(":").map(Number);
              onChange(applyTimeOfDay(value, h + m / 60));
            }
          }}
        />
      </label>
      <StudioInput
        aria-label="Time-of-day progression"
        type="range"
        min={0}
        max={23.75}
        step={0.25}
        value={value.timeOfDay}
        onChange={(e) => onChange(applyTimeOfDay(value, +e.target.value))}
      />
      <div className="daytime-presets">
        {[
          [6.5, "Morning"],
          [12, "Noon"],
          [17.25, "Evening"],
          [22, "Night"],
        ].map(([hour, label]) => (
          <button
            key={label}
            onClick={() => onChange(applyTimeOfDay(value, +hour))}
          >
            {label}
          </button>
        ))}
      </div>
      <p className="inspector-intro">Time-of-day lighting.</p>
    </fieldset>
  );
}

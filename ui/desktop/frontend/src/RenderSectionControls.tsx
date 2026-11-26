import type { RenderSection } from "./render-section";
import StudioInput from "./StudioInput";

export default function RenderSectionControls({
  value,
  onChange,
  disabled,
  worldBounds,
}: {
  value: RenderSection;
  onChange: (value: RenderSection) => void;
  disabled: boolean;
  worldBounds: () => Pick<RenderSection, "min" | "max"> | undefined;
}) {
  const axis = ["X", "Y", "Z"] as const;
  const coordinate = (edge: "min" | "max", index: number) => (
    <StudioInput
      aria-label={
        edge === "min" ? "Minimum block coordinate" : "Maximum block coordinate"
      }
      type="number"
      increment={1}
      min={edge === "min" ? -30000000 : value.min[index] + 1}
      max={edge === "max" ? 30000000 : value.max[index] - 1}
      value={value[edge][index]}
      onChange={(event) => {
        const next = Number(event.target.value);
        if (
          !event.target.value ||
          !Number.isInteger(next) ||
          Math.abs(next) > 30000000
        )
          return;
        if (
          edge === "min" ? next >= value.max[index] : next <= value.min[index]
        )
          return;
        const coordinates = [...value[edge]] as RenderSection["min"];
        coordinates[index] = next;
        onChange({ ...value, [edge]: coordinates });
      }}
    />
  );
  return (
    <fieldset data-inspector="detail" disabled={disabled}>
      <legend>Section view</legend>
      <label className="render-control" data-render-search="section">
        Section mode
        <select
          value={value.mode}
          onChange={(event) => {
            const mode = event.target.value as RenderSection["mode"];
            const bounds =
              value.mode === "off" &&
              mode !== "off" &&
              value.position === undefined
                ? worldBounds()
                : undefined;
            onChange({ ...value, ...bounds, mode });
          }}
        >
          <option value="off">Off</option>
          <option value="slice">Cutting plane</option>
          <option value="box">Section box</option>
        </select>
      </label>
      {value.mode === "slice" && (
        <>
          <label className="render-control">
            Cutting axis
            <select
              value={value.axis}
              onChange={(event) =>
                onChange({
                  ...value,
                  axis: Number(event.target.value) as RenderSection["axis"],
                })
              }
            >
              {axis.map((name, index) => (
                <option key={name} value={index}>
                  {name}
                </option>
              ))}
            </select>
          </label>
          <label className="render-control">
            <span>Cut position</span>
            <StudioInput
              type="number"
              increment={1}
              min={-30000000}
              max={30000000}
              value={value.position ?? value.max[value.axis]}
              onChange={(event) => {
                const position = Number(event.target.value);
                if (
                  event.target.value &&
                  Number.isInteger(position) &&
                  Math.abs(position) <= 30000000
                )
                  onChange({ ...value, position });
              }}
            />
          </label>
          <label className="render-check">
            <StudioInput
              type="checkbox"
              checked={value.flip}
              onChange={(event) =>
                onChange({ ...value, flip: event.target.checked })
              }
            />{" "}
            Reverse cut
          </label>
        </>
      )}
      {value.mode === "box" &&
        axis.map((name, index) => (
          <div key={name} className="render-control">
            <span>{name}</span>
            <div className="render-light-vector">
              {coordinate("min", index)}
              {coordinate("max", index)}
            </div>
          </div>
        ))}
      {value.mode === "box" && (
        <button
          type="button"
          className="inspector-next"
          onClick={() => {
            const bounds = worldBounds();
            if (bounds) onChange({ ...value, ...bounds });
          }}
        >
          Fit section to scene
        </button>
      )}
    </fieldset>
  );
}

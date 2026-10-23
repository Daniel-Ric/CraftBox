import {
  createStudioLight,
  type StudioLight,
  type LightVector,
} from "./studio-lights";

export type LightEditorState = {
  selected?: string;
  placing?: boolean;
  target?: boolean;
  onSelect: (id: string | undefined) => void;
  onPosition: (id: string, position: LightVector, target: boolean) => void;
};

export default function StudioLightControls({
  lights,
  selected,
  onSelect,
  onChange,
  onPlace,
  disabled,
  placing,
  target,
  onTarget,
}: {
  lights: StudioLight[];
  selected?: string;
  onSelect: (id: string | undefined) => void;
  onChange: (lights: StudioLight[]) => void;
  onPlace: (id: string) => void;
  disabled: boolean;
  placing?: boolean;
  target?: boolean;
  onTarget?: (target: boolean) => void;
}) {
  const active = lights.find((l) => l.id === selected);
  const edit = (patch: Partial<StudioLight>) =>
    onChange(lights.map((l) => (l.id === selected ? { ...l, ...patch } : l)));
  const add = (type: StudioLight["type"]) => {
    const light = createStudioLight(type);
    onChange([...lights, light]);
    onSelect(light.id);
  };
  const numeric = (
    label: string,
    key: "power" | "range" | "angle" | "penumbra" | "width" | "height",
    min: number,
    max: number,
    step: number,
  ) =>
    active && (
      <label className="render-control">
        {label}
        <input
          aria-label={label}
          type="number"
          min={min}
          max={max}
          step={step}
          value={active[key]}
          onChange={(e) => {
            if (e.target.value && Number.isFinite(+e.target.value))
              edit({ [key]: Math.max(min, Math.min(max, +e.target.value)) });
          }}
        />
      </label>
    );
  return (
    <fieldset
      disabled={disabled}
      data-inspector="light"
      className="studio-light-controls"
    >
      <legend>Scene lights</legend>
      <p className="inspector-intro">
        Place lights freely. Positions and sizes are relative to the world radius.
      </p>
      <div className="studio-light-actions">
        {(["point", "spot", "area"] as const).map((type) => (
          <button
            type="button"
            key={type}
            disabled={lights.length >= 16}
            onClick={() => add(type)}
          >
            Add{" "}
            {
              { point: "point light", spot: "spot light", area: "area light" }[
                type
              ]
            }
          </button>
        ))}
      </div>
      <div
        className="studio-light-list"
        role="group"
        aria-label="Scene lights"
      >
        {!lights.length && (
          <p className="inspector-intro">No custom lights yet.</p>
        )}
        {lights.map((light) => (
          <div key={light.id} className="studio-light-row">
            <input
              type="checkbox"
              aria-label={`${light.name} aktiv`}
              checked={light.enabled}
              onChange={(e) =>
                onChange(
                  lights.map((l) =>
                    l.id === light.id ? { ...l, enabled: e.target.checked } : l,
                  ),
                )
              }
            />
            <button
              type="button"
              aria-pressed={selected === light.id}
              onClick={() =>
                onSelect(selected === light.id ? undefined : light.id)
              }
            >
              {light.name}
            </button>
          </div>
        ))}
      </div>
      {active && (
        <>
          <label className="render-control">
            Name
            <input
              aria-label="Light name"
              maxLength={100}
              value={active.name}
              onChange={(e) => {
                if (e.target.value.trim()) edit({ name: e.target.value });
              }}
            />
          </label>
          <label className="render-color">
            Color
            <input
              aria-label="Scene light color"
              type="color"
              value={active.color}
              onChange={(e) => edit({ color: e.target.value })}
            />
          </label>
          {numeric("Light output (lumens)", "power", 0, 1000000, 100)}
          {active.type !== "area" &&
            numeric("Range (0 = unlimited)", "range", 0, 100, 0.1)}
          {active.type === "spot" && (
            <>
              {numeric("Spot half-angle", "angle", 1, 89, 1)}
              {numeric("Soft cone edge", "penumbra", 0, 1, 0.05)}
            </>
          )}
          {active.type === "area" && (
            <>
              {numeric("Area light width", "width", 0.001, 100, 0.05)}
              {numeric("Area light height", "height", 0.001, 100, 0.05)}
              <p className="inspector-intro">
                Area lights illuminate one side. Real-time area lights do not currently cast shadows.
              </p>
            </>
          )}
          {active.type !== "area" && (
            <label className="render-check">
              <input
                type="checkbox"
                aria-label="Scene light shadows"
                checked={active.shadows}
                disabled={
                  !active.shadows && lights.filter((l) => l.shadows).length >= 4
                }
                onChange={(e) => edit({ shadows: e.target.checked })}
              />
              Cast shadows
            </label>
          )}
          {(["position", "target"] as const)
            .filter((key) => key === "position" || active.type !== "point")
            .map((key) => (
              <div className="render-control" key={key}>
                <span>{key === "position" ? "Position" : "Look-at target"}</span>
                <div className="render-light-vector">
                  {(["X", "Y", "Z"] as const).map((axis, i) => (
                    <label key={axis}>
                      {axis}
                      <input
                        type="number"
                        aria-label={`Light ${key === "position" ? "position" : "look-at target"} ${axis}`}
                        min={-100}
                        max={100}
                        step={0.05}
                        value={Number(active[key][i].toFixed(4))}
                        onChange={(e) => {
                          if (
                            !e.target.value ||
                            !Number.isFinite(+e.target.value)
                          )
                            return;
                          const v: LightVector = [...active[key]];
                          v[i] = Math.max(-100, Math.min(100, +e.target.value));
                          edit({ [key]: v });
                        }}
                      />
                    </label>
                  ))}
                </div>
              </div>
            ))}
          {active.type !== "point" && (
            <label className="render-check">
              <input
                type="checkbox"
                checked={target ?? false}
                onChange={(e) => onTarget?.(e.target.checked)}
              />
              Edit look-at target in view
            </label>
          )}
          <button
            type="button"
            className="inspector-next"
            aria-pressed={placing ?? false}
            onClick={() => onPlace(active.id)}
          >
            {placing ? "Cancel placement" : "Place in view"}
          </button>
          {placing && (
            <p role="status">
              Click a surface. Dragging still moves the camera.
            </p>
          )}
          <div className="studio-light-actions">
            <button
              type="button"
              disabled={lights.length >= 16}
              onClick={() => {
                const copy = {
                  ...active,
                  id: crypto.randomUUID(),
                  name: (active.name + " Copy").slice(0, 100),
                  position: [...active.position] as LightVector,
                  target: [...active.target] as LightVector,
                  shadows: false,
                };
                onChange([...lights, copy]);
                onSelect(copy.id);
              }}
            >
              Duplicate light
            </button>
            <button
              type="button"
              onClick={() => {
                onChange(lights.filter((l) => l.id !== active.id));
                onSelect(undefined);
              }}
            >
              Delete light
            </button>
          </div>
        </>
      )}
    </fieldset>
  );
}

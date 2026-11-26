import StudioInput from "@/StudioInput";
import { useEffect, useRef, useState } from "react";
import { formatLocale, text, useLocale } from "./i18n";
import { notifyFailure } from "./StudioFlyouts";
import {
  createStudioLight,
  keyStudioLight,
  sampleStudioLights,
  receiverBlocks,
  validLightReceivers,
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
  playhead,
  onSeek,
  editTime,
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
  playhead?: number;
  onSeek?: (time: number) => void;
  editTime?: boolean;
}) {
  useLocale();
  const source = lights.find((l) => l.id === selected);
  const active = source && sampleStudioLights([source], playhead)[0];
  const currentKey = active?.keys?.find((key) => key.time === playhead);
  const [receiver, setReceiver] = useState("");
  const activeSelection = useRef(selected);
  const projectionInput = useRef<HTMLInputElement>(null);
  activeSelection.current = selected;
  const latestLights = useRef(lights);
  latestLights.current = lights;
  useEffect(() => { setReceiver(active?.receiver ?? ""); }, [active?.id, active?.receiver]);
  const edit = (patch: Partial<StudioLight>) => onChange(latestLights.current.map((light) => {
    if (light.id !== selected) return light;
    if (playhead !== undefined && light.keys?.length &&
      (["position", "target", "color", "power", "angle", "penumbra"] as const).some((key) => patch[key] !== undefined))
      return { ...light, keys: keyStudioLight({ ...sampleStudioLights([light], playhead)[0], ...patch }, playhead).keys };
    return { ...light, ...patch };
  }));
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
        <StudioInput
          aria-label={label}
          type="number"
          min={min}
          max={max}
          increment={step}
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
        Position and size in world-radius units.
      </p>
      <div className="studio-light-actions">
        {(["point", "spot", "area"] as const).map((type) => (
          <button
            type="button"
            key={type}
            data-render-search={type === "spot" ? "add-spot" : undefined}
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
      <div className="studio-light-list" role="group" aria-label="Scene lights">
        {!lights.length && (
          <p className="inspector-intro">No custom lights yet.</p>
        )}
        {lights.map((light) => (
          <div key={light.id} className="studio-light-row">
            <StudioInput
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
            <StudioInput
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
            <StudioInput
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
              <div className="render-control" data-render-search="gobo">
                <span>Spotlight projection</span>
                <div className="studio-light-actions">
                  <button type="button" onClick={() => projectionInput.current?.click()}>Load projection PNG</button>
                  {active.gobo && <button type="button" onClick={() => edit({ gobo: undefined })}>Remove projection</button>}
                </div>
                <input ref={projectionInput} type="file" accept="image/png" hidden onChange={async (event) => {
                  const file = event.currentTarget.files?.[0];
                  event.currentTarget.value = "";
                  if (!file) return;
                  try {
                    if (file.type !== "image/png" || file.size > 1300000)
                      throw new Error(text("Choose a PNG smaller than 1.3 MB."));
                    const image = await createImageBitmap(file);
                    try {
                      if (Math.max(image.width, image.height) > 2048)
                        throw new Error(text("Spotlight masks can be up to 2048 pixels wide or tall."));
                      const canvas = document.createElement("canvas");
                      canvas.width = image.width;
                      canvas.height = image.height;
                      const context = canvas.getContext("2d")!;
                      context.fillStyle = "black";
                      context.fillRect(0, 0, canvas.width, canvas.height);
                      context.drawImage(image, 0, 0);
                      const gobo = canvas.toDataURL("image/png");
                      if (gobo.length > 1800000) throw new Error(text("Choose a PNG smaller than 1.3 MB."));
                      if (activeSelection.current === selected) edit({ gobo });
                    } finally { image.close(); }
                  } catch (error) { notifyFailure(error, "Could not load spotlight projection"); }
                }} />
              </div>
            </>
          )}
          {active.type === "area" && (
            <>
              {numeric("Area light width", "width", 0.001, 100, 0.05)}
              {numeric("Area light height", "height", 0.001, 100, 0.05)}
              <p className="inspector-intro">
                One-sided lighting. No real-time shadows.
              </p>
            </>
          )}
          <div className="render-control" data-render-search="light-link">
            <label className="render-control">Light receiver blocks
              <StudioInput value={receiver} maxLength={1000} placeholder="minecraft:stone, minecraft:glass"
                aria-invalid={!!receiver.trim() && !validLightReceivers(receiver.toLowerCase())}
                onChange={(event) => setReceiver(event.target.value)} onBlur={() => {
                  const value = receiver.trim().toLowerCase();
                  if (!value || validLightReceivers(value)) {
                    const names = receiverBlocks(value).join(", ");
                    setReceiver(names);
                    edit({ receiver: names || undefined });
                  }
                }} onKeyDown={(event) => { if (event.key === "Enter") event.currentTarget.blur(); }} />
            </label>
            <p className="inspector-intro">Enter up to eight block IDs, separated by commas.</p>
            {!!active.receiver && <label className="render-control">Light linking mode
              <select value={active.receiverMode ?? "include"} onChange={(event) => edit({ receiverMode: event.target.value === "exclude" ? "exclude" : "include" })}>
                <option value="include">Illuminate listed blocks</option>
                <option value="exclude">Exclude listed blocks</option>
              </select>
            </label>}
          </div>
          {active.type !== "area" && (
            <label className="render-check">
              <StudioInput
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
                <span>
                  {key === "position" ? "Position" : "Look-at target"}
                </span>
                <div className="render-light-vector">
                  {(["X", "Y", "Z"] as const).map((axis, i) => (
                    <label key={axis}>
                      {axis}
                      <StudioInput
                        type="number"
                        aria-label={`Light ${key === "position" ? "position" : "look-at target"} ${axis}`}
                        min={-100}
                        max={100}
                        increment={0.05}
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
              <StudioInput
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
          {placing && <p role="status">Click to place. Drag to orbit.</p>}
          {playhead !== undefined && <div className="render-control" data-render-search="light-keys">
            <span>Light keyframes</span>
            {editTime && <label className="render-control">Light preview time (seconds)
              <StudioInput type="number" min={0} max={86400} increment={0.1} value={playhead} onChange={(event) => {
                const time = Number(event.target.value);
                if (event.target.value && Number.isFinite(time)) onSeek?.(Math.round(Math.max(0, Math.min(86400, time)) * 1000) / 1000);
              }} />
            </label>}
            <div className="studio-light-actions">
              <button type="button" disabled={(active.keys?.length ?? 0) >= 2000 && !currentKey} onClick={() => edit({ keys: keyStudioLight(active, playhead).keys })}>{editTime ? "Key at preview time" : "Key at playhead"}</button>
              <button type="button" disabled={!currentKey} onClick={() => edit({ keys: active.keys?.filter((key) => key.time !== playhead) })}>Remove current key</button>
            </div>
            {currentKey && <label className="render-control">Keyframe interpolation
              <select value={currentKey.ease ?? "linear"} onChange={(event) => {
                const ease = event.target.value as typeof currentKey.ease;
                edit({ keys: active.keys?.map((key) => key === currentKey ? { ...key, ease } : key) });
              }}><option value="linear">Linear</option><option value="smooth">Smooth</option><option value="hold">Hold</option></select>
            </label>}
            {!!active.keys?.length && <select aria-label="Go to light keyframe" value="" onChange={(event) => onSeek?.(Number(event.target.value))}>
              <option value="">Go to light keyframe</option>
              {active.keys.map((key) => <option key={key.time} value={key.time}>{new Intl.NumberFormat(formatLocale(), { maximumFractionDigits: 3 }).format(key.time)} s</option>)}
            </select>}
          </div>}
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

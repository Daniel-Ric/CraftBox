import { useRef, useState } from "react";
import StudioInput from "./StudioInput";
import { notifyFailure } from "./StudioFlyouts";
import type { StudioTake } from "./studio-document";

export default function RenderTakeControls({
  takes,
  disabled,
  onApply,
  onChange,
  capture,
}: {
  takes: StudioTake[];
  disabled: boolean;
  onApply: (take: StudioTake) => void;
  onChange: (takes: StudioTake[]) => void;
  capture: (id: string, name: string) => Promise<StudioTake>;
}) {
  const [selected, setSelected] = useState("");
  const [name, setName] = useState("");
  const [capturing, setCapturing] = useState(false);
  const latest = useRef(takes);
  latest.current = takes;
  const take = takes.find((variant) => variant.id === selected);
  const save = async (existing?: StudioTake) => {
    setCapturing(true);
    try {
      const variant = await capture(
        existing?.id ?? crypto.randomUUID(),
        name.trim(),
      );
      onChange(
        existing
          ? latest.current.map((item) =>
              item.id === existing.id ? variant : item,
            )
          : [...latest.current, variant],
      );
      setSelected(variant.id);
    } catch (error) {
      notifyFailure(error, "Could not save scene take");
    } finally {
      setCapturing(false);
    }
  };
  return (
    <fieldset data-inspector="looks" disabled={disabled || capturing}>
      <legend>Scene takes</legend>
      <label className="render-control" data-render-search="takes">
        Take name
        <StudioInput
          value={name}
          maxLength={100}
          onChange={(event) => setName(event.target.value)}
        />
      </label>
      <button
        type="button"
        className="inspector-next"
        disabled={!name.trim() || takes.length >= 16}
        onClick={() => void save()}
      >
        Save current take
      </button>
      {!!takes.length && (
        <label className="render-control">
          Saved takes
          <select
            value={selected}
            onChange={(event) => {
              const next = takes.find(
                (variant) => variant.id === event.target.value,
              );
              setSelected(next?.id ?? "");
              setName(next?.name ?? "");
            }}
          >
            <option value="">Select a take</option>
            {takes.map((variant) => (
              <option key={variant.id} value={variant.id} translate="no">
                {variant.name}
              </option>
            ))}
          </select>
        </label>
      )}
      {take && (
        <div className="studio-light-actions">
          <button type="button" onClick={() => onApply(take)}>
            Apply take
          </button>
          <button
            type="button"
            disabled={!name.trim()}
            onClick={() => void save(take)}
          >
            Update take
          </button>
          <button
            type="button"
            onClick={() => {
              onChange(takes.filter((item) => item !== take));
              setSelected("");
            }}
          >
            Delete take
          </button>
        </div>
      )}
    </fieldset>
  );
}

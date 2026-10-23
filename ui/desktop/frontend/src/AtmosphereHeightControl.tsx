import { useEffect, useState } from "react";
import type { HeightReference } from "./atmosphere-settings";
export default function AtmosphereHeightControl({
  label,
  value,
  reference,
  onValue,
  onReference,
}: {
  label: string;
  value: number;
  reference: HeightReference;
  onValue: (n: number) => void;
  onReference: (r: HeightReference) => void;
}) {
  const [draft, setDraft] = useState(String(value));
  useEffect(() => setDraft(String(value)), [value]);
  const commit = () => {
    const n = Number(draft.replace(",", "."));
    if (draft.trim() && Number.isFinite(n)) {
      const v = Math.max(-2048, Math.min(4096, n));
      onValue(v);
      setDraft(String(v));
    } else setDraft(String(value));
  };
  const prefix = label === "Cloud height" ? "Cloud" : "Fog";
  return (
    <div className="atmosphere-height">
      <label className="render-control">
        Height reference
        <select
          aria-label={`${prefix} height reference`}
          value={reference}
          onChange={(e) => onReference(e.target.value as HeightReference)}
        >
          <option value="world">Absolute world height (Y)</option>
          <option value="bottom">From model base</option>
          <option value="center">From model center</option>
          <option value="top">From model top</option>
        </select>
      </label>
      <label className="render-control">
        {label}
        <small>blocks · negative values allowed</small>
        <div className="atmosphere-height-number">
          <button
            aria-label={`${prefix} 10 blocks lower`}
            onClick={() => onValue(Math.max(-2048, value - 10))}
          >
            −10
          </button>
          <input
            aria-label={label}
            inputMode="decimal"
            type="text"
            value={draft}
            onChange={(e) => {
              const text = e.target.value;
              if (/^[+-]?\d*([.,]\d*)?$/.test(text)) {
                setDraft(text);
                if (
                  text !== "" &&
                  text !== "-" &&
                  text !== "+" &&
                  Number.isFinite(Number(text.replace(",", ".")))
                )
                  onValue(
                    Math.max(
                      -2048,
                      Math.min(4096, Number(text.replace(",", "."))),
                    ),
                  );
              }
            }}
            onBlur={commit}
            onKeyDown={(e) => {
              if (e.key === "Enter") commit();
              if (e.key === "Escape") setDraft(String(value));
            }}
          />
          <button
            aria-label={`${prefix} 10 blocks higher`}
            onClick={() => onValue(Math.min(4096, value + 10))}
          >
            +10
          </button>
        </div>
      </label>
      <input
        aria-label={`${prefix} height slider`}
        type="range"
        min={Math.min(-256, value)}
        max={Math.max(512, value)}
        step={1}
        value={value}
        onChange={(e) => onValue(+e.target.value)}
      />
      <div className="atmosphere-height-quick">
        {[-64, -16, 0, 16, 64].map((n) => (
          <button key={n} onClick={() => onValue(n)}>
            {n > 0 ? "+" : ""}
            {n}
          </button>
        ))}
      </div>
    </div>
  );
}

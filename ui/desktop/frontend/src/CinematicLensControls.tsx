import StudioInput from "@/StudioInput";
import type { Sequence, CameraPose } from "./cinematic";
export default function CinematicLensControls(p: {
  lens: NonNullable<Sequence["lens"]>;
  pose: CameraPose;
  scale: number;
  picking: boolean;
  disabled: boolean;
  onChange: (lens: NonNullable<Sequence["lens"]>) => void;
  onKey: (field: "focusDistance" | "aperture", value: number) => void;
  onPick: () => void;
  onScale?: (scale: number) => void;
}) {
  return (
    <section data-camera-panel="camera" className="cinematic-lens-controls">
      <h3>Focus &amp; aperture</h3>
      <label>
        <StudioInput
          type="checkbox"
          aria-label="Camera depth of field"
          checked={p.lens.enabled}
          disabled={p.disabled}
          onChange={(e) => p.onChange({ ...p.lens, enabled: e.target.checked })}
        />
        Depth of field for this camera sequence
      </label>
      <button disabled={p.disabled} aria-pressed={p.picking} onClick={p.onPick}>
        {p.picking ? "Cancel focus picking" : "Pick camera focus"}
      </button>
      {p.picking && <p role="status">Click a subject to keyframe focus.</p>}
      {(
        [
          {
            field: "focusDistance",
            label: "Camera focus distance (blocks)",
            fallback: p.lens.distance,
            min: 0.001,
            max: 100000,
          },
          {
            field: "aperture",
            label: "Camera aperture (f/)",
            fallback: p.lens.aperture,
            min: 0.7,
            max: 64,
          },
        ] as const
      ).map((c) => (
        <label key={c.field}>
          {c.label}
          <StudioInput
            type="number"
            aria-label={c.label}
            disabled={p.disabled}
            min={c.min}
            max={c.max}
            step="any"
            increment={0.1}
            value={Number((p.pose[c.field] ?? c.fallback).toFixed(4))}
            onChange={(e) => {
              if (e.target.value && Number.isFinite(+e.target.value))
                p.onKey(
                  c.field,
                  Math.max(c.min, Math.min(c.max, +e.target.value)),
                );
            }}
          />
        </label>
      ))}
      <p>Changes add keyframes. Edit focus and aperture in Curves.</p>
      {p.onScale && (
        <label>
          Miniature scale (blocks per meter)
          <StudioInput
            type="number"
            aria-label="Camera scene scale"
            min={0.01}
            max={1000}
            step="any"
            increment={1}
            value={p.scale}
            disabled={p.disabled}
            onChange={(e) => {
              if (e.target.value && Number.isFinite(+e.target.value))
                p.onScale?.(Math.max(0.01, Math.min(1000, +e.target.value)));
            }}
          />
        </label>
      )}
    </section>
  );
}

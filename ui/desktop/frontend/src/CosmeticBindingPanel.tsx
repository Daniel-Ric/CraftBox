import type { CosmeticBinding } from "./asset-geometry";
import type { RigPart } from "./human-skeleton";
import { Button } from "@/components/ui/button";

export default function CosmeticBindingPanel(p: {
  name: string;
  layers: number;
  settings: CosmeticBinding;
  parts: RigPart[];
  onChange: (patch: CosmeticBinding) => void;
  onReset: () => void;
  onSelect?: () => void;
  onBegin: () => void;
  onEnd: () => void;
}) {
  const mode = p.settings.binding ?? "auto",
    following = mode !== "none";
  return (
    <div className="cosmetic-binding-panel">
      <div className="cosmetic-binding-heading">
        <strong>{p.name}</strong>
        <span>
          {p.layers} {p.layers === 1 ? "layer" : "layers"}
        </span>
      </div>
      <label>
        <input
          type="checkbox"
          checked={p.settings.visible !== false}
          onChange={(e) => p.onChange({ visible: e.target.checked })}
        />
        Show cosmetic
      </label>
      <label className="cosmetic-binding-target">
        Follow target
        <select
          aria-label={`Follow target for ${p.name}`}
          value={mode}
          onChange={(e) => p.onChange({ binding: e.target.value })}
        >
          <option value="auto">Automatic (pack rig)</option>
          <option value="none">Character only</option>
          {p.parts
            .filter(
              (part) =>
                (part.kind === "bone" || part.kind === "joint") &&
                part.id !== "character",
            )
            .map((part) => (
              <option key={part.id} value={part.id}>
                {part.label}
              </option>
            ))}
        </select>
      </label>
      <p>
        {mode === "auto"
          ? "Each part follows its matching pack bone. Position and rotation below apply to every layer."
          : mode === "none"
            ? "Moves with the whole character, without following individual joints."
            : "All layers follow this joint together. The neutral placement is preserved; use Move and Rotate to adjust it."}
      </p>
      <div className="cosmetic-binding-flags">
        {[
          ["followPosition", "Follow position"],
          ["followRotation", "Follow rotation"],
          ["followScale", "Follow scale"],
        ].map(([key, label]) => (
          <label key={key}>
            <input
              type="checkbox"
              disabled={!following}
              checked={p.settings[key as keyof CosmeticBinding] !== false}
              onChange={(e) => p.onChange({ [key]: e.target.checked })}
            />
            {label}
          </label>
        ))}
      </div>
      <label className="cosmetic-binding-strength">
        Influence
        <input
          aria-label={`Influence for ${p.name}`}
          disabled={!following}
          type="range"
          min={0}
          max={100}
          value={Math.round((p.settings.influence ?? 1) * 100)}
          onPointerDown={p.onBegin}
          onPointerUp={p.onEnd}
          onPointerCancel={p.onEnd}
          onBlur={p.onEnd}
          onKeyDown={(e) => {
            if (e.key.startsWith("Arrow")) p.onBegin();
          }}
          onKeyUp={p.onEnd}
          onChange={(e) => p.onChange({ influence: +e.target.value / 100 })}
        />
        <span>{Math.round((p.settings.influence ?? 1) * 100)}%</span>
      </label>
      <div className="cosmetic-binding-actions">
        {p.onSelect && (
          <Button variant="outline" onClick={p.onSelect}>
            Adjust placement
          </Button>
        )}
        <Button variant="ghost" onClick={p.onReset}>
          Reset binding
        </Button>
      </div>
    </div>
  );
}

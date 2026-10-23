import type { CameraPose, Sequence, Vec3 } from "./cinematic";
import {
  addTrackingTarget,
  editTargetPosition,
  removeTrackingTarget,
  sampleTarget,
  type CameraTracking,
} from "./cinematic-tracking";
import { addChannelKey } from "./cinematic-tracks";
import { sampleTrack } from "./animation-tracks";

export default function CinematicTracking(p: {
  project: Sequence;
  source: Sequence;
  pose: CameraPose;
  time: number;
  selected?: string;
  editing: boolean;
  picking: boolean;
  disabled: boolean;
  onProject: (project: Sequence) => void;
  onSource: (source: Sequence) => void;
  onSelect: (id: string) => void;
  onEdit: (edit: boolean) => void;
  onPick: () => void;
}) {
  const targets = p.project.targets ?? [],
    target = targets.find((t) => t.id === p.selected) ?? targets[0];
  const constraint = p.source.tracking;
  const weightTrack = p.source.tracks?.find(
    (t) => t.channel === "trackingWeight",
  );
  const weight = weightTrack
    ? sampleTrack(
        weightTrack,
        p.time,
        weightTrack.defaultValue ?? constraint?.weight ?? 1,
      )
    : (constraint?.weight ?? 1);
  const position = target ? sampleTarget(target, p.time) : [0, 0, 0];
  function tracking(change: Partial<CameraTracking> = {}): CameraTracking {
    return {
      enabled: true,
      targetId: target!.id,
      offset: [0, 0, 0],
      weight: 1,
      focus: false,
      ...constraint,
      ...change,
    };
  }
  function constrain(change: Partial<CameraTracking>) {
    if (target) p.onSource({ ...p.source, tracking: tracking(change) });
  }
  return (
    <section data-camera-panel="tracking">
      <h3>Tracking targets</h3>
      <p>
        Shared targets move in sequence time. Cameras follow them with independent offsets and animatable influence.
      </p>
      <button
        disabled={p.disabled || targets.length >= 64}
        onClick={() => {
          const next = addTrackingTarget(p.project, [...p.pose.target]);
          p.onProject(next);
          p.onSelect(next.targets!.at(-1)!.id);
        }}
      >
        Add tracking target
      </button>
      {target && (
        <>
          <label>
            Tracking target
            <select
              aria-label="Edit tracking target"
              disabled={p.disabled}
              value={target.id}
              onChange={(e) => p.onSelect(e.target.value)}
            >
              {targets.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Name
            <input
              aria-label="Target name"
              disabled={p.disabled}
              value={target.name}
              maxLength={200}
              onChange={(e) =>
                p.onProject({
                  ...p.project,
                  targets: targets.map((t) =>
                    t.id === target.id ? { ...t, name: e.target.value } : t,
                  ),
                })
              }
            />
          </label>
          <div className="cinema-pair">
            <button
              disabled={p.disabled}
              aria-pressed={p.picking}
              onClick={p.onPick}
            >
              {p.picking ? "Cancel target picking" : "Click to place target"}
            </button>
            <button
              disabled={p.disabled}
              onClick={() =>
                p.onProject(removeTrackingTarget(p.project, target.id))
              }
            >
              Delete tracking target
            </button>
          </div>
          {p.picking && (
            <p role="status">
              Click a surface. Its position is saved as a target keyframe at the playhead.
            </p>
          )}
          <div className="cinema-vector">
            {["X", "Y", "Z"].map((axis, i) => (
              <label key={axis}>
                {axis}
                <input
                  aria-label={`Target position ${axis}`}
                  type="number"
                  disabled={p.disabled}
                  value={Number(position[i].toFixed(5))}
                  step={0.05}
                  min={-10000}
                  max={10000}
                  onChange={(e) => {
                    if (!e.target.value || !Number.isFinite(+e.target.value))
                      return;
                    const next = [...position] as Vec3;
                    next[i] = Math.max(
                      -10000,
                      Math.min(10000, +e.target.value),
                    );
                    p.onProject(
                      editTargetPosition(p.project, target.id, p.time, next),
                    );
                  }}
                />
              </label>
            ))}
          </div>
          <button
            disabled={p.disabled}
            aria-pressed={p.editing}
            onClick={() => p.onEdit(!p.editing)}
          >
            {p.editing
              ? "Edit camera tracks"
              : "Edit target tracks in the curve editor"}
          </button>
          <h3>Tracking for this camera</h3>
          <label>
            <input
              aria-label="Enable target tracking"
              type="checkbox"
              disabled={p.disabled}
              checked={!!constraint?.enabled}
              onChange={(e) => constrain({ enabled: e.target.checked })}
            />
            Target tracking
          </label>
          <label>
            Tracked target
            <select
              aria-label="Tracked target"
              disabled={p.disabled}
              value={constraint?.targetId ?? target.id}
              onChange={(e) => constrain({ targetId: e.target.value })}
            >
              {targets.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </label>
          <div className="cinema-vector">
            {["X", "Y", "Z"].map((axis, i) => (
              <label key={axis}>
                {axis}
                <input
                  aria-label={`Target offset ${axis}`}
                  type="number"
                  disabled={p.disabled}
                  value={constraint?.offset[i] ?? 0}
                  step={0.05}
                  min={-10000}
                  max={10000}
                  onChange={(e) => {
                    if (!e.target.value || !Number.isFinite(+e.target.value))
                      return;
                    const offset = [
                      ...(constraint?.offset ?? [0, 0, 0]),
                    ] as Vec3;
                    offset[i] = Math.max(
                      -10000,
                      Math.min(10000, +e.target.value),
                    );
                    constrain({ offset });
                  }}
                />
              </label>
            ))}
          </div>
          <label>
            Influence
            <input
              aria-label="Target tracking influence"
              type="range"
              min={0}
              max={1}
              step={0.01}
              disabled={p.disabled}
              value={Math.max(0, Math.min(1, weight))}
              onChange={(e) => {
                const value = +e.target.value;
                if (weightTrack?.keys.length)
                  p.onSource(
                    addChannelKey(
                      { ...p.source, tracking: tracking() },
                      "trackingWeight",
                      p.time,
                      value,
                    ),
                  );
                else constrain({ weight: value });
              }}
            />
          </label>
          <button
            disabled={p.disabled}
            onClick={() =>
              p.onSource(
                addChannelKey(
                  { ...p.source, tracking: tracking() },
                  "trackingWeight",
                  p.time,
                  Math.max(0, Math.min(1, weight)),
                ),
              )
            }
          >
            Add influence keyframe
          </button>
          <label>
            <input
              aria-label="Focus follows target"
              type="checkbox"
              disabled={p.disabled}
              checked={constraint?.focus ?? false}
              onChange={(e) => constrain({ focus: e.target.checked })}
            />
            Focus follows target
          </label>
          <p>
            Linked focus uses the subject's depth along the camera direction. Depth of field and aperture remain adjustable in the camera settings.
          </p>
        </>
      )}
    </section>
  );
}

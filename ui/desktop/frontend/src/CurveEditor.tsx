import StudioInput from "@/StudioInput";
import { useMemo, useRef, useState } from "react";
import {
  channelLabels,
  channelLimits,
  moveTrackKey,
  segmentControls,
  type AnimationChannel,
  type AnimationTrack,
  type ScalarKey,
} from "./animation-tracks";
import {
  autoCurveKey,
  graphPath,
  sampleVelocity,
  setCurveTangent,
  type TangentMode,
} from "./animation-curves";
import { sampleTrack } from "./animation-tracks";

type Props<C extends string> = {
  track: AnimationTrack<C>;
  duration: number;
  fps: number;
  time: number;
  snap: boolean;
  disabled: boolean;
  label?: string;
  limits?: [number, number];
  selected?: string;
  onSelect: (id: string) => void;
  onScrub: (time: number) => void;
  onChange: (next: AnimationTrack<C>, before?: AnimationTrack<C>) => void;
  onPreview: (track: AnimationTrack<C>) => void;
  onCancel?: () => void;
};
export default function CurveEditor<C extends string = AnimationChannel>(
  p: Props<C>,
) {
  const [mode, setMode] = useState<"value" | "speed">("value");
  const [tangentMode, setTangentMode] = useState<TangentMode>("aligned");
  const drag = useRef<{
    track: AnimationTrack<C>;
    next: AnimationTrack<C>;
    key: string;
    side?: "incoming" | "outgoing";
    domain: [number, number];
  } | null>(null);
  const samples = useMemo(
    () =>
      Array.from({ length: 241 }, (_, i) => {
        const time = (p.duration * i) / 240;
        return mode === "speed"
          ? sampleVelocity(p.track, time)
          : sampleTrack(p.track, time, p.track.defaultValue ?? 0);
      }),
    [p.track, p.duration, mode],
  );
  const extremes = [
    ...samples,
    ...(mode === "value"
      ? p.track.keys.flatMap((k) => [
          k.value,
          k.value + k.incoming[1],
          k.value + k.outgoing[1],
        ])
      : [0]),
  ];
  const low = Math.min(0, ...extremes),
    high = Math.max(1, ...extremes),
    margin = (high - low) * 0.12;
  const domain: [number, number] = drag.current?.domain ?? [
    low - margin,
    high + margin,
  ];
  const x = (time: number) => 56 + (time / p.duration) * 920;
  const y = (value: number) =>
    190 - ((value - domain[0]) / (domain[1] - domain[0])) * 164;
  const selected = p.track.keys.find((k) => k.id === p.selected);
  const replace = (key: ScalarKey) => ({
    ...p.track,
    keys: p.track.keys.map((k) => (k.id === key.id ? key : k)),
  });
  function pointerValue(event: React.PointerEvent<SVGElement>) {
    const rect = event.currentTarget.ownerSVGElement!.getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * 1000,
      py = ((event.clientY - rect.top) / rect.height) * 220;
    return {
      time: Math.max(0, Math.min(p.duration, ((px - 56) / 920) * p.duration)),
      value: domain[0] + ((190 - py) / 164) * (domain[1] - domain[0]),
    };
  }
  function begin(
    event: React.PointerEvent<SVGElement>,
    key: ScalarKey,
    side?: "incoming" | "outgoing",
  ) {
    if (p.disabled || mode !== "value") return;
    event.stopPropagation();
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    p.onSelect(key.id);
    drag.current = { track: p.track, next: p.track, key: key.id, side, domain };
  }
  function move(event: React.PointerEvent<SVGElement>) {
    const action = drag.current;
    if (!action) return;
    const value = pointerValue(event),
      original = action.track.keys.find((k) => k.id === action.key)!;
    let next = action.track;
    if (action.side) {
      const index = action.track.keys.indexOf(original),
        before = action.track.keys[index - 1],
        after = action.track.keys[index + 1];
      const limit =
        action.side === "incoming"
          ? -(original.time - (before?.time ?? original.time)) / 2
          : ((after?.time ?? original.time) - original.time) / 2;
      const dt =
        action.side === "incoming"
          ? Math.max(limit, Math.min(0, value.time - original.time))
          : Math.min(limit, Math.max(0, value.time - original.time));
      const changed = setCurveTangent(
        original,
        action.side,
        [
          dt,
          Math.max(-1000000, Math.min(1000000, value.value - original.value)),
        ],
        tangentMode,
      );
      next = {
        ...action.track,
        keys: action.track.keys.map((k) => (k.id === changed.id ? changed : k)),
      };
    } else {
      const t = p.snap ? Math.round(value.time * p.fps) / p.fps : value.time;
      if (p.snap)
        next = moveTrackKey(action.track, action.key, t, p.fps, p.duration);
      else if (
        !action.track.keys.some(
          (k) => k.id !== action.key && Math.abs(k.time - t) < 1e-8,
        )
      )
        next = {
          ...action.track,
          keys: action.track.keys
            .map((k) => (k.id === action.key ? { ...k, time: t } : k))
            .sort((a, b) => a.time - b.time),
        };
      const limits =
        p.limits ?? channelLimits[action.track.channel as AnimationChannel];
      next = {
        ...next,
        keys: next.keys.map((k) =>
          k.id === action.key
            ? {
                ...k,
                value: Math.max(limits[0], Math.min(limits[1], value.value)),
              }
            : k,
        ),
      };
    }
    action.next = next;
    p.onPreview(next);
  }
  function end(cancelled = false) {
    const action = drag.current;
    drag.current = null;
    if (!action) return;
    if (cancelled) {
      p.onPreview(action.track);
      p.onCancel?.();
    } else if (JSON.stringify(action.next) !== JSON.stringify(action.track))
      p.onChange(action.next, action.track);
  }
  const path =
    mode === "value"
      ? graphPath(p.track, x, y)
      : samples
          .map((v, i) => `${i ? "L" : "M"}${x((i * p.duration) / 240)},${y(v)}`)
          .join(" ");
  return (
    <div className="animation-curve-editor">
      <div className="animation-curve-toolbar">
        <label>
          Graph
          <select
            aria-label="Curve view"
            value={mode}
            onChange={(e) => setMode(e.target.value as typeof mode)}
          >
            <option value="value">Values</option>
            <option value="speed">Speed</option>
          </select>
        </label>
        <label>
          Tangents
          <select
            aria-label="Tangent linking"
            value={tangentMode}
            onChange={(e) => setTangentMode(e.target.value as TangentMode)}
          >
            <option value="free">Free</option>
            <option value="aligned">Aligned</option>
            <option value="mirrored">Mirrored</option>
          </select>
        </label>
        <button
          disabled={p.disabled || !selected}
          onClick={() =>
            selected && p.onChange(autoCurveKey(p.track, selected.id))
          }
        >
          Automatic tangents
        </button>
        <button
          disabled={p.disabled || !selected}
          onClick={() =>
            selected &&
            p.onChange(
              replace({
                ...selected,
                interpolation: "bezier",
                incoming: [selected.incoming[0], 0],
                outgoing: [selected.outgoing[0], 0],
              }),
            )
          }
        >
          Flatten tangents
        </button>
      </div>
      <svg
        viewBox="0 0 1000 220"
        preserveAspectRatio="none"
        className="animation-graph"
        aria-label={`${mode === "value" ? "Value curve" : "Speed curve"} ${p.label ?? channelLabels[p.track.channel as AnimationChannel] ?? p.track.channel}`}
      >
        {Array.from({ length: 5 }, (_, i) => {
          const v = domain[0] + ((domain[1] - domain[0]) * i) / 4;
          return (
            <g key={i}>
              <line
                x1={56}
                x2={976}
                y1={y(v)}
                y2={y(v)}
                className="animation-grid"
              />
              <text x={50} y={y(v) + 4} textAnchor="end">
                {Number(v.toFixed(2))}
                {mode === "speed" ? "/s" : ""}
              </text>
            </g>
          );
        })}
        {Array.from({ length: 9 }, (_, i) => (
          <g key={i}>
            <line
              x1={x((i * p.duration) / 8)}
              x2={x((i * p.duration) / 8)}
              y1={26}
              y2={190}
              className="animation-grid"
            />
            <text x={x((i * p.duration) / 8)} y={210} textAnchor="middle">
              {Number(((i * p.duration) / 8).toFixed(2))}s
            </text>
          </g>
        ))}
        <path d={path} className="animation-curve-path" />
        <line
          x1={x(p.time)}
          x2={x(p.time)}
          y1={24}
          y2={192}
          className="animation-cursor"
        />
        {mode === "value" &&
          p.track.keys.map((key, index) => (
            <g key={key.id}>
              {key.id === p.selected &&
                (["incoming", "outgoing"] as const).map((side) => {
                  const neighbor =
                    side === "incoming"
                      ? p.track.keys[index - 1]
                      : p.track.keys[index + 1];
                  if (
                    !neighbor ||
                    (side === "outgoing" ? key : neighbor).interpolation !==
                      "bezier"
                  )
                    return null;
                  const c =
                    side === "outgoing"
                      ? segmentControls(key, neighbor)
                      : segmentControls(neighbor, key);
                  const hx = side === "outgoing" ? c.x1 : c.x2,
                    hy = side === "outgoing" ? c.y1 : c.y2;
                  return (
                    <g key={side}>
                      <line
                        x1={x(key.time)}
                        y1={y(key.value)}
                        x2={x(hx)}
                        y2={y(hy)}
                        className="animation-handle-line"
                      />
                      <circle
                        cx={x(hx)}
                        cy={y(hy)}
                        r={5}
                        role="button"
                        tabIndex={p.disabled ? -1 : 0}
                        aria-label={`${side === "incoming" ? "Incoming" : "Outgoing"} Drag tangent`}
                        className="animation-tangent"
                        onPointerDown={(e) => begin(e, key, side)}
                        onPointerMove={move}
                        onPointerUp={() => end()}
                        onPointerCancel={() => end(true)}
                      />
                    </g>
                  );
                })}
              <circle
                cx={x(key.time)}
                cy={y(key.value)}
                r={key.id === p.selected ? 7 : 5}
                role="button"
                tabIndex={p.disabled ? -1 : 0}
                aria-label={`Curve keyframe ${index + 1}`}
                className={
                  key.id === p.selected
                    ? "animation-graph-key selected"
                    : "animation-graph-key"
                }
                onClick={() => {
                  p.onSelect(key.id);
                  p.onScrub(key.time);
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter") p.onSelect(key.id);
                }}
                onPointerDown={(e) => begin(e, key)}
                onPointerMove={move}
                onPointerUp={() => end()}
                onPointerCancel={() => end(true)}
              />
            </g>
          ))}
      </svg>
      {selected && mode === "value" && (
        <div className="animation-tangent-values">
          {(["incoming", "outgoing"] as const).map((side) => (
            <div key={side}>
              {side === "incoming" ? "Incoming" : "Outgoing"}
              {(["Time", "Value"] as const).map((label, i) => (
                <label key={label}>
                  {label}
                  <StudioInput
                    type="number"
                    aria-label={`${side === "incoming" ? "Incoming" : "Outgoing"} Tangent ${label}`}
                    value={Number(selected[side][i].toFixed(5))}
                    increment={i === 0 ? 1 / p.fps : 0.05}
                    disabled={p.disabled}
                    onChange={(e) => {
                      if (!e.target.value || !Number.isFinite(+e.target.value))
                        return;
                      const handle: [number, number] = [...selected[side]];
                      handle[i] = Math.max(
                        i === 0
                          ? side === "incoming"
                            ? -p.duration
                            : 0
                          : -1000000,
                        Math.min(
                          i === 0
                            ? side === "incoming"
                              ? 0
                              : p.duration
                            : 1000000,
                          +e.target.value,
                        ),
                      );
                      p.onChange(
                        replace(
                          setCurveTangent(selected, side, handle, tangentMode),
                        ),
                      );
                    }}
                  />
                </label>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

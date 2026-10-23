import type { Sequence } from "./cinematic";
import {
  addMarker,
  adjacentMarker,
  editMarker,
  type TimelineMarker,
} from "./cinematic-markers";

export function MarkerInspector(p: {
  project: Sequence;
  time: number;
  selected?: string;
  disabled: boolean;
  onSelect: (id: string) => void;
  onScrub: (time: number) => void;
  onChange: (project: Sequence) => void;
}) {
  const marker =
    p.project.markers?.find((m) => m.id === p.selected) ??
    p.project.markers?.[0];
  const edit = (change: Partial<TimelineMarker>) => {
    if (marker) p.onChange(editMarker(p.project, marker.id, change));
  };
  return (
    <section data-camera-panel="markers">
      <h3>Sequence markers</h3>
      <button
        disabled={p.disabled}
        onClick={() => {
          const next = addMarker(p.project, p.time),
            created = next.markers?.find(
              (m) => !p.project.markers?.some((old) => old.id === m.id),
            );
          p.onChange(next);
          if (created) p.onSelect(created.id);
        }}
      >
        Add marker at playhead
      </button>
      <div className="cinema-pair">
        {([-1, 1] as const).map((direction) => (
          <button
            key={direction}
            disabled={
              p.disabled || !adjacentMarker(p.project, p.time, direction)
            }
            onClick={() => {
              const m = adjacentMarker(p.project, p.time, direction);
              if (m) {
                p.onSelect(m.id);
                p.onScrub(m.time);
              }
            }}
          >
            {direction < 0 ? "Previous marker" : "Next marker"}
          </button>
        ))}
      </div>
      {marker ? (
        <>
          <label>
            Marker
            <select
              aria-label="Sequence markers"
              value={marker.id}
              onChange={(e) => p.onSelect(e.target.value)}
              disabled={p.disabled}
            >
              {p.project.markers!.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.time.toFixed(3)}s · {m.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Name
            <input
              aria-label="Marker name"
              value={marker.name}
              maxLength={200}
              disabled={p.disabled}
              onChange={(e) => edit({ name: e.target.value })}
            />
          </label>
          <label>
            Comment
            <textarea
              aria-label="Marker comment"
              value={marker.comment}
              maxLength={2000}
              disabled={p.disabled}
              onChange={(e) => edit({ comment: e.target.value })}
            />
          </label>
          <div className="cinema-pair">
            <label>
              Time
              <input
                aria-label="Marker time"
                type="number"
                min={0}
                max={p.project.duration}
                step={1 / p.project.fps}
                value={Number(marker.time.toFixed(5))}
                disabled={p.disabled}
                onChange={(e) => {
                  if (e.target.value && Number.isFinite(+e.target.value))
                    edit({ time: +e.target.value });
                }}
              />
            </label>
            <label>
              Range duration
              <input
                aria-label="Marker duration"
                type="number"
                min={0}
                max={p.project.duration - marker.time}
                step={1 / p.project.fps}
                value={Number(marker.duration.toFixed(5))}
                disabled={p.disabled}
                onChange={(e) => {
                  if (e.target.value && Number.isFinite(+e.target.value))
                    edit({ duration: +e.target.value });
                }}
              />
            </label>
          </div>
          <label>
            Color
            <input
              aria-label="Marker color"
              type="color"
              value={marker.color}
              disabled={p.disabled}
              onChange={(e) => edit({ color: e.target.value })}
            />
          </label>
          <div className="cinema-pair">
            <button
              disabled={p.disabled}
              onClick={() => p.onScrub(marker.time)}
            >
              Go to marker
            </button>
            <button
              disabled={p.disabled}
              onClick={() =>
                p.onChange({
                  ...p.project,
                  markers: p.project.markers!.filter((m) => m.id !== marker.id),
                })
              }
            >
              Delete marker
            </button>
          </div>
        </>
      ) : (
        <p>
          Press M to add a named marker at the playhead. Use range markers for sections such as music cues or shots.
        </p>
      )}
    </section>
  );
}

import { useRef } from "react";
export function MarkerLane(p: {
  project: Sequence;
  selected?: string;
  disabled: boolean;
  onSelect: (id: string) => void;
  onScrub: (time: number) => void;
  onChange: (project: Sequence, before?: Sequence) => void;
  onPreview: (project: Sequence) => void;
}) {
  const drag = useRef<{
    before: Sequence;
    next: Sequence;
    id: string;
    startX: number;
    width: number;
    time: number;
  } | null>(null);
  return (
    <div className="cinematic-marker-lane" aria-label="Marker track">
      {p.project.markers?.map((m) => (
        <button
          key={m.id}
          className={p.selected === m.id ? "selected" : ""}
          aria-label={`Marker ${m.name}`}
          title={m.comment || m.name}
          disabled={p.disabled}
          style={{
            left: `${(m.time / p.project.duration) * 100}%`,
            width: m.duration
              ? `${(m.duration / p.project.duration) * 100}%`
              : undefined,
            color: m.color,
            borderColor: m.color,
          }}
          onClick={() => {
            p.onSelect(m.id);
            p.onScrub(m.time);
          }}
          onKeyDown={(e) => {
            if (!["ArrowLeft", "ArrowRight"].includes(e.key)) return;
            e.preventDefault();
            e.stopPropagation();
            p.onChange(
              editMarker(p.project, m.id, {
                time: m.time + (e.key === "ArrowLeft" ? -1 : 1) / p.project.fps,
              }),
            );
          }}
          onPointerDown={(e) => {
            e.stopPropagation();
            e.currentTarget.setPointerCapture(e.pointerId);
            p.onSelect(m.id);
            drag.current = {
              before: p.project,
              next: p.project,
              id: m.id,
              startX: e.clientX,
              width:
                e.currentTarget.parentElement!.getBoundingClientRect().width,
              time: m.time,
            };
          }}
          onPointerMove={(e) => {
            const action = drag.current;
            if (!action || action.id !== m.id) return;
            action.next = editMarker(action.before, m.id, {
              time:
                action.time +
                ((e.clientX - action.startX) / action.width) *
                  action.before.duration,
            });
            p.onPreview(action.next);
          }}
          onPointerUp={() => {
            const action = drag.current;
            drag.current = null;
            if (action && action.next !== action.before)
              p.onChange(action.next, action.before);
          }}
          onPointerCancel={() => {
            if (drag.current) p.onPreview(drag.current.before);
            drag.current = null;
          }}
        >
          ⚑ {m.name}
        </button>
      ))}
    </div>
  );
}

import { useRef, useState } from "react";
import {
  channelLabels,
  channelLimits,
  moveTrackKey,
  type AnimationChannel,
  type AnimationTrack,
  type ScalarKey,
} from "./animation-tracks";
import { graphPath } from "./animation-curves";
import {
  addChannelKey,
  channelValue,
  duplicateChannelKey,
} from "./cinematic-tracks";
import { sampleSequence, type Sequence } from "./cinematic";
import CurveEditor from "./CurveEditor";
import "./animation-editor.css";

export default function CinematicTimeline(p: {
  sequence: Sequence;
  time: number;
  zoom: number;
  snap: boolean;
  disabled: boolean;
  channels?: readonly AnimationChannel[];
  onScrub: (time: number) => void;
  onChange: (next: Sequence, before?: Sequence) => void;
  onPreview: (sequence: Sequence) => void;
}) {
  const [chosenChannel, setChannel] = useState<AnimationChannel>("position.x"),
    [selected, setSelected] = useState<string>(),
    [graph, setGraph] = useState(false);
  const channels =
    p.channels ?? (Object.keys(channelLabels) as AnimationChannel[]);
  const channel = channels.includes(chosenChannel)
    ? chosenChannel
    : channels[0];
  const tracks = p.sequence.tracks ?? [],
    track = tracks.find((t) => t.channel === channel) ?? { channel, keys: [] };
  const key = track.keys.find((k) => k.id === selected) ?? track.keys[0];
  const drag = useRef<{
    start: Sequence;
    next: Sequence;
    channel: AnimationChannel;
    id: string;
  } | null>(null);
  const swap = (s: Sequence, value: AnimationTrack): Sequence => ({
    ...s,
    tracks: [
      ...(s.tracks ?? []).filter((t) => t.channel !== value.channel),
      value,
    ],
  });
  function updateTrack(next: AnimationTrack, before?: AnimationTrack) {
    p.onChange(
      swap(p.sequence, next),
      before ? swap(p.sequence, before) : undefined,
    );
  }
  function add(time = p.time) {
    const next = addChannelKey(p.sequence, channel, time);
    const frame = Math.round(time * p.sequence.fps);
    setSelected(
      next
        .tracks!.find((t) => t.channel === channel)!
        .keys.find((k) => Math.round(k.time * p.sequence.fps) === frame)?.id,
    );
    p.onChange(next);
  }
  function edit(patch: Partial<ScalarKey>) {
    if (!key) return;
    let next = track;
    if (patch.time !== undefined)
      next = moveTrackKey(
        track,
        key.id,
        patch.time,
        p.sequence.fps,
        p.sequence.duration,
      );
    else
      next = {
        ...track,
        keys: track.keys.map((k) => (k.id === key.id ? { ...k, ...patch } : k)),
      };
    updateTrack(next);
  }
  const limits = channelLimits[channel];
  return (
    <div className="animation-workspace">
      <div className="animation-toolbar">
        <button aria-pressed={!graph} onClick={() => setGraph(false)}>
          Tracks
        </button>
        <button aria-pressed={graph} onClick={() => setGraph(true)}>
          Curve editor
        </button>
        <label>
          Track
          <select
            aria-label="Active animation track"
            value={channel}
            onChange={(e) => {
              setChannel(e.target.value as AnimationChannel);
              setSelected(undefined);
            }}
          >
            {channels.map((id) => (
              <option value={id} key={id}>
                {channelLabels[id]}
              </option>
            ))}
          </select>
        </label>
        <button
          disabled={p.disabled}
          onClick={() => add()}
          aria-label="Add track keyframe"
        >
          Keyframe in track
        </button>
        <span>{track.keys.length} Track keys</span>
        <label>
          <input
            type="checkbox"
            aria-label="Mute track"
            checked={!!track.muted}
            disabled={p.disabled}
            onChange={(e) => updateTrack({ ...track, muted: e.target.checked })}
          />
          Mute
        </label>
      </div>
      {key ? (
        <fieldset className="animation-key-properties" disabled={p.disabled}>
          <label>
            Key
            <select
              aria-label="Selected track keyframe"
              value={key.id}
              onChange={(e) => {
                setSelected(e.target.value);
                const key = track.keys.find((k) => k.id === e.target.value);
                if (key) p.onScrub(key.time);
              }}
            >
              {track.keys.map((k, i) => (
                <option key={k.id} value={k.id}>
                  {i + 1} · {Number(k.time.toFixed(4))}s
                </option>
              ))}
            </select>
          </label>
          <label>
            Time
            <input
              aria-label="Track keyframe time"
              type="number"
              min={0}
              max={p.sequence.duration}
              step={1 / p.sequence.fps}
              value={Number(key.time.toFixed(5))}
              onChange={(e) => {
                if (e.target.value) edit({ time: +e.target.value });
              }}
            />
          </label>
          <label>
            Value
            <input
              aria-label="Track value"
              type="number"
              min={limits[0]}
              max={limits[1]}
              step={0.05}
              value={Number(key.value.toFixed(5))}
              onChange={(e) => {
                if (e.target.value && Number.isFinite(+e.target.value))
                  edit({
                    value: Math.max(
                      limits[0],
                      Math.min(limits[1], +e.target.value),
                    ),
                  });
              }}
            />
          </label>
          <label>
            Interpolation
            <select
              aria-label="Track interpolation"
              value={key.interpolation}
              onChange={(e) =>
                edit({
                  interpolation: e.target.value as ScalarKey["interpolation"],
                })
              }
            >
              <option value="bezier">Bézier</option>
              <option value="linear">Linear</option>
              <option value="hold">Hold / Cut</option>
            </select>
          </label>
          <button
            onClick={() => {
              const next = duplicateChannelKey(p.sequence, channel, key.id);
              const added = next.tracks
                ?.find((t) => t.channel === channel)
                ?.keys.find((k) => !track.keys.some((old) => old.id === k.id));
              if (added) {
                setSelected(added.id);
                p.onScrub(added.time);
                p.onChange(next);
              }
            }}
          >
            Duplicate track key
          </button>
          <button
            onClick={() =>
              updateTrack({
                ...track,
                defaultValue: key.value,
                keys: track.keys.filter((k) => k.id !== key.id),
              })
            }
          >
            Delete track key
          </button>
        </fieldset>
      ) : (
        <p className="animation-empty">
          No keys in this track. Base value:{" "}
          {Number(
            channelValue(sampleSequence(p.sequence, p.time), channel).toFixed(
              3,
            ),
          )}
          . Add a keyframe at the playhead.
        </p>
      )}
      <div className="animation-content">
        {graph ? (
          <CurveEditor
            track={track}
            duration={p.sequence.duration}
            fps={p.sequence.fps}
            time={p.time}
            snap={p.snap}
            disabled={p.disabled}
            selected={key?.id}
            onSelect={setSelected}
            onScrub={p.onScrub}
            onChange={updateTrack}
            onPreview={(next) => p.onPreview(swap(p.sequence, next))}
          />
        ) : (
          <div className="animation-track-scroll">
            <div
              className="animation-tracks"
              style={{ minWidth: `${p.zoom * 100}%` }}
            >
              <div className="animation-ruler">
                <span>Channels</span>
                <div>
                  {Array.from({ length: 9 }, (_, i) => (
                    <button
                      key={i}
                      style={{ left: `${(i / 8) * 100}%` }}
                      onClick={() => p.onScrub((i * p.sequence.duration) / 8)}
                    >
                      {Number(((i * p.sequence.duration) / 8).toFixed(2))}s
                    </button>
                  ))}
                </div>
              </div>
              {channels.map((field) => {
                const lane = tracks.find((t) => t.channel === field) ?? {
                  channel: field,
                  keys: [],
                };
                const min = Math.min(0, ...lane.keys.map((k) => k.value)),
                  max = Math.max(1, ...lane.keys.map((k) => k.value));
                return (
                  <div
                    key={field}
                    className={`animation-lane ${field === channel ? "active" : ""} ${lane.muted ? "muted" : ""}`}
                    data-channel={field}
                  >
                    <button
                      onClick={() => {
                        setChannel(field);
                        setSelected(undefined);
                      }}
                    >
                      {channelLabels[field]}
                    </button>
                    <div
                      className="animation-lane-body"
                      onPointerDown={(e) => {
                        if (p.disabled || e.target !== e.currentTarget) return;
                        const r = e.currentTarget.getBoundingClientRect();
                        setChannel(field);
                        p.onScrub(
                          ((e.clientX - r.left) / r.width) *
                            p.sequence.duration,
                        );
                      }}
                    >
                      <svg
                        viewBox="0 0 1000 32"
                        preserveAspectRatio="none"
                        aria-hidden="true"
                      >
                        <path
                          d={graphPath(
                            lane,
                            (t) => (t / p.sequence.duration) * 1000,
                            (v) => 27 - ((v - min) / (max - min)) * 22,
                          )}
                        />
                      </svg>
                      {lane.keys.map((k, i) => (
                        <button
                          key={k.id}
                          className={`animation-key ${field === channel && key?.id === k.id ? "selected" : ""}`}
                          aria-label={`${channelLabels[field]} Keyframe ${i + 1}`}
                          disabled={p.disabled}
                          style={{
                            left: `${(k.time / p.sequence.duration) * 100}%`,
                          }}
                          onClick={() => {
                            setChannel(field);
                            setSelected(k.id);
                            p.onScrub(k.time);
                          }}
                          onKeyDown={(e) => {
                            if (e.key !== "ArrowLeft" && e.key !== "ArrowRight")
                              return;
                            e.preventDefault();
                            e.stopPropagation();
                            setChannel(field);
                            setSelected(k.id);
                            const next = moveTrackKey(
                              lane,
                              k.id,
                              k.time +
                                (e.key === "ArrowLeft" ? -1 : 1) /
                                  p.sequence.fps,
                              p.sequence.fps,
                              p.sequence.duration,
                            );
                            p.onChange(swap(p.sequence, next));
                          }}
                          onPointerDown={(e) => {
                            e.stopPropagation();
                            e.currentTarget.setPointerCapture(e.pointerId);
                            setChannel(field);
                            setSelected(k.id);
                            drag.current = {
                              start: p.sequence,
                              next: p.sequence,
                              channel: field,
                              id: k.id,
                            };
                          }}
                          onPointerMove={(e) => {
                            const action = drag.current;
                            if (!action || action.id !== k.id) return;
                            const r =
                              e.currentTarget.parentElement!.getBoundingClientRect();
                            const t = Math.max(
                              0,
                              Math.min(
                                p.sequence.duration,
                                ((e.clientX - r.left) / r.width) *
                                  p.sequence.duration,
                              ),
                            );
                            const original = action.start.tracks!.find(
                              (track) => track.channel === action.channel,
                            )!;
                            let moved = p.snap
                              ? moveTrackKey(
                                  original,
                                  k.id,
                                  t,
                                  p.sequence.fps,
                                  p.sequence.duration,
                                )
                              : {
                                  ...original,
                                  keys: original.keys
                                    .map((key) =>
                                      key.id === k.id
                                        ? { ...key, time: t }
                                        : key,
                                    )
                                    .sort((a, b) => a.time - b.time),
                                };
                            if (
                              !p.snap &&
                              original.keys.some(
                                (key) =>
                                  key.id !== k.id &&
                                  Math.abs(key.time - t) < 1e-8,
                              )
                            )
                              moved = original;
                            action.next = swap(action.start, moved);
                            p.onPreview(action.next);
                          }}
                          onPointerUp={() => {
                            const action = drag.current;
                            drag.current = null;
                            if (action && action.next !== action.start)
                              p.onChange(action.next, action.start);
                          }}
                          onPointerCancel={() => {
                            if (drag.current) p.onPreview(drag.current.start);
                            drag.current = null;
                          }}
                        >
                          ◆
                        </button>
                      ))}
                      <i
                        className="animation-playhead"
                        style={{
                          left: `${(p.time / p.sequence.duration) * 100}%`,
                        }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

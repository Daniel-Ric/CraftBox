import { useRef, useState } from "react";
import type { AtmosphereSettings } from "./atmosphere-settings";
import {
  cloudChannelLabels,
  cloudChannelLimits,
  cloudChannelValue,
  createCloudLayer,
  keyCloudValue,
  sampleCloudLayer,
  type CloudChannel,
  type CloudLayer,
  type CloudVector,
} from "./cloud-layers";
import { splitCurveAt } from "./animation-curves";
import type { AnimationTrack } from "./animation-tracks";
import CurveEditor from "./CurveEditor";
import "./cloud-layer-controls.css";
export type CloudEditorState = {
  selected?: string;
  placing: boolean;
  mode: "translate" | "rotate" | "scale";
  onSelect: (id: string) => void;
  onTransform: (
    id: string,
    change: Partial<Pick<CloudLayer, "position" | "size" | "rotation">>,
  ) => void;
};
export type CloudAnimationContext = {
  time: number;
  duration: number;
  fps: number;
  onScrub: (time: number) => void;
};
export default function CloudLayerControls(p: {
  value: AtmosphereSettings;
  onChange: (value: AtmosphereSettings) => void;
  disabled: boolean;
  editor?: CloudEditorState;
  onPlace?: () => void;
  onMode?: (mode: CloudEditorState["mode"]) => void;
  animation?: CloudAnimationContext;
}) {
  const [selected, select] = useState<string>(),
    [channel, setChannel] = useState<CloudChannel>("position.x"),
    [keyId, setKeyId] = useState<string>(),
    [past, setPast] = useState<AtmosphereSettings[]>([]),
    [future, setFuture] = useState<AtmosphereSettings[]>([]);
  const curveBaseline = useRef<AtmosphereSettings | null>(null);
  const layers = p.value.cloudLayers ?? [],
    active =
      layers.find((l) => l.id === (p.editor?.selected ?? selected)) ??
      layers.at(-1),
    time = p.animation?.time ?? 0;
  const shown = active ? sampleCloudLayer(active, time) : undefined;
  function choose(id: string) {
    curveBaseline.current = null;
    select(id);
    p.editor?.onSelect(id);
    setKeyId(undefined);
  }
  function commit(next: AtmosphereSettings, before = p.value) {
    setPast((s) => [...s.slice(-49), before]);
    setFuture([]);
    p.onChange(next);
  }
  function replace(layer: CloudLayer, before?: AtmosphereSettings) {
    commit(
      {
        ...p.value,
        cloudLayers: layers.map((l) => (l.id === layer.id ? layer : l)),
      },
      before,
    );
  }
  function add(kind: CloudLayer["kind"]) {
    const layer = createCloudLayer(kind);
    commit({ ...p.value, cloudLayers: [...layers, layer] });
    choose(layer.id);
  }
  function edit(patch: Partial<CloudLayer>) {
    if (active) replace({ ...active, ...patch });
  }
  function number(
    label: string,
    value: number,
    min: number,
    max: number,
    step: number,
    onChange: (n: number) => void,
  ) {
    return (
      <label className="render-control" key={label}>
        {label}
        <input
          aria-label={label}
          type="number"
          min={min}
          max={max}
          step={step}
          value={Number(value.toFixed(5))}
          onChange={(e) => {
            if (e.target.value && Number.isFinite(+e.target.value))
              onChange(Math.max(min, Math.min(max, +e.target.value)));
          }}
        />
      </label>
    );
  }
  const track: AnimationTrack<CloudChannel> | undefined = active
    ? (active.tracks.find((t) => t.channel === channel) ?? {
        channel,
        keys: [],
        defaultValue: cloudChannelValue(active, channel),
      })
    : undefined;
  function trackChange(
    next: AnimationTrack<CloudChannel>,
    before?: AtmosphereSettings,
  ) {
    if (active)
      replace(
        {
          ...active,
          tracks: [
            ...active.tracks.filter((t) => t.channel !== next.channel),
            next,
          ],
        },
        before,
      );
  }
  function framePreset() {
    const bank = {
      ...createCloudLayer("bank"),
      name: "Lower cloud sea",
      position: [0, -0.8, 0] as CloudVector,
      size: [4, 1.2, 4] as CloudVector,
      seed: 42,
    };
    const back = {
      ...createCloudLayer("volume"),
      name: "Rear cloud bank",
      position: [-0.8, -0.45, -1.1] as CloudVector,
      size: [2, 1.2, 1.8] as CloudVector,
      seed: 16,
    };
    const front = {
      ...createCloudLayer("volume"),
      name: "Foreground",
      position: [0.9, -0.7, 0.8] as CloudVector,
      size: [2, 0.9, 1.5] as CloudVector,
      seed: 90,
    };
    commit({
      ...p.value,
      clouds: false,
      cloudLayers: [bank, back, front],
      cloudOpening: {
        ...p.value.cloudOpening,
        strength: 1,
        radius: [0.75, 0.75],
        floor: -0.5,
      },
    });
    choose(bank.id);
  }
  return (
    <section className="cloud-authoring" data-inspector="atmosphere">
      <fieldset disabled={p.disabled}>
        <legend>Cloud regions</legend>
        <p className="inspector-intro">
          Independent layers and volumetric clouds. Position and size are relative to the scene; 1 equals the scene radius.
        </p>
        <div className="cloud-actions">
          <button disabled={layers.length >= 8} onClick={() => add("bank")}>
            Add cloud layer
          </button>
          <button disabled={layers.length >= 8} onClick={() => add("volume")}>
            Add cloud volume
          </button>
        </div>
        <button onClick={framePreset}>Frame island with clouds</button>
        <div className="cloud-object-list" aria-label="Cloud regions">
          {layers.map((l) => (
            <div key={l.id}>
              <input
                type="checkbox"
                aria-label={`${l.name} Enable cloud`}
                checked={l.enabled}
                onChange={(e) => replace({ ...l, enabled: e.target.checked })}
              />
              <button
                aria-pressed={active?.id === l.id}
                onClick={() => choose(l.id)}
              >
                {l.name}
              </button>
              <small>{l.kind === "bank" ? "Layer" : "Volume"}</small>
            </div>
          ))}
        </div>
        <div className="cloud-actions">
          <button
            disabled={!past.length}
            onClick={() => {
              const before = past.at(-1)!;
              setFuture((f) => [p.value, ...f]);
              setPast((v) => v.slice(0, -1));
              p.onChange(before);
            }}
          >
            Undo cloud edit
          </button>
          <button
            disabled={!future.length}
            onClick={() => {
              setPast((v) => [...v, p.value]);
              p.onChange(future[0]);
              setFuture((v) => v.slice(1));
            }}
          >
            Redo cloud edit
          </button>
        </div>
        {active && shown && (
          <>
            <label className="render-control">
              Name
              <input
                aria-label="Cloud region name"
                value={active.name}
                maxLength={200}
                onChange={(e) => {
                  if (e.target.value.trim()) edit({ name: e.target.value });
                }}
              />
            </label>
            {p.onPlace && (
              <button
                aria-pressed={p.editor?.placing}
                onClick={() => {
                  if (!p.editor?.placing) choose(active.id);
                  p.onPlace?.();
                }}
              >
                {p.editor?.placing
                  ? "Cancel cloud placement"
                  : "Place cloud in view"}
              </button>
            )}
            {p.onMode && (
              <label>
                Tool
                <select
                  aria-label="Cloud tool"
                  value={p.editor?.mode ?? "translate"}
                  onChange={(e) =>
                    p.onMode?.(e.target.value as CloudEditorState["mode"])
                  }
                >
                  <option value="translate">Move</option>
                  <option value="rotate">Rotate</option>
                  <option value="scale">Scale</option>
                </select>
              </label>
            )}
            {(["position", "size", "rotation"] as const).map((property) => (
              <div className="cloud-vector" key={property}>
                {(["x", "y", "z"] as const).map((axis, i) => {
                  const c = `${property}.${axis}` as CloudChannel,
                    [min, max] = cloudChannelLimits[c];
                  return (
                    <div key={axis}>
                      {number(
                        cloudChannelLabels[c],
                        shown[property][i],
                        min,
                        max,
                        property === "rotation" ? 1 : 0.01,
                        (n) => {
                          if (
                            active.tracks.some((t) => t.channel === c) &&
                            p.animation
                          )
                            replace(
                              keyCloudValue(
                                active,
                                c,
                                time,
                                n,
                                p.animation.fps,
                              ),
                            );
                          else {
                            const v = [...active[property]] as CloudVector;
                            v[i] = n;
                            edit({ [property]: v });
                          }
                        },
                      )}
                    </div>
                  );
                })}
              </div>
            ))}
            <label className="render-control">
              Cloud color
              <input
                aria-label="Cloud region color"
                type="color"
                value={active.color}
                onChange={(e) => edit({ color: e.target.value })}
              />
            </label>
            {(
              [
                ["density", "Region cloud density", 0, 4],
                ["coverage", "Region coverage", 0, 1],
                ["dissolve", "Region breakup", 0, 1],
                ["softness", "Cloud edge softness", 0, 1],
                ["detail", "Region detail", 0, 1],
                ["erosion", "Region erosion", 0, 1],
                ["turbulence", "Region turbulence", 0, 1],
                ["shadows", "Region shadow casting", 0, 1],
                ["windSpeed", "Region wind speed", 0, 40],
                ["windDirection", "Region wind direction", 0, 360],
                ["seed", "Region seed", 0, 999999],
              ] as const
            ).map(([field, label, min, max]) => (
              <div key={field}>
                {number(
                  label,
                  shown[field] ?? 0,
                  min,
                  max,
                  field === "seed" ? 1 : 0.05,
                  (n) => {
                    if (
                      field in cloudChannelLimits &&
                      active.tracks.some((t) => t.channel === field) &&
                      p.animation
                    )
                      replace(
                        keyCloudValue(
                          active,
                          field as CloudChannel,
                          time,
                          n,
                          p.animation.fps,
                        ),
                      );
                    else edit({ [field]: n });
                  },
                )}
              </div>
            ))}
            <div className="cloud-actions">
              <button
                disabled={layers.length >= 8}
                onClick={() => {
                  const copy = {
                    ...active,
                    id: crypto.randomUUID(),
                    name: `${active.name.slice(0, 190)} Copy`,
                    position: [...active.position] as CloudVector,
                    size: [...active.size] as CloudVector,
                    rotation: [...active.rotation] as CloudVector,
                    tracks: structuredClone(active.tracks),
                  };
                  commit({ ...p.value, cloudLayers: [...layers, copy] });
                  choose(copy.id);
                }}
              >
                Duplicate cloud region
              </button>
              <button
                onClick={() => {
                  const remaining = layers.filter((l) => l.id !== active.id);
                  commit({ ...p.value, cloudLayers: remaining });
                  choose(remaining.at(-1)?.id ?? "");
                }}
              >
                Delete cloud region
              </button>
            </div>
          </>
        )}
      </fieldset>
      <fieldset disabled={p.disabled}>
        <legend>Opening around map</legend>
        {number(
          "Opening strength",
          p.value.cloudOpening.strength,
          0,
          1,
          0.05,
          (n) =>
            commit({
              ...p.value,
              cloudOpening: { ...p.value.cloudOpening, strength: n },
            }),
        )}
        {(["center", "radius"] as const).map((field) => (
          <div className="cloud-vector" key={field}>
            {[0, 1].map((i) =>
              number(
                `Opening ${field === "center" ? "Center" : "Radius"} ${i === 0 ? "X" : "Z"}`,
                p.value.cloudOpening[field][i],
                field === "radius" ? 0.01 : -100,
                100,
                0.05,
                (n) => {
                  const v = [...p.value.cloudOpening[field]] as [
                    number,
                    number,
                  ];
                  v[i] = n;
                  commit({
                    ...p.value,
                    cloudOpening: { ...p.value.cloudOpening, [field]: v },
                  });
                },
              ),
            )}
          </div>
        ))}
        {(
          [
            ["floor", "Opening bottom edge", -100, 100],
            ["softness", "Opening transition", 0.01, 1],
            ["irregularity", "Opening irregularity", 0, 1],
          ] as const
        ).map(([field, label, min, max]) =>
          number(label, p.value.cloudOpening[field], min, max, 0.05, (n) =>
            commit({
              ...p.value,
              cloudOpening: { ...p.value.cloudOpening, [field]: n },
            }),
          ),
        )}
      </fieldset>
      <fieldset disabled={p.disabled}>
        <legend>Cloud lighting and quality</legend>
        {number(
          "Cloud silver lining",
          p.value.cloudSilverLining,
          0,
          2,
          0.05,
          (n) => commit({ ...p.value, cloudSilverLining: n }),
        )}
        {number(
          "Cloud multiple scattering",
          p.value.cloudMultipleScattering,
          0,
          1,
          0.05,
          (n) => commit({ ...p.value, cloudMultipleScattering: n }),
        )}
        <label>
          Preview
          <select
            aria-label="Cloud preview quality"
            value={p.value.cloudPreviewQuality}
            onChange={(e) =>
              commit({
                ...p.value,
                cloudPreviewQuality: e.target
                  .value as AtmosphereSettings["cloudPreviewQuality"],
              })
            }
          >
            <option value="auto">Automatic</option>
            <option value="low">Fast</option>
            <option value="medium">Balanced</option>
            <option value="high">Fine</option>
          </select>
        </label>
        <label>
          Export
          <select
            aria-label="Cloud export quality"
            value={p.value.cloudExportQuality}
            onChange={(e) =>
              commit({
                ...p.value,
                cloudExportQuality: e.target
                  .value as AtmosphereSettings["cloudExportQuality"],
              })
            }
          >
            <option value="medium">Balanced</option>
            <option value="high">High</option>
            <option value="ultra">Ultra</option>
          </select>
        </label>
      </fieldset>
      {p.animation && active && track && (
        <fieldset disabled={p.disabled} className="cloud-animation">
          <legend>Cloud animation · sequence time</legend>
          <p>
            {time.toFixed(3)} s · {active.name}
          </p>
          <label>
            Channel
            <select
              aria-label="Cloud animation channel"
              value={channel}
              onChange={(e) => {
                setChannel(e.target.value as CloudChannel);
                setKeyId(undefined);
              }}
            >
              {Object.entries(cloudChannelLabels).map(([id, label]) => (
                <option key={id} value={id}>
                  {label}
                </option>
              ))}
            </select>
          </label>
          <button
            onClick={() => {
              const t = Math.round(time * p.animation!.fps) / p.animation!.fps;
              if (track.keys.length) {
                const next = splitCurveAt(
                  track,
                  t,
                  crypto.randomUUID(),
                  cloudChannelLimits[channel],
                );
                trackChange(next);
                setKeyId(
                  next.keys.find((k) => Math.abs(k.time - t) < 1e-8)?.id,
                );
              } else {
                const next = keyCloudValue(
                  active,
                  channel,
                  t,
                  cloudChannelValue(active, channel),
                  p.animation!.fps,
                );
                replace(next);
                setKeyId(
                  next.tracks.find((t) => t.channel === channel)?.keys[0]?.id,
                );
              }
            }}
          >
            Add cloud keyframe
          </button>
          {track.keys.length > 0 && (
            <>
              <label>
                Key
                <select
                  aria-label="Cloud keyframe"
                  value={keyId ?? track.keys[0].id}
                  onChange={(e) => setKeyId(e.target.value)}
                >
                  {track.keys.map((k) => (
                    <option key={k.id} value={k.id}>
                      {k.time.toFixed(3)} s
                    </option>
                  ))}
                </select>
              </label>
              {(() => {
                const k =
                  track.keys.find((k) => k.id === keyId) ?? track.keys[0];
                return (
                  <>
                    {number(
                      "Cloud key value",
                      k.value,
                      ...cloudChannelLimits[channel],
                      0.01,
                      (n) =>
                        trackChange({
                          ...track,
                          keys: track.keys.map((x) =>
                            x.id === k.id ? { ...x, value: n } : x,
                          ),
                        }),
                    )}
                    <label>
                      Interpolation
                      <select
                        aria-label="Cloud key interpolation"
                        value={k.interpolation}
                        onChange={(e) =>
                          trackChange({
                            ...track,
                            keys: track.keys.map((x) =>
                              x.id === k.id
                                ? {
                                    ...x,
                                    interpolation: e.target
                                      .value as typeof k.interpolation,
                                  }
                                : x,
                            ),
                          })
                        }
                      >
                        <option value="bezier">Bézier</option>
                        <option value="linear">Linear</option>
                        <option value="hold">Hold</option>
                      </select>
                    </label>
                    <button
                      onClick={() =>
                        trackChange({
                          ...track,
                          keys: track.keys.filter((x) => x.id !== k.id),
                        })
                      }
                    >
                      Delete cloud keyframe
                    </button>
                  </>
                );
              })()}
            </>
          )}
          <CurveEditor
            onCancel={() => {
              curveBaseline.current = null;
            }}
            track={track}
            label={cloudChannelLabels[channel]}
            limits={cloudChannelLimits[channel]}
            duration={p.animation.duration}
            fps={p.animation.fps}
            time={time}
            snap
            disabled={p.disabled}
            selected={keyId ?? track.keys[0]?.id}
            onSelect={setKeyId}
            onScrub={p.animation.onScrub}
            onPreview={(next) => {
              curveBaseline.current ??= p.value;
              p.onChange({
                ...p.value,
                cloudLayers: layers.map((l) =>
                  l.id === active.id
                    ? {
                        ...l,
                        tracks: [
                          ...l.tracks.filter((t) => t.channel !== channel),
                          next,
                        ],
                      }
                    : l,
                ),
              });
            }}
            onChange={(next, before) => {
              const baseline = curveBaseline.current ?? {
                ...p.value,
                cloudLayers: layers.map((l) =>
                  l.id === active.id && before
                    ? {
                        ...l,
                        tracks: [
                          ...l.tracks.filter((t) => t.channel !== channel),
                          before,
                        ],
                      }
                    : l,
                ),
              };
              curveBaseline.current = null;
              trackChange(next, baseline);
            }}
          />
        </fieldset>
      )}
    </section>
  );
}

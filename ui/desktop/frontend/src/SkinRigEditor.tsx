import HeaderActionIcon from "./HeaderActionIcon";
import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import AssetViewport, {
  type AssetViewportHandle,
  type SkinRigScene,
} from "./AssetViewport";
import {
  neutralPose,
  type SkinPose,
  type CosmeticBinding,
} from "./asset-geometry";
import CosmeticBindingPanel from "./CosmeticBindingPanel";
import WireframeControls from "./WireframeControls";
import type { WireframeSettings, WireframeStats } from "./wireframe-settings";
import { limitedAxes, mirrorPose, type RigPart } from "./human-skeleton";
import type { PackObjectPreview } from "./bridge";
import { Button } from "@/components/ui/button";
import {
  Undo2,
  Redo2,
  RotateCcw,
  Move,
  Rotate3D,
  Focus,
  FlipHorizontal2,
  Save,
  Play,
  Pause,
  Search,
} from "lucide-react";
import "./skin-rig-editor.css";

type Vec3 = [number, number, number];
const clone = (pose: SkinPose) => structuredClone(pose);
export const skinPosePresets: Record<string, SkinPose> = {
  Standing: { ...neutralPose },
  "A pose": {
    ...neutralPose,
    axes: { leftArm: [0, 0, -25], rightArm: [0, 0, 25] },
  },
  "T pose": {
    ...neutralPose,
    axes: { leftArm: [0, 0, -90], rightArm: [0, 0, 90] },
  },
  Sitting: {
    ...neutralPose,
    axes: {
      leftLeg: [90, 0, 0],
      rightLeg: [90, 0, 0],
      leftKnee: [-90, 0, 0],
      rightKnee: [-90, 0, 0],
      leftElbow: [15, 0, 0],
      rightElbow: [15, 0, 0],
    },
    positions: { character: [0, -5, 0] },
  },
  Wave: {
    ...neutralPose,
    axes: {
      rightArm: [145, 0, -35],
      rightElbow: [80, 0, 0],
      rightWrist: [0, 0, 25],
      head: [0, -15, 0],
    },
  },
  Running: {
    ...neutralPose,
    axes: {
      body: [12, 0, 0],
      leftArm: [-45, 0, 0],
      rightArm: [40, 0, 0],
      leftElbow: [85, 0, 0],
      rightElbow: [85, 0, 0],
      leftLeg: [55, 0, 0],
      rightLeg: [-30, 0, 0],
      leftKnee: [-20, 0, 0],
      rightKnee: [-110, 0, 0],
    },
  },
};

export default function SkinRigEditor(p: {
  scene: SkinRigScene;
  pose: SkinPose;
  setPose: (pose: SkinPose) => void;
  walking: boolean;
  setWalking: (value: boolean) => void;
  attachments: PackObjectPreview[];
  effects: boolean;
  setEffects: (value: boolean) => void;
  children: ReactNode;
}) {
  const viewport = useRef<AssetViewportHandle>(null),
    current = useRef(p.pose),
    dragStart = useRef<SkinPose | undefined>(undefined);
  current.current = p.pose;
  const [parts, setParts] = useState<RigPart[]>([]),
    [selected, setSelected] = useState("character"),
    [mode, setMode] = useState<"translate" | "rotate">("rotate"),
    [space, setSpace] = useState<"local" | "world">("local"),
    [bones, setBones] = useState(true),
    [tab, setTab] = useState<"pose" | "cosmetics">("pose"),
    [query, setQuery] = useState(""),
    [error, setError] = useState("");
  const [wireframeStats, setWireframeStats] = useState<WireframeStats>();
  const [wireframe, setWireframe] = useState(false),
    [wireframeOptions, setWireframeOptions] = useState<WireframeSettings>({});
  const [undo, setUndo] = useState<SkinPose[]>([]),
    [redo, setRedo] = useState<SkinPose[]>([]),
    [presetName, setPresetName] = useState("My pose");
  const [saved, setSaved] = useState<Record<string, SkinPose>>(() => {
    try {
      return JSON.parse(
        localStorage.getItem("craftbox:skin-rig-poses-v1") ?? "{}",
      );
    } catch {
      return {};
    }
  });
  const selectedPart = parts.find((part) => part.id === selected),
    isTarget = selectedPart?.kind === "target" || selectedPart?.kind === "pole";
  const commit = (next: SkinPose) => {
    setUndo((old) => [...old.slice(-49), clone(current.current)]);
    setRedo([]);
    p.setWalking(false);
    p.setPose(next);
  };
  const beginEdit = () => {
    dragStart.current ??= clone(current.current);
  };
  const endEdit = () => {
    if (
      dragStart.current &&
      JSON.stringify(dragStart.current) !== JSON.stringify(current.current)
    ) {
      const start = dragStart.current;
      setUndo((old) => [...old.slice(-49), start]);
      setRedo([]);
    }
    dragStart.current = undefined;
  };
  const cosmetics = useMemo(() => {
    const groups = new Map<
      string,
      { key: string; name: string; layers: number }
    >();
    for (const item of p.attachments) {
      const key = item.object.definition;
      const group = groups.get(key);
      if (group) group.layers++;
      else
        groups.set(key, {
          key,
          name: item.object.name.replace(/^[^:]+:/, "").replaceAll("_", " "),
          layers: 1,
        });
    }
    return [...groups.values()];
  }, [p.attachments]);
  const changeBinding = (key: string, patch?: CosmeticBinding) => {
    const cosmetics = { ...current.current.cosmetics };
    if (patch) cosmetics[key] = { ...cosmetics[key], ...patch };
    else delete cosmetics[key];
    const next = { ...current.current, cosmetics };
    if (dragStart.current) {
      current.current = next;
      p.setWalking(false);
      p.setPose(next);
    } else commit(next);
  };
  const bindingPanel = (group: (typeof cosmetics)[number], select = false) => (
    <CosmeticBindingPanel
      key={group.key}
      name={group.name}
      layers={group.layers}
      settings={p.pose.cosmetics?.[group.key] ?? {}}
      parts={parts}
      onChange={(patch) => changeBinding(group.key, patch)}
      onReset={() => changeBinding(group.key)}
      onBegin={beginEdit}
      onEnd={endEdit}
      onSelect={
        select
          ? () => {
              setSelected(`cosmetic:${group.key}`);
              setTab("pose");
            }
          : undefined
      }
    />
  );
  const undoPose = () => {
    const previous = undo.at(-1);
    if (!previous) return;
    setRedo((old) => [...old, clone(current.current)]);
    setUndo((old) => old.slice(0, -1));
    p.setWalking(false);
    p.setPose(clone(previous));
  };
  const redoPose = () => {
    const next = redo.at(-1);
    if (!next) return;
    setUndo((old) => [...old, clone(current.current)]);
    setRedo((old) => old.slice(0, -1));
    p.setWalking(false);
    p.setPose(clone(next));
  };
  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (
        (event.target as HTMLElement)?.closest(
          "input,textarea,select,[contenteditable]",
        )
      )
        return;
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "z") {
        event.preventDefault();
        event.stopPropagation();
        event.shiftKey ? redoPose() : undoPose();
      } else if (event.key.toLowerCase() === "w") setMode("translate");
      else if (event.key.toLowerCase() === "e") setMode("rotate");
      else if (event.key.toLowerCase() === "f")
        viewport.current?.focusRig(selected);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [undo, redo, selected]);
  const angles = p.pose.axes?.[selected] ?? [0, 0, 0];
  const field = isTarget
    ? selectedPart?.kind === "pole"
      ? "poles"
      : "targets"
    : "positions";
  const position = p.pose[field]?.[selected] ??
    (isTarget ? viewport.current?.rigPosition(selected) : undefined) ?? [
      0, 0, 0,
    ];
  const values = mode === "translate" || isTarget ? position : angles;
  const updateAxis = (axis: number, value: number) => {
    if (!Number.isFinite(value)) return;
    const next: [number, number, number] = [...values];
    next[axis] = value;
    if (mode === "rotate" && !isTarget)
      commit({
        ...p.pose,
        axes: {
          ...p.pose.axes,
          [selected]: limitedAxes(selected, next, p.pose.limits !== false),
        },
      });
    else {
      const limit =
        p.pose.limits !== false &&
        !isTarget &&
        selected !== "character" &&
        selectedPart?.kind !== "cosmetic"
          ? 2
          : 128;
      next[axis] = Math.max(-limit, Math.min(limit, value));
      commit({ ...p.pose, [field]: { ...p.pose[field], [selected]: next } });
    }
  };
  const enableIK = (id: string, value: boolean) => {
    if (value && parts.find((p) => p.id === id)?.available === false) {
      setError(
        "This imported chain uses indirect joints. Use its existing FK controls.",
      );
      return;
    }
    const targets = { ...p.pose.targets };
    if (value) targets[id] = viewport.current?.rigPosition(id) ?? [0, 0, 0];
    else delete targets[id];
    commit({ ...p.pose, targets });
    if (value) {
      setSelected(id);
      setMode("translate");
    }
  };
  const common = [
    "character",
    "waist",
    "body",
    "head",
    "leftArm",
    "leftElbow",
    "leftWrist",
    "rightArm",
    "rightElbow",
    "rightWrist",
    "leftLeg",
    "leftKnee",
    "leftAnkle",
    "rightLeg",
    "rightKnee",
    "rightAnkle",
  ];
  const ordered = useMemo(
    () =>
      [...parts].sort((a, b) => {
        const ai = common.findIndex(
            (n) => n.toLowerCase() === a.id.toLowerCase(),
          ),
          bi = common.findIndex((n) => n.toLowerCase() === b.id.toLowerCase());
        return (
          (ai < 0 ? 1000 : ai) - (bi < 0 ? 1000 : bi) ||
          a.label.localeCompare(b.label)
        );
      }),
    [parts],
  );
  return (
    <div className="skin-rig-editor">
      <div className="skin-rig-toolbar">
        <div role="group" aria-label="Rig tool">
          <Button title="Move" aria-label="Move" className="toolbar-icon-action"
            variant={mode === "translate" ? "secondary" : "ghost"}
            onClick={() => setMode("translate")}
          >
            <Move />
            <span className="toolbar-action-label">Move</span><kbd>W</kbd>
          </Button>
          <Button title="Rotate" aria-label="Rotate" className="toolbar-icon-action"
            variant={mode === "rotate" ? "secondary" : "ghost"}
            disabled={isTarget}
            onClick={() => setMode("rotate")}
          >
            <Rotate3D />
            <span className="toolbar-action-label">Rotate</span><kbd>E</kbd>
          </Button>
        </div>
        <select
          aria-label="Transform space"
          value={space}
          onChange={(e) => setSpace(e.target.value as "local" | "world")}
        >
          <option value="local">Local axes</option>
          <option value="world">World axes</option>
        </select>
        <Button
          variant="ghost"
          aria-label="Undo pose"
          disabled={!undo.length}
          onClick={undoPose}
        >
          <Undo2 />
        </Button>
        <Button
          variant="ghost"
          aria-label="Redo pose"
          disabled={!redo.length}
          onClick={redoPose}
        >
          <Redo2 />
        </Button>
        <Button title="Focus" aria-label="Focus" className="toolbar-icon-action"
          variant="ghost"
          onClick={() => viewport.current?.focusRig(selected)}
        >
          <Focus />
          <span className="toolbar-action-label">Focus</span></Button>
        <Button title="Fit character" aria-label="Fit character" className="toolbar-icon-action" variant="ghost" onClick={() => viewport.current?.reset()}><HeaderActionIcon label="Fit character"/>
          <span className="toolbar-action-label">Fit character</span></Button>
        <select
          aria-label="Rig camera view"
          defaultValue="perspective"
          onChange={(e) => {
            if (e.target.value !== "perspective")
              viewport.current?.view(
                e.target.value as "front" | "back" | "left" | "right" | "top",
              );
          }}
        >
          <option value="perspective">Camera view</option>
          <option value="front">Front</option>
          <option value="back">Back</option>
          <option value="left">Left</option>
          <option value="right">Right</option>
          <option value="top">Top</option>
        </select>
        <label>
          <input
            type="checkbox"
            checked={bones}
            onChange={(e) => setBones(e.target.checked)}
          />
          Rig overlay
        </label>
        <Button title="Pause walk" aria-label="Pause walk" className="toolbar-icon-action" variant="outline" onClick={() => p.setWalking(!p.walking)}>
          {p.walking ? <Pause /> : <Play />}
          <span className="toolbar-action-label">{p.walking ? "Pause walk" : "Play walk"}</span>
        </Button>
      </div>
      <div className="skin-rig-workspace">
        <aside className="studio-sidebar skin-rig-parts">
          <div className="skin-rig-tabs">
            <button
              aria-pressed={tab === "pose"}
              onClick={() => setTab("pose")}
            >
              Rig
            </button>
            <button
              aria-pressed={tab === "cosmetics"}
              onClick={() => setTab("cosmetics")}
            >
              Cosmetics
            </button>
          </div>
          <label className="skin-rig-search">
            <Search size={14} />
            <input
              aria-label="Find rig control"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Find a joint…"
            />
          </label>
          <div role="listbox" aria-label="Rig controls">
            {ordered
              .filter((part) =>
                part.label.toLowerCase().includes(query.toLowerCase()),
              )
              .map((part) => (
                <button
                  key={part.id}
                  role="option"
                  disabled={part.available === false}
                  aria-selected={selected === part.id}
                  onClick={() => {
                    setSelected(part.id);
                    setTab("pose");
                    if (part.kind === "target" || part.kind === "pole")
                      setMode("translate");
                  }}
                  className={`rig-part kind-${part.kind}`}
                >
                  <span className="rig-part-dot" />
                  {part.label}
                  <small>
                    {part.kind === "target"
                      ? "IK"
                      : part.kind === "pole"
                        ? "Pole"
                        : ""}
                  </small>
                </button>
              ))}
          </div>
        </aside>
        <div className="skin-rig-preview">
          <AssetViewport
            {...p.scene}
            ref={viewport}
            skin
            attachments={p.attachments}
            pose={p.pose}
            walking={p.walking}
            outer={p.scene.outer ?? true}
            showCape={p.scene.showCape ?? true}
            rotate={false}
            wireframe={wireframe}
            wireframeOptions={wireframeOptions}
            onWireframeStats={setWireframeStats}
            light={p.scene.light ?? 3}
            effects={p.effects && p.walking}
            onError={setError}
            rigEditor={{
              selected,
              mode,
              space,
              showSkeleton: bones,
              onSelect: (id) => {
                setSelected(id);
                setTab("pose");
              },
              onPose: (next) => {
                current.current = next;
                p.setWalking(false);
                p.setPose(next);
              },
              onBegin: beginEdit,
              onEnd: endEdit,
              onParts: setParts,
            }}
          />
          <div className="skin-rig-viewport-label">
            {selectedPart?.label ?? "Choose a control"}
            <span>
              {isTarget
                ? "Drag the target to bend the limb"
                : "Drag an axis or rotation ring"}
            </span>
          </div>
          <div className="skin-rig-viewport-help">
            Click a joint to select it. Drag outside the gizmo to orbit.
            Right-drag to pan. Scroll to zoom.
          </div>
          {error && (
            <FeedbackFlyout title="Skin rig">
              {error}
            </FeedbackFlyout>
          )}
        </div>
        <aside className="skin-rig-inspector">
          <section>
            <WireframeControls
              enabled={wireframe}
              settings={wireframeOptions}
              stats={wireframeStats}
              onToggle={setWireframe}
              onChange={setWireframeOptions}
            />
          </section>
          {tab === "cosmetics" ? (
            <>
              <section>
                <h3>Cosmetic bindings</h3>
                {cosmetics.length ? (
                  cosmetics.map((group) => bindingPanel(group, true))
                ) : (
                  <p>Add a cosmetic to configure its rig binding.</p>
                )}
              </section>
              {p.children}
            </>
          ) : (
            <>
              {cosmetics
                .filter((group) => selected === `cosmetic:${group.key}`)
                .map((group) => bindingPanel(group))}
              <section>
                <h3>{selectedPart?.label ?? "Character"}</h3>
                <p>
                  {isTarget
                    ? "Inverse kinematics moves the connected joints toward this target."
                    : "Rotate in the selected axes, or move the control in model pixels."}
                </p>
                <div className="skin-rig-axis-fields">
                  {["X", "Y", "Z"].map((axis, index) => (
                    <label key={axis} className={`axis-${axis.toLowerCase()}`}>
                      <span>{axis}</span>
                      <input
                        aria-label={`${axis} ${mode === "rotate" && !isTarget ? "rotation" : "position"}`}
                        type="number"
                        step={mode === "rotate" && !isTarget ? 1 : 0.25}
                        value={Math.round(values[index] * 100) / 100}
                        onChange={(e) => updateAxis(index, +e.target.value)}
                      />
                      <small>
                        {mode === "rotate" && !isTarget ? "°" : "px"}
                      </small>
                    </label>
                  ))}
                </div>
                {mode === "rotate" &&
                  !isTarget &&
                  ["X", "Y", "Z"].map((axis, index) => (
                    <label className="skin-rig-slider" key={axis}>
                      <span>{axis}</span>
                      <input
                        aria-label={`${axis} rotation slider`}
                        type="range"
                        min={
                          p.pose.limits !== false
                            ? (selectedPart?.limits[0][index] ?? -180)
                            : -180
                        }
                        max={
                          p.pose.limits !== false
                            ? (selectedPart?.limits[1][index] ?? 180)
                            : 180
                        }
                        value={angles[index]}
                        onPointerDown={() => {
                          dragStart.current = clone(current.current);
                        }}
                        onChange={(e) => {
                          const next: Vec3 = [...angles];
                          next[index] = +e.target.value;
                          p.setWalking(false);
                          p.setPose({
                            ...p.pose,
                            axes: { ...p.pose.axes, [selected]: next },
                          });
                        }}
                        onPointerUp={() => {
                          if (dragStart.current) {
                            const start = dragStart.current;
                            setUndo((old) => [...old.slice(-49), start]);
                            setRedo([]);
                            dragStart.current = undefined;
                          }
                        }}
                      />
                    </label>
                  ))}
                <Button
                  variant="outline"
                  onClick={() => {
                    const axes = { ...p.pose.axes },
                      positions = { ...p.pose.positions },
                      targets = { ...p.pose.targets },
                      poles = { ...p.pose.poles };
                    delete axes[selected];
                    delete positions[selected];
                    delete targets[selected];
                    delete poles[selected];
                    commit({ ...p.pose, axes, positions, targets, poles });
                  }}
                >
                  <RotateCcw />
                  Reset control
                </Button>
              </section>
              <section>
                <h3>Joint limits</h3>
                <label>
                  <input
                    type="checkbox"
                    checked={p.pose.limits !== false}
                    onChange={(e) =>
                      commit({ ...p.pose, limits: e.target.checked })
                    }
                  />
                  Keep natural movement ranges
                </label>
                <p>
                  Turn limits off for stylized poses. Imported bones keep their
                  original pivots.
                </p>
              </section>
              <section>
                <h3>Hand and foot targets</h3>
                <p>
                  Enable a target to position a hand or plant a foot. Move its
                  pole control to choose the bend direction.
                </p>
                {["leftHand", "rightHand", "leftFoot", "rightFoot"].map(
                  (id) => (
                    <label key={id}>
                      <input
                        type="checkbox"
                        checked={!!p.pose.targets?.[id]}
                        disabled={
                          parts.find((part) => part.id === id)?.available ===
                          false
                        }
                        onChange={(e) => enableIK(id, e.target.checked)}
                      />
                      {id.replace(/([A-Z])/g, " $1")}
                    </label>
                  ),
                )}
                <Button
                  variant="outline"
                  onClick={() => {
                    const targets = { ...p.pose.targets };
                    for (const id of ["leftFoot", "rightFoot"])
                      targets[id] = viewport.current?.rigPosition(id) ?? [
                        0, 0, 0,
                      ];
                    commit({ ...p.pose, targets });
                  }}
                >
                  Plant both feet
                </Button>
              </section>
              <section>
                <h3>Pose presets</h3>
                <div className="skin-controls-presets">
                  {Object.keys(skinPosePresets).map((name) => (
                    <Button
                      key={name}
                      variant="outline"
                      onClick={() => commit(clone(skinPosePresets[name]))}
                    >
                      {name}
                    </Button>
                  ))}
                </div>
                <Button
                  variant="outline"
                  onClick={() => commit(mirrorPose(p.pose))}
                >
                  <FlipHorizontal2 />
                  Mirror pose
                </Button>
                <Button
                  variant="outline"
                  onClick={() => commit({ ...neutralPose })}
                >
                  Reset whole rig
                </Button>
              </section>
              <section>
                <h3>Saved poses</h3>
                <div className="skin-rig-save">
                  <input
                    aria-label="Pose name"
                    value={presetName}
                    maxLength={40}
                    onChange={(e) => setPresetName(e.target.value)}
                  />
                  <Button
                    variant="outline"
                    aria-label="Save pose preset"
                    disabled={!presetName.trim()}
                    onClick={() => {
                      const next = Object.fromEntries([
                        ...Object.entries(saved)
                          .filter(([name]) => name !== presetName)
                          .slice(-19),
                        [presetName, clone(p.pose)],
                      ]);
                      setSaved(next);
                      try {
                        localStorage.setItem(
                          "craftbox:skin-rig-poses-v1",
                          JSON.stringify(next),
                        );
                      } catch {
                        setError("The pose could not be saved on this device.");
                      }
                    }}
                  >
                    <Save />
                  </Button>
                </div>
                {Object.entries(saved).map(([name, pose]) => (
                  <Button
                    key={name}
                    variant="ghost"
                    onClick={() => commit(clone(pose))}
                  >
                    {name}
                  </Button>
                ))}
              </section>
            </>
          )}
        </aside>
      </div>
      <div className="skin-rig-status">
        <span>
          {parts.filter((part) => part.kind === "joint").length} articulated
          joints
        </span>
        <span>Changes also appear in the main preview and model export.</span>
        <label>
          <input
            type="checkbox"
            checked={p.effects}
            onChange={(e) => p.setEffects(e.target.checked)}
          />
          Cosmetic effects during playback
        </label>
      </div>
    </div>
  );
}

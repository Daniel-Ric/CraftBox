import FeedbackFlyout from "./FeedbackFlyout";
import { type ReactNode, useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import StudioInput from "./StudioInput";
import { fileNameError } from "./input-rules";
import { modelFormats, type ModelFormat } from "./export-formats";
import { api } from "./bridge";
import { notifySaved } from "./StudioFlyouts";
import "./export-studio.css";
import ExportImagePreview from "./ExportImagePreview";

export function ExportStudio({
  open,
  onOpenChange,
  title,
  subtitle,
  formats,
  format,
  onFormat,
  children,
  overview,
  footer,
  busy = false,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  title: string;
  subtitle: string;
  formats: readonly { id: string; label: string; detail: string }[];
  format: string;
  onFormat: (s: string) => void;
  children: ReactNode;
  overview: ReactNode;
  footer: ReactNode;
  busy?: boolean;
}) {
  return (
    <Dialog
      lockWhileBusy={busy}
      open={open&&!busy}
      onOpenChange={(v) => {
        if (!busy) onOpenChange(v);
      }}
    >
      <DialogContent
        className="studio-window export-studio"
        showCloseButton={!busy}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{subtitle}</DialogDescription>
        </DialogHeader>
        <div className="export-studio-grid">
          <nav aria-label="Export format">
            {formats.map((f) => (
              <button
                key={f.id}
                disabled={busy}
                aria-pressed={format === f.id}
                onClick={() => onFormat(f.id)}
              >
                <strong>{f.label}</strong>
                <small>{f.detail}</small>
              </button>
            ))}
          </nav>
          <fieldset disabled={busy} className="export-studio-options">
            {children}
          </fieldset>
          <aside className="studio-sidebar export-studio-overview">{overview}</aside>
        </div>
        <DialogFooter>{footer}</DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
export function ExportFilename({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <label>
      File name
      <StudioInput
        value={value}
        validate={fileNameError}
        maxLength={120}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}
export function ModelExportDialog({
  open,
  onOpenChange,
  worldId,
  name,
  preview,
  bounds,
  onBusy,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  worldId: string;
  name: string;
  preview?: string;
  bounds?: import("./bridge").ChunkBounds;
  onBusy?: (busy: boolean) => void;
}) {
  const [format, setFormat] = useState<ModelFormat>("glb"),
    [filename, setFilename] = useState(name),
    [scale, setScale] = useState(1),
    [origin, setOrigin] = useState("center"),
    [upAxis, setUpAxis] = useState("y"),
    [textures, setTextures] = useState(true),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [progress, setProgress] = useState({
      running: false,
      stage: "",
      done: 0,
      total: 0,
    });
  useEffect(() => {
    onBusy?.(busy);
    return () => onBusy?.(false);
  }, [busy, onBusy]);
  useEffect(() => {
    if (open) {
      setFilename(name);
      setError("");
    }
  }, [open, name]);
  useEffect(() => {
    if (!busy) return;
    let active = true;
    const poll = () =>
      void api
        .GetModelExportProgress()
        .then((p) => {
          if (active) setProgress(p);
        })
        .catch(() => {});
    poll();
    const timer = setInterval(poll, 250);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [busy]);
  const selected = modelFormats.find((f) => f.id === format)!;
  const files =
    format === "gltf"
      ? ["scene.gltf", "scene.bin"]
      : format === "obj"
        ? [
            "scene.obj",
            "scene.mtl",
            ...(textures ? ["atlas.png", "opacity.png"] : []),
          ]
        : format === "fbx"
          ? ["scene.fbx", ...(textures ? ["atlas.png", "opacity.png"] : [])]
          : [];
  const valid =
    !fileNameError(filename) &&
    Number.isFinite(scale) &&
    scale >= 0.000001 &&
    scale <= 1000000;
  async function save() {
    setBusy(true);
    setError("");
    setProgress({ running: false, stage: "", done: 0, total: 0 });
    try {
      const path = await api.ExportModel({
        worldId,
        name: filename.trim(),
        format,
        scale,
        origin,
        upAxis: format === "glb" || format === "gltf" ? "y" : upAxis,
        textures: format !== "stl" && textures,
        ...(bounds ? { bounds } : {}),
      });
      if (path) {
        notifySaved(path);
        onOpenChange(false);
      }
    } catch (e) {
      if (!/context cancel|cancelled|canceled/i.test(String(e)))
        setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <ExportStudio
      open={open}
      onOpenChange={onOpenChange}
      title="Export 3D model"
      subtitle={name + " · Map geometry"}
      formats={modelFormats}
      format={format}
      onFormat={(f) => setFormat(f as ModelFormat)}
      busy={busy}
      overview={
        <>
          <h3>Output</h3>
          {preview && (
            <ExportImagePreview src={preview} alt="Model source preview" />
          )}
          <strong className="export-studio-file">
            {filename || name}
            {selected.extension}
          </strong>
          <p>{selected.use}</p>
          <dl>
            <div>
              <dt>Geometry</dt>
              <dd>Static triangle meshes</dd>
            </div>
            <div>
              <dt>Scope</dt>
              <dd>
                {bounds
                  ? `Selected chunks · X ${bounds.minX}…${bounds.maxX}, Z ${bounds.minZ}…${bounds.maxZ}`
                  : "Entire source"}
              </dd>
            </div>
            <div>
              <dt>Coordinates</dt>
              <dd>
                {origin === "center" ? "Centered" : "World origin"} ·{" "}
                {(format === "glb" || format === "gltf"
                  ? "y"
                  : upAxis
                ).toUpperCase()}{" "}
                up
              </dd>
            </div>
            <div>
              <dt>Scale</dt>
              <dd>{scale} unit(s) per block</dd>
            </div>
            <div>
              <dt>Texture images</dt>
              <dd>
                {format === "stl" || !textures
                  ? "Not included"
                  : format === "glb" || format === "gltf"
                    ? "Embedded in binary buffer"
                    : "PNG atlas + opacity map"}
              </dd>
            </div>
          </dl>
          {files.length > 0 && (
            <div>
              <h3>Package contents</h3>
              <ul className="export-studio-files">
                {files.map((file) => (
                  <li key={file}>{file}</li>
                ))}
              </ul>
            </div>
          )}
          {selected.extension.endsWith("zip") && (
            <p>
              Extract the ZIP, then import the model. Keep the files in the same
              folder.
            </p>
          )}
          <p>
            This exports the map geometry. Camera movement, lighting effects and
            post-processing stay in CraftBox.
          </p>
        </>
      }
      footer={
        <>
          <div className="export-studio-status" role="status">
            {error ? (
              <FeedbackFlyout title="Export">{error}</FeedbackFlyout>
            ) : busy ? (
              `${progress.stage || "Choose save location"}${progress.total ? ` · ${progress.done} / ${progress.total}` : ""}`
            ) : (
              "Choose a format and export settings."
            )}
            {busy && progress.total > 0 && (
              <progress value={progress.done} max={progress.total} />
            )}
          </div>
          <button
            className="studio-button"
            onClick={() =>
              busy ? void api.CancelGLBExport() : onOpenChange(false)
            }
          >
            {busy ? "Cancel export" : "Cancel"}
          </button>
          <button
            className="studio-button primary"
            disabled={busy || !valid}
            onClick={() => void save()}
          >
            {busy ? "Exporting …" : "Export …"}
          </button>
        </>
      }
    >
      <h3>Model settings</h3>
      <label>
        Workflow preset
        <select
          aria-label="Workflow preset"
          value=""
          onChange={(e) => {
            if (e.target.value === "blender") {
              setFormat("glb");
              setScale(1);
              setOrigin("center");
              setUpAxis("y");
              setTextures(true);
            } else if (e.target.value === "exchange") {
              setFormat("fbx");
              setScale(1);
              setOrigin("center");
              setUpAxis("z");
              setTextures(true);
            } else if (e.target.value === "geometry") {
              setFormat("stl");
              setScale(1);
              setOrigin("center");
              setUpAxis("z");
            }
          }}
        >
          <option value="">Custom settings</option>
          <option value="blender">Blender · Textured GLB</option>
          <option value="exchange">DCC exchange · FBX, Z up</option>
          <option value="geometry">Geometry / CAD · STL, Z up</option>
        </select>
      </label>
      <ExportFilename value={filename} onChange={setFilename} />
      <label>
        Origin
        <select value={origin} onChange={(e) => setOrigin(e.target.value)}>
          <option value="center">Center source bounds</option>
          <option value="world">Preserve world coordinates</option>
        </select>
      </label>
      <label>
        Scale · units per block
        <StudioInput
          type="number"
          min={0.000001}
          max={1000000}
          step="any"
          value={scale}
          onChange={(e) => setScale(+e.target.value)}
        />
      </label>
      <label>
        Up axis
        <select
          disabled={format === "glb" || format === "gltf"}
          value={format === "glb" || format === "gltf" ? "y" : upAxis}
          onChange={(e) => setUpAxis(e.target.value)}
        >
          <option value="y">Y up · glTF standard</option>
          <option value="z">Z up · Blender coordinates</option>
        </select>
      </label>
      <label className="export-studio-check">
        <StudioInput
          type="checkbox"
          checked={format !== "stl" && textures}
          disabled={format === "stl"}
          onChange={(e) => setTextures(e.target.checked)}
        />
        Include texture images
      </label>
      <p>
        GLB and glTF retain glTF material settings. OBJ and FBX use basic
        diffuse materials. At scale 1, one block equals one unit.
      </p>
    </ExportStudio>
  );
}

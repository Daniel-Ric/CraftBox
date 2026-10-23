import { useEffect, useState } from "react";
import { FolderOpen, FileArchive } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { renderDimensions } from "./render-settings";
import { exportDimensions } from "./cinematic-export";
export function ImportWorldDialog({
  open,
  onOpenChange,
  kind,
  onImport,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  kind: "archive" | "folder";
  onImport: (kind: "archive" | "folder") => void;
}) {
  const [source, setSource] = useState(kind);
  useEffect(() => setSource(kind), [kind, open]);
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="import-world-dialog">
        <DialogHeader>
          <DialogTitle>Import world</DialogTitle>
          <DialogDescription>
            Open a local Bedrock world for renders and camera sequences.
          </DialogDescription>
        </DialogHeader>
        <div className="import-body">
          <nav className="import-sources" aria-label="Import source">
            <button
              aria-pressed={source === "archive"}
              onClick={() => setSource("archive")}
            >
              <FileArchive />
              World file
            </button>
            <button
              aria-pressed={source === "folder"}
              onClick={() => setSource("folder")}
            >
              <FolderOpen />
              World folder
            </button>
          </nav>
          <section className="studio-dialog-body">
            <h2>
              {source === "archive"
                ? "Open Bedrock world file"
                : "Open local world folder"}
            </h2>
            <p>
              {source === "archive"
                ? "Import a world file exported from Minecraft. Included resources are used when building the scene."
                : "Select a Minecraft Bedrock world folder containing level.dat and its world database."}
            </p>
            <dl className="properties-list">
              <div>
                <dt>Source</dt>
                <dd>
                  {source === "archive"
                    ? ".mcworld file"
                    : "Bedrock world folder"}
                </dd>
              </div>
              <div>
                <dt>Workspaces</dt>
                <dd>Render · Cinematic</dd>
              </div>
            </dl>
            <p>
              Next, select{" "}
              {source === "archive" ? "the file" : "the folder"} on your computer. Once loaded, the world will be prepared for preview.
            </p>
          </section>
        </div>
        <DialogFooter>
          <button className="studio-button" onClick={() => onOpenChange(false)}>
            Cancel
          </button>
          <button
            className="studio-button primary"
            onClick={() => {
              onOpenChange(false);
              onImport(source);
            }}
          >
            Select …
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
export type MediaFormat = "png" | "gif" | "mp4" | "webm";
export function MediaExportDialog({
  open,
  onOpenChange,
  format,
  onFormat,
  edge,
  onEdge,
  aspect,
  name,
  fps = 24,
  duration = 0,
  gifFps = 12,
  onGifFps,
  transparent = false,
  onTransparent,
  onExport,
  preview,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  format: MediaFormat;
  onFormat?: (v: MediaFormat) => void;
  edge: number;
  onEdge: (v: number) => void;
  aspect: number;
  name: string;
  fps?: number;
  duration?: number;
  gifFps?: number;
  onGifFps?: (v: number) => void;
  transparent?: boolean;
  onTransparent?: (v: boolean) => void;
  onExport: (name: string) => void;
  preview?: string;
}) {
  const [filename, setFilename] = useState(name);
  useEffect(() => {
    if (open) setFilename(name);
  }, [open, name]);
  const size =
      format === "png"
        ? renderDimensions(edge, aspect)
        : exportDimensions(edge, aspect),
    video = format !== "png";
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="media-export-dialog">
        <DialogHeader>
          <DialogTitle>Export as</DialogTitle>
          <DialogDescription>
            {name} · {video ? "Camera sequence" : "Still render"}
          </DialogDescription>
        </DialogHeader>
        <div
          className={`media-export-body ${onFormat ? "has-format-navigation" : ""}`}
        >
          {onFormat && (
            <nav className="export-format-navigation" aria-label="Export format">
              {(["mp4", "webm", "gif"] as const).map((f) => (
                <button
                  key={f}
                  aria-pressed={format === f}
                  onClick={() => onFormat(f)}
                >
                  <strong>{f.toUpperCase()}</strong>
                  <small>
                    {f === "mp4"
                      ? "H.264 · Frame-accurate"
                      : f === "webm"
                        ? "Frame-accurate video"
                        : "Animated preview"}
                  </small>
                </button>
              ))}
            </nav>
          )}
          <section className="export-overview">
            <h3>{name} · Aktuelle Ansicht</h3>
            <div className="export-frame">
              {preview ? (
                <img src={preview} alt="Current render view" />
              ) : (
                <p>Preview is not available yet.</p>
              )}
            </div>
            <dl>
              <div>
                <dt>Dimensions</dt>
                <dd>{size.width + " × " + size.height + " pixels"}</dd>
              </div>
              {video && (
                <>
                  <div>
                    <dt>Duration</dt>
                    <dd>{duration} seconds</dd>
                  </div>
                  <div>
                    <dt>Frame rate</dt>
                    <dd>{format === "gif" ? gifFps : fps} fps</dd>
                  </div>
                </>
              )}
              <div>
                <dt>Color</dt>
                <dd>sRGB · 8 Bit</dd>
              </div>
            </dl>
          </section>
          <section className="document-properties">
            <h3>Export settings</h3>
            <label>
              File name
              <input
                value={filename}
                maxLength={120}
                onChange={(e) => setFilename(e.target.value)}
              />
            </label>
            <p className="export-filename">
              {filename.trim() || name}.{format}
            </p>
            {!onFormat ? (
              <label>
                File format
                <select aria-label="File format" value={format} disabled>
                  {[format].map((f) => (
                    <option key={f} value={f}>
                      {f === "mp4"
                        ? "MP4 · H.264"
                        : f === "webm"
                          ? "WebM · VP9 / VP8"
                          : f === "gif"
                            ? "GIF · Animation"
                            : "PNG · Lossless"}
                    </option>
                  ))}
                </select>
              </label>
            ) : (
              <dl className="properties-list export-format-detail">
                <div>
                  <dt>File format</dt>
                  <dd>
                    {format === "mp4"
                      ? "MP4 · H.264"
                      : format === "webm"
                        ? "WebM · VP9 / VP8"
                        : "GIF · Animation"}
                  </dd>
                </div>
              </dl>
            )}
            {
              <label>
                Longest edge
                <select
                  aria-label="Longest edge"
                  value={edge}
                  onChange={(e) => onEdge(+e.target.value)}
                >
                  {!(
                    format === "gif"
                      ? [320, 480, 640, 960]
                      : [720, 1080, 1920, 2560, 3840, 4096]
                  ).includes(edge) && (
                    <option value={edge}>
                      {edge} Pixels · Custom
                    </option>
                  )}
                  {(format === "gif"
                    ? [320, 480, 640, 960]
                    : [720, 1080, 1920, 2560, 3840, 4096]
                  ).map((e) => (
                    <option key={e} value={e}>
                      {e} pixels
                    </option>
                  ))}
                </select>
              </label>
            }
            {format === "gif" && (
              <label>
                GIF frame rate
                <select
                  aria-label="GIF frame rate"
                  value={gifFps}
                  onChange={(e) => onGifFps?.(+e.target.value)}
                >
                  {[8, 12, 15, 24].map((f) => (
                    <option key={f} value={f}>
                      {f} fps
                    </option>
                  ))}
                </select>
              </label>
            )}
            {format === "png" && (
              <label className="export-checkbox">
                <input
                  type="checkbox"
                  checked={transparent}
                  onChange={(e) => onTransparent?.(e.target.checked)}
                />
                Transparent background
              </label>
            )}
            <p>
              {format === "png"
                ? "A lossless still image with your lighting and color settings."
                : format === "mp4"
                  ? "H.264 video with frame-accurate output."
                  : format === "gif"
                    ? "Animated preview with an optimized color palette."
                    : "VP9/VP8 video with frame-accurate output."}{" "}
              Choose where to save the file in the next step.
            </p>
          </section>
        </div>
        <DialogFooter>
          <button className="studio-button" onClick={() => onOpenChange(false)}>
            Cancel
          </button>
          <button
            className="studio-button primary"
            onClick={() => {
              onOpenChange(false);
              onExport(filename.trim() || name);
            }}
          >
            Export …
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

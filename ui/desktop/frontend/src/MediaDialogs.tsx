import StudioInput from "@/StudioInput";
import { fileNameError } from "./input-rules";
import { useEffect, useState, type ReactNode } from "react";
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
import { ExportStudio, ExportFilename } from "./ExportStudio";
import ExportImagePreview from "./ExportImagePreview";
import {
  imageFormats,
  videoFormats,
  isImageFormat,
  type ExportOptions,
} from "./export-formats";
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
          <DialogDescription>Import a Bedrock world.</DialogDescription>
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
                ? "Bedrock world archive, including resources."
                : "Bedrock folder with level.dat and db."}
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
              Next, select {source === "archive" ? "the file" : "the folder"}{" "}
              from your computer.
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
export type MediaFormat = ExportOptions["format"];
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
  dimensions,
  children,
  busy = false,
  status,
  onCancel,
  title,
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
  onGifFps?: (n: number) => void;
  transparent?: boolean;
  onTransparent?: (v: boolean) => void;
  onExport: (name: string, options: ExportOptions) => void;
  preview?: string;
  dimensions?: { width: number; height: number };
  children?: ReactNode;
  busy?: boolean;
  status?: string;
  onCancel?: () => void;
  title?: string;
}) {
  const [filename, setFilename] = useState(name),
    [quality, setQuality] = useState(90),
    [background, setBackground] = useState("#ffffff"),
    [frameRate, setFrameRate] = useState(fps),
    [bitrate, setBitrate] = useState(12),
    [audio, setAudio] = useState(true),
    [start, setStart] = useState(0),
    [end, setEnd] = useState(duration);
  useEffect(() => {
    if (open) {
      setFilename(name);
      setEnd(duration);
      setStart(0);
      setFrameRate(fps);
    }
  }, [open, name, duration, fps]);
  const image = isImageFormat(format),
    formats = image ? imageFormats : videoFormats,
    selected = formats.find((f) => f.id === format)!,
    alpha = image && imageFormats.find((f) => f.id === format)!.alpha;
  const size =
    dimensions ||
    (image ? renderDimensions(edge, aspect) : exportDimensions(edge, aspect));
  const valid =
    !fileNameError(filename) &&
    size.width > 0 &&
    size.height > 0 &&
    (image ||
      (Number.isFinite(start) &&
        Number.isFinite(end) &&
        start >= 0 &&
        end > start &&
        end <= duration));
  const extension = format === "jpeg" ? "jpg" : format;
  return (
    <ExportStudio
      open={open}
      onOpenChange={onOpenChange}
      title={title || (image ? "Export image" : "Export cinematic")}
      subtitle={name + " · " + (image ? "Current view" : "Camera sequence")}
      formats={formats}
      format={format}
      onFormat={(f) => onFormat?.(f as MediaFormat)}
      busy={busy}
      overview={
        <>
          <h3>Output preview</h3>
          {preview ? (
            <ExportImagePreview src={preview} alt="Current render view" />
          ) : (
            <div className="export-studio-preview">
              <p>
                {image
                  ? "The current view will be saved at this size."
                  : "The export follows your camera sequence."}
              </p>
            </div>
          )}
          <strong className="export-studio-file">
            {filename.trim() || name}.{extension}
          </strong>
          <dl>
            <div>
              <dt>Dimensions</dt>
              <dd>{size.width + " × " + size.height + " pixels"}</dd>
            </div>
            <div>
              <dt>Color</dt>
              <dd>
                sRGB · 8 bit
                {image ? (alpha && transparent ? " · Alpha" : " · Opaque") : ""}
              </dd>
            </div>
            {!image && (
              <>
                <div>
                  <dt>Duration</dt>
                  <dd>{Math.max(0, end - start).toFixed(2)} seconds</dd>
                </div>
                <div>
                  <dt>Frame rate</dt>
                  <dd>
                    {format === "gif" ? gifFps : frameRate} fps ·{" "}
                    {Math.ceil(
                      Math.max(0, end - start) *
                        (format === "gif" ? gifFps : frameRate),
                    )}{" "}
                    frames
                  </dd>
                </div>
                <div>
                  <dt>Audio</dt>
                  <dd>
                    {format === "gif"
                      ? "Not supported"
                      : audio
                        ? "Enabled tracks"
                        : "Muted"}
                  </dd>
                </div>
              </>
            )}
          </dl>
          <p>{selected.use}</p>
          {image && (format === "jpeg" || format === "bmp") && (
            <p>
              Transparency is composited over the selected background color.
            </p>
          )}
          {!image && (
            <p>
              Available encoders are checked before rendering. MOV uses H.264, not ProRes.
            </p>
          )}
        </>
      }
      footer={
        <>
          <div className="export-studio-status" role="status">
            {status ||
              (busy ? "Preparing export …" : "Choose where to save the file.")}
          </div>
          <button
            className="studio-button"
            onClick={() => (busy ? onCancel?.() : onOpenChange(false))}
          >
            {busy ? "Cancel export" : "Cancel"}
          </button>
          <button
            className="studio-button primary"
            disabled={busy || !valid}
            onClick={() => {
              onOpenChange(false);
              onExport(filename.trim() || name, {
                format,
                quality,
                background,
                fps: format === "gif" ? gifFps : frameRate,
                bitrate: bitrate * 1000000,
                includeAudio: audio,
                start,
                end,
              });
            }}
          >
            Export …
          </button>
        </>
      }
    >
      <h3>{image ? "Image settings" : "Sequence settings"}</h3>
      <ExportFilename value={filename} onChange={setFilename} />
      {!dimensions && (
        <label>
          Longest edge
          <select
            aria-label="Longest edge"
            value={edge}
            onChange={(e) => onEdge(+e.target.value)}
          >
            {[
              ...new Set([
                edge,
                ...(format === "gif"
                  ? [320, 480, 640, 960]
                  : [720, 1080, 1920, 2560, 3840, 4096]),
              ]),
            ]
              .sort((a, b) => a - b)
              .map((n) => (
                <option value={n} key={n}>
                  {n} pixels
                </option>
              ))}
          </select>
        </label>
      )}
      {children}
      {image ? (
        <>
          <label className="export-studio-check">
            <StudioInput
              type="checkbox"
              checked={transparent}
              disabled={!onTransparent}
              onChange={(e) => onTransparent?.(e.target.checked)}
            />
            {alpha ? "Transparent background" : "Use matte background"}
          </label>
          {(format === "jpeg" || format === "webp") && (
            <label>
              Quality · {quality}%
              <StudioInput
                type="range"
                min={1}
                max={100}
                value={quality}
                onChange={(e) => setQuality(+e.target.value)}
              />
            </label>
          )}
          {!alpha && onTransparent && (
            <label>
              Background color
              <StudioInput
                type="color"
                value={background}
                disabled={!transparent}
                onChange={(e) => setBackground(e.target.value)}
              />
            </label>
          )}
          {!alpha && onTransparent && (
            <p>
              Turn on the matte background to use this color. Turn it off to keep the scene background.
            </p>
          )}
          <p>
            {format === "png" || format === "tiff"
              ? "Lossless compression preserves the rendered pixels."
              : format === "bmp"
                ? "Uncompressed output creates larger files."
                : "Higher quality increases detail and file size."}
          </p>
        </>
      ) : (
        <>
          <div className="export-studio-row">
            <label>
              Start · seconds
              <StudioInput
                type="number"
                min={0}
                max={duration}
                step={0.01}
                value={start}
                onChange={(e) => setStart(+e.target.value)}
              />
            </label>
            <label>
              End · seconds
              <StudioInput
                type="number"
                min={0}
                max={duration}
                step={0.01}
                value={end}
                onChange={(e) => setEnd(+e.target.value)}
              />
            </label>
          </div>
          <label>
            {format === "gif" ? "GIF frame rate" : "Frame rate"}
            <select
              aria-label={format === "gif" ? "GIF frame rate" : "Frame rate"}
              value={format === "gif" ? gifFps : frameRate}
              onChange={(e) =>
                format === "gif"
                  ? onGifFps?.(+e.target.value)
                  : setFrameRate(+e.target.value)
              }
            >
              {[
                ...new Set([
                  format === "gif" ? gifFps : frameRate,
                  ...(format === "gif"
                    ? [8, 12, 15, 24]
                    : [24, 25, 30, 48, 50, 60]),
                ]),
              ]
                .sort((a, b) => a - b)
                .map((n) => (
                  <option key={n} value={n}>
                    {n} fps
                  </option>
                ))}
            </select>
          </label>
          {format !== "gif" && (
            <>
              <label>
                Video bitrate
                <select
                  value={bitrate}
                  onChange={(e) => setBitrate(+e.target.value)}
                >
                  {[4, 8, 12, 20, 40].map((n) => (
                    <option key={n} value={n}>
                      {n} Mbps
                    </option>
                  ))}
                </select>
              </label>
              <label className="export-studio-check">
                <StudioInput
                  type="checkbox"
                  checked={audio}
                  onChange={(e) => setAudio(e.target.checked)}
                />
                Include enabled audio tracks
              </label>
            </>
          )}
          <p>
            The export follows the timeline frame by frame. For files over 190 MiB, shorten the range or lower the resolution or bitrate.
          </p>
        </>
      )}
    </ExportStudio>
  );
}

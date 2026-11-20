import { formatLocale } from "./i18n";
import { useEffect, useState } from "react";
import {
  Download,
  FolderOpen,
  LoaderCircle,
  Play,
  SlidersHorizontal,
} from "lucide-react";
import { api, type World } from "./bridge";
import { RenderIcon, CinematicIcon } from "./studio-icons";

type Props = {
  world: World;
  parentName?: string;
  onOpenEditor: (world: World, page: "render" | "cinematic") => void;
  onFolder: (path: string) => void;
  onWorldSettings?: (world: World) => void;
  onWorldCopy?: (world: World) => void;
  onImportWorld?: (world: World) => Promise<void>;
};
type FileDetails = Awaited<ReturnType<typeof api.GetWorldFileDetails>>;
const coordinates = (values: number[]) =>
  values.map((value) => Math.round(value * 100) / 100).join(", ");
const date = (value: string) => new Date(value).toLocaleString(formatLocale());

export default function WorldDetails(p: Props) {
  const { world } = p;
  const saved = world.state === "finished";
  const [file, setFile] = useState<FileDetails>();
  const [fileError, setFileError] = useState("");
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState("");
  const [opened, setOpened] = useState(false);
  useEffect(() => {
    setFile(undefined);
    setFileError("");
    setOpened(false);
    setImportError("");
    if (!saved || !world.path) return;
    let active = true;
    api
      .GetWorldFileDetails(world.id)
      .then((value) => {
        if (active) setFile(value);
      })
      .catch((error) => {
        if (active)
          setFileError(error instanceof Error ? error.message : String(error));
      });
    return () => {
      active = false;
    };
  }, [world.id, world.path, saved]);
  async function importWorld() {
    if (importing || !p.onImportWorld) return;
    setImporting(true);
    setImportError("");
    setOpened(false);
    try {
      await p.onImportWorld(world);
      setOpened(true);
    } catch (error) {
      setImportError(error instanceof Error ? error.message : String(error));
    } finally {
      setImporting(false);
    }
  }
  return (
    <>
      <div className="world-detail-actions">
        {p.onImportWorld && (
          <button
            className="studio-button primary"
            disabled={!saved || !world.path || importing}
            onClick={() => void importWorld()}
          >
            {importing ? (
              <LoaderCircle size={14} className="spin" />
            ) : (
              <Play size={14} />
            )}
            Import into Minecraft
          </button>
        )}
        <button
          className="studio-button"
          disabled={!world.chunks}
          onClick={() => p.onOpenEditor(world, "render")}
        >
          <RenderIcon size={14} />
          Create render
        </button>
        <button
          className="studio-button"
          disabled={!world.chunks}
          onClick={() => p.onOpenEditor(world, "cinematic")}
        >
          <CinematicIcon size={14} />
          Create camera sequence
        </button>
        {p.onWorldSettings && (
          <button
            className="studio-button"
            onClick={() => p.onWorldSettings?.(world)}
          >
            <SlidersHorizontal size={14} />
            Export settings
          </button>
        )}
        {p.onWorldCopy && saved && (
          <button
            className="studio-button"
            onClick={() => p.onWorldCopy?.(world)}
          >
            <Download size={14} />
            Export copy
          </button>
        )}
        {world.path && (
          <button
            className="studio-button"
            onClick={() => p.onFolder(world.path)}
          >
            <FolderOpen size={14} />
            Open folder
          </button>
        )}
      </div>
      {importError && (
        <p role="alert" className="world-details-error">
          {importError}
        </p>
      )}
      {opened && (
        <p role="status" className="world-details-empty">
          Minecraft import opened.
        </p>
      )}
      {!saved && (
        <p className="world-details-empty">
          Import becomes available once the world is saved.
        </p>
      )}
      <dl className="world-metadata">
        <div>
          <dt>Status</dt>
          <dd>{saved ? "Saved" : world.state || "Capturing"}</dd>
        </div>
        <div>
          <dt>Chunks</dt>
          <dd>{world.chunks.toLocaleString(formatLocale())}</dd>
        </div>
        <div>
          <dt>Entities</dt>
          <dd>{world.entities.toLocaleString(formatLocale())}</dd>
        </div>
        <div>
          <dt>Captured area</dt>
          <dd>{(world.chunks * 256).toLocaleString(formatLocale())} blocks²</dd>
        </div>
        {world.server && (
          <div>
            <dt>Server</dt>
            <dd>{world.server}</dd>
          </div>
        )}
        {world.startedAt && (
          <div>
            <dt>Capture started</dt>
            <dd>{date(world.startedAt)}</dd>
          </div>
        )}
        {world.heightRange && world.heightRange[1] > world.heightRange[0] && (
          <div>
            <dt>Build height</dt>
            <dd>
              {world.heightRange[0]} to {world.heightRange[1]} ·{" "}
              {world.heightRange[1] - world.heightRange[0] + 1} blocks
            </dd>
          </div>
        )}
        {world.spawn && (
          <div>
            <dt>Spawn point</dt>
            <dd>{coordinates(world.spawn)}</dd>
          </div>
        )}
        {world.hasPlayer && (
          <div>
            <dt>Player position</dt>
            <dd>{coordinates(world.player)}</dd>
          </div>
        )}
        {p.parentName && (
          <div>
            <dt>Source world</dt>
            <dd>{p.parentName}</dd>
          </div>
        )}
        {file && (
          <>
            <div>
              <dt>World file size</dt>
              <dd>{(file.bytes / 1048576).toFixed(1)} MB</dd>
            </div>
            <div>
              <dt>Last saved</dt>
              <dd>{date(file.modified)}</dd>
            </div>
            <div>
              <dt>Resource packs</dt>
              <dd>{file.resourcePacks}</dd>
            </div>
            <div>
              <dt>Behavior packs</dt>
              <dd>{file.behaviorPacks}</dd>
            </div>
          </>
        )}
        <div>
          <dt>Capture points</dt>
          <dd>{world.capturePoints?.length || 0}</dd>
        </div>
      </dl>
      {!!world.capturePoints?.length && (
        <ul className="world-detail-points">
          {world.capturePoints.slice(0, 6).map((point) => (
            <li key={point.id}>
              <strong>{point.name || point.kind}</strong>
              <span>{coordinates(point.position)}</span>
            </li>
          ))}
          {world.capturePoints.length > 6 && (
            <li>{world.capturePoints.length - 6} more points</li>
          )}
        </ul>
      )}
      {world.path && (
        <div className="world-detail-file">
          <span>World file</span>
          <p>{world.path}</p>
        </div>
      )}
      {fileError && (
        <p className="world-details-empty">
          File details unavailable: {fileError}
        </p>
      )}
      {!world.chunks && !saved && (
        <p className="world-details-empty">
          Terrain counts update as the capture is saved.
        </p>
      )}
      <button
        className="world-packet-link"
        onClick={() => {
          sessionStorage.setItem(
            "craftbox:packet-navigation",
            JSON.stringify({ search: "name:LevelChunk" }),
          );
          window.dispatchEvent(new CustomEvent("craftbox:open-packets"));
        }}
      >
        Inspect chunk packets
      </button>
    </>
  );
}

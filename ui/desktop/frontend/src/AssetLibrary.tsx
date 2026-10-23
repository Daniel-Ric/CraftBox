import { useState } from "react";
import {
  Globe2,
  Package,
  Users,
  Search,
  LayoutGrid,
  List,
  FolderOpen,
  Maximize2,
  Aperture,
  Clapperboard,
  ImageDown,
} from "lucide-react";
import type { World, Pack, Skin } from "./bridge";
const size = (bytes: number) =>
  bytes >= 1048576
    ? `${(bytes / 1048576).toFixed(1)} MB`
    : `${Math.round(bytes / 1024)} KB`;
export default function AssetLibrary(p: {
  page: string;
  worlds: World[];
  packs: Pack[];
  skins: Skin[];
  onOpenEditor: (w: World, page: "render" | "cinematic") => void;
  onFolder: (path: string) => void;
  onMap: () => void;
  onSaveSkin: (skin: Skin) => void;
  onSource: () => void;
}) {
  const [query, setQuery] = useState(""),
    [view, setView] = useState<"grid" | "list">(
      p.page === "skins" ? "grid" : "list",
    ),
    [selected, setSelected] = useState(""),
    [skinSize, setSkinSize] = useState("");
  const world = p.worlds.find((w) => w.id === selected),
    pack = p.packs.find((w) => w.id === selected),
    skin = p.skins.find((_, i) => String(i) === selected);
  const current =
    p.page === "worlds" ? world : p.page === "packs" ? pack : skin;
  const Icon =
    p.page === "worlds" ? Globe2 : p.page === "packs" ? Package : Users;
  const title =
    p.page === "worlds"
      ? "World overview"
      : p.page === "packs"
        ? "Pack library"
        : "Skin gallery";
  const items =
    p.page === "worlds"
      ? p.worlds.map((w) => ({
          id: w.id,
          name: w.name,
          detail: `${w.chunks || 0} Chunks · ${w.entities || 0} Entities`,
          status: w.state === "finished" ? "Saved" : w.state,
          image: "",
          path: w.path,
        }))
      : p.page === "packs"
        ? p.packs.map((w) => ({
            id: w.id,
            name: w.name || "Resource pack",
            detail: `${w.version || "Version not specified"} · ${size(w.size || 0)}`,
            status:
              w.error ||
              (w.done
                ? "Saved"
                : `${size(w.received || 0)} / ${size(w.size || 0)}`),
            image: w.icon,
            path: w.path,
          }))
        : p.skins.map((w, i) => ({
            id: String(i),
            name: w.name,
            detail: "Player skin",
            status: "PNG texture",
            image: w.image,
            path: "",
          }));
  const filtered = items.filter((x) =>
    `${x.name} ${x.detail} ${x.status}`
      .toLocaleLowerCase()
      .includes(query.toLocaleLowerCase()),
  );
  return (
    <section className={`asset-library library-${p.page}`} aria-label={title}>
      <header className="library-heading">
        <Icon size={15} />
        <h2>{title}</h2>
        <span>
          {items.length}{" "}
          {p.page === "worlds"
            ? "Worlds"
            : p.page === "packs"
              ? "Packs"
              : "Skins"}
        </span>
        {p.page === "worlds" && (
          <button
            className="studio-button"
            aria-label="Open live map"
            onClick={p.onMap}
          >
            <Maximize2 size={14} />
          </button>
        )}
      </header>
      <div className="library-toolbar">
        <label>
          <Search size={14} />
          <input
            aria-label="Filter results"
            placeholder={
              p.page === "worlds"
                ? "Search worlds"
                : p.page === "packs"
                  ? "Search packs"
                  : "Search players"
            }
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <div role="group" aria-label="Library view">
          <button
            aria-label="List view"
            aria-pressed={view === "list"}
            onClick={() => setView("list")}
          >
            <List size={15} />
          </button>
          <button
            aria-label="Grid view"
            aria-pressed={view === "grid"}
            onClick={() => setView("grid")}
          >
            <LayoutGrid size={15} />
          </button>
        </div>
      </div>
      <div className="library-content">
        <div
          className={`library-items view-${p.page === "skins" ? "grid" : view}`}
        >
          {filtered.map((x) => (
            <button
              key={x.id}
              aria-label={x.name}
              aria-pressed={selected === x.id}
              onClick={() => {
                if (selected !== x.id) setSkinSize("");
                setSelected(x.id);
              }}
              onDoubleClick={() => {
                if (p.page === "worlds") {
                  const w = p.worlds.find((w) => w.id === x.id);
                  if (w) p.onOpenEditor(w, "render");
                }
              }}
            >
              <div className="library-thumbnail">
                {x.image ? (
                  <img src={x.image} alt="" />
                ) : (
                  <Icon size={p.page === "skins" ? 32 : 24} />
                )}
              </div>
              <div className="library-item-name">
                <strong>{x.name}</strong>
                <small>{x.detail}</small>
                {p.page === "packs" &&
                  (() => {
                    const pack = p.packs.find((p) => p.id === x.id);
                    return pack && !pack.done && !pack.error ? (
                      <progress
                        aria-label={`Download ${pack.name}`}
                        max={100}
                        value={
                          pack.size
                            ? Math.min(100, (pack.received / pack.size) * 100)
                            : 0
                        }
                      />
                    ) : null;
                  })()}
              </div>
              <span className="library-item-status">{x.status}</span>
            </button>
          ))}
          {!filtered.length && (
            <div className="library-empty">
              <Icon size={38} />
              <h3>
                {query
                  ? "No matching results"
                  : p.page === "worlds"
                    ? "Your worlds at a glance"
                    : p.page === "packs"
                      ? "No resource packs yet"
                      : "No player skins yet"}
              </h3>
              <p>
                {query
                  ? "Change your search to see more results."
                  : p.page === "worlds"
                    ? "Start a capture on the right or open a local world in the Render editor."
                    : p.page === "packs"
                      ? "Resource packs received from the server appear here with their status and version."
                      : "Skins of visible players appear in this gallery during the session."}
              </p>
              {!query && (
                <button className="studio-button" onClick={p.onSource}>
                  Select source
                </button>
              )}
            </div>
          )}
        </div>
        <aside
          className="library-properties"
          aria-label="Selection properties"
        >
          <h3>Properties</h3>
          {current ? (
            <>
              <div
                className={
                  "library-detail-preview " +
                  (p.page === "skins" ? "skin-preview" : "")
                }
              >
                {p.page === "skins" && skin?.image ? (
                  <img
                    src={skin.image}
                    alt={`Skin of ${skin.name}`}
                    onLoad={(e) =>
                      setSkinSize(
                        `${e.currentTarget.naturalWidth} × ${e.currentTarget.naturalHeight} pixels`,
                      )
                    }
                  />
                ) : p.page === "packs" && pack?.icon ? (
                  <img src={pack.icon} alt="" />
                ) : (
                  <Icon size={44} />
                )}
              </div>
              <h4>{current.name}</h4>
              <dl className="properties-list">
                {p.page === "worlds" && world && (
                  <>
                    <div>
                      <dt>Terrain</dt>
                      <dd>{world.chunks || 0} Chunks</dd>
                    </div>
                    <div>
                      <dt>Entities</dt>
                      <dd>{world.entities || 0}</dd>
                    </div>
                    <div>
                      <dt>Status</dt>
                      <dd>
                        {world.state === "finished"
                          ? "Saved"
                          : world.state}
                      </dd>
                    </div>
                    {world.spawn && (
                      <div>
                        <dt>Spawn point</dt>
                        <dd>{world.spawn.join(", ")}</dd>
                      </div>
                    )}
                  </>
                )}
                {p.page === "packs" && pack && (
                  <>
                    <div>
                      <dt>Version</dt>
                      <dd>{pack.version || "—"}</dd>
                    </div>
                    <div>
                      <dt>Size</dt>
                      <dd>{size(pack.size || 0)}</dd>
                    </div>
                    <div>
                      <dt>Status</dt>
                      <dd>
                        {pack.error ||
                          (pack.done ? "Saved" : "Receiving")}
                      </dd>
                    </div>
                  </>
                )}
                {p.page === "skins" && skin && (
                  <>
                    <div>
                      <dt>File type</dt>
                      <dd>PNG texture</dd>
                    </div>
                    {skinSize && (
                      <div>
                        <dt>Dimensions</dt>
                        <dd>{skinSize}</dd>
                      </div>
                    )}
                  </>
                )}
              </dl>
              {p.page === "worlds" && world && (
                <div className="library-property-actions">
                  <button
                    className="studio-button primary"
                    disabled={!world.chunks}
                    onClick={() => p.onOpenEditor(world, "render")}
                  >
                    <Aperture size={14} />
                    Create render
                  </button>
                  <button
                    className="studio-button"
                    disabled={!world.chunks}
                    onClick={() => p.onOpenEditor(world, "cinematic")}
                  >
                    <Clapperboard size={14} />
                    Create camera sequence
                  </button>
                </div>
              )}
              {p.page === "skins" && skin && (
                <button
                  className="studio-button"
                  disabled={!skin.image}
                  onClick={() => p.onSaveSkin(skin)}
                >
                  <ImageDown size={14} />
                  Save skin as PNG
                </button>
              )}
              {p.page !== "skins" &&
                (p.page === "worlds" ? world?.path : pack?.path) && (
                  <>
                    <p className="library-file-path">
                      {p.page === "worlds" ? world?.path : pack?.path}
                    </p>
                    <button
                      className="studio-button"
                      aria-label={
                        p.page === "packs"
                          ? "Open pack folder"
                          : `Open folder for ${world?.name}`
                      }
                      onClick={() =>
                        p.onFolder(
                          (p.page === "worlds" ? world?.path : pack?.path)!,
                        )
                      }
                    >
                      <FolderOpen size={14} />
                      Open folder
                    </button>
                  </>
                )}
            </>
          ) : (
            <div className="library-no-selection">
              <Icon size={26} />
              <p>
                Select{" "}
                {p.page === "worlds"
                  ? "a world"
                  : p.page === "packs"
                    ? "a resource pack"
                    : "a skin"}
                {" "}to view details and actions.
              </p>
            </div>
          )}
        </aside>
      </div>
    </section>
  );
}

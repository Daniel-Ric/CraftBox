import { useState, type ComponentProps } from "react";
import {
  Aperture,
  Clapperboard,
  Globe2,
  FolderOpen,
  Plus,
  FileImage,
  Film,
  Search,
  ArrowUpRight,
  Package,
  Users,
  ArrowLeftRight,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import type { World } from "./bridge";
import { projectDefaults, type StudioProject } from "./start-project";
import { renderPresetLabels } from "./render-settings";
import { lookCatalog } from "./look-catalog";
import { lookCategories } from "./scene-looks";
import StudioBrand from "./StudioBrand";
const modes = [
  {
    id: "render",
    name: "Still render",
    icon: Aperture,
    description: "A still image with lighting, shadows and color grading.",
  },
  {
    id: "cinematic",
    name: "Camera sequence",
    icon: Clapperboard,
    description: "An animated camera sequence with a timeline and keyframes.",
  },
  {
    id: "worlds",
    name: "Capture",
    icon: Globe2,
    description: "Capture worlds, resource packs or player skins.",
  },
] as const;
const formats = [
  {
    id: "landscape",
    name: "Widescreen",
    detail: "1920 × 1080",
    edge: 1920,
    aspect: 16 / 9,
  },
  {
    id: "ultra",
    name: "Ultra HD",
    detail: "3840 × 2160",
    edge: 3840,
    aspect: 16 / 9,
  },
  {
    id: "cinema",
    name: "CinemaScope",
    detail: "2560 × 1071",
    edge: 2560,
    aspect: 2.39,
  },
  {
    id: "square",
    name: "Square",
    detail: "2048 × 2048",
    edge: 2048,
    aspect: 1,
  },
  {
    id: "vertical",
    name: "Vertical",
    detail: "1080 × 1920",
    edge: 1920,
    aspect: 9 / 16,
  },
  {
    id: "portrait",
    name: "Portrait",
    detail: "1600 × 2000",
    edge: 2000,
    aspect: 4 / 5,
  },
];
type Shared = { worlds: World[] };
export function NewProjectDialog({
  open,
  onOpenChange,
  worlds,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onCreate: (p: StudioProject) => void;
} & Shared) {
  const [mode, setMode] = useState<"render" | "cinematic" | "worlds">("render"),
    [format, setFormat] = useState("landscape");
  const [width, setWidth] = useState(1920),
    [height, setHeight] = useState(1080);
  function chooseFormat(id: string) {
    const p = formats.find((f) => f.id === id)!;
    setFormat(id);
    setWidth(Math.round(p.aspect >= 1 ? p.edge : p.edge * p.aspect));
    setHeight(Math.round(p.aspect >= 1 ? p.edge / p.aspect : p.edge));
  }
  const [name, setName] = useState("Untitled"),
    [look, setLook] = useState("island"),
    [duration, setDuration] = useState(12),
    [fps, setFps] = useState(24),
    [worldId, setWorldId] = useState(""),
    [capture, setCapture] = useState<"worlds" | "packs" | "skins">("worlds"),
    [address, setAddress] = useState("");
  const preset = formats.find((f) => f.id === format);
  const validSize =
    Number.isInteger(width) &&
    Number.isInteger(height) &&
    width >= 64 &&
    height >= 64 &&
    width <= 4096 &&
    height <= 4096 &&
    width / height >= 0.25 &&
    width / height <= 4;
  function create() {
    onCreate({
      ...projectDefaults,
      id: crypto.randomUUID(),
      name: name.trim() || "Untitled",
      page: mode === "worlds" ? capture : mode,
      worldId: worldId === "archive" || worldId === "folder" ? "" : worldId,
      openKind:
        worldId === "archive" || worldId === "folder" ? worldId : undefined,
      edge: Math.max(width, height),
      aspect: width / height,
      look,
      duration: Math.min(600, Math.max(1, duration || 12)),
      fps,
      address: address.trim(),
      updated: Date.now(),
    });
    onOpenChange(false);
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="new-project-dialog">
        <DialogHeader>
          <DialogTitle>New project</DialogTitle>
          <DialogDescription>
            Set the format and workspace for your next project.
          </DialogDescription>
        </DialogHeader>
        <div
          className="new-project-tabs"
          role="tablist"
          aria-label="Project type"
        >
          {modes.map((m) => (
            <button
              role="tab"
              aria-selected={mode === m.id}
              key={m.id}
              onClick={() => setMode(m.id)}
            >
              <m.icon size={15} />
              {m.name}
            </button>
          ))}
        </div>
        <div className="new-project-body">
          <section
            className="document-presets"
            role="tabpanel"
            aria-label={modes.find((m) => m.id === mode)!.name}
          >
            <h3>{mode === "worlds" ? "Capture type" : "Format presets"}</h3>
            <p>{modes.find((m) => m.id === mode)!.description}</p>
            {mode === "worlds" ? (
              <div className="capture-presets">
                {[
                  {
                    id: "worlds",
                    name: "World & builds",
                    description: "Save terrain, blocks and entities.",
                    icon: Globe2,
                  },
                  {
                    id: "packs",
                    name: "Resource packs",
                    description: "Save textures and server resources.",
                    icon: Package,
                  },
                  {
                    id: "skins",
                    name: "Player skins",
                    description: "Collect skins from a session.",
                    icon: Users,
                  },
                ].map((c) => (
                  <button
                    key={c.id}
                    aria-pressed={capture === c.id}
                    onClick={() => setCapture(c.id as typeof capture)}
                  >
                    <c.icon />
                    <span>
                      <strong>{c.name}</strong>
                      <small>{c.description}</small>
                    </span>
                  </button>
                ))}
              </div>
            ) : (
              <>
                <select
                  className="sr-only"
                  aria-label="Format preset"
                  value={format}
                  onChange={(e) => chooseFormat(e.target.value)}
                >
                  {format === "custom" && (
                    <option value="custom">Custom</option>
                  )}
                  {formats.map((f) => (
                    <option key={f.id} value={f.id}>
                      {f.name}
                    </option>
                  ))}
                </select>
                <div className="format-grid">
                  {formats.map((f) => (
                    <button
                      key={f.id}
                      aria-pressed={format === f.id}
                      onClick={() => chooseFormat(f.id)}
                    >
                      <span className="format-preview">
                        <i style={{ aspectRatio: f.aspect }}>
                          {mode === "cinematic" ? (
                            <Film size={19} />
                          ) : (
                            <FileImage size={19} />
                          )}
                        </i>
                      </span>
                      <strong>{f.name}</strong>
                      <small>{f.detail} pixels</small>
                    </button>
                  ))}
                </div>
              </>
            )}
          </section>
          <section className="document-properties">
            <h3>Project settings</h3>
            <label>
              Project name
              <input
                autoFocus
                value={name}
                maxLength={100}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            {mode === "worlds" ? (
              <>
                <label>
                  Server address
                  <input
                    value={address}
                    placeholder="e.g. mco.cubecraft.net:19132"
                    onChange={(e) => setAddress(e.target.value)}
                  />
                </label>
                <p>
                  Configure connection options and capture settings in the Capture workspace next.
                </p>
              </>
            ) : (
              <>
                <label>
                  World
                  <select
                    aria-label="World"
                    value={worldId}
                    onChange={(e) => setWorldId(e.target.value)}
                  >
                    <option value="">Select later</option>
                    {worlds.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.name}
                      </option>
                    ))}
                    <option value="archive">Open world file …</option>
                    <option value="folder">Open world folder …</option>
                  </select>
                </label>
                <div className="document-size">
                  <label>
                    Width
                    <input
                      type="number"
                      min={64}
                      max={4096}
                      value={width || ""}
                      onChange={(e) => {
                        setWidth(+e.target.value);
                        setFormat("custom");
                      }}
                    />
                  </label>
                  <label>
                    Height
                    <input
                      type="number"
                      min={64}
                      max={4096}
                      value={height || ""}
                      onChange={(e) => {
                        setHeight(+e.target.value);
                        setFormat("custom");
                      }}
                    />
                  </label>
                  <button
                    className="studio-button"
                    aria-label="Swap orientation"
                    title="Swap orientation"
                    onClick={() => {
                      setWidth(height);
                      setHeight(width);
                      setFormat("custom");
                    }}
                  >
                    <ArrowLeftRight size={14} />
                  </button>
                </div>
                {!validSize && (
                  <p role="alert">
                    64 to 4096 pixels per side. Aspect ratio must be between 1:4 and 4:1.
                  </p>
                )}
                <label>
                  Image look
                  <select
                    aria-label="Image look"
                    value={look}
                    onChange={(e) => setLook(e.target.value)}
                  >
                    {Object.entries(lookCategories).map(([category, label]) => (
                      <optgroup key={category} label={label}>
                        {lookCatalog
                          .filter((l) => l.category === category)
                          .map((l) => (
                            <option key={l.id} value={l.id}>
                              {renderPresetLabels[l.id]}
                            </option>
                          ))}
                      </optgroup>
                    ))}
                  </select>
                </label>
                {mode === "cinematic" && (
                  <div className="sequence-fields">
                    <label>
                      Duration (seconds)
                      <input
                        type="number"
                        min={1}
                        max={600}
                        value={duration || ""}
                        onChange={(e) => setDuration(+e.target.value)}
                      />
                    </label>
                    <label>
                      Frame rate
                      <select
                        aria-label="Frame rate"
                        value={fps}
                        onChange={(e) => setFps(+e.target.value)}
                      >
                        {[24, 25, 30, 50, 60].map((f) => (
                          <option key={f} value={f}>
                            {f} fps
                          </option>
                        ))}
                      </select>
                    </label>
                  </div>
                )}
                <p>
                  {mode === "cinematic"
                    ? "A new timeline with three camera keyframes will be created."
                    : "A new still render will be created with the selected look."}{" "}
                  You can change all settings later.
                </p>
              </>
            )}
          </section>
        </div>
        <DialogFooter>
          <span className="new-project-summary">
            {mode === "worlds"
              ? "Configure capture"
              : `${preset?.name || "Custom"} · ${width} × ${height} pixels`}
          </span>
          <button className="studio-button" onClick={() => onOpenChange(false)}>
            Cancel
          </button>
          <button
            className="studio-button primary"
            disabled={mode !== "worlds" && !validSize}
            onClick={create}
          >
            Create
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
export function StartScreen({
  recent,
  worlds,
  onCreate,
  onOpen,
  onOpenProject,
  onResume,
  onPage,
}: {
  recent: StudioProject[];
  onCreate: () => void;
  onOpen: (kind: "archive" | "folder") => void;
  onOpenProject?:()=>void;
  onResume: (p: StudioProject) => void;
  onPage: (page: string) => void;
} & Shared) {
  const [query, setQuery] = useState("");
  const [area, setArea] = useState("all");
  const filtered = recent.filter(
    (p) =>
      p.name.toLocaleLowerCase().includes(query.toLocaleLowerCase()) &&
      (area === "all" ||
        (area === "capture"
          ? !["render", "cinematic"].includes(p.page)
          : p.page === area)),
  );
  return (
    <section className="start-screen">
      <header className="start-heading">
        <div>
          <h2>Get started</h2>
        </div>
        <div className="start-actions">
          <button className="studio-button primary" onClick={onCreate}>
            <Plus size={15} />
            Create new
          </button>
          <button className="studio-button" onClick={() => onOpen("archive")}>
            <FolderOpen size={15} />
            Open world file
          </button>
          {onOpenProject&&<button className="studio-button" onClick={onOpenProject}><FolderOpen size={15}/>Open project</button>}
          <button className="studio-button" onClick={() => onOpen("folder")}>
            <FolderOpen size={15} />
            Open world folder
          </button>
        </div>
      </header>
      <div className="start-launchpad">
        <h2>Workspaces</h2>
        <div className="start-workflows">
          {modes.map((m) => (
            <button key={m.id} onClick={() => onPage(m.id)}>
              <m.icon size={23} />
              <strong>{m.name}</strong>
              <span>{m.description}</span>
            </button>
          ))}
        </div>
      </div>
      <div className="recent-heading">
        <h2>Recent</h2>
        <select
          aria-label="Filter project workspace"
          value={area}
          onChange={(e) => setArea(e.target.value)}
        >
          <option value="all">All workspaces</option>
          <option value="render">Still renders</option>
          <option value="cinematic">Camera sequences</option>
          <option value="capture">Captures</option>
        </select>
        <label>
          <Search size={14} />
          <input
            aria-label="Search projects"
            placeholder="Filter by name"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
      </div>
      {filtered.length ? (
        <div className="recent-projects">
          <div className="recent-columns">
            <span>Name</span>
            <span>Workspace</span>
            <span>Format</span>
            <span>Last opened</span>
          </div>
          {filtered.map((project) => (
            <button key={project.id} onClick={() => onResume(project)}>
              <span>
                {project.page === "cinematic" ? (
                  <Clapperboard />
                ) : project.page === "render" ? (
                  <Aperture />
                ) : (
                  <Globe2 />
                )}
                <span>
                  <strong>{project.name}</strong>
                  <small>
                    {project.worldId &&
                    !worlds.some((w) => w.id === project.worldId)
                      ? "Select world again"
                      : worlds.find((w) => w.id === project.worldId)?.name ||
                        "Project template"}
                  </small>
                </span>
              </span>
              <span>
                {project.page === "cinematic"
                  ? "Cinematic"
                  : project.page === "render"
                    ? "Render"
                    : project.page === "packs"
                      ? "Resource packs"
                      : project.page === "skins"
                        ? "Skins"
                        : "Capture"}
              </span>
              <span>
                {["render", "cinematic"].includes(project.page)
                  ? `${project.edge} px${project.page === "cinematic" ? ` / ${project.fps} fps` : ""}`
                  : "Bedrock"}
              </span>
              <time>
                {new Date(project.updated).toLocaleDateString("en-US", {
                  day: "2-digit",
                  month: "short",
                  year: "numeric",
                })}
              </time>
            </button>
          ))}
        </div>
      ) : (
        <div className="recent-empty">
          <FolderOpen size={29} />
          <h3>
            {query || area !== "all"
              ? "No matching projects"
              : "No projects yet"}
          </h3>
          <p>
            {query || area !== "all"
              ? "Change your search or workspace filter."
              : "Create a project or open a world to get started."}
          </p>
        </div>
      )}
      {worlds.length > 0 && (
        <section className="start-worlds">
          <h2>Worlds in this session</h2>
          {worlds.map((w) => (
            <button
              key={w.id}
              onClick={() =>
                onResume({
                  ...projectDefaults,
                  id: crypto.randomUUID(),
                  name: w.name,
                  page: "render",
                  worldId: w.id,
                  updated: Date.now(),
                })
              }
            >
              <Globe2 size={17} />
              <span>
                <strong>{w.name}</strong>
                <small>{w.chunks} Chunks</small>
              </span>
              <ArrowUpRight size={14} />
            </button>
          ))}
        </section>
      )}
    </section>
  );
}
export function StartDialog({
  open,
  onOpenChange,
  ...props
}: ComponentProps<typeof StartScreen> & {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="welcome-dialog">
        <DialogHeader>
          <StudioBrand size={22} />
          <DialogTitle>Welcome to CraftBox</DialogTitle>
          <DialogDescription>
            Start a new project or continue your work.
          </DialogDescription>
        </DialogHeader>
        <StartScreen {...props} />
        <DialogFooter>
          <span className="welcome-shortcut">
            New project <kbd>Ctrl + N</kbd>
          </span>
          <button className="studio-button" onClick={() => onOpenChange(false)}>
            Open workspace
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

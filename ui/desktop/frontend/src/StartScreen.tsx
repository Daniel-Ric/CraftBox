import StudioInput from "@/StudioInput";
import CollectionToolbar from "./CollectionToolbar";
import {collectionSelection} from "./collection-selection";
import { sourceAddressError } from "./input-rules";
import { useEffect, useRef, useState, type ComponentProps } from "react";
import {
  FolderOpen,
  Plus,
  FileImage,
  Film,
  Search,
  ArrowUpRight,
  ArrowLeftRight,
  Pin,
  Activity,
  Store,
  History,
  Archive,
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
import {
  newProjectDefaults,
  projectLabels,
  suggestProjectName,
  uniqueProjectName,
  type StudioProject,
  type RecentProjectAction,
} from "./start-project";
import RecentProjectDetails from "./RecentProjectDetails";
import { Button } from "@/components/ui/button";
import {
  RenderIcon,
  CinematicIcon,
  CaptureIcon,
  WorldIcon,
  PackIcon,
  SkinIcon,
  WorkspaceIcon,
} from "./studio-icons";
import "./recent-projects.css";
import "./welcome-home.css";
import { renderPresetLabels } from "./render-settings";
import { lookCatalog } from "./look-catalog";
import { lookCategories } from "./scene-looks";
const modes = [
  {
    id: "render",
    name: "Still render",
    icon: RenderIcon,
    description: "Lighting, shadows and color.",
  },
  {
    id: "cinematic",
    name: "Camera sequence",
    icon: CinematicIcon,
    description: "Timeline and keyframes.",
  },
  {
    id: "worlds",
    name: "Capture",
    icon: CaptureIcon,
    description: "Worlds, packs and skins.",
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
  welcome,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onCreate: (p: StudioProject, chooseSource?: boolean) => void;
  welcome?: {
    recent: StudioProject[];
    onOpen: (kind: "archive" | "folder") => void;
    onOpenProject?: () => void;
    onRecentAction?: (id: string, action: RecentProjectAction) => void;
    onPage?: (page: string) => void;
  };
} & Shared) {
  const [mode, setMode] = useState<"render" | "cinematic" | "worlds">("render"),
    [format, setFormat] = useState("landscape");
  const [recentView, setRecentView] = useState(!!welcome),
    [recentId, setRecentId] = useState(welcome?.recent[0]?.id);
  const [recentMultiple,setRecentMultiple]=useState(false),[recentSelected,setRecentSelected]=useState<string[]>(welcome?.recent[0]?[welcome.recent[0].id]:[]),[recentAnchor,setRecentAnchor]=useState(welcome?.recent[0]?.id??"");
  const welcomeFocus = useRef<HTMLDivElement>(null);
  const [recentQuery, setRecentQuery] = useState(""),
    [recentType, setRecentType] = useState("all"),
    [recentSort, setRecentSort] = useState("recent"),
    [pinnedOnly, setPinnedOnly] = useState(false);
  const filteredRecent = (welcome?.recent ?? [])
    .filter(
      (p) =>
        `${p.name} ${p.sourceName ?? ""} ${p.address ?? ""}`
          .toLocaleLowerCase()
          .includes(recentQuery.trim().toLocaleLowerCase()) &&
        (recentType === "all" ||
          (recentType === "capture"
            ? !["render", "cinematic"].includes(p.page)
            : p.page === recentType)) &&
        (!pinnedOnly || p.pinned),
    )
    .sort(
      (a, b) =>
        Number(!!b.pinned) - Number(!!a.pinned) ||
        (recentSort === "name"
          ? a.name.localeCompare(b.name)
          : b.updated - a.updated),
    );
  const recentProject =
    filteredRecent.find((p) => p.id === recentId) ?? filteredRecent[0];
  const [width, setWidth] = useState(1920),
    [height, setHeight] = useState(1080);
  useEffect(() => {
    if (!open) return;
    const defaults = newProjectDefaults();
    setWidth(
      Math.round(
        defaults.aspect >= 1 ? defaults.edge : defaults.edge * defaults.aspect,
      ),
    );
    setHeight(
      Math.round(
        defaults.aspect >= 1 ? defaults.edge / defaults.aspect : defaults.edge,
      ),
    );
    setFormat(
      formats.find(
        (f) => f.edge === defaults.edge && f.aspect === defaults.aspect,
      )?.id ?? "custom",
    );
  }, [open]);
  function chooseFormat(id: string) {
    const p = formats.find((f) => f.id === id)!;
    setFormat(id);
    setWidth(Math.round(p.aspect >= 1 ? p.edge : p.edge * p.aspect));
    setHeight(Math.round(p.aspect >= 1 ? p.edge / p.aspect : p.edge));
  }
  const [name, setName] = useState(""),
    [autoName, setAutoName] = useState(true),
    [look, setLook] = useState("island"),
    [duration, setDuration] = useState(12),
    [fps, setFps] = useState(24),
    [worldId, setWorldId] = useState(""),
    [capture, setCapture] = useState<"worlds" | "packs" | "skins">("worlds"),
    [address, setAddress] = useState("");
  const projectPage = mode === "worlds" ? capture : mode;
  const sourceName =
    mode === "worlds" ? undefined : worlds.find((w) => w.id === worldId)?.name;
  const displayName = autoName
    ? suggestProjectName({
        page: projectPage,
        sourceName,
        address: mode === "worlds" ? address : "",
        updated: Date.now(),
      })
    : name;
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
    if (recentView) {
      if (recentProject) {
        onCreate(recentProject);
        onOpenChange(false);
      }
      return;
    }
    const createProject =
      mode === "worlds"
        ? (project: StudioProject) => onCreate(project, true)
        : onCreate;
    createProject({
      ...newProjectDefaults(),
      id: crypto.randomUUID(),
      name: autoName
        ? uniqueProjectName(displayName, welcome?.recent ?? [])
        : name.trim(),
      nameMode: autoName ? "auto" : "custom",
      sourceName,
      page: mode === "worlds" ? capture : mode,
      worldId:
        mode === "worlds" || worldId === "archive" || worldId === "folder"
          ? ""
          : worldId,
      openKind:
        mode !== "worlds" && (worldId === "archive" || worldId === "folder")
          ? worldId
          : undefined,
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
      <DialogContent
        ref={welcomeFocus}
        initialFocus={welcome ? welcomeFocus : undefined}
        className={`new-project-dialog${welcome ? " welcome-dialog welcome-document-dialog" : ""}`}
      >
        <DialogHeader>
          <DialogTitle>
            {welcome ? "Welcome to CraftBox" : "New project"}
          </DialogTitle>
          <DialogDescription>Project setup</DialogDescription>
        </DialogHeader>
        <div className={welcome ? "welcome-layout" : "project-entry-layout"}>
          {welcome ? (
            <nav
              className="studio-sidebar welcome-sidebar"
              aria-label="Welcome navigation"
              onKeyDown={(e) => {
                if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key))
                  return;
                const buttons = Array.from(
                  e.currentTarget.querySelectorAll<HTMLButtonElement>(
                    ".welcome-nav-button",
                  ),
                );
                const at = buttons.indexOf(e.target as HTMLButtonElement);
                if (at < 0) return;
                e.preventDefault();
                const next =
                  e.key === "Home"
                    ? 0
                    : e.key === "End"
                      ? buttons.length - 1
                      : (at +
                          (e.key === "ArrowDown" ? 1 : -1) +
                          buttons.length) %
                        buttons.length;
                buttons[next]?.focus();
              }}
            >
              <button
                className="welcome-nav-button"
                aria-pressed={recentView}
                onClick={() => setRecentView(true)}
              >
                <History size={16} />
                <span>Recent</span>
              </button>
              <h3 className="welcome-sidebar-heading">Create</h3>
              {modes.map((m) => (
                <button
                  className="welcome-nav-button"
                  key={m.id}
                  aria-pressed={!recentView && mode === m.id}
                  title={m.description}
                  onClick={() => {
                    setMode(m.id);
                    setRecentView(false);
                  }}
                >
                  <m.icon size={16} />
                  <span>{m.name}</span>
                </button>
              ))}
              {welcome.onPage && (
                <>
                  <h3 className="welcome-sidebar-heading">Explore</h3>
                  {[
                    {
                      id: "packs",
                      name: "Resource packs",
                      description: "Models, textures and audio tracks",
                      icon: PackIcon,
                    },
                    {
                      id: "skins",
                      name: "Player skins",
                      description: "Slim skins, posing and cosmetics",
                      icon: SkinIcon,
                    },
                    {
                      id: "packets",
                      name: "Packet Inspector",
                      description: "Record and inspect NetherNet traffic",
                      icon: Activity,
                    },
                    {
                      id: "marketplace",
                      name: "Marketplace",
                      description: "Server offers and collections",
                      icon: Store,
                    },
                    {
                      id: "bedrock",
                      name: "Archive",
                      description: "Preview archives, schemas and version changes",
                      icon: Archive,
                    },
                  ].map((tool) => (
                    <button
                      className="welcome-nav-button"
                      key={tool.id}
                      title={tool.description}
                      onClick={() => {
                        welcome.onPage?.(tool.id);
                        onOpenChange(false);
                      }}
                    >
                      <tool.icon size={16} />
                      <span>{tool.name}</span>
                    </button>
                  ))}
                </>
              )}
              {welcome && (
                <div className="welcome-open-actions">
                  <button
                    className="studio-button"
                    onClick={() => welcome.onOpen("archive")}
                  >
                    <FolderOpen size={14} />
                    Open world file
                  </button>
                  <button
                    className="studio-button"
                    onClick={() => welcome.onOpen("folder")}
                  >
                    <FolderOpen size={14} />
                    Open world folder
                  </button>
                  {welcome.onOpenProject && (
                    <button
                      className="studio-button"
                      onClick={welcome.onOpenProject}
                    >
                      Open project
                    </button>
                  )}
                </div>
              )}
            </nav>
          ) : (
            <div
              className="new-project-tabs"
              role="tablist"
              aria-label="Project type"
            >
              {welcome && (
                <button
                  role="tab"
                  aria-selected={recentView}
                  onClick={() => setRecentView(true)}
                >
                  Recent
                </button>
              )}
              {modes.map((m) => (
                <button
                  role="tab"
                  aria-selected={!recentView && mode === m.id}
                  key={m.id}
                  onClick={() => {
                    setMode(m.id);
                    setRecentView(false);
                  }}
                >
                  <m.icon size={15} />
                  {m.name}
                </button>
              ))}
            </div>
          )}
          <div className="new-project-body">
            <section
              className="document-presets"
              role="tabpanel"
              aria-label={
                recentView
                  ? "Recent projects"
                  : modes.find((m) => m.id === mode)!.name
              }
            >
              <h3>
                {recentView
                  ? "Recent projects"
                  : mode === "worlds"
                    ? "Capture type"
                    : "Document presets"}
              </h3>
              {recentView ? (
                <>
                  <div className="recent-project-toolbar">
                    <label className="recent-search">
                      <Search size={14} />
                      <StudioInput
                        aria-label="Search recent projects"
                        placeholder="Search projects or sources"
                        value={recentQuery}
                        onChange={(e) => setRecentQuery(e.target.value)}
                      />
                    </label>
                    <select
                      aria-label="Filter recent projects"
                      value={recentType}
                      onChange={(e) => setRecentType(e.target.value)}
                    >
                      <option value="all">All projects</option>
                      <option value="render">Still renders</option>
                      <option value="cinematic">Camera sequences</option>
                      <option value="capture">Captures</option>
                    </select>
                    <select
                      aria-label="Sort recent projects"
                      value={recentSort}
                      onChange={(e) => setRecentSort(e.target.value)}
                    >
                      <option value="recent">Last opened</option>
                      <option value="name">Name</option>
                    </select>
                    <button
                      className="studio-button welcome-pin-filter"
                      aria-label="Pinned only"
                      title="Pinned only"
                      aria-pressed={pinnedOnly}
                      onClick={() => setPinnedOnly((v) => !v)}
                    >
                      <Pin size={14} />
                      <span className="welcome-pin-label">Pinned only</span>
                    </button>
                  </div>
                  <div className="format-grid recent-preset-grid">
                    <div className="recent-selection-toolbar"><CollectionToolbar ids={filteredRecent.map(project=>project.id)} selected={recentSelected} multiple={recentMultiple} onMultiple={setRecentMultiple} onSelected={setRecentSelected}
                      onDelete={welcome?.onRecentAction?ids=>{ids.forEach(id=>welcome.onRecentAction?.(id,{type:"remove"}));setRecentSelected([]);}:undefined}/></div>
                    {filteredRecent.length ? (
                      filteredRecent.map((project) => (
                        <button
                          key={project.id}
                          aria-label={project.name}
                          title={project.name}
                          aria-pressed={recentMultiple?recentSelected.includes(project.id):recentProject?.id === project.id}
                          onClick={event=>{
                            const next=collectionSelection(filteredRecent.map(value=>value.id),recentSelected,recentAnchor,project.id,event,recentMultiple);
                            setRecentSelected(next.selected);setRecentMultiple(next.multiple);setRecentAnchor(next.anchor);
                            if(!next.multiple)setRecentId(project.id);
                          }}
                          onDoubleClick={() => {
                            if(recentMultiple)return;
                            onCreate(project);
                            onOpenChange(false);
                          }}
                        >
                          {project.pinned && (
                            <Pin
                              size={14}
                              className="recent-pin"
                              aria-label="Pinned"
                            />
                          )}
                          <span
                            className="format-preview project-kind-preview"
                            aria-hidden="true"
                          >
                            <WorkspaceIcon page={project.page} size={36} />
                          </span>
                          <strong>{project.name}</strong>
                          <small>{projectLabels[project.page]}</small>
                          <small>
                            {new Date(project.updated).toLocaleDateString(
                              undefined,
                              { month: "short", day: "numeric" },
                            )}
                            {["render", "cinematic"].includes(project.page)
                              ? ` · ${project.edge} px`
                              : ""}
                          </small>
                        </button>
                      ))
                    ) : (
                      <p className="secondary">
                        {welcome?.recent.length
                          ? "No matching projects."
                          : "No recent projects."}
                      </p>
                    )}
                  </div>
                </>
              ) : mode === "worlds" ? (
                <div className="capture-presets">
                  {[
                    {
                      id: "worlds",
                      name: "World & builds",
                      description: "Terrain and entities.",
                      icon: WorldIcon,
                    },
                    {
                      id: "packs",
                      name: "Resource packs",
                      description: "Textures and resources.",
                      icon: PackIcon,
                    },
                    {
                      id: "skins",
                      name: "Player skins",
                      description: "Player skins.",
                      icon: SkinIcon,
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
              {recentView ? (
                <>
                  <h3>Project details</h3>
                  {!recentProject && <strong>Select a project</strong>}
                  {recentProject && (
                    <RecentProjectDetails
                      project={recentProject}
                      worlds={worlds}
                      onAction={welcome?.onRecentAction}
                    />
                  )}
                </>
              ) : (
                <>
                  <h3>Preset details</h3>
                  <label>
                    Project name
                    <StudioInput
                      autoFocus
                      required
                      value={displayName}
                      maxLength={100}
                      onChange={(e) => {
                        setName(e.target.value);
                        setAutoName(false);
                      }}
                    />
                  </label>
                  <label className="auto-project-name">
                    <input
                      type="checkbox"
                      checked={autoName}
                      onChange={(e) => {
                        if (!e.target.checked) setName(displayName);
                        setAutoName(e.target.checked);
                      }}
                    />
                    Name automatically
                  </label>
                  {mode === "worlds" ? (
                    <>
                      <label>
                        Server address
                        <StudioInput
                          value={address}
                          placeholder="e.g. mco.cubecraft.net:19132"
                          validate={sourceAddressError}
                          onChange={(e) => setAddress(e.target.value)}
                        />
                      </label>
                      <p>Choose the connection settings in Capture.</p>
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
                          <StudioInput
                            type="number"
                            min={64}
                            max={4096}
                            step={1}
                            value={width || ""}
                            onChange={(e) => {
                              setWidth(+e.target.value);
                              setFormat("custom");
                            }}
                          />
                        </label>
                        <label>
                          Height
                          <StudioInput
                            type="number"
                            min={64}
                            max={4096}
                            step={1}
                            value={height || ""}
                            error={
                              !validSize
                                ? "Use an aspect ratio from 1:4–4:1."
                                : undefined
                            }
                            onChange={(e) => {
                              setHeight(+e.target.value);
                              setFormat("custom");
                            }}
                          />
                        </label>
                        <button
                          className="studio-button"
                          aria-label="Swap orientation"
                          data-tooltip="Swap orientation"
                          onClick={() => {
                            setWidth(height);
                            setHeight(width);
                            setFormat("custom");
                          }}
                        >
                          <ArrowLeftRight size={14} />
                        </button>
                      </div>
                      <label>
                        Image look
                        <select
                          aria-label="Image look"
                          value={look}
                          onChange={(e) => setLook(e.target.value)}
                        >
                          {Object.entries(lookCategories).map(
                            ([category, label]) => (
                              <optgroup key={category} label={label}>
                                {lookCatalog
                                  .filter((l) => l.category === category)
                                  .map((l) => (
                                    <option key={l.id} value={l.id}>
                                      {renderPresetLabels[l.id]}
                                    </option>
                                  ))}
                              </optgroup>
                            ),
                          )}
                        </select>
                      </label>
                      {mode === "cinematic" && (
                        <div className="sequence-fields">
                          <label>
                            Duration (seconds)
                            <StudioInput
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
                </>
              )}
            </section>
          </div>
        </div>
        <DialogFooter>
          <span className="new-project-summary">
            {recentView
              ? `${filteredRecent.length} recent project${filteredRecent.length === 1 ? "" : "s"}`
              : mode === "worlds"
                ? "Configure capture"
                : `${preset?.name || "Custom"} · ${width} × ${height} pixels`}
          </span>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            {welcome ? "Open workspace" : "Cancel"}
          </Button>
          <Button
            disabled={
              recentView
                ? !recentProject
                : !displayName.trim() || (mode !== "worlds" && !validSize)
            }
            onClick={create}
          >
            {recentView ? "Open" : "Create"}
          </Button>
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
  onOpenProject?: () => void;
  onResume: (p: StudioProject) => void;
  onPage: (page: string) => void;
  onRecentAction?: (id: string, action: RecentProjectAction) => void;
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
          <h2>Start a project</h2>
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
          {onOpenProject && (
            <button className="studio-button" onClick={onOpenProject}>
              <FolderOpen size={15} />
              Open project
            </button>
          )}
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
          <StudioInput
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
                <WorkspaceIcon page={project.page} />
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
              : "Create a project or open a world."}
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
                  ...newProjectDefaults(),
                  id: crypto.randomUUID(),
                  name: w.name,
                  page: "render",
                  worldId: w.id,
                  updated: Date.now(),
                })
              }
            >
              <WorldIcon size={17} />
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
    <NewProjectDialog
      open={open}
      onOpenChange={onOpenChange}
      worlds={props.worlds}
      onCreate={props.onResume}
      welcome={{
        recent: props.recent,
        onOpen: props.onOpen,
        onOpenProject: props.onOpenProject,
        onRecentAction: props.onRecentAction,
        onPage: props.onPage,
      }}
    />
  );
}

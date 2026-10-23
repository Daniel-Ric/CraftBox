import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { useStudioProjectGuard } from "./StudioProjectGuard";
import {
  parseStudioDocument,
  type StudioDocument,
  type StudioSession,
} from "./studio-document";
import type { RenderWorld } from "./bridge";
import type { EditorCommand } from "./editor-commands";
import {
  Globe2,
  Package,
  Users,
  Terminal,
  ChevronDown,
  ChevronRight,
  ArrowUpRight,
  FolderOpen,
  Play,
  Square,
  Server,
  Radio,
  Copy,
  Check,
  LogIn,
  LogOut,
  RefreshCw,
  Search,
  SlidersHorizontal,
  File,
  LoaderCircle,
  AlertCircle,
  Map as MapIcon,
  Flag,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { TooltipProvider } from "@/components/ui/tooltip";
import { api, on } from "./bridge";
import type {
  Bootstrap,
  State,
  Source,
  Option,
  Tile,
  Log,
  ChunkBounds,
} from "./bridge";
import WorldMap from "./WorldMap";
import DesktopChrome from "./DesktopChrome";
import AssetLibrary from "./AssetLibrary";
import StudioSupportDialogs from "./StudioSupportDialogs";
import PanelResizeHandle from "./PanelResizeHandle";
import PreferencesDialog, {
  loadWorkspacePreferences,
  applyWorkspacePreferences,
} from "./PreferencesDialog";
import { StartDialog, NewProjectDialog } from "./StartScreen";
import {
  loadRecent,
  rememberProject,
  projectDefaults,
  type StudioProject,
} from "./start-project";
import { dispatchEditorCommand, idleEditor } from "./editor-commands";
const WorldScene = lazy(() => import("./WorldScene"));
const RenderEditor = lazy(() => import("./RenderEditor"));
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
const tileKey = (tile: Tile) => `${tile.worldId ?? ""}:${tile.x},${tile.z}`;

const names: Record<string, string> = {
  worlds: "Worlds",
  packs: "Resource packs",
  skins: "Player skins",
  render: "Render editor",
  cinematic: "Cinematic editor",
};

const descriptions: Record<string, string> = {
  worlds: "Explore the server. Save your world.",
  packs: "Download and save server resource packs.",
  skins: "Collect player skins during your session.",
};
const optionLabels: Record<string, string> = {
  capture: "Record packets",
  "client-cache": "Use client cache",
  void: "Generate a void world",
  image: "Save map image",
  "save-entities": "Save entities",
  "save-custom-entities": "Record custom entities",
  "save-players": "Save other players as models",
  "save-inventories": "Save inventories",
  "block-updates": "Capture block updates",
  "entity-culling": "Cull removed entities",
  debug: "Debug logs",
  "extra-debug": "Verbose debug logs",
  listen: "Local listening address",
  "exclude-mobs": "Exclude mobs",
  "chunk-radius": "Chunk radius",
  script: "Script file",
};
const primaryFlags = new Set([
  "capture",
  "client-cache",
  "void",
  "image",
  "save-entities",
  "save-custom-entities",
  "save-inventories",
]);
const optionHelp: Record<string, string> = {
  capture: "Record the session for later replay.",
  "client-cache": "Use data cached by the Minecraft client.",
  void: "Leave the world empty outside captured chunks.",
  image: "Also save a PNG map of the captured world.",
  "save-entities": "Include mobs and other captured entities.",
  "save-custom-entities":
    "Include custom server models and textures in the world, 3D view and exports. Off by default.",
  "save-players":
    "Optionally export visible players as models. Off by default.",
  "save-inventories": "Save captured inventories and containers.",
  "block-updates": "Include block updates received during the session.",
  "entity-culling":
    "Remove entities that despawn or die. Experimental.",
  listen: "Address and port for the local Minecraft connection.",
  "exclude-mobs": "Separate multiple mob names with commas.",
  "chunk-radius": "Limit the requested view distance in chunks.",
  script: "Optional JavaScript file to run during the session.",
};
const sourceNames: Record<string, string> = {
  realms: "Realms",
  events: "Events",
  featured: "Featured servers",
  worlds: "Friends' worlds",
};
const connectionNames = [
  "Ready",
  "Preparing",
  "Waiting for Minecraft",
  "Connecting to server",
  "Receiving resources",
  "Connected",
  "Disconnected",
];
function errorText(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
const emptyState: State = {
  revision: 0,
  command: "",
  running: false,
  stopping: false,
  connection: 0,
  listenAddr: "",
  localIP: "",
  worlds: [],
  packs: [],
  skins: [],
  player: [0, 0, 0],
  hasPlayer: false,
  account: "",
  loggedIn: false,
  authCode: "",
  authURI: "",
  authenticating: false,
  updateVersion: "",
  updateProgress: 0,
  error: "",
};

export default function App() {
  useEffect(() => applyWorkspacePreferences(loadWorkspacePreferences()), []);
  const [regionEditor, setRegionEditor] = useState(false);
  const [regionBounds, setRegionBounds] = useState<ChunkBounds>();
  const [regionCoordinates, setRegionCoordinates] = useState<
    Record<keyof ChunkBounds, string>
  >({ minX: "", minZ: "", maxX: "", maxZ: "" });
  const requestedRegionBounds = Object.values(regionCoordinates).some(
    (value) => !value.trim() || !Number.isFinite(Number(value)),
  )
    ? undefined
    : (Object.fromEntries(
        Object.entries(regionCoordinates).map(([key, value]) => [
          key,
          Math.floor(Number(value) / 16),
        ]),
      ) as ChunkBounds);
  function selectRegion(bounds: ChunkBounds) {
    setRegionBounds(bounds);
    setRegionCoordinates({
      minX: String(bounds.minX * 16),
      minZ: String(bounds.minZ * 16),
      maxX: String(bounds.maxX * 16 + 15),
      maxZ: String(bounds.maxZ * 16 + 15),
    });
  }
  function snapRegionCoordinate(key: keyof ChunkBounds) {
    const value = regionCoordinates[key];
    if (!value.trim() || !Number.isFinite(Number(value))) return;
    const chunk = Math.floor(Number(value) / 16);
    setRegionBounds((previous) => ({
      ...(previous ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 }),
      [key]: chunk,
    }));
    setRegionCoordinates((previous) => ({
      ...previous,
      [key]: String(chunk * 16 + (key.startsWith("max") ? 15 : 0)),
    }));
  }
  const [regionName, setRegionName] = useState("");
  const [regionMove, setRegionMove] = useState(true);
  const [regionBusy, setRegionBusy] = useState(false);
  const [regionError, setRegionError] = useState("");
  const [spawnEditor, setSpawnEditor] = useState(false);
  const [spawnValues, setSpawnValues] = useState<[number, number, number]>([
    0, 70, 0,
  ]);
  const [renaming, setRenaming] = useState<{ id: string; name: string }>();
  const [selectedWorldID, setSelectedWorldID] = useState("");
  const [mapMode, setMapMode] = useState<"2d" | "3d">("2d");
  const [sceneCentered, setSceneCentered] = useState(false);
  const worldSessionActive = useRef(false),
    minecraftOpened = useRef(false);
  const [bootstrap, setBootstrap] = useState<Bootstrap>(),
    [state, setState] = useState<State>(emptyState),
    [page, setPage] = useState("worlds");
  const [startOpen, setStartOpen] = useState(
    () => loadWorkspacePreferences().showWelcome,
  );
  const [consoleHeight, setConsoleHeight] = useState(260);
  const [aboutOpen, setAboutOpen] = useState(false),
    [preferenceCategory, setPreferenceCategory] = useState("appearance");
  const returnToStart = useRef(false);
  function navigateWorkspace(next: string) {
    if (next === page && !startOpen) return;
    projectGuard.run(() => {
      if (next === "home") {
        setStartOpen(true);
        setPage("worlds");
        return;
      }
      setStartOpen(false);
      setPage(next);
    });
  }
  function showNewProject() {
    if (newProjectOpen) return;
    projectGuard.run(() => {
      returnToStart.current = startOpen;
      setStartOpen(false);
      setNewProjectOpen(true);
      setPage("worlds");
      setProject(undefined);
      setOpenedStudio(undefined);
    });
  }
  function changeNewProjectOpen(open: boolean) {
    setNewProjectOpen(open);
    if (!open && returnToStart.current) {
      returnToStart.current = false;
      setStartOpen(true);
    }
  }
  const [newProjectOpen, setNewProjectOpen] = useState(false),
    [recent, setRecent] = useState(loadRecent),
    [project, setProject] = useState<StudioProject>();
  const [openedStudio, setOpenedStudio] = useState<{
    world: RenderWorld;
    document: StudioDocument;
    path: string;
  }>();
  const [studioGeneration, setStudioGeneration] = useState(0);
  const projectGuard = useStudioProjectGuard();
  const registerStudio = useCallback((session: StudioSession | undefined) => {
    projectGuard.session.current = session;
    void window.go?.desktop.App?.SetStudioProjectDirty?.(
      session?.dirty ?? false,
    )?.catch(() => {});
  }, []);
  const transition = useRef(projectGuard.run);
  transition.current = projectGuard.run;
  useEffect(
    () =>
      on("studio:close-requested", () =>
        transition.current(() => {
          void api
            .ConfirmStudioProjectClose()
            .catch((e) => setError(String(e)));
        }),
      ),
    [],
  );
  async function openStudio() {
    let pendingId: string | undefined;
    try {
      const result = await api.OpenStudioProject();
      if (!result.path) return;
      pendingId = result.world.id;
      const document = parseStudioDocument(result.document);
      setOpenedStudio({ ...result, document });
      setStudioGeneration((n) => n + 1);
      setProject(undefined);
      setStartOpen(false);
      setPage(document.editor);
    } catch (e) {
      if (pendingId) {
        try {
          await api.CloseRenderWorld(pendingId);
        } catch {

        }
      }
      setError(String(e));
    }
  }
  function studioCommand(command: EditorCommand) {
    if (command === "open-project") projectGuard.run(() => void openStudio());
    else if (command === "save-project" && projectGuard.session.current)
      void projectGuard.session.current
        .save()
        .catch((e) => setError(String(e)));
    else dispatchEditorCommand(command);
  }
  function launchProject(next: StudioProject) {
    projectGuard.run(() => {
      setOpenedStudio(undefined);
      returnToStart.current = false;
      setStartOpen(false);
      const available = {
        ...next,
        worldId: state.worlds.some((w) => w.id === next.worldId)
          ? next.worldId
          : "",
        openKind:
          next.worldId && !state.worlds.some((w) => w.id === next.worldId)
            ? next.sourceKind
            : next.openKind,
      };
      setProject(available);
      setRecent(rememberProject(next));
      if (!["render", "cinematic"].includes(next.page))
        setAddress(next.address || "");
      setPage(next.page);
    });
  }
  function openFromStart(kind: "archive" | "folder") {
    launchProject({
      ...projectDefaults,
      id: crypto.randomUUID(),
      name: "Untitled",
      openKind: kind,
      updated: Date.now(),
    });
  }
  const [values, setValues] = useState<Record<string, Record<string, string>>>(
      {},
    ),
    [address, setAddress] = useState(""),
    [selected, setSelected] = useState<Source>();
  const [advanced, setAdvanced] = useState(false),
    [sourceKind, setSourceKind] = useState("featured"),
    [sourceOpen, setSourceOpen] = useState(false),
    [sources, setSources] = useState<Source[]>([]),
    [sourceLoading, setSourceLoading] = useState(false),
    [sourceError, setSourceError] = useState(""),
    [sourceSearch, setSourceSearch] = useState("");
  const [accountOpen, setAccountOpen] = useState(false),
    [accountName, setAccountName] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [initError, setInitError] = useState("");
  const [logsOpen, setLogsOpen] = useState(false),
    [logs, setLogs] = useState<Log[]>([]),
    [logSearch, setLogSearch] = useState(""),
    [logLevel, setLogLevel] = useState("all"),
    [tiles, setTiles] = useState<Map<string, Tile>>(new globalThis.Map());
  const [mapGeneration, setMapGeneration] = useState(0);
  const [helpOpen, setHelpOpen] = useState(false),
    [settingsOpen, setSettingsOpen] = useState(false);
  const [editorStatus, setEditorStatus] = useState(idleEditor);
  const [theme, setTheme] = useState(
      localStorage.getItem("CraftBox-theme") ?? "dark",
    ),
    [copied, setCopied] = useState(false),
    [updateBusy, setUpdateBusy] = useState(false);
  const mounted = useRef(true);
  const hydrationSequence = useRef(0),
    sourceSequence = useRef(0);
  const pending = useRef({
    active: true,
    state: undefined as State | undefined,
    tiles: new globalThis.Map<string, Tile>(),
    reset: false,
    logs: [] as Log[],
  });
  function applyState(value: State) {
    setState((previous) =>
      (value.revision ?? 0) < (previous.revision ?? 0) ? previous : value,
    );
  }
  function addTiles(items: Tile[]) {
    if (pending.current.active)
      items.forEach((tile) => pending.current.tiles.set(tileKey(tile), tile));
    setTiles((previous) => {
      const next = new globalThis.Map(previous);
      items.forEach((tile) => next.set(tileKey(tile), tile));
      return next;
    });
  }
  async function load() {
    const sequence = ++hydrationSequence.current;
    pending.current = {
      active: true,
      state: undefined,
      tiles: new globalThis.Map(),
      reset: false,
      logs: [],
    };
    setInitError("");
    try {
      const data = await api.GetBootstrap();
      if (!mounted.current || sequence !== hydrationSequence.current) return;
      setBootstrap(data);
      if (data.mapWindow && data.theme) setTheme(data.theme);
      applyState(pending.current.state ?? data.state);
      setValues((old) =>
        Object.fromEntries(
          data.commands.map((cmd) => [
            cmd.name,
            Object.fromEntries(
              cmd.options
                .filter((o) => o.type !== "connectInfo")
                .map((o) => [
                  o.flag,
                  old[cmd.name]?.[o.flag] ?? o.default ?? "",
                ]),
            ),
          ]),
        ),
      );
      setAccountName(data.state.account);
      const [map, history] = await Promise.all([
        api.GetMapTiles(),
        api.GetLogs(),
      ]);
      if (mounted.current && sequence === hydrationSequence.current) {
        const buffer = pending.current;
        setTiles(
          new globalThis.Map([
            ...(buffer.reset ? [] : map.map((t) => [tileKey(t), t] as const)),
            ...buffer.tiles.entries(),
          ]),
        );
        const merged = new globalThis.Map(
          [...history, ...buffer.logs].map((log) => [
            log.id ?? `${log.time}:${log.source}:${log.message}`,
            log,
          ]),
        );
        setLogs([...merged.values()].slice(-2000));
        if (buffer.state) applyState(buffer.state);
        buffer.active = false;
      }
    } catch (e) {
      if (mounted.current && sequence === hydrationSequence.current)
        setInitError(errorText(e));
    }
  }
  useEffect(() => {
    mounted.current = true;
    const unsubscribe = [
      on<string>("desktop:theme", setTheme),
      on<State>("desktop:state", (value) => {
        if (pending.current.active) pending.current.state = value;
        applyState(value);
      }),
      on<Tile[]>("desktop:tiles", addTiles),
      on<string | null>("desktop:reset-map", (id) => {
        if (!id) {
          pending.current.reset = true;
          pending.current.tiles.clear();
          setTiles(new globalThis.Map());
        } else {
          for (const [key, tile] of pending.current.tiles) {
            if (tile.worldId === id) pending.current.tiles.delete(key);
          }
          setTiles(
            (previous) =>
              new globalThis.Map(
                [...previous].filter(([, tile]) => tile.worldId !== id),
              ),
          );
        }
        setMapGeneration((v) => v + 1);
      }),
      on<Log>("desktop:log", (entry) => {
        if (pending.current.active) pending.current.logs.push(entry);
        setLogs((previous) => [...previous.slice(-1999), entry]);
      }),
    ];
    void load();
    return () => {
      mounted.current = false;
      hydrationSequence.current++;
      unsubscribe.forEach((off) => off());
    };
  }, []);
  useEffect(() => {
    const media = window.matchMedia?.("(prefers-color-scheme: dark)");
    const apply = () =>
      document.documentElement.classList.toggle(
        "dark",
        ["dark", "graphite", "gray"].includes(theme) ||
          (theme === "system" && Boolean(media?.matches)),
      );
    apply();
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("CraftBox-theme", theme);
    media?.addEventListener("change", apply);
    return () => media?.removeEventListener("change", apply);
  }, [theme]);
  useEffect(() => {
    if (!bootstrap) return;
    const active = state.running && state.command === "worlds";
    if (active && !worldSessionActive.current && !bootstrap?.mapWindow)
      void action(() => api.OpenMapWindow(theme));
    worldSessionActive.current = active;
    if (!state.running) {
      minecraftOpened.current = false;
      return;
    }
    if (
      !bootstrap?.mapWindow &&
      state.connection === 2 &&
      state.listenAddr &&
      !minecraftOpened.current
    ) {
      minecraftOpened.current = true;
      void api.OpenMinecraft().catch((error) => setError(errorText(error)));
    }
  }, [
    state.running,
    state.command,
    state.connection,
    state.listenAddr,
    bootstrap?.mapWindow,
  ]);
  useEffect(() => {
    if (state.activeWorldId) setSelectedWorldID(state.activeWorldId);
  }, [state.activeWorldId]);
  useEffect(() => {
    if (state.worlds.length === 0) setSelectedWorldID("");
  }, [state.worlds.length]);
  const viewWorldID = selectedWorldID || state.activeWorldId || "";
  const selectedWorld = state.worlds.find((world) => world.id === viewWorldID);
  const canEditWorld =
    !!selectedWorld &&
    (selectedWorld.state === "finished" ||
      (state.running &&
        (selectedWorld.id === state.activeWorldId ||
          selectedWorld.parentId === state.activeWorldId)));
  async function editWorld(request: {
    worldId: string;
    name?: string;
    spawn?: [number, number, number];
  }) {
    setRegionBusy(true);
    setRegionError("");
    try {
      await api.EditWorld(request);
      setRenaming(undefined);
      setSpawnEditor(false);
    } catch (error) {
      setRegionError(errorText(error));
    } finally {
      setRegionBusy(false);
    }
  }
  useEffect(() => {
    setSpawnEditor(false);
    setRegionEditor(false);
    setRegionBounds(undefined);
    setRegionCoordinates({ minX: "", minZ: "", maxX: "", maxZ: "" });
    setRegionError("");
  }, [viewWorldID]);
  useEffect(() => {
    if (!regionEditor) return;
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setRegionEditor(false);
    };
    window.addEventListener("keydown", escape);
    return () => window.removeEventListener("keydown", escape);
  }, [regionEditor]);
  const canEditRegion =
    state.running &&
    state.command === "worlds" &&
    viewWorldID === state.activeWorldId &&
    !selectedWorld?.parentId;
  async function createRegion() {
    if (!regionName.trim() || regionBusy || !requestedRegionBounds) return;
    const bounds = requestedRegionBounds;
    setRegionBusy(true);
    setRegionError("");
    try {
      const region = await api.CreateWorldRegion({
        worldId: viewWorldID,
        name: regionName.trim(),
        bounds,
        move: regionMove,
      });
      setSelectedWorldID(region.id);
      setRegionEditor(false);
      setRegionName("");
      setRegionBounds(undefined);
    } catch (error) {
      setRegionError(errorText(error));
    } finally {
      setRegionBusy(false);
    }
  }
  const visibleTiles = new globalThis.Map(
    [...tiles].filter(([, tile]) => (tile.worldId ?? "") === viewWorldID),
  );
  async function action(fn: () => Promise<unknown>) {
    try {
      await fn();
    } catch (e) {
      setError(errorText(e));
    }
  }
  async function loadSources(kind = sourceKind) {
    const sequence = ++sourceSequence.current;
    setSourceLoading(true);
    setSourceError("");
    try {
      const result = await api.ListSources(kind);
      if (sequence === sourceSequence.current) setSources(result);
    } catch (e) {
      if (sequence === sourceSequence.current) {
        setSources([]);
        setSourceError(errorText(e));
      }
    } finally {
      if (sequence === sourceSequence.current) setSourceLoading(false);
    }
  }
  useEffect(() => {
    setSources([]);
    if (sourceOpen) void loadSources(sourceKind);
    return () => {
      sourceSequence.current++;
    };
  }, [sourceOpen, sourceKind, state.loggedIn, state.account]);
  const cmd = bootstrap?.commands.find((c) => c.name === page),
    options = cmd?.options ?? [],
    current = values[page] ?? {};
  const hasAddress = options.some((o) => o.type === "connectInfo"),
    ready = Boolean(
      bootstrap &&
      !state.running &&
      !state.authenticating &&
      !busy &&
      (!hasAddress || address.trim()) &&
      (!state.account || state.loggedIn || address.endsWith(".pcap2")),
    );
  function setOption(flag: string, value: string) {
    setValues((previous) => ({
      ...previous,
      [page]: { ...previous[page], [flag]: value },
    }));
  }
  async function start() {
    if (!ready) return;
    setBusy(true);
    setError("");
    try {
      await api.StartCommand({
        command: page,
        target: selected?.id ?? address.trim(),
        values: current,
      });
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch (e) {
      setError(errorText(e));
    }
  }
  const visibleLogs = logs.filter(
    (log) =>
      (logLevel === "all" || log.level === logLevel) &&
      `${log.message} ${log.source}`
        .toLowerCase()
        .includes(logSearch.toLowerCase()),
  );
  function field(option: Option) {
    const title = optionLabels[option.flag] ?? option.name ?? option.flag,
      id = `option-${page}-${option.flag}`,
      value = current[option.flag] ?? option.default;
    return (
      <div
        className={option.type === "bool" ? "option-row" : "field-row"}
        key={option.flag}
      >
        <div>
          <Label htmlFor={id}>{title || option.flag}</Label>
          {(optionHelp[option.flag] || option.description) && (
            <p>{optionHelp[option.flag] || option.description}</p>
          )}
        </div>
        {option.type === "bool" ? (
          <Switch
            id={id}
            checked={value === "true"}
            onCheckedChange={(checked) =>
              setOption(option.flag, String(checked))
            }
            disabled={state.running}
          />
        ) : (
          <div className="field-input">
            <Input
              id={id}
              value={value}
              type={option.type === "int" ? "number" : "text"}
              placeholder={option.default || title}
              onChange={(e) => setOption(option.flag, e.target.value)}
              disabled={state.running}
            />
            {option.fileType?.startsWith("file") && (
              <Button
                variant="outline"
                aria-label={`${title} select`}
                disabled={state.running}
                onClick={() =>
                  void action(async () => {
                    const path = await api.PickFile();
                    if (path) setOption(option.flag, path);
                  })
                }
              >
                <File />
              </Button>
            )}
          </div>
        )}
      </div>
    );
  }
  if (bootstrap?.mapWindow)
    return (
      <TooltipProvider>
        <main className="world-window" aria-label="World maps" role="region">
          <DialogHeader className="world-map-dialog-header">
            <div className="world-map-dialog-title">
              <MapIcon />
              <h1>Live world map</h1>
              <div
                className="map-mode-switch"
                role="group"
                aria-label="View"
              >
                <Button
                  variant={mapMode === "2d" ? "secondary" : "ghost"}
                  aria-pressed={mapMode === "2d"}
                  onClick={() => setMapMode("2d")}
                >
                  2D
                </Button>
                <Button
                  variant={mapMode === "3d" ? "secondary" : "ghost"}
                  aria-pressed={mapMode === "3d"}
                  onClick={() => {
                    setMapMode("3d");
                    setRegionEditor(false);
                    setSpawnEditor(false);
                  }}
                >
                  3D
                </Button>
              </div>
              <Button
                variant="outline"
                disabled={!canEditRegion || regionBusy}
                aria-pressed={regionEditor}
                onClick={() => {
                  setRegionEditor((v) => !v);
                  setMapMode("2d");
                  setSpawnEditor(false);
                  setRegionError("");
                }}
              >
                <Square />
                Select region
              </Button>
              <Button
                variant="outline"
                disabled={!canEditWorld || regionBusy}
                aria-pressed={spawnEditor}
                onClick={() => {
                  setRegionEditor(false);
                  setSpawnEditor((v) => !v);
                  setMapMode("2d");
                  setRegionError("");
                  setSpawnValues(
                    selectedWorld?.spawn ??
                      (selectedWorld?.player?.map(Math.floor) as [
                        number,
                        number,
                        number,
                      ]) ?? [0, 70, 0],
                  );
                }}
              >
                <Flag />
                Spawn point
              </Button>
              <Badge
                variant="outline"
                className="map-status-pill"
                role="status"
              >
                <span className="live-dot" />
                {connectionNames[state.connection] ??
                  "Preparing session"}
              </Badge>
            </div>
            <p className="map-window-description">
              Explore your captured terrain. The map updates automatically during the session.
            </p>
          </DialogHeader>
          <Dialog
            open={!!renaming}
            onOpenChange={(open) => {
              if (!open) setRenaming(undefined);
            }}
          >
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Rename world</DialogTitle>
                <DialogDescription>
                  This name will be saved in the world.
                </DialogDescription>
              </DialogHeader>
              <form
                id="rename-world-form"
                className="studio-dialog-body rename-world-form"
                onSubmit={(event) => {
                  event.preventDefault();
                  if (renaming)
                    void editWorld({
                      worldId: renaming.id,
                      name: renaming.name,
                    });
                }}
              >
                <Label htmlFor="rename-world">World name</Label>
                <Input
                  id="rename-world"
                  autoFocus
                  maxLength={80}
                  value={renaming?.name ?? ""}
                  onChange={(event) =>
                    setRenaming((old) =>
                      old ? { ...old, name: event.target.value } : old,
                    )
                  }
                />
                {regionError && <p role="alert">{regionError}</p>}
              </form>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setRenaming(undefined)}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  form="rename-world-form"
                  disabled={regionBusy || !renaming?.name.trim()}
                >
                  Save name
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <Tabs
            className="world-map-tabs"
            value={viewWorldID || "current"}
            onValueChange={(value) => setSelectedWorldID(String(value))}
          >
            <TabsList
              className="world-map-tab-list"
              variant="line"
              aria-label="Captured worlds"
            >
              {state.worlds.length ? (
                state.worlds.map((world, index) => (
                  <TabsTrigger
                    key={world.id || index}
                    value={world.id || "current"}
                    title="Double-click to rename"
                    onDoubleClick={() => {
                      setSelectedWorldID(world.id);
                      setRegionError("");
                      setRenaming({ id: world.id, name: world.name });
                    }}
                  >
                    {world.name} · {index + 1}
                  </TabsTrigger>
                ))
              ) : (
                <TabsTrigger value="current">Current world</TabsTrigger>
              )}
            </TabsList>
            <TabsContent
              value={viewWorldID || "current"}
              className="world-map-tab-panel"
            >
              <div
                className={`map-editor-layout${regionEditor || spawnEditor ? " editing" : ""}`}
              >
                {mapMode === "3d" ? (
                  <Suspense
                    fallback={
                      <div role="status">Loading 3D view …</div>
                    }
                  >
                    <WorldScene
                      key={`${viewWorldID}:${mapGeneration}`}
                      worldId={viewWorldID}
                      name={selectedWorld?.name ?? "Current world"}
                      player={selectedWorld?.player ?? state.player}
                      hasPlayer={selectedWorld?.hasPlayer ?? state.hasPlayer}
                      initialCentered={sceneCentered}
                      onCenteredChange={setSceneCentered}
                    />
                  </Suspense>
                ) : (
                  <WorldMap
                    key={`${viewWorldID}:${mapGeneration}`}
                    tiles={visibleTiles}
                    player={selectedWorld?.player ?? state.player}
                    hasPlayer={selectedWorld?.hasPlayer ?? state.hasPlayer}
                    selecting={regionEditor && !regionBusy}
                    selection={regionEditor ? regionBounds : undefined}
                    onSelectionChange={selectRegion}
                    pickingSpawn={spawnEditor && !regionBusy}
                    spawn={spawnEditor ? spawnValues : selectedWorld?.spawn}
                    onSpawnPick={(x, z) =>
                      setSpawnValues((old) => [x, old[1], z])
                    }
                  />
                )}
                {spawnEditor && (
                  <aside
                    className="region-editor"
                    aria-label="Edit spawn point"
                  >
                    <h2>Spawn point</h2>
                    <p>Click the terrain or enter coordinates.</p>
                    <form
                      onSubmit={(event) => {
                        event.preventDefault();
                        void editWorld({
                          worldId: viewWorldID,
                          spawn: spawnValues,
                        });
                      }}
                    >
                      {(["X", "Y", "Z"] as const).map((label, i) => (
                        <div key={label}>
                          <Label htmlFor={`spawn-${label}`}>{label}</Label>
                          <Input
                            id={`spawn-${label}`}
                            type="number"
                            step={1}
                            required
                            value={spawnValues[i]}
                            onChange={(event) =>
                              setSpawnValues(
                                (old) =>
                                  old.map((v, j) =>
                                    j === i ? Number(event.target.value) : v,
                                  ) as [number, number, number],
                              )
                            }
                          />
                        </div>
                      ))}
                      {regionError && (
                        <p role="alert" className="region-error">
                          {regionError}
                        </p>
                      )}
                      <Button type="submit" disabled={regionBusy}>
                        Save spawn point
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() => setSpawnEditor(false)}
                      >
                        Cancel
                      </Button>
                    </form>
                  </aside>
                )}
                {regionEditor && (
                  <aside
                    className="region-editor"
                    aria-label="Edit world region"
                  >
                    <h2>Create a separate world</h2>
                    <p>Drag a rectangle · Hold the middle mouse button to pan.</p>
                    <form
                      onSubmit={(event) => {
                        event.preventDefault();
                        void createRegion();
                      }}
                    >
                      <Label htmlFor="region-name">World name</Label>
                      <Input
                        id="region-name"
                        value={regionName}
                        maxLength={80}
                        placeholder="e.g. SkyWars island"
                        disabled={regionBusy}
                        onChange={(event) => setRegionName(event.target.value)}
                      />
                      <div className="region-coordinates">
                        {(
                          [
                            ["minX", "X min"],
                            ["maxX", "X max"],
                            ["minZ", "Z min"],
                            ["maxZ", "Z max"],
                          ] as const
                        ).map(([key, label]) => (
                          <div key={key}>
                            <Label htmlFor={`region-${key}`}>{label}</Label>
                            <Input
                              id={`region-${key}`}
                              type="number"
                              step={1}
                              disabled={regionBusy}
                              value={regionCoordinates[key]}
                              onChange={(event) =>
                                setRegionCoordinates((previous) => ({
                                  ...previous,
                                  [key]: event.target.value,
                                }))
                              }
                              onBlur={() => snapRegionCoordinate(key)}
                            />
                          </div>
                        ))}
                      </div>
                      <p className="region-size">
                        {regionBounds
                          ? `${regionBounds.maxX - regionBounds.minX + 1} × ${regionBounds.maxZ - regionBounds.minZ + 1} Chunks`
                          : "No region selected"}{" "}
                        · full world height
                      </p>
                      <div className="region-move">
                        <Switch
                          id="region-move"
                          checked={regionMove}
                          disabled={regionBusy}
                          onCheckedChange={setRegionMove}
                        />
                        <Label htmlFor="region-move">
                          Move out of source world
                        </Label>
                      </div>
                      <p>
                        {regionMove
                          ? "Keep only in the new world."
                          : "Create an additional copy."}
                      </p>
                      {regionError && (
                        <p role="alert" className="region-error">
                          {regionError}
                        </p>
                      )}
                      <Button
                        type="submit"
                        disabled={
                          regionBusy ||
                          !regionName.trim() ||
                          !requestedRegionBounds ||
                          requestedRegionBounds.minX >
                            requestedRegionBounds.maxX ||
                          requestedRegionBounds.minZ >
                            requestedRegionBounds.maxZ
                        }
                      >
                        {regionBusy ? (
                          <LoaderCircle className="spin" />
                        ) : (
                          <Check />
                        )}
                        Create world from selection
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        disabled={regionBusy}
                        onClick={() => setRegionEditor(false)}
                      >
                        Cancel
                      </Button>
                    </form>
                  </aside>
                )}
              </div>
            </TabsContent>
          </Tabs>
          <div className="world-map-dialog-footer">
            <span>
              {mapMode === "3d"
                ? "Drag to orbit · WASD to fly"
                : "Drag to pan · Scroll to zoom"}
            </span>
            <span className="map-session-status">
              <span className="live-dot" />
              {selectedWorld?.state === "finished"
                ? "Saved"
                : state.running
                  ? "Session running in the background"
                  : "Captured world"}
            </span>
          </div>
        </main>
      </TooltipProvider>
    );
  return (
    <TooltipProvider>
      <div className="workspace desktop-workspace">
        {projectGuard.dialog}
        <DesktopChrome
          onNew={showNewProject}
          page={startOpen ? "home" : page}
          onPage={navigateWorkspace}
          onCommand={studioCommand}
          onAccount={() => setAccountOpen(true)}
          onSettings={() => {
            setPreferenceCategory("appearance");
            setSettingsOpen(true);
          }}
          onLogs={() => setLogsOpen((v) => !v)}
          onHelp={() => setHelpOpen(true)}
          onAbout={() => setAboutOpen(true)}
          onUpdates={() => {
            setPreferenceCategory("version");
            setSettingsOpen(true);
          }}
          onTheme={setTheme}
          theme={theme}
          account={state.loggedIn ? state.account || "Microsoft account" : ""}
          running={state.running}
          status={
            state.running
              ? (connectionNames[state.connection] ?? "Session running")
              : "Ready"
          }
          logsOpen={logsOpen}
          editor={editorStatus}
        />
        <div className="main-area">
          {initError ? (
            <div className="startup-error">
              <AlertCircle />
              <h1>Desktop connection unavailable</h1>
              <p>{initError}</p>
              <Button onClick={() => void load()}>
                <RefreshCw />
                Reconnect
              </Button>
            </div>
          ) : !bootstrap ? (
            <div className="startup-error">
              <LoaderCircle className="spin" />
              <p>Loading workspace …</p>
            </div>
          ) : page === "render" || page === "cinematic" ? (
            <Suspense
              fallback={
                <div className="startup-error">
                  <LoaderCircle className="spin" />
                  Loading editor …
                </div>
              }
            >
              <RenderEditor
                key={page + (project?.id || "default") + studioGeneration}
                project={project?.page === page ? project : undefined}
                loadedProject={openedStudio}
                onStudioSession={registerStudio}
                onStudioSaved={setOpenedStudio}
                onStudioOpen={() => studioCommand("open-project")}
                requestTransition={projectGuard.run}
                onProjectSourceChange={(worldId, sourceKind) => {
                  if (
                    project &&
                    (project.worldId !== worldId ||
                      project.sourceKind !== sourceKind)
                  ) {
                    const next = { ...project, worldId, sourceKind };
                    setProject(next);
                    setRecent(rememberProject(next));
                  }
                }}
                cinematic={page === "cinematic"}
                worlds={state.worlds}
                activeWorldId={state.activeWorldId}
                onEditorStateChange={setEditorStatus}
              />
            </Suspense>
          ) : page === "settings" ? null : (
            <div className={`page-body asset-workspace asset-${page}`}>
              <div className="page-heading">
                <div>
                  <h1>{names[page]}</h1>
                  <p>{descriptions[page]}</p>
                </div>
                <div className="heading-icon">
                  {page === "worlds" ? (
                    <Globe2 />
                  ) : page === "packs" ? (
                    <Package />
                  ) : (
                    <Users />
                  )}
                </div>
              </div>
              <div className="configuration-grid">
                <section className="configuration">
                  <div className="section-heading">
                    <Server size={18} />
                    <h2>Select source</h2>
                  </div>
                  <p className="section-description">
                    Enter a server address or select an available world.
                  </p>
                  <Label htmlFor="server-address">Server address</Label>
                  <div className="address-row">
                    <Input
                      id="server-address"
                      placeholder="e.g. mco.cubecraft.net:19132"
                      value={address}
                      disabled={state.running}
                      onChange={(e) => {
                        setAddress(e.target.value);
                        setSelected(undefined);
                      }}
                    />
                    <Button
                      variant="outline"
                      size="icon"
                      aria-label="Open capture"
                      disabled={state.running}
                      onClick={() =>
                        void action(async () => {
                          const file = await api.PickFile();
                          if (file) {
                            setAddress(file);
                            setSelected(undefined);
                          }
                        })
                      }
                    >
                      <FolderOpen />
                    </Button>
                  </div>
                  {selected && (
                    <p className="source-selected">
                      <Check size={14} />
                      {selected.name}
                    </p>
                  )}
                  <div className="source-choices">
                    {Object.entries(sourceNames).map(([kind, name]) => (
                      <Button
                        key={kind}
                        variant="outline"
                        disabled={state.running}
                        onClick={() => {
                          setSourceKind(kind);
                          setSourceOpen(true);
                        }}
                      >
                        {kind === "realms" ? (
                          <Globe2 />
                        ) : kind === "worlds" ? (
                          <Users />
                        ) : (
                          <Radio />
                        )}
                        {name}
                      </Button>
                    ))}
                  </div>
                  <div className="settings-divider" />
                  <div className="section-heading">
                    <SlidersHorizontal size={18} />
                    <h2>What would you like to save?</h2>
                  </div>
                  <div className="options-list">
                    {options
                      .filter(
                        (o) =>
                          primaryFlags.has(o.flag) && o.type !== "connectInfo",
                      )
                      .map(field)}
                    {!options.some((o) => primaryFlags.has(o.flag)) && (
                      <p className="section-description">
                        Default options for {names[page]} are already configured.
                      </p>
                    )}
                  </div>
                  <button
                    className="advanced-toggle"
                    aria-expanded={advanced}
                    onClick={() => setAdvanced((v) => !v)}
                  >
                    <SlidersHorizontal size={16} />
                    Advanced options
                    <span>
                      {advanced ? (
                        <ChevronDown size={16} />
                      ) : (
                        <ChevronRight size={16} />
                      )}
                    </span>
                  </button>
                  {advanced && (
                    <div className="advanced-options">
                      {options
                        .filter(
                          (o) =>
                            !primaryFlags.has(o.flag) &&
                            o.type !== "connectInfo",
                        )
                        .map(field)}
                    </div>
                  )}
                </section>
                <aside className="summary-rail">
                  <div className="summary-footer">
                    {state.running ? (
                      <Button
                        className="start-button"
                        variant="destructive"
                        disabled={state.stopping}
                        onClick={() => void action(() => api.StopCommand())}
                      >
                        {state.stopping ? (
                          <LoaderCircle className="spin" />
                        ) : (
                          <Square />
                        )}
                        {state.stopping ? "Stopping …" : "End session"}
                      </Button>
                    ) : (
                      <Button
                        className="start-button"
                        disabled={!ready}
                        onClick={() => void start()}
                      >
                        {busy ? <LoaderCircle className="spin" /> : <Play />}
                        Start download
                      </Button>
                    )}
                    <p>
                      {!address
                        ? "Select a source first."
                        : state.running
                          ? "You can keep playing Minecraft during the download."
                          : "Minecraft will then connect through the local proxy."}
                    </p>
                  </div>
                </aside>
              </div>
              <AssetLibrary
                key={page}
                page={page}
                worlds={state.worlds}
                packs={state.packs}
                skins={state.skins}
                onOpenEditor={(world, page) =>
                  launchProject({
                    ...projectDefaults,
                    id: crypto.randomUUID(),
                    name: world.name,
                    page,
                    worldId: world.id,
                    updated: Date.now(),
                  })
                }
                onFolder={(path) =>
                  void action(() => api.OpenResultFolder(path))
                }
                onMap={() => void action(() => api.OpenMapWindow(theme))}
                onSaveSkin={(skin) =>
                  void action(() => api.SaveRenderPNG(skin.name, skin.image))
                }
                onSource={() => {
                  setSourceKind("featured");
                  setSourceOpen(true);
                }}
              />
            </div>
          )}
          <div
            className={`logs-drawer ${logsOpen ? "expanded" : ""}`}
            style={logsOpen ? { height: consoleHeight } : undefined}
          >
            {logsOpen && (
              <PanelResizeHandle
                label="Console height"
                value={consoleHeight}
                min={140}
                max={Math.max(160, Math.floor(window.innerHeight * 0.6))}
                onChange={setConsoleHeight}
              />
            )}
            <div className="logs-heading">
              <button onClick={() => setLogsOpen((v) => !v)}>
                <Terminal size={16} />
                <strong>Console</strong>
                <span>{logs.length} entries</span>
                <ChevronDown className={logsOpen ? "rotate" : ""} size={16} />
              </button>
              {logsOpen && (
                <div className="console-session-tab">
                  <Terminal size={12} />
                  <span>
                    {state.running
                      ? names[state.command] || "Session"
                      : "CraftBox"}
                  </span>
                  <i className={state.running ? "active" : ""} />
                  <button
                    aria-label="Close console"
                    onClick={() => setLogsOpen(false)}
                  >
                    ×
                  </button>
                </div>
              )}
              {logsOpen && (
                <div className="log-actions">
                  <Input
                    aria-label="Search logs"
                    placeholder="Search logs …"
                    value={logSearch}
                    onChange={(e) => setLogSearch(e.target.value)}
                  />
                  <select
                    aria-label="Log level"
                    value={logLevel}
                    onChange={(e) => setLogLevel(e.target.value)}
                  >
                    <option value="all">All levels</option>
                    {["error", "warning", "info", "debug"].map((level) => (
                      <option key={level}>{level}</option>
                    ))}
                  </select>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Copy logs"
                    onClick={() =>
                      void copy(
                        visibleLogs
                          .map(
                            (log) =>
                              `${log.time} ${log.level} ${log.source} ${log.message}`,
                          )
                          .join("\n"),
                      )
                    }
                  >
                    <Copy />
                  </Button>
                  <Button variant="ghost" onClick={() => setLogs([])}>
                    Clear
                  </Button>
                </div>
              )}
            </div>
            {logsOpen && (
              <div className="log-content">
                {visibleLogs.length ? (
                  visibleLogs.map((log, index) => (
                    <div className={`log-line ${log.level}`} key={index}>
                      <time>
                        {Number.isNaN(Date.parse(log.time))
                          ? log.time
                          : new Date(log.time).toLocaleTimeString("en-US")}
                      </time>
                      <span className="log-level">{log.level}</span>
                      <span>
                        {log.source && <b>{log.source}: </b>}
                        {log.message}
                      </span>
                    </div>
                  ))
                ) : (
                  <p>No matching log entries.</p>
                )}
              </div>
            )}
          </div>
        </div>
        <footer className="application-statusbar">
          <span>
            {bootstrap?.version ?? "CraftBox"}
            <i />
            Minecraft {bootstrap?.minecraft ?? "—"}
          </span>
          <span>
            {state.worlds.length} Worlds
            <i />
            {state.packs.length} Packs
            <i />
            {state.skins.length} Skins
          </span>
        </footer>
      </div>
      <PreferencesDialog
        initialCategory={preferenceCategory}
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        theme={theme}
        onTheme={setTheme}
        version={bootstrap?.version}
        minecraft={bootstrap?.minecraft}
        updateBusy={updateBusy}
        updateVersion={state.updateVersion}
        updateProgress={state.updateProgress}
        onCheckUpdate={() =>
          void action(async () => {
            setUpdateBusy(true);
            try {
              await api.CheckUpdate();
            } finally {
              setUpdateBusy(false);
            }
          })
        }
        onInstallUpdate={
          state.running
            ? undefined
            : () =>
                void action(async () => {
                  setUpdateBusy(true);
                  try {
                    await api.InstallUpdate();
                  } finally {
                    setUpdateBusy(false);
                  }
                })
        }
      />
      <StartDialog
        open={startOpen && !!bootstrap && !initError}
        onOpenChange={setStartOpen}
        recent={recent}
        worlds={state.worlds}
        onCreate={showNewProject}
        onOpen={openFromStart}
        onOpenProject={() => studioCommand("open-project")}
        onResume={launchProject}
        onPage={navigateWorkspace}
      />
      <NewProjectDialog
        open={newProjectOpen}
        onOpenChange={changeNewProjectOpen}
        worlds={state.worlds}
        onCreate={launchProject}
      />
      <StudioSupportDialogs
        helpOpen={helpOpen}
        onHelpOpen={setHelpOpen}
        aboutOpen={aboutOpen}
        onAboutOpen={setAboutOpen}
        version={bootstrap?.version}
        minecraft={bootstrap?.minecraft}
      />
      <Dialog open={sourceOpen} onOpenChange={setSourceOpen}>
        <DialogContent className="source-dialog">
          <DialogHeader>
            <DialogTitle>Select source</DialogTitle>
            <DialogDescription>
              Select the server or world to download from.
            </DialogDescription>
          </DialogHeader>
          <div className="source-tabs" role="tablist" aria-label="Source type">
            {Object.entries(sourceNames).map(([key, name]) => (
              <Button
                key={key}
                role="tab"
                aria-selected={sourceKind === key}
                variant={sourceKind === key ? "default" : "ghost"}
                onClick={() => setSourceKind(key)}
              >
                {name}
              </Button>
            ))}
          </div>
          <div className="search-field">
            <Search />
            <Input
              aria-label="Browse sources"
              placeholder="Search by name …"
              value={sourceSearch}
              onChange={(e) => setSourceSearch(e.target.value)}
            />
          </div>
          <div className="source-results">
            {sourceLoading ? (
              <div className="dialog-empty">
                <LoaderCircle className="spin" />
                <p>Loading sources …</p>
              </div>
            ) : sourceError ? (
              <div className="dialog-empty">
                <AlertCircle />
                <p>{sourceError}</p>
                {!state.loggedIn ? (
                  <Button
                    onClick={() => {
                      setSourceOpen(false);
                      setAccountOpen(true);
                    }}
                  >
                    <LogIn />
                    Sign in
                  </Button>
                ) : (
                  <Button variant="outline" onClick={() => void loadSources()}>
                    <RefreshCw />
                    Try again
                  </Button>
                )}
              </div>
            ) : (
              sources
                .filter((source) =>
                  `${source.name} ${source.description}`
                    .toLowerCase()
                    .includes(sourceSearch.toLowerCase()),
                )
                .map((source) => (
                  <button
                    className="source-result"
                    disabled={!source.available}
                    key={source.id}
                    onClick={() => {
                      setSelected(source);
                      setAddress(source.address);
                      setSourceOpen(false);
                    }}
                  >
                    <div className="source-result-icon">
                      <Globe2 />
                    </div>
                    <div>
                      <strong>{source.name}</strong>
                      <span>{source.description || source.address}</span>
                    </div>
                    {source.available ? (
                      <ChevronRight size={17} />
                    ) : (
                      <Badge variant="outline">Unavailable</Badge>
                    )}
                  </button>
                ))
            )}
            {!sourceLoading && !sourceError && !sources.length && (
              <div className="dialog-empty">
                <Globe2 />
                <p>No sources are currently available here.</p>
                <Button variant="outline" onClick={() => void loadSources()}>
                  Refresh
                </Button>
              </div>
            )}
          </div>
          <DialogFooter>
            <span className="dialog-footer-note">
              {sourceNames[sourceKind]} · {sources.length} Sources
            </span>
            <Button variant="outline" onClick={() => setSourceOpen(false)}>
              Cancel
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={accountOpen || Boolean(state.authCode)}
        onOpenChange={(open) => {
          setAccountOpen(open);
          if (!open && state.authenticating)
            void action(() => api.CancelLogin());
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {state.authCode ? "Microsoft sign-in" : "Your account"}
            </DialogTitle>
            <DialogDescription>
              Sign in with your Microsoft account to access Realms, events and friends' worlds.
            </DialogDescription>
          </DialogHeader>
          {state.authCode ? (
            <div className="auth-flow">
              <p>Open the Microsoft sign-in page and enter this code:</p>
              <div className="auth-code">
                {state.authCode}
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Copy sign-in code"
                  onClick={() => void copy(state.authCode)}
                >
                  {copied ? <Check /> : <Copy />}
                </Button>
              </div>
              <Button
                onClick={() => void action(() => api.OpenURL(state.authURI))}
              >
                Open Microsoft sign-in page
                <ArrowUpRight />
              </Button>
              <p>
                <LoaderCircle size={15} className="spin" />
                Waiting for confirmation in your browser …
              </p>
              <Button
                variant="ghost"
                onClick={() => void action(() => api.CancelLogin())}
              >
                Cancel sign-in
              </Button>
            </div>
          ) : (
            <div className="account-settings">
              <div className="account-identity">
                <Users size={30} />
                <div>
                  <h3>Microsoft account</h3>
                  <p>
                    {state.loggedIn
                      ? `Signed in as ${state.account || "Default account"}`
                      : "Sign in to use Realms, events and friends' worlds."}
                  </p>
                </div>
              </div>
              <Label htmlFor="account-name">Account profile</Label>
              <Input
                id="account-name"
                value={accountName}
                placeholder="Default account"
                onChange={(e) => setAccountName(e.target.value)}
                disabled={state.running || state.authenticating}
              />
              <p className="secondary">
                Use a unique profile name to keep saved accounts separate.
              </p>
              <Button
                variant="outline"
                disabled={state.running || state.authenticating}
                onClick={() =>
                  void action(() => api.SwitchAccount(accountName))
                }
              >
                <Users />
                Switch account
              </Button>
              <div className="settings-divider" />
              {state.loggedIn ? (
                <>
                  <p className="account-status">
                    <Check />
                    Signed in as {state.account || "Default account"}
                  </p>
                  <Button
                    variant="outline"
                    disabled={state.running || state.authenticating}
                    onClick={() => void action(() => api.Logout())}
                  >
                    <LogOut />
                    Sign out
                  </Button>
                </>
              ) : (
                <Button
                  disabled={state.running || state.authenticating}
                  onClick={() => void action(() => api.Login())}
                >
                  {state.authenticating ? (
                    <LoaderCircle className="spin" />
                  ) : (
                    <LogIn />
                  )}
                  {state.authenticating
                    ? "Preparing sign-in …"
                    : "Sign in with Microsoft"}
                </Button>
              )}
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setAccountOpen(false);
                if (state.authenticating) void action(() => api.CancelLogin());
              }}
            >
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(error || state.error)}
        onOpenChange={(open) => {
          if (!open) {
            setError("");
            void api.DismissError();
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Could not complete this action</DialogTitle>
            <DialogDescription>
              Use the details below to check your connection or input.
            </DialogDescription>
          </DialogHeader>
          <div className="error-body">
            <p>
              {/https?:|http |status|stack|exception|\n/i.test(
                error || state.error,
              ) || (error || state.error || "").length > 180
                ? "Check your connection and input. You can copy the technical details and try again."
                : error || state.error}
            </p>
            {(/https?:|http |status|stack|exception|\n/i.test(
              error || state.error,
            ) ||
              (error || state.error || "").length > 180) && (
              <details>
                <summary>Show technical details</summary>
                <pre className="error-details">{error || state.error}</pre>
              </details>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => void copy(error || state.error)}
            >
              <Copy />
              Copy details
            </Button>
            <Button
              onClick={() => {
                setError("");
                void api.DismissError();
              }}
            >
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  );
}

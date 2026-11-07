import { requestSessionQuit } from "./session-quit";
import FeedbackFlyout from "./FeedbackFlyout";
import DocumentWorkspaces, {
  type DocumentWorkspacesHandle,
} from "./DocumentWorkspaces";
import ActionErrorDialog from "./ActionErrorDialog";
import MinecraftJoinStatus from "./MinecraftJoinStatus";
import NotificationCenter from "./NotificationCenter";
import {selectionRoots} from "./collection-selection";
import { useMinecraftJoin } from "./use-minecraft-join";
import {connectionValues} from "./connection-settings";
import { notify, notifyAction, notifySaved } from "./StudioFlyouts";
import FirstStartDialog, {
  accountOnboardingComplete,
  rememberAccountOnboarding,
} from "./FirstStartDialog";
import { useCaptureFlyouts } from "./use-capture-flyouts";
import WorldExportSettingsDialog from "./WorldExportSettingsDialog";
import WorldCopyDialog from "./WorldCopyDialog";
import StudioForm from "@/StudioForm";
import { useLocalPathChecks } from "./path-validation";
import { profileNameError, rangeOrderError } from "./input-rules";
import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";

import { configureSceneBudget } from "./scene-state";
import { useWorkspacePreferences } from "./workspace-preferences";
import { useReleaseUpdates } from "./use-release-updates";
import { parseStudioDocument, type StudioDocument } from "./studio-document";
import type { RenderWorld } from "./bridge";
import type { EditorCommand } from "./editor-commands";
import {
  ChevronDown,
  ChevronRight,
  ArrowUpRight,
  FolderOpen,
  Play,
  Square,
  Server,
  Copy,
  Check,
  LogIn,
  LogOut,
  RefreshCw,
  SlidersHorizontal,
  File,
  LoaderCircle,
  AlertCircle,
  Map as MapIcon,
  Flag,
  X,
  Pencil,
} from "lucide-react";
import { AccountIcon, SourceKindIcon } from "./studio-icons";
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
import WorldComposer from "./WorldComposer";
import {
  dispatchWorkspaceAction,
  type WorkspaceAction,
} from "./workspace-actions";
import DesktopChrome from "./DesktopChrome";
import AssetLibrary from "./AssetLibrary";
import AccountIdentity from "./AccountIdentity";
import StudioSupportDialogs from "./StudioSupportDialogs";
import ConsoleDock from "./ConsoleDock";
import ExportTasks from "./ExportTasks";
import "./world-capture-tabs.css";
import SourcePicker, { sourceNames } from "./SourcePicker";
import PreferencesDialog, {
  loadWorkspacePreferences,
  applyWorkspacePreferences,
} from "./PreferencesDialog";
import { StartDialog, NewProjectDialog } from "./StartScreen";
import {
  loadRecent,
  rememberProject,
  newProjectDefaults,
  withProjectSource,
  withProjectAddress,
  manageRecentProject,
  syncRecentProjectSources,
  type RecentProjectAction,
  type StudioProject,
} from "./start-project";
import { dispatchEditorCommand, idleEditor } from "./editor-commands";
const WorldScene = lazy(() => import("./WorldScene"));

const Marketplace = lazy(() => import("./Marketplace"));
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
const tileKey = (tile: Tile) => `${tile.worldId ?? ""}:${tile.x},${tile.z}`;

const names: Record<string, string> = {
  worlds: "Worlds",
  packs: "Resource packs",
  skins: "Player skins",
  packets: "Packet Inspector",
  marketplace: "Marketplace",
  render: "Render editor",
  cinematic: "Cinematic editor",
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
  capture: "Save a replay.",
  "client-cache": "Use Minecraft cache.",
  void: "Empty uncaptured terrain.",
  image: "Export a PNG map.",
  "save-entities": "Include mobs and entities.",
  "save-custom-entities": "Custom models and textures. Off by default.",
  "save-players": "Player models. Off by default.",
  "save-inventories": "Include inventories and containers.",
  "block-updates": "Capture block changes.",
  "entity-culling": "Remove dead or despawned entities. Experimental.",
  listen: "Address and port for the local Minecraft connection.",
  "exclude-mobs": "Comma-separated mob names.",
  "chunk-radius": "View distance in chunks.",
  script: "Optional session script.",
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
  useEffect(() => {
    const report = (value: unknown, label: string) => {
      const error = value instanceof Error ? value : undefined;
      void Promise.resolve()
        .then(() =>
          api.ReportFrontendError(
            error?.message || String(value || label),
            error?.stack || label,
          ),
        )
        .catch(() => {});
    };
    const error = (event: ErrorEvent) =>
      report(event.error || event.message, "window.error");
    const rejection = (event: PromiseRejectionEvent) =>
      report(event.reason, "unhandledrejection");
    window.addEventListener("error", error);
    window.addEventListener("unhandledrejection", rejection);
    return () => {
      window.removeEventListener("error", error);
      window.removeEventListener("unhandledrejection", rejection);
    };
  }, []);
  const [worldExportID, setWorldExportID] = useState<string>();
  const [worldCopyID, setWorldCopyID] = useState<string>();
  const workspacePrefs = useWorkspacePreferences();
  const [deletingWorld, setDeletingWorld] = useState<{
    id: string;
    name: string;
    others?: {id:string;name:string}[];
  }>();
  const [deleteBusy, setDeleteBusy] = useState(false),
    [deleteError, setDeleteError] = useState("");
  useEffect(() => applyWorkspacePreferences(loadWorkspacePreferences()), []);
  const [regionEditor, setRegionEditor] = useState(false);
  const [assetCaptureOpen, setAssetCaptureOpen] = useState(false);
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
  const [compositionOpen, setCompositionOpen] = useState(false);
  const [sceneCentered, setSceneCentered] = useState(false);
  const [mapSceneFocus, setMapSceneFocus] = useState<{
    worldId: string;
    point: [number, number, number];
  }>();
  const worldSessionActive = useRef(false),
    minecraftOpened = useRef<number|undefined>(undefined);
  const [bootstrap, setBootstrap] = useState<Bootstrap>(),
    [state, setState] = useState<State>(emptyState),
    [page, setPage] = useState(() => {
      const prefs = loadWorkspacePreferences();
      const last = localStorage.getItem("craftbox-last-workspace");
      return prefs.rememberWorkspace &&
        [
          "worlds",
          "packs",
          "skins",
          "packets",
          "marketplace",
          "render",
          "cinematic",
        ].includes(last || "")
        ? last!
        : prefs.startupPage;
    });
  const releaseUpdates = useReleaseUpdates(!!bootstrap && !bootstrap.mapWindow);
  useCaptureFlyouts(state.worlds, state.packs, !bootstrap?.mapWindow);
  useEffect(() => {
    try {
      localStorage.setItem("craftbox-last-workspace", page);
    } catch {}
  }, [page]);
  useEffect(() => {
    configureSceneBudget(workspacePrefs.sceneBudgetMB * 1024 * 1024);
  }, [bootstrap, workspacePrefs.sceneBudgetMB]);
  const [startOpen, setStartOpen] = useState(
    () => loadWorkspacePreferences().showWelcome,
  );
  const [completedAccountIntroScope, setCompletedAccountIntroScope] = useState<
    string | null
  >(null);
  const linkingAccountIntro = useRef(false);
  const accountIntroScope = bootstrap?.installationId ?? "";
  const accountIntroComplete =
    completedAccountIntroScope === accountIntroScope ||
    accountOnboardingComplete(accountIntroScope);
  const finishAccountIntro = () => {
    rememberAccountOnboarding(accountIntroScope);
    setCompletedAccountIntroScope(accountIntroScope);
    linkingAccountIntro.current = false;
    setStartOpen(true);
  };
  const needsAccountIntro =
    !!bootstrap && !bootstrap.mapWindow && !accountIntroComplete;
  useEffect(() => {
    if (
      bootstrap &&
      !bootstrap.mapWindow &&
      linkingAccountIntro.current &&
      state.loggedIn &&
      !accountIntroComplete
    )
      finishAccountIntro();
  }, [bootstrap, state.loggedIn, accountIntroComplete, accountIntroScope]);
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
  function workspaceAction(next: string, action: WorkspaceAction) {
    projectGuard.run(() => {
      setStartOpen(false);
      setPage(next);
      dispatchWorkspaceAction(next, action);
    });
  }
  function showNewProject() {
    if (page === "structures") { dispatchWorkspaceAction("structures", "new"); return; }
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
  useEffect(() => {
    const packets = () => navigateWorkspace("packets");
    const marketplace = () => navigateWorkspace("marketplace");
    const content = (e: Event) => {
      const kind = (e as CustomEvent).detail?.kind;
      if (["worlds", "packs", "skins"].includes(kind)) navigateWorkspace(kind);
    };
    window.addEventListener("craftbox:open-marketplace", marketplace);
    window.addEventListener("craftbox:open-packets", packets);
    window.addEventListener("craftbox:open-content", content);
    return () => {
      window.removeEventListener("craftbox:open-marketplace", marketplace);
      window.removeEventListener("craftbox:open-packets", packets);
      window.removeEventListener("craftbox:open-content", content);
    };
  });
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
  const [quitOpen, setQuitOpen] = useState(false);
  useEffect(() => on("desktop:close-requested", () => setQuitOpen(true)), []);
  const documentWorkspaces = useRef<DocumentWorkspacesHandle>(null);
  const projectGuard = { run: (action: () => void) => action(), dialog: null };
  async function openStudio() {
    if (page === "structures") { dispatchWorkspaceAction("structures", "import-files"); return; }
    let pendingId: string | undefined;
    try {
      const result = await api.OpenStudioProject();
      if (!result.path) return;
      if(result.structure){documentWorkspaces.current?.openStructure(result.structure.id,result.structure.name);setStartOpen(false);return;}
      if (result.workspaceId) {
        documentWorkspaces.current?.openWorkspace(
          result.document,
          result.workspaceId,
          result.path,
        );
        setStartOpen(false);
        return;
      }
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
        } catch {}
      }
      setError(String(e));
    }
  }
  function studioCommand(command: EditorCommand) {
    if (command === "open-project") projectGuard.run(() => void openStudio());
    else if (command === "save-project" || command === "save-project-as")
      void documentWorkspaces.current
        ?.save(command === "save-project-as")
        .catch((e) => setError(String(e)));
    else if (command === "open-archive" || command === "open-folder")
      documentWorkspaces.current?.openWorld(
        command === "open-folder" ? "folder" : "archive",
      );
    else dispatchEditorCommand(command);
  }
  function launchProject(next: StudioProject, chooseSource = false) {
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
      const named = withProjectSource(
        available,
        available.worldId,
        state.worlds.find((w) => w.id === available.worldId)?.name,
      );
      const entries = rememberProject({ ...named, worldId: next.worldId });
      setProject({
        ...named,
        name: entries.find((p) => p.id === named.id)?.name ?? named.name,
      });
      setStudioGeneration((n) => n + 1);
      setRecent(entries);
      if (!["render", "cinematic"].includes(next.page))
        setAddress(next.address || "");
      setPage(next.page);
      if (chooseSource && ["worlds", "packs", "skins"].includes(next.page)) {
        setSourceKind("custom");
        setSourceOpen(true);
      }
    });
  }
  function manageProject(id: string, action: RecentProjectAction) {
    try {
      const next = manageRecentProject(id, action);
      setRecent(next);
      const current = next.find((p) => p.id === project?.id);
      if (current && id === project?.id) setProject(current);
    } catch (e) {
      setError(errorText(e));
    }
  }
  function openFromStart(kind: "archive" | "folder") {
    launchProject({
      ...newProjectDefaults(),
      id: crypto.randomUUID(),
      name: "",
      nameMode: "auto",
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
    [sourceKind, setSourceKind] = useState("custom"),
    [sourceOpen, setSourceOpen] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false),
    [accountName, setAccountName] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [initError, setInitError] = useState("");
  const [logsOpen, setLogsOpen] = useState(false),
    [logs, setLogs] = useState<Log[]>([]),
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
  const hydrationSequence = useRef(0);
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
      configureSceneBudget(data.sceneBudgetBytes);
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
  const captureLaunchSettings=useRef<{captureID?:number;openMinecraft:boolean;openMap:boolean}>({openMinecraft:true,openMap:true});
  const minecraftJoin = useMinecraftJoin(state, !!bootstrap && !bootstrap.mapWindow, api.OpenMinecraft);
  useEffect(() => {
    if (!bootstrap) return;
	if(state.running&&captureLaunchSettings.current.captureID!==state.captureID){captureLaunchSettings.current={captureID:state.captureID,openMinecraft:workspacePrefs.autoOpenMinecraft,openMap:workspacePrefs.autoOpenCaptureMap};worldSessionActive.current=false;}
    const active = state.running && state.command === "worlds";
    if (active && !worldSessionActive.current && !bootstrap?.mapWindow && captureLaunchSettings.current.openMap)
      void action(() => api.OpenMapWindow(theme));
    worldSessionActive.current = active;
    if (!state.running) {
      minecraftOpened.current = undefined;
	  captureLaunchSettings.current.captureID=undefined;
      return;
    }
    if (
      !bootstrap?.mapWindow &&
	  captureLaunchSettings.current.openMinecraft &&
      state.connection === 2 &&
	  minecraftJoin.canJoin &&
	  state.captureID !== undefined &&
      state.listenAddr &&
      minecraftOpened.current !== state.captureID
    ) {
      minecraftOpened.current = state.captureID;
      void minecraftJoin.join().catch((error) => setError(errorText(error)));
    }
  }, [
    state.running,
    state.command,
    state.connection,
    state.listenAddr,
	state.minecraftJoinReady,
	minecraftJoin.canJoin,
	state.captureID,
	state.stopping,
	workspacePrefs.autoOpenMinecraft,
	workspacePrefs.autoOpenCaptureMap,
    bootstrap?.mapWindow,
  ]);
	const lastJoinNotice=useRef("");
	useEffect(()=>{
		if(!state.running){lastJoinNotice.current="";return}
		if(bootstrap?.mapWindow||!state.minecraftJoinAddress)return;
		const key=state.expectedJoinAddress+"|"+state.minecraftJoinAddress;
		if(lastJoinNotice.current===key)return;lastJoinNotice.current=key;
		notify({id:"minecraft-join-hostname",tone:state.joinHostnameMatches?"info":"warning",title:state.joinHostnameMatches?"Minecraft reports the destination hostname":"Minecraft used a different address",description:state.minecraftJoinAddress+(state.joinHostnameMatches?"":" · Reconnect through CraftBox using "+state.expectedJoinAddress+" instead of a saved localhost or LAN entry.")});
	},[state.running,state.minecraftJoinAddress,state.expectedJoinAddress,state.joinHostnameMatches,bootstrap?.mapWindow]);
  useEffect(() => {
    if (state.activeWorldId) setSelectedWorldID(state.activeWorldId);
  }, [state.activeWorldId]);
  useEffect(() => {
    const next = syncRecentProjectSources(state.worlds);
    if (JSON.stringify(next) !== JSON.stringify(recent)) setRecent(next);
    if (project) {
      const source = state.worlds.find((w) => w.id === project.worldId);
      if (source && source.name !== project.sourceName)
        setProject(
          next.find((p) => p.id === project.id) ??
            withProjectSource(project, project.worldId, source.name),
        );
    }
  }, [state.worlds]);
  useEffect(() => {
    if (state.worlds.length === 0) setSelectedWorldID("");
  }, [state.worlds.length]);
  const viewWorldID = state.worlds.some((w) => w.id === selectedWorldID)
    ? selectedWorldID
    : state.worlds.some((w) => w.id === state.activeWorldId)
      ? (state.activeWorldId ?? "")
      : (state.worlds.at(-1)?.id ?? "");
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
  const visibleTiles = useMemo(
    () =>
      new globalThis.Map(
        [...tiles].filter(([, tile]) => (tile.worldId ?? "") === viewWorldID),
      ),
    [tiles, viewWorldID],
  );
  async function action(fn: () => Promise<unknown>) {
    try {
      await fn();
    } catch (e) {
      setError(errorText(e));
      void Promise.resolve()
        .then(() =>
          api.ReportFrontendError(
            errorText(e),
            e instanceof Error
              ? e.stack || "Handled workspace action"
              : "Handled workspace action",
          ),
        )
        .catch(() => {});
    }
  }
  const cmd = bootstrap?.commands.find((c) => c.name === page),
    options = cmd?.options ?? [],
    current = values[page] ?? {};
  const pathOptions = options.filter((o) => o.fileType?.startsWith("file"));
  const pathChecks = useLocalPathChecks(
    pathOptions.map((o) => ({
      id: `option-${page}-${o.flag}`,
      value: current[o.flag] ?? o.default,
      kind: "file",
    })),
    !state.running,
  );
  const pathsReady = Object.values(pathChecks).every(
    (p) => !p.pending && !p.error,
  );
  const hasAddress = options.some((o) => o.type === "connectInfo"),
    ready = Boolean(
      bootstrap &&
      !state.running &&
      !state.authenticating &&
      !busy &&
      pathsReady &&
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
        values: {
          ...connectionValues(options,current,workspacePrefs),
          ...(page === "packets" ? { capture: "false" } : {}),
        },
      });
      if (project && project.page === page) {
        const next = withProjectAddress(project, address, selected?.name);
        const entries = rememberProject(next);
        setProject(entries.find((p) => p.id === next.id) ?? next);
        setRecent(entries);
      }
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      notifyAction("Copied to clipboard");
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch (e) {
      setError(errorText(e));
    }
  }
  function field(option: Option) {
    const title = optionLabels[option.flag] ?? option.name ?? option.flag,
      id = `option-${page}-${option.flag}`,
      value = (option.flag==="listen" ? connectionValues(options,current,workspacePrefs).listen : current[option.flag]) ?? option.default;
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
              step={option.type === "int" ? 1 : undefined}
              placeholder={option.default || title}
              pending={pathChecks[id]?.pending}
              error={pathChecks[id]?.error}
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
  const deleteDialog = (
    <Dialog
      open={!!deletingWorld}
      onOpenChange={(open) => {
        if (!open && !deleteBusy) setDeletingWorld(undefined);
      }}
    >
      <DialogContent
        resizable={false}
        tone="warning"
        className="capture-delete-dialog"
        showCloseButton={!deleteBusy}
      >
        <DialogHeader>
          <DialogTitle>{deletingWorld?.others?.length ? `Delete ${deletingWorld.others.length+1} captures?` : "Delete capture?"}</DialogTitle>
          <DialogDescription>{deletingWorld?.name}</DialogDescription>
        </DialogHeader>
        <div className="capture-delete-body">
          <strong>{deletingWorld?.name}</strong>
          {deletingWorld?.others?.map(world=><p key={world.id}>{world.name}</p>)}
          <p>Deletes this recording and its files.</p>
          {(deletingWorld?.id === state.activeWorldId || deletingWorld?.others?.some(world=>world.id===state.activeWorldId)) && (
            <p>Capture resumes in the next world. The connection stays open.</p>
          )}
          {state.worlds.some((w) => w.parentId === deletingWorld?.id) && (
            <p>Includes all child regions.</p>
          )}
          {deleteError && <FeedbackFlyout title="CraftBox">{deleteError}</FeedbackFlyout>}
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            disabled={deleteBusy}
            onClick={() => setDeletingWorld(undefined)}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={deleteBusy}
            onClick={() =>
              void (async () => {
                if (!deletingWorld) return;
                setDeleteBusy(true);
                setDeleteError("");
                const targets=[deletingWorld,...(deletingWorld.others??[])];
                let completed=0;
                try {
                  for(const target of targets){await api.DeleteCapturedWorld(target.id);completed++;}
                  setDeletingWorld(undefined);
                } catch (e) {
                  const remaining=targets.slice(completed);
                  if(remaining.length)setDeletingWorld({...remaining[0],others:remaining.slice(1)});
                  setDeleteError(errorText(e));
                } finally {
                  setDeleteBusy(false);
                }
              })()
            }
          >
            {deleteBusy ? "Deleting…" : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
  if (bootstrap?.mapWindow)
    return (
      <TooltipProvider>
        <main className="world-window" aria-label="World maps" role="region">
          <DialogHeader className="world-map-dialog-header">
            <div className="world-map-dialog-title">
              <MapIcon />
              <h1>Live world map</h1>
              <div className="map-mode-switch" role="group" aria-label="View">
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
                {connectionNames[state.connection] ?? "Preparing session"}
              </Badge>
              <Button
                variant="outline"
                disabled={!state.worlds.length}
                onClick={() => setCompositionOpen(true)}
              >
                Merge worlds…
              </Button>
            </div>
            <p className="map-window-description">Live captured terrain.</p>
          </DialogHeader>
          {compositionOpen && (
            <WorldComposer
              open
              onOpenChange={setCompositionOpen}
              worlds={state.worlds}
              currentWorldId={viewWorldID}
              selection={regionBounds}
              tiles={tiles}
              onCreated={(world) => {
                setSelectedWorldID(world.id);
                setMapMode("2d");
              }}
            />
          )}
          <Dialog
            open={!!renaming}
            onOpenChange={(open) => {
              if (!open) setRenaming(undefined);
            }}
          >
            <DialogContent resizable={false}>
              <DialogHeader>
                <DialogTitle>Rename world</DialogTitle>
                <DialogDescription>
                  This name will be saved in the world.
                </DialogDescription>
              </DialogHeader>
              <StudioForm
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
                  required
                  autoFocus
                  maxLength={80}
                  value={renaming?.name ?? ""}
                  onChange={(event) =>
                    setRenaming((old) =>
                      old ? { ...old, name: event.target.value } : old,
                    )
                  }
                />
                {regionError && <FeedbackFlyout title="CraftBox">{regionError}</FeedbackFlyout>}
              </StudioForm>
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
                  <div className="world-capture-tab" key={world.id || index}>
                    <TabsTrigger
                      key={world.id || index}
                      value={world.id || "current"}
                      data-tooltip="Double-click to rename"
                      onDoubleClick={() => {
                        setSelectedWorldID(world.id);
                        setRegionError("");
                        setRenaming({ id: world.id, name: world.name });
                      }}
                    >
                      {world.name} · {index + 1}
                    </TabsTrigger>
                    {world.id && (
                      <button
                        className="world-capture-rename"
                        type="button"
                        aria-label={`Rename capture ${world.name}`}
                        data-tooltip="Rename capture"
                        disabled={
                          !(
                            world.state === "finished" ||
                            (state.running &&
                              (world.id === state.activeWorldId ||
                                world.parentId === state.activeWorldId))
                          )
                        }
                        onClick={() => {
                          setSelectedWorldID(world.id);
                          setRegionError("");
                          setRenaming({ id: world.id, name: world.name });
                        }}
                      >
                        <Pencil size={12} />
                      </button>
                    )}
                    {world.id && (
                      <button
                        className="world-capture-close"
                        type="button"
                        aria-label={`Delete capture ${world.name}`}
                        data-tooltip="Delete capture"
                        onClick={(event) => {
                          event.stopPropagation();
                          setDeleteError("");
                          setDeletingWorld({ id: world.id, name: world.name });
                        }}
                      >
                        <X size={12} />
                      </button>
                    )}
                  </div>
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
                    fallback={<div role="status">Loading 3D view …</div>}
                  >
                    <WorldScene
                      key={`${viewWorldID}:${mapGeneration}`}
                      worldId={viewWorldID}
                      name={selectedWorld?.name ?? "Current world"}
                      player={selectedWorld?.player ?? state.player}
                      hasPlayer={selectedWorld?.hasPlayer ?? state.hasPlayer}
                      initialCentered={sceneCentered}
                      initialFocus={
                        mapSceneFocus?.worldId === viewWorldID
                          ? mapSceneFocus.point
                          : undefined
                      }
                      onCenteredChange={setSceneCentered}
                    />
                  </Suspense>
                ) : (
                  <WorldMap
					capturePoints={selectedWorld?.capturePoints}
                    key={`${viewWorldID}:${mapGeneration}`}
                    tiles={visibleTiles}
                    worldId={viewWorldID}
                    name={selectedWorld?.name ?? "World map"}
                    onRequestRegion={
                      canEditRegion
                        ? () => {
                            setRegionEditor(true);
                            setSpawnEditor(false);
                          }
                        : undefined
                    }
                    onOpen3D={(point) => {
                      setMapSceneFocus({ worldId: viewWorldID, point });
                      setSceneCentered(false);
                      setMapMode("3d");
                    }}
                    player={selectedWorld?.player ?? state.player}
                    hasPlayer={selectedWorld?.hasPlayer ?? state.hasPlayer}
                    selecting={regionEditor && !regionBusy}
                    selection={regionEditor ? regionBounds : undefined}
                    onSelectionChange={selectRegion}
                    onSelectionClear={() => {
                      setRegionBounds(undefined);
                      setRegionCoordinates({
                        minX: "",
                        maxX: "",
                        minZ: "",
                        maxZ: "",
                      });
                    }}
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
                    <StudioForm
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
                            min={i === 1 ? -32768 : -30000000}
                            max={i === 1 ? 32767 : 30000000}
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
                        <FeedbackFlyout title="CraftBox">
                          {regionError}
                        </FeedbackFlyout>
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
                    </StudioForm>
                  </aside>
                )}
                {regionEditor && (
                  <aside
                    className="region-editor"
                    aria-label="Edit world region"
                  >
                    <h2>Create a separate world</h2>
                    <p>
                      Drag a rectangle · Hold the middle mouse button to pan.
                    </p>
                    <StudioForm
                      onSubmit={(event) => {
                        event.preventDefault();
                        void createRegion();
                      }}
                    >
                      <Label htmlFor="region-name">World name</Label>
                      <Input
                        id="region-name"
                        required
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
                              min={-30000000}
                              max={30000000}
                              error={
                                key.endsWith("X")
                                  ? rangeOrderError(
                                      regionCoordinates.minX,
                                      regionCoordinates.maxX,
                                    )
                                  : rangeOrderError(
                                      regionCoordinates.minZ,
                                      regionCoordinates.maxZ,
                                    )
                              }
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
                        <FeedbackFlyout title="CraftBox">
                          {regionError}
                        </FeedbackFlyout>
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
                    </StudioForm>
                  </aside>
                )}
              </div>
            </TabsContent>
          </Tabs>
          {deleteDialog}
          <div className="world-map-dialog-footer">
            <ExportTasks/>
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
      <div
        className={`workspace desktop-workspace${page !== "worlds" ? " workbench-view" : ""}`}
        data-workspace={page}
      >
        {projectGuard.dialog}
        <DesktopChrome
          onNew={showNewProject}
          page={startOpen ? "home" : page}
          onPage={navigateWorkspace}
          onCommand={studioCommand}
          onWorkspaceAction={workspaceAction}
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
        <MinecraftJoinStatus
          phase={minecraftJoin.phase}
          address={state.listenAddr}
          launching={minecraftJoin.launching}
          onRetry={() => void action(minecraftJoin.join)}
          onCopy={(address) => void copy(address)}
        />
        <div
          className={`main-area${assetCaptureOpen && ["packs", "skins"].includes(page) ? " asset-capture-layout" : ""}`}
        >
          {bootstrap && !initError && (
            <DocumentWorkspaces
              ref={documentWorkspaces}
              page={page}
              state={state}
              project={project}
              opened={openedStudio}
              generation={studioGeneration}
              canStart={ready}
              source={selected?.name || address}
              onSource={() => {
                if (["packs", "skins"].includes(page))
                  setAssetCaptureOpen(true);
                setSourceKind("custom");
                setSourceOpen(true);
              }}
              onStart={() => void start()}
              onStop={() => void api.StopCommand()}
              onOpen={() => void openStudio()}
              onNew={showNewProject}
              onNavigate={navigateWorkspace}
              onStatus={setEditorStatus}
              onProjectChanged={(next) => {
                const entries = rememberProject(next);
                setRecent(entries);
                if (project?.id === next.id)
                  setProject(entries.find((p) => p.id === next.id) ?? next);
              }}
            />
          )}
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
          ) : page === "marketplace" ? (
            <Suspense
              fallback={
                <div className="startup-error">Loading Marketplace...</div>
              }
            >
              <Marketplace
                loggedIn={state.loggedIn}
                account={state.account}
                onLogin={() => setAccountOpen(true)}
              />
            </Suspense>
          ) : ["render", "cinematic", "packets", "structures"].includes(page) ||
            (["packs", "skins"].includes(page) &&
              !assetCaptureOpen) ? null : page === "settings" ? null : (
            <div
              className={`page-body asset-workspace asset-${page} ${assetCaptureOpen ? "asset-capture-open" : ""}`}
            >
              <div className="configuration-grid">
                {(page === "skins" || page === "packs") && (
                  <Button
                    variant="ghost"
                    onClick={() => setAssetCaptureOpen(false)}
                  >
                    Close capture settings
                  </Button>
                )}
                <section className="configuration">
                  <div className="section-heading">
                    <Server size={18} />
                    <h2>Select source</h2>
                  </div>
                  <p className="section-description">
                    Servers, worlds and replays.
                  </p>
                  <Label htmlFor="server-address">Selected source</Label>
                  <Input
                    id="server-address"
                    aria-label="Server address"
                    readOnly
                    value={selected?.name || address}
                    placeholder="No source selected"
                  />
                  {selected && (
                    <p className="source-summary-details">
                      {selected.probe?.reachable
                        ? "NetherNet reachable"
                        : (selected.route ?? "Connection not checked")}
                      {selected.probe?.players !== undefined
                        ? ` ? ${selected.probe.players} / ${selected.probe.maxPlayers ?? "?"} players`
                        : ""}
                    </p>
                  )}
                  <div className="source-summary-actions">
                    <Button
                      variant="outline"
                      disabled={state.running}
                      onClick={() => {
                        setSourceKind("custom");
                        setSourceOpen(true);
                      }}
                    >
                      <Server />
                      {address ? "Change source" : "Choose source"}
                    </Button>
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
                  <details className="source-shortcuts">
                    <summary>Browse online sources</summary>
                    <div className="source-choices">
                      {Object.entries(sourceNames)
                        .filter(([kind]) => kind !== "custom")
                        .map(([kind, name]) => (
                          <Button
                            key={kind}
                            variant="outline"
                            disabled={state.running}
                            onClick={() => {
                              setSourceKind(kind);
                              setSourceOpen(true);
                            }}
                          >
                            <SourceKindIcon kind={kind} />
                            {name}
                          </Button>
                        ))}
                    </div>
                  </details>
                  {state.running && state.listenAddr && (
                    <div className="source-shortcuts">
                      <Label htmlFor="minecraft-join-endpoint">Minecraft connection</Label>
                      <Input id="minecraft-join-endpoint" readOnly value={state.listenAddr} title={state.listenAddr}/>
                      <div className="source-summary-actions">
                        <Button variant="outline" disabled={!minecraftJoin.canJoin||minecraftJoin.launching} onClick={()=>void action(minecraftJoin.join)}>{minecraftJoin.phase === "delayed" ? "Retry join" : "Open Minecraft"}</Button>
                        <Button variant="outline" size="icon" aria-label="Copy Minecraft connection" onClick={()=>void copy(state.listenAddr)}><Copy/></Button>
                      </div>
                      <p className="source-summary-details">Use this address instead of a saved localhost entry.</p>
                    </div>
                  )}
                  <div className="settings-divider" />
                  <div className="section-heading">
                    <SlidersHorizontal size={18} />
                    <h2>Capture options</h2>
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
                        Default options for {names[page]} are already
                        configured.
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
							 o.flag !== "hostname-alias" &&
							 o.flag !== "lan-discovery" &&
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
                    {!pathsReady && !state.running && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setAdvanced(true)}
                      >
                        {Object.values(pathChecks).some((p) => p.error)
                          ? "Check advanced options"
                          : "Checking paths…"}
                      </Button>
                    )}
                    <p>
                      {!address
                        ? "Select a source first."
                        : state.running
                          ? "You can keep playing Minecraft during the download."
                          : !state.loggedIn &&
                              !address.toLowerCase().endsWith(".pcap2")
                            ? "Microsoft sign-in required."
                            : "Minecraft will then connect through the local proxy."}
                    </p>
                  </div>
                </aside>
              </div>
              {page === "worlds" && (
                <AssetLibrary
                  key={page}
                  page={page}
                  worlds={state.worlds}
                  packs={state.packs}
                  skins={state.skins}
                  onOpenEditor={(world, page) =>
                    launchProject({
                      ...newProjectDefaults(),
                      id: crypto.randomUUID(),
                      name: "",
                      nameMode: "auto",
                      sourceName: world.name,
                      page,
                      worldId: world.id,
                      updated: Date.now(),
                    })
                  }
                  onFolder={(path) =>
                    void action(() => api.OpenResultFolder(path))
                  }
                  onMap={() => void action(() => api.OpenMapWindow(theme))}
                  onWorldSettings={(world) => setWorldExportID(world.id)}
                  onWorldCopy={(world) => setWorldCopyID(world.id)}
                  onDeleteWorld={(world) => {
                    setDeleteError("");
                    setDeletingWorld({ id: world.id, name: world.name });
                  }}
                  onDeleteWorlds={(worlds)=>{
                    const roots=selectionRoots(worlds,state.worlds);
                    if(!roots.length)return;
                    setDeleteError("");setDeletingWorld({...roots[0],others:roots.slice(1)});
                  }}
                  onSaveSkin={(skin) =>
                    void action(async () => {
                      const path = await api.SaveRenderPNG(
                        skin.name,
                        skin.image,
                      );
                      if (path) notifySaved(path);
                    })
                  }
                  onSource={() => {
                    setAssetCaptureOpen(true);
                    setSourceKind("featured");
                    setSourceOpen(true);
                  }}
                />
              )}
            </div>
          )}
          <ConsoleDock
            open={logsOpen}
            onOpenChange={setLogsOpen}
            logs={logs}
            sessionName={names[state.command] || "CraftBox"}
            running={state.running}
            onNavigate={navigateWorkspace}
            onStop={() => api.StopCommand()}
            onExport={(name, text) => api.SaveConsoleLog(name, text)}
            diagnostics={`CraftBox ${bootstrap?.version || ""}\nMinecraft ${bootstrap?.minecraft || ""}\nSession: ${state.command}\nRunning: ${state.running}\nWorlds: ${state.worlds.length}\nPacks: ${state.packs.length}\nSkins: ${state.skins.length}`}
          />
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
            <NotificationCenter />
          </span>
        </footer>
      </div>
      {deleteDialog}
      {worldExportID && (
        <WorldExportSettingsDialog
          worldId={worldExportID}
          onClose={() => setWorldExportID(undefined)}
        />
      )}
      {worldCopyID && (
        <WorldCopyDialog
          worldId={worldCopyID}
          onClose={() => setWorldCopyID(undefined)}
        />
      )}
      <PreferencesDialog
        initialCategory={preferenceCategory}
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        theme={theme}
        onTheme={setTheme}
        version={bootstrap?.version}
        minecraft={bootstrap?.minecraft}
        updateBusy={updateBusy || releaseUpdates.busy}
        updateInfo={releaseUpdates.info}
        updateError={releaseUpdates.error}
        updateVersion={state.updateVersion}
        updateProgress={state.updateProgress}
        onCheckUpdate={() => void releaseUpdates.check()}
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
      <FirstStartDialog
        open={
          needsAccountIntro && !accountOpen && !state.authCode && !initError
        }
        loggedIn={state.loggedIn}
        account={state.account}
        onSkip={finishAccountIntro}
        onLink={() => {
          linkingAccountIntro.current = true;
          setAccountOpen(true);
          void action(() => api.Login());
        }}
      />
      <StartDialog
        open={
          startOpen &&
          !!bootstrap &&
          !initError &&
          !needsAccountIntro &&
          !accountOpen &&
          !state.authCode
        }
        onOpenChange={setStartOpen}
        recent={recent}
        worlds={state.worlds}
        onCreate={showNewProject}
        onOpen={openFromStart}
        onOpenProject={() => studioCommand("open-project")}
        onResume={launchProject}
        onRecentAction={manageProject}
        onPage={navigateWorkspace}
      />
      <Dialog open={quitOpen} onOpenChange={setQuitOpen}>
        <DialogContent resizable={false} tone="warning">
          <DialogHeader>
            <DialogTitle>End session?</DialogTitle>
            <DialogDescription>
              Captured world data will be saved.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setQuitOpen(false)}>
              Keep working
            </Button>
            <Button
              onClick={() => {
                setQuitOpen(false);
                requestSessionQuit(
                  (done) => documentWorkspaces.current?.quit(done),
                  setError,
                );
              }}
            >
              Quit
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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
      <SourcePicker
        open={sourceOpen}
        onOpenChange={setSourceOpen}
        initialKind={sourceKind}
        loggedIn={state.loggedIn}
        minecraft={bootstrap?.minecraft}
        accountKey={state.account}
        onSignIn={() => {
          setSourceOpen(false);
          setAccountOpen(true);
        }}
        onSelect={(source) => {
          setSelected(source);
          setAddress(source.address);
        }}
        onOpenCapture={() =>
          void action(async () => {
            const file = await api.PickFile();
            if (file) {
              setAddress(file);
              setSelected(undefined);
              setSourceOpen(false);
            }
          })
        }
      />
      <Dialog
        open={accountOpen || Boolean(state.authCode)}
        onOpenChange={(open) => {
          setAccountOpen(open);
          if (!open && state.authenticating)
            void action(() => api.CancelLogin());
        }}
      >
        <DialogContent className="account-dialog">
          <DialogHeader>
            <DialogTitle>
              {state.authCode ? "Microsoft sign-in" : "Your account"}
            </DialogTitle>
            <DialogDescription>Microsoft account</DialogDescription>
          </DialogHeader>
          {state.authCode ? (
            <div className="auth-flow">
              <p>Enter this code at Microsoft sign-in:</p>
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
              <AccountIdentity
                loggedIn={state.loggedIn}
                accountKey={state.account}
              />
              <section className="account-local-profile">
                <h3>Sign-in profiles</h3>
                <p className="secondary">
                  Keep separate local sign-ins for different Microsoft accounts.
                  This name labels the saved profile; it does not rename your
                  gamertag.
                </p>
                <Label htmlFor="account-name">Profile name</Label>
                <Input
                  id="account-name"
                  maxLength={80}
                  validate={profileNameError}
                  value={accountName}
                  placeholder="Default account"
                  onChange={(e) => setAccountName(e.target.value)}
                  disabled={state.running || state.authenticating}
                />

                <Button
                  variant="outline"
                  disabled={
                    state.running ||
                    state.authenticating ||
                    !!profileNameError(accountName)
                  }
                  onClick={() =>
                    void action(() => api.SwitchAccount(accountName))
                  }
                >
                  <AccountIcon />
                  Switch account
                </Button>
              </section>
              <section className="account-session-actions">
                <h3>
                  {state.loggedIn ? "Current session" : "Connect your account"}
                </h3>
                <p className="secondary">
                  {state.running
                    ? "End the active capture session to switch accounts or sign out."
                    : state.loggedIn
                      ? "Sign out of this local profile. Your saved worlds, assets and recordings are kept."
                      : "Sign in securely on the Microsoft website. CraftBox does not ask for your password."}
                </p>
                {state.loggedIn ? (
                  <>
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
              </section>
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
      {(error || state.error) && <ActionErrorDialog
        key={error || state.error}
        message={error || state.error}
        connection={error ? undefined : state.errorConnection}
        sessionID={state.logSessionID}
        onCopy={() => void copy(error || state.error)}
        onClose={() => { setError(""); void api.DismissError(); }}
      />}
    </TooltipProvider>
  );
}

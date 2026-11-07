import HeaderActionIcon from "./HeaderActionIcon";
import FeedbackFlyout from "./FeedbackFlyout";
import { documentLabel, useDocumentScope } from "./document-scope";
import { notifySaved, notifyFailure } from "./StudioFlyouts";
import { useOperationState } from "./use-operation-state";
import {isExportCancellation} from "./export-tasks";
import {
  preferredRenderSettings,
  preferredFlightSettings,
} from "./preferred-editor-settings";
import StudioInput from "@/StudioInput";

import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { cinematicSignature } from "./cinematic-signature";
import {
  FolderOpen,
  ImageDown,
  Download,
  LoaderCircle,
  RefreshCw,
  Sun,
} from "lucide-react";
import { RenderIcon, CinematicIcon, WorldIcon } from "./studio-icons";
import { Button } from "@/components/ui/button";
import { api, type RenderWorld, type World, type ChunkBounds } from "./bridge";
import RenderRegionDialog from "./RenderRegionDialog";
import RenderPackDialog, { ResourceTraceDialog } from "./RenderPackDialog";
import {
  renderDefaults,
  renderDimensions,
  applyRenderPreset,
  renderPresetLabels,
  renderQuality,
  type RenderSettings,
} from "./render-settings";
import type { RenderCanvasHandle } from "./RenderCanvas";
import FlightControls, { type FlightSettings } from "./FlightControls";
import { renderSkies } from "./render-skies";
import LightingControls from "./LightingControls";
import AutoDirectionControls from "./AutoDirectionControls";
import DaytimeControl from "./DaytimeControl";
import LookLibrary from "./LookLibrary";
import EditorLoadingOverlay from "./EditorLoadingOverlay";
import MapPresentationControls from "./MapPresentationControls";
import AtlasMap, { type AtlasMapHandle } from "./AtlasMap";
import { mapDefaults, mapRenderSettings } from "./map-presentation";
import {
  captureStudioSky,
  restoreStudioSky,
  type StudioDocument,
  type StudioSession,
  type CinematicViewState,
  type SavedCameraView,
} from "./studio-document";
import { rememberDirectionExample } from "./direction-learning";
import "./render-inspector.css";
import {
  useEditorCommands,
  dispatchEditorCommand,
  idleEditor,
  type EditorStatus,
} from "./editor-commands";
import { ImportWorldDialog, MediaExportDialog } from "./MediaDialogs";
import { ModelExportDialog } from "./ExportStudio";
import {
  imageExportData,
  type ImageFormat,
  type ExportOptions,
} from "./export-formats";
import ViewportMenu from "./ViewportMenu";
import AtmosphereControls from "./AtmosphereControls";
import type { CloudEditorState } from "./CloudLayerControls";
import { keyCloudValue, type CloudChannel } from "./cloud-layers";
import RenderStyleDialog from "./RenderStyleDialog";
import { normalizeAtmosphere } from "./atmosphere-settings";
import type { StudioProject } from "./start-project";
const RenderCanvas = lazy(() => import("./RenderCanvas"));
import StudioLightControls, {
  type LightEditorState,
} from "./StudioLightControls";
import { parseStudioLights } from "./studio-lights";
import DepthOfFieldControls from "./DepthOfFieldControls";

const CinematicStudio = lazy(() => import("./CinematicStudio"));

export default function RenderEditor({
  worlds,
  activeWorldId,
  cinematic = false,
  onEditorStateChange,
  project,
  onProjectSourceChange,
  loadedProject,
  onStudioSession,
  onStudioSaved,
  requestTransition,
  onStudioOpen,
}: {
  loadedProject?: {
    world: RenderWorld;
    document: StudioDocument;
    path: string;
  };
  onStudioSession?: (session: StudioSession | undefined) => void;
  onStudioSaved?: (value: {
    world: RenderWorld;
    document: StudioDocument;
    path: string;
  }) => void;
  requestTransition?: (action: () => void) => void;
  onStudioOpen?: () => void;
  worlds: World[];
  activeWorldId?: string;
  cinematic?: boolean;
  onEditorStateChange?: (status: EditorStatus) => void;
  project?: StudioProject;
  onProjectSourceChange?: (
    worldId: string,
    sourceKind?: "archive" | "folder",
    sourceName?: string,
  ) => void;
}) {
  const documentScope = useDocumentScope();
	const [viewportQuality,setViewportQuality]=useState<"editing"|"full">("editing");
  const ownedSources = useRef(new Set<string>());
  const [selected, setSelected] = useState(
    loadedProject?.world.id ??
      (project ? project.worldId : (activeWorldId ?? worlds[0]?.id ?? "")),
  );
  const [renderBounds, setRenderBounds] = useState<ChunkBounds | undefined>(
      loadedProject?.document.selection,
    ),
    [mapSelection, setMapSelection] = useState<ChunkBounds | undefined>(
      loadedProject?.document.selection,
    ),
    [regionToCreate, setRegionToCreate] = useState<ChunkBounds>(),
    [derivedMaps, setDerivedMaps] = useState<RenderWorld[]>([]);
  const previousSource = useRef(selected);
  useEffect(() => {
    if (previousSource.current !== selected) {
      previousSource.current = selected;
      setRenderBounds(undefined);
      setMapSelection(undefined);
    }
  }, [selected]);
  const [flying, setFlying] = useState(false),
    [flightSettings, setFlightSettings] = useState<FlightSettings>(
      preferredFlightSettings,
    );
  useEffect(() => setFlying(false), [selected]);
  const [importKind, setImportKind] = useState<"archive" | "folder">("archive"),
    [importOpen, setImportOpen] = useState(false),
    [exportOpen, setExportOpen] = useState(false);
  const [imported, setImported] = useState<RenderWorld | undefined>(
    loadedProject?.world,
  );
  const [exportPreview, setExportPreview] = useState<string>();
  const [panoramaOpen, setPanoramaOpen] = useState(false);
  const [panoramaWidth, setPanoramaWidth] = useState(2048);
  const [panoramaName] = useState("Panorama");
  const [panoramaProgress, setPanoramaProgress] = useState<number>();
  const panoramaCancelled = useRef(false);
  const [settings, setSettings] = useState<RenderSettings>(() => {
    if (loadedProject) return loadedProject.document.settings;
    if (!project) return preferredRenderSettings();
    const defaults = {
      ...applyRenderPreset(preferredRenderSettings(), project.look),
      quality: preferredRenderSettings().quality,
      transparent: preferredRenderSettings().transparent,
      edge: project.edge,
      aspect: project.aspect,
    };
    try {
      const stored = JSON.parse(
        localStorage.getItem("craftbox-render-project:" + project.id) || "null",
      );
      return stored
        ? {
            ...defaults,
            ...(stored.dofMode === undefined
              ? {
                  dofMode: "lens" as const,
                  dofNearStrength: 1,
                  dofFarStrength: 1,
                }
              : {}),
            ...stored,
            studioLights: parseStudioLights(stored.studioLights),
            atmosphere: normalizeAtmosphere(stored.atmosphere),
          }
        : defaults;
    } catch {
      return defaults;
    }
  });
  const [skyFile, setSkyFile] = useState<File | undefined>(() =>
    restoreStudioSky(loadedProject?.document.customSky),
  );
  const [mapView, setMapView] = useState(
    loadedProject?.document.mapView ?? mapDefaults,
  );
  const [placingMarker, setPlacingMarker] = useState<string>();
  const [atlasMode, setAtlasMode] = useState(true);
  const [atlasFocus, setAtlasFocus] = useState<[number, number, number]>();
  const atlasCanvas = useRef<AtlasMapHandle>(null);
  const emptyMapTiles = useRef(new Map());
  const [selectedLight, setSelectedLight] = useState<string>();
  const [placingLight, setPlacingLight] = useState(false);
  const [lightTarget, setLightTarget] = useState(false);
  const [pickingFocus, setPickingFocus] = useState(false);
  const [focusGuides, setFocusGuides] = useState(true);
  const [selectedCloud, setSelectedCloud] = useState<string>();
  const [placingCloud, setPlacingCloud] = useState(false);
  const [cloudMode, setCloudMode] =
    useState<CloudEditorState["mode"]>("translate");
  const cloudEditor: CloudEditorState = {
    selected: selectedCloud,
    placing: placingCloud,
    mode: cloudMode,
    onSelect: (id) => {
      setSelectedCloud(id);
      setSelectedLight(undefined);
      setPlacingCloud(false);
    },
    onTransform: (id, change) => {
      setSettings((s) => ({
        ...s,
        atmosphere: {
          ...s.atmosphere,
          cloudLayers: s.atmosphere.cloudLayers.map((layer) => {
            if (layer.id !== id) return layer;
            let next = { ...layer, ...change };
            if (cinematic && cinemaDocument)
              for (const property of ["position", "size", "rotation"] as const)
                if (change[property])
                  for (let i = 0; i < 3; i++) {
                    const channel =
                      `${property}.${["x", "y", "z"][i]}` as CloudChannel;
                    if (layer.tracks.some((t) => t.channel === channel))
                      next = keyCloudValue(
                        next,
                        channel,
                        cinemaDocument.time,
                        change[property]![i],
                        cinemaDocument.sequence.fps,
                      );
                  }
            return next;
          }),
        },
      }));
      setPlacingCloud(false);
    },
  };
  const onFocusPick = (distance: number, point: [number, number, number]) => {
    setSettings((s) => ({
      ...s,
      dofEnabled: true,
      dofDistance: Math.max(0.001, Math.min(100000, distance)),
      dofTargetLocked: true,
      dofTargetX: point[0],
      dofTargetY: point[1],
      dofTargetZ: point[2],
    }));
    setPickingFocus(false);
  };
  const lightEditor: LightEditorState = {
    selected: selectedLight,
    placing: placingLight,
    target: lightTarget,
    onSelect: (id) => {
      setSelectedCloud(undefined);
      setPlacingCloud(false);
      setSelectedLight(id);
      setPlacingLight(false);
      setLightTarget(false);
    },
    onPosition: (id, position, target) => {
      setSettings((s) => ({
        ...s,
        studioLights: s.studioLights.map((l) =>
          l.id === id
            ? { ...l, [target ? "target" : "position"]: position }
            : l,
        ),
      }));
      setPlacingLight(false);
    },
  };
  const [cinemaDocument, setCinemaDocument] = useState<
    CinematicViewState | undefined
  >(loadedProject?.document.cinematic);
  const [viewPose, setViewPose] = useState<SavedCameraView | undefined>(
    loadedProject?.document.view,
  );
  const [viewEdited, setViewEdited] = useState(false);
  const viewRevision = useRef(0);
  const [projectPath, setProjectPath] = useState(loadedProject?.path);
  const [savedSignature, setSavedSignature] = useState<string>();
  const isMapView = !cinematic && mapView.enabled;
  const canvasSettings = mapRenderSettings(settings, {
    ...mapView,
    enabled: isMapView && !atlasMode,
  });
  const [skyLoading, setSkyLoading] = useState(false);
  const [selectedLook, setSelectedLook] = useState(
    loadedProject
      ? (loadedProject.document.look ?? "")
      : project?.look || "daylight",
  );
  const [showLighting, setShowLighting] = useState(false);
  const [styleOpen, setStyleOpen] = useState(false);
  const [inspectorTab, setInspectorTab] = useState("looks");
  const [history, setHistory] = useState({ undo: false, redo: false });
  const [progress, setProgress] = useState({
    loaded: 0,
    total: 0,
    triangles: 0,
  });
  const [renderReady, setRenderReady] = useState(false),
    [atlasReady, setAtlasReady] = useState(false),
    [error, setError] = useState("");
  const [busy, setBusy, busyOperation] = useOperationState();
  const [saving, setSaving, savingOperation] = useOperationState();
  const setResult = notifySaved;
  const ready = isMapView && atlasMode ? atlasReady : renderReady;
  const [generation, setGeneration] = useState(0);
  const modelExporting = false;
  const [modelOpen, setModelOpen] = useState(false),
    [imageFormat, setImageFormat] = useState<ImageFormat>("png"),
    [panoramaFormat, setPanoramaFormat] = useState<ImageFormat>("png");
  const [pickingResource, setPickingResource] = useState(false);
  const [resourceTrace, setResourceTrace] = useState<Awaited<
    ReturnType<typeof api.TraceSceneResource>
  > | null>(null);
  const canvas = useRef<RenderCanvasHandle>(null),
    mounted = useRef(true),
    pending = useRef(false),
    importId = useRef("");
  function saveGLB() {
    setModelOpen(true);
  }
  useEffect(() => {
    if (!exportOpen) return;
    let next = 0;
    const frame = requestAnimationFrame(() => {
      next = requestAnimationFrame(() =>
        setExportPreview(canvas.current?.preview?.()),
      );
    });
    return () => {
      cancelAnimationFrame(frame);
      cancelAnimationFrame(next);
    };
  }, [exportOpen, settings]);
  useEffect(() => {
    if (selected.startsWith("render-") || selected.startsWith("project-"))
      ownedSources.current.add(selected);
  }, [selected]);
  const sourceName =
    derivedMaps.find((w) => w.id === selected)?.name ||
    loadedProject?.document.name ||
    project?.name ||
    (imported?.id === selected
      ? imported.name
      : (worlds.find((w) => w.id === selected)?.name ?? "Map render"));
  const signature = JSON.stringify({
    renderBounds,
    settings,
    mapView,
    cinemaDocument: cinematicSignature(cinemaDocument),
    viewPose,
    selected,
    sky: skyFile ? `${skyFile.name}:${skyFile.size}` : null,
  });
  const latestSignature = useRef(signature);
  latestSignature.current = signature;

  const originalFileSignature = useRef(loadedProject ? signature : undefined);
  const documentDirty =
    (!!selected || !!project) &&
    (signature !== (savedSignature ?? originalFileSignature.current) ||
      viewEdited);
  const currentDocumentDirty = useRef(documentDirty);
  currentDocumentDirty.current = documentDirty;
  async function saveStudio(forceAs = false) {
    if (!ready || !selected)
      throw new Error(
        "The world must finish loading before the project can be saved.",
      );
    if (savingOperation.current || busyOperation.current) return false;
    const startingSignature = signature;
    const camera = canvas.current?.snapshot();
    const capturedViewRevision = viewRevision.current;
    const capturedCinema = canvas.current?.studioState?.() ?? cinemaDocument;
    if (camera) setViewPose(camera);
    if (capturedCinema) setCinemaDocument(capturedCinema);
    const capturedSignature = JSON.stringify({
      renderBounds,
      settings,
      mapView,
      cinemaDocument: cinematicSignature(capturedCinema),
      viewPose: camera ?? viewPose,
      selected,
      sky: skyFile ? `${skyFile.name}:${skyFile.size}` : null,
    });
    setSaving(true);
    setError("");
    try {
      const document: StudioDocument = {
        selection: renderBounds,
        format: "craftbox-studio",
        version: 1,
        name: sourceName,
        editor: cinematic ? "cinematic" : "render",
        settings,
        look: selectedLook,
        mapView: {
          ...mapView,
          zoom: mapView.enabled ? (camera?.zoom ?? mapView.zoom) : mapView.zoom,
        },
        view: camera ?? viewPose,
        cinematic: capturedCinema,
        customSky: await captureStudioSky(skyFile),
      };
      const path = await api.SaveStudioProjectForTab(
        documentScope.id || selected,
        selected,
        sourceName,
        JSON.stringify(document),
        forceAs || !projectPath,
      );
      if (!path) return false;
      setProjectPath(path);
      setSavedSignature(capturedSignature);
      if (viewRevision.current === capturedViewRevision) setViewEdited(false);
      setResult(path);
      const world =
        imported?.id === selected
          ? imported
          : { id: selected, name: sourceName, chunks: progress.total };
      onStudioSaved?.({ world, document, path });
      const unchanged =
        (latestSignature.current === capturedSignature ||
          latestSignature.current === startingSignature) &&
        viewRevision.current === capturedViewRevision;
      if (unchanged) currentDocumentDirty.current = false;
      return unchanged;
    } finally {
      if (mounted.current) setSaving(false);
    }
  }
  useEffect(() => {
    onStudioSession?.({
      name: documentLabel(sourceName, projectPath),
      path: projectPath,
      get dirty() {
        return currentDocumentDirty.current;
      },
      ready,
      get busy() {
        return savingOperation.current || busyOperation.current;
      },
      save: saveStudio,
      dispose: () => {
        for (const id of ownedSources.current) void api.CloseRenderWorld(id);
        void api.ReleaseDocumentTab(documentScope.id).catch(() => {});
      },
    });
  }, [
    signature,
    viewEdited,
    documentDirty,
    busy,
    saving,
    ready,
    projectPath,
    sourceName,
    onStudioSession,
  ]);
  useEffect(() => () => onStudioSession?.(undefined), [onStudioSession]);
  useEffect(() => {
    if (project)
      try {
        localStorage.setItem(
          "craftbox-render-project:" + project.id,
          JSON.stringify(settings),
        );
      } catch {}
  }, [settings, project]);
  useEffect(() => {
    if (selected && selected !== imported?.id)
      onProjectSourceChange?.(
        selected,
        undefined,
        worlds.find((w) => w.id === selected)?.name,
      );
  }, [selected]);
  useEffect(() => {
    if (!project?.openKind) return;
    const timer = window.setTimeout(() => void open(project.openKind!), 0);
    return () => window.clearTimeout(timer);
  }, []);
  useEffect(() => {
    onEditorStateChange?.({ ready, busy: busy || saving, ...history });
  }, [ready, busy, saving, history, onEditorStateChange]);
  useEffect(
    () => () => onEditorStateChange?.(idleEditor),
    [onEditorStateChange],
  );
  useEditorCommands((command) => {
    if (saving || busy) return;
    if (command === "open-archive") requestImport("archive");
    else if (command === "open-folder") requestImport("folder");
    else if (command === "reload" && selected) setGeneration((g) => g + 1);
    else if (command === "export-png" && ready) void save();
    else if (command === "export-glb" && ready) void saveGLB();
    else if (command === "fit" && ready) canvas.current?.fit();
    else if (command === "isometric" && ready) canvas.current?.view(45, 35);
    else if (command === "top" && ready) canvas.current?.view(0, 89);
    else if (command === "image-panel") setShowLighting((v) => !v);
    else if (command === "save-project")
      void saveStudio().catch((e) =>
        notifyFailure(e, "Project could not be saved"),
      );
    else if (command === "save-project-as")
      void saveStudio(true).catch((e) =>
        notifyFailure(e, "Project could not be saved"),
      );
    else if (command === "reset-look") {
      setSettings(renderDefaults);
      setSelectedLook("daylight");
    }
  });
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      if (pending.current) void api.CancelRenderImportForTab(documentScope.id);
    };
  }, []);
  useEffect(() => {
    if (!project && !selected && worlds.length)
      setSelected(activeWorldId ?? worlds[0].id);
  }, [worlds, activeWorldId, selected]);
  useEffect(() => {
    setRenderReady(false);
    setAtlasReady(false);
    setError("");
    setResult("");
    setProgress({ loaded: 0, total: 0, triangles: 0 });
  }, [selected, generation]);
  useEffect(() => setAtlasFocus(undefined), [selected]);
  const update = <K extends keyof RenderSettings>(
    key: K,
    value: RenderSettings[K],
  ) => {
    setSelectedLook("");
    setSettings((s) => ({
      ...s,
      [key]: value,
      ...([
        "azimuth",
        "elevation",
        "sun",
        "ambient",
        "sky",
        "skyRotation",
        "environment",
        "warmth",
        "sunLinked",
      ].includes(key)
        ? { timeLinked: false }
        : {}),
    }));
  };
  async function open(kind: "archive" | "folder") {
    setBusy(true);
    pending.current = true;
    setError("");
    try {
      const doc = await api.OpenRenderWorldForTab(documentScope.id, kind);
      if (!mounted.current) {
        if (doc.id) void api.CloseRenderWorld(doc.id);
        return;
      }
      if (doc.id) {
        importId.current = doc.id;
        setImported(doc);
        setSelected(doc.id);
        onProjectSourceChange?.(doc.id, kind, doc.name);
      }
    } catch (e) {
      if (mounted.current) setError(e instanceof Error ? e.message : String(e));
    } finally {
      pending.current = false;
      if (mounted.current) setBusy(false);
    }
  }
  function requestImport(kind: "archive" | "folder") {
    const action = () => {
      setImportKind(kind);
      setImportOpen(true);
    };
    if (requestTransition) requestTransition(action);
    else action();
  }

  async function switchDimension(dimension: number) {
    if (!imported || busy) return;
    setBusy(true);
    pending.current = true;
    setError("");
    try {
      const doc = await api.SwitchRenderDimensionForTab(
        documentScope.id,
        imported.id,
        dimension,
      );
      if (!mounted.current) {
        if (doc.id) void api.CloseRenderWorld(doc.id);
        return;
      }
      importId.current = doc.id;
      setImported(doc);
      setSelected(doc.id);
      onProjectSourceChange?.(doc.id, importKind, doc.name);
    } catch (e) {
      if (mounted.current) setError(e instanceof Error ? e.message : String(e));
    } finally {
      pending.current = false;
      if (mounted.current) setBusy(false);
    }
  }
  function save() {
    if (isMapView && atlasMode) {
      atlasCanvas.current?.export();
      return;
    }
    setExportPreview(canvas.current?.preview?.());
    setExportOpen(true);
  }
  async function savePNG(filename = sourceName, options: ExportOptions) {
    setSaving(true);
    setError("");
    setResult("");
    try {
      const data = await canvas.current!.capture();
      if (!mounted.current) return;
      const path = await api.SaveImage({
        name: filename,
        format: options.format,
        data: await imageExportData(
          data,
          options.format as ImageFormat,
          options.quality,
        ),
        quality: options.quality,
        background: options.background,
      });
      if (mounted.current) setResult(path);
    } catch (e) {
      if (mounted.current&&!isExportCancellation(e)) notifyFailure(e, "Image export failed");
    } finally {
      if (mounted.current) setSaving(false);
    }
  }
  async function savePanorama(filename: string, options: ExportOptions) {
    if (!canvas.current?.capturePanorama || !ready || saving) return;
    setPanoramaOpen(false);
    setSaving(true);
    setError("");
    setPanoramaProgress(0);
    panoramaCancelled.current = false;
    try {
      const data = await canvas.current.capturePanorama(
        panoramaWidth,
        setPanoramaProgress,
        () => panoramaCancelled.current,
      );
      if (!mounted.current || panoramaCancelled.current) return;
      const path = await api.SaveImage({
        name: filename,
        format: options.format,
        data: await imageExportData(
          data,
          options.format as ImageFormat,
          options.quality,
        ),
        quality: options.quality,
        background: options.background,
      });
      if (mounted.current) setResult(path);
    } catch (error) {
      if (mounted.current && !panoramaCancelled.current&&!isExportCancellation(error))
        notifyFailure(error, "Panorama export failed");
    } finally {
      if (mounted.current) {
        setSaving(false);
        setPanoramaProgress(undefined);
      }
    }
  }
  function slider(
    label: string,
    key:
      | "azimuth"
      | "elevation"
      | "sun"
      | "ambient"
      | "exposure"
      | "ao"
      | "environment"
      | "skyRotation"
      | "skyBrightness"
      | "softness"
      | "aoRadius"
      | "skyElevation"
      | "reflections"
      | "roughness"
      | "waterOpacity"
      | "waterVisibility"
      | "waterRoughness"
      | "waterReflections"
      | "waterWaves"
      | "waterColorStrength"
      | "waterColorDepth"
      | "waterTextureStrength"
      | "waterWaveScale"
      | "waterTransparency"
      | "waterDeepLight"
      | "waterShoreSoftness"
      | "frameMargin"
      | "frameVerticalShift"
      | "detailProtection"
      | "bloom"
      | "saturation"
      | "vibrance"
      | "contrast"
      | "temperature"
      | "tint"
      | "highlights"
      | "shadows"
      | "sharpen"
      | "vignette",
    min: number,
    max: number,
    step: number,
  ) {
    const framingDisabled =
      (key === "frameMargin" || key === "frameVerticalShift") &&
      (!!cinematic || isMapView);
    return (
      <div className="render-control">
        <span>
          {label}
          <StudioInput
            className="inspector-number"
            aria-label={`${label} Value`}
            type="number"
            min={min}
            max={max}
            increment={step}
            value={settings[key]}
            disabled={saving || framingDisabled}
            onChange={(e) => {
              const n = Number(e.target.value);
              if (Number.isFinite(n))
                update(key, Math.min(max, Math.max(min, n)));
            }}
          />
        </span>
        <StudioInput
          type="range"
          aria-label={label}
          disabled={
            saving ||
            framingDisabled ||
            ((key === "azimuth" || key === "elevation") &&
              settings.sunLinked &&
              settings.sky !== "none")
          }
          min={min}
          max={max}
          step={step}
          value={settings[key]}
          onChange={(e) => update(key, Number(e.target.value))}
        />
      </div>
    );
  }
  const viewMenu = (
    <ViewportMenu
      disabled={!ready || saving}
      locked={settings.pitchLocked}
      onLock={() => setSettings((s) => ({ ...s, pitchLocked: !s.pitchLocked }))}
      onFit={() => {
        dispatchEditorCommand("fit");
      }}
      onView={(angle, tilt) => {
        dispatchEditorCommand("inspect-view");
        canvas.current?.view(angle, tilt);
      }}
    />
  );
  const dimensions = renderDimensions(settings.edge, settings.aspect);
  return (
    <section
      className={`render-editor ${cinematic ? "cinematic-editor" : ""}`}
      aria-label={cinematic ? "Cinematic editor" : "Render editor"}
    >
      <ImportWorldDialog
        open={importOpen}
        onOpenChange={setImportOpen}
        kind={importKind}
        onImport={(kind) => void open(kind)}
      />
      <MediaExportDialog
        open={exportOpen}
        onOpenChange={setExportOpen}
        format={imageFormat}
        onFormat={(f) => setImageFormat(f as ImageFormat)}
        edge={settings.edge}
        onEdge={(edge) => setSettings((s) => ({ ...s, edge }))}
        aspect={settings.aspect}
        name={sourceName}
        preview={exportPreview}
        transparent={canvasSettings.transparent}
        onTransparent={(transparent) =>
          mapView.enabled
            ? setMapView((s) => ({ ...s, transparent }))
            : setSettings((s) => ({ ...s, transparent }))
        }
        onExport={(name, options) => void savePNG(name, options)}
      />
      <ModelExportDialog
        onBusy={setSaving}
        open={modelOpen}
        onOpenChange={setModelOpen}
        worldId={selected}
        name={sourceName}
        preview={exportPreview}
        bounds={renderBounds}
      />
      <RenderRegionDialog
        bounds={regionToCreate}
        onClose={() => setRegionToCreate(undefined)}
        worldId={selected}
        sourceName={sourceName}
        onBusy={setSaving}
        onCreated={(result) => {
          setDerivedMaps((maps) => [...maps, result.world]);
          setSelected(result.world.id);
          setRenderBounds(undefined);
          setMapSelection(undefined);
          setViewPose(undefined);
          setProjectPath(undefined);
          setMapView((s) => ({ ...s, enabled: false }));
          setGeneration((g) => g + 1);
          notifySaved(result.path);
        }}
      />
      <MediaExportDialog
        open={panoramaOpen}
        onOpenChange={setPanoramaOpen}
        title="Export 360° panorama"
        format={panoramaFormat}
        onFormat={(f) => setPanoramaFormat(f as ImageFormat)}
        edge={panoramaWidth}
        onEdge={setPanoramaWidth}
        aspect={2}
        dimensions={{ width: panoramaWidth, height: panoramaWidth / 2 }}
        name={panoramaName || "Panorama"}
        onExport={(name, options) => void savePanorama(name, options)}
      >
        <label>
          Panorama resolution
          <select
            aria-label="Panorama resolution"
            value={panoramaWidth}
            onChange={(e) => setPanoramaWidth(+e.target.value)}
          >
            {[1024, 2048, 4096].map((w) => (
              <option value={w} key={w}>
                {w} × {w / 2} pixels
              </option>
            ))}
          </select>
        </label>
        <p>
          A 360° panorama in 2:1 format, with your lighting and atmosphere.
          Depth of field and vignette are turned off.
        </p>
      </MediaExportDialog>{" "}
      <div className="render-header studio-view-heading">
        <div>
          <h1>
            {cinematic ? <CinematicIcon /> : <RenderIcon />}{" "}
            {cinematic ? "Cinematic editor" : "Render editor"}
          </h1>
          <p>
            {cinematic
              ? "Set up camera movement, shots and lighting."
              : "Render the map with lighting and shadows. Custom blocks are included; entities are not."}
          </p>
        </div>
        {cinematic && (
          <div
            className="cinema-workspace-switch"
            aria-label="Cinematic workspace"
          >
            <Button title="Camera &amp; sequence" className="toolbar-icon-action"
              disabled={saving}
              variant="outline"
              aria-label="Edit camera and sequence"
              aria-pressed={!showLighting}
              onClick={() => setShowLighting(false)}
            ><HeaderActionIcon label="Camera &amp; sequence"/>
              <span className="toolbar-action-label">Camera &amp; sequence</span></Button>
            <Button title="Lighting" className="toolbar-icon-action"
              disabled={saving}
              variant="outline"
              aria-label="Edit lighting"
              onClick={() => {
                setInspectorTab("light");
                setShowLighting(true);
              }}
            ><HeaderActionIcon label="Lighting"/>
              <span className="toolbar-action-label">Lighting</span></Button>
            <Button title="Atmosphere" className="toolbar-icon-action"
              disabled={saving}
              variant="outline"
              aria-label="Edit clouds and fog"
              onClick={() => {
                setInspectorTab("atmosphere");
                setShowLighting(true);
              }}
            ><HeaderActionIcon label="Atmosphere"/>
              <span className="toolbar-action-label">Atmosphere</span></Button>
            <Button title="Color" className="toolbar-icon-action"
              disabled={saving}
              variant="outline"
              aria-label="Edit colors"
              onClick={() => {
                setInspectorTab("color");
                setShowLighting(true);
              }}
            ><HeaderActionIcon label="Color"/>
              <span className="toolbar-action-label">Color</span></Button>
            <Button title="Scene looks" className="toolbar-icon-action"
              disabled={saving}
              variant="outline"
              aria-label="Edit scene looks"
              aria-pressed={showLighting && inspectorTab === "looks"}
              onClick={() => {
                setInspectorTab("looks");
                setShowLighting(true);
              }}
            ><HeaderActionIcon label="Scene looks"/>
              <span className="toolbar-action-label">Scene looks</span></Button>
          </div>
        )}
        <AutoDirectionControls
          key={`${selected}:${generation}`}
          ready={ready && !saving && !busy && !skyLoading && !isMapView}
          settings={settings}
          analyse={(progress) => canvas.current!.analyse(progress)}
          onApply={(s) => {
            setSettings(s);
            setSelectedLook("");
          }}
        />
        {onStudioOpen && (
          <Button title="Open project" aria-label="Open project" className="toolbar-icon-action"
            variant="outline"
            disabled={busy || saving}
            onClick={onStudioOpen}
          ><HeaderActionIcon label="Open project"/>
            <span className="toolbar-action-label">Open project</span></Button>
        )}
        <Button title="Save project" aria-label="Save project" className="toolbar-icon-action"
          variant="outline"
          disabled={!ready || busy || saving}
          onClick={() =>
            void saveStudio().catch((e) =>
              notifyFailure(e, "Project could not be saved"),
            )
          }
        ><HeaderActionIcon label="Save project"/>
          <span className="toolbar-action-label">Save project</span></Button>
        {!cinematic && (
          <Button title="360° panorama" aria-label="360° panorama" className="toolbar-icon-action"
            variant="outline"
            disabled={!ready || saving || busy || skyLoading || isMapView}
            onClick={() => setPanoramaOpen(true)}
          ><HeaderActionIcon label="360° panorama"/>
            <span className="toolbar-action-label">360° panorama</span></Button>
        )}
        <Button
          onClick={() => void save()}
          disabled={!ready || saving || busy || skyLoading}
        >
          {saving && !modelExporting ? (
            <LoaderCircle className="spin" />
          ) : (
            <ImageDown />
          )}
          {saving && !modelExporting ? "Rendering image …" : "Export image"}
        </Button>
        <Button
          variant="outline"
          disabled={!selected || saving || busy}
          onClick={() => void saveGLB()}
        >
          <Download /> Export 3D
        </Button>
      </div>
      <div data-editor-tab-host />
      <div className="render-source">
        <RenderPackDialog
          worldID={selected}
          onApplied={() => setGeneration((g) => g + 1)}
        />
		<label title="Keeps original geometry and textures. Fewer pixels and effects during movement; progressive refinement when still. Exports use full quality.">Viewport <select aria-label="Viewport quality" value={viewportQuality} disabled={!ready||saving||busy} onChange={event=>setViewportQuality(event.target.value as "editing"|"full")}><option value="editing">Interactive</option><option value="full">Full preview</option></select></label>
        <Button title="Click a block face…" aria-label="Click a block face…" className="toolbar-icon-action"
          variant="outline"
          disabled={!selected || saving || busy || isMapView}
          aria-pressed={pickingResource}
          onClick={() => setPickingResource((v) => !v)}
        ><HeaderActionIcon label="Click a block face…"/>
          <span className="toolbar-action-label">{pickingResource ? "Click a block face…" : "Inspect texture"}</span>
        </Button>
        {!cinematic && (
          <div
            role="group"
            aria-label="Editor view"
            style={{ display: "flex", gap: 4 }}
          >
            <Button title="3D view" aria-label="3D view" className="toolbar-icon-action"
              variant="outline"
              aria-pressed={!mapView.enabled}
              disabled={saving || busy}
              onClick={() => setMapView((s) => ({ ...s, enabled: false }))}
            ><HeaderActionIcon label="3D view"/>
              <span className="toolbar-action-label">3D view</span></Button>
            <Button title="2D map" aria-label="2D map" className="toolbar-icon-action"
              variant="outline"
              aria-pressed={mapView.enabled}
              disabled={saving || busy}
              onClick={() => {
                if (!isMapView) setAtlasReady(false);
                setMapView((s) => ({ ...s, enabled: true }));
              }}
            ><HeaderActionIcon label="2D map"/>
              <span className="toolbar-action-label">2D map</span></Button>
          </div>
        )}
        <label>
          Map
          <select
            aria-label="Map to render"
            value={selected}
            onChange={(e) => {
              const id = e.target.value;
              const action = () => setSelected(id);
              if (requestTransition) requestTransition(action);
              else action();
            }}
            disabled={busy || saving}
          >
            <option value="">Select map</option>
            {worlds.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name || "Recording"}
              </option>
            ))}
            {imported && (
              <option value={imported.id}>{imported.name} · File</option>
            )}
            {derivedMaps
              .filter((w) => !worlds.some((original) => original.id === w.id))
              .map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
          </select>
        </label>
        {renderBounds && (
          <div className="render-selection-status" role="status">
            Selection · X {renderBounds.minX}…{renderBounds.maxX}, Z{" "}
            {renderBounds.minZ}…{renderBounds.maxZ}
            <Button title="Show full world" aria-label="Show full world" className="toolbar-icon-action"
              variant="outline"
              disabled={busy || saving}
              onClick={() => {
                setRenderBounds(undefined);
                setGeneration((g) => g + 1);
              }}
            ><HeaderActionIcon label="Show full world"/>
              <span className="toolbar-action-label">Show full world</span></Button>
          </div>
        )}
        {imported?.id === selected &&
          imported.dimensions &&
          imported.dimensions.length > 1 && (
            <label className="render-dimension-select">
              <span>Dimension</span>
              <select
                aria-label="Imported world dimension"
                disabled={busy || saving}
                value={imported.dimension ?? 0}
                onChange={(e) => {
                  const dimension = Number(e.target.value);
                  const action = () => void switchDimension(dimension);
                  if (requestTransition) requestTransition(action);
                  else action();
                }}
              >
                {imported.dimensions.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name} · {d.chunks} chunks
                  </option>
                ))}
              </select>
            </label>
          )}
        <Button title="Open world file" aria-label="Open world file" className="toolbar-icon-action"
          variant="outline"
          onClick={() => requestImport("archive")}
          disabled={busy || saving}
        >
          <FolderOpen /> <span className="toolbar-action-label">Open world file</span></Button>
        <Button title="World folder" aria-label="World folder" className="toolbar-icon-action"
          variant="ghost"
          onClick={() => requestImport("folder")}
          disabled={busy || saving}
        ><HeaderActionIcon label="World folder"/>
          <span className="toolbar-action-label">World folder</span></Button>
        {busy ? (
          <Button title="Cancel" aria-label="Cancel" className="toolbar-icon-action"
            variant="ghost"
            onClick={() => void api.CancelRenderImportForTab(documentScope.id)}
          ><HeaderActionIcon label="Cancel"/>
            <span className="toolbar-action-label">Cancel</span></Button>
        ) : (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Reload render data"
            onClick={() => setGeneration((g) => g + 1)}
            disabled={!selected || saving}
          >
            <RefreshCw />
          </Button>
        )}
      </div>
      <div
        className={`render-workspace ${showLighting ? "cinema-light-open" : ""}`}
      >
        {cinematic && selected ? (
          <Suspense
            fallback={
              <div className="render-empty">Loading cinematic studio …</div>
            }
          >
            <CinematicStudio
              sourceName={sourceName}
				 viewportQuality={viewportQuality}
				 viewportFocus={worlds.find(world=>world.id===selected)?.player.filter((_,i)=>i!==1).map(value=>Math.floor(value/16)) as [number,number]|undefined}
				capturePoints={worlds.find(world=>world.id===selected)?.capturePoints??(imported?.id===selected?imported.capturePoints:undefined)}
              active={documentScope.active}
              key={`${selected}:${generation}`}
              ref={canvas}
              worldId={selected}
              settings={settings}
              initialCinematic={cinemaDocument}
              onLensScale={(scale) =>
                setSettings((s) => ({ ...s, dofWorldScale: scale }))
              }
              lightEditor={saving ? undefined : lightEditor}
              cloudEditor={saving ? undefined : cloudEditor}
              pickingFocus={pickingFocus && !saving}
              onFocusPick={onFocusPick}
              onFocusCancel={() => setPickingFocus(false)}
              focusGuides={focusGuides}
              onCinematicChange={setCinemaDocument}
              initialView={viewPose}
              onViewChange={setViewPose}
              onViewDirty={() => {
                viewRevision.current++;
                setViewEdited(true);
              }}
              skyFile={skyFile}
              onSkyLoading={setSkyLoading}
              onExporting={setSaving}
              onHistoryChange={setHistory}
              viewMenu={viewMenu}
              projectId={project?.id}
              projectName={project?.name}
              projectDuration={project?.duration}
              projectFps={project?.fps}
              onReady={() => setRenderReady(true)}
              onError={setError}
              onProgress={(loaded, total, triangles) =>
                setProgress({ loaded, total, triangles })
              }
            />
          </Suspense>
        ) : (
          <div className={`render-stage ${isMapView ? "map-2d" : ""}`}>
            {selected && isMapView && atlasMode ? (
              <AtlasMap
				 capturePoints={worlds.find(world=>world.id===selected)?.capturePoints??(imported?.id===selected?imported.capturePoints:undefined)}
                key={`atlas:${selected}:${generation}`}
                ref={atlasCanvas}
                worldId={selected}
                selection={mapSelection}
                onSelectionChange={setMapSelection}
                onSelectionClear={() => setMapSelection(undefined)}
                onOpenSelection3D={(bounds) => {
                  setRenderBounds(bounds);
                  setViewPose(undefined);
                  setAtlasFocus(undefined);
                  setSettings((s) => ({ ...s, projection: "perspective" }));
                  setMapView((s) => ({ ...s, enabled: false }));
                  setFlying(false);
                  setGeneration((g) => g + 1);
                }}
                onCreateSelectionMap={setRegionToCreate}
                name={sourceName}
                tiles={emptyMapTiles.current}
                player={[0, 64, 0]}
                hasPlayer={false}
                presentation={mapView}
                onPresentationChange={setMapView}
                placingMarker={!!placingMarker}
                onMapPlace={(x, z) => {
                  setMapView((s) => ({
                    ...s,
                    markers:
                      placingMarker === "new"
                        ? [
                            ...s.markers,
                            {
                              id: crypto.randomUUID(),
                              x: Math.round(x),
                              z: Math.round(z),
                              label: "New marker",
                              icon: "📍",
                              color: "#69b84d",
                              background: "#202020",
                              scale: 1,
                            },
                          ]
                        : s.markers.map((m) =>
                            m.id === placingMarker
                              ? { ...m, x: Math.round(x), z: Math.round(z) }
                              : m,
                          ),
                  }));
                  setPlacingMarker(undefined);
                }}
                onReady={() => setAtlasReady(true)}
                onOpen3D={(point) => {
                  setAtlasFocus(point);
                  setSettings((s) => ({ ...s, projection: "perspective" }));
                  setMapView((s) => ({ ...s, enabled: false }));
                  setFlying(false);

                  setGeneration((g) => g + 1);
                }}
              />
            ) : null}
            {selected ? (
              <Suspense
                fallback={
                  <div className="render-empty">
                    <LoaderCircle className="spin" />
                    Loading renderer …
                  </div>
                }
              >
                <RenderCanvas
                  sourceName={sourceName}
				viewportQuality={viewportQuality}
				viewportFocus={worlds.find(world=>world.id===selected)?.player.filter((_,i)=>i!==1).map(value=>Math.floor(value/16)) as [number,number]|undefined}
                  key={`${selected}:${generation}`}
                  ref={canvas}
                  worldId={selected}
                  selectionBounds={renderBounds}
                  active={documentScope.active && (!isMapView || !atlasMode)}
                  settings={canvasSettings}
                  pickingResource={pickingResource}
                  onResourcePick={(position, face) => {
                    setPickingResource(false);
                    void api
                      .TraceSceneResource(selected, position, face)
                      .then(setResourceTrace)
                      .catch((e) => notifyFailure(String(e)));
                  }}
                  flight={{
                    ...flightSettings,
                    enabled: flying && !isMapView,
                    onExit: () => setFlying(false),
                    onFov: (fov) => setFlightSettings((v) => ({ ...v, fov })),
                    onSpeed: (scale) =>
                      setFlightSettings((v) => ({
                        ...v,
                        speed: Math.max(0.1, Math.min(4096, v.speed * scale)),
                      })),
                  }}
                  lightEditor={
                    saving || isMapView || flying ? undefined : lightEditor
                  }
                  cloudEditor={
                    saving || isMapView || flying ? undefined : cloudEditor
                  }
                  pickingFocus={
                    pickingFocus && !saving && !isMapView && !flying
                  }
                  onFocusPick={onFocusPick}
                  onFocusCancel={() => setPickingFocus(false)}
                  focusGuides={focusGuides}
                  mapView={atlasMode ? { ...mapView, enabled: false } : mapView}
                  initialView={viewPose}
                  initialMapFocus={atlasFocus}
                  onViewChange={setViewPose}
                  onViewDirty={() => {
                    viewRevision.current++;
                    setViewEdited(true);
                  }}
                  placingMarker={!!placingMarker}
                  onMapPlace={(x, z) => {
                    setMapView((s) => ({
                      ...s,
                      markers:
                        placingMarker === "new"
                          ? [
                              ...s.markers,
                              {
                                id: crypto.randomUUID(),
                                x: Math.round(x),
                                z: Math.round(z),
                                label: "New marker",
                                icon: "📍",
                                color: "#69b84d",
                                background: "#202020",
                                scale: 1,
                              },
                            ]
                          : s.markers.map((m) =>
                              m.id === placingMarker
                                ? { ...m, x: Math.round(x), z: Math.round(z) }
                                : m,
                            ),
                    }));
                    setPlacingMarker(undefined);
                  }}
                  skyFile={skyFile}
                  onSkyLoading={setSkyLoading}
                  onReady={() => {
                    setRenderReady(true);
                    setAtlasFocus(undefined);
                  }}
                  onError={setError}
                  onProgress={(loaded, total, triangles) =>
                    setProgress({ loaded, total, triangles })
                  }
                />
              </Suspense>
            ) : (
              <div className="render-empty">
                <WorldIcon size={42} />
                <h2>No world selected</h2>
                <p>Select or import a world.</p>
                <Button
                  variant="outline"
                  onClick={() => requestImport("archive")}
                  disabled={busy}
                >
                  <FolderOpen /> Open .mcworld
                </Button>
              </div>
            )}
            <div className="render-stage-tools">
              {mapView.enabled && !atlasMode ? (
                <Button variant="outline" onClick={() => canvas.current?.fit()}>
                  Fit map to view
                </Button>
              ) : !mapView.enabled ? (
                viewMenu
              ) : null}
              {!isMapView && !flying && (
                <Button
                  variant="outline"
                  disabled={!ready || saving}
                  aria-pressed={flying}
                  onClick={() => {
                    if (flying) {
                      setFlying(false);
                      return;
                    }
                    setFlying(true);
                    void canvas.current?.beginFlight?.({
                      ...flightSettings,
                      enabled: true,
                      onExit: () => setFlying(false),
                      onFov: (fov) => setFlightSettings((v) => ({ ...v, fov })),
                      onSpeed: (scale) =>
                        setFlightSettings((v) => ({
                          ...v,
                          speed: Math.max(0.1, Math.min(4096, v.speed * scale)),
                        })),
                    });
                  }}
                >
                  {flying ? "Orbit mode" : "First person"}
                </Button>
              )}
            </div>
            {flying && !isMapView && selected && (
              <FlightControls
                canvas={canvas}
                value={flightSettings}
                onChange={setFlightSettings}
                onExit={() => setFlying(false)}
                disabled={saving}
              />
            )}
            <div className="render-caption">
              {progress.loaded
                ? `${progress.loaded} Chunks · ${new Intl.NumberFormat("en-US").format(progress.triangles)} Triangles`
                : flying
                  ? "First-person camera"
                  : "Orthographic camera"}
              <span>
                {mapView.enabled
                  ? "Drag to pan · Scroll to zoom"
                  : flying
                    ? "WASD fly · E/Q height · Scroll speed"
                    : pickingFocus
                      ? "Click to focus · Esc cancel"
                      : "Drag to orbit · Right-drag to pan · Scroll to zoom"}
              </span>
            </div>
          </div>
        )}
        <aside
          className={`render-inspector inspector-tab-${inspectorTab}`}
          aria-label="Render settings"
        >
          {isMapView ? (
            <>
              <div
                className="atlas-mode-switch"
                role="group"
                aria-label="2D renderer"
              >
                <Button
                  variant={atlasMode ? "secondary" : "ghost"}
                  onClick={() => {
                    if (!atlasMode) setAtlasReady(false);
                    setAtlasMode(true);
                  }}
                >
                  Atlas
                </Button>
                <Button
                  variant={!atlasMode ? "secondary" : "ghost"}
                  onClick={() => setAtlasMode(false)}
                >
                  Rendered
                </Button>
              </div>
              <MapPresentationControls
                value={mapView}
                onChange={setMapView}
                onPlace={(id) => setPlacingMarker(id ?? "new")}
                placing={!!placingMarker}
                onCancel={() => setPlacingMarker(undefined)}
              />
            </>
          ) : (
            <>
              <header className="inspector-heading">
                <div>
                  <strong>Scene styling</strong>
                  <span>
                    {selectedLook ? renderPresetLabels[selectedLook] : "Custom"}
                  </span>
                </div>
                <button
                  disabled={saving}
                  data-tooltip="Reset scene styling"
                  aria-label="Reset scene styling"
                  onClick={() => {
                    setSettings(renderDefaults);
                    setSelectedLook("daylight");
                  }}
                >
                  <RefreshCw size={13} />
                </button>
              </header>
              <button
                className="render-style-launcher"
                disabled={saving}
                onClick={() => setStyleOpen(true)}
              >
                Save / load presets …
              </button>
              <nav
                className="inspector-tabs"
                role="tablist"
                aria-label="Scene styling"
              >
                {[
                  ["looks", "Looks"],
                  ["light", "Lighting"],
                  ["atmosphere", "Atmosphere"],
                  ["color", "Color"],
                  ["detail", "Details"],
                  ["output", "Output"],
                ].map(([id, label]) => (
                  <button
                    key={id}
                    role="tab"
                    aria-selected={inspectorTab === id}
                    aria-controls={`inspector-${id}`}
                    id={`inspector-tab-${id}`}
                    onClick={() => setInspectorTab(id)}
                  >
                    {label}
                  </button>
                ))}
              </nav>
              <div
                className="inspector-panel"
                role="tabpanel"
                id={`inspector-${inspectorTab}`}
                aria-labelledby={`inspector-tab-${inspectorTab}`}
              >
                <fieldset disabled={saving} data-inspector="looks">
                  <RenderStyleDialog
                    open={styleOpen}
                    onOpenChange={setStyleOpen}
                    settings={settings}
                    skyFile={skyFile}
                    onApply={(s, file) => {
                      setSettings({ ...s, atmosphere: { ...s.atmosphere } });
                      setSkyFile(file);
                      setSelectedLook("");
                    }}
                  />
                  <LookLibrary
                    selected={selectedLook}
                    disabled={saving}
                    onSelect={(id) => {
                      setSelectedLook(id);
                      setSettings((s) => applyRenderPreset(s, id));
                    }}
                    onLearn={
                      ready
                        ? async () => {
                            const current = canvas.current;
                            if (!current) throw new Error("Model unavailable");
                            const analysis = await current.analyse();
                            if (canvas.current !== current)
                              throw new Error("The model view has changed.");
                            rememberDirectionExample(analysis, selectedLook);
                          }
                        : undefined
                    }
                  />
                  <button
                    className="inspector-next"
                    onClick={() => setInspectorTab("color")}
                  >
                    Refine colors
                  </button>
                </fieldset>
                <AtmosphereControls
                  value={settings.atmosphere}
                  onChange={(value) => update("atmosphere", value)}
                  disabled={saving}
                  cloudEditor={cloudEditor}
                  onCloudPlace={() => {
                    setPlacingCloud((p) => !p);
                    setPlacingLight(false);
                    setPickingFocus(false);
                  }}
                  onCloudMode={setCloudMode}
                  animation={
                    cinematic && cinemaDocument
                      ? {
                          time: cinemaDocument.time,
                          duration: cinemaDocument.sequence.duration,
                          fps: cinemaDocument.sequence.fps,
                          onScrub: (t) => canvas.current?.seekTime?.(t),
                        }
                      : undefined
                  }
                />
                {!cinematic && (
                  <DepthOfFieldControls
                    value={settings}
                    onChange={setSettings}
                    disabled={saving || isMapView}
                    picking={pickingFocus}
                    guides={focusGuides}
                    onGuides={setFocusGuides}
                    onPick={() => {
                      if (flying) setFlying(false);
                      setPickingFocus((p) => !p);
                      setPlacingLight(false);
                    }}
                  />
                )}
                <StudioLightControls
                  lights={settings.studioLights ?? []}
                  selected={selectedLight}
                  onSelect={lightEditor.onSelect}
                  onChange={(studioLights) =>
                    setSettings((s) => ({ ...s, studioLights }))
                  }
                  onPlace={(id) => {
                    setSelectedLight(id);
                    setPlacingLight((p) => !p);
                    setPickingFocus(false);
                  }}
                  placing={placingLight}
                  target={lightTarget}
                  onTarget={setLightTarget}
                  disabled={saving}
                />
                <LightingControls
                  value={settings}
                  disabled={saving}
                  onChange={(s) => {
                    setSettings(s);
                    setSelectedLook("");
                  }}
                />
                <DaytimeControl
                  value={settings}
                  disabled={saving}
                  onChange={(s) => {
                    setSettings(s);
                    setSelectedLook("");
                  }}
                />
                <fieldset disabled={saving} data-inspector="light">
                  <legend>
                    <Sun size={15} /> Light source
                  </legend>
                  {slider("Sun direction", "azimuth", 0, 360, 1)}
                  {slider("Sun elevation", "elevation", 5, 85, 1)}
                  {slider("Sunlight", "sun", 0, 6, 0.1)}
                  {slider("Ambient light", "ambient", 0.1, 3, 0.05)}
                  {slider("Exposure", "exposure", 0.3, 1.8, 0.05)}
                  <label className="render-color">
                    Light color
                    <StudioInput
                      aria-label="Light color"
                      type="color"
                      value={settings.warmth}
                      onChange={(e) => update("warmth", e.target.value)}
                    />
                  </label>
                </fieldset>
                <fieldset disabled={saving} data-inspector="light">
                  <legend>Sky &amp; environment</legend>
                  <label className="render-control">
                    Skybox
                    <select
                      aria-label="Skybox"
                      value={settings.sky}
                      onChange={(e) => update("sky", e.target.value)}
                    >
                      {Object.entries(renderSkies).map(([id, sky]) => (
                        <option key={id} value={id}>
                          {sky.label}
                        </option>
                      ))}
                      <option value="none">Solid background</option>
                      {skyFile && (
                        <option value="custom">{skyFile.name}</option>
                      )}
                    </select>
                  </label>
                  <label className="render-control">
                    Custom HDR skybox
                    <StudioInput
                      aria-label="Custom HDR skybox"
                      type="file"
                      accept=".hdr"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          setSkyFile(file);
                          update("sky", "custom");
                        }
                      }}
                    />
                  </label>
                  {slider("Sky rotation", "skyRotation", 0, 360, 1)}
                  {slider("Horizon correction", "skyElevation", -30, 30, 1)}
                  {slider("Sky brightness", "skyBrightness", 0.05, 2, 0.05)}
                  {slider("Sky lighting", "environment", 0, 3, 0.05)}
                  <label className="render-check">
                    <StudioInput
                      type="checkbox"
                      checked={settings.sunLinked}
                      onChange={(e) => update("sunLinked", e.target.checked)}
                    />{" "}
                    Align sun with skybox
                  </label>
                  <p>
                    Real 360° HDR panoramas from{" "}
                    <a
                      href="https://polyhaven.com/license"
                      target="_blank"
                      rel="noreferrer"
                    >
                      Poly Haven · CC0
                    </a>
                    . Offline; lighting stays active.
                  </p>
                </fieldset>
                <fieldset disabled={saving} data-inspector="color">
                  <legend>Color grading</legend>
                  <label className="render-check">
                    <StudioInput
                      type="checkbox"
                      aria-label="Enable color grading"
                      checked={settings.grading}
                      onChange={(e) => update("grading", e.target.checked)}
                    />{" "}
                    Enable color grading
                  </label>
                  <p>Toggle to compare.</p>
                  {slider("Saturation", "saturation", 0, 1.8, 0.02)}
                  {slider("Vibrance", "vibrance", -0.5, 0.6, 0.02)}
                  {slider("Contrast", "contrast", 0.7, 1.4, 0.02)}
                  {slider("Temperature", "temperature", -1, 1, 0.02)}
                  {slider("Green / Magenta", "tint", -1, 1, 0.02)}
                </fieldset>
                <fieldset disabled={saving} data-inspector="color">
                  <legend>Tone &amp; image style</legend>
                  {slider("Highlights", "highlights", -1, 1, 0.02)}
                  {slider("Shadows", "shadows", -1, 1, 0.02)}
                  {slider("Vignette", "vignette", 0, 0.6, 0.02)}
                </fieldset>
                <fieldset disabled={saving} data-inspector="detail">
                  <legend>Water</legend>
                  {(
                    [
                      ["Shallow water color", "waterShallowColor"],
                      ["Deep water color", "waterDeepColor"],
                      ["Depth light color", "waterDeepLightColor"],
                    ] as const
                  ).map(([label, key]) => (
                    <label className="render-color" key={key}>
                      {label}
                      <StudioInput
                        aria-label={label}
                        type="color"
                        value={settings[key]}
                        onChange={(e) =>
                          setSettings((s) => ({ ...s, [key]: e.target.value }))
                        }
                      />
                    </label>
                  ))}
                  {slider(
                    "Water palette strength",
                    "waterColorStrength",
                    0,
                    1,
                    0.02,
                  )}
                  {slider(
                    "Color depth (blocks)",
                    "waterColorDepth",
                    0.25,
                    100,
                    0.25,
                  )}
                  {slider(
                    "Depth light intensity",
                    "waterDeepLight",
                    0,
                    2,
                    0.02,
                  )}
                  {slider("Water surface opacity", "waterOpacity", 0, 1, 0.02)}
                  {slider(
                    "Overall water transparency",
                    "waterTransparency",
                    0,
                    1,
                    0.02,
                  )}
                  {slider(
                    "Underwater visibility (blocks)",
                    "waterVisibility",
                    0.25,
                    100,
                    0.25,
                  )}
                  {slider(
                    "Shore softness (blocks)",
                    "waterShoreSoftness",
                    0.01,
                    8,
                    0.1,
                  )}
                  {slider("Water reflections", "waterReflections", 0, 3, 0.05)}
                  {slider("Water roughness", "waterRoughness", 0.04, 1, 0.02)}
                  {slider("Water ripples", "waterWaves", 0, 1, 0.02)}
                  {slider("Ripple scale", "waterWaveScale", 0.1, 8, 0.1)}
                  {slider(
                    "Water texture detail",
                    "waterTextureStrength",
                    0,
                    2,
                    0.05,
                  )}
                  <label className="render-color">
                    Water tint
                    <StudioInput
                      aria-label="Water tint"
                      type="color"
                      value={settings.waterTint}
                      onChange={(e) =>
                        setSettings((s) => ({
                          ...s,
                          waterTint: e.target.value,
                        }))
                      }
                    />
                  </label>
                  <p>
                    Palette 0: pack colors. Transparency 1: invisible. Water
                    only.
                  </p>
                </fieldset>
                <fieldset disabled={saving} data-inspector="detail">
                  <legend>Detail &amp; shadows</legend>
                  <div className="render-presets render-quality">
                    {Object.entries(renderQuality).map(([id, quality]) => (
                      <button
                        key={id}
                        aria-pressed={settings.quality === id}
                        onClick={() =>
                          setSettings((s) => ({
                            ...s,
                            quality: id as RenderSettings["quality"],
                            shadow: quality.shadow,
                          }))
                        }
                      >
                        {quality.label}
                      </button>
                    ))}
                  </div>
                  {slider("Ambient occlusion", "ao", 0, 1.5, 0.05)}
                  {slider("Contact shadow radius", "aoRadius", 0.25, 4, 0.25)}
                  {slider("Shadow softness", "softness", 0.5, 5, 0.25)}
                  {slider("HDR reflections", "reflections", 0, 3, 0.05)}
                  {slider("Material roughness", "roughness", 0.1, 1, 0.05)}
                  {slider("Glow / Bloom", "bloom", 0, 1, 0.02)}
                  {slider("Sharpness", "sharpen", 0, 0.8, 0.02)}
                  {slider(
                    "Protect fine detail",
                    "detailProtection",
                    0,
                    1,
                    0.02,
                  )}
                  <label className="render-control">
                    Shadow detail
                    <select
                      aria-label="Shadow detail"
                      value={settings.shadow}
                      onChange={(e) => update("shadow", Number(e.target.value))}
                    >
                      <option value={1024}>Fast · 1024</option>
                      <option value={2048}>Balanced · 2048</option>
                      <option value={4096}>Fine · 4096</option>
                    </select>
                  </label>
                  <p>Contact shadows refine when the camera stops.</p>
                </fieldset>
                <fieldset disabled={saving} data-inspector="output">
                  <legend>Image output</legend>
                  {slider("Fit margin", "frameMargin", 0.02, 0.4, 0.01)}
                  {slider(
                    "Vertical frame shift",
                    "frameVerticalShift",
                    -0.3,
                    0.3,
                    0.01,
                  )}
                  <p>Negative shift raises the map. Fit keeps it in frame.</p>
                  <label className="render-control">
                    Image sampling
                    <select
                      aria-label="Image sampling"
                      value={settings.exportSamples}
                      onChange={(e) =>
                        setSettings((s) => ({
                          ...s,
                          exportSamples: +e.target.value,
                        }))
                      }
                    >
                      <option value={1}>Quick · 1 sample</option>
                      <option value={4}>Clean · 4 samples</option>
                      <option value={16}>Print · 16 samples</option>
                    </select>
                  </label>
                  <p>PNG anti-aliasing. More samples take longer.</p>
                  <label className="render-control">
                    Image format
                    <select
                      aria-label="Image format"
                      value={settings.aspect}
                      onChange={(e) => update("aspect", Number(e.target.value))}
                    >
                      {![16 / 9, 1, 3 / 4].includes(settings.aspect) && (
                        <option value={settings.aspect}>
                          Custom · {dimensions.width} × {dimensions.height}
                        </option>
                      )}
                      <option value={16 / 9}>Widescreen · 16:9</option>
                      <option value={1}>Square · 1:1</option>
                      <option value={3 / 4}>Portrait · 3:4</option>
                    </select>
                  </label>
                  <label className="render-control">
                    Resolution
                    <select
                      aria-label="Resolution"
                      value={settings.edge}
                      onChange={(e) => update("edge", Number(e.target.value))}
                    >
                      {![1920, 2560, 3840].includes(settings.edge) && (
                        <option value={settings.edge}>
                          {settings.edge} Pixels · Custom
                        </option>
                      )}
                      <option value={1920}>1920 pixels</option>
                      <option value={2560}>2560 pixels</option>
                      <option value={3840}>4K · 3840 pixels</option>
                    </select>
                  </label>
                  <label className="render-color">
                    Background
                    <StudioInput
                      type="color"
                      value={settings.background}
                      onChange={(e) => update("background", e.target.value)}
                      disabled={settings.transparent || settings.sky !== "none"}
                    />
                  </label>
                  <label className="render-check">
                    <StudioInput
                      type="checkbox"
                      checked={settings.transparent}
                      onChange={(e) => update("transparent", e.target.checked)}
                    />{" "}
                    Transparent background
                  </label>
                  <p>
                    {dimensions.width} × {dimensions.height} pixels · PNG
                  </p>
                </fieldset>
              </div>
              <footer className="inspector-footer">
                {settings.quality === "fine"
                  ? "Fine"
                  : settings.quality === "balanced"
                    ? "Balanced"
                    : "Fast"}{" "}
                Preview · {dimensions.width} × {dimensions.height}
              </footer>
            </>
          )}
        </aside>
        {!error && (busy || (selected && !ready) || skyLoading) && (
          <EditorLoadingOverlay
            label={
              busy
                ? "Reading world and resources …"
                : skyLoading
                  ? "Preparing scene styling …"
                  : "Loading world …"
            }
            loaded={progress.loaded}
            total={busy || skyLoading ? 0 : progress.total}
          />
        )}
      </div>
      {panoramaProgress !== undefined && (
        <div className="render-feedback" role="status">
          Rendering panorama: {Math.round(panoramaProgress * 100)} %
          <progress
            value={panoramaProgress}
            max={1}
            aria-label="Panorama progress"
          />
          <Button
            variant="outline"
            onClick={() => {
              panoramaCancelled.current = true;
            }}
          >
            Cancel panorama
          </Button>
        </div>
      )}
      {error && (
        <FeedbackFlyout title="Render Editor">
          {error}
        </FeedbackFlyout>
      )}
      <ResourceTraceDialog
        trace={resourceTrace}
        onClose={() => setResourceTrace(null)}
      />
    </section>
  );
}

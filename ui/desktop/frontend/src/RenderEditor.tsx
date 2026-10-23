import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { cinematicSignature } from "./cinematic-signature";
import {
  Aperture,
  FolderOpen,
  ImageDown,
  LoaderCircle,
  RefreshCw,
  Sun,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { api, type RenderWorld, type World } from "./bridge";
import {
  renderDefaults,
  renderDimensions,
  applyRenderPreset,
  renderPresetLabels,
  renderQuality,
  type RenderSettings,
} from "./render-settings";
import type { RenderCanvasHandle } from "./RenderCanvas";
import { renderSkies } from "./render-skies";
import LightingControls from "./LightingControls";
import AutoDirectionControls from "./AutoDirectionControls";
import DaytimeControl from "./DaytimeControl";
import LookLibrary from "./LookLibrary";
import EditorLoadingOverlay from "./EditorLoadingOverlay";
import MapPresentationControls from "./MapPresentationControls";
import { mapDefaults, mapRenderSettings } from "./map-presentation";
import {
  captureStudioSky,
  restoreStudioSky,
  type StudioDocument,
  type StudioSession,
  type CinematicViewState,
} from "./studio-document";
import type { CameraPose } from "./cinematic";
import { rememberDirectionExample } from "./direction-learning";
import "./render-inspector.css";
import {
  useEditorCommands,
  dispatchEditorCommand,
  idleEditor,
  type EditorStatus,
} from "./editor-commands";
import { ImportWorldDialog, MediaExportDialog } from "./MediaDialogs";
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
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
  ) => void;
}) {
  const [selected, setSelected] = useState(
    loadedProject?.world.id ??
      (project ? project.worldId : (activeWorldId ?? worlds[0]?.id ?? "")),
  );
  const [importKind, setImportKind] = useState<"archive" | "folder">("archive"),
    [importOpen, setImportOpen] = useState(false),
    [exportOpen, setExportOpen] = useState(false);
  const [imported, setImported] = useState<RenderWorld | undefined>(
    loadedProject?.world,
  );
  const [exportPreview, setExportPreview] = useState<string>();
  const [panoramaOpen, setPanoramaOpen] = useState(false);
  const [panoramaWidth, setPanoramaWidth] = useState(2048);
  const [panoramaName, setPanoramaName] = useState("Panorama");
  const [panoramaProgress, setPanoramaProgress] = useState<number>();
  const panoramaCancelled = useRef(false);
  const [settings, setSettings] = useState<RenderSettings>(() => {
    if (loadedProject) return loadedProject.document.settings;
    if (!project) return renderDefaults;
    const defaults = {
      ...applyRenderPreset(renderDefaults, project.look),
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
  const [selectedLight, setSelectedLight] = useState<string>();
  const [placingLight, setPlacingLight] = useState(false);
  const [lightTarget, setLightTarget] = useState(false);
  const [pickingFocus, setPickingFocus] = useState(false);
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
  const onFocusPick = (distance: number) => {
    setSettings((s) => ({
      ...s,
      dofEnabled: true,
      dofDistance: Math.max(0.001, Math.min(100000, distance)),
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
  const [viewPose, setViewPose] = useState<
    (CameraPose & { zoom?: number }) | undefined
  >(loadedProject?.document.view);
  const [viewEdited, setViewEdited] = useState(false);
  const viewRevision = useRef(0);
  const [projectPath, setProjectPath] = useState(loadedProject?.path);
  const [savedSignature, setSavedSignature] = useState<string>();
  const isMapView = !cinematic && mapView.enabled;
  const canvasSettings = mapRenderSettings(settings, {
    ...mapView,
    enabled: isMapView,
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
  const [ready, setReady] = useState(false),
    [busy, setBusy] = useState(false),
    [saving, setSaving] = useState(false),
    [error, setError] = useState(""),
    [result, setResult] = useState("");
  const [generation, setGeneration] = useState(0);
  const canvas = useRef<RenderCanvasHandle>(null),
    mounted = useRef(true),
    pending = useRef(false),
    importId = useRef("");
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
  const sourceName =
    loadedProject?.document.name ||
    project?.name ||
    (imported?.id === selected
      ? imported.name
      : (worlds.find((w) => w.id === selected)?.name ?? "Map render"));
  const signature = JSON.stringify({
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
  async function saveStudio(forceAs = false) {
    if (!ready || !selected)
      throw new Error(
        "The world must finish loading before the project can be saved.",
      );
    if (saving || busy) return false;
    const startingSignature = signature;
    const camera = canvas.current?.snapshot();
    const capturedViewRevision = viewRevision.current;
    const capturedCinema = canvas.current?.studioState?.() ?? cinemaDocument;
    if (camera) setViewPose(camera);
    if (capturedCinema) setCinemaDocument(capturedCinema);
    const capturedSignature = JSON.stringify({
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
      const path = await api.SaveStudioProject(
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
      return (
        (latestSignature.current === capturedSignature ||
          latestSignature.current === startingSignature) &&
        viewRevision.current === capturedViewRevision
      );
    } finally {
      if (mounted.current) setSaving(false);
    }
  }
  useEffect(() => {
    onStudioSession?.({
      name: sourceName,
      dirty: documentDirty,
      busy: saving,
      save: saveStudio,
    });
  }, [
    signature,
    viewEdited,
    documentDirty,
    busy,
    saving,
    ready,
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
      } catch {

      }
  }, [settings, project]);
  useEffect(() => {
    if (selected && selected !== imported?.id)
      onProjectSourceChange?.(selected);
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
    else if (command === "fit" && ready) canvas.current?.fit();
    else if (command === "isometric" && ready) canvas.current?.view(45, 35);
    else if (command === "top" && ready) canvas.current?.view(0, 89);
    else if (command === "image-panel") setShowLighting((v) => !v);
    else if (command === "save-project")
      void saveStudio().catch((e) => setError(String(e)));
    else if (command === "save-project-as")
      void saveStudio(true).catch((e) => setError(String(e)));
    else if (command === "reset-look") {
      setSettings(renderDefaults);
      setSelectedLook("daylight");
    }
  });
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      if (pending.current) void api.CancelRenderImport();

    };
  }, []);
  useEffect(() => {
    if (!project && !selected && worlds.length)
      setSelected(activeWorldId ?? worlds[0].id);
  }, [worlds, activeWorldId, selected]);
  useEffect(() => {
    setReady(false);
    setError("");
    setResult("");
    setProgress({ loaded: 0, total: 0, triangles: 0 });
  }, [selected, generation]);
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
      const doc = await api.OpenRenderWorld(kind);
      if (!mounted.current) {
        if (doc.id) void api.CloseRenderWorld(doc.id);
        return;
      }
      if (doc.id) {
        importId.current = doc.id;
        setImported(doc);
        setSelected(doc.id);
        onProjectSourceChange?.(doc.id, kind);
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
  function save() {
    setExportPreview(canvas.current?.preview?.());
    setExportOpen(true);
  }
  async function savePNG(filename = sourceName) {
    setSaving(true);
    setError("");
    setResult("");
    try {
      const data = await canvas.current!.capture();
      if (!mounted.current) return;
      const path = await api.SaveRenderPNG(filename, data);
      if (mounted.current) setResult(path);
    } catch (e) {
      if (mounted.current) setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (mounted.current) setSaving(false);
    }
  }
  async function savePanorama() {
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
      const path = await api.SaveRenderPNG(
        panoramaName.trim() || "Panorama",
        data,
      );
      if (mounted.current) setResult(path);
    } catch (error) {
      if (mounted.current && !panoramaCancelled.current)
        setError(error instanceof Error ? error.message : String(error));
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
    return (
      <div className="render-control">
        <span>
          {label}
          <input
            className="inspector-number"
            aria-label={`${label} Value`}
            type="number"
            min={min}
            max={max}
            step={step}
            value={settings[key]}
            disabled={saving}
            onChange={(e) => {
              const n = Number(e.target.value);
              if (Number.isFinite(n))
                update(key, Math.min(max, Math.max(min, n)));
            }}
          />
        </span>
        <input
          type="range"
          aria-label={label}
          disabled={
            saving ||
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
        format="png"
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
        onExport={(name) => void savePNG(name)}
      />
      <Dialog open={panoramaOpen} onOpenChange={setPanoramaOpen}>
        <DialogContent className="studio-dialog">
          <DialogHeader>
            <DialogTitle>Export 360° panorama</DialogTitle>
            <DialogDescription>
              A full spherical view as a 2:1 equirectangular PNG. The current camera determines position and orientation.
            </DialogDescription>
          </DialogHeader>
          <label className="render-control">
            File name
            <input
              aria-label="Panorama file name"
              maxLength={100}
              value={panoramaName}
              onChange={(e) => setPanoramaName(e.target.value)}
            />
          </label>
          <label className="render-control">
            Resolution
            <select
              aria-label="Panorama resolution"
              value={panoramaWidth}
              onChange={(e) => setPanoramaWidth(+e.target.value)}
            >
              {[1024, 2048, 4096].map((width) => (
                <option value={width} key={width}>
                  {width} × {width / 2} pixels
                </option>
              ))}
            </select>
          </label>
          <p>
            Lighting, sky, atmosphere and color grading are included. Depth of field and vignette are disabled for the spherical view.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPanoramaOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void savePanorama()}>
              Render panorama
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <div className="render-header">
        <div>
          <h1>
            <Aperture /> {cinematic ? "Cinematic editor" : "Render editor"}
          </h1>
          <p>
            {cinematic
              ? "Compose camera sequences with coordinated motion, framing and lighting."
              : "Your map with light and shadow. Includes custom blocks; excludes entities."}
          </p>
        </div>
        {cinematic && (
          <div
            className="cinema-workspace-switch"
            aria-label="Cinematic workspace"
          >
            <Button
              disabled={saving}
              variant="outline"
              aria-label="Edit camera and sequence"
              aria-pressed={!showLighting}
              onClick={() => setShowLighting(false)}
            >
              Camera &amp; sequence
            </Button>
            <Button
              disabled={saving}
              variant="outline"
              aria-label="Edit lighting"
              onClick={() => {
                setInspectorTab("light");
                setShowLighting(true);
              }}
            >
              Lighting
            </Button>
            <Button
              disabled={saving}
              variant="outline"
              aria-label="Edit clouds and fog"
              onClick={() => {
                setInspectorTab("atmosphere");
                setShowLighting(true);
              }}
            >
              Atmosphere
            </Button>
            <Button
              disabled={saving}
              variant="outline"
              aria-label="Edit colors"
              onClick={() => {
                setInspectorTab("color");
                setShowLighting(true);
              }}
            >
              Color
            </Button>
            <Button
              disabled={saving}
              variant="outline"
              aria-label="Edit scene looks"
              aria-pressed={showLighting && inspectorTab === "looks"}
              onClick={() => {
                setInspectorTab("looks");
                setShowLighting(true);
              }}
            >
              Scene looks
            </Button>
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
          <Button
            variant="outline"
            disabled={busy || saving}
            onClick={onStudioOpen}
          >
            Open project
          </Button>
        )}
        <Button
          variant="outline"
          disabled={!ready || busy || saving}
          onClick={() => void saveStudio().catch((e) => setError(String(e)))}
        >
          Save project
        </Button>
        {!cinematic && (
          <Button
            variant="outline"
            disabled={!ready || saving || busy || skyLoading || isMapView}
            onClick={() => setPanoramaOpen(true)}
          >
            360° panorama
          </Button>
        )}
        <Button
          onClick={() => void save()}
          disabled={!ready || saving || busy || skyLoading}
        >
          {saving ? <LoaderCircle className="spin" /> : <ImageDown />}
          {saving ? "Rendering image …" : "Export PNG"}
        </Button>
      </div>
      <div className="render-source">
        {!cinematic && (
          <div
            role="group"
            aria-label="Editor view"
            style={{ display: "flex", gap: 4 }}
          >
            <Button
              variant="outline"
              aria-pressed={!mapView.enabled}
              disabled={saving || busy}
              onClick={() => setMapView((s) => ({ ...s, enabled: false }))}
            >
              3D view
            </Button>
            <Button
              variant="outline"
              aria-pressed={mapView.enabled}
              disabled={saving || busy}
              onClick={() => setMapView((s) => ({ ...s, enabled: true }))}
            >
              2D map
            </Button>
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
          </select>
        </label>
        <Button
          variant="outline"
          onClick={() => requestImport("archive")}
          disabled={busy || saving}
        >
          <FolderOpen /> Open world file
        </Button>
        <Button
          variant="ghost"
          onClick={() => requestImport("folder")}
          disabled={busy || saving}
        >
          World folder
        </Button>
        {busy ? (
          <Button variant="ghost" onClick={() => void api.CancelRenderImport()}>
            Cancel
          </Button>
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
              <div className="render-empty">
                Loading cinematic studio …
              </div>
            }
          >
            <CinematicStudio
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
              onReady={() => setReady(true)}
              onError={setError}
              onProgress={(loaded, total, triangles) =>
                setProgress({ loaded, total, triangles })
              }
            />
          </Suspense>
        ) : (
          <div className={`render-stage ${isMapView ? "map-2d" : ""}`}>
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
                  key={`${selected}:${generation}`}
                  ref={canvas}
                  worldId={selected}
                  settings={canvasSettings}
                  lightEditor={saving || isMapView ? undefined : lightEditor}
                  cloudEditor={saving || isMapView ? undefined : cloudEditor}
                  pickingFocus={pickingFocus && !saving && !isMapView}
                  onFocusPick={onFocusPick}
                  mapView={mapView}
                  initialView={viewPose}
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
                  onReady={() => setReady(true)}
                  onError={setError}
                  onProgress={(loaded, total, triangles) =>
                    setProgress({ loaded, total, triangles })
                  }
                />
              </Suspense>
            ) : (
              <div className="render-empty">
                <Aperture size={42} />
                <h2>Bring a world into focus</h2>
                <p>
                  Select a captured map or open a Bedrock world file.
                </p>
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
              {mapView.enabled ? (
                <Button variant="outline" onClick={() => canvas.current?.fit()}>
                  Fit map to view
                </Button>
              ) : (
                viewMenu
              )}
            </div>
            <div className="render-caption">
              {progress.loaded
                ? `${progress.loaded} Chunks · ${new Intl.NumberFormat("en-US").format(progress.triangles)} Triangles`
                : "Orthographic camera"}
              <span>
                {mapView.enabled
                  ? "Drag to pan · Scroll to zoom"
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
            <MapPresentationControls
              value={mapView}
              onChange={setMapView}
              onPlace={(id) => setPlacingMarker(id ?? "new")}
              placing={!!placingMarker}
              onCancel={() => setPlacingMarker(undefined)}
            />
          ) : (
            <>
              <header className="inspector-heading">
                <div>
                  <strong>Scene styling</strong>
                  <span>
                    {selectedLook
                      ? renderPresetLabels[selectedLook]
                      : "Custom"}
                  </span>
                </div>
                <button
                  disabled={saving}
                  title="Reset scene styling"
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
                              throw new Error(
                                "The model view has changed.",
                              );
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
                    onPick={() => {
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
                    <input
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
                    <input
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
                    <input
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
                    . Available offline. Sky lighting is retained even with a transparent background.
                  </p>
                </fieldset>
                <fieldset disabled={saving} data-inspector="color">
                  <legend>Color grading</legend>
                  <label className="render-check">
                    <input
                      type="checkbox"
                      aria-label="Enable color grading"
                      checked={settings.grading}
                      onChange={(e) => update("grading", e.target.checked)}
                    />{" "}
                    Enable color grading
                  </label>
                  <p>
                    Turn off grading for a direct before-and-after comparison.
                  </p>
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
                  <p>
                    The preview stays responsive while you move the camera. Fine contact shadows appear when it stops.
                  </p>
                </fieldset>
                <fieldset disabled={saving} data-inspector="output">
                  <legend>Image output</legend>
                  <label className="render-control">
                    Image format
                    <select
                      aria-label="Image format"
                      value={settings.aspect}
                      onChange={(e) => update("aspect", Number(e.target.value))}
                    >
                      {![16 / 9, 1, 3 / 4].includes(settings.aspect) && (
                        <option value={settings.aspect}>
                          Custom · {dimensions.width} ×{" "}
                          {dimensions.height}
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
                    <input
                      type="color"
                      value={settings.background}
                      onChange={(e) => update("background", e.target.value)}
                      disabled={settings.transparent || settings.sky !== "none"}
                    />
                  </label>
                  <label className="render-check">
                    <input
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
        <div className="render-feedback error" role="alert">
          {error}
        </div>
      )}
      {result && (
        <div className="render-feedback" role="status">
          Saved: {result}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void api.OpenResultFolder(result)}
          >
            Open folder
          </Button>
        </div>
      )}
    </section>
  );
}

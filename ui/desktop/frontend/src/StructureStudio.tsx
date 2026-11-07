import FeedbackFlyout from "./FeedbackFlyout";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Blocks,
  Plus,
  FolderOpen,
  Download,
  Save,
  Undo2,
  Redo2,
  RotateCw,
  FlipHorizontal,
  Copy,
  Scissors,
  ClipboardPaste,
  Trash2,
  MousePointer2,
  Pipette,
  Paintbrush,
  Layers,
  Box as BoxIcon,
  ChevronLeft,
  ChevronRight,
  Search,
  Printer,
  FileSpreadsheet,
  Move3D,
  AlertTriangle,
  X,
  Settings2,
} from "lucide-react";
import { Button } from "./components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./components/ui/dialog";
import { useDocumentScope } from "./document-scope";
import { useWorkspaceActions } from "./workspace-actions";
import { useEditorCommands, type EditorStatus } from "./editor-commands";
import { on, type State } from "./bridge";
import {
  structures,
  exportDefaults,
  normalized,
  volume,
  blockColor,
  type StructureDocument,
  type Vec3,
  type Box,
  type BlockState,
  type StructureEdit,
  type Materials,
  type Comparison,
  type Recovery,
  type Job,
  type ExportOptions,
  type Layer,
} from "./structure-api";
import StructureViewport from "./StructureViewport";
import StructureLibrary from "./StructureLibrary";
import EditorLoadingOverlay from "./EditorLoadingOverlay";
import "./structure-studio.css";

type Props = { worlds: State["worlds"]; onStatus: (s: EditorStatus) => void };
const initialBox: Box = { min: [0, 0, 0], max: [0, 0, 0] };
const fmt = (n: number) => n.toLocaleString();
function VectorField({
  label,
  value,
  onChange,
  min,
  max,
}: {
  label: string;
  value: Vec3;
  onChange: (v: Vec3) => void;
  min?: number;
  max?: Vec3;
}) {
  return (
    <fieldset className="structure-vector">
      <legend>{label}</legend>
      {["X", "Y", "Z"].map((axis, i) => (
        <label key={axis}>
          <span>{axis}</span>
          <input
            aria-label={`${label} ${axis}`}
            type="number"
            step="1"
            min={min}
            max={max?.[i]}
            value={value[i]}
            onChange={(e) => {
              const next = [...value] as Vec3;
              const n = Number(e.target.value);
              if (Number.isSafeInteger(n)) {
                next[i] = n;
                onChange(next);
              }
            }}
          />
        </label>
      ))}
    </fieldset>
  );
}
export default function StructureStudio({ worlds, onStatus }: Props) {
  const scope = useDocumentScope();
  const [propertiesOpen, setPropertiesOpen] = useState(false);
  const [doc, setDoc] = useState<StructureDocument>();
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [message, setMessage] = useState("");
  const operationRef = useRef("");
  const busyRef = useRef(false);
  const [selection, setSelection] = useState<Box>(initialBox),
    [corner, setCorner] = useState<0 | 1>(0),
    [regions, setRegions] = useState<Box[]>([]);
  const [layer, setLayer] = useState(0),
    [layerOnly, setLayerOnly] = useState(false),
    [adjacent, setAdjacent] = useState(true),
    [top, setTop] = useState(false);
  const [tool, setTool] = useState("select"),
    [brush, setBrush] = useState<BlockState>({
      name: "minecraft:stone",
      states: {},
    }),
    [stateText, setStateText] = useState("{}"),
    [paletteQuery, setPaletteQuery] = useState("");
  const [catalog, setCatalog] = useState<BlockState[]>([]);
  useEffect(() => {
    if (!doc) return;
    let stopped = false;
    const timer = window.setTimeout(
      () =>
        void structures
          .GetStructureBlockCatalog(paletteQuery)
          .then((rows) => {
            if (!stopped) setCatalog(rows);
          })
          .catch(() => {}),
      150,
    );
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [doc?.id, paletteQuery]);
  const [panel, setPanel] = useState("selection"),
    [materials, setMaterials] = useState<Materials>(),
    [materialScope, setMaterialScope] = useState("all"),
    [layerData, setLayerData] = useState<Layer>(),
    [layerX, setLayerX] = useState(0),
    [layerZ, setLayerZ] = useState(0);
  const [recovery, setRecovery] = useState<Recovery[]>([]);
  const [libraryOpen, setLibraryOpen] = useState(false);
  const [libraryQuery, setLibraryQuery] = useState("");
  const [documentLoading, setDocumentLoading] = useState(!!scope.sourceId);
  const [dialog, setDialog] = useState<
    "" | "new" | "import" | "place" | "transform"
  >("");
  const [name, setName] = useState("Untitled structure"),
    [size, setSize] = useState<Vec3>([32, 32, 32]);
  const [importPath, setImportPath] = useState(""),
    [importWorld, setImportWorld] = useState(""),
    [importBounds, setImportBounds] = useState<Box>({
      min: [0, 0, 0],
      max: [31, 63, 31],
    }),
    [dimension, setDimension] = useState(0);
  const [regionName, setRegionName] = useState("Selection 1"),
    [replaceFrom, setReplaceFrom] = useState("minecraft:stone");
  const [options, setOptions] = useState<ExportOptions>(exportDefaults),
    [job, setJob] = useState<Job>();
  const [targetID, setTargetID] = useState(""),
    [target, setTarget] = useState<StructureDocument>(),
    [placementAnchor, setPlacementAnchor] = useState<Vec3>([0, 64, 0]),
    [placementMode, setPlacementMode] = useState("ignore-air"),
    [placementSource, setPlacementSource] = useState("");
  const [compareOffset, setCompareOffset] = useState<Vec3>([0, 0, 0]),
    [comparison, setComparison] = useState<Comparison>(),
    [diffFilter, setDiffFilter] = useState("all"),
    [ghost, setGhost] = useState(false);
  const [transform, setTransform] = useState<StructureEdit>({
      kind: "rotate",
      turns: 1,
    }),
    [allowUnsupported, setAllowUnsupported] = useState(false);
  const latest = useRef({
    doc,
    busy,
    selection,
    regions,
    brush,
    scope,
    options,
    layer,
    layerOnly,
    stateText,
  });
  latest.current = {
    doc,
    busy,
    selection,
    regions,
    brush,
    scope,
    options,
    layer,
    layerOnly,
    stateText,
  };
  const mounted = useRef(true);
  useEffect(
    () => () => {
      mounted.current = false;
    },
    [],
  );
  const run = useCallback(
    async <T,>(action: () => Promise<T>): Promise<T | undefined> => {
      if (busyRef.current) return;
      busyRef.current = true;
      operationRef.current = crypto.randomUUID();
      setBusy(true);
      setError("");
      setMessage("");
      try {
        return await action();
      } catch (e) {
        setError(String(e));
        return undefined;
      } finally {
        busyRef.current = false;
        setBusy(false);
      }
    },
    [],
  );
  const refreshLibrary = useCallback(
    () =>
      structures
        .ListStructureRecovery()
        .then(setRecovery)
        .catch((e) => setError(String(e))),
    [],
  );
  useEffect(() => {
    void refreshLibrary();
    return on<{ id: string; name: string }>(
      "structure-created",
      () => void refreshLibrary(),
    );
  }, [refreshLibrary]);
  useEffect(() => {
    if (!scope.sourceId) return;
    let stopped = false;
    setDocumentLoading(true);
    void structures
      .GetStructure(scope.sourceId)
      .catch(() => structures.RecoverStructure(scope.sourceId!))
      .then((d) => {
        if (!stopped) {
          setDoc(d);
          setSelection({
            min: [0, 0, 0],
            max: d.size.map((n) => n - 1) as Vec3,
          });
          setLayer(
            typeof d.view.layer === "number"
              ? Math.max(0, Math.min(d.size[1] - 1, d.view.layer))
              : 0,
          );
          setLayerOnly(d.view.layerOnly === true);
        }
      })
      .catch((e) => { if (!stopped) setError(String(e)); })
      .finally(() => { if (!stopped) setDocumentLoading(false); });
    return () => {
      stopped = true;
    };
  }, [scope.sourceId]);
  const open = useCallback(
    (d: StructureDocument) => {
      void refreshLibrary();
      if (scope.onOpen) {
        scope.onOpen(d.id, d.name, true, true);
      } else {
        setDoc(d);
        setSelection({ min: [0, 0, 0], max: d.size.map((n) => n - 1) as Vec3 });
      }
    },
    [scope.onOpen, refreshLibrary],
  );
  const accept = (d: StructureDocument) => {
    setDoc(d);
    setLayer((y) => Math.min(y, d.size[1] - 1));
    setSelection((b) => ({
      min: b.min.map((n, i) => Math.min(n, d.size[i] - 1)) as Vec3,
      max: b.max.map((n, i) => Math.min(n, d.size[i] - 1)) as Vec3,
    }));
    setComparison(undefined);
  };
  const edit = useCallback(
    (e: StructureEdit) =>
      run(async () => {
        const d = latest.current.doc;
        if (!d) return;
        if (e.block) {
          const states = JSON.parse(latest.current.stateText);
          if (!states || Array.isArray(states) || typeof states !== "object")
            throw Error("Block states must be a JSON object");
          e = { ...e, block: { ...e.block, states } };
        }
        const next = await structures.EditStructure(d.id, e);
        accept(next);
        return next;
      }),
    [run],
  );
  const undo = useCallback(
    (redo = false) =>
      run(async () => {
        if (latest.current.doc)
          accept(
            await structures.StructureHistory(latest.current.doc.id, redo),
          );
      }),
    [run],
  );
  const save = useCallback(
    async (saveAs = false) => {
      const d = latest.current.doc;
      if (!d) return false;
      clearTimeout(viewTimer.current);
      const result = await run(async () => {
        if (Object.keys(viewPatch.current).length)
          await structures.SetStructureView(d.id, {
            ...d.view,
            ...viewPatch.current,
          });
        return structures.SaveStructureDialog(
          d.id,
          true,
          saveAs,
          latest.current.options,
        );
      });
      if (result) accept(result);
      return !!result?.path && !result.dirty;
    },
    [run],
  );
  useEffect(() => {
    scope.register?.(
      doc
        ? {
            name: doc.name,
            dirty: doc.dirty,
            ready: true,
            busy,
            path: doc.path,
            save,
            dispose: () => {
              void structures.CloseStructure(doc.id);
            },
          }
        : busy
          ? {
              name: "Structure import",
              dirty: false,
              ready: false,
              busy: true,
              save: async () => false,
            }
          : undefined,
    );
    onStatus({ ready: !!doc, busy, undo: !!doc?.undo, redo: !!doc?.redo });
    return () => scope.register?.(undefined);
  }, [doc, busy, save, scope.register, onStatus]);
  useEffect(() => {
    if (!busy) return;
    const timer = window.setInterval(
      () =>
        void structures
          .GetStructureJob(operationRef.current)
          .catch(() =>
            doc ? structures.GetStructureJob(doc.id) : Promise.reject(),
          )
          .then(setJob)
          .catch(() => {}),
      350,
    );
    return () => {
      clearInterval(timer);
      setJob(undefined);
    };
  }, [doc?.id, busy]);
  const boxes = useCallback(
    () => [...latest.current.regions, normalized(latest.current.selection)],
    [],
  );
  const perform = useCallback(
    (kind: string) => {
      void edit({
        kind,
        bounds: boxes(),
        ...(kind === "fill" ? { block: latest.current.brush } : {}),
        destination: normalized(latest.current.selection).min,
      });
    },
    [edit, boxes],
  );
  const chooseOpen = useCallback(
    () =>
      run(async () => {
        const path = await structures.ChooseStructureSource("structure");
        if (path)
          open(await structures.OpenStructureFile(path, operationRef.current));
      }),
    [run, open],
  );
  useWorkspaceActions("structures", (action) => {
    if (action === "new") setDialog("new");
    else if (action === "import-files") void chooseOpen();
    else if (action === "import-folder") {
      setDialog("import");
    } else if (action === "export" && doc) {
      setPanel("export");
    } else if (action === "refresh") void refreshLibrary();
  });
  useEditorCommands((command) => {
    if (command === "undo") void undo();
    else if (command === "redo") void undo(true);
    else if (command === "open-project") void chooseOpen();
    else if (command === "save-project") void save();
    else if (command === "save-project-as") void save(true);
    else if (command === "top") setTop(true);
    else if (command === "isometric") setTop(false);
  });
  useEffect(() => {
    if (!scope.active) return;
    const handler = (e: KeyboardEvent) => {
      if (
        e.defaultPrevented ||
        busy ||
        !doc ||
        (e.target as HTMLElement)?.closest?.(
          "input,textarea,[contenteditable=true]",
        ) ||
        document.querySelector('[role="dialog"][data-open]')
      )
        return;
      const mod = e.ctrlKey || e.metaKey;
      const key = e.key.toLowerCase();
      if (mod && ["c", "x", "v", "z", "y"].includes(key)) {
        e.preventDefault();
        if (key === "z" || key === "y") void undo(key === "y" || e.shiftKey);
        else perform(key === "c" ? "copy" : key === "x" ? "cut" : "paste");
      } else if (key === "delete") {
        e.preventDefault();
        perform("erase");
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [scope.active, busy, doc, perform, undo]);
  const viewTimer = useRef<number | undefined>(undefined);
  const viewPatch = useRef<Record<string, unknown>>({});
  const updateView = useCallback((v: Record<string, unknown>) => {
    viewPatch.current = { ...viewPatch.current, ...v };
    clearTimeout(viewTimer.current);
    viewTimer.current = window.setTimeout(() => {
      const d = latest.current.doc;
      if (d)
        void structures
          .SetStructureView(d.id, { ...d.view, ...viewPatch.current })
          .then((next) => {
            if (next && latest.current.doc?.id === d.id)
              setDoc((current) =>
                current && current.revision > next.revision
                  ? {
                      ...current,
                      view: next.view,
                      dirty: current.dirty || next.dirty,
                    }
                  : next,
              );
          })
          .catch((e) => setError(String(e)));
    }, 700);
  }, []);
  useEffect(() => {
    if (doc) updateView({ layer, layerOnly });
  }, [layer, layerOnly]);
  useEffect(() => () => clearTimeout(viewTimer.current), []);
  useEffect(() => {
    if (!doc || panel !== "materials") return;
    let stopped = false;
    const bounds =
      materialScope === "selection"
        ? boxes()
        : materialScope === "layer"
          ? [
              {
                min: [0, layer, 0] as Vec3,
                max: [doc.size[0] - 1, layer, doc.size[2] - 1] as Vec3,
              },
            ]
          : [];
    void structures
      .GetStructureMaterials(doc.id, bounds)
      .then((m) => {
        if (!stopped) setMaterials(m);
      })
      .catch((e) => setError(String(e)));
    return () => {
      stopped = true;
    };
  }, [doc?.revision, doc?.id, panel, materialScope, selection, regions, layer]);
  useEffect(() => {
    if (!doc || panel !== "layers") return;
    let stopped = false;
    const x = Math.max(0, Math.min(layerX, doc.size[0] - 1)),
      z = Math.max(0, Math.min(layerZ, doc.size[2] - 1));
    void structures
      .GetStructureLayer(
        doc.id,
        layer,
        x,
        z,
        Math.min(64, doc.size[0] - x),
        Math.min(64, doc.size[2] - z),
      )
      .then((data) => {
        if (!stopped) setLayerData(data);
      })
      .catch((e) => setError(String(e)));
    return () => {
      stopped = true;
    };
  }, [doc?.id, doc?.revision, layer, panel, layerX, layerZ]);
  const select = useCallback(
    (inside: Vec3, outside: Vec3) => {
      const p = latest.current,
        d = p.doc;
      if (!d) return;
      const point = tool === "paint" ? outside : inside;
      if (point.some((v, i) => v < 0 || v >= d.size[i])) return;
      if (tool === "eyedropper") {
        void run(async () => {
          const state = await structures.GetStructureBlock(d.id, point);
          setBrush(state);
          setStateText(JSON.stringify(state.states, null, 2));
          setTool("select");
        });
      } else if (tool === "paint" || tool === "erase") {
        void edit({
          kind: tool === "paint" ? "fill" : "erase",
          bounds: [{ min: point, max: point }],
          block: p.brush,
        });
      } else {
        setSelection((b) =>
          corner === 0 ? { ...b, min: point } : { ...b, max: point },
        );
        setCorner((c) => (c === 0 ? 1 : 0));
      }
    },
    [tool, corner, run, edit],
  );
  const materialBoxes = () =>
    materialScope === "selection"
      ? boxes()
      : materialScope === "layer" && doc
        ? [
            {
              min: [0, layer, 0] as Vec3,
              max: [doc.size[0] - 1, layer, doc.size[2] - 1] as Vec3,
            },
          ]
        : [];
  const differences = (comparison?.samples || []).filter(
    (d) =>
      (diffFilter === "all" || d.kind === diffFilter) &&
      (!layerOnly || d.position[1] === layer),
  );
  const loadTarget = async (id: string) => {
    setTargetID(id);
    if (!id) {
      setTarget(undefined);
      return;
    }
    const d = await structures.RecoverStructure(id);
    setTarget(d);
    setPlacementAnchor(normalized(latest.current.selection).min);
    setComparison(undefined);
  };
  const transformStart = (e: StructureEdit) => {
    setTransform(e);
    setAllowUnsupported(false);
    setDialog("transform");
  };
  return (
    <div
      className={`structure-studio${propertiesOpen ? " structure-properties-open" : ""}`}
    >
      <header className="structure-toolbar studio-view-heading">
        <div className="structure-title">
          <Blocks size={17} />
          <strong>Structure Studio</strong>
          {doc && <span>{doc.name}</span>}
        </div>
        <div className="structure-toolbar-actions">
          <Button title="Blueprint library" aria-label="Blueprint library" className="toolbar-icon-action" variant="ghost" size="sm" disabled={busy} onClick={() => { setLibraryQuery(""); setLibraryOpen(true); void refreshLibrary(); }}>
            <FolderOpen size={14} /><span className="toolbar-action-label">Blueprint library</span></Button>
          <Button
            className="structure-properties-toggle"
            variant="ghost"
            size="icon-sm"
            aria-label="Toggle structure properties"
            aria-pressed={propertiesOpen}
            title="Properties"
            onClick={() => setPropertiesOpen((open) => !open)}
          >
            <Settings2 size={15} />
          </Button>
          <Button title="New" aria-label="New" className="toolbar-icon-action"
            variant="ghost"
            size="sm"
            disabled={busy}
            onClick={() => {
              setName("Untitled structure");
              setDialog("new");
            }}
          >
            <Plus size={14} />
            <span className="toolbar-action-label">New</span></Button>
          <Button title="Open…" aria-label="Open…" className="toolbar-icon-action"
            variant="ghost"
            size="sm"
            disabled={busy}
            onClick={() => void chooseOpen()}
          >
            <FolderOpen size={14} />
            <span className="toolbar-action-label">Open…</span></Button>
          <Button title="From world…" aria-label="From world…" className="toolbar-icon-action"
            variant="ghost"
            size="sm"
            disabled={busy}
            onClick={() => setDialog("import")}
          >
            <Download size={14} />
            <span className="toolbar-action-label">From world…</span></Button>
          {doc && (
            <Button title="Save project" aria-label="Save project" className="toolbar-icon-action" size="sm" disabled={busy} onClick={() => void save()}>
              <Save size={14} />
              <span className="toolbar-action-label">Save project</span></Button>
          )}
        </div>
      </header>
      {error && (
        <FeedbackFlyout title="Structure Studio">
          <AlertTriangle size={15} />
          <span>{error}</span>
          <button onClick={() => setError("")} aria-label="Dismiss error">
            <X size={14} />
          </button>
        </FeedbackFlyout>
      )}
      {message && (
        <div className="structure-notice" role="status">
          {message}
        </div>
      )}
      {!doc ? (
        <StructureLibrary
          items={recovery}
          busy={busy}
          onNew={() => setDialog("new")}
          onFile={() => void chooseOpen()}
          onWorld={() => setDialog("import")}
          onRefresh={() => void refreshLibrary()}
          onOpen={(id) =>
            run(async () => open(await structures.RecoverStructure(id)))
          }
        />
      ) : (
        <>
          <div className="structure-workarea">
            <aside
              className="studio-sidebar structure-sidebar"
              aria-label="Block palette and regions"
            >
              <div className="structure-sidebar-heading">Tools</div>
              <div className="structure-tools">
                {[
                  ["select", MousePointer2, "Select two corners"],
                  ["paint", Paintbrush, "Set one block"],
                  ["erase", Trash2, "Remove one block"],
                  ["eyedropper", Pipette, "Pick block and states"],
                ].map(([id, Icon, label]) => {
                  const I = Icon as typeof MousePointer2;
                  return (
                    <button
                      key={String(id)}
                      aria-label={String(label)}
                      title={String(label)}
                      className={tool === id ? "selected" : ""}
                      onClick={() => setTool(String(id))}
                    >
                      <I size={17} />
                    </button>
                  );
                })}
              </div>
              <div className="structure-sidebar-heading">
                Brush & block palette
              </div>
              <label className="structure-search">
                <Search size={14} />
                <input
                  placeholder="Find block…"
                  value={paletteQuery}
                  onChange={(e) => setPaletteQuery(e.target.value)}
                />
              </label>
              <div className="structure-palette">
                {[
                  ...doc.palette,
                  ...catalog.filter(
                    (s) => !doc.palette.some((p) => p.name === s.name),
                  ),
                ]
                  .filter((s) =>
                    s.name.toLowerCase().includes(paletteQuery.toLowerCase()),
                  )
                  .slice(0, 200)
                  .map((s, i) => (
                    <button
                      key={i}
                      className={brush === s ? "selected" : ""}
                      onClick={() => {
                        setBrush(s);
                        setStateText(JSON.stringify(s.states, null, 2));
                      }}
                      title={JSON.stringify(s.states)}
                    >
                      <i style={{ background: blockColor(s.name) }} />
                      <span>
                        {s.name.replace("minecraft:", "").replaceAll("_", " ")}
                      </span>
                    </button>
                  ))}
              </div>
              <label className="structure-field">
                Block identifier
                <input
                  value={brush.name}
                  onChange={(e) =>
                    setBrush((s) => ({ ...s, name: e.target.value }))
                  }
                />
              </label>
              <label className="structure-field">
                Block states (JSON)
                <textarea
                  rows={5}
                  value={stateText}
                  onChange={(e) => setStateText(e.target.value)}
                  onBlur={() => {
                    try {
                      const states = JSON.parse(stateText);
                      if (
                        !states ||
                        Array.isArray(states) ||
                        typeof states !== "object"
                      )
                        throw Error("Use a JSON object for states");
                      setBrush((s) => ({ ...s, states }));
                    } catch (e) {
                      setError(String(e));
                    }
                  }}
                />
              </label>
              <details>
                <summary>State NBT types</summary>
                <p>
                  Picked states keep their original types. For a new byte state
                  add it below.
                </p>
                <input
                  aria-label="Byte state name"
                  placeholder="State name, e.g. upper_block_bit"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      const key = e.currentTarget.value.trim();
                      if (key)
                        setBrush((b) => ({
                          ...b,
                          types: { ...b.types, [key]: "byte" },
                        }));
                      e.currentTarget.value = "";
                    }
                  }}
                />
                {Object.entries(brush.types || {}).map(([k, v]) => (
                  <small key={k}>
                    {k}: {v}
                    <br />
                  </small>
                ))}
              </details>
              <div className="structure-sidebar-heading">Subregions</div>
              <div className="structure-regions">
                {doc.regions.map((r) => (
                  <button
                    key={r.id}
                    onClick={() => {
                      setRegions([]);
                      setSelection(r.bounds);
                    }}
                  >
                    <BoxIcon size={13} />
                    {r.name}
                    <small>{fmt(volume(r.bounds))} blocks</small>
                  </button>
                ))}
              </div>
              <p className="structure-muted">
                {fmt(doc.volume)} cells ·{" "}
                {(doc.memoryBytes / 1048576).toFixed(1)} MiB allocated
                <br />
                {doc.blockEntities} block entities · {doc.entities} entities
              </p>
              <div className="structure-sidebar-heading">Blueprint library</div>
              <div className="structure-regions">
                {recovery
                  .filter((r) => r.id !== doc.id)
                  .map((r) => (
                    <button
                      key={r.id}
                      disabled={busy}
                      onClick={() =>
                        void run(async () =>
                          open(await structures.RecoverStructure(r.id)),
                        )
                      }
                    >
                      <FolderOpen size={13} />
                      {r.name}
                    </button>
                  ))}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => void refreshLibrary()}
              >
                Refresh library
              </Button>
            </aside>
            <main className="structure-center">
              <div data-editor-tab-host />
              <div className="structure-edit-toolbar">
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busy || !doc.undo}
                  aria-label="Undo"
                  title="Undo · Ctrl+Z"
                  onClick={() => void undo()}
                >
                  <Undo2 size={15} />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busy || !doc.redo}
                  aria-label="Redo"
                  title="Redo · Ctrl+Y"
                  onClick={() => void undo(true)}
                >
                  <Redo2 size={15} />
                </Button>
                <span className="structure-separator" />
                {[
                  ["copy", Copy, "Copy"],
                  ["cut", Scissors, "Cut"],
                  ["paste", ClipboardPaste, "Paste"],
                  ["fill", Paintbrush, "Fill"],
                  ["erase", Trash2, "Delete"],
                ].map(([id, Icon, label]) => {
                  const I = Icon as typeof Copy;
                  return (
                    <Button
                      key={String(id)}
                      variant="ghost"
                      size="sm"
                      disabled={busy}
                      onClick={() => perform(String(id))}
                      title={String(label)}
                    >
                      <I size={14} />
                      <span>{String(label)}</span>
                    </Button>
                  );
                })}
                <span className="structure-separator" />
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busy}
                  onClick={() => transformStart({ kind: "rotate", turns: 1 })}
                >
                  <RotateCw size={14} />
                  90°
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={busy}
                  onClick={() =>
                    transformStart({ kind: "mirror", mirror: "x" })
                  }
                >
                  <FlipHorizontal size={14} />
                  Mirror
                </Button>
              </div>
              <StructureViewport
                doc={doc}
                selection={selection}
                corner={corner}
                layer={layer}
                layerOnly={layerOnly}
                adjacent={adjacent}
                top={top}
                ghost={
                  ghost && target
                    ? { doc: target, anchor: placementAnchor }
                    : undefined
                }
                differences={differences}
                onSelect={select}
                onResize={(c, p) =>
                  setSelection((b) =>
                    c === 0 ? { ...b, min: p } : { ...b, max: p },
                  )
                }
                onView={updateView}
                onGhostAnchor={setPlacementAnchor}
              />
              <div className="structure-layerbar">
                <Layers size={15} />
                <button
                  title="Previous layer"
                  disabled={layer <= 0}
                  onClick={() => setLayer((n) => n - 1)}
                >
                  <ChevronLeft size={16} />
                </button>
                <span>
                  Y {layer}
                  <small>world {doc.origin[1] + layer}</small>
                </span>
                <input
                  aria-label="Layer"
                  type="range"
                  min="0"
                  max={doc.size[1] - 1}
                  value={layer}
                  onChange={(e) => setLayer(Number(e.target.value))}
                />
                <button
                  title="Next layer"
                  disabled={layer >= doc.size[1] - 1}
                  onClick={() => setLayer((n) => n + 1)}
                >
                  <ChevronRight size={16} />
                </button>
                <label>
                  <input
                    type="checkbox"
                    checked={layerOnly}
                    onChange={(e) => setLayerOnly(e.target.checked)}
                  />
                  Isolate
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={adjacent}
                    onChange={(e) => setAdjacent(e.target.checked)}
                  />
                  Adjacent
                </label>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setTop((b) => !b)}
                >
                  {top ? "3D view" : "Top view"}
                </Button>
              </div>
            </main>
            <aside
              className="structure-inspector"
              aria-label="Structure properties"
            >
              <header className="structure-section-heading">
                <strong>Properties</strong>
                <span>{doc.size.join(" \u00d7 ")}</span>
              </header>
              <div className="structure-panel-tabs" role="tablist">
                {["selection", "layers", "materials", "compare", "export"].map(
                  (t) => (
                    <button
                      key={t}
                      role="tab"
                      aria-selected={panel === t}
                      onClick={() => setPanel(t)}
                    >
                      {t[0].toUpperCase() + t.slice(1)}
                    </button>
                  ),
                )}
              </div>
              <div className="structure-panel-body">
                {panel === "selection" && (
                  <>
                    <h3>Block selection</h3>
                    <div className="structure-toggle">
                      <button
                        className={corner === 0 ? "selected" : ""}
                        onClick={() => setCorner(0)}
                      >
                        Corner 1
                      </button>
                      <button
                        className={corner === 1 ? "selected" : ""}
                        onClick={() => setCorner(1)}
                      >
                        Corner 2
                      </button>
                    </div>
                    <VectorField
                      label="Corner 1"
                      value={selection.min}
                      min={0}
                      max={doc.size.map((n) => n - 1) as Vec3}
                      onChange={(min) => setSelection((b) => ({ ...b, min }))}
                    />
                    <VectorField
                      label="Corner 2"
                      value={selection.max}
                      min={0}
                      max={doc.size.map((n) => n - 1) as Vec3}
                      onChange={(max) => setSelection((b) => ({ ...b, max }))}
                    />
                    <p className="structure-muted">
                      {fmt(volume(selection))} selected cells. Drag the colored
                      axes in 3D to resize the active corner.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setRegions((a) => [...a, normalized(selection)]);
                        setSelection(initialBox);
                      }}
                    >
                      Add another region ({regions.length + 1})
                    </Button>
                    {regions.length > 0 && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setRegions([])}
                      >
                        Clear extra selections
                      </Button>
                    )}
                    <label className="structure-field">
                      Subregion name
                      <input
                        value={regionName}
                        onChange={(e) => setRegionName(e.target.value)}
                      />
                    </label>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={busy}
                      onClick={() =>
                        void edit({
                          kind: "regions",
                          regions: [
                            ...doc.regions,
                            {
                              id: crypto.randomUUID(),
                              name: regionName,
                              bounds: normalized(selection),
                            },
                          ],
                        })
                      }
                    >
                      Save named region
                    </Button>
                    <h3>Replace blocks</h3>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={busy}
                      onClick={() =>
                        void run(async () =>
                          open(
                            await structures.ExtractStructure(
                              doc.id,
                              doc.name.slice(0, 65) + " selection",
                              boxes(),
                            ),
                          ),
                        )
                      }
                    >
                      Extract selection to new document
                    </Button>
                    <label className="structure-field">
                      Find block identifier
                      <input
                        value={replaceFrom}
                        onChange={(e) => setReplaceFrom(e.target.value)}
                      />
                    </label>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={busy}
                      onClick={() =>
                        void edit({
                          kind: "replace",
                          from: replaceFrom,
                          block: brush,
                          bounds: boxes(),
                        })
                      }
                    >
                      Replace with brush
                    </Button>
                    <h3>Placement anchor</h3>
                    <VectorField
                      label="Anchor"
                      value={doc.anchor}
                      min={0}
                      max={doc.size.map((n) => n - 1) as Vec3}
                      onChange={(anchor) =>
                        void edit({ kind: "anchor", anchor })
                      }
                    />
                    <h3>Document</h3>
                    <label className="structure-field">
                      Name
                      <input
                        defaultValue={doc.name}
                        key={doc.name}
                        onBlur={(e) => {
                          if (e.target.value !== doc.name)
                            void edit({ kind: "rename", name: e.target.value });
                        }}
                      />
                    </label>
                    <p className="structure-muted">
                      {doc.size.join(" × ")} blocks · origin{" "}
                      {doc.origin.join(", ")}
                      <br />
                      Each edit can be undone. Recovery is saved automatically.
                    </p>
                  </>
                )}
                {panel === "layers" && (
                  <>
                    <h3>Build layer Y {layer}</h3>
                    <p className="structure-muted">
                      Click a cell to select it. Colors identify blocks; hatched
                      cells are unrecorded. Empty outlined cells are structure
                      void.
                    </p>
                    <div className="structure-two">
                      <label className="structure-field">
                        Start X
                        <input
                          type="number"
                          min="0"
                          max={doc.size[0] - 1}
                          value={layerX}
                          onChange={(e) => setLayerX(Number(e.target.value))}
                        />
                      </label>
                      <label className="structure-field">
                        Start Z
                        <input
                          type="number"
                          min="0"
                          max={doc.size[2] - 1}
                          value={layerZ}
                          onChange={(e) => setLayerZ(Number(e.target.value))}
                        />
                      </label>
                    </div>
                    {layerData && (
                      <>
                        <div className="structure-layer-coordinates">
                          X {layerData.x}–{layerData.x + layerData.width - 1} ·
                          Z {layerData.z}–{layerData.z + layerData.depth - 1}
                        </div>
                        <div className="structure-layer-grid-scroll">
                          <div
                            className="structure-layer-grid"
                            style={{
                              gridTemplateColumns: `repeat(${layerData.width},18px)`,
                            }}
                          >
                            {layerData.cells.map((c, index) => {
                              const x = (index % layerData.width) + layerData.x,
                                z =
                                  Math.floor(index / layerData.width) +
                                  layerData.z,
                                s =
                                  c.layers[0] >= 0
                                    ? doc.palette[c.layers[0]]
                                    : undefined;
                              return (
                                <button
                                  key={index}
                                  title={`${x}, ${layer}, ${z}: ${!c.known ? "Unknown" : s?.name || "Structure void"}`}
                                  className={
                                    !c.known
                                      ? "unknown"
                                      : !s
                                        ? "void"
                                        : s.name === "minecraft:air"
                                          ? "air"
                                          : ""
                                  }
                                  style={{
                                    background:
                                      c.known && s && s.name !== "minecraft:air"
                                        ? blockColor(s.name)
                                        : undefined,
                                  }}
                                  onClick={() =>
                                    select([x, layer, z], [x, layer, z])
                                  }
                                />
                              );
                            })}
                          </div>
                        </div>
                        <div className="structure-layer-legend">
                          {[
                            ...new Set(
                              layerData.cells.map((c) =>
                                c.known ? c.layers[0] : -2,
                              ),
                            ),
                          ]
                            .filter(
                              (n) =>
                                n >= 0 &&
                                doc.palette[n]?.name !== "minecraft:air",
                            )
                            .map((n) => (
                              <div key={n}>
                                <i
                                  style={{
                                    background: blockColor(doc.palette[n].name),
                                  }}
                                />
                                {doc.palette[n].name.replace("minecraft:", "")}
                              </div>
                            ))}
                        </div>
                      </>
                    )}
                  </>
                )}
                {panel === "materials" && (
                  <div className="structure-materials-print">
                    <h3>{doc.name} · Materials</h3>
                    <label className="structure-field">
                      Count scope
                      <select
                        value={materialScope}
                        onChange={(e) => setMaterialScope(e.target.value)}
                      >
                        <option value="all">Entire structure</option>
                        <option value="selection">
                          Selection & extra regions
                        </option>
                        <option value="layer">Current layer</option>
                      </select>
                    </label>
                    <p className="structure-muted">
                      Blocks and item estimates are counted separately. Stacks
                      use 64 items; fluids and custom blocks are estimates.
                    </p>
                    <div className="structure-two">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          void run(() =>
                            structures.ExportStructureMaterials(
                              doc.id,
                              materialBoxes(),
                            ),
                          )
                        }
                      >
                        <FileSpreadsheet size={14} />
                        CSV
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => window.print()}
                      >
                        <Printer size={14} />
                        Print
                      </Button>
                    </div>
                    {materials && (
                      <>
                        <p>
                          {fmt(materials.blocks)} blocks ·{" "}
                          {fmt(materials.unknown)} unknown ·{" "}
                          {fmt(materials.void)} void
                        </p>
                        <table>
                          <thead>
                            <tr>
                              <th>Block</th>
                              <th>Blocks</th>
                              <th>Items</th>
                              <th>64 + rest</th>
                            </tr>
                          </thead>
                          <tbody>
                            {materials.rows.map((r) => (
                              <tr key={r.name}>
                                <td>
                                  {r.name
                                    .replace("minecraft:", "")
                                    .replaceAll("_", " ")}
                                  {r.estimated ? " *" : ""}
                                </td>
                                <td>{fmt(r.blocks)}</td>
                                <td>{fmt(r.items)}</td>
                                <td>
                                  {r.stacks} + {r.remainder}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </>
                    )}
                  </div>
                )}
                {panel === "compare" && (
                  <>
                    <h3>Compare & assemble</h3>
                    <label className="structure-field">
                      Other blueprint
                      <select
                        value={targetID}
                        onChange={(e) =>
                          void run(() => loadTarget(e.target.value))
                        }
                      >
                        <option value="">Choose a blueprint…</option>
                        {recovery
                          .filter((r) => r.id !== doc.id)
                          .map((r) => (
                            <option key={r.id} value={r.id}>
                              {r.name}
                            </option>
                          ))}
                      </select>
                    </label>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => void refreshLibrary()}
                    >
                      Refresh blueprints
                    </Button>
                    <VectorField
                      label="Comparison offset"
                      value={compareOffset}
                      onChange={setCompareOffset}
                    />
                    <Button
                      disabled={busy || !target}
                      onClick={() =>
                        void run(async () => {
                          const c = await structures.CompareStructures(
                            doc.id,
                            target!.id,
                            compareOffset,
                          );
                          setComparison(c);
                        })
                      }
                    >
                      Compare with current blueprint
                    </Button>
                    <p className="structure-muted">
                      Expected: current blueprint. Actual: chosen blueprint at
                      the offset above. Unknown capture areas remain unknown.
                    </p>
                    {comparison && (
                      <>
                        <div className="structure-compare-counts">
                          {[
                            "correct",
                            "missing",
                            "extra",
                            "wrong",
                            "orientation",
                            "unknown",
                          ].map((k) => (
                            <div key={k}>
                              <strong>
                                {fmt(
                                  comparison[k as keyof Comparison] as number,
                                )}
                              </strong>
                              <span>
                                {k === "orientation" ? "wrong state" : k}
                              </span>
                            </div>
                          ))}
                        </div>
                        <label className="structure-field">
                          Show differences
                          <select
                            value={diffFilter}
                            onChange={(e) => setDiffFilter(e.target.value)}
                          >
                            <option value="all">All differences</option>
                            {[
                              "missing",
                              "extra",
                              "wrong",
                              "state",
                              "unknown",
                            ].map((k) => (
                              <option key={k}>{k}</option>
                            ))}
                          </select>
                        </label>
                        <p className="structure-muted">
                          {comparison.truncated
                            ? "First 10,000 results; 3D shows up to 3,000 markers."
                            : `${differences.length} results. Click to inspect a layer.`}
                        </p>
                        <div className="structure-difference-list">
                          {differences.slice(0, 150).map((d, i) => (
                            <button
                              key={i}
                              onClick={() => {
                                setSelection({
                                  min: d.position,
                                  max: d.position,
                                });
                                setLayer(d.position[1]);
                                setLayerOnly(true);
                              }}
                            >
                              <strong>{d.kind}</strong>
                              {d.position.join(", ")}
                              <small>
                                {d.expected} → {d.actual}
                              </small>
                            </button>
                          ))}
                        </div>
                      </>
                    )}
                    <h3>Place chosen blueprint here</h3>
                    <VectorField
                      label="Ghost anchor"
                      value={placementAnchor}
                      onChange={setPlacementAnchor}
                    />
                    <label className="structure-field">
                      Placement mode
                      <select
                        value={placementMode}
                        onChange={(e) => setPlacementMode(e.target.value)}
                      >
                        <option value="ignore-air">Ignore source air</option>
                        <option value="replace">Replace all</option>
                        <option value="air-only">Only into air</option>
                      </select>
                    </label>
                    <label>
                      <input
                        type="checkbox"
                        checked={ghost}
                        disabled={!target}
                        onChange={(e) => setGhost(e.target.checked)}
                      />
                      Show movable ghost preview
                    </label>
                    <Button
                      disabled={!target || busy}
                      onClick={() =>
                        void run(async () =>
                          accept(
                            await structures.PlaceStructureDocument(
                              target!.id,
                              doc.id,
                              placementAnchor,
                              placementMode,
                              options.entities,
                            ),
                          ),
                        )
                      }
                    >
                      <Move3D size={15} />
                      Place into this document
                    </Button>
                    <p className="structure-muted">
                      Placement can be undone. This changes the blueprint
                      document only.
                    </p>
                  </>
                )}
                {panel === "export" && (
                  <>
                    <h3>Bedrock export</h3>
                    <p className="structure-muted">
                      Portable projects retain editing data, view, anchors and
                      dependencies. Bedrock structures are exported separately.
                    </p>
                    {(
                      [
                        ["entities", "Include entities (never players)"],
                        ["inventories", "Include inventories"],
                        ["blockData", "Include signs & block entity data"],
                        ["packs", "Embed dependency packs"],
                        ["omitUnknown", "Omit unknown areas as structure void"],
                      ] as const
                    ).map(([key, label]) => (
                      <label className="structure-check" key={key}>
                        <input
                          type="checkbox"
                          checked={options[key]}
                          onChange={(e) =>
                            setOptions((o) => ({
                              ...o,
                              [key]: e.target.checked,
                            }))
                          }
                        />
                        {label}
                      </label>
                    ))}
                    <Button disabled={busy} onClick={() => void save(true)}>
                      <Save size={14} />
                      Save portable .craftbox…
                    </Button>
                    <Button
                      variant="outline"
                      disabled={busy}
                      onClick={() =>
                        void run(async () =>
                          accept(
                            await structures.SaveStructureDialog(
                              doc.id,
                              false,
                              true,
                              options,
                            ),
                          ),
                        )
                      }
                    >
                      <Download size={14} />
                      Export .mcstructure…
                    </Button>
                    <h3>Place in a world</h3>
                    <p className="structure-muted">
                      Export a new .mcworld copy of a local world, or create an
                      empty world. The original world is preserved.
                    </p>
                    <Button
                      variant="outline"
                      disabled={busy}
                      onClick={() => {
                        setPlacementAnchor([0, 64, 0]);
                        setDimension(0);
                        setDialog("place");
                      }}
                    >
                      <Move3D size={14} />
                      Export new world copy…
                    </Button>
                    <h3>Resource dependencies</h3>
                    <div className="structure-two">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={busy}
                        onClick={() =>
                          void run(async () => {
                            const path =
                              await structures.ChooseStructureSource("pack");
                            if (path)
                              accept(
                                await structures.AttachStructurePack(
                                  doc.id,
                                  path,
                                  "",
                                ),
                              );
                          })
                        }
                      >
                        Add .mcpack…
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={busy}
                        onClick={() =>
                          void run(async () => {
                            const path =
                              await structures.ChooseStructureSource(
                                "pack-folder",
                              );
                            if (path)
                              accept(
                                await structures.AttachStructurePack(
                                  doc.id,
                                  path,
                                  "",
                                ),
                              );
                          })
                        }
                      >
                        Add folder…
                      </Button>
                    </div>
                    {doc.packs.length === 0 ? (
                      <p className="structure-muted">
                        No embedded resource packs. Imported custom blocks may
                        need their original packs.
                      </p>
                    ) : (
                      doc.packs.map((p) => (
                        <div
                          className={`structure-dependency ${p.missing ? "missing" : ""}`}
                          key={p.id}
                        >
                          <Blocks size={15} />
                          <div>
                            <strong>{p.name || p.id}</strong>
                            <small>
                              {p.kind} ·{" "}
                              {p.missing ? "Missing dependency" : "Available"}
                            </small>
                            <code>{p.uuid}</code>
                            {p.missing && (
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={busy}
                                onClick={() =>
                                  void run(async () => {
                                    const path =
                                      await structures.ChooseStructureSource(
                                        "pack",
                                      );
                                    if (path)
                                      accept(
                                        await structures.AttachStructurePack(
                                          doc.id,
                                          path,
                                          p.id,
                                        ),
                                      );
                                  })
                                }
                              >
                                Locate original pack…
                              </Button>
                            )}
                          </div>
                        </div>
                      ))
                    )}
                    <p className="structure-muted">
                      Missing custom blocks retain their identifier and states.
                      Use Replace blocks to explicitly substitute them.
                    </p>
                  </>
                )}
              </div>
            </aside>
          </div>
          <footer className="structure-status">
            <span>
              {busy
                ? job?.stage || "Working…"
                : doc.dirty
                  ? "Unsaved changes · recovery saved"
                  : "Project saved"}
            </span>
            <span>
              {doc.size.join(" × ")} · {fmt(volume(selection))} selected
            </span>
            {busy && (
              <>
                <progress
                  max={job?.total || 1}
                  value={job?.total ? job.done : undefined}
                />
                <button
                  onClick={() => void structures.CancelStructureJob(doc.id)}
                >
                  Cancel
                </button>
              </>
            )}
          </footer>
        </>
      )}
      {documentLoading && <EditorLoadingOverlay label="Loading structure" />}
      <Dialog open={libraryOpen} onOpenChange={setLibraryOpen}>
        <DialogContent className="studio-window studio-dialog structure-dialog">
          <DialogHeader><DialogTitle>Blueprint library</DialogTitle></DialogHeader>
          <div className="studio-dialog-body structure-dialog-body">
            <DialogDescription>Open a saved blueprint as another project. Ingame snapshots appear here automatically.</DialogDescription>
            <input type="search" aria-label="Find saved blueprint" placeholder="Search blueprints..." value={libraryQuery} onChange={e => setLibraryQuery(e.target.value)} />
            <div className="structure-regions structure-project-picker">
              {recovery.filter(r => r.name.toLocaleLowerCase().includes(libraryQuery.trim().toLocaleLowerCase())).sort((a,b) => b.modified-a.modified).map(r => (
                <button key={r.id} disabled={busy} onClick={() => void run(async () => {
                  const saved = await structures.GetStructure(r.id).catch(() => structures.RecoverStructure(r.id));
                  setLibraryOpen(false); open(saved);
                })}><FolderOpen size={16} /><span>{r.name}</span>{r.id === doc?.id && <small>Current project</small>}</button>
              ))}
              {!recovery.length && <p className="structure-muted">No saved blueprints yet. Use /cb structure save in Minecraft.</p>}
            </div>
          </div>
          <DialogFooter><Button variant="ghost" disabled={busy} onClick={() => void refreshLibrary()}>Refresh library</Button><Button variant="outline" onClick={() => setLibraryOpen(false)}>Close</Button></DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={!!dialog}
        onOpenChange={(v) => {
          if (!v && !busy) setDialog("");
        }}
      >
        <DialogContent className="structure-dialog">
          <DialogHeader>
            <DialogTitle>
              {dialog === "new"
                ? "New structure"
                : dialog === "import"
                  ? "Extract a Bedrock region"
                  : dialog === "transform"
                    ? "Transform structure"
                    : "Place in a new world copy"}
            </DialogTitle>
          </DialogHeader>
          <div
            className="studio-dialog-body structure-dialog-body"
            role="region"
            aria-label="Structure settings"
          >
            <DialogDescription>
              {dialog === "import"
                ? "Coordinates are inclusive and refer to the source world. Unrecorded cells remain unknown."
                : dialog === "new"
                  ? "Start with an empty, editable volume. Projects are stored independently from Minecraft worlds."
                  : dialog === "transform"
                    ? "Rotate or mirror the complete structure, including block directions and the placement anchor."
                    : "The result is written to a new .mcworld file. Existing source files are preserved."}
            </DialogDescription>

            {dialog === "new" && (
              <>
                <label className="structure-field">
                  Name
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </label>
                <VectorField
                  label="Size"
                  value={size}
                  min={1}
                  onChange={setSize}
                />
                <p className="structure-muted">
                  {fmt(size.reduce((a, b) => a * b, 1))} cells · up to{" "}
                  {((size.reduce((a, b) => a * b, 1) * 8.2) / 1048576).toFixed(
                    1,
                  )}{" "}
                  MiB for dense block data. Large builds are processed by
                  section. Maximum 32 million cells per document.
                </p>
              </>
            )}
            {dialog === "import" && (
              <>
                <label className="structure-field">
                  Blueprint name
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </label>
                <label className="structure-field">
                  Captured world
                  <select
                    value={importWorld}
                    onChange={(e) => {
                      setImportWorld(e.target.value);
                      setImportPath("");
                    }}
                  >
                    <option value="">Choose local file or folder below…</option>
                    {worlds.map((w) => (
                      <option value={w.id} key={w.id}>
                        {w.name}
                      </option>
                    ))}
                  </select>
                </label>
                <div className="structure-two">
                  <Button
                    variant="outline"
                    disabled={busy}
                    onClick={() =>
                      void run(async () => {
                        const path =
                          await structures.ChooseStructureSource("world");
                        if (path) {
                          setImportPath(path);
                          setImportWorld("");
                        }
                      })
                    }
                  >
                    Choose .mcworld…
                  </Button>
                  <Button
                    variant="outline"
                    disabled={busy}
                    onClick={() =>
                      void run(async () => {
                        const path =
                          await structures.ChooseStructureSource("folder");
                        if (path) {
                          setImportPath(path);
                          setImportWorld("");
                        }
                      })
                    }
                  >
                    Choose folder…
                  </Button>
                </div>
                {importPath && <p className="structure-path">{importPath}</p>}
                <label className="structure-field">
                  Dimension
                  <select
                    value={dimension}
                    onChange={(e) => setDimension(Number(e.target.value))}
                  >
                    <option value={0}>Overworld</option>
                    <option value={1}>Nether</option>
                    <option value={2}>The End</option>
                  </select>
                </label>
                <VectorField
                  label="World corner 1"
                  value={importBounds.min}
                  onChange={(min) => setImportBounds((b) => ({ ...b, min }))}
                />
                <VectorField
                  label="World corner 2"
                  value={importBounds.max}
                  onChange={(max) => setImportBounds((b) => ({ ...b, max }))}
                />
                <p className="structure-muted">
                  {fmt(volume(importBounds))} cells · estimated dense block
                  storage {((volume(importBounds) * 8.2) / 1048576).toFixed(1)}{" "}
                  MiB
                </p>
              </>
            )}
            {dialog === "transform" && (
              <>
                <label className="structure-field">
                  Rotation
                  <select
                    value={transform.turns || 0}
                    onChange={(e) =>
                      setTransform((t) => ({
                        ...t,
                        turns: Number(e.target.value),
                        kind: "rotate",
                      }))
                    }
                  >
                    <option value="0">0°</option>
                    <option value="1">90° clockwise</option>
                    <option value="2">180°</option>
                    <option value="3">270° clockwise</option>
                  </select>
                </label>
                <label className="structure-field">
                  Mirror
                  <select
                    value={transform.mirror || ""}
                    onChange={(e) =>
                      setTransform((t) => ({
                        ...t,
                        mirror: e.target.value,
                        kind: "rotate",
                      }))
                    }
                  >
                    <option value="">None</option>
                    <option value="x">X axis</option>
                    <option value="z">Z axis</option>
                  </select>
                </label>
                <p className="structure-muted">
                  Unsupported orientation states block the operation and are
                  listed in the error details before any edit is applied.
                </p>
                <label>
                  <input
                    type="checkbox"
                    checked={allowUnsupported}
                    onChange={(e) => setAllowUnsupported(e.target.checked)}
                  />
                  Apply despite unsupported orientation states
                </label>
              </>
            )}
            {dialog === "place" && (
              <>
                <label className="structure-field">
                  Target dimension
                  <select
                    disabled={!placementSource}
                    value={placementSource ? dimension : 0}
                    onChange={(e) => setDimension(Number(e.target.value))}
                  >
                    <option value={0}>Overworld</option>
                    <option value={1}>Nether</option>
                    <option value={2}>The End</option>
                  </select>
                </label>
                <label className="structure-field">
                  Base world
                  <input
                    readOnly
                    value={placementSource || "New empty world"}
                  />
                </label>
                <div className="structure-two">
                  <Button
                    variant="outline"
                    disabled={busy}
                    onClick={() =>
                      void run(async () => {
                        const path =
                          await structures.ChooseStructureSource("world");
                        if (path) setPlacementSource(path);
                      })
                    }
                  >
                    Choose .mcworld…
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setPlacementSource("")}
                  >
                    Use empty world
                  </Button>
                </div>
                <VectorField
                  label="World anchor"
                  value={placementAnchor}
                  onChange={setPlacementAnchor}
                />
                <label className="structure-field">
                  Placement mode
                  <select
                    value={placementMode}
                    onChange={(e) => setPlacementMode(e.target.value)}
                  >
                    <option value="ignore-air">Ignore source air</option>
                    <option value="replace">Replace all</option>
                    <option value="air-only">Only place into air</option>
                  </select>
                </label>
                <p className="structure-muted">
                  Uses export settings for entities and dependency packs.
                  Unknown cells must be resolved before placement.
                </p>
              </>
            )}
          </div>
          <DialogFooter>
            {busy && (
              <span role="status">
                {job?.stage || "Working…"}
                {job?.total ? ` · ${job.done} / ${job.total}` : ""}
              </span>
            )}
            <Button
              variant="outline"
              onClick={() =>
                busy
                  ? void structures.CancelStructureJob(
                      dialog === "import" || !doc
                        ? operationRef.current
                        : doc.id,
                    )
                  : setDialog("")
              }
            >
              Cancel
            </Button>
            <Button
              disabled={
                busy || (dialog === "import" && !importWorld && !importPath)
              }
              onClick={() =>
                void run(async () => {
                  if (dialog === "new") {
                    open(await structures.NewStructure(name, size));
                    setDialog("");
                  } else if (dialog === "import") {
                    open(
                      await structures.ImportStructureWorld({
                        jobId: operationRef.current,
                        name,
                        worldId: importWorld,
                        path: importPath,
                        bounds: importBounds,
                        dimension,
                      }),
                    );
                    setDialog("");
                  } else if (dialog === "transform" && doc) {
                    accept(
                      await structures.EditStructure(doc.id, {
                        ...transform,
                        allowUnsupported,
                      }),
                    );
                    setDialog("");
                  } else if (dialog === "place" && doc) {
                    const path = await structures.ExportStructureWorldDialog({
                      id: doc.id,
                      source: placementSource,
                      anchor: placementAnchor,
                      mode: placementMode,
                      dimension: placementSource ? dimension : 0,
                      entities: options.entities,
                      packs: options.packs,
                      inventories: options.inventories,
                      blockData: options.blockData,
                    });
                    if (path) {
                      setMessage(`World copy exported: ${path}`);
                      setDialog("");
                    }
                  }
                })
              }
            >
              {busy
                ? "Working…"
                : dialog === "new"
                  ? "Create structure"
                  : dialog === "import"
                    ? "Extract region"
                    : dialog === "transform"
                      ? "Apply transform"
                      : "Export new world…"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

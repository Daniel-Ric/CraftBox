import FeedbackFlyout from "./FeedbackFlyout";
import {MediaExportDialog} from "./MediaDialogs";
import {exportTasks} from "./export-tasks";
import {imageExportData,type ImageFormat,type ExportOptions} from "./export-formats";
import {notifyAction,notifyFailure} from "./StudioFlyouts";
import StudioForm from "@/StudioForm";
import StudioInput from "@/StudioInput";
import { rangeOrderError } from "./input-rules";
import { DialogValidation } from "./form-validation";
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Crosshair,
  Minus,
  Plus,
  Scan,
  Layers,
  MapPin,
  Ruler,
  Navigation,
  Download,
  Box,
  Hand,
  SquareDashed,
  X,
  Copy,
  Trash2,
} from "lucide-react";
import { WorldIcon } from "./studio-icons";
import { Button } from "@/components/ui/button";

import { api, type Tile, type ChunkBounds } from "./bridge";
import { fitTiles } from "./map-camera";
import { selectedChunks } from "./map-selection";
import {
  mapCoverage,
  normalBounds,
  terrainBounds,
  pointAt,
  metadataAt,
  biomeColor,
  normalizeMapStyle,
  normalizeSlice,
  type MapStyle,
  type MapView,
} from "./map-workbench";
import { zoomAt } from "./map-workbench";
import { MapImages, paintMap, type MapDraw } from "./map-painter";
import { useMapTiles } from "./use-map-tiles";
import type { MapPresentation, MapMarker } from "./map-presentation";
import "./atlas-map.css";

export type AtlasMapHandle = {
  capture: () => Promise<string>;
  export: () => void;
};
export type AtlasMapProps = {
	 capturePoints?: import("./bridge").CapturePoint[];
  worldId?: string;
  name?: string;
  tiles: Map<string, Tile>;
  player: [number, number, number];
  hasPlayer: boolean;
  selecting?: boolean;
  selection?: ChunkBounds;
  onSelectionChange?: (b: ChunkBounds) => void;
  onSelectionClear?: () => void;
  pickingSpawn?: boolean;
  spawn?: [number, number, number];
  onSpawnPick?: (x: number, z: number) => void;
  onOpen3D?: (point: [number, number, number]) => void;
  onOpenSelection3D?:(bounds:ChunkBounds)=>void;
  onCreateSelectionMap?:(bounds:ChunkBounds)=>void;
  onReady?: () => void;
  presentation?: MapPresentation;
  onPresentationChange?: (s: MapPresentation) => void;
  placingMarker?: boolean;
  onMapPlace?: (x: number, z: number) => void;
  onRequestRegion?: () => void;
};
const EMPTY = new Map<string, Tile>();
const bounded = (n: number) => Number.isFinite(n) && Math.abs(n) <= 30_000_000;
function readSaved(key: string) {
  try {
    const data = JSON.parse(
      localStorage.getItem(`CraftBox-atlas:${key}`) ?? "{}",
    );
    return data && typeof data === "object" && !Array.isArray(data) ? data : {};
  } catch {
    return {};
  }
}

const AtlasMap = forwardRef<AtlasMapHandle, AtlasMapProps>(
  function AtlasMap(p, ref) {
    const host = useRef<HTMLDivElement>(null),
      canvas = useRef<HTMLCanvasElement>(null),
      cache = useRef(new MapImages()),
      cancelExport = useRef(false),
      mounted = useRef(true),
      fitted = useRef(false),
      recent = useRef(new Map<string, number>()),
      imageNotification = useRef(0),
      knownVersions = useRef(new Map<string, string>());
    const tileElevations = useRef(new WeakMap<Tile, [number, number]>());
    const storageKey = p.worldId ?? "current",
      saved = useMemo(() => readSaved(storageKey), [storageKey]);
    const [style, setStyle] = useState<MapStyle>(() =>
        normalizeMapStyle(p.presentation?.atlas ?? saved.style),
      ),
      [y, setY] = useState<number | undefined>(
        normalizeSlice(
          p.presentation?.atlas ? p.presentation.atlas.y : saved.y,
        ),
      ),
      [view, setView] = useState<MapView>({ x: 0, z: 0, scale: 2 }),
      [size, setSize] = useState({ width: 700, height: 420 }),
      [revision, setRevision] = useState(0),
      [follow, setFollow] = useState(false),
      [panel, setPanel] = useState(""),
      [tool, setTool] = useState("pan"),
      [localSelection, setLocalSelection] = useState<ChunkBounds>(),
      [pointer, setPointer] = useState<{ x: number; z: number }>(),
      [markers, setMarkers] = useState<MapMarker[]>(
        Array.isArray(saved.markers) ? saved.markers.slice(0, 512) : [],
      ),
      [measure, setMeasure] = useState<{ x: number; z: number }[]>([]),
      [closed, setClosed] = useState(false),
      [markerLabel, setMarkerLabel] = useState("New marker"),
      [markerColor, setMarkerColor] = useState("#f1cf78"),
      [go, setGo] = useState({ x: "0", z: "0" }),
      [exportOpen, setExportOpen] = useState(false),
      [exportScope, setExportScope] = useState("all"),
      [exportDensity, setExportDensity] = useState(4),
      [transparent, setTransparent] = useState(true),
      [annotations, setAnnotations] = useState(true),
      [exporting, setExporting] = useState(false),
      [exportProgress, setExportProgress] = useState(0);
    const setNotice=(message:string)=>notifyAction(message,"Map");
    const drag = useRef<{ x: number; y: number } | undefined>(undefined),
      selectionStart = useRef<{ x: number; z: number } | undefined>(undefined);
    const activeStyle = {
      ...style,
      ...(p.presentation
        ? {
            rotation: p.presentation.rotation,
            relief: p.presentation.shading,
            contours: p.presentation.contours,
            contourStep: p.presentation.contourStep,
            heightTint: p.presentation.heightTint,
            saturation: p.presentation.saturation,
            grid: p.presentation.grid,
            gridSize: p.presentation.gridSize,
            gridOpacity: p.presentation.gridOpacity,
          }
        : {}),
    };
    const actualView = { ...view, rotation: activeStyle.rotation },
      selection = p.selection ?? localSelection,
      activeMarkers = p.presentation?.markers ?? markers;
    const pixels =
      style.detail === "textures" && view.scale >= 6
        ? 16
        : view.scale >= 0.7
          ? 4
          : 1;
    const angle = (activeStyle.rotation * Math.PI) / 180;
    const rx =
      (Math.abs(size.width * Math.cos(angle)) +
        Math.abs(size.height * Math.sin(angle))) /
      view.scale /
      2;
    const rz =
      (Math.abs(size.width * Math.sin(angle)) +
        Math.abs(size.height * Math.cos(angle))) /
      view.scale /
      2;
    const bounds = useMemo<[number, number, number, number] | undefined>(() => {
      const b: [number, number, number, number] = [
        Math.floor((view.x - rx) / 16),
        Math.floor((view.z - rz) / 16),
        Math.floor((view.x + rx) / 16),
        Math.floor((view.z + rz) / 16),
      ];
      return (b[2] - b[0] + 1) * (b[3] - b[1] + 1) * (pixels * 16) ** 2 <=
        16 * 1024 * 1024
        ? b
        : undefined;
    }, [
      Math.floor((view.x - rx) / 16),
      Math.floor((view.z - rz) / 16),
      Math.floor((view.x + rx) / 16),
      Math.floor((view.z + rz) / 16),
      pixels,
    ]);
    const data = useMapTiles(p.worldId, p.tiles ?? EMPTY, y, bounds, pixels),
      tiles = data.tiles;
    const locations = useMemo(
      () => data.coverage?.map(([x, z]) => ({ x, z })) ?? [...tiles.values()],
      [data.coverage, tiles],
    );
    const elevationRange = useMemo<[number, number]>(() => {
      let min = Infinity,
        max = -Infinity;
      for (const t of tiles.values()) {
        let range = tileElevations.current.get(t);
        if (!range) {
          let lo = Infinity,
            hi = -Infinity;
          for (const h of t.heights ?? []) {
            if (h === -32768) continue;
            lo = Math.min(lo, h);
            hi = Math.max(hi, h);
          }
          range = [lo, hi];
          tileElevations.current.set(t, range);
        }
        min = Math.min(min, range[0]);
        max = Math.max(max, range[1]);
      }
      return min === Infinity ? [-64, 320] : [min, Math.max(min + 1, max)];
    }, [tiles]);
    const coverage = selection ? mapCoverage(locations, selection) : undefined;
    const setSelection = (b: ChunkBounds) => {
      const normalized = normalBounds(b);
      setLocalSelection(normalized);
      p.onSelectionChange?.(normalized);
    };
    const changeMarkers = (next: MapMarker[]) => {
      if (p.presentation && p.onPresentationChange)
        p.onPresentationChange({ ...p.presentation, markers: next });
      else setMarkers(next);
    };
	const importedCapturePoints=useRef(new Set<string>(p.presentation?.captureImports??(Array.isArray(saved.captureImports)?saved.captureImports:[])));
	useEffect(()=>{
	 if(p.presentation&&JSON.stringify(p.presentation.atlas)!==JSON.stringify({...style,y}))return;
	 const fresh=(p.capturePoints??[]).filter(point=>!importedCapturePoints.current.has(point.id));
	 if(!fresh.length)return;
	 for(const point of fresh)importedCapturePoints.current.add(point.id);
	 const known=new Set(activeMarkers.map(marker=>marker.id));
	 const added=fresh.filter(point=>!known.has(point.id)).map(point=>({id:point.id,x:point.position[0],z:point.position[2],label:point.name,icon:point.icon,color:"#f1cf78",background:"#202020",scale:1}));
	 const next=[...activeMarkers,...added].slice(0,512);
	 if(p.presentation&&p.onPresentationChange)p.onPresentationChange({...p.presentation,markers:next,captureImports:[...importedCapturePoints.current]});else if(added.length)setMarkers(next);
	},[p.capturePoints,activeMarkers,p.presentation?.atlas,style,y]);
    const effectiveTool = p.selecting
      ? "select"
      : p.pickingSpawn
        ? "spawn"
        : p.placingMarker
          ? "external-marker"
          : tool;
    const state: MapDraw = {
      tiles,
      view: actualView,
      style: {
        ...activeStyle,
        elevationMin: elevationRange[0],
        elevationMax: elevationRange[1],
      },
      width: size.width,
      height: size.height,
      selection,
      markers: activeMarkers,
      measure: closed && measure.length ? [...measure, measure[0]] : measure,
      player: p.hasPlayer ? p.player : undefined,
      spawn: p.spawn,
      coverage: data.coverage,
      recent: recent.current,
      annotations: true,
      frameBudgetMs: 8,
    };
    const current = useRef(state);
    current.current = state;
    useEffect(() => {
      mounted.current = true;
      return () => {
        mounted.current = false;
        cancelExport.current = true;
        cache.current.dispose();
        cancelAnimationFrame(imageNotification.current);
      };
    }, []);
    useEffect(() => {
      try {
        localStorage.setItem(
          `CraftBox-atlas:${storageKey}`,
          JSON.stringify({ style, y, markers,captureImports:[...importedCapturePoints.current] }),
        );
      } catch {}
    }, [storageKey, style, y, markers,p.capturePoints]);
    useEffect(() => {
      if (
        p.presentation &&
        p.onPresentationChange &&
        JSON.stringify(p.presentation.atlas) !== JSON.stringify({ ...style, y })
      )
        p.onPresentationChange({ ...p.presentation, atlas: { ...style, y } });
    }, [style, y]);
    const lastZoom = useRef(1);
    useEffect(() => {
      const zoom = p.presentation?.zoom ?? 1;
      if (zoom !== lastZoom.current) {
        const factor = zoom / lastZoom.current;
        setView((v) => ({
          ...v,
          scale: Math.max(0.001, Math.min(64, v.scale * factor)),
        }));
        lastZoom.current = zoom;
      }
    }, [p.presentation?.zoom]);
    useEffect(() => {
      if (!host.current) return;
      const observer = new ResizeObserver((entries) => {
        const b = entries[0]?.contentRect;
        if (b)
          setSize({
            width: Math.max(1, b.width),
            height: Math.max(1, b.height),
          });
      });
      observer.observe(host.current);
      return () => observer.disconnect();
    }, []);
    useEffect(() => {
      cache.current.sync(tiles, () => {
        if (mounted.current && !imageNotification.current)
          imageNotification.current = requestAnimationFrame(() => {
            imageNotification.current = 0;
            if (mounted.current) setRevision((v) => v + 1);
          });
      });
      const now = Date.now();
      for (const [k, t] of tiles) {
        const key = String(t.version ?? t.image);
        if (
          knownVersions.current.has(k) &&
          knownVersions.current.get(k) !== key
        )
          recent.current.set(k, now);
        knownVersions.current.set(k, key);
      }
      for (const [k, time] of recent.current)
        if (now - time > 2500) recent.current.delete(k);
    }, [tiles]);
    useEffect(() => {
      if (!fitted.current && tiles.size) {
        fitted.current = true;
        const fittedView = fitTiles(locations, size.width, size.height);
        setView({
          ...fittedView,
          scale: Math.max(
            0.001,
            Math.min(64, fittedView.scale * (p.presentation?.zoom ?? 1)),
          ),
        });
        p.onReady?.();
      }
    }, [tiles, locations, size.width, size.height]);
    useEffect(() => {
      if (follow && p.hasPlayer)
        setView((v) => ({ ...v, x: p.player[0], z: p.player[2] }));
    }, [follow, p.player[0], p.player[2], p.hasPlayer]);
    useEffect(() => {
      const element = canvas.current;
      if (!element) return;
      let frame = 0;
      let animationTimer: ReturnType<typeof setTimeout> | undefined;
      const draw = () => {
        const ctx = element.getContext("2d");
        if (!ctx) return;
        const dpr = Math.min(window.devicePixelRatio || 1, 2);
        if (
          element.width !== Math.round(size.width * dpr) ||
          element.height !== Math.round(size.height * dpr)
        ) {
          element.width = Math.round(size.width * dpr);
          element.height = Math.round(size.height * dpr);
        }
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        const pending = paintMap(ctx, cache.current, current.current);
        if (pending) frame = requestAnimationFrame(draw);
        else if (recent.current.size && current.current.style.recent)
          animationTimer = setTimeout(() => {
            frame = requestAnimationFrame(draw);
          }, 100);
      };
      frame = requestAnimationFrame(draw);
      return () => {
        cancelAnimationFrame(frame);
        clearTimeout(animationTimer);
      };
    }, [
      tiles,
      view,
      size,
      revision,
      style,
      p.presentation,
      selection,
      activeMarkers,
      measure,
      closed,
      p.player[0],
      p.player[1],
      p.player[2],
      p.hasPlayer,
      p.spawn?.[0],
      p.spawn?.[1],
      p.spawn?.[2],
    ]);
    const zoom = (factor: number, x = size.width / 2, z = size.height / 2) => {
      setFollow(false);
      setView((v) =>
        zoomAt(
          { ...v, rotation: activeStyle.rotation },
          factor,
          x,
          z,
          size.width,
          size.height,
        ),
      );
    };
    function at(e: React.PointerEvent | React.MouseEvent) {
      const b = canvas.current!.getBoundingClientRect();
      return pointAt(
        e.clientX - b.left,
        e.clientY - b.top,
        b.width,
        b.height,
        actualView,
      );
    }
    function open3D(point = pointer ?? { x: view.x, z: view.z }) {
      const h = metadataAt(tiles, point.x, point.z).height;
      p.onOpen3D?.([
        point.x,
        h === undefined || h === -32768 ? Math.max(64, p.player[1]) : h + 2,
        point.z,
      ]);
    }
    async function exportImage(save = true,filename=p.name??"World map",options:Partial<ExportOptions>={}) {
      const all = terrainBounds(locations);
      const selected = exportScope === "selection" ? selection : all;
      if (!selected) throw new Error("No map area selected");
      const spanX = (selected.maxX - selected.minX + 1) * 16,
        spanZ = (selected.maxZ - selected.minZ + 1) * 16,
        density = Math.min(
          exportDensity,
          16384 / spanX,
          16384 / spanZ,
          Math.sqrt((64 * 1024 * 1024) / (spanX * spanZ)),
        ),
        width = Math.max(1, Math.floor(spanX * density)),
        height = Math.max(1, Math.floor(spanZ * density));
      cancelExport.current = false;
      setExporting(true);
      setNotice("");
      setExportProgress(0);
      const task=save?exportTasks.begin("Render 2D map",{cancelable:true}):undefined;
      let rendered=0;
      try {
        const c = document.createElement("canvas");
        c.width = width;
        c.height = height;
        const ctx = c.getContext("2d");
        if (!ctx) throw new Error("Map canvas unavailable");
        const draw: MapDraw = {
          ...current.current,
          width,
          height,
          view: {
            x: (selected.minX + selected.maxX + 1) * 8,
            z: (selected.minZ + selected.maxZ + 1) * 8,
            scale: density,
            rotation: 0,
          },
          recent: undefined,
          style: { ...current.current.style, rotation: 0, recent: false },
          player: undefined,
          spawn: p.spawn,
          selection: undefined,
          annotations: false,
          clear: false,
          frameBudgetMs: undefined,
        };
        const inside = (t: Tile) =>
          t.x >= selected.minX &&
          t.x <= selected.maxX &&
          t.z >= selected.minZ &&
          t.z <= selected.maxZ;
        async function drawPage(batch: Tile[]) {
          const pageTiles = new Map(
            batch.filter(inside).map((t) => [`${t.x},${t.z}`, t]),
          );
          const local = new MapImages();
          try {
            local.sync(pageTiles, () => {});
            await Promise.all(
              [...local.images.values()].map(({ image }) => image.decode()),
            );
            if (cancelExport.current || !mounted.current || task?.signal.aborted)
              throw new Error("Export cancelled");
            const metadataTiles = new Map(tiles);
            for (const [k, t] of pageTiles) metadataTiles.set(k, t);
            paintMap(ctx!, local, { ...draw, tiles: pageTiles, metadataTiles });
          } finally {
            local.dispose();
          }
        }
        if (p.worldId) {
          let cursor: [number, number] | null = null,
            target = 0;
          do {
            if (cancelExport.current || !mounted.current || task?.signal.aborted)
              throw new Error("Export cancelled");
            const page = await api.GetMapPage(p.worldId, {
              cursor,
              target,
              y,
              pixels: density > 4 ? 16 : density > 1 ? 4 : 1,
              bounds: [
                selected.minX,
                selected.minZ,
                selected.maxX,
                selected.maxZ,
              ],
              limit: 8,
            });
            target = page.sequence;
            await drawPage(page.tiles);
            rendered+=page.tiles.length;task?.update("Drawing map tiles…",rendered,page.total);
            cursor = page.next;
            if (mounted.current)
              setExportProgress((v) => v + page.tiles.length);
            if (page.complete) break;
          } while (true);
        } else {
          const selectedTiles = [...tiles.values()].filter(inside);
          for (let i = 0; i < selectedTiles.length; i += 8) {
            await drawPage(selectedTiles.slice(i, i + 8));
            task?.update("Drawing map tiles…",Math.min(i+8,selectedTiles.length),selectedTiles.length);
            setExportProgress(i + 8);
          }
        }
        if (annotations) {
          const overlay = new MapImages();
          paintMap(ctx, overlay, {
            ...draw,
            tiles: new Map(),
            annotations: true,
          });
          overlay.dispose();
        }
        if (!transparent) {
          ctx.globalCompositeOperation = "destination-over";
          ctx.fillStyle = p.presentation?.background ?? "#202020";
          ctx.fillRect(0, 0, width, height);
        }
        const png = c.toDataURL("image/png");
        if (save) {
          task?.signal.throwIfAborted();task?.update("Encoding image…");
          const encoded=await imageExportData(png,(options.format||imageFormat) as ImageFormat,options.quality||90);
          task?.signal.throwIfAborted();task?.complete();
          const path = await api.SaveImage({name:filename,format:options.format||imageFormat,data:encoded,quality:options.quality||90,background:options.background||"#ffffff"});
          if (mounted.current) {
            setNotice(path ? `Saved: ${path}` : "Export cancelled");
            setExportOpen(false);
          }
        }
        return png;
      } catch(error){task?.fail(error);throw error} finally {
        if (mounted.current) setExporting(false);
      }
    }
    useImperativeHandle(ref, () => ({
      capture: () => exportImage(false),
      export: () => setExportOpen(true),
    }));
    const metadata = pointer
      ? metadataAt(tiles, pointer.x, pointer.z)
      : undefined;
    let length = 0,
      area = 0;
    for (let i = 1; i < measure.length; i++)
      length += Math.hypot(
        measure[i].x - measure[i - 1].x,
        measure[i].z - measure[i - 1].z,
      );
    if (closed && measure.length > 2) {
      length += Math.hypot(
        measure[0].x - measure.at(-1)!.x,
        measure[0].z - measure.at(-1)!.z,
      );
      for (let i = 0; i < measure.length; i++) {
        const a = measure[i],
          b = measure[(i + 1) % measure.length];
        area += a.x * b.z - b.x * a.z;
      }
      area = Math.abs(area) / 2;
    }
    const [imageFormat,setImageFormat]=useState<ImageFormat>("png");
    const exportBounds =
        exportScope === "selection" ? selection : terrainBounds(locations),
      sx = exportBounds ? (exportBounds.maxX - exportBounds.minX + 1) * 16 : 0,
      sz = exportBounds ? (exportBounds.maxZ - exportBounds.minZ + 1) * 16 : 0,
      exportScale =
        sx && sz
          ? Math.min(
              exportDensity,
              16384 / sx,
              16384 / sz,
              Math.sqrt((64 * 1024 * 1024) / (sx * sz)),
            )
          : 1;
    const toggle = (name: string) => setPanel((v) => (v === name ? "" : name));
    const check = (
      label: string,
      key: "relief" | "contours" | "grid" | "coverage" | "recent",
    ) => (
      <label className="atlas-check">
        <StudioInput
          type="checkbox"
          checked={activeStyle[key]}
          onChange={(e) => {
            setStyle((s) => ({ ...s, [key]: e.target.checked }));
            if (
              p.presentation &&
              p.onPresentationChange &&
              ["relief", "contours", "grid"].includes(key)
            )
              p.onPresentationChange({
                ...p.presentation,
                [key === "relief" ? "shading" : key]: e.target.checked,
              });
          }}
        />
        {label}
      </label>
    );
    return (
      <div className="world-map atlas-map" ref={host}>
        <canvas
          ref={canvas}
          tabIndex={0}
          aria-label="2D world map"
          className={`atlas-tool-${effectiveTool}`}
          onWheel={(e) => {
            e.preventDefault();
            const b = e.currentTarget.getBoundingClientRect();
            zoom(
              Math.exp(-Math.max(-100, Math.min(100, e.deltaY)) * 0.002),
              e.clientX - b.left,
              e.clientY - b.top,
            );
          }}
          onContextMenu={(e) => {
            e.preventDefault();
            setPointer(at(e));
            setPanel("point");
          }}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              setTool("pan");
              setPanel("");
              selectionStart.current = undefined;
              drag.current = undefined;
            }
            if (e.key === "+" || e.key === "=") zoom(1.4);
            if (e.key === "-") zoom(1 / 1.4);
            if (
              ["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(
                e.key,
              )
            ) {
              e.preventDefault();
              setFollow(false);
              setView((v) => ({
                ...v,
                x:
                  v.x +
                  (e.key === "ArrowRight"
                    ? 64
                    : e.key === "ArrowLeft"
                      ? -64
                      : 0) /
                    v.scale,
                z:
                  v.z +
                  (e.key === "ArrowDown" ? 64 : e.key === "ArrowUp" ? -64 : 0) /
                    v.scale,
              }));
            }
          }}
          onDoubleClick={(e) => {
            if (effectiveTool === "pan") open3D(at(e));
          }}
          onPointerDown={(e) => {
            if (e.button !== 0 && e.button !== 1) return;
            e.preventDefault();
            e.currentTarget.focus();
            e.currentTarget.setPointerCapture(e.pointerId);
            const point = at(e);
            setPointer(point);
            setFollow(false);
            if (e.button === 1 || effectiveTool === "pan") {
              drag.current = { x: e.clientX, y: e.clientY };
              return;
            }
            if (effectiveTool === "spawn") {
              p.onSpawnPick?.(Math.floor(point.x), Math.floor(point.z));
              return;
            }
            if (effectiveTool === "external-marker") {
              p.onMapPlace?.(point.x, point.z);
              return;
            }
            if (effectiveTool === "marker") {
              if (activeMarkers.length >= 512) {
                setNotice("Marker limit reached");
                return;
              }
              changeMarkers([
                ...activeMarkers,
                {
                  id: crypto.randomUUID(),
                  x: Math.floor(point.x),
                  z: Math.floor(point.z),
                  label: markerLabel.trim().slice(0, 100) || "Marker",
                  color: markerColor,
                  background: "#202020",
                  icon: "📍",
                  scale: 1,
                },
              ]);
              return;
            }
            if (effectiveTool === "measure") {
              setMeasure((old) =>
                closed ? [point] : [...old.slice(0, 511), point],
              );
              setClosed(false);
              return;
            }
            if (effectiveTool === "select") {
              let anchor = point;
              if (selection) {
                const b = selection;
                const corners = [
                  [b.minX * 16, b.minZ * 16],
                  [b.maxX * 16 + 16, b.minZ * 16],
                  [b.minX * 16, b.maxZ * 16 + 16],
                  [b.maxX * 16 + 16, b.maxZ * 16 + 16],
                ];
                const hit = corners.findIndex(
                  ([x, z]) =>
                    Math.hypot(point.x - x, point.z - z) * view.scale < 10,
                );
                if (hit >= 0) {
                  const q = corners[3 - hit];
                  anchor = {
                    x: q[0] + (hit % 2 ? 0 : -1),
                    z: q[1] + (hit < 2 ? -1 : 0),
                  };
                }
              }
              selectionStart.current = anchor;
              setSelection(selectedChunks(anchor, point));
            }
          }}
          onPointerMove={(e) => {
            const point = at(e);
            setPointer(point);
            if (selectionStart.current) {
              setSelection(selectedChunks(selectionStart.current, point));
              return;
            }
            if (drag.current) {
              const a = ((activeStyle.rotation ?? 0) * Math.PI) / 180,
                dx = (e.clientX - drag.current.x) / view.scale,
                dz = (e.clientY - drag.current.y) / view.scale;
              drag.current = { x: e.clientX, y: e.clientY };
              setView((v) => ({
                ...v,
                x: v.x - dx * Math.cos(a) - dz * Math.sin(a),
                z: v.z + dx * Math.sin(a) - dz * Math.cos(a),
              }));
            }
          }}
          onPointerUp={() => {
            drag.current = undefined;
            selectionStart.current = undefined;
          }}
          onPointerCancel={() => {
            drag.current = undefined;
            selectionStart.current = undefined;
          }}
        />
        {!tiles.size && !data.loading && (
          <div className="map-empty">
            <WorldIcon />
            <h3>No captured terrain</h3>
          </div>
        )}
        <div className="atlas-toolbar" role="toolbar" aria-label="Map tools">
          <Button
            variant={tool === "pan" ? "secondary" : "ghost"}
            size="icon"
            data-tooltip="Pan"
            aria-label="Pan"
            onClick={() => {
              setTool("pan");
              setPanel("");
            }}
          >
            <Hand />
          </Button>
          <Button
            variant={panel === "layers" ? "secondary" : "ghost"}
            size="icon"
            data-tooltip="Map layers"
            aria-label="Map layers"
            onClick={() => toggle("layers")}
          >
            <Layers />
          </Button>
          <Button
            variant={effectiveTool === "select" ? "secondary" : "ghost"}
            size="icon"
            data-tooltip="Select area"
            aria-label="Select area"
            onClick={() => {
              setTool("select");
              setPanel("selection");
            }}
          >
            <SquareDashed />
          </Button>
          <Button
            variant={tool === "marker" ? "secondary" : "ghost"}
            size="icon"
            data-tooltip="Markers"
            aria-label="Markers"
            onClick={() => {
              setTool("marker");
              toggle("markers");
            }}
          >
            <MapPin />
          </Button>
          <Button
            variant={tool === "measure" ? "secondary" : "ghost"}
            size="icon"
            data-tooltip="Measure"
            aria-label="Measure"
            onClick={() => {
              setTool("measure");
              setPanel("measure");
            }}
          >
            <Ruler />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            data-tooltip="Go to coordinates"
            aria-label="Go to coordinates"
            onClick={() => toggle("goto")}
          >
            <Navigation />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            data-tooltip="Export map image"
            aria-label="Export map image"
            disabled={!tiles.size}
            onClick={() => setExportOpen(true)}
          >
            <Download />
          </Button>
        </div>
        <div className="atlas-status">
          {data.loading
            ? "Loading map…"
            : `${locations.length.toLocaleString("en-US")} chunks`}
          {y !== undefined && ` · Y ≤ ${y}`}
        </div>
        {panel && (
          <section className="atlas-panel" aria-label="Map inspector">
            <header>
              <strong>
                {
                  (
                    {
                      layers: "Layers & detail",
                      selection: "Selected area",
                      markers: "Markers",
                      measure: "Measure",
                      goto: "Go to",
                      point: "Position",
                    } as Record<string, string>
                  )[panel]
                }
              </strong>
              <Button
                size="icon"
                variant="ghost"
                aria-label="Close map inspector"
                onClick={() => setPanel("")}
              >
                <X />
              </Button>
            </header>
            {panel === "layers" && (
              <>
                <label>
                  Layer
                  <select
                    aria-label="Map layer"
                    value={style.layer}
                    onChange={(e) =>
                      setStyle((s) => ({
                        ...s,
                        layer: e.target.value as MapStyle["layer"],
                      }))
                    }
                  >
                    <option value="surface">Surface</option>
                    <option value="height">Elevation</option>
                    <option value="biome">Biomes</option>
                  </select>
                </label>
                <label>
                  Detail
                  <select
                    aria-label="Map detail"
                    value={style.detail}
                    onChange={(e) =>
                      setStyle((s) => ({
                        ...s,
                        detail: e.target.value as MapStyle["detail"],
                      }))
                    }
                  >
                    <option value="textures">Resource pack textures</option>
                    <option value="simple">Simple colors</option>
                  </select>
                </label>
                <label className="atlas-check">
                  <StudioInput
                    type="checkbox"
                    checked={y !== undefined}
                    onChange={(e) =>
                      setY(
                        e.target.checked
                          ? Math.round(p.player[1] || 64)
                          : undefined,
                      )
                    }
                  />
                  Y slice
                </label>
                {y !== undefined && (
                  <label>
                    Maximum Y
                    <StudioInput
                      aria-label="Slice height"
                      type="number"
                      min={-32768}
                      max={32767}
                      value={y}
                      onChange={(e) => {
                        const n = Number(e.target.value);
                        if (Number.isInteger(n) && n >= -32768 && n <= 32767)
                          setY(n);
                      }}
                    />
                  </label>
                )}
                {check("Terrain relief", "relief")}
                {check("Contours", "contours")}
                {activeStyle.contours && (
                  <label>
                    Contour interval
                    <StudioInput
                      aria-label="Contour interval"
                      type="number"
                      min={1}
                      max={128}
                      value={activeStyle.contourStep}
                      onChange={(e) => {
                        const n = Number(e.target.value);
                        if (n >= 1 && n <= 128) {
                          setStyle((s) => ({ ...s, contourStep: n }));
                          if (p.presentation && p.onPresentationChange)
                            p.onPresentationChange({
                              ...p.presentation,
                              contourStep: n,
                            });
                        }
                      }}
                    />
                  </label>
                )}
                {check("Coordinate grid", "grid")}
                {check("Capture coverage", "coverage")}
                {check("Recent updates", "recent")}
                {style.layer === "height" && (
                  <div className="atlas-height-legend">
                    {elevationRange[0]} <span />
                    {elevationRange[1]}
                  </div>
                )}
                {style.layer === "biome" && (
                  <div className="atlas-biomes">
                    {[
                      ...new Set(
                        [...tiles.values()].flatMap((t) => t.biomes ?? []),
                      ),
                    ]
                      .slice(0, 32)
                      .map((id) => (
                        <span key={id}>
                          <i
                            style={{
                              background: `rgb(${biomeColor(id).join(",")})`,
                            }}
                          />
                          {[...tiles.values()].find(
                            (t) => t.biomeNames?.[String(id)],
                          )?.biomeNames?.[String(id)] ?? `Biome ${id}`}
                        </span>
                      ))}
                  </div>
                )}
              </>
            )}
            {panel === "selection" && (
              <>
                {selection ? (
                  <DialogValidation>
                    <div className="atlas-coordinate-fields">
                      {(["minX", "maxX", "minZ", "maxZ"] as const).map(
                        (key) => (
                          <label key={key}>
                            {key.replace("min", "Min ").replace("max", "Max ")}
                            <StudioInput
                              aria-label={`Selection ${key}`}
                              type="number"
                              min={-1875000}
                              max={1875000}
                              step={1}
                              error={
                                key.endsWith("X")
                                  ? rangeOrderError(
                                      selection.minX,
                                      selection.maxX,
                                    )
                                  : rangeOrderError(
                                      selection.minZ,
                                      selection.maxZ,
                                    )
                              }
                              value={selection[key]}
                              onChange={(e) => {
                                const n = Number(e.target.value);
                                if (Number.isInteger(n) && bounded(n * 16))
                                  setSelection({ ...selection, [key]: n });
                              }}
                            />
                          </label>
                        ),
                      )}
                    </div>
                    <p>
                      {coverage?.captured} / {coverage?.total} chunks captured
                    </p>
                    <p className={coverage?.missing ? "atlas-warning" : ""}>
                      {coverage?.missing} missing ·{" "}
                      {(coverage?.total ?? 0) * 256} blocks²
                    </p>
                    {p.onRequestRegion && (
                      <Button
                        requiresValid
                        disabled={
                          !!(
                            rangeOrderError(selection.minX, selection.maxX) ||
                            rangeOrderError(selection.minZ, selection.maxZ)
                          )
                        }
                        onClick={p.onRequestRegion}
                      >
                        Save region…
                      </Button>
                    )}
                    {p.onOpenSelection3D&&<Button requiresValid disabled={!coverage?.captured} onClick={()=>p.onOpenSelection3D?.({...selection})}><Box/>Open selection in 3D</Button>}
                    {p.onCreateSelectionMap&&<Button requiresValid variant="outline" disabled={!coverage?.captured} onClick={()=>p.onCreateSelectionMap?.({...selection})}>Create separate map…</Button>}
                    <Button
                      requiresValid
                      variant="outline"
                      disabled={
                        !!(
                          rangeOrderError(selection.minX, selection.maxX) ||
                          rangeOrderError(selection.minZ, selection.maxZ)
                        )
                      }
                      onClick={() => {
                        setExportScope("selection");
                        setExportOpen(true);
                      }}
                    >
                      Export selection…
                    </Button>
                    <Button
                      variant="ghost"
                      onClick={() => {
                        setLocalSelection(undefined);
                        p.onSelectionClear?.();
                        setTool("pan");
                      }}
                    >
                      Clear selection
                    </Button>
                  </DialogValidation>
                ) : (
                  <p>Drag to select. Drag corners to resize.</p>
                )}
              </>
            )}
            {panel === "markers" && (
              <>
                <label>
                  Name
                  <StudioInput
                    aria-label="Marker name"
                    maxLength={100}
                    value={markerLabel}
                    onChange={(e) => setMarkerLabel(e.target.value)}
                  />
                </label>
                <label>
                  Color
                  <StudioInput
                    aria-label="Marker color"
                    type="color"
                    value={markerColor}
                    onChange={(e) => setMarkerColor(e.target.value)}
                  />
                </label>
                <p>Click the map to place.</p>
                <div className="atlas-marker-list">
                  {activeMarkers.map((m) => (
                    <div key={m.id}>
                      <button
                        onClick={() => {
                          setFollow(false);
                          setView((v) => ({ ...v, x: m.x, z: m.z }));
                        }}
                      >
                        {m.label}
                        <small>
                          {m.x}, {m.z}
                        </small>
                      </button>
                      <Button
                        size="icon"
                        variant="ghost"
                        aria-label={`Delete marker ${m.label}`}
                        onClick={() =>
                          changeMarkers(
                            activeMarkers.filter((n) => n.id !== m.id),
                          )
                        }
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  ))}
                </div>
              </>
            )}
            {panel === "measure" && (
              <>
                <p>
                  {measure.length} points · {length.toFixed(1)} blocks
                </p>
                {closed && <p>{area.toFixed(1)} blocks²</p>}
                <p>Click points to measure horizontal distances.</p>
                <Button
                  disabled={measure.length < 3}
                  variant="outline"
                  onClick={() => setClosed(true)}
                >
                  Close polygon
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => {
                    setMeasure([]);
                    setClosed(false);
                  }}
                >
                  Clear
                </Button>
              </>
            )}
            {panel === "goto" && (
              <StudioForm
                onSubmit={(e) => {
                  e.preventDefault();
                  if (bounded(+go.x) && bounded(+go.z)) {
                    setView((v) => ({ ...v, x: +go.x, z: +go.z }));
                    setFollow(false);
                  }
                }}
              >
                <div className="atlas-coordinate-fields">
                  {(["x", "z"] as const).map((k) => (
                    <label key={k}>
                      {k.toUpperCase()}
                      <StudioInput
                        type="number"
                        aria-label={`Go to ${k.toUpperCase()}`}
                        min={-30000000}
                        max={30000000}
                        step="any"
                        value={go[k]}
                        onChange={(e) =>
                          setGo((v) => ({ ...v, [k]: e.target.value }))
                        }
                      />
                    </label>
                  ))}
                </div>
                <Button type="submit">Go</Button>
              </StudioForm>
            )}
            {panel === "point" && pointer && (
              <>
                <p>
                  X {Math.floor(pointer.x)} · Z {Math.floor(pointer.z)}
                </p>
                <p>
                  {metadata?.block?.replace("minecraft:", "") ?? "Not captured"}
                </p>
                <Button
                  variant="outline"
                  onClick={() => {
                    void navigator.clipboard
                      .writeText(
                        `${Math.floor(pointer.x)} ${metadata?.height ?? "~"} ${Math.floor(pointer.z)}`,
                      )
                      .then(
                        () => setNotice("Coordinates copied"),
                        () => setNotice("Clipboard unavailable"),
                      );
                  }}
                >
                  <Copy />
                  Copy coordinates
                </Button>
                {p.onOpen3D && (
                  <Button onClick={() => open3D()}>
                    <Box />
                    Open in 3D
                  </Button>
                )}
              </>
            )}
          </section>
        )}
        <div className="atlas-navigation">
          <Button
            size="icon"
            variant="outline"
            data-tooltip="Show all terrain"
            aria-label="Show all terrain"
            onClick={() => {
              setView(fitTiles(locations, size.width, size.height));
              setFollow(false);
            }}
          >
            <Scan />
          </Button>
          <Button
            size="icon"
            variant="outline"
            aria-label="Zoom in on map"
            onClick={() => zoom(1.4)}
          >
            <Plus />
          </Button>
          <Button
            size="icon"
            variant="outline"
            aria-label="Zoom out on map"
            onClick={() => zoom(1 / 1.4)}
          >
            <Minus />
          </Button>
          <Button
            size="icon"
            variant={follow ? "secondary" : "outline"}
            aria-label="Follow player"
            aria-pressed={follow}
            disabled={!p.hasPlayer}
            onClick={() => setFollow((v) => !v)}
          >
            <Crosshair />
          </Button>
          {p.onOpen3D && (
            <Button
              size="icon"
              variant="outline"
              aria-label="Open map center in 3D"
              onClick={() => open3D({ x: view.x, z: view.z })}
            >
              <Box />
            </Button>
          )}
        </div>
        <div className="atlas-coordinate">
          {pointer ? (
            <>
              X {Math.floor(pointer.x)} ·{" "}
              {metadata?.height !== undefined && metadata.height !== -32768
                ? `Y ${metadata.height} · `
                : ""}
              Z {Math.floor(pointer.z)}
              <span>
                Chunk {Math.floor(pointer.x / 16)}, {Math.floor(pointer.z / 16)}
                {metadata?.biome !== undefined
                  ? ` · ${metadata.biomeName ?? `Biome ${metadata.biome}`}`
                  : ""}
                {metadata?.block
                  ? ` · ${metadata.block.replace("minecraft:", "")}`
                  : ""}
              </span>
            </>
          ) : (
            "Drag to pan · Scroll to zoom"
          )}
        </div>
        {data.error && (
          <FeedbackFlyout title="World map">
            {data.error}
          </FeedbackFlyout>
        )}
        <MediaExportDialog open={exportOpen} onOpenChange={v=>{if(!exporting)setExportOpen(v);}} title="Export map image" format={imageFormat} onFormat={f=>setImageFormat(f as ImageFormat)} edge={1920} onEdge={()=>{}} aspect={sx/Math.max(1,sz)} name={p.name??"World map"} dimensions={{width:Math.floor(sx*exportScale),height:Math.floor(sz*exportScale)}} transparent={transparent} onTransparent={setTransparent} busy={exporting} status={exporting?`Preparing ${exportProgress} chunks …`:undefined} onCancel={()=>{cancelExport.current=true;}} onExport={(name,options)=>{setExportOpen(true);void exportImage(true,name,options).catch(e=>notifyFailure(e,"Map export failed"));}}>
         <label>Area<select aria-label="Export map area" value={exportScope} onChange={e=>setExportScope(e.target.value)}><option value="all">Entire map</option><option value="selection" disabled={!selection}>Selection</option></select></label>
         <label>Pixels per block<select aria-label="Export map density" value={exportDensity} onChange={e=>setExportDensity(+e.target.value)}>{[1,2,4,8,16].map(n=><option key={n} value={n}>{n}</option>)}</select></label>
         <label className="export-studio-check"><StudioInput type="checkbox" checked={annotations} onChange={e=>setAnnotations(e.target.checked)}/>Markers, grid & scale</label>
         {exportScale<exportDensity&&<p>Resolution fitted to the 64 MP budget.</p>}
        </MediaExportDialog>
      </div>
    );
  },
);
export default AtlasMap;

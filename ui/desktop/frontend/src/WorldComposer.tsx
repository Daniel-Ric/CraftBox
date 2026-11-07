import FeedbackFlyout from "./FeedbackFlyout";
import StudioInput from "@/StudioInput";
import { rangeOrderError } from "./input-rules";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  Plus,
  Trash2,
  Scan,
  Move,
  LoaderCircle,
  ArrowRight,
  ArrowDown,
  Copy,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  api,
  type World,
  type Tile,
  type ChunkBounds,
  type WorldCompositionPreview,
} from "./bridge";
import {
  arrangePlacements,
  placedBounds,
  type Placement,
} from "./world-composition";
import {
  pointAt,
  screenAt,
  zoomAt,
  terrainBounds,
  type MapView,
} from "./map-workbench";
import "./world-composer.css";
const colors = [
  "#7cb7df",
  "#dca873",
  "#8cc492",
  "#bb9bdf",
  "#dd929b",
  "#88c6c8",
];
export default function WorldComposer({
  open,
  onOpenChange,
  worlds,
  currentWorldId,
  selection,
  tiles,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  worlds: World[];
  currentWorldId: string;
  selection?: ChunkBounds;
  tiles: Map<string, Tile>;
  onCreated: (w: World) => void;
}) {
  const [parts, setParts] = useState<Placement[]>([]),
    [selected, setSelected] = useState(""),
    [name, setName] = useState("Merged world"),
    [gap, setGap] = useState(16),
    [snap, setSnap] = useState(true),
    [phase, setPhase] = useState<"idle" | "preparing" | "saving">("idle"),
    [error, setError] = useState(""),
    [preview, setPreview] = useState<WorldCompositionPreview>(),
    [progress, setProgress] = useState({ done: 0, total: 0, stage: "" }),
    [nativeTiles, setNativeTiles] = useState(new Map<string, Tile>()),
    [nativeBounds, setNativeBounds] = useState<Record<string, ChunkBounds>>({}),
    [view, setView] = useState<MapView>({ x: 0, z: 0, scale: 1 }),
    [size, setSize] = useState({ width: 720, height: 600 });
  const canvas = useRef<HTMLCanvasElement>(null),
    [hostNode, setHostNode] = useState<HTMLElement | null>(null),
    images = useRef(new Map<string, HTMLImageElement>()),
    [imageRevision, setImageRevision] = useState(0),
    drag = useRef<
      | { id: string; x: number; z: number; offset: [number, number, number] }
      | undefined
    >(undefined),
    pan = useRef<{ x: number; y: number } | undefined>(undefined),
    needsArrange = useRef(false),
    alive = useRef(true),
    pendingPlacement = useRef(new Set<string>());
  const busy = phase !== "idle",
    sourceIDs = [...new Set(parts.map((p) => p.worldId))].join("|");
  const sourceTiles = useMemo(() => {
    const map = new Map<string, Tile>();
    for (const t of tiles.values())
      if (t.worldId) map.set(`${t.worldId}:${t.x},${t.z}`, t);
    for (const [k, t] of nativeTiles) map.set(k, t);
    return map;
  }, [tiles, nativeTiles]);
  const bounds = useMemo(() => {
    const result = { ...nativeBounds };
    for (const id of worlds.map((p) => p.id)) {
      if (!result[id]) {
        const b = terrainBounds(
          [...sourceTiles.values()].filter((t) => t.worldId === id),
        );
        if (b) result[id] = b;
      }
    }
    return result;
  }, [nativeBounds, sourceTiles, worlds]);
  const pSelected = parts.find((p) => p.id === selected);
  const layoutReady =
    parts.length > 0 &&
    parts.every((p) => !!bounds[p.worldId]) &&
    !needsArrange.current &&
    !pendingPlacement.current.size;
  function fit(list = parts) {
    if (!list.length) return;
    const boxes = list.map((p) =>
        placedBounds(
          p,
          bounds[p.worldId] ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 },
        ),
      ),
      minX = Math.min(...boxes.map((b) => b[0])),
      minZ = Math.min(...boxes.map((b) => b[1])),
      maxX = Math.max(...boxes.map((b) => b[2])),
      maxZ = Math.max(...boxes.map((b) => b[3]));
    setView({
      x: (minX + maxX) / 2,
      z: (minZ + maxZ) / 2,
      scale: Math.max(
        0.001,
        Math.min(
          16,
          (size.width - 80) / Math.max(16, maxX - minX),
          (size.height - 80) / Math.max(16, maxZ - minZ),
        ),
      ),
    });
  }
  useEffect(() => {
    if (!open) return;
    const id = currentWorldId || worlds[0]?.id;
    if (!id) return;
    const part: Placement = {
      id: crypto.randomUUID(),
      worldId: id,
      offset: [0, 0, 0],
      bounds: selection,
      color: colors[0],
    };
    setParts([part]);
    setSelected(part.id);
    pendingPlacement.current.clear();
    setPreview(undefined);
    setError("");
    needsArrange.current = true;
  }, [open]);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  useEffect(() => {
    if (!open || !hostNode) return;
    const observer = new ResizeObserver((items) => {
      const box = items[0]?.contentRect;
      if (box) setSize({ width: box.width, height: box.height });
    });
    observer.observe(hostNode);
    return () => observer.disconnect();
  }, [open, hostNode]);
  useEffect(() => {
    if (!open) return;
    let active = true;
    void (async () => {
      for (const id of sourceIDs.split("|").filter(Boolean)) {
        try {
          let cursor: [number, number] | null = null,
            target = 0;
          do {
            const page = await api.GetMapPage(id, {
              cursor,
              target,
              pixels: 1,
              limit: 32,
            });
            if (!active) return;
            target = page.sequence;
            if (page.coverage) {
              const b = terrainBounds(
                page.coverage.map(([x, z]) => ({ x, z })),
              );
              if (b) setNativeBounds((old) => ({ ...old, [id]: b }));
            }
            setNativeTiles((old) => {
              const next = new Map(old);
              for (const t of page.tiles)
                next.set(`${id}:${t.x},${t.z}`, { ...t, worldId: id });
              return next;
            });
            cursor = page.next;
            if (page.complete) break;
          } while (active);
        } catch (e) {
          if (active) setError(String(e));
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [open, sourceIDs]);
  useEffect(() => {
    if (
      needsArrange.current &&
      parts.length &&
      parts.every((p) => bounds[p.worldId])
    ) {
      needsArrange.current = false;
      const next = arrangePlacements(parts, bounds, gap);
      setParts(next);
      fit(next);
    }
  }, [bounds, parts, gap]);
  useEffect(() => {
    const ready = parts.filter(
      (p) => pendingPlacement.current.has(p.id) && bounds[p.worldId],
    );
    if (!ready.length) return;
    let next = [...parts];
    for (const part of ready) {
      pendingPlacement.current.delete(part.id);
      const b = part.bounds ?? bounds[part.worldId];
      const others = next.filter((p) => p.id !== part.id);
      const right = Math.max(
        0,
        ...others.map(
          (p) =>
            placedBounds(
              p,
              bounds[p.worldId] ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 },
            )[2],
        ),
      );
      next = next.map((p) =>
        p.id === part.id
          ? {
              ...p,
              offset: [
                Math.ceil((right + gap - b.minX * 16) / (snap ? 16 : 1)) *
                  (snap ? 16 : 1),
                p.offset[1],
                -b.minZ * 16,
              ],
            }
          : p,
      );
    }
    setParts(next);
    fit(next);
  }, [bounds, parts, gap, snap]);
  useEffect(() => {
    if (!busy) return;
    let active = true;
    const timer = setInterval(
      () =>
        void api
          .GetCompositionProgress()
          .then((p) => {
            if (active) setProgress({ ...p, stage: p.stage ?? "" });
          })
          .catch(() => {}),
      300,
    );
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [busy]);
  useEffect(() => {
    for (const [k, t] of sourceTiles) {
      const old = images.current.get(k);
      if (old?.dataset.source === t.image) continue;
      const img = new Image();
      img.dataset.source = t.image;
      img.onload = () => {
        if (alive.current) setImageRevision((v) => v + 1);
      };
      img.src = t.image;
      images.current.set(k, img);
    }
  }, [sourceTiles]);
  useEffect(() => {
    if (!open || !canvas.current) return;
    const element = canvas.current,
      ctx = element.getContext("2d");
    if (!ctx) return;
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    element.width = Math.max(1, Math.round(size.width * dpr));
    element.height = Math.max(1, Math.round(size.height * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, size.width, size.height);
    ctx.save();
    ctx.translate(size.width / 2, size.height / 2);
    ctx.scale(view.scale, view.scale);
    ctx.translate(-view.x, -view.z);
    ctx.imageSmoothingEnabled = view.scale < 1;
    for (const p of parts) {
      const b = placedBounds(
        p,
        bounds[p.worldId] ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 },
      );
      ctx.save();
      ctx.beginPath();
      ctx.rect(b[0], b[1], b[2] - b[0], b[3] - b[1]);
      ctx.clip();
      for (const [k, t] of sourceTiles) {
        if (t.worldId !== p.worldId) continue;
        const image = images.current.get(k);
        if (image?.complete && image.naturalWidth)
          ctx.drawImage(
            image,
            t.x * 16 + p.offset[0],
            t.z * 16 + p.offset[2],
            16,
            16,
          );
      }
      ctx.restore();
      ctx.fillStyle = p.color + "22";
      ctx.fillRect(b[0], b[1], b[2] - b[0], b[3] - b[1]);
      ctx.strokeStyle = p.color;
      ctx.lineWidth = (p.id === selected ? 3 : 1) / view.scale;
      ctx.strokeRect(b[0], b[1], b[2] - b[0], b[3] - b[1]);
    }
    ctx.restore();
    ctx.font = "12px Segoe UI, sans-serif";
    for (const p of parts) {
      const b = placedBounds(
          p,
          bounds[p.worldId] ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 },
        ),
        q = screenAt(b[0], b[1], size.width, size.height, view),
        label = worlds.find((w) => w.id === p.worldId)?.name ?? p.worldId;
      ctx.fillStyle = "#222";
      ctx.fillRect(q.x, q.y - 22, ctx.measureText(label).width + 12, 20);
      ctx.fillStyle = p.color;
      ctx.fillText(label, q.x + 6, q.y - 8);
    }
  }, [open, sourceTiles, parts, bounds, view, size, selected, imageRevision]);
  useEffect(
    () => () => {
      for (const image of images.current.values()) image.onload = null;
      images.current.clear();
    },
    [],
  );
  function updatePart(change: Partial<Placement>) {
    pendingPlacement.current.delete(selected);
    setParts((old) =>
      old.map((p) => (p.id === selected ? { ...p, ...change } : p)),
    );
    setError("");
  }
  function add(id: string, original?: Placement) {
    if (parts.length >= 16) return;
    const b = original?.bounds ??
      bounds[id] ?? { minX: 0, minZ: 0, maxX: 0, maxZ: 0 };
    const right = Math.max(
      ...parts.map((p) => placedBounds(p, bounds[p.worldId] ?? b)[2]),
      0,
    );
    const part: Placement = {
      id: crypto.randomUUID(),
      worldId: id,
      bounds: original?.bounds ? { ...original.bounds } : undefined,
      offset: [
        Math.ceil((right + gap - b.minX * 16) / (snap ? 16 : 1)) *
          (snap ? 16 : 1),
        original?.offset[1] ?? 0,
        -b.minZ * 16,
      ],
      color: colors[parts.length % colors.length],
    };
    if (!bounds[id]) pendingPlacement.current.add(part.id);
    setParts((old) => [...old, part]);
    setSelected(part.id);
    if (bounds[id]) fit([...parts, part]);
    setError("");
  }
  async function savePrepared(value: WorldCompositionPreview, allow: boolean) {
    setPhase("saving");
    setError("");
    try {
      const world = await api.SaveWorldComposition(value.id, allow);
      if (alive.current) {
        setPreview(undefined);
        onCreated(world);
        onOpenChange(false);
      }
    } catch (e) {
      if (alive.current) setError(String(e));
    } finally {
      if (alive.current) setPhase("idle");
    }
  }
  async function prepare() {
    if (!layoutReady) {
      setError("Wait for source maps to load");
      return;
    }
    setPhase("preparing");
    setError("");
    try {
      const value = await api.PrepareWorldComposition({
        name,
        parts: parts.map(({ worldId, offset, bounds }) => ({
          worldId,
          offset,
          bounds,
        })),
      });
      if (!alive.current) {
        void api.DiscardComposition(value.id);
        return;
      }
      if (value.conflicts) {
        setPreview(value);
        setPhase("idle");
      } else await savePrepared(value, false);
    } catch (e) {
      if (alive.current) {
        setError(String(e));
        setPhase("idle");
      }
    }
  }
  const close = () => {
    if (busy) return;
    if (preview) void api.DiscardComposition(preview.id);
    setPreview(undefined);
    onOpenChange(false);
  };
  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(v) => {
          if (!v) close();
        }}
      >
        <DialogContent className="world-composer">
          <DialogHeader>
            <DialogTitle>Merge worlds</DialogTitle>
            <DialogDescription>
              Place captured worlds or regions in a new world
            </DialogDescription>
          </DialogHeader>
          <div className="composition-body">
            <section className="composition-canvas" ref={setHostNode}>
              <canvas
                ref={canvas}
                aria-label="World placement preview"
                onWheel={(e) => {
                  e.preventDefault();
                  const b = e.currentTarget.getBoundingClientRect();
                  setView((v) =>
                    zoomAt(
                      v,
                      e.deltaY < 0 ? 1.15 : 1 / 1.15,
                      e.clientX - b.left,
                      e.clientY - b.top,
                      size.width,
                      size.height,
                    ),
                  );
                }}
                onPointerDown={(e) => {
                  if (busy) return;
                  e.currentTarget.setPointerCapture(e.pointerId);
                  const b = e.currentTarget.getBoundingClientRect(),
                    q = pointAt(
                      e.clientX - b.left,
                      e.clientY - b.top,
                      size.width,
                      size.height,
                      view,
                    );
                  const hit = [
                    ...parts.filter((p) => p.id === selected),
                    ...parts.filter((p) => p.id !== selected).reverse(),
                  ].find((p) => {
                    const b = placedBounds(
                      p,
                      bounds[p.worldId] ?? {
                        minX: 0,
                        minZ: 0,
                        maxX: 0,
                        maxZ: 0,
                      },
                    );
                    return (
                      q.x >= b[0] && q.z >= b[1] && q.x <= b[2] && q.z <= b[3]
                    );
                  });
                  if (hit && e.button === 0) {
                    setSelected(hit.id);
                    drag.current = {
                      id: hit.id,
                      x: q.x,
                      z: q.z,
                      offset: [...hit.offset],
                    };
                  } else pan.current = { x: e.clientX, y: e.clientY };
                }}
                onPointerMove={(e) => {
                  if (busy) return;
                  const b = e.currentTarget.getBoundingClientRect(),
                    q = pointAt(
                      e.clientX - b.left,
                      e.clientY - b.top,
                      size.width,
                      size.height,
                      view,
                    );
                  if (drag.current) {
                    const d = drag.current,
                      unit = snap ? 16 : 1;
                    setParts((old) =>
                      old.map((p) =>
                        p.id === d.id
                          ? {
                              ...p,
                              offset: [
                                Math.round((d.offset[0] + q.x - d.x) / unit) *
                                  unit,
                                d.offset[1],
                                Math.round((d.offset[2] + q.z - d.z) / unit) *
                                  unit,
                              ],
                            }
                          : p,
                      ),
                    );
                  } else if (pan.current) {
                    const dx = (e.clientX - pan.current.x) / view.scale,
                      dz = (e.clientY - pan.current.y) / view.scale;
                    pan.current = { x: e.clientX, y: e.clientY };
                    setView((v) => ({ ...v, x: v.x - dx, z: v.z - dz }));
                  }
                }}
                onPointerUp={() => {
                  drag.current = undefined;
                  pan.current = undefined;
                }}
                onPointerCancel={() => {
                  drag.current = undefined;
                  pan.current = undefined;
                }}
              />
              <div className="composition-canvas-tools">
                <Button variant="outline" onClick={() => fit()}>
                  <Scan />
                  Fit
                </Button>
                <span>
                  <Move size={13} />
                  Drag to place · Scroll to zoom
                </span>
              </div>
            </section>
            <aside className="composition-inspector">
              <label>
                World name
                <StudioInput
                  aria-label="Merged world name"
                  required
                  maxLength={100}
                  value={name}
                  disabled={busy}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <section
                className="composition-sources"
                aria-label="Captured worlds"
              >
                <strong>Captured worlds</strong>
                {worlds.map((w) => (
                  <div key={w.id} className="composition-source">
                    <div>
                      <strong>{w.name}</strong>
                      <small>
                        {w.chunks.toLocaleString("en-US")} chunks ·{" "}
                        {["recording", "downloading", "capturing"].includes(
                          w.state.toLowerCase(),
                        )
                          ? "Recording"
                          : "Captured"}
                      </small>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      aria-label={`Add ${w.name}`}
                      disabled={busy || parts.length >= 16 || !w.chunks}
                      onClick={() => add(w.id)}
                    >
                      <Plus />
                      Add
                    </Button>
                  </div>
                ))}
              </section>
              <div
                className="composition-parts"
                role="list"
                aria-label="World placements"
              >
                {parts.map((p, i) => (
                  <div
                    key={p.id}
                    className={p.id === selected ? "selected" : ""}
                    role="listitem"
                  >
                    <button disabled={busy} onClick={() => setSelected(p.id)}>
                      <i style={{ background: p.color }} />
                      {i + 1}.{" "}
                      {worlds.find((w) => w.id === p.worldId)?.name ??
                        p.worldId}
                    </button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Duplicate placement ${i + 1}`}
                      data-tooltip="Duplicate"
                      disabled={busy || parts.length >= 16}
                      onClick={() => add(p.worldId, p)}
                    >
                      <Copy />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Remove placement ${i + 1}`}
                      disabled={busy}
                      onClick={() => {
                        pendingPlacement.current.delete(p.id);
                        setParts((old) => old.filter((n) => n.id !== p.id));
                        if (p.id === selected)
                          setSelected(
                            parts.find((n) => n.id !== p.id)?.id ?? "",
                          );
                      }}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                ))}
              </div>
              {pSelected && (
                <fieldset disabled={busy}>
                  <legend>Selected placement</legend>
                  <div className="composition-offsets">
                    {(["X", "Y", "Z"] as const).map((axis, i) => (
                      <label key={axis}>
                        Offset {axis}
                        <StudioInput
                          type="number"
                          aria-label={`Placement offset ${axis}`}
                          value={pSelected.offset[i]}
                          step={1}
                          min={i === 1 ? -383 : -30000000}
                          max={i === 1 ? 383 : 30000000}
                          onChange={(e) => {
                            const n = Number(e.target.value);
                            if (!Number.isInteger(n) || Math.abs(n) > 30000000)
                              return;
                            const offset: [number, number, number] = [
                              ...pSelected.offset,
                            ];
                            offset[i] = n;
                            updatePart({ offset });
                          }}
                        />
                      </label>
                    ))}
                  </div>
                  <label className="composition-check">
                    <StudioInput
                      type="checkbox"
                      checked={!!pSelected.bounds}
                      onChange={(e) =>
                        updatePart({
                          bounds: e.target.checked
                            ? ((pSelected.worldId === currentWorldId
                                ? selection
                                : undefined) ?? bounds[pSelected.worldId])
                            : undefined,
                        })
                      }
                    />
                    Source region
                  </label>
                  {pSelected.bounds && (
                    <div className="composition-bounds">
                      {(["minX", "maxX", "minZ", "maxZ"] as const).map(
                        (key) => (
                          <label key={key}>
                            {key}
                            <StudioInput
                              type="number"
                              aria-label={`Source ${key}`}
                              value={pSelected.bounds![key]}
                              min={-1875000}
                              max={1875000}
                              step={1}
                              error={
                                key.endsWith("X")
                                  ? rangeOrderError(
                                      pSelected.bounds!.minX,
                                      pSelected.bounds!.maxX,
                                    )
                                  : rangeOrderError(
                                      pSelected.bounds!.minZ,
                                      pSelected.bounds!.maxZ,
                                    )
                              }
                              onChange={(e) => {
                                const n = Number(e.target.value);
                                if (
                                  Number.isInteger(n) &&
                                  Math.abs(n) <= 1875000
                                )
                                  updatePart({
                                    bounds: { ...pSelected.bounds!, [key]: n },
                                  });
                              }}
                            />
                          </label>
                        ),
                      )}
                    </div>
                  )}
                  {selection && pSelected.worldId === currentWorldId && (
                    <Button
                      variant="outline"
                      onClick={() => updatePart({ bounds: selection })}
                    >
                      Use selected area
                    </Button>
                  )}
                </fieldset>
              )}
              <fieldset disabled={busy}>
                <legend>Arrange</legend>
                <label>
                  Gap in blocks
                  <StudioInput
                    type="number"
                    aria-label="Placement gap"
                    step={1}
                    min={0}
                    max={1000000}
                    value={gap}
                    onChange={(e) =>
                      setGap(
                        Math.max(
                          0,
                          Math.min(1000000, Math.floor(+e.target.value) || 0),
                        ),
                      )
                    }
                  />
                </label>
                <div className="composition-arrange">
                  {(["east", "south"] as const).map((direction) => (
                    <Button
                      key={direction}
                      variant="outline"
                      onClick={() => {
                        const next = arrangePlacements(
                          parts,
                          bounds,
                          gap,
                          direction,
                        );
                        setParts(next);
                        fit(next);
                      }}
                    >
                      {direction === "east" ? <ArrowRight /> : <ArrowDown />}
                      {direction === "east" ? "Side by side" : "North to south"}
                    </Button>
                  ))}
                </div>
                <label className="composition-check">
                  <StudioInput
                    type="checkbox"
                    checked={snap}
                    onChange={(e) => setSnap(e.target.checked)}
                  />
                  Snap to chunks
                </label>
              </fieldset>
              <p className="secondary">
                Later placements replace earlier blocks. Air is ignored.
              </p>
            </aside>
          </div>
          <div className="composition-status" role="status">
            {error ||
              (busy
                ? `${progress.stage || (phase === "preparing" ? "Checking blocks" : "Saving world")}… ${progress.done} / ${progress.total} chunks`
                : !layoutReady
                  ? "Loading source maps…"
                  : `${parts.length} placements · Original worlds stay unchanged`)}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                if (busy) void api.CancelComposition();
                else close();
              }}
            >
              {busy ? "Cancel operation" : "Cancel"}
            </Button>
            <Button
              disabled={busy || !layoutReady || !name.trim()}
              onClick={() => void prepare()}
            >
              {busy && <LoaderCircle className="spin" />}Save merged world
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={!!preview}
        onOpenChange={(v) => {
          if (!v && !busy && preview) {
            void api.DiscardComposition(preview.id);
            setPreview(undefined);
          }
        }}
      >
        <DialogContent resizable={false} tone="warning">
          <DialogHeader>
            <DialogTitle>Blocks overlap</DialogTitle>
            <DialogDescription>Confirm the placement</DialogDescription>
          </DialogHeader>
          <div className="composition-confirm">
            <strong>
              {preview?.conflicts.toLocaleString("en-US")} non-air block
              conflicts
            </strong>
            <p>
              Later placements overwrite earlier blocks at these positions. Air
              never overwrites blocks.
            </p>
            <p>
              {preview?.examples
                .slice(0, 4)
                .map((p) => p.join(", "))
                .join(" · ")}
            </p>
            {error && <FeedbackFlyout title="World composition">{error}</FeedbackFlyout>}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => {
                if (preview) void api.DiscardComposition(preview.id);
                setPreview(undefined);
              }}
            >
              Adjust placements
            </Button>
            <Button
              variant="destructive"
              disabled={busy}
              onClick={() => {
                if (preview) void savePrepared(preview, true);
              }}
            >
              Merge anyway
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

import { useEffect, useRef, useState } from "react";
import {
  Crosshair,
  Minus,
  Plus,
  RotateCcw,
  Map as MapIcon,
  Scan,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import type { Tile, ChunkBounds } from "./bridge";
import { fitTiles } from "./map-camera";
import { selectedChunks, worldAtPointer } from "./map-selection";

export default function WorldMap({
  tiles,
  player,
  hasPlayer,
  selecting = false,
  selection,
  onSelectionChange,
  pickingSpawn = false,
  spawn,
  onSpawnPick,
}: {
  tiles: Map<string, Tile>;
  player: [number, number, number];
  hasPlayer: boolean;
  selecting?: boolean;
  selection?: ChunkBounds;
  onSelectionChange?: (bounds: ChunkBounds) => void;
  pickingSpawn?: boolean;
  spawn?: [number, number, number];
  onSpawnPick?: (x: number, z: number) => void;
}) {
  const selectionStart = useRef<{ x: number; z: number } | null>(null);
  const canvas = useRef<HTMLCanvasElement>(null),
    container = useRef<HTMLDivElement>(null);
  const images = useRef(
      new Map<string, { source: string; image: HTMLImageElement }>(),
    ),
    drag = useRef<{ x: number; y: number } | null>(null);
  const [view, setView] = useState({ x: 0, z: 0, scale: 2 }),
    [size, setSize] = useState({ width: 700, height: 420 }),
    [revision, setRevision] = useState(0),
    [follow, setFollow] = useState(false);
  const fitted = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useEffect(() => {
    const element = container.current;
    if (!element) return;
    const observer = new ResizeObserver((entries) => {
      const box = entries[0]?.contentRect;
      if (box) setSize({ width: box.width, height: box.height });
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    for (const [key, tile] of tiles) {
      if (images.current.get(key)?.source === tile.image) continue;
      const image = new Image();
      image.onload = () => {
        if (mounted.current) setRevision((v) => v + 1);
      };
      image.src = tile.image;
      images.current.set(key, { source: tile.image, image });
    }
    for (const key of images.current.keys())
      if (!tiles.has(key)) images.current.delete(key);
  }, [tiles]);
  useEffect(() => {
    if (follow) {
      const center = hasPlayer
        ? { x: player[0], z: player[2] }
        : fitTiles(tiles.values(), size.width, size.height);
      setView((old) => ({ ...old, ...center }));
    } else if (!fitted.current && tiles.size) {
      fitted.current = true;
      setView(fitTiles(tiles.values(), size.width, size.height));
    }
  }, [follow, hasPlayer, player[0], player[2], tiles]);
  useEffect(() => {
    const element = canvas.current,
      context = element?.getContext("2d");
    if (!element || !context) return;
    const dpr = window.devicePixelRatio || 1;
    element.width = Math.round(size.width * dpr);
    element.height = Math.round(size.height * dpr);
    context.setTransform(dpr, 0, 0, dpr, 0, 0);
    context.clearRect(0, 0, size.width, size.height);
    context.imageSmoothingEnabled = false;
    context.save();
    context.translate(size.width / 2, size.height / 2);
    context.scale(view.scale, view.scale);
    context.translate(-view.x, -view.z);
    for (const [key, tile] of tiles) {
      const image = images.current.get(key)?.image;
      if (image?.complete && image.naturalWidth)
        context.drawImage(image, tile.x * 16, tile.z * 16, 16, 16);
    }
    context.restore();
    if (selection) {
      const x = (selection.minX * 16 - view.x) * view.scale + size.width / 2;
      const y = (selection.minZ * 16 - view.z) * view.scale + size.height / 2;
      const width = (selection.maxX - selection.minX + 1) * 16 * view.scale;
      const height = (selection.maxZ - selection.minZ + 1) * 16 * view.scale;
      context.fillStyle = "rgba(21,209,160,.16)";
      context.fillRect(x, y, width, height);
      context.strokeStyle = "#15D1A0";
      context.lineWidth = 2;
      context.setLineDash([6, 4]);
      context.strokeRect(x, y, width, height);
      context.setLineDash([]);
    }
    if (hasPlayer && tiles.size) {
      const x = (player[0] - view.x) * view.scale + size.width / 2,
        y = (player[2] - view.z) * view.scale + size.height / 2;
      context.beginPath();
      context.arc(x, y, 6, 0, Math.PI * 2);
      context.fillStyle = "#15D1A0";
      context.fill();
      context.strokeStyle = "#FFFFFF";
      context.lineWidth = 2;
      context.stroke();
    }
    if (spawn) {
      const x = (spawn[0] - view.x) * view.scale + size.width / 2,
        y = (spawn[2] - view.z) * view.scale + size.height / 2;
      context.strokeStyle = "#f4c95d";
      context.lineWidth = 3;
      context.strokeRect(x - 5, y - 5, 10, 10);
      context.beginPath();
      context.moveTo(x, y - 5);
      context.lineTo(x, y - 18);
      context.lineTo(x + 11, y - 13);
      context.lineTo(x, y - 10);
      context.stroke();
    }
  }, [tiles, view, size, revision, player, hasPlayer, selection, spawn]);
  function zoom(factor: number) {
    setView((old) => ({
      ...old,
      scale: Math.max(0.001, Math.min(32, old.scale * factor)),
    }));
  }
  return (
    <div className="world-map" ref={container}>
      <canvas
        ref={canvas}
        aria-label="Live map of the captured world"
        className={selecting || pickingSpawn ? "map-selecting" : undefined}
        onAuxClick={(event) => event.preventDefault()}
        onWheel={(event) => {
          event.preventDefault();
          zoom(event.deltaY < 0 ? 1.15 : 1 / 1.15);
        }}
        onPointerDown={(event) => {
          if (event.button !== 0 && event.button !== 1) return;
          event.preventDefault();
          event.currentTarget.setPointerCapture(event.pointerId);
          if (event.button === 1) {
            drag.current = { x: event.clientX, y: event.clientY };
            selectionStart.current = null;
            setFollow(false);
            return;
          }
          if (pickingSpawn) {
            const box = event.currentTarget.getBoundingClientRect();
            const point = worldAtPointer(
              event.clientX - box.left,
              event.clientY - box.top,
              box.width,
              box.height,
              view,
            );
            onSpawnPick?.(Math.floor(point.x), Math.floor(point.z));
            return;
          }
          if (selecting) {
            const box = event.currentTarget.getBoundingClientRect();
            const point = worldAtPointer(
              event.clientX - box.left,
              event.clientY - box.top,
              box.width,
              box.height,
              view,
            );
            selectionStart.current = point;
            onSelectionChange?.(selectedChunks(point, point));
            setFollow(false);
            return;
          }
          drag.current = { x: event.clientX, y: event.clientY };
          setFollow(false);
        }}
        onPointerMove={(event) => {
          if (selectionStart.current) {
            const box = event.currentTarget.getBoundingClientRect();
            const point = worldAtPointer(
              event.clientX - box.left,
              event.clientY - box.top,
              box.width,
              box.height,
              view,
            );
            onSelectionChange?.(selectedChunks(selectionStart.current, point));
            return;
          }
          if (!drag.current) return;
          const dx = event.clientX - drag.current.x,
            dz = event.clientY - drag.current.y;
          drag.current = { x: event.clientX, y: event.clientY };
          setView((old) => ({
            ...old,
            x: old.x - dx / old.scale,
            z: old.z - dz / old.scale,
          }));
        }}
        onPointerUp={() => {
          drag.current = null;
          selectionStart.current = null;
        }}
        onPointerCancel={() => {
          drag.current = null;
          selectionStart.current = null;
        }}
      />
      {!tiles.size && (
        <div className="map-empty">
          <MapIcon />
          <h3>Your world takes shape here</h3>
          <p>
            Explore in Minecraft. Received chunks appear live on the map.
          </p>
        </div>
      )}
      <div className="map-label">
        <span className="live-dot" />
        Live map <span>{tiles.size} Chunks</span>
      </div>
      <div className="map-controls">
        <Button
          variant="outline"
          size="icon"
          aria-label="Show all terrain"
          title="Show all terrain"
          onClick={() => {
            setFollow(false);
            setView(fitTiles(tiles.values(), size.width, size.height));
          }}
        >
          <Scan />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Zoom in on map"
          onClick={() => zoom(1.4)}
        >
          <Plus />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Zoom out on map"
          onClick={() => zoom(1 / 1.4)}
        >
          <Minus />
        </Button>
        <Button
          variant={follow ? "default" : "outline"}
          size="icon"
          aria-label="Follow player"
          aria-pressed={follow}
          onClick={() => setFollow((v) => !v)}
        >
          <Crosshair />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Reset map view"
          onClick={() => {
            setView({ x: player[0], z: player[2], scale: 2 });
            setFollow(true);
          }}
        >
          <RotateCcw />
        </Button>
      </div>
      <div className="map-coordinate">
        X {Math.round(player[0])} <span>Y {Math.round(player[1])}</span> Z{" "}
        {Math.round(player[2])}
      </div>
    </div>
  );
}

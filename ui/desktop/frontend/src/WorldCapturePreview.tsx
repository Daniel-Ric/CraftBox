import { useEffect, useRef, useState } from "react";
import { LoaderCircle, Map as MapIcon } from "lucide-react";
import type { Tile, World } from "./bridge";
import { useMapTiles } from "./use-map-tiles";

const noTiles = new Map<string, Tile>();

// Reuse the map's low-resolution terrain tiles, including incremental updates
// for active captures. This does not open a 3D renderer or read arbitrary files.
export default function WorldCapturePreview({ world }: { world: World }) {
  const { tiles, loading, error } = useMapTiles(
    world.id,
    noTiles,
    undefined,
    undefined,
    1,
  );
  const canvas = useRef<HTMLCanvasElement>(null);
  const [ready, setReady] = useState(false);
  const images = useRef(
    new Map<string, { source: string; image: HTMLImageElement }>(),
  );
  useEffect(() => {
    let active = true,
      frame = 0;
    function draw() {
      frame = 0;
      const el = canvas.current,
        ctx = el?.getContext("2d");
      if (!active || !el || !ctx) return;
      ctx.clearRect(0, 0, el.width, el.height);
      if (!tiles.size) {
        setReady(false);
        return;
      }
      let minX = Infinity,
        minZ = Infinity,
        maxX = -Infinity,
        maxZ = -Infinity;
      for (const tile of tiles.values()) {
        minX = Math.min(minX, tile.x);
        minZ = Math.min(minZ, tile.z);
        maxX = Math.max(maxX, tile.x);
        maxZ = Math.max(maxZ, tile.z);
      }
      const spanX = maxX - minX + 1,
        spanZ = maxZ - minZ + 1;
      const size = Math.min((el.width - 24) / spanX, (el.height - 24) / spanZ);
      const x = (el.width - spanX * size) / 2,
        z = (el.height - spanZ * size) / 2;
      ctx.imageSmoothingEnabled = false;
      let painted = false;
      for (const [key, tile] of tiles) {
        const img = images.current.get(key)?.image;
        if (img?.complete && img.naturalWidth) {
          ctx.drawImage(
            img,
            x + (tile.x - minX) * size,
            z + (tile.z - minZ) * size,
            size,
            size,
          );
          painted = true;
        }
      }
      setReady(painted);
    }
    function schedule() {
      if (active && !frame) frame = requestAnimationFrame(draw);
    }
    for (const [key, tile] of tiles) {
      const cached = images.current.get(key);
      if (cached?.source === tile.image) {
        cached.image.onload = schedule;
        cached.image.onerror = schedule;
        continue;
      }
      const image = new Image();
      image.onload = schedule;
      image.onerror = schedule;
      images.current.set(key, { source: tile.image, image });
      image.src = tile.image;
    }
    for (const key of images.current.keys())
      if (!tiles.has(key)) images.current.delete(key);
    schedule();
    return () => {
      active = false;
      cancelAnimationFrame(frame);
    };
  }, [tiles]);
  return (
    <div className="world-capture-preview">
      <canvas
        ref={canvas}
        width={480}
        height={280}
        role="img"
        aria-label={`Terrain preview of ${world.name}`}
        hidden={!ready}
      />
      {!ready && (
        <div className="world-preview-status" role="status">
          {loading ? (
            <LoaderCircle className="spin" size={24} />
          ) : (
            <MapIcon size={28} />
          )}
          <span>
            {loading
              ? "Loading terrain preview..."
              : error
                ? "Terrain preview unavailable"
                : "No terrain preview yet"}
          </span>
        </div>
      )}
    </div>
  );
}

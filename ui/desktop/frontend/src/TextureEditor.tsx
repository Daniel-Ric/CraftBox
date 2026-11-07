import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Undo2, Redo2, Pencil, Eraser, Pipette } from "lucide-react";

export default function TextureEditor({
  image,
  onSave,
  busy,
  initialDraft,
  onDraftChange,
}: {
  image: string;
  onSave: (data: string) => void;
  busy: boolean;
  initialDraft?: string;
  onDraftChange?: (png: string) => void;
}) {
  const draft = useRef(initialDraft);
  draft.current = initialDraft;
  const [ready, setReady] = useState(false);
  const canvas = useRef<HTMLCanvasElement>(null),
    history = useRef<ImageData[]>([]),
    future = useRef<ImageData[]>([]),
    drawing = useRef(false),
    last = useRef<[number, number] | null>(null);
  const [color, setColor] = useState("#387fff"),
    [tool, setTool] = useState("pencil"),
    [size, setSize] = useState(1),
    [version, setVersion] = useState(0),
    [error, setError] = useState("");
  useEffect(() => {
    let cancelled = false;
    setReady(false);
    const img = new Image();
    img.onload = () => {
      if (cancelled || !canvas.current) return;
      if (img.width * img.height > 4 * 1024 * 1024) {
        setError(
          "Edit textures up to 4 megapixels. Export larger textures to an image editor.",
        );
        return;
      }
      const c = canvas.current;
      c.width = img.width;
      c.height = img.height;
      c.getContext("2d")!.drawImage(img, 0, 0);
      history.current = [];
      future.current = [];
      setVersion(draft.current ? 1 : 0);
      setReady(true);
      setError("");
    };
    img.onerror = () => setError("Texture could not be opened.");
    img.src = draft.current || image;
    return () => {
      cancelled = true;
    };
  }, [image]);
  const snapshot = () => {
    const c = canvas.current!;
    return c.getContext("2d")!.getImageData(0, 0, c.width, c.height);
  };
  function paint(e: React.PointerEvent<HTMLCanvasElement>) {
    const c = canvas.current!,
      rect = c.getBoundingClientRect(),
      x = Math.min(
        c.width - 1,
        Math.max(
          0,
          Math.floor(((e.clientX - rect.left) * c.width) / rect.width),
        ),
      ),
      y = Math.min(
        c.height - 1,
        Math.max(
          0,
          Math.floor(((e.clientY - rect.top) * c.height) / rect.height),
        ),
      ),
      ctx = c.getContext("2d")!;
    if (tool === "picker") {
      const pixel = ctx.getImageData(x, y, 1, 1).data;
      setColor(
        "#" +
          [pixel[0], pixel[1], pixel[2]]
            .map((n) => n.toString(16).padStart(2, "0"))
            .join(""),
      );
      return;
    }
    const [px, py] = last.current ?? [x, y],
      steps = Math.max(Math.abs(x - px), Math.abs(y - py), 1);
    for (let i = 0; i <= steps; i++) {
      const nx = Math.round(px + ((x - px) * i) / steps),
        ny = Math.round(py + ((y - py) * i) / steps);
      if (tool === "eraser") ctx.clearRect(nx, ny, size, size);
      else {
        ctx.fillStyle = color;
        ctx.fillRect(nx, ny, size, size);
      }
    }
    last.current = [x, y];
  }
  function restore(
    from: React.RefObject<ImageData[]>,
    to: React.RefObject<ImageData[]>,
  ) {
    const data = from.current.pop();
    if (data) {
      to.current.push(snapshot());
      canvas.current!.getContext("2d")!.putImageData(data, 0, 0);
      setVersion((v) => v + 1);
      onDraftChange?.(canvas.current!.toDataURL("image/png"));
    }
  }
  return (
    <div className="asset-texture-editor">
      <div className="asset-tools">
        {[
          [Pencil, "pencil", "Pencil"],
          [Eraser, "eraser", "Eraser"],
          [Pipette, "picker", "Pick colour"],
        ].map(([Icon, value, label]) => {
          const I = Icon as typeof Pencil;
          return (
            <Button
              key={value as string}
              variant={tool === value ? "secondary" : "ghost"}
              aria-label={label as string}
              aria-pressed={tool === value}
              onClick={() => setTool(value as string)}
            >
              <I />
            </Button>
          );
        })}
        <input
          aria-label="Brush colour"
          type="color"
          value={color}
          onChange={(e) => setColor(e.target.value)}
        />
        <label>
          Brush
          <select value={size} onChange={(e) => setSize(+e.target.value)}>
            {[1, 2, 4, 8, 16].map((n) => (
              <option key={n} value={n}>
                {n} px
              </option>
            ))}
          </select>
        </label>
        <Button
          variant="ghost"
          aria-label="Undo edit"
          disabled={!history.current.length}
          onClick={() => restore(history, future)}
        >
          <Undo2 />
        </Button>
        <Button
          variant="ghost"
          aria-label="Redo edit"
          disabled={!future.current.length}
          onClick={() => restore(future, history)}
        >
          <Redo2 />
        </Button>
      </div>
      <div className="asset-texture-canvas">
        <canvas
          ref={canvas}
          onPointerDown={(e) => {
            if (busy || error || !ready) return;
            const maxHistory = Math.max(
              1,
              Math.min(
                32,
                Math.floor(
                  (32 << 20) /
                    (canvas.current!.width * canvas.current!.height * 4),
                ),
              ),
            );
            history.current.push(snapshot());
            if (history.current.length > maxHistory) history.current.shift();
            future.current = [];
            drawing.current = true;
            last.current = null;
            e.currentTarget.setPointerCapture(e.pointerId);
            paint(e);
            setVersion((v) => v + 1);
            onDraftChange?.(canvas.current!.toDataURL("image/png"));
          }}
          onPointerMove={(e) => {
            if (drawing.current) paint(e);
          }}
          onPointerUp={() => {
            drawing.current = false;
            last.current = null;
            if (ready) onDraftChange?.(canvas.current!.toDataURL("image/png"));
          }}
          onPointerCancel={() => {
            drawing.current = false;
            last.current = null;
            if (ready) onDraftChange?.(canvas.current!.toDataURL("image/png"));
          }}
        />
      </div>
      {error && <FeedbackFlyout title="Texture editor">{error}</FeedbackFlyout>}
      <div className="asset-tools">
        <p>
          Edits are saved as a new library item. The original stays available.
        </p>
        <Button
          disabled={busy || !ready || !version || !!error}
          onClick={() => onSave(canvas.current!.toDataURL("image/png"))}
        >
          Save edited copy
        </Button>
      </div>
    </div>
  );
}

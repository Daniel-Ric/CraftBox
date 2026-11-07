import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";

export type TextureViewportHandle = { reset: () => void };
export default forwardRef<TextureViewportHandle, { src: string; name: string }>(
  function TextureViewport({ src, name }, ref) {
    const host = useRef<HTMLDivElement>(null),
      image = useRef<HTMLImageElement>(null);
    const [view, setView] = useState({ x: 0, y: 0, scale: 1 });
    const drag = useRef<{ x: number; y: number; id: number } | undefined>(
      undefined,
    );
    function fit() {
      const h = host.current,
        i = image.current;
      if (!h || !i?.naturalWidth) return;
      setView({
        x: 0,
        y: 0,
        scale: Math.max(
          0.02,
          Math.min(
            (h.clientWidth - 64) / i.naturalWidth,
            (h.clientHeight - 100) / i.naturalHeight,
            16,
          ),
        ),
      });
    }
    useImperativeHandle(ref, () => ({ reset: fit }));
    useEffect(() => {
      setView({ x: 0, y: 0, scale: 1 });
    }, [src]);
    useEffect(() => {
      const h = host.current;
      if (!h) return;
      const wheel = (event: WheelEvent) => {
        event.preventDefault();
        const rect = h.getBoundingClientRect(),
          x = event.clientX - rect.left - rect.width / 2,
          y = event.clientY - rect.top - rect.height / 2;
        setView((old) => {
          const scale = Math.max(
              0.02,
              Math.min(128, old.scale * Math.exp(-event.deltaY * 0.0015)),
            ),
            ratio = scale / old.scale;
          return {
            x: x - (x - old.x) * ratio,
            y: y - (y - old.y) * ratio,
            scale,
          };
        });
      };
      h.addEventListener("wheel", wheel, { passive: false });
      return () => h.removeEventListener("wheel", wheel);
    }, []);
    const zoom = (factor: number) =>
      setView((old) => ({
        ...old,
        scale: Math.max(0.02, Math.min(128, old.scale * factor)),
      }));
    return (
      <div
        ref={host}
        className="texture-viewport"
        role="region"
        aria-label={`Texture preview: ${name}`}
        tabIndex={0}
        onContextMenu={(e) => e.preventDefault()}
        onPointerDown={(e) => {
          if ((e.target as HTMLElement).closest("button")) return;
          e.currentTarget.setPointerCapture(e.pointerId);
          drag.current = { x: e.clientX, y: e.clientY, id: e.pointerId };
        }}
        onPointerMove={(e) => {
          const d = drag.current;
          if (!d || d.id !== e.pointerId) return;
          const dx = e.clientX - d.x,
            dy = e.clientY - d.y;
          d.x = e.clientX;
          d.y = e.clientY;
          setView((old) => ({ ...old, x: old.x + dx, y: old.y + dy }));
        }}
        onPointerUp={() => {
          drag.current = undefined;
        }}
        onPointerCancel={() => {
          drag.current = undefined;
        }}
        onKeyDown={(e) => {
          if (e.key === "+" || e.key === "=") {
            e.preventDefault();
            zoom(1.25);
          } else if (e.key === "-") {
            e.preventDefault();
            zoom(0.8);
          } else if (e.key === "0") {
            e.preventDefault();
            fit();
          }
        }}
      >
        <img
          ref={image}
          src={src}
          alt={name}
          draggable={false}
          onLoad={fit}
          style={{
            transform: `translate(-50%, -50%) translate(${view.x}px, ${view.y}px) scale(${view.scale})`,
          }}
        />
        <div className="texture-viewport-tools">
          <Button
            variant="outline"
            onClick={() => zoom(0.8)}
            aria-label="Zoom out"
          >
            −
          </Button>
          <span>{Math.round(view.scale * 100)}%</span>
          <Button
            variant="outline"
            onClick={() => zoom(1.25)}
            aria-label="Zoom in"
          >
            +
          </Button>
          <Button
            variant="outline"
            onClick={() => setView({ x: 0, y: 0, scale: 1 })}
          >
            100%
          </Button>
          <Button variant="outline" onClick={fit}>
            Fit view
          </Button>
        </div>
      </div>
    );
  },
);

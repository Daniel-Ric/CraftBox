import { useRef } from "react";
export default function PanelResizeHandle(p: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
}) {
  const drag = useRef<{ y: number; height: number } | null>(null);
  const change = (height: number) =>
    p.onChange(Math.max(p.min, Math.min(p.max, Math.round(height))));
  return (
    <div
      className="panel-resize-handle"
      role="separator"
      tabIndex={0}
      aria-label={p.label}
      aria-orientation="horizontal"
      aria-valuenow={p.value}
      aria-valuemin={p.min}
      aria-valuemax={p.max}
      onPointerDown={(e) => {
        if (e.button !== 0) return;
        e.preventDefault();
        drag.current = { y: e.clientY, height: p.value };
        e.currentTarget.setPointerCapture(e.pointerId);
      }}
      onPointerMove={(e) => {
        if (drag.current)
          change(drag.current.height + drag.current.y - e.clientY);
      }}
      onPointerUp={(e) => {
        drag.current = null;
        if (e.currentTarget.hasPointerCapture(e.pointerId))
          e.currentTarget.releasePointerCapture(e.pointerId);
      }}
      onLostPointerCapture={() => {
        drag.current = null;
      }}
      onKeyDown={(e) => {
        if (["ArrowUp", "ArrowDown", "Home", "End"].includes(e.key)) {
          e.preventDefault();
          change(
            e.key === "Home"
              ? p.min
              : e.key === "End"
                ? p.max
                : p.value + (e.key === "ArrowUp" ? 20 : -20),
          );
        }
      }}
    />
  );
}

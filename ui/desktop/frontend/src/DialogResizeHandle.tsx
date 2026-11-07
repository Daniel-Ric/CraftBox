import { useEffect, useRef, type RefObject } from "react";
import "./dialog-resize.css";
export default function DialogResizeHandle({
  dialog,
}: {
  dialog: RefObject<HTMLElement | null>;
}) {
  const minimum = useRef<{ width: number; height: number } | null>(null),
    resized = useRef(false);
  const measure = () => {
    const node = dialog.current;
    if (!node) return null;
    const rect = node.getBoundingClientRect(),
      scale = node.offsetWidth
        ? rect.width / node.offsetWidth
        : parseFloat(getComputedStyle(document.documentElement).zoom) || 1;
    return {
      node,
      rect,
      scale,
      width: node.offsetWidth || rect.width / scale,
      height: node.offsetHeight || rect.height / scale,
    };
  };
  const size = (width: number, height: number) => {
    const m = measure();
    if (!m) return;
    minimum.current ??= { width: m.width, height: m.height };
    const maxW = (innerWidth - 16) / m.scale,
      maxH = (innerHeight - 16) / m.scale;
    m.node.style.width = `${Math.min(maxW, Math.max(minimum.current.width, width))}px`;
    m.node.style.height = `${Math.min(maxH, Math.max(minimum.current.height, height))}px`;
    m.node.style.maxWidth = `${maxW}px`;
    m.node.style.maxHeight = `${maxH}px`;
    const r = m.node.getBoundingClientRect();
    const dx =
      r.left < 8
        ? 8 - r.left
        : r.right > innerWidth - 8
          ? innerWidth - 8 - r.right
          : 0;
    const dy =
      r.top < 8
        ? 8 - r.top
        : r.bottom > innerHeight - 8
          ? innerHeight - 8 - r.bottom
          : 0;
    const x = (Number(m.node.dataset.dragX) || 0) + dx / m.scale,
      y = (Number(m.node.dataset.dragY) || 0) + dy / m.scale;
    m.node.dataset.dragX = String(x);
    m.node.dataset.dragY = String(y);
    m.node.style.translate = `${x}px ${y}px`;
    resized.current = true;
  };
  useEffect(() => {
    const update = () => {
      const m = measure();
      if (m && !resized.current && m.width > 0 && m.height > 0)
        minimum.current = { width: m.width, height: m.height };
    };
    update();
    const node = dialog.current;
    const observer =
      typeof ResizeObserver === "undefined"
        ? undefined
        : new ResizeObserver(update);
    if (node) observer?.observe(node);
    const fit = () => {
      const m = measure();
      if (m && resized.current) size(m.width, m.height);
    };
    window.addEventListener("resize", fit);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", fit);
    };
  }, [dialog]);
  const drag = useRef<{
    pointer: number;
    x: number;
    y: number;
    width: number;
    height: number;
    scale: number;
  } | null>(null);
  return (
    <button
      type="button"
      className="dialog-resize-handle"
      aria-label="Resize dialog"
      title="Resize dialog · Arrow keys resize · Home restores default size"
      onPointerDown={(event) => {
        if (event.button !== 0) return;
        const m = measure();
        if (!m) return;
        minimum.current ??= { width: m.width, height: m.height };
        drag.current = {
          pointer: event.pointerId,
          x: event.clientX,
          y: event.clientY,
          width: m.width,
          height: m.height,
          scale: m.scale,
        };
        event.currentTarget.setPointerCapture(event.pointerId);
        event.preventDefault();
        event.stopPropagation();
      }}
      onPointerMove={(event) => {
        const start = drag.current;
        if (!start || start.pointer !== event.pointerId) return;
        size(
          start.width + (2 * (event.clientX - start.x)) / start.scale,
          start.height + (2 * (event.clientY - start.y)) / start.scale,
        );
        event.stopPropagation();
      }}
      onPointerUp={() => {
        drag.current = null;
      }}
      onPointerCancel={() => {
        drag.current = null;
      }}
      onLostPointerCapture={() => {
        drag.current = null;
      }}
      onKeyDown={(event) => {
        const m = measure();
        if (!m) return;
        const step = event.shiftKey ? 40 : 10;
        minimum.current ??= { width: m.width, height: m.height };
        if (event.key === "Home")
          size(minimum.current.width, minimum.current.height);
        else if (event.key === "ArrowRight") size(m.width + step, m.height);
        else if (event.key === "ArrowLeft") size(m.width - step, m.height);
        else if (event.key === "ArrowDown") size(m.width, m.height + step);
        else if (event.key === "ArrowUp") size(m.width, m.height - step);
        else return;
        event.preventDefault();
        event.stopPropagation();
      }}
    >
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        <path
          d="M4 10 10 4M8 10l2-2"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.3"
        />
      </svg>
    </button>
  );
}

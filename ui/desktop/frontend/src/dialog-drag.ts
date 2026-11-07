import { useEffect, useRef, type PointerEvent, type MouseEvent } from "react";

const interactive =
  "button,input,select,textarea,a,[role=button],[contenteditable=true]";
function dragHandle(target: EventTarget, dialog: HTMLElement) {
  if (!(target instanceof Element) || target.closest(interactive)) return null;
  const header = target.closest(
    '[data-slot="dialog-header"],.studio-window-titlebar',
  );
  return header instanceof HTMLElement &&
    header.closest('[data-slot="dialog-content"]') === dialog
    ? header
    : null;
}
function position(dialog: HTMLElement, x: number, y: number) {
  dialog.dataset.dragX = String(x);
  dialog.dataset.dragY = String(y);
  dialog.style.translate = `${x}px ${y}px`;
}
function keepVisible(
  dialog: HTMLElement,
  left: number,
  top: number,
  rect: DOMRect,
  baseX: number,
  baseY: number,
) {
  const x = Math.max(
    8,
    Math.min(Math.max(8, window.innerWidth - rect.width - 8), left),
  );
  const y = Math.max(
    8,
    Math.min(Math.max(8, window.innerHeight - rect.height - 8), top),
  );
  const scale = dialog.offsetWidth > 0 ? rect.width / dialog.offsetWidth : 1;
  position(
    dialog,
    baseX + (x - rect.left) / scale,
    baseY + (y - rect.top) / scale,
  );
}
export function useDialogDrag() {
  const ref = useRef<HTMLElement | null>(null);
  const drag = useRef<{
    pointer: number;
    x: number;
    y: number;
    baseX: number;
    baseY: number;
    rect: DOMRect;
    capture: HTMLElement;
    dialog: HTMLElement;
  } | null>(null);
  useEffect(() => {
    const resize = () => {
      const dialog = ref.current;
      if (!dialog || !dialog.style.translate) return;
      const rect = dialog.getBoundingClientRect();
      keepVisible(
        dialog,
        rect.left,
        rect.top,
        rect,
        Number(dialog.dataset.dragX) || 0,
        Number(dialog.dataset.dragY) || 0,
      );
    };
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, []);
  const finish = (e: { pointerId: number }) => {
    const current = drag.current;
    if (!current || current.pointer !== e.pointerId) return;
    drag.current = null;
    delete current.dialog.dataset.dragging;
    if (current.capture.hasPointerCapture?.(e.pointerId))
      current.capture.releasePointerCapture(e.pointerId);
  };
  const move = (e: { pointerId: number; clientX: number; clientY: number }) => {
    const start = drag.current;
    if (!start || start.pointer !== e.pointerId) return;
    keepVisible(
      start.dialog,
      start.rect.left + e.clientX - start.x,
      start.rect.top + e.clientY - start.y,
      start.rect,
      start.baseX,
      start.baseY,
    );
  };
  useEffect(() => {
                                                                                 
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", finish);
    window.addEventListener("pointercancel", finish);
    return () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", finish);
      window.removeEventListener("pointercancel", finish);
      drag.current = null;
    };
  }, []);
  return {
    ref,
    onPointerDown(e: PointerEvent<HTMLElement>) {
      if (document.documentElement.dataset.dialogDragging === "false") return;
      const capture = dragHandle(e.target, e.currentTarget);
      if (e.button !== 0 || !capture || drag.current) return;
      const dialog = e.currentTarget;
      drag.current = {
        pointer: e.pointerId,
        x: e.clientX,
        y: e.clientY,
        baseX: Number(dialog.dataset.dragX) || 0,
        baseY: Number(dialog.dataset.dragY) || 0,
        rect: dialog.getBoundingClientRect(),
        capture,
        dialog,
      };
      dialog.dataset.dragging = "";
      capture.setPointerCapture?.(e.pointerId);
      e.preventDefault();
    },
    onPointerMove: move,
    onPointerUp: finish,
    onPointerCancel: finish,
    onLostPointerCapture: finish,
    onDoubleClick(e: MouseEvent<HTMLElement>) {
      if (document.documentElement.dataset.dialogDragging === "false") return;
      if (!dragHandle(e.target, e.currentTarget)) return;
      e.currentTarget.style.removeProperty("translate");
      delete e.currentTarget.dataset.dragX;
      delete e.currentTarget.dataset.dragY;
    },
  };
}

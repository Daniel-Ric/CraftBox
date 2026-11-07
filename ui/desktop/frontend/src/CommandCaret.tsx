import { useLayoutEffect, useState, type RefObject } from "react";
export default function CommandCaret({
  input,
  root,
  focused,
  value,
}: {
  input: RefObject<HTMLInputElement | null>;
  root: RefObject<HTMLDivElement | null>;
  focused: boolean;
  value: string;
}) {
  const [caret, setCaret] = useState<{ left: number; top: number } | null>(
    null,
  );
  useLayoutEffect(() => {
    const node = input.current,
      container = root.current;
    if (!node || !container) return;
    let composing = false;
    const mirror = document.createElement("span");
    Object.assign(mirror.style, {
      position: "fixed",
      visibility: "hidden",
      whiteSpace: "pre",
      pointerEvents: "none",
    });
    document.body.appendChild(mirror);
    const update = () => {
      if (!focused || composing || node.selectionStart !== node.selectionEnd) {
        setCaret(null);
        delete container.dataset.customCaret;
        return;
      }
      const style = getComputedStyle(node),
        box = node.getBoundingClientRect(),
        base = container.getBoundingClientRect();
      const scale = container.offsetWidth
        ? base.width / container.offsetWidth
        : 1;
      mirror.style.font = style.font;
      mirror.style.letterSpacing = style.letterSpacing;
      mirror.textContent = node.value.slice(0, node.selectionStart ?? 0);
      const left =
        (box.left - base.left) / scale +
        (parseFloat(style.paddingLeft) || 0) +
        (parseFloat(style.borderLeftWidth) || 0) +
        mirror.getBoundingClientRect().width / scale -
        node.scrollLeft;
      const top = (box.top - base.top) / scale + (box.height / scale - 14) / 2;
      if (
        left < (box.left - base.left) / scale ||
        left > (box.right - base.left) / scale
      ) {
        setCaret(null);
        delete container.dataset.customCaret;
        return;
      }
      container.dataset.customCaret = "true";
      setCaret((old) =>
        old?.left === left && old?.top === top ? old : { left, top },
      );
    };
    const start = () => {
        composing = true;
        update();
      },
      end = () => {
        composing = false;
        update();
      };
    update();
    const events = ["input", "select", "keyup", "scroll", "click"];
    events.forEach((name) => node.addEventListener(name, update));
    node.addEventListener("compositionstart", start);
    node.addEventListener("compositionend", end);
    document.addEventListener("selectionchange", update);
    window.addEventListener("resize", update);
    const resize =
      typeof ResizeObserver === "undefined"
        ? undefined
        : new ResizeObserver(update);
    resize?.observe(node);
    return () => {
      mirror.remove();
      resize?.disconnect();
      events.forEach((name) => node.removeEventListener(name, update));
      node.removeEventListener("compositionstart", start);
      node.removeEventListener("compositionend", end);
      document.removeEventListener("selectionchange", update);
      window.removeEventListener("resize", update);
      delete container.dataset.customCaret;
    };
  }, [input, root, focused, value]);
  return (
    caret && (
      <span
        key={`${caret.left}:${caret.top}`}
        className="command-caret"
        style={caret}
        aria-hidden="true"
      />
    )
  );
}

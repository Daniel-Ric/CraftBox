import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {loadWorkspacePreferences} from "./workspace-preferences";

                                                                                
export default function StudioTooltipLayer() {
  const id = useId(),
    node = useRef<HTMLDivElement>(null);
  const [tip, setTip] = useState<{ anchor: HTMLElement; text: string }>();
  const [position, setPosition] = useState({ left: 8, top: 8 });
  useEffect(() => {
    let anchor: HTMLElement | undefined,
      timer: ReturnType<typeof setTimeout> | undefined,
      keyboard = false;
    const hide = () => {
      clearTimeout(timer);
      if (anchor) {
        const tokens = (anchor.getAttribute("aria-describedby") ?? "")
          .split(/\s+/)
          .filter((v) => v && v !== id);
        if (tokens.length)
          anchor.setAttribute("aria-describedby", tokens.join(" "));
        else anchor.removeAttribute("aria-describedby");
      }
      anchor = undefined;
      setTip(undefined);
    };
    const schedule = (target: EventTarget | null) => {
      const prefs=loadWorkspacePreferences();
      if(!prefs.tooltips){hide();return;}
      const next =
        target instanceof Element
          ? target.closest<HTMLElement>("[data-tooltip]")
          : null;
      if (!next || next === anchor) return;
      hide();
      anchor = next;
      timer = setTimeout(() => {
        if (!next.isConnected || document.pointerLockElement) return;
        const text = next.dataset.tooltip?.trim();
        if (!text) return;
        const tokens = new Set(
          (next.getAttribute("aria-describedby") ?? "")
            .split(/\s+/)
            .filter(Boolean),
        );
        tokens.add(id);
        next.setAttribute("aria-describedby", [...tokens].join(" "));
        setTip({ anchor: next, text });
      }, prefs.tooltipDelay);
    };
    const over = (e: PointerEvent) => schedule(e.target);
    const out = (e: PointerEvent) => {
      if (
        anchor &&
        !(e.relatedTarget instanceof Node && anchor.contains(e.relatedTarget))
      )
        hide();
    };
    const focus = (e: FocusEvent) => {
      if (keyboard) schedule(e.target);
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") hide();
      else if (
        ["Tab", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"].includes(
          e.key,
        )
      )
        keyboard = true;
    };
    const down = () => {
      keyboard = false;
      hide();
    };
    document.addEventListener("pointerover", over);
    document.addEventListener("pointerout", out);
    document.addEventListener("focusin", focus);
    document.addEventListener("blur", hide, true);
    document.addEventListener("keydown", key, true);
    document.addEventListener("pointerdown", down, true);
    document.addEventListener("scroll", hide, true);
    window.addEventListener("resize", hide);
    return () => {
      hide();
      document.removeEventListener("pointerover", over);
      document.removeEventListener("pointerout", out);
      document.removeEventListener("focusin", focus);
      document.removeEventListener("blur", hide, true);
      document.removeEventListener("keydown", key, true);
      document.removeEventListener("pointerdown", down, true);
      document.removeEventListener("scroll", hide, true);
      window.removeEventListener("resize", hide);
    };
  }, [id]);
  useLayoutEffect(() => {
    if (!tip || !node.current) return;
    const r = tip.anchor.getBoundingClientRect(),
      t = node.current.getBoundingClientRect();
    const left = Math.max(8, Math.min(r.left, window.innerWidth - t.width - 8));
    const top =
      r.bottom + 8 + t.height < window.innerHeight - 8
        ? r.bottom + 8
        : Math.max(8, r.top - t.height - 8);
    setPosition({ left, top });
  }, [tip]);
  return tip
    ? createPortal(
        <div
          ref={node}
          id={id}
          role="tooltip"
          className="studio-tooltip"
          style={position}
        >
          {tip.text}
        </div>,
        document.body,
      )
    : null;
}

import { useEffect } from "react";
import "./studio-ranges.css";

// Native sliders retain keyboard, touch and accessibility behavior. Update only
// their visual fill, including controlled values changed by presets or playback.
export default function StudioRangeLayer() {
  useEffect(() => {
    const update = (input: HTMLInputElement) => {
      if (input.type !== "range") return;
      const min = input.min === "" ? 0 : Number(input.min);
      const max = input.max === "" ? 100 : Number(input.max);
      const fraction = max > min ? (Number(input.value) - min) / (max - min) : 0;
      input.style.setProperty("--range-progress", `${Math.max(0, Math.min(1, fraction)) * 100}%`);
    };
    const scan = (node: Node) => {
      if (node instanceof HTMLInputElement) update(node);
      if (node instanceof Element) node.querySelectorAll<HTMLInputElement>('input[type="range"]').forEach(update);
    };
    scan(document.documentElement);
    const changed = (event: Event) => { if (event.target instanceof HTMLInputElement) update(event.target); };
    const observer = new MutationObserver(records => records.forEach(record => {
      if (record.type === "attributes") scan(record.target);
      else record.addedNodes.forEach(scan);
    }));
    observer.observe(document.documentElement, {subtree: true, childList: true, attributes: true, attributeFilter: ["value", "min", "max", "type"]});
    document.addEventListener("input", changed, true);
    document.addEventListener("change", changed, true);
    const reset = () => requestAnimationFrame(() => scan(document.documentElement));
    document.addEventListener("reset", reset, true);
    return () => {
      observer.disconnect();
      document.removeEventListener("input", changed, true);
      document.removeEventListener("change", changed, true);
      document.removeEventListener("reset", reset, true);
    };
  }, []);
  return null;
}

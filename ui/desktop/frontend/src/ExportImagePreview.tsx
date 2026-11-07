import { useCallback, useRef } from "react";
import { ZoomIn } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "./components/ui/dialog";
import "./export-image-preview.css";

export default function ExportImagePreview({ src, alt }: { src: string; alt: string }) {
  const origin = useRef<DOMRect | null>(null);
  const animateOpen = useCallback((node: HTMLDivElement | null) => {
    if (!node || !origin.current || typeof node.animate !== "function") return;
    if (document.documentElement.dataset.motion === "reduced" || window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return;
    const from = origin.current, to = node.getBoundingClientRect();
    if (!to.width || !to.height) return;
    const zoom = to.width / node.offsetWidth || 1;
    const x = (from.x + from.width / 2 - to.x - to.width / 2) / zoom;
    const y = (from.y + from.height / 2 - to.y - to.height / 2) / zoom;
    const scale = Math.min(from.width / to.width, from.height / to.height);
    const animation = node.animate([
      { transform: `translate(-50%, -50%) translate(${x}px, ${y}px) scale(${scale})`, opacity: 0.4 },
      { transform: "translate(-50%, -50%) translate(0, 0) scale(1)", opacity: 1 },
    ], { duration: 300, easing: "cubic-bezier(0.2, 0.8, 0.2, 1)" });
    return () => animation.cancel();
  }, []);
  return <Dialog>
    <DialogTrigger className="export-studio-preview export-image-trigger" aria-label="Enlarge image preview"
      onClick={event => { origin.current = event.currentTarget.getBoundingClientRect(); }}>
      <img src={src} alt={alt} />
      <span className="export-image-zoom" aria-hidden="true"><ZoomIn size={16} /></span>
    </DialogTrigger>
    <DialogContent className="export-image-dialog" ref={animateOpen}>
      <DialogHeader>
        <DialogTitle>Image preview</DialogTitle>
        <DialogDescription className="sr-only">Enlarged export preview. Press Escape to return to export settings.</DialogDescription>
      </DialogHeader>
      <div className="export-image-stage"><img src={src} alt={alt} /></div>
    </DialogContent>
  </Dialog>;
}

import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useState } from "react";
import { ArrowDown, ArrowUp, Layers } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { api, type LibraryAsset } from "./bridge";
import "./asset-studio.css";

export default function RenderPackDialog({
  worldID,
  onApplied,
}: {
  worldID: string;
  onApplied: () => void;
}) {
  const [open, setOpen] = useState(false),
    [items, setItems] = useState<LibraryAsset[]>([]),
    [ids, setIDs] = useState<string[]>([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    if (!open) return;
    let dead = false;
    setError("");
    void api
      .GetAssetLibraryForKind("packs")
      .then((result) => {
        if (dead) return;
        const packs = result.items.filter((v) => v.kind === "packs");
        setItems(packs);
        try {
          const stored = JSON.parse(
            localStorage.getItem("craftbox.asset-stack") ?? "[]",
          );
          if (Array.isArray(stored))
            setIDs(
              stored.filter(
                (id: unknown) =>
                  typeof id === "string" && packs.some((v) => v.id === id),
              ),
            );
        } catch {
          setIDs([]);
        }
      })
      .catch((e) => {
        if (!dead) setError(String(e));
      });
    return () => {
      dead = true;
    };
  }, [open]);
  async function apply(reset: boolean) {
    setBusy(true);
    setError("");
    try {
      await api.ApplyAssetStack(worldID, reset ? [] : ids);
      if (!reset)
        localStorage.setItem("craftbox.asset-stack", JSON.stringify(ids));
      onApplied();
      setOpen(false);
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  function move(index: number, delta: number) {
    setIDs((old) => {
      const next = [...old];
      [next[index], next[index + delta]] = [next[index + delta], next[index]];
      return next;
    });
  }
  return (
    <>
      <Button
        variant="outline"
        disabled={!worldID}
        onClick={() => setOpen(true)}
      >
        <Layers />
        Resource packs
      </Button>
      <Dialog
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
      >
        <DialogContent className="render-pack-dialog">
          <DialogHeader>
            <DialogTitle>Resource packs for this preview</DialogTitle>
            <DialogDescription>
              Select packs from your library. The first pack has priority. The
              original world files stay unchanged.
            </DialogDescription>
          </DialogHeader>
          <div className="render-pack-list">
            {ids.map((id, i) => (
              <div className="asset-stack-row" key={id}>
                <span>{i + 1}</span>
                <strong>{items.find((v) => v.id === id)?.name}</strong>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Move up"
                  disabled={busy || !i}
                  onClick={() => move(i, -1)}
                >
                  <ArrowUp />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Move down"
                  disabled={busy || i === ids.length - 1}
                  onClick={() => move(i, 1)}
                >
                  <ArrowDown />
                </Button>
              </div>
            ))}
            {items.map((v) => (
              <label key={v.id}>
                <input
                  type="checkbox"
                  disabled={busy}
                  checked={ids.includes(v.id)}
                  onChange={(e) =>
                    setIDs((old) =>
                      e.target.checked
                        ? [...old, v.id]
                        : old.filter((id) => id !== v.id),
                    )
                  }
                />
                {v.image && <img src={v.image} alt="" />}
                <span>{v.name}</span>
                <small>{v.version}</small>
              </label>
            ))}
            {!items.length && (
              <p>Import resource packs in the Resource packs tab first.</p>
            )}
            {error && (
              <FeedbackFlyout title="Render resource packs">
                {error}
              </FeedbackFlyout>
            )}
            <p>
              Changes affect the 3D preview and exports from this scene. The 2D
              map keeps its captured colours. Saved geometry-only projects
              require the original Bedrock world to change packs.
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => void apply(true)}
            >
              Restore original packs
            </Button>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button
              disabled={busy || !ids.length}
              onClick={() => void apply(false)}
            >
              {busy ? "Updating preview…" : "Apply packs"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function ResourceTraceDialog({
  trace,
  onClose,
}: {
  trace: Awaited<ReturnType<typeof api.TraceSceneResource>> | null;
  onClose: () => void;
}) {
  return (
    <Dialog
      open={!!trace}
      onOpenChange={(value) => {
        if (!value) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Block resources</DialogTitle>
          <DialogDescription>
            {trace?.block} · {trace?.face}
          </DialogDescription>
        </DialogHeader>
        {trace && (
          <div className="asset-dialog-body">
            <dl>
              <dt>Block position</dt>
              <dd>{trace.position.join(", ")}</dd>
              <dt>Texture</dt>
              <dd className="asset-mono">{trace.texture}</dd>
              <dt>Source</dt>
              <dd>{trace.found ? trace.pack : "No readable texture found"}</dd>
              <dt>Block state</dt>
              <dd className="asset-mono">{JSON.stringify(trace.states)}</dd>
            </dl>
            <p>{trace.note}</p>
          </div>
        )}
        <DialogFooter>
          <Button onClick={onClose}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import StudioInput from "./StudioInput";
import { fileNameError } from "./input-rules";
import { api, type ChunkBounds, type RenderWorld } from "./bridge";

export default function RenderRegionDialog({
  bounds,
  onClose,
  worldId,
  sourceName,
  onCreated,
  onBusy,
}: {
  bounds?: ChunkBounds;
  onClose: () => void;
  worldId: string;
  sourceName: string;
  onCreated: (result: { world: RenderWorld; path: string }) => void;
  onBusy?: (v: boolean) => void;
}) {
  const [name, setName] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [progress, setProgress] = useState({ stage: "", completed: 0, total: 0 });
  useEffect(() => {
    if (bounds) {
      setName((sourceName + " selection").slice(0, 80));
      setError("");
    }
  }, [bounds, sourceName]);
  useEffect(() => {
    if (!busy) return;
    let active = true;
    const poll = () =>
      void api
        .GetWorldExportProgress()
        .then((p) => {
          if (active) setProgress(p);
        })
        .catch(() => {});
    poll();
    const timer = setInterval(poll, 250);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [busy]);
  async function create() {
    if (!bounds) return;
    setBusy(true);
    onBusy?.(true);
    setError("");
    try {
      const result = await api.CreateRenderRegion({
        worldId,
        name: name.trim(),
        bounds,
      });
      onCreated(result);
      onClose();
    } catch (reason) {
      if (!/context cancel|cancelled|canceled/i.test(String(reason)))
        setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
      onBusy?.(false);
    }
  }
  return (
    <Dialog
      open={!!bounds}
      onOpenChange={(v) => {
        if (!v && !busy) onClose();
      }}
    >
      <DialogContent className="studio-dialog" showCloseButton={!busy}>
        <DialogHeader>
          <DialogTitle>Create a separate map</DialogTitle>
          <DialogDescription>
            Create a Bedrock world from the selected area. The original is kept.
          </DialogDescription>
        </DialogHeader>
        <div className="studio-dialog-body">
          <label>
            Map name
            <StudioInput
              value={name}
              validate={fileNameError}
              maxLength={80}
              disabled={busy}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          {bounds && (
            <p>
              X {bounds.minX}…{bounds.maxX} · Z {bounds.minZ}…{bounds.maxZ}{" "}
              chunks · Full world height, current dimension
            </p>
          )}
          <p>
            The copy includes the resource packs and opens in 3D. Its spawn is placed inside the selected area. You can open the saved .mcworld from the confirmation.
          </p>
          {error && <FeedbackFlyout title="Region extraction">{error}</FeedbackFlyout>}
          {busy && (
            <div role="status">
              {progress.stage || "Preparing copy"}
              {progress.total > 0 && (
                <>
                  <span>
                    {" "}
                    · {progress.completed}/{progress.total}
                  </span>
                  <progress
                    style={{ width: "100%" }}
                    value={progress.completed}
                    max={progress.total}
                  />
                </>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <button
            className="studio-button"
            onClick={() => (busy ? void api.CancelWorldExport() : onClose())}
          >
            {busy ? "Cancel creation" : "Cancel"}
          </button>
          <button
            className="studio-button primary"
            disabled={busy || !!fileNameError(name) || !name.trim()}
            onClick={() => void create()}
          >
            {busy ? "Creating …" : "Create map"}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

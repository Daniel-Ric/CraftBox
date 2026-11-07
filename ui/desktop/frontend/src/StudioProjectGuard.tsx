import FeedbackFlyout from "./FeedbackFlyout";
import { useRef, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { StudioSession } from "./studio-document";
export function useStudioProjectGuard() {
  const session = useRef<StudioSession | undefined>(undefined);
  const [pending, setPending] = useState<(() => void) | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const run = (action: () => void) => {
    if (session.current?.dirty) {
      setError("");
      setPending(() => action);
    } else action();
  };
  const dialog = (
    <Dialog
      open={!!pending}
      onOpenChange={(value) => {
        if (!value && !busy) setPending(null);
      }}
    >
      <DialogContent resizable={false} tone="confirmation" className="studio-confirm-dialog">
        <DialogHeader>
          <DialogTitle>Save changes?</DialogTitle>
          <DialogDescription>
            “{session.current?.name || "Untitled"}” has unsaved changes.
          </DialogDescription>
        </DialogHeader>
        <p>Save the world, styling and cameras.</p>
        {error && <FeedbackFlyout title="Project changes">{error}</FeedbackFlyout>}
        <DialogFooter>
          <Button
            variant="outline"
            disabled={busy}
            onClick={() => setPending(null)}
          >
            Cancel
          </Button>
          <Button
            variant="outline"
            disabled={busy || session.current?.busy}
            onClick={() => {
              const next = pending;
              setPending(null);
              next?.();
            }}
          >
            Discard
          </Button>
          <Button
            disabled={busy || session.current?.busy}
            onClick={async () => {
              setBusy(true);
              setError("");
              try {
                if (await session.current?.save()) {
                  const next = pending;
                  setPending(null);
                  next?.();
                } else
                  setError(
                    "The project was not saved. You can keep working or try saving again.",
                  );
              } catch (e) {
                setError(String(e));
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? "Saving project …" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
  return { session, run, dialog };
}

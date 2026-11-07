import { useEffect, useId, useRef } from "react";
import { toast } from "sonner";
import { AlertCircle, Copy, LoaderCircle, LogIn, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { MinecraftJoinPhase } from "./use-minecraft-join";
import "./minecraft-join.css";

export default function MinecraftJoinStatus({
  phase,
  address,
  launching,
  onRetry,
  onCopy,
}: {
  phase: MinecraftJoinPhase;
  address: string;
  launching: boolean;
  onRetry: () => void;
  onCopy: (address: string) => void;
}) {
  const id = useId();
  const actions = useRef({ onRetry, onCopy });
  actions.current = { onRetry, onCopy };
  useEffect(() => {
    if (phase === "idle") {
      toast.dismiss(id);
      return;
    }
    const delayed = phase === "delayed";
    toast.custom(
      () => (
        <section
          className="studio-flyout minecraft-join-flyout"
          data-tone={delayed ? "warning" : "info"}
          role="status"
          aria-live="polite"
          aria-atomic="true"
        >
          <div className="flyout-symbol">
            {delayed ? (
              <AlertCircle size={18} />
            ) : (
              <LoaderCircle size={18} className="spin" />
            )}
          </div>
          <div className="flyout-content">
            <strong>
              {delayed
                ? "Minecraft has not connected yet"
                : "Waiting for Minecraft to connect..."}
            </strong>
            <p>
              {delayed
                ? "If Minecraft is already open, return to its main menu, then retry the join. A slow startup can take longer; the proxy stays ready."
                : "The join request was sent. Waiting for Minecraft to reach the local proxy."}
            </p>
            {delayed && (
              <p>
                If retrying has no effect, close Minecraft normally and retry.
                Use this connection address instead of a saved localhost entry:{" "}
                <code>{address}</code>
              </p>
            )}
            <div className="minecraft-join-actions">
              {delayed && (
                <Button
                  variant="outline"
                  disabled={launching}
                  onClick={() => actions.current.onRetry()}
                >
                  <LogIn size={15} />
                  Retry join
                </Button>
              )}
              <Button
                variant="outline"
                onClick={() => actions.current.onCopy(address)}
              >
                <Copy size={15} />
                Copy connection address
              </Button>
            </div>
          </div>
          <button
            className="flyout-dismiss"
            aria-label="Dismiss notification"
            onClick={() => toast.dismiss(id)}
          >
            <X size={14} />
          </button>
        </section>
      ),
      {
        id,
        duration: Infinity,
        className: "minecraft-join-toast",
        style: {
          width: "min(460px, calc(100vw - 40px))",
          right: 0,
          left: "auto",
        },
      },
    );
  }, [id, phase, address, launching]);
  useEffect(
    () => () => {
      toast.dismiss(id);
    },
    [id],
  );
  return null;
}

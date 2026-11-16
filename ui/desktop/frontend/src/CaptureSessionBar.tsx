import {
  LoaderCircle,
  Play,
  Server,
  SlidersHorizontal,
  Square,
} from "lucide-react";
import { Button } from "./components/ui/button";

type ActionProps = {
  page: string;
  running: boolean;
  stopping: boolean;
  busy: boolean;
  ready: boolean;
  onStart: () => void;
  onStop: () => void;
};
export function CaptureSessionAction(p: ActionProps) {
  const active = p.running || p.stopping;
  return (
    <Button
      className={p.page === "worlds" ? "start-button capture-primary" : "capture-primary"}
      variant={active ? "destructive" : "default"}
      disabled={active ? p.stopping : !p.ready || p.busy}
      onClick={active ? p.onStop : p.onStart}
    >
      {p.stopping || p.busy ? (
        <LoaderCircle className="spin" />
      ) : active ? (
        <Square />
      ) : (
        <Play />
      )}
      {p.stopping
        ? "Stopping…"
        : active
          ? "End session"
          : p.busy
            ? "Starting…"
            : p.page === "worlds"
              ? "Start download"
              : "Start capture"}
    </Button>
  );
}
export default function CaptureSessionBar(
  p: ActionProps & {
    source: string;
    command: string;
    onConfigure: () => void;
    settingsOpen?: boolean;
  },
) {
  const labels: Record<string, string> = {
    worlds: "Worlds",
    packs: "Resource packs",
    skins: "Player skins",
    packets: "Packet Inspector",
  };
  const otherSession = p.running && p.command !== p.page;
  return (
    <section className="capture-session-bar" aria-label="Capture session">
      <div className="capture-session-source">
        <Server size={16} />
        <div>
          <strong>
            {p.running
              ? `${labels[p.command] ?? "Capture"} session running`
              : "Capture session"}
          </strong>
          <span title={p.source}>
            {otherSession
              ? "End the active session before starting another"
              : p.source || "Choose a source to start a capture"}
          </span>
        </div>
      </div>
      <div className="capture-session-actions">
        <Button
          variant="ghost"
          aria-expanded={p.settingsOpen}
          onClick={p.onConfigure}
        >
          <SlidersHorizontal size={15} />
          <span>Capture settings</span>
        </Button>
        <CaptureSessionAction {...p} />
      </div>
    </section>
  );
}

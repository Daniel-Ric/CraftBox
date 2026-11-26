import {
  ChevronDown,
  CircleAlert,
  LoaderCircle,
  Play,
  Server,
  SlidersHorizontal,
  Square,
} from "lucide-react";
import { useId } from "react";
import { Button } from "./components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "./components/ui/dropdown-menu";

type ActionProps = {
  page: string;
  running: boolean;
  stopping: boolean;
  busy: boolean;
  ready: boolean;
  onStart: () => void;
  onStop: () => void;
  flarial?: {
    clients: string[];
    selected: string;
    blocker?: string;
    onChange: (value: string) => void;
  };
};
const flarialLabels: Record<string, string> = {
  release: "Flarial Release",
  beta: "Flarial Beta",
  alpha: "Flarial Alpha",
};
export function CaptureSessionAction(p: ActionProps) {
  const active = p.running || p.stopping;
  const launcher = !active && p.flarial?.clients.length ? p.flarial : undefined;
  const startDisabled = !p.ready || p.busy || Boolean(launcher?.blocker);
  const description = useId();
  const label = p.stopping
    ? "Stopping…"
    : active
      ? "End session"
      : p.busy
        ? "Starting…"
        : p.page === "worlds"
          ? "Start download"
          : "Start capture";
  const button = (
    <Button
      className={
        p.page === "worlds" ? "start-button capture-primary" : "capture-primary"
      }
      variant={active ? "destructive" : launcher && startDisabled ? "ghost" : "default"}
      aria-label={label}
      aria-describedby={launcher?.blocker ? description : undefined}
      disabled={active ? p.stopping : startDisabled}
      onClick={active ? p.onStop : p.onStart}
    >
      <span className="capture-primary-label">
        {p.stopping || p.busy ? (
          <LoaderCircle className="spin" />
        ) : active ? (
          <Square />
        ) : (
          <Play />
        )}
        {label}
      </span>
      {launcher && (
        <small className="capture-launcher-caption">
          {flarialLabels[launcher.selected] ?? "Standard Minecraft"}
        </small>
      )}
    </Button>
  );
  if (!launcher) return button;
  return (
    <div className="capture-launcher-action">
      <div className="capture-launcher-split" data-start-disabled={startDisabled || undefined}>
        {button}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<Button className="capture-launcher-trigger" variant={startDisabled ? "ghost" : "default"} />}
            disabled={p.busy}
            aria-label="Minecraft launcher"
            title="Minecraft launcher"
          >
            <ChevronDown />
          </DropdownMenuTrigger>
          <DropdownMenuContent className="capture-launcher-menu" align="end" sideOffset={5}>
            <DropdownMenuRadioGroup
              value={launcher.selected}
              onValueChange={launcher.onChange}
            >
              <DropdownMenuRadioItem value="">
                Standard Minecraft
              </DropdownMenuRadioItem>
              {launcher.clients.map((channel) => (
                <DropdownMenuRadioItem key={channel} value={channel}>
                  {flarialLabels[channel]}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div id={description} className="capture-launcher-blocker" role="status">
        {launcher.blocker && (
          <>
            <CircleAlert aria-hidden="true" />
            <span>{launcher.blocker}</span>
          </>
        )}
      </div>
    </div>
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

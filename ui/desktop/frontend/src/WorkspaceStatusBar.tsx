import { Activity, Check, Copy, Info, LoaderCircle } from "lucide-react";
import { WorldIcon, PackIcon, SkinIcon } from "./studio-icons";
import NotificationCenter from "./NotificationCenter";
import "./workspace-statusbar.css";

type Props = {
  version: string;
  minecraft: string;
  running: boolean;
  stopping: boolean;
  session: string;
  worlds: number;
  packs: number;
  skins: number;
  onNavigate: (page: string) => void;
  onUpdates: () => void;
  onCopyDiagnostics: () => void;
  onConsole: () => void;
};

export default function WorkspaceStatusBar(p: Props) {
  const version = p.version.includes("-local") ? `${p.version.split("-")[0]} · Local` : p.version;
  const state = p.stopping ? "Stopping" : p.running ? "Running" : "Ready";
  const session = p.running || p.stopping ? `${state}: ${p.session}` : state;
  const counts = [
    {
      page: "worlds",
      count: p.worlds,
      noun: "world",
      label: "Worlds",
      icon: WorldIcon,
    },
    {
      page: "packs",
      count: p.packs,
      noun: "pack",
      label: "Resource packs",
      icon: PackIcon,
    },
    {
      page: "skins",
      count: p.skins,
      noun: "skin",
      label: "Player skins",
      icon: SkinIcon,
    },
  ];
  return (
    <footer
      className="application-statusbar workspace-statusbar"
      aria-label="Workspace status"
    >
      <div className="statusbar-session-group">
        <button
          className="statusbar-session"
          data-running={p.running}
          onClick={p.onConsole}
          aria-label={`${session} — open console`}
          data-tooltip={`${session} · Open console`}
        >
          {p.stopping ? (
            <LoaderCircle size={12} className="spin" />
          ) : p.running ? (
            <Activity size={12} />
          ) : (
            <Check size={12} />
          )}
          <span>{session}</span>
        </button>
        <button
          className="statusbar-version"
          onClick={p.onUpdates}
          aria-label="Version and updates"
          data-tooltip={`CraftBox ${p.version} · Version and updates`}
        >
          <Info size={12} />
          <span>CraftBox {version}</span>
        </button>
        <span
          className="statusbar-minecraft"
          data-tooltip={`Minecraft protocol version ${p.minecraft}`}
        >
          Minecraft {p.minecraft}
        </span>
      </div>
      <div className="statusbar-actions">
        <div className="statusbar-resources">
          {counts.map((item) => {
            const label = `${item.count} ${item.noun}${item.count === 1 ? "" : "s"} — open ${item.label}`;
            return (
              <button
                key={item.page}
                onClick={() => p.onNavigate(item.page)}
                aria-label={label}
                data-tooltip={label}
              >
                <item.icon size={12} />
                <span>{item.count}</span>
                <span className="statusbar-resource-name">
                  {item.label === "Resource packs"
                    ? "Packs"
                    : item.label === "Player skins"
                      ? "Skins"
                      : item.label}
                </span>
              </button>
            );
          })}
        </div>
        <button
          className="statusbar-diagnostics"
          aria-label="Copy diagnostics"
          data-tooltip="Copy diagnostics for a bug report"
          onClick={p.onCopyDiagnostics}
        >
          <Copy size={12} />
        </button>
        <NotificationCenter />
      </div>
    </footer>
  );
}

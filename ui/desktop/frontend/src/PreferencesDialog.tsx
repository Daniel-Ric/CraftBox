import { useEffect, useState } from "react";
import { Sun, Moon, Monitor, RefreshCw, Download } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import StudioBrand from "./StudioBrand";
export type WorkspacePreferences = {
  showWelcome: boolean;
  density: "compact" | "comfortable";
  reducedMotion: boolean;
};
const defaults: WorkspacePreferences = {
  showWelcome: true,
  density: "compact",
  reducedMotion: false,
};
const key = "craftbox-workspace-preferences";
export function loadWorkspacePreferences(): WorkspacePreferences {
  try {
    const p = JSON.parse(localStorage.getItem(key) || "null");
    return {
      showWelcome: typeof p?.showWelcome === "boolean" ? p.showWelcome : true,
      density: p?.density === "comfortable" ? "comfortable" : "compact",
      reducedMotion: p?.reducedMotion === true,
    };
  } catch {
    return defaults;
  }
}
export function applyWorkspacePreferences(p: WorkspacePreferences) {
  document.documentElement.dataset.density = p.density;
  document.documentElement.dataset.motion = p.reducedMotion
    ? "reduced"
    : "system";
}
export default function PreferencesDialog(p: {
  initialCategory?: string;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  theme: string;
  onTheme: (v: string) => void;
  version?: string;
  minecraft?: string;
  updateBusy: boolean;
  updateVersion?: string;
  updateProgress?: number;
  onCheckUpdate: () => void;
  onInstallUpdate?: () => void;
}) {
  const [category, setCategory] = useState("appearance"),
    [prefs, setPrefs] = useState(loadWorkspacePreferences);
  useEffect(() => applyWorkspacePreferences(prefs), [prefs]);
  useEffect(() => {
    if (p.open) setCategory(p.initialCategory || "appearance");
  }, [p.open, p.initialCategory]);
  const change = (next: Partial<WorkspacePreferences>) => {
    const value = { ...prefs, ...next };
    setPrefs(value);
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch {

    }
  };
  return (
    <Dialog open={p.open} onOpenChange={p.onOpenChange}>
      <DialogContent className="preferences-dialog">
        <DialogHeader>
          <DialogTitle>Preferences</DialogTitle>
          <DialogDescription>
            Customize the interface and application.
          </DialogDescription>
        </DialogHeader>
        <div className="preferences-body">
          <nav
            role="tablist"
            aria-label="Settings categories"
            aria-orientation="vertical"
          >
            {[
              ["appearance", "Appearance"],
              ["workspace", "Workspace"],
              ["version", "Version & updates"],
            ].map(([id, name]) => (
              <button
                key={id}
                id={"preferences-" + id}
                role="tab"
                aria-selected={category === id}
                aria-controls="preferences-content"
                onClick={() => setCategory(id)}
              >
                {name}
              </button>
            ))}
          </nav>
          <section
            className="preferences-content studio-dialog-body"
            role="tabpanel"
            id="preferences-content"
            aria-labelledby={"preferences-" + category}
          >
            {category === "appearance" ? (
              <>
                <h2>Appearance</h2>
                <p>Choose the interface brightness.</p>
                <div className="theme-swatches">
                  {[
                    {
                      id: "graphite",
                      name: "Charcoal",
                      color: "#181818",
                      icon: Moon,
                    },
                    {
                      id: "dark",
                      name: "Dark",
                      color: "#303030",
                      icon: Moon,
                    },
                    {
                      id: "gray",
                      name: "Mid-gray",
                      color: "#606060",
                      icon: Monitor,
                    },
                    { id: "light", name: "Light", color: "#ededed", icon: Sun },
                    {
                      id: "system",
                      name: "System",
                      color: "linear-gradient(90deg,#303030 50%,#ededed 50%)",
                      icon: Monitor,
                    },
                  ].map((t) => (
                    <button
                      key={t.id}
                      aria-pressed={p.theme === t.id}
                      onClick={() => p.onTheme(t.id)}
                    >
                      <i style={{ background: t.color }} />
                      <span>
                        <t.icon size={13} />
                        {t.name}
                      </span>
                    </button>
                  ))}
                </div>
                <div className="preference-note">
                  This applies to windows, dialogs and panels. It does not change the colors of your renders.
                </div>
              </>
            ) : category === "workspace" ? (
              <>
                <h2>Workspace</h2>
                <p>
                  Configure startup behavior and controls for your workflow.
                </p>
                <label className="studio-form-row">
                  <span>On startup</span>
                  <span className="preference-check">
                    <input
                      type="checkbox"
                      aria-label="Show welcome dialog on startup"
                      checked={prefs.showWelcome}
                      onChange={(e) =>
                        change({ showWelcome: e.target.checked })
                      }
                    />
                    Show welcome dialog on startup
                  </span>
                </label>
                <label className="studio-form-row">
                  <span>Controls</span>
                  <select
                    aria-label="Controls"
                    value={prefs.density}
                    onChange={(e) =>
                      change({
                        density: e.target
                          .value as WorkspacePreferences["density"],
                      })
                    }
                  >
                    <option value="compact">Compact</option>
                    <option value="comfortable">Comfortable</option>
                  </select>
                </label>
                <label className="studio-form-row">
                  <span>Animations</span>
                  <span className="preference-check">
                    <input
                      type="checkbox"
                      checked={prefs.reducedMotion}
                      onChange={(e) =>
                        change({ reducedMotion: e.target.checked })
                      }
                    />
                    Reduce motion
                  </span>
                </label>
                <p className="preference-note">
                  Changes apply immediately and are saved for the next launch.
                </p>
              </>
            ) : (
              <>
                <div className="preferences-identity">
                  <StudioBrand size={48} />
                  <div>
                    <h2>CraftBox</h2>
                    <p>
                      Capture worlds. Compose images. Create camera sequences.
                    </p>
                  </div>
                </div>
                <dl className="properties-list">
                  <div>
                    <dt>Version</dt>
                    <dd>{p.version || "—"}</dd>
                  </div>
                  <div>
                    <dt>Minecraft</dt>
                    <dd>{p.minecraft || "—"}</dd>
                  </div>
                  <div>
                    <dt>Connection</dt>
                    <dd>NetherNet</dd>
                  </div>
                  {p.updateVersion && (
                    <div>
                      <dt>Available</dt>
                      <dd>{p.updateVersion}</dd>
                    </div>
                  )}
                </dl>
                <div className="preference-update-actions">
                  <button
                    className="studio-button"
                    disabled={p.updateBusy}
                    onClick={p.onCheckUpdate}
                  >
                    <RefreshCw
                      size={14}
                      className={p.updateBusy ? "spin" : ""}
                    />
                    Check for updates
                  </button>
                  {p.updateVersion && p.onInstallUpdate && (
                    <button
                      className="studio-button primary"
                      disabled={p.updateBusy}
                      onClick={p.onInstallUpdate}
                    >
                      <Download size={14} />
                      Install update
                    </button>
                  )}
                </div>
                {!!p.updateProgress && (
                  <progress
                    max={100}
                    value={p.updateProgress}
                    aria-label="Update progress"
                  />
                )}
              </>
            )}
          </section>
        </div>
        <DialogFooter>
          <span className="dialog-footer-note">
            CraftBox · Preferences
          </span>
          <button
            className="studio-button primary"
            onClick={() => p.onOpenChange(false)}
          >
            Done
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

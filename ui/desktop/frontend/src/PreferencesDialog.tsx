import FeedbackFlyout from "./FeedbackFlyout";
import StudioInput from "@/StudioInput";
import DiagnosticsSettings from "./DiagnosticsSettings";
import DiscordPresenceSettings from "./DiscordPresenceSettings";
import "./preferences.css";
import { useEffect, useState } from "react";
import {
  Sun,
  Moon,
  Monitor,
  RefreshCw,
  Download,
  Search,
  ExternalLink,
  RotateCcw,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { api, type ReleaseCheck } from "./bridge";
import {
  interfaceKeys,
  interfacePreferences,
  interfaceSettings,
} from "./ingame-interface";
import {
  saveWorkspacePreferences,
  preferenceDefaults,
  preferenceCategories,
  preferenceSettings,
  useWorkspacePreferences,
  loadWorkspacePreferences,
  type PreferenceCategory,
  type WorkspacePreferences,
} from "./workspace-preferences";
export {
  loadWorkspacePreferences,
  applyWorkspacePreferences,
} from "./workspace-preferences";
export type { WorkspacePreferences } from "./workspace-preferences";
export default function PreferencesDialog(p: {
  initialCategory?: string;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  theme: string;
  onTheme: (v: string) => void;
  version?: string;
  minecraft?: string;
  updateInfo?: ReleaseCheck;
  updateError?: string;
  updateBusy: boolean;
  updateInstalling?: boolean;
  updateReady?: boolean;
  onRestartUpdate?: () => void;
  updateVersion?: string;
  onCheckUpdate: () => void;
  onInstallUpdate?: () => void;
}) {
  const [category, setCategory] = useState<PreferenceCategory>("appearance"),
    [query, setQuery] = useState(""),
    [storageError, setStorageError] = useState("");
  const prefs = useWorkspacePreferences();
  const [feedbackSaving, setFeedbackSaving] = useState(false);
  useEffect(() => {
    if (!p.open) return;
    let alive = true;
    void api
      .GetIngameInterfaceSettings()
      .then((settings) => {
        if (alive)
          saveWorkspacePreferences({
            ...loadWorkspacePreferences(),
            ...interfacePreferences(settings),
          });
      })
      .catch((e) => {
        if (alive) setStorageError(String(e));
      });
    return () => {
      alive = false;
    };
  }, [p.open]);
  useEffect(() => {
    if (p.open) {
      setCategory(
        preferenceCategories.some((c) => c.id === p.initialCategory)
          ? (p.initialCategory as PreferenceCategory)
          : "appearance",
      );
      setQuery("");
    }
  }, [p.open, p.initialCategory]);
  useEffect(() => {
    if (
      p.open &&
      category === "version" &&
      !p.updateReady &&
      !p.updateInstalling
    )
      p.onCheckUpdate();
  }, [p.open, category]);
  const change = (next: Partial<WorkspacePreferences>) => {
    if (interfaceKeys.some((key) => next[key] !== undefined)) {
      if (feedbackSaving) return;
      setFeedbackSaving(true);
      void api
        .GetIngameInterfaceSettings()
        .then(async (current) => {
          const desired = interfaceSettings({
            ...loadWorkspacePreferences(),
            ...interfacePreferences(current),
            ...next,
          });
          await api.SetIngameInterfaceSettings(desired);
          return desired;
        })
        .then((settings) => {
          saveWorkspacePreferences({
            ...loadWorkspacePreferences(),
            ...next,
            ...interfacePreferences(settings),
          });
          setStorageError("");
        })
        .catch((e) => setStorageError(String(e)))
        .finally(() => setFeedbackSaving(false));
      return;
    }
    try {
      saveWorkspacePreferences({ ...prefs, ...next });
      setStorageError("");
    } catch {
      setStorageError(
        "Preferences could not be saved. Check available storage and try again.",
      );
    }
  };
  const matches = (text: string) =>
    text.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase());
  const themeMatches = matches(
    "Appearance theme interface brightness charcoal dark mid-gray light system",
  );
  const settings = preferenceSettings.filter((s) =>
    query.trim()
      ? matches(
          s.label +
            " " +
            s.description +
            " " +
            preferenceCategories.find((c) => c.id === s.category)?.name,
        )
      : s.category === category,
  );
  const groups = preferenceCategories.filter((c) =>
    query.trim()
      ? settings.some((s) => s.category === c.id) ||
        (c.id === "appearance" && themeMatches) ||
        (c.id === "diagnostics" &&
          matches(
            "Diagnostics logs crash reports storage retention saved sessions",
          ))
      : c.id === category,
  );
  const reset = () => {
    if (category === "appearance") p.onTheme("dark");
    const patch: Partial<WorkspacePreferences> = {};
    for (const s of preferenceSettings)
      if (s.category === category)
        Object.assign(patch, { [s.key]: preferenceDefaults[s.key] });
    change(patch);
  };
  const settingField = (s: (typeof preferenceSettings)[number]) => (
    <PreferenceField
      key={s.key}
      setting={s}
      value={
        s.key === "lanDiscovery" && prefs.destinationHostname
          ? false
          : prefs[s.key]
      }
      disabled={
        (s.key === "lanDiscovery" && prefs.destinationHostname) ||
        (interfaceKeys.includes(s.key) && feedbackSaving) ||
        (s.key === "discordState" && prefs.discordAutomatic) ||
        (s.key === "discordSmallText" && !prefs.discordSmallImage.trim())
      }
      onChange={(value) => change({ [s.key]: value })}
    />
  );
  return (
    <Dialog open={p.open} onOpenChange={p.onOpenChange}>
      <DialogContent className="preferences-dialog">
        <DialogHeader>
          <DialogTitle>Preferences</DialogTitle>
          <DialogDescription>
            Customize your workspace, editors and updates.
          </DialogDescription>
        </DialogHeader>
        <div className="preferences-search">
          <Search size={14} />
          <StudioInput
            type="search"
            aria-label="Search settings"
            placeholder="Search settings"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <span className="preferences-search-status">
            {query
              ? settings.length +
                (themeMatches ? 1 : 0) +
                (groups.some((g) => g.id === "diagnostics") ? 1 : 0) +
                " results"
              : category === "diagnostics"
                ? "Save to apply log storage limits"
                : "Saved automatically"}
          </span>
        </div>
        <div className="preferences-body">
          <nav
            className="studio-sidebar"
            role="tablist"
            aria-label="Settings categories"
            aria-orientation="vertical"
            onKeyDown={(e) => {
              const tabs = [
                ...e.currentTarget.querySelectorAll<HTMLButtonElement>(
                  "[role=tab]",
                ),
              ];
              const index = tabs.indexOf(
                document.activeElement as HTMLButtonElement,
              );
              const next =
                e.key === "ArrowDown"
                  ? (index + 1) % tabs.length
                  : e.key === "ArrowUp"
                    ? (index - 1 + tabs.length) % tabs.length
                    : e.key === "Home"
                      ? 0
                      : e.key === "End"
                        ? tabs.length - 1
                        : -1;
              if (next >= 0) {
                e.preventDefault();
                tabs[next].click();
                tabs[next].focus();
              }
            }}
          >
            {preferenceCategories.map((c) => (
              <button
                key={c.id}
                id={"preferences-" + c.id}
                role="tab"
                tabIndex={category === c.id ? 0 : -1}
                aria-selected={!query && category === c.id}
                aria-controls="preferences-content"
                onClick={() => {
                  setCategory(c.id);
                  setQuery("");
                }}
              >
                {c.name}
              </button>
            ))}
          </nav>
          <section
            className="preferences-content studio-dialog-body"
            role={query ? "region" : "tabpanel"}
            id="preferences-content"
            aria-label={query ? "Search results" : undefined}
            aria-labelledby={query ? undefined : "preferences-" + category}
          >
            {storageError && (
              <FeedbackFlyout title="Settings">{storageError}</FeedbackFlyout>
            )}
            {!groups.length && (
              <p role="status">No settings found. Try another search.</p>
            )}
            {groups.map((group) => (
              <section className="preference-group" key={group.id}>
                <div className="preference-group-heading">
                  <div>
                    <h2>{group.name}</h2>
                    <p>{group.description}</p>
                  </div>
                  {!query && group.id !== "diagnostics" && (
                    <button
                      className="studio-button"
                      aria-label="Restore category defaults"
                      onClick={reset}
                    >
                      <RotateCcw size={13} />
                      Restore defaults
                    </button>
                  )}
                </div>
                {group.id === "diagnostics" && <DiagnosticsSettings />}
                {group.id === "discord" && !query.trim() && (
                  <DiscordPresenceSettings
                    field={(key) => {
                      const setting = preferenceSettings.find(
                        (s) => s.key === key,
                      );
                      return setting ? settingField(setting) : null;
                    }}
                  />
                )}
                {group.id === "appearance" && (!query || themeMatches) && (
                  <>
                    {" "}
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
                        {
                          id: "light",
                          name: "Light",
                          color: "#ededed",
                          icon: Sun,
                        },
                        {
                          id: "system",
                          name: "System",
                          color:
                            "linear-gradient(90deg,#303030 50%,#ededed 50%)",
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
                  </>
                )}
                {settings
                  .filter(
                    (s) =>
                      s.category === group.id &&
                      (group.id !== "discord" || !!query.trim()),
                  )
                  .map(settingField)}
                {group.id === "version" && (
                  <div className="update-details">
                    <div className="preferences-version">
                      <strong>
                        {p.updateInfo?.currentRelease?.name ||
                          `CraftBox ${p.version || "Development build"}`}
                      </strong>
                      <span>Minecraft {p.minecraft || "—"}</span>
                    </div>
                    <p className="preference-note">
                      Updates from Daniel-Ric/CraftBox on GitHub. Downloads are
                      verified with SHA-256.
                    </p>
                    {p.updateReady ? (
                      <p role="status">
                        Update installed. Restart CraftBox when you are ready.
                      </p>
                    ) : p.updateBusy ? (
                      <p role="status">
                        {p.updateInstalling
                          ? "Installing update. Follow progress in the update dialog."
                          : "Checking GitHub for updates…"}
                      </p>
                    ) : p.updateError ? (
                      <FeedbackFlyout title="Settings">
                        {p.updateError}
                      </FeedbackFlyout>
                    ) : p.updateInfo ? (
                      <p role="status">
                        {p.updateInfo.available
                          ? "A newer release is available."
                          : p.updateInfo.release
                            ? "You are up to date on this channel."
                            : "No published release is available on this channel."}
                      </p>
                    ) : (
                      <p role="status">Check GitHub for the latest release.</p>
                    )}
                    {p.updateInfo && (
                      <p className="preference-note">
                        Last checked:{" "}
                        {new Date(p.updateInfo.checkedAt).toLocaleString()}
                      </p>
                    )}
                    {p.updateInfo?.release && (
                      <div className="release-summary">
                        <strong>
                          {p.updateInfo.release.name ||
                            p.updateInfo.release.version}{" "}
                          {p.updateInfo.release.prerelease && (
                            <span className="release-badge">Preview</span>
                          )}
                        </strong>
                        <p>
                          {p.updateInfo.release.version}
                          {p.updateInfo.release.publishedAt &&
                            " · " +
                              new Date(
                                p.updateInfo.release.publishedAt,
                              ).toLocaleDateString()}
                        </p>
                        {p.updateInfo.release.assetName && (
                          <p>
                            {p.updateInfo.release.assetName} ·{" "}
                            {(p.updateInfo.release.size / 1048576).toFixed(1)}{" "}
                            MB
                          </p>
                        )}
                        {p.updateInfo.release.unavailableReason && (
                          <p>{p.updateInfo.release.unavailableReason}</p>
                        )}
                        <details>
                          <summary>What's new</summary>
                          <pre>
                            {p.updateInfo.release.notes ||
                              "No release notes provided."}
                          </pre>
                        </details>
                      </div>
                    )}
                    <div className="preference-update-actions">
                      <button
                        className="studio-button"
                        disabled={p.updateBusy || p.updateReady}
                        onClick={p.onCheckUpdate}
                      >
                        <RefreshCw
                          size={14}
                          className={p.updateBusy ? "spin" : ""}
                        />
                        {p.updateInstalling
                          ? "Installing…"
                          : p.updateBusy
                            ? "Checking…"
                            : "Check for updates"}
                      </button>
                      {p.updateReady && (
                        <button
                          className="studio-button primary"
                          onClick={p.onRestartUpdate}
                        >
                          Restart to apply update…
                        </button>
                      )}
                      {!p.updateReady &&
                        p.updateInfo?.available &&
                        !p.updateError &&
                        p.updateInfo.release?.installable && (
                          <button
                            className="studio-button primary"
                            disabled={p.updateBusy || !p.onInstallUpdate}
                            onClick={p.onInstallUpdate}
                          >
                            <Download size={14} />
                            Install update
                          </button>
                        )}
                      <button
                        className="studio-button"
                        onClick={() =>
                          void api
                            .OpenURL(
                              p.updateInfo?.release?.releaseURL ||
                                "https://github.com/Daniel-Ric/CraftBox/releases",
                            )
                            .catch((e) => setStorageError(String(e)))
                        }
                      >
                        <ExternalLink size={14} />
                        GitHub releases
                      </button>
                    </div>
                    {!!p.updateVersion && !p.onInstallUpdate && (
                      <p className="preference-note">
                        End the active capture session before installing an
                        update.
                      </p>
                    )}
                  </div>
                )}
              </section>
            ))}
          </section>
        </div>
        <DialogFooter>
          <span className="dialog-footer-note">
            {groups.some((g) => g.id === "diagnostics")
              ? "Save log settings to apply limits · Other preferences save automatically"
              : "Changes apply automatically · Default settings apply to new projects"}
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
function PreferenceField({
  setting: s,
  value,
  onChange,
  disabled = false,
}: {
  setting: (typeof preferenceSettings)[number];
  value: WorkspacePreferences[keyof WorkspacePreferences];
  onChange: (v: string | number | boolean) => void;
  disabled?: boolean;
}) {
  const [draft, setDraft] = useState(String(value));
  useEffect(() => setDraft(String(value)), [value]);
  return (
    <label
      className="preference-setting"
      style={disabled ? { opacity: 0.5 } : undefined}
    >
      <span>
        <strong>{s.label}</strong>
        <small>{s.description}</small>
      </span>
      {typeof value === "boolean" ? (
        <StudioInput
          type="checkbox"
          disabled={disabled}
          aria-label={s.label}
          checked={value}
          onChange={(e) => onChange(e.target.checked)}
        />
      ) : s.choices ? (
        <select
          aria-label={s.label}
          value={String(value)}
          disabled={disabled}
          onChange={(e) =>
            onChange(
              typeof value === "number"
                ? Number(e.target.value)
                : e.target.value,
            )
          }
        >
          {s.choices.map((c) => (
            <option key={c[0]} value={c[0]}>
              {c[1]}
            </option>
          ))}
        </select>
      ) : typeof value === "string" ? (
        <StudioInput
          type="text"
          disabled={disabled}
          aria-label={s.label}
          placeholder={
            s.key.endsWith("Url") || s.key === "discordSmallImage"
              ? "https://example.com"
              : undefined
          }
          validate={(value) =>
            (s.key.endsWith("Url") || s.key === "discordSmallImage") &&
            value.trim() &&
            !/^https?:\/\/[^\s/]+(?:\/[^\s]*)?$/.test(value.trim())
              ? "Enter a full link, such as https://example.com."
              : undefined
          }
          maxLength={s.maxLength ?? 128}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => onChange(draft)}
        />
      ) : (
        <StudioInput
          type="number"
          disabled={disabled}
          aria-label={s.label}
          min={s.min}
          max={s.max}
          step={s.step}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => {
            const n = Number(draft);
            if (
              draft.trim() &&
              Number.isFinite(n) &&
              (s.key !== "localConnectionPort" || Number.isInteger(n)) &&
              n >= (s.min ?? -Infinity) &&
              n <= (s.max ?? Infinity)
            )
              onChange(n);
            else setDraft(String(value));
          }}
        />
      )}
    </label>
  );
}

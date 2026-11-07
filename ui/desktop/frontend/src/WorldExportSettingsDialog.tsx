import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useState } from "react";
import {
  api,
  type WorldExportConfiguration,
  type WorldExportSettings,
} from "./bridge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import StudioInput from "./StudioInput";
import { notify, notifyFailure } from "./StudioFlyouts";
import "./world-export-settings.css";

const rules = [
  ["daylightCycle", "Daylight cycle"],
  ["weatherCycle", "Weather cycle"],
  ["mobSpawning", "Mob spawning"],
  ["mobGriefing", "Mob griefing"],
  ["fireTick", "Fire spread"],
  ["keepInventory", "Keep inventory"],
  ["tileDrops", "Block drops"],
  ["entityDrops", "Entity drops"],
  ["showCoordinates", "Show coordinates"],
  ["commandBlocks", "Command blocks"],
  ["commands", "Allow commands"],
] as const;
export default function WorldExportSettingsDialog({
  worldId,
  onClose,
}: {
  worldId: string;
  onClose: () => void;
}) {
  const [config, setConfig] = useState<WorldExportConfiguration>(),
    [profile, setProfile] = useState("original"),
    [settings, setSettings] = useState<WorldExportSettings>({ rules: {} }),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [spawnEnabled, setSpawnEnabled] = useState(false),
    [spawn, setSpawn] = useState(["0", "80", "0"]),
    [time, setTime] = useState(""),
    [randomTicks, setRandomTicks] = useState("");
  useEffect(() => {
    let current = true;
    setConfig(undefined);
    setError("");
    void api
      .GetWorldExportConfig(worldId)
      .then((value) => {
        if (!current) return;
        setConfig(value);
        setProfile(value.options.profile || "original");
        setSettings(value.options.settings || { rules: {} });
        setTime(String(value.options.settings?.time ?? ""));
        setRandomTicks(
          String(value.options.settings?.rules?.randomTickSpeed ?? ""),
        );
        setSpawnEnabled(!!value.options.settings?.spawn);
        setSpawn((value.options.settings?.spawn ?? [0, 80, 0]).map(String));
      })
      .catch((e) => {
        if (current) setError(String(e));
      });
    return () => {
      current = false;
    };
  }, [worldId]);
  const change = (patch: Partial<WorldExportSettings>) =>
    setSettings((value) => ({ ...value, ...patch }));
  async function save() {
    if (!config || busy) return;
    let coordinates: [number, number, number] | undefined;
    if (spawnEnabled) {
      const values = spawn.map(Number);
      if (
        spawn.some((v) => !v.trim()) ||
        values.some((v) => !Number.isInteger(v))
      ) {
        setError("Spawn coordinates must be whole numbers.");
        return;
      }
      coordinates = values as [number, number, number];
    }
    setBusy(true);
    setError("");
    try {
      const payload = { ...settings, rules: { ...settings.rules } };
      delete payload.spawn;
      delete payload.time;
      delete payload.rules.randomTickSpeed;
      if (time.trim()) payload.time = Number(time);
      if (randomTicks.trim())
        payload.rules.randomTickSpeed = Number(randomTicks);
      if (coordinates) payload.spawn = coordinates;
      if (payload.name !== undefined) {
        payload.name = payload.name.trim();
        if (!payload.name) delete payload.name;
      }
      await api.EditWorld({ worldId, export: { profile, settings: payload } });
      notify({
        tone: "success",
        title: "World export settings saved",
        description: config.capturing
          ? "The settings will be applied when this capture finishes."
          : "The world archive has been updated.",
      });
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      notifyFailure(e, "World export settings could not be saved");
    } finally {
      setBusy(false);
    }
  }
  function choice(
    label: string,
    key: "gameMode" | "difficulty" | "weather",
    values: readonly (readonly [string, string])[],
  ) {
    return (
      <label>
        <span>{label}</span>
        <select
          aria-label={label}
          disabled={!config || busy}
          value={String(settings[key] ?? "")}
          onChange={(e) => {
            const value = e.target.value;
            change({
              [key]:
                value === ""
                  ? undefined
                  : key === "weather"
                    ? value
                    : Number(value),
            });
          }}
        >
          <option value="">Use profile / captured value</option>
          {values.map(([id, name]) => (
            <option key={id} value={id}>
              {name}
            </option>
          ))}
        </select>
      </label>
    );
  }
  return (
    <Dialog
      open
      onOpenChange={(v) => {
        if (!v && !busy) onClose();
      }}
    >
      <DialogContent className="world-export-settings-dialog">
        <DialogHeader>
          <DialogTitle>World export settings</DialogTitle>
          <DialogDescription>
            {config?.name || "Loading world settings…"}
          </DialogDescription>
        </DialogHeader>
        <div className="world-export-settings-body studio-dialog-body">
          {error && (
            <FeedbackFlyout title="World export settings">
              {error}
            </FeedbackFlyout>
          )}
          <fieldset disabled={!config || busy}>
            <legend>Export profile</legend>
            <label>
              <span>Profile</span>
              <select
                aria-label="Export profile"
                value={profile}
                onChange={(e) => setProfile(e.target.value)}
              >
                {(
                  config?.profiles ?? [
                    {
                      id: "original",
                      name: "Original capture",
                      description: "",
                    },
                  ]
                ).map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </label>
            <p className="world-export-settings-note">
              {config?.profiles.find((p) => p.id === profile)?.description}{" "}
              Changes below override the profile. Leave a setting unchanged to keep the profile or captured value.
            </p>
          </fieldset>
          <fieldset disabled={!config || busy}>
            <legend>World and spawn</legend>
            <label>
              <span>World name</span>
              <StudioInput
                aria-label="World name"
                maxLength={80}
                placeholder={config?.name || "Keep world name"}
                value={settings.name ?? ""}
                onChange={(e) => change({ name: e.target.value })}
              />
            </label>
            <label className="world-export-spawn-check">
              <StudioInput
                type="checkbox"
                aria-label="Set spawn point"
                checked={spawnEnabled}
                onChange={(e) => setSpawnEnabled(e.target.checked)}
              />
              <span>Set spawn point in captured terrain</span>
            </label>
            {spawnEnabled && (
              <div className="world-export-spawn">
                {["X", "Y", "Z"].map((axis, i) => (
                  <label key={axis}>
                    <span>{axis}</span>
                    <StudioInput
                      type="number"
                      step={1}
                      aria-label={"Spawn " + axis}
                      value={spawn[i]}
                      onChange={(e) =>
                        setSpawn((values) =>
                          values.map((v, j) => (i === j ? e.target.value : v)),
                        )
                      }
                    />
                  </label>
                ))}
              </div>
            )}
          </fieldset>
          <fieldset disabled={!config || busy}>
            <legend>Gameplay</legend>
            <div className="world-export-settings-grid">
              {choice("Game mode", "gameMode", [
                ["0", "Survival"],
                ["1", "Creative"],
                ["2", "Adventure"],
              ])}
              {choice("Difficulty", "difficulty", [
                ["0", "Peaceful"],
                ["1", "Easy"],
                ["2", "Normal"],
                ["3", "Hard"],
              ])}
            </div>
          </fieldset>
          <fieldset disabled={!config || busy}>
            <legend>Time and weather</legend>
            <div className="world-export-settings-grid">
              <label>
                <span>World time (ticks)</span>
                <StudioInput
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  validate={(v) =>
                    v && !Number.isSafeInteger(Number(v))
                      ? "Enter a whole number of ticks."
                      : undefined
                  }
                  aria-label="World time (ticks)"
                  placeholder="Use profile / captured value"
                  value={time}
                  onChange={(e) => setTime(e.target.value)}
                />
              </label>
              {choice("Weather", "weather", [
                ["clear", "Clear"],
                ["rain", "Rain"],
                ["thunder", "Thunderstorm"],
              ])}
            </div>
          </fieldset>
          <fieldset disabled={!config || busy}>
            <legend>Game rules</legend>
            <div className="world-export-settings-grid">
              {rules.map(([key, label]) => (
                <label key={key}>
                  <span>{label}</span>
                  <select
                    aria-label={label}
                    value={String(settings.rules?.[key] ?? "")}
                    onChange={(e) =>
                      change({
                        rules: {
                          ...settings.rules,
                          [key]:
                            e.target.value === ""
                              ? undefined
                              : e.target.value === "true",
                        },
                      })
                    }
                  >
                    <option value="">Use profile / captured value</option>
                    <option value="true">Enabled</option>
                    <option value="false">Disabled</option>
                  </select>
                </label>
              ))}
              <label>
                <span>Random tick speed</span>
                <StudioInput
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  validate={(v) =>
                    v && (!Number.isInteger(Number(v)) || Number(v) > 4096)
                      ? "Enter a whole number from 0 to 4096."
                      : undefined
                  }
                  aria-label="Random tick speed"
                  placeholder="Use profile / captured value"
                  value={randomTicks}
                  onChange={(e) => setRandomTicks(e.target.value)}
                />
              </label>
            </div>
          </fieldset>
          {config?.capturing && (
            <p className="world-export-settings-note">
              Changes apply to the completed world archive when the capture
              finishes.
            </p>
          )}
        </div>
        <DialogFooter>
          <button className="studio-button" disabled={busy} onClick={onClose}>
            Cancel
          </button>
          <button
            className="studio-button primary"
            disabled={!config || busy}
            onClick={() => void save()}
          >
            {busy ? "Saving…" : "Save export settings"}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

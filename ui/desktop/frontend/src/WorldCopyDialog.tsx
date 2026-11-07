import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useRef, useState } from "react";
import {
  api,
  type WorldExportConfiguration,
  type WorldExportOptions,
  type WorldExportPlan,
  type WorldExportProgress,
  type WorldDimensionRule,
} from "./bridge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import StudioInput from "./StudioInput";
import { notifyFailure, notifySaved } from "./StudioFlyouts";
import "./world-export-settings.css";

type DimensionDraft = {
  mode: "keep" | "exclude" | "remap";
  target: string;
  crop: boolean;
  bounds: [string, string, string, string];
};
type BiomeDraft = { source: string; target: string };
const int32 = (s: string) =>
  s.trim() !== "" &&
  Number.isInteger(Number(s)) &&
  Number(s) >= -2147483648 &&
  Number(s) <= 2147483647;

export default function WorldCopyDialog({
  worldId,
  onClose,
}: {
  worldId: string;
  onClose: () => void;
}) {
  const [config, setConfig] = useState<WorldExportConfiguration>(),
    [catalog, setCatalog] = useState<WorldExportPlan>(),
    [plan, setPlan] = useState<WorldExportPlan>();
  const [profile, setProfile] = useState("original"),
    [name, setName] = useState(""),
    [compact, setCompact] = useState(false),
    [dimensions, setDimensions] = useState<Record<number, DimensionDraft>>({}),
    [biomes, setBiomes] = useState<BiomeDraft[]>([]);
  const [relocate, setRelocate] = useState(false),
    [playerDimension, setPlayerDimension] = useState("0"),
    [spawn, setSpawn] = useState(["0", "80", "0"]);
  const [mapDimension, setMapDimension] = useState(0),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [previewing, setPreviewing] = useState(false),
    [progress, setProgress] = useState<WorldExportProgress>();
  const mounted = useRef(true),
    sequence = useRef(0),
    [validSignature, setValidSignature] = useState("");
  useEffect(() => {
    mounted.current = true;
    let current = true;
    void api
      .GetWorldExportConfig(worldId)
      .then(async (value) => {
        if (!current) return;
        setConfig(value);
        setName(value.name + " export");
        setProfile(value.options.profile || "original");
        const initial = await api.GetWorldExportCatalog(worldId);
        if (!current) return;
        setCatalog(initial);
        setMapDimension(initial.dimensions[0]?.id ?? 0);
        setPlayerDimension(String(initial.dimensions[0]?.id ?? 0));
      })
      .catch((e) => {
        if (current) setError(String(e));
      });
    return () => {
      current = false;
      mounted.current = false;
      sequence.current++;
    };
  }, [worldId]);
  useEffect(() => {
    if (!busy) return;
    let active = true;
    const poll = () =>
      void api
        .GetWorldExportProgress()
        .then((value) => {
          if (active) setProgress(value);
        })
        .catch(() => {});
    poll();
    const timer = setInterval(poll, 350);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [busy]);
  function buildOptions(): WorldExportOptions {
    const rules: WorldDimensionRule[] = [];
    for (const [key, draft] of Object.entries(dimensions)) {
      const source = Number(key);
      if (draft.mode === "keep" && !draft.crop) continue;
      if (draft.mode === "remap" && !int32(draft.target))
        throw Error("Enter a whole target dimension ID.");
      const rule: WorldDimensionRule = {
        source,
        target:
          draft.mode === "exclude"
            ? null
            : draft.mode === "remap"
              ? Number(draft.target)
              : source,
      };
      if (draft.crop && draft.mode !== "exclude") {
        if (draft.bounds.some((v) => !int32(v)))
          throw Error("Enter four whole chunk coordinates for the crop.");
        const [minX, minZ, maxX, maxZ] = draft.bounds.map(Number);
        if (minX > maxX || minZ > maxZ)
          throw Error("Crop minimum coordinates must not exceed the maximum.");
        rule.bounds = { minX, minZ, maxX, maxZ };
      }
      rules.push(rule);
    }
    const mappings = biomes.map((row) => {
      if (
        !int32(row.source) ||
        Number(row.source) < 0 ||
        !int32(row.target) ||
        Number(row.target) < 0
      )
        throw Error("Choose a source and target biome for every replacement.");
      return { source: Number(row.source), target: Number(row.target) };
    });
    const options: WorldExportOptions = {
      profile,
      settings: {
        ...config?.options.settings,
        name: name.trim() || undefined,
        rules: { ...config?.options.settings.rules },
      },
      dimensions: rules,
      biomes: mappings,
      compact,
    };
    if (relocate) {
      if (!int32(playerDimension) || spawn.some((v) => !int32(v)))
        throw Error(
          "Enter a source dimension and three whole spawn coordinates.",
        );
      options.playerDimension = Number(playerDimension);
      options.settings.spawn = spawn.map(Number) as [number, number, number];
    }
    return options;
  }
  function signature() {
    try {
      return JSON.stringify(buildOptions());
    } catch {
      return "";
    }
  }
  const currentSignature = signature(),
    isValid = !!plan && validSignature === currentSignature;
  async function preview() {
    const request = ++sequence.current;
    setError("");
    setPreviewing(true);
    try {
      const options = buildOptions();
      const value = await api.PreviewWorldExport({ worldId, options });
      if (!mounted.current || request !== sequence.current) return;
      setPlan(value);
      setValidSignature(JSON.stringify(options));
    } catch (e) {
      if (mounted.current && request === sequence.current) {
        setError(String(e));
        setPlan(undefined);
        setValidSignature("");
      }
    } finally {
      if (mounted.current && request === sequence.current) setPreviewing(false);
    }
  }
  async function exportCopy() {
    if (!isValid || busy) return;
    setBusy(true);
    setError("");
    try {
      const result = await api.ExportWorldCopy({
        worldId,
        options: buildOptions(),
      });
      notifySaved(result.path);
      onClose();
    } catch (e) {
      if (mounted.current) setError(String(e));
      notifyFailure(e, "World copy could not be exported");
    } finally {
      if (mounted.current) setBusy(false);
    }
  }
  function editDimension(id: number, patch: Partial<DimensionDraft>) {
    setDimensions((value) => ({
      ...value,
      [id]: {
        ...(value[id] ?? {
          mode: "keep",
          target: String(id),
          crop: false,
          bounds: ["0", "0", "0", "0"],
        }),
        ...patch,
      },
    }));
  }
  const shown = plan ?? catalog,
    rows = shown?.chunks.filter((c) => c.dimension === mapDimension) ?? [];
  const xs = rows.map((c) => c.x),
    zs = rows.map((c) => c.z),
    minX = xs.length ? Math.min(...xs) : 0,
    minZ = zs.length ? Math.min(...zs) : 0,
    maxX = xs.length ? Math.max(...xs) : 1,
    maxZ = zs.length ? Math.max(...zs) : 1;
  const chunkPath = (included: boolean) =>
    rows
      .filter((c) => c.included === included)
      .map((c) => `M${c.x} ${c.z}h1v1h-1z`)
      .join("");
  return (
    <Dialog
      open={!busy}
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
    >
      <DialogContent className="world-export-settings-dialog world-copy-dialog">
        <DialogHeader>
          <DialogTitle>Export world copy</DialogTitle>
          <DialogDescription>
            {config?.name || "Loading world…"}
          </DialogDescription>
        </DialogHeader>
        <div className="world-export-settings-body studio-dialog-body">
          {error && (
            <FeedbackFlyout title="World copy">
              {error}
            </FeedbackFlyout>
          )}
          <fieldset disabled={!catalog || busy || previewing}>
            <legend>Destination and profile</legend>
            <div className="world-export-settings-grid">
              <label>
                <span>Copy name</span>
                <StudioInput
                  aria-label="Copy name"
                  maxLength={80}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <label>
                <span>Profile</span>
                <select
                  aria-label="Copy profile"
                  value={profile}
                  onChange={(e) => setProfile(e.target.value)}
                >
                  {config?.profiles.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <p className="world-export-settings-note">
              Saves a separate .mcworld beside the original, using your World export settings.
            </p>
          </fieldset>
          <fieldset disabled={!catalog || busy || previewing}>
            <legend>Dimensions and chunk selection</legend>
            {catalog?.dimensions.map((d) => {
              const draft = dimensions[d.id] ?? {
                mode: "keep",
                target: String(d.id),
                crop: false,
                bounds: ["0", "0", "0", "0"] as [
                  string,
                  string,
                  string,
                  string,
                ],
              };
              return (
                <div className="world-copy-dimension" key={d.id}>
                  <div className="world-export-settings-grid">
                    <label>
                      <span>
                        {d.name} · {d.chunks} chunks
                      </span>
                      <select
                        aria-label={"Export " + d.name}
                        value={draft.mode}
                        onChange={(e) =>
                          editDimension(d.id, {
                            mode: e.target.value as DimensionDraft["mode"],
                          })
                        }
                      >
                        <option value="keep">Keep dimension</option>
                        <option value="exclude">Exclude dimension</option>
                        <option value="remap">Map to another dimension</option>
                      </select>
                    </label>
                    {draft.mode === "remap" && (
                      <label>
                        <span>Target dimension ID</span>
                        <StudioInput
                          aria-label={"Target dimension " + d.id}
                          value={draft.target}
                          onChange={(e) =>
                            editDimension(d.id, { target: e.target.value })
                          }
                        />
                        <small>
                          0 Overworld · 1 Nether · 2 End · custom ID
                        </small>
                      </label>
                    )}
                  </div>
                  {draft.mode !== "exclude" && (
                    <>
                      <label className="world-export-spawn-check">
                        <StudioInput
                          type="checkbox"
                          checked={draft.crop}
                          onChange={(e) =>
                            editDimension(d.id, { crop: e.target.checked })
                          }
                        />
                        <span>Crop {d.name} to chunk bounds</span>
                      </label>
                      {draft.crop && (
                        <div className="world-copy-bounds">
                          {["Min X", "Min Z", "Max X", "Max Z"].map(
                            (label, i) => (
                              <label key={label}>
                                <span>{label}</span>
                                <StudioInput
                                  aria-label={`${d.name} ${label}`}
                                  inputMode="numeric"
                                  value={draft.bounds[i]}
                                  onChange={(e) => {
                                    const bounds = [
                                      ...draft.bounds,
                                    ] as DimensionDraft["bounds"];
                                    bounds[i] = e.target.value;
                                    editDimension(d.id, { bounds });
                                  }}
                                />
                              </label>
                            ),
                          )}
                        </div>
                      )}
                    </>
                  )}
                </div>
              );
            })}
          </fieldset>
          <fieldset disabled={!catalog || busy || previewing}>
            <legend>Player and spawn</legend>
            <label className="world-export-spawn-check">
              <StudioInput
                type="checkbox"
                checked={relocate}
                onChange={(e) => setRelocate(e.target.checked)}
              />
              <span>Move the player and spawn into the selected area</span>
            </label>
            {relocate && (
              <>
                <label>
                  <span>Source dimension</span>
                  <select
                    aria-label="Player source dimension"
                    value={playerDimension}
                    onChange={(e) => setPlayerDimension(e.target.value)}
                  >
                    {catalog?.dimensions.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.name}
                      </option>
                    ))}
                  </select>
                </label>
                <div className="world-export-spawn">
                  {["X", "Y", "Z"].map((axis, i) => (
                    <label key={axis}>
                      <span>{axis}</span>
                      <StudioInput
                        aria-label={"Copy spawn " + axis}
                        inputMode="numeric"
                        value={spawn[i]}
                        onChange={(e) =>
                          setSpawn((values) =>
                            values.map((v, j) =>
                              i === j ? e.target.value : v,
                            ),
                          )
                        }
                      />
                    </label>
                  ))}
                </div>
              </>
            )}
          </fieldset>
          <fieldset disabled={!catalog || busy || previewing}>
            <legend>Biome replacements</legend>
            <p className="world-export-settings-note">
              Replacements apply once to biome IDs. Block textures and blocks
              remain unchanged.
            </p>
            {biomes.map((row, i) => (
              <div className="world-copy-biome" key={i}>
                <label>
                  <span>Source biome ID</span>
                  <StudioInput
                    aria-label={"Source biome " + (i + 1)}
                    inputMode="numeric"
                    value={row.source}
                    onChange={(e) =>
                      setBiomes((values) =>
                        values.map((v, j) =>
                          i === j ? { ...v, source: e.target.value } : v,
                        ),
                      )
                    }
                  />
                </label>
                <label>
                  <span>Replace with</span>
                  <select
                    aria-label={"Target biome " + (i + 1)}
                    value={row.target}
                    onChange={(e) =>
                      setBiomes((values) =>
                        values.map((v, j) =>
                          i === j ? { ...v, target: e.target.value } : v,
                        ),
                      )
                    }
                  >
                    <option value="">Choose biome</option>
                    {catalog?.biomes.map((b) => (
                      <option key={b.id} value={b.id}>
                        {b.name} · {b.id}
                      </option>
                    ))}
                  </select>
                </label>
                <button
                  type="button"
                  className="studio-button"
                  onClick={() =>
                    setBiomes((values) => values.filter((_, j) => j !== i))
                  }
                >
                  Remove
                </button>
              </div>
            ))}
            <button
              type="button"
              className="studio-button"
              onClick={() =>
                setBiomes((values) => [...values, { source: "", target: "" }])
              }
            >
              Add biome replacement
            </button>
          </fieldset>
          <fieldset disabled={!catalog || busy || previewing}>
            <legend>Storage</legend>
            <label className="world-export-spawn-check">
              <StudioInput
                type="checkbox"
                aria-label="Compact export database"
                checked={compact}
                onChange={(e) => setCompact(e.target.checked)}
              />
              <span>Compact export database</span>
            </label>
            <p className="world-export-settings-note">
              Can reduce database size after processing. Compaction takes
              additional time.
            </p>
          </fieldset>
          <section
            className="world-copy-preview"
            aria-label="Chunk export preview"
          >
            <div className="world-copy-preview-header">
              <strong>Chunk preview</strong>
              <select
                aria-label="Preview dimension"
                value={mapDimension}
                onChange={(e) => setMapDimension(Number(e.target.value))}
              >
                {catalog?.dimensions.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </div>
            <svg
              role="img"
              aria-label="Included and excluded chunks"
              viewBox={`${minX - 1} ${minZ - 1} ${maxX - minX + 3} ${maxZ - minZ + 3}`}
              preserveAspectRatio="xMidYMid meet"
            >
              <path className="world-copy-kept" d={chunkPath(true)} />
              <path className="world-copy-excluded" d={chunkPath(false)} />
            </svg>
            <div className="world-copy-legend">
              <span>■ Included</span>
              <span>■ Excluded</span>
            </div>
            {plan && (
              <p>
                {plan.keptChunks.toLocaleString()} included ·{" "}
                {plan.excludedChunks.toLocaleString()} excluded
                {!isValid ? " · Preview needs updating" : ""}
              </p>
            )}
            {shown?.previewTruncated && (
              <p className="world-export-settings-note">
                Preview shows the first 50,000 chunks. Counts include the
                complete world.
              </p>
            )}
          </section>
          {busy && (
            <div className="world-copy-progress" role="status">
              <span>{progress?.stage || "Preparing export"}</span>
              <progress
                max={progress?.total || 1}
                value={progress?.total ? progress.completed : undefined}
              />
              <button
                className="studio-button"
                onClick={() => void api.CancelWorldExport()}
              >
                Cancel export
              </button>
            </div>
          )}
        </div>
        <DialogFooter>
          <button className="studio-button" disabled={busy} onClick={onClose}>
            Close
          </button>
          <button
            className="studio-button"
            disabled={!catalog || busy || previewing}
            onClick={() => void preview()}
          >
            {previewing ? "Checking…" : "Preview export"}
          </button>
          <button
            className="studio-button primary"
            disabled={!isValid || busy || previewing}
            onClick={() => void exportCopy()}
          >
            {busy ? "Exporting…" : "Export world copy"}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

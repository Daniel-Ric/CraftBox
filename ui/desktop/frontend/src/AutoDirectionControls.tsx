import HeaderActionIcon from "./HeaderActionIcon";
import FeedbackFlyout from "./FeedbackFlyout";
import StudioInput from "@/StudioInput";
import { useEffect, useRef, useState } from "react";
import { WandSparkles, LoaderCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  directModel,
  type ModelAnalysis,
  type DirectionResult,
  type DirectionOptions,
} from "./auto-direction";
import type { RenderSettings } from "./render-settings";
import { formatDaytime } from "./render-daytime";
import { sceneThemeLabels } from "./scene-context";
import "./auto-direction.css";
import {
  analysisYield,
  checkAnalysisAbort,
  type AnalysisProgress,
  type AnalysisProgressCallback,
} from "./scene-analysis";

const names: Record<string, string> = {
  pitchLocked: "Lock vertical rotation",
  timeOfDay: "Time of day",
  timeLinked: "Time-of-day link",
  azimuth: "Sun direction",
  elevation: "Light elevation",
  sun: "Sunlight",
  ambient: "Ambient light",
  exposure: "Exposure",
  ao: "Ambient occlusion",
  shadow: "Shadow resolution",
  background: "Background color",
  transparent: "Transparent background",
  edge: "Output size",
  aspect: "Aspect ratio",
  sky: "Sky",
  skyRotation: "Sky rotation",
  skyBrightness: "Sky brightness",
  environment: "Sky lighting",
  warmth: "Light color",
  softness: "Shadow softness",
  aoRadius: "Contact shadow radius",
  quality: "Quality",
  sunLinked: "Align sun with HDR",
  skyElevation: "Horizon correction",
  reflections: "Reflections",
  roughness: "Roughness",
  bloom: "Bloom",
  saturation: "Saturation",
  vibrance: "Vibrance",
  contrast: "Contrast",
  temperature: "Temperature",
  tint: "Tint",
  highlights: "Highlights",
  shadows: "Shadows",
  sharpen: "Sharpness",
  vignette: "Vignette",
  grading: "Color grading",
  blockLights: "Block lighting",
  blockDaylightAdaptation: "Adapt block lighting to daylight",
  blockLightPower: "Block light intensity",
  blockLightRange: "Block light range",
  blockLightShadows: "Block light shadows",
  lightBudget: "Light budget",
  emissionPower: "Surface emission",
  volumeLightPower: "Volumetric lighting",
  studioSpot: "Cinematic light",
  spotPower: "Cinematic light intensity",
  spotColor: "Cinematic light color",
  spotX: "Cinematic light X",
  spotY: "Cinematic light Y",
  spotZ: "Cinematic light Z",
  spotTargetX: "Look-at target X",
  spotTargetY: "Look-at target Y",
  spotTargetZ: "Look-at target Z",
  spotAngle: "Beam angle",
  spotSoftness: "Beam edge",
  fog: "Fog",
  fogColor: "Fog color",
  fogDensity: "Fog density",
  fogHeight: "Fog height",
  fogFalloff: "Fog falloff",
  fogVariation: "Fog variation",
  fogSize: "Wisp size",
  fogStart: "Fog near distance",
  fogScattering: "Fog scattering",
  fogAnisotropy: "Scattering direction",
  clouds: "Clouds",
  cloudType: "Cloud type",
  cloudModel: "Cloud model",
  cloudColor: "Cloud color",
  cloudDensity: "Cloud density",
  cloudHeight: "Cloud height",
  cloudThickness: "Cloud thickness",
  cloudSize: "Cloud size",
  coverage: "Coverage",
  cloudShadows: "Cloud shadows",
  windSpeed: "Wind speed",
  windDirection: "Wind direction",
  animate: "Animate atmosphere",
  time: "Atmosphere time",
  detail: "Structure detail",
  erosion: "Edge erosion",
  turbulence: "Distortion",
  cloudReference: "Cloud height reference",
  fogReference: "Fog height reference",
  cloudSeed: "Cloud distribution",
  fogSeed: "Fog distribution",
  cloudClumping: "Cloud clustering",
  cloudRoundness: "Roundness",
  fogType: "Fog type",
  fogTurbulence: "Fog distortion",
  distance: "Visibility range",
  lightAbsorption: "Light absorption",
  cloudExtent: "Cloud belt extent",
  cloudAvoidance: "Keep clouds clear of the model",
  cloudClearance: "Cloud clearance around model",
};
const display = (v: unknown) =>
  typeof v === "boolean"
    ? v
      ? "On"
      : "Off"
    : typeof v === "number"
      ? Number(v.toFixed(3)).toLocaleString("en-US")
      : String(v);

export default function AutoDirectionControls({
  ready,
  settings,
  onApply,
  analyse,
}: {
  ready: boolean;
  settings: RenderSettings;
  onApply: (settings: RenderSettings) => void;
  analyse: (
    progress?: AnalysisProgressCallback,
    signal?: AbortSignal,
  ) => Promise<ModelAnalysis>;
}) {
  const [open, setOpen] = useState(false),
    [busy, setBusy] = useState(false),
    [progress, setProgress] = useState(0),
    [error, setError] = useState(""),
    [result, setResult] = useState<DirectionResult>(),
    [previous, setPrevious] = useState<RenderSettings>();
  const [intent, setIntent] = useState<DirectionOptions["intent"]>("auto"),
    [atmosphere, setAtmosphere] =
      useState<DirectionOptions["atmosphere"]>("auto"),
    [time, setTime] = useState("auto"),
    [hour, setHour] = useState(17.25),
    [resolution, setResolution] = useState(3840);
  const token = useRef(0),
    mounted = useRef(true);
  const [theme, setTheme] = useState<DirectionOptions["theme"]>("auto");
  const [autoCloudColor, setAutoCloudColor] = useState(false);
  const controller = useRef<AbortController | undefined>(undefined);
  const [status, setStatus] = useState<AnalysisProgress>({
    stage: "prepare",
    label: "Preparing scene analysis",
  });
  const [stages, setStages] = useState<string[]>([]);
  const [elapsed, setElapsed] = useState(0);
  useEffect(() => {
    if (!busy) return;
    const started = performance.now();
    setElapsed(0);
    const timer = setInterval(
      () => setElapsed((performance.now() - started) / 1000),
      250,
    );
    return () => clearInterval(timer);
  }, [busy]);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      token.current++;
      controller.current?.abort();
    };
  }, []);
  const cancel = () => {
    controller.current?.abort();
    token.current++;
    setBusy(false);
  };
  const run = async () => {
    const id = ++token.current;
    setOpen(true);
    setBusy(true);
    setError("");
    setProgress(0);
    setResult(undefined);
    setStatus({ stage: "prepare", label: "Preparing scene analysis" });
    setStages([]);
    controller.current?.abort();
    const runController = new AbortController();
    controller.current = runController;
    try {
      const evidence = await analyse((v, update) => {
        if (mounted.current && id === token.current) {
          setProgress((old) => Math.max(old, Math.min(0.93, v)));
          if (update) {
            setStatus(update);
            setStages((old) =>
              old.at(-1) === update.label
                ? old
                : [...old.slice(-5), update.label],
            );
          }
        }
      }, runController.signal);
      if (!mounted.current || id !== token.current) return;
      checkAnalysisAbort(runController.signal);
      setProgress(0.95);
      setStatus({
        stage: "matching",
        label: "Matching scene looks and fitting clouds around the terrain",
      });
      await analysisYield();
      checkAnalysisAbort(runController.signal);
      const directed = directModel(evidence, {
        autoCloudColor,
        cloudColor: settings.atmosphere.cloudColor,
        theme,
        intent,
        atmosphere,
        hour: time === "fixed" ? hour : undefined,
        resolution,
      });
      setPrevious(structuredClone(settings));
      setResult(directed);
      onApply(directed.settings);
      setProgress(1);
    } catch (e) {
      if (mounted.current && id === token.current)
        setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (mounted.current && id === token.current) setBusy(false);
    }
  };
  return (
    <>
      <div className="auto-direction-launch">
        <Button title={busy ? "Analyzing model …" : "Auto art direction"} className="toolbar-icon-action"
          variant="outline"
          disabled={!ready || busy}
          onClick={() => void run()}
          aria-label="Auto-style scene"
        >
          <WandSparkles />
          <span className="toolbar-action-label">{busy ? "Analyzing model …" : "Auto art direction"}</span>
        </Button>
        <Button title="Advanced …" className="toolbar-icon-action"
          variant="ghost"
          disabled={!ready || busy}
          onClick={() => setOpen(true)}
          aria-label="Advanced auto styling"
        ><HeaderActionIcon label="Advanced …"/>
          <span className="toolbar-action-label">Advanced …</span></Button>
      </div>
      <Dialog
        open={open}
        onOpenChange={(value) => {
          if (!value) cancel();
          setOpen(value);
        }}
      >
        <DialogContent className="auto-direction-dialog">
          <DialogHeader>
            <DialogTitle>Auto art direction</DialogTitle>
            <DialogDescription>Automatic scene styling.</DialogDescription>
          </DialogHeader>
          <div className="auto-direction-body">
            <aside className="studio-sidebar auto-direction-options">
              <h3>Style settings</h3>
              <fieldset disabled={busy}>
                <label className="auto-cloud-tint">
                  <input
                    type="checkbox"
                    checked={autoCloudColor}
                    onChange={(e) => setAutoCloudColor(e.target.checked)}
                  />{" "}
                  Automatically match cloud color
                </label>
                <p className="auto-option-hint">
                  Optional. When off, your current cloud tint stays unchanged.
                </p>
                <label>
                  Scene character
                  <select
                    aria-label="Scene character"
                    value={theme}
                    onChange={(e) =>
                      setTheme(e.target.value as DirectionOptions["theme"])
                    }
                  >
                    <option value="auto">Use independent scene evidence</option>
                    <option value="none">No seasonal / event styling</option>
                    {Object.entries(sceneThemeLabels).map(([id, label]) => (
                      <option key={id} value={id}>
                        {label} (manual override)
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Scene mood
                  <select
                    aria-label="Scene mood"
                    value={intent}
                    onChange={(e) => setIntent(e.target.value as typeof intent)}
                  >
                    <option value="auto">Choose automatically</option>
                    <option value="vivid">Vibrant &amp; colorful</option>
                    <option value="cinematic">Cinematic &amp; dramatic</option>
                    <option value="neutral">Neutral &amp; clear</option>
                  </select>
                </label>
                <label>
                  Atmosphere
                  <select
                    aria-label="Automatic atmosphere"
                    value={atmosphere}
                    onChange={(e) =>
                      setAtmosphere(e.target.value as typeof atmosphere)
                    }
                  >
                    <option value="auto">Choose automatically</option>
                    <option value="clouds">Clouds below the map</option>
                    <option value="clear">Clear view</option>
                  </select>
                </label>
                <label>
                  Time of day
                  <select
                    aria-label="Automatic time of day"
                    value={time}
                    onChange={(e) => setTime(e.target.value)}
                  >
                    <option value="auto">Match the scene's mood</option>
                    <option value="fixed">Specified time</option>
                  </select>
                </label>
                {time === "fixed" && (
                  <label>
                    Time
                    <StudioInput
                      aria-label="Specified time"
                      type="time"
                      required
                      step={60}
                      value={formatDaytime(hour)}
                      onChange={(e) => {
                        if (e.target.value) {
                          const [h, m] = e.target.value.split(":").map(Number);
                          setHour(h + m / 60);
                        }
                      }}
                    />
                  </label>
                )}
                <label>
                  Output
                  <select
                    aria-label="Automatic output size"
                    value={resolution}
                    onChange={(e) => setResolution(+e.target.value)}
                  >
                    <option value={1920}>Full HD · 1920</option>
                    <option value={2560}>QHD · 2560</option>
                    <option value={3840}>Ultra HD · 3840</option>
                  </select>
                </label>
              </fieldset>
              <p>Local analysis. Fully editable.</p>
              <p>Replaces styling. Keeps cameras and keyframes.</p>
            </aside>
            <section
              className="auto-direction-report"
              aria-label="Model analysis"
            >
              {busy ? (
                <div className="auto-direction-progress" role="status">
                  <LoaderCircle className="spin" />
                  <h3>{status.label}</h3>
                  <progress
                    max={1}
                    value={progress}
                    aria-label="Auto styling progress"
                  />
                  <p>
                    {Math.round(progress * 100)} % ·{" "}
                    <span aria-live="off">{elapsed.toFixed(1)} s elapsed</span>
                  </p>
                  {status.detail && (
                    <p className="auto-analysis-detail">{status.detail}</p>
                  )}
                  <ol
                    className="auto-analysis-stages"
                    aria-label="Analysis steps"
                  >
                    {stages.slice(0, -1).map((label) => (
                      <li key={label}>✓ {label}</li>
                    ))}
                  </ol>
                </div>
              ) : result ? (
                <>
                  <h2>{result.title}</h2>
                  <div className="auto-context-report">
                    <strong>
                      {result.context.theme === "unknown"
                        ? "Season / event uncertain"
                        : `${sceneThemeLabels[result.context.theme]} evidence`}
                    </strong>
                    <p>
                      Block metadata{" "}
                      {result.context.hasBlockEvidence
                        ? "available"
                        : "unavailable"}
                      . Themes need independent supporting signals; color alone
                      is not a label.
                    </p>
                    {result.context.evidence.length > 0 && (
                      <ul>
                        {result.context.evidence.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    )}
                  </div>
                  <p className="auto-direction-applied">
                    Applied to the render view
                  </p>
                  <div
                    className="auto-direction-palette"
                    aria-label="Measured model palette"
                  >
                    {result.analysis.palette.map((c, i) => (
                      <span
                        key={i}
                        style={{ background: c }}
                        data-tooltip={c}
                      />
                    ))}
                  </div>
                  <dl>
                    <div>
                      <dt>Model dimensions</dt>
                      <dd>
                        {result.analysis.size
                          .map((x) => Math.round(x))
                          .join(" × ")}{" "}
                        blocks
                      </dd>
                    </div>
                    <div>
                      <dt>Analysis</dt>
                      <dd>
                        {result.analysis.samples.toLocaleString("en-US")}{" "}
                        Surface samples
                      </dd>
                    </div>
                    <div>
                      <dt>Light sources</dt>
                      <dd>{result.analysis.lights.toLocaleString("en-US")}</dd>
                    </div>
                    {result.cloudComposition && (
                      <div>
                        <dt>Cloud composition</dt>
                        <dd>
                          {result.cloudComposition.mode === "terrain"
                            ? "Terrain and camera fitted"
                            : "Curated cloud reference"}
                          {result.cloudComposition.candidates > 1 &&
                            ` · ${result.cloudComposition.candidates} layouts compared`}
                        </dd>
                      </div>
                    )}
                    <div>
                      <dt>Time of day</dt>
                      <dd>{formatDaytime(result.settings.timeOfDay)}</dd>
                    </div>
                  </dl>
                  <ul>
                    {result.reasons.map((reason) => (
                      <li key={reason}>{reason}</li>
                    ))}
                  </ul>
                  <details className="auto-alternatives">
                    <summary>Other suitable scene references</summary>
                    <ul>
                      {result.reference.alternatives.map((look) => (
                        <li key={look.id}>{look.name}</li>
                      ))}
                    </ul>
                  </details>
                  <details className="auto-direction-audit">
                    <summary>
                      All{" "}
                      {Object.keys(result.decisions).length -
                        1 +
                        Object.keys(result.atmosphereDecisions).length}{" "}
                      Styling decisions
                    </summary>
                    <table>
                      <thead>
                        <tr>
                          <th>Parameter</th>
                          <th>Value</th>
                          <th>Basis</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(result.settings)
                          .filter(([k]) => k !== "atmosphere")
                          .map(([k, v]) => (
                            <tr key={k}>
                              <td>{names[k] ?? k}</td>
                              <td>{display(v)}</td>
                              <td>
                                {result.decisions[k as keyof RenderSettings]}
                              </td>
                            </tr>
                          ))}
                        {Object.entries(result.settings.atmosphere).map(
                          ([k, v]) => (
                            <tr key={k}>
                              <td>{names[k] ?? k}</td>
                              <td>{display(v)}</td>
                              <td>{result.atmosphereDecisions[k]}</td>
                            </tr>
                          ),
                        )}
                      </tbody>
                    </table>
                  </details>
                </>
              ) : (
                <div className="auto-direction-intro">
                  <WandSparkles />
                  <h2>One model. One cohesive scene.</h2>
                  <p>Based on geometry, palette, materials and lights.</p>
                  <p>Includes output, shadows and atmosphere.</p>
                </div>
              )}
              {error && <FeedbackFlyout title="Auto art direction">{error}</FeedbackFlyout>}
            </section>
          </div>
          <DialogFooter>
            {busy ? (
              <Button variant="outline" onClick={cancel}>
                Cancel analysis
              </Button>
            ) : (
              <>
                {previous && (
                  <Button
                    variant="outline"
                    aria-label="Restore previous settings"
                    onClick={() => {
                      onApply(previous);
                      setPrevious(undefined);
                      setResult(undefined);
                    }}
                  >
                    Undo
                  </Button>
                )}
                <Button variant="outline" onClick={() => setOpen(false)}>
                  Close
                </Button>
                <Button disabled={!ready} onClick={() => void run()}>
                  <WandSparkles />
                  {result ? "Restyle" : "Analyze model & style scene"}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

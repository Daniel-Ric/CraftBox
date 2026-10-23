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
import "./auto-direction.css";

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
  analyse: (progress?: (value: number) => void) => Promise<ModelAnalysis>;
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
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      token.current++;
    };
  }, []);
  const cancel = () => {
    token.current++;
    setBusy(false);
  };
  const run = async () => {
    const id = ++token.current;
    setOpen(true);
    setBusy(true);
    setError("");
    setProgress(0);
    try {
      const evidence = await analyse((v) => {
        if (mounted.current && id === token.current) setProgress(v);
      });
      if (!mounted.current || id !== token.current) return;
      const directed = directModel(evidence, {
        intent,
        atmosphere,
        hour: time === "fixed" ? hour : undefined,
        resolution,
      });
      setPrevious(structuredClone(settings));
      setResult(directed);
      onApply(directed.settings);
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
        <Button
          variant="outline"
          disabled={!ready || busy}
          onClick={() => void run()}
          aria-label="Auto-style scene"
        >
          <WandSparkles />
          {busy ? "Analyzing model …" : "Auto art direction"}
        </Button>
        <Button
          variant="ghost"
          disabled={!ready || busy}
          onClick={() => setOpen(true)}
          aria-label="Advanced auto styling"
        >
          Advanced …
        </Button>
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
            <DialogDescription>
              Analyze the model and coordinate the entire scene's visual style.
            </DialogDescription>
          </DialogHeader>
          <div className="auto-direction-body">
            <aside className="auto-direction-options">
              <h3>Creative preferences</h3>
              <fieldset disabled={busy}>
                <label>
                  Creative intent
                  <select
                    aria-label="Automatic creative intent"
                    value={intent}
                    onChange={(e) => setIntent(e.target.value as typeof intent)}
                  >
                    <option value="auto">Infer from model</option>
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
                    <option value="auto">Infer from shape</option>
                    <option value="clouds">Sea of clouds around base</option>
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
                    <input
                      aria-label="Specified time"
                      type="time"
                      step={900}
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
              <p>
                Geometry and color are analyzed locally on this computer. The result is a creative recommendation that remains fully editable.
              </p>
              <p>
                This replaces all render settings. Camera position and keyframes are preserved.
              </p>
            </aside>
            <section
              className="auto-direction-report"
              aria-label="Model analysis"
            >
              {busy ? (
                <div className="auto-direction-progress" role="status">
                  <LoaderCircle className="spin" />
                  <h3>Analyze surfaces and materials</h3>
                  <progress max={1} value={progress} />
                  <p>
                    {Math.round(progress * 100)} % · then match scene looks and styling
                  </p>
                </div>
              ) : result ? (
                <>
                  <h2>{result.title}</h2>
                  <p className="auto-direction-applied">
                    Applied to the render view
                  </p>
                  <div
                    className="auto-direction-palette"
                    aria-label="Measured model palette"
                  >
                    {result.analysis.palette.map((c, i) => (
                      <span key={i} style={{ background: c }} title={c} />
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
                  <p>
                    The area-weighted palette, height distribution, transparency, vegetation, shape contrast and light sources guide lighting, materials, color and atmosphere.
                  </p>
                  <p>
                    Output format, shadow quality and all fog and cloud settings are also configured.
                  </p>
                </div>
              )}
              {error && <p role="alert">{error}</p>}
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
                  {result
                    ? "Restyle"
                    : "Analyze model & style scene"}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

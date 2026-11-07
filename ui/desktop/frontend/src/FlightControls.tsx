import StudioForm from "@/StudioForm";
import FeedbackFlyout from "./FeedbackFlyout";
import StudioInput from "@/StudioInput";
import { useEffect, useState, type RefObject } from "react";
import type { RenderCanvasHandle } from "./RenderCanvas";
import { Button } from "@/components/ui/button";
import { SlidersHorizontal } from "lucide-react";
import "./flight-controls.css";

export type FlightSettings = {
  speed: number;
  fov: number;
  sensitivity: number;
  level: boolean;
  smoothing: number;
};
export const flightDefaults: FlightSettings = {
  speed: 6,
  fov: 70,
  sensitivity: 0.0015,
  level: false,
  smoothing: 0.18,
};

export default function FlightControls({
  canvas,
  value,
  onChange,
  onExit,
  disabled,
}: {
  canvas: RefObject<RenderCanvasHandle | null>;
  value: FlightSettings;
  onChange: (v: FlightSettings) => void;
  onExit: () => void;
  disabled: boolean;
}) {
  const [position, setPosition] = useState<[number, number, number]>([0, 0, 0]),
    [destination, setDestination] = useState(["0", "80", "0"]),
    [expanded, setExpanded] = useState(false),
    [locked, setLocked] = useState(Boolean(document.pointerLockElement)),
    [message, setMessage] = useState("");
  useEffect(() => {
    const onLock = () => setLocked(Boolean(document.pointerLockElement));
    document.addEventListener("pointerlockchange", onLock);
    return () => document.removeEventListener("pointerlockchange", onLock);
  }, []);
  useEffect(() => {
    const timer = window.setInterval(() => {
      const p = canvas.current?.flightPosition?.();
      if (p) setPosition(p);
    }, 200);
    return () => clearInterval(timer);
  }, [canvas]);
  const valid = destination.every(
    (s) =>
      s.trim() !== "" &&
      Number.isFinite(Number(s)) &&
      Math.abs(Number(s)) <= 30000000,
  );
  return (
    <section
      className="flight-controls"
      aria-label="First-person controls"
      data-position={position.map((n) => n.toFixed(2)).join(",")}
    >
      <div className="flight-control-bar">
        <strong>Fly</strong>
        <span className="flight-speed">{value.speed.toFixed(1)} b/s</span>
        <StudioInput
          aria-label="Quick flight speed"
          className="flight-quick-speed"
          type="range"
          min="-1"
          max="2.4"
          step="0.05"
          value={Math.log10(value.speed)}
          disabled={disabled}
          onChange={(e) =>
            onChange({
              ...value,
              speed: Math.round(10 ** Number(e.target.value) * 10) / 10,
            })
          }
        />
        <Button
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() =>
            void canvas.current
              ?.lockMouse?.()
              .then((ok) =>
                setMessage(
                  ok ? "" : "Mouse lock unavailable. Right-drag to look.",
                ),
              )
          }
        >
          {locked ? "Mouse locked" : "Mouse lock"}
        </Button>
        <Button variant="ghost" size="sm" disabled={disabled} onClick={onExit}>
          Exit
        </Button>
        <Button
          variant="ghost"
          size="sm"
          aria-label="Flight settings"
          aria-expanded={expanded}
          onClick={() => setExpanded((v) => !v)}
        >
          <SlidersHorizontal size={14} />
        </Button>
      </div>
      {expanded && (
        <div className="flight-hud">
          <span>
            X {position[0].toFixed(1)} · Y {position[1].toFixed(1)} · Z{" "}
            {position[2].toFixed(1)}
          </span>
        </div>
      )}
      <p className="flight-help">
        WASD · Space/Q · Shift boost · Alt slow · Tab controls · Esc exit
        {!locked && " · Right-drag to look"}
      </p>
      {message && <FeedbackFlyout title="Mouse lock" tone="warning">{message}</FeedbackFlyout>}
      {expanded && (
        <fieldset disabled={disabled}>
          <label>
            Speed{" "}
            <StudioInput
              aria-label="Flight speed"
              type="number"
              min="0.1"
              max="4096"
              increment={0.1}
              value={value.speed}
              onChange={(e) => {
                const n = Number(e.target.value);
                if (n >= 0.1 && n <= 4096) onChange({ ...value, speed: n });
              }}
            />
          </label>
          <label>
            Field of view{" "}
            <StudioInput
              aria-label="Flight field of view"
              type="range"
              min="1"
              max="150"
              step="0.1"
              value={value.fov}
              onChange={(e) =>
                onChange({ ...value, fov: Number(e.target.value) })
              }
            />
            <span>{Number(value.fov.toFixed(1))}°</span>
          </label>
          <label>
            Mouse sensitivity{" "}
            <StudioInput
              aria-label="Flight mouse sensitivity"
              type="range"
              min="0.0005"
              max="0.01"
              step="0.0005"
              value={value.sensitivity}
              onChange={(e) =>
                onChange({ ...value, sensitivity: Number(e.target.value) })
              }
            />
          </label>
          <label className="flight-level">
            Movement smoothing
            <StudioInput
              aria-label="Flight smoothing"
              type="range"
              min="0"
              max="0.5"
              step="0.02"
              value={value.smoothing}
              onChange={(e) =>
                onChange({ ...value, smoothing: Number(e.target.value) })
              }
            />
          </label>
          <label className="flight-level">
            <StudioInput
              aria-label="Level flight"
              type="checkbox"
              checked={value.level}
              onChange={(e) => onChange({ ...value, level: e.target.checked })}
            />
            Level flight
          </label>
          <StudioForm
            onSubmit={(e) => {
              e.preventDefault();
              if (valid)
                canvas.current?.teleport?.(
                  destination.map(Number) as [number, number, number],
                );
            }}
          >
            <div className="flight-xyz">
              {["X", "Y", "Z"].map((axis, i) => (
                <label key={axis}>
                  {axis}
                  <StudioInput
                    aria-label={`Fly to ${axis}`}
                    type="number"
                    min={-30000000}
                    max={30000000}
                    step="any"
                    value={destination[i]}
                    onChange={(e) =>
                      setDestination((v) =>
                        v.map((n, j) => (i === j ? e.target.value : n)),
                      )
                    }
                  />
                </label>
              ))}
            </div>
            <div className="flight-actions">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() =>
                  setDestination(position.map((n) => n.toFixed(1)))
                }
              >
                Current
              </Button>
              <Button type="submit" size="sm" disabled={!valid}>
                Go
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => canvas.current?.flightReturn?.()}
              >
                Return to start
              </Button>
            </div>
          </StudioForm>
          <p>Scroll: speed · No collision</p>
        </fieldset>
      )}
    </section>
  );
}

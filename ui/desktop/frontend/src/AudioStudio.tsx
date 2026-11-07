import FeedbackFlyout from "./FeedbackFlyout";
import { useDocumentScope } from "./document-scope";
import { memo, useEffect, useRef, useState } from "react";
import { api, type AssetFile, type AssetFilePreview } from "./bridge";
import { AudioTransport } from "./audio-transport";
import {
  audioChannels,
  arrangeAudio,
  audioTime,
  parseTime,
  wavePeaks,
  type AudioClip,
} from "./audio-model";
import { Button } from "@/components/ui/button";
import { Play, Pause, Square, Volume2, Trash2, Headphones } from "lucide-react";
import "./audio-studio.css";

export type AudioWorkspaceState = {
  clips: Omit<AudioClip, "id" | "duration">[];
  position: number;
  speed: number;
  master: number;
  zoom: number;
  loop: boolean;
  rangeStart: number;
  rangeEnd: number;
};
type LoadedClip = AudioClip & {
  peaks: Float32Array[];
  channels: number;
  sampleRate: number;
};
const palette = ["#55b8ee", "#b197ef", "#e2aa63", "#66c5b3", "#de85aa"];
const Waveform = memo(function Waveform(p: {
  clip: LoadedClip;
  color: string;
  width: number;
}) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const w = Math.max(1, Math.min(8192, Math.ceil(p.width))),
      h = 76;
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.fillStyle = p.color;
    ctx.globalAlpha = 0.8;
    for (let channel = 0; channel < p.clip.peaks.length; channel++) {
      const data = p.clip.peaks[channel],
        bins = data.length / 2,
        band = h / p.clip.peaks.length,
        center = band * (channel + 0.5);
      for (let x = 0; x < w; x++) {
        const from = Math.floor(
            ((p.clip.trimStart +
              (x / w) * (p.clip.trimEnd - p.clip.trimStart)) /
              p.clip.duration) *
              bins,
          ),
          to = Math.min(
            bins,
            Math.max(
              from + 1,
              Math.ceil(
                ((p.clip.trimStart +
                  ((x + 1) / w) * (p.clip.trimEnd - p.clip.trimStart)) /
                  p.clip.duration) *
                  bins,
              ),
            ),
          );
        let low = 0,
          high = 0;
        for (let b = from; b < to; b++) {
          low = Math.min(low, data[b * 2] ?? 0);
          high = Math.max(high, data[b * 2 + 1] ?? 0);
        }
        ctx.fillRect(
          x,
          center - high * band * 0.44,
          1,
          Math.max(1, (high - low) * band * 0.44),
        );
      }
    }
  }, [p.clip.peaks, p.clip.trimStart, p.clip.trimEnd, p.width, p.color]);
  return (
    <canvas
      ref={ref}
      role="img"
      aria-label={`${p.clip.channels} channel waveform for ${p.clip.path}`}
      style={{ width: "100%", height: 76 }}
    />
  );
});
async function metadataDuration(url: string, signal: AbortSignal) {
  return new Promise<number>((resolve, reject) => {
    const audio = new Audio();
    const finish = (error?: string) => {
      clearTimeout(timer);
      signal.removeEventListener("abort", abort);
      audio.onloadedmetadata = null;
      audio.onerror = null;
      const duration = audio.duration;
      audio.removeAttribute("src");
      audio.load();
      error ? reject(new Error(error)) : resolve(duration);
    };
    const abort = () => finish("Audio load cancelled");
    const timer = setTimeout(
      () => finish("Audio metadata could not be read in time"),
      12000,
    );
    audio.preload = "metadata";
    audio.onloadedmetadata = () => finish();
    audio.onerror = () => finish("This audio format could not be decoded");
    signal.addEventListener("abort", abort, { once: true });
    if (signal.aborted) {
      abort();
      return;
    }
    audio.src = url;
  });
}
export default function AudioStudio(p: {
  assetID: string;
  files: AssetFile[];
  available: AssetFile[];
  initial?: AssetFilePreview;
  initialState?: AudioWorkspaceState;
  onStateChange?: (value: AudioWorkspaceState) => void;
  onLoading?: (loading: boolean) => void;
}) {
  const scope = useDocumentScope();
  const restored = useRef(p.initialState),
    initialized = useRef(false),
    userPosition = useRef(p.initialState?.position || 0),
    stateChanged = useRef(p.onStateChange);
  stateChanged.current = p.onStateChange;
  const engine = useRef<AudioTransport | null>(null),
    timeline = useRef<HTMLDivElement | null>(null),
    simplePlayer = useRef<HTMLAudioElement | null>(null),
    buffers = useRef(new Map<string, AudioBuffer>()),
    current = useRef<LoadedClip[]>([]),
    loadAbort = useRef<AbortController | null>(null),
    drag = useRef<{
      id: string;
      x: number;
      start: number;
      pointer: number;
      zoom: number;
    } | null>(null),
    generation = useRef(0);
  const [clips, setClips] = useState<LoadedClip[]>([]),
    [errors, setErrors] = useState<string[]>([]),
    [loading, setLoading] = useState(false),
    [progress, setProgress] = useState(""),
    [playing, setPlaying] = useState(false),
    [position, setPosition] = useState(p.initialState?.position ?? 0),
    [speed, setSpeed] = useState(p.initialState?.speed ?? 1),
    [master, setMaster] = useState(p.initialState?.master ?? 0.8),
    [zoom, setZoom] = useState(p.initialState?.zoom ?? 50),
    [loop, setLoop] = useState(p.initialState?.loop ?? false),
    [rangeStart, setRangeStart] = useState(p.initialState?.rangeStart ?? 0),
    [rangeEnd, setRangeEnd] = useState(p.initialState?.rangeEnd ?? 0),
    [addPath, setAddPath] = useState(""),
    [seekText, setSeekText] = useState("");
  const [simpleData, setSimpleData] = useState("");
  useEffect(() => {
    const player = simplePlayer.current;
    return () => {
      if (player) {
        player.pause();
        player.removeAttribute("src");
        player.load();
      }
    };
  }, [simpleData]);
  current.current = clips;
  useEffect(() => {
    const move = (event: PointerEvent) => {
      const active = drag.current;
      if (!active || active.pointer !== event.pointerId) return;
      const start =
        Math.round(
          Math.max(
            0,
            Math.min(
              7200,
              active.start + (event.clientX - active.x) / active.zoom,
            ),
          ) * 100,
        ) / 100;
      setClips((old) =>
        old.map((c) => (c.id === active.id ? { ...c, start } : c)),
      );
    };
    const end = () => {
      drag.current = null;
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", end);
    window.addEventListener("pointercancel", end);
    window.addEventListener("blur", end);
    return () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", end);
      window.removeEventListener("pointercancel", end);
      window.removeEventListener("blur", end);
    };
  }, []);
  const duration = Math.max(
      0,
      ...clips.map((c) => c.start + c.trimEnd - c.trimStart),
    ),
    end = rangeEnd > rangeStart ? Math.min(duration, rangeEnd) : duration;
  const controls = useRef({ loop, rangeStart, end, speed });
  controls.current = { loop, rangeStart, end, speed };
  const pause = () => {
    const e = engine.current;
    if (e) {
      const time = e.time();
      e.stop(time);
      setPosition(time);
    }
    setPlaying(false);
  };
  const seek = (time: number) => {
    const e = engine.current,
      value = Math.max(0, Math.min(duration, time));
    userPosition.current = value;
    setPosition(value);
    if (e?.playing)
      void e
        .play(current.current, buffers.current, value, end, speed)
        .catch(fail);
    else e?.stop(value);
  };
  const fail = (e: unknown) => {
    engine.current?.stop();
    setPlaying(false);
    setErrors((old) => [...old.slice(-9), String(e)]);
  };
  const play = () => {
    simplePlayer.current?.pause();
    const e = engine.current;
    if (!e || !clips.length || end <= rangeStart) return;
    const from =
      position >= end || position < rangeStart ? rangeStart : position;
    void e
      .play(clips, buffers.current, from, end, speed)
      .then(() => setPlaying(e.playing))
      .catch(fail);
  };
  useEffect(() => {
    const e = new AudioTransport();
    engine.current = e;
    let frame = 0,
      last = 0;
    const tick = (now: number) => {
      if (e.playing && now - last > 30) {
        last = now;
        let time = e.time();
        const c = controls.current;
        if (time >= c.end) {
          if (c.loop && c.end > c.rangeStart) {
            time = c.rangeStart;
            void e
              .play(current.current, buffers.current, time, c.end, c.speed)
              .catch(fail);
          } else {
            time = c.end;
            e.stop(time);
            setPlaying(false);
          }
        }
        setPosition(Math.max(0, time));
      }
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => {
      generation.current++;
      loadAbort.current?.abort();
      cancelAnimationFrame(frame);
      e.close();
      engine.current = null;
      buffers.current.clear();
    };
  }, []);
  useEffect(() => {
    if (engine.current)
      engine.current.master.gain.setTargetAtTime(
        master,
        engine.current.context.currentTime,
        0.015,
      );
  }, [master]);
  const load = async (files: AssetFile[], append: boolean) => {
    const e = engine.current;
    if (!e) return;
    const run = ++generation.current;
    loadAbort.current?.abort();
    const controller = new AbortController();
    loadAbort.current = controller;
    e.stop();
    setPlaying(false);
    setPosition(0);
    setLoading(true);
    setErrors([]);
    setSimpleData("");
    if (!append) {
      buffers.current.clear();
      setClips([]);
    }
    const loaded: LoadedClip[] = append ? [...current.current] : [];
    let bytes = [...buffers.current.values()].reduce(
      (n, b) => n + b.length * b.numberOfChannels * 4,
      0,
    );
    if (files.length + loaded.length > 64)
      setErrors([
        "A session holds up to 64 clips. Open a smaller group or add a selection of files.",
      ]);
    for (const file of files.slice(0, Math.max(0, 64 - loaded.length))) {
      if (run !== generation.current) return;
      setProgress(file.path);
      try {
        if (file.bytes > 16 * 1024 * 1024)
          throw Error("File exceeds the 16 MiB waveform limit");
        const preview =
          p.initial?.path === file.path
            ? p.initial
            : await api.ReadAssetFile(p.assetID, file.path);
        if (run !== generation.current) return;
        const seconds = await metadataDuration(preview.data, controller.signal);
        if (!Number.isFinite(seconds) || seconds <= 0 || seconds > 600)
          throw Error(
            "Audio exceeds the session waveform memory limit; choose fewer or shorter clips",
          );
        if (run !== generation.current) return;
        const data = await (
          await fetch(preview.data, { signal: controller.signal })
        ).arrayBuffer();
        if (run !== generation.current) return;
        const channels = audioChannels(new Uint8Array(data));
        if (
          channels > 32 ||
          bytes + (seconds + 0.1) * e.context.sampleRate * channels * 4 >
            128 * 1024 * 1024
        )
          throw Error(
            "Audio exceeds the session waveform memory limit; choose fewer or shorter clips",
          );
        const buffer = await e.context.decodeAudioData(data);
        if (run !== generation.current) return;
        const size = buffer.length * buffer.numberOfChannels * 4;
        if (bytes + size > 128 * 1024 * 1024)
          throw Error("Decoded audio exceeds 128 MiB");
        bytes += size;
        const id = `clip-${run}-${loaded.length}`,
          start = loaded.reduce(
            (n, c) => Math.max(n, c.start + c.trimEnd - c.trimStart),
            0,
          );
        buffers.current.set(id, buffer);
        const saved = !append
          ? restored.current?.clips[loaded.length]
          : undefined;
        loaded.push({
          id,
          path: file.path,
          duration: buffer.duration,
          start,
          trimStart: 0,
          trimEnd: buffer.duration,
          gain: 0,
          pan: 0,
          mute: false,
          solo: false,
          ...(saved && saved.path === file.path
            ? {
                start: Math.max(0, saved.start),
                trimStart: Math.max(
                  0,
                  Math.min(buffer.duration, saved.trimStart),
                ),
                trimEnd: Math.max(0, Math.min(buffer.duration, saved.trimEnd)),
                gain: saved.gain,
                pan: saved.pan,
                mute: saved.mute,
                solo: saved.solo,
              }
            : {}),
          channels: buffer.numberOfChannels,
          sampleRate: buffer.sampleRate,
          peaks: Array.from(
            { length: Math.min(2, buffer.numberOfChannels) },
            (_, i) => wavePeaks(buffer.getChannelData(i)),
          ),
        });
        setClips([...loaded]);
      } catch (error) {
        if (run === generation.current)
          setErrors((old) => [
            ...old.slice(-9),
            `${file.path}: ${String(error)}`,
          ]);
      }
    }
    if (run === generation.current) {
      initialized.current = true;
      setLoading(false);
      setProgress("");
      setRangeStart(!append ? restored.current?.rangeStart || 0 : 0);
      setRangeEnd(!append ? restored.current?.rangeEnd || 0 : 0);
      setPosition(!append ? restored.current?.position || 0 : 0);
    }
  };
  useEffect(() => {
    const savedFiles = restored.current?.clips
      .map((c) => p.available.find((f) => f.path === c.path))
      .filter((f): f is AssetFile => !!f);
    void load(savedFiles ?? p.files, false);
    return () => {
      generation.current++;
      loadAbort.current?.abort();
    };
  }, [p.assetID, p.files]);
  useEffect(() => {
    p.onLoading?.(loading);
    return () => p.onLoading?.(false);
  }, [loading, p.onLoading]);
  useEffect(() => {
    if (!scope.active) {
      pause();
      simplePlayer.current?.pause();
    }
  }, [scope.active]);
  const state: AudioWorkspaceState = {
    clips: clips.map(
      ({ path, start, trimStart, trimEnd, gain, pan, mute, solo }) => ({
        path,
        start,
        trimStart,
        trimEnd,
        gain,
        pan,
        mute,
        solo,
      }),
    ),
    position: playing ? userPosition.current : position,
    speed,
    master,
    zoom,
    loop,
    rangeStart,
    rangeEnd,
  };
  const stateKey = JSON.stringify(state);
  useEffect(() => {
    if (initialized.current && !loading) stateChanged.current?.(state);
  }, [stateKey, loading]);
  const timing = JSON.stringify(
    clips.map((c) => [c.id, c.start, c.trimStart, c.trimEnd]),
  );
  useEffect(() => {
    const e = engine.current;
    if (e?.playing)
      void e
        .play(
          current.current,
          buffers.current,
          Math.min(e.time(), end),
          end,
          speed,
        )
        .catch(fail);
  }, [timing, speed, rangeStart, end]);
  useEffect(() => {
    engine.current?.mix(clips);
  }, [clips]);
  useEffect(() => {
    setRangeStart((v) => Math.min(v, Math.max(0, duration - 0.01)));
    setRangeEnd((v) => (v ? Math.min(v, duration) : 0));
  }, [duration]);
  const patch = (id: string, value: Partial<AudioClip>) =>
    setClips((old) => old.map((c) => (c.id === id ? { ...c, ...value } : c)));
  const step =
    zoom < 4 ? 60 : zoom < 10 ? 30 : zoom < 40 ? 10 : zoom < 100 ? 5 : 1;
  const ticks = Math.min(250, Math.ceil(Math.max(1, duration) / step)),
    width = Math.max(200, duration * zoom + 40);
  return (
    <section
      className="audio-studio"
      aria-label="Audio workstation"
      onKeyDown={(event) => {
        if (
          event.code === "Space" &&
          !(event.target as HTMLElement).closest("input,select,button")
        ) {
          event.preventDefault();
          playing ? pause() : play();
        }
      }}
      tabIndex={0}
    >
      <header>
        <div>
          <Headphones size={18} />
          <strong>Audio studio</strong>
          <span>
            {clips.length} clips · {audioTime(duration)}
          </span>
        </div>
        <p>Arrange clips, compare sounds and preview a mix.</p>
      </header>
      <div className="audio-transport">
        <Button
          disabled={!clips.length || loading}
          onClick={() => (playing ? pause() : play())}
        >
          {playing ? <Pause size={15} /> : <Play size={15} />}{" "}
          {playing ? "Pause" : "Play"}
        </Button>
        <Button
          variant="ghost"
          aria-label="Stop audio"
          onClick={() => {
            engine.current?.stop(rangeStart);
            setPlaying(false);
            setPosition(rangeStart);
          }}
        >
          <Square size={14} />
        </Button>
        <output aria-label="Audio position">
          {audioTime(position)} / {audioTime(duration)}
        </output>
        <input
          className="audio-seek-input"
          aria-label="Seek to time"
          placeholder="Go to 0:00"
          value={seekText}
          onChange={(e) => setSeekText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              const time = parseTime(seekText);
              if (time !== null) seek(time);
            }
          }}
        />
        <label>
          Speed
          <input
            aria-label="Audio speed"
            type="number"
            min={0.25}
            max={2}
            step={0.25}
            value={speed}
            onChange={(e) =>
              setSpeed(Math.max(0.25, Math.min(2, +e.target.value || 1)))
            }
          />
        </label>
        <label>
          <Volume2 size={14} />
          <input
            aria-label="Master volume"
            type="range"
            min={0}
            max={1}
            step={0.01}
            value={master}
            onChange={(e) => setMaster(+e.target.value)}
          />
        </label>
      </div>
      <div className="audio-session-tools">
        <label>
          Zoom
          <input
            aria-label="Waveform zoom"
            type="range"
            min={1}
            max={240}
            value={zoom}
            onChange={(e) => setZoom(+e.target.value)}
          />
        </label>
        <Button
          variant="ghost"
          onClick={() =>
            setZoom(
              Math.max(
                1,
                Math.min(
                  240,
                  ((timeline.current?.clientWidth ?? 800) - 270) /
                    Math.max(duration, 0.1),
                ),
              ),
            )
          }
        >
          Fit sequence
        </Button>
        <label>
          <input
            type="checkbox"
            checked={loop}
            onChange={(e) => setLoop(e.target.checked)}
          />
          Loop
        </label>
        <label>
          In
          <input
            aria-label="Loop in"
            type="number"
            min={0}
            max={duration}
            step={0.1}
            value={rangeStart}
            onChange={(e) =>
              setRangeStart(
                Math.min(Math.max(0, +e.target.value), Math.max(0, end - 0.01)),
              )
            }
          />
        </label>
        <label>
          Out
          <input
            aria-label="Loop out"
            type="number"
            min={0}
            max={duration}
            step={0.1}
            value={rangeEnd || duration}
            onChange={(e) =>
              setRangeEnd(
                Math.max(
                  rangeStart + 0.01,
                  Math.min(duration, +e.target.value),
                ),
              )
            }
          />
        </label>
        <Button
          variant="ghost"
          disabled={loading}
          onClick={() => setClips((old) => arrangeAudio(old, "sequence"))}
        >
          Sequence
        </Button>
        <Button
          variant="ghost"
          disabled={loading}
          onClick={() => setClips((old) => arrangeAudio(old, "parallel"))}
        >
          Layer sounds
        </Button>
      </div>
      {loading && <p role="status">Loading waveforms… {progress}</p>}
      {p.files.length === 1 && (
        <div className="audio-simple-player">
          <Button
            variant="ghost"
            disabled={loading}
            onClick={async () => {
              const run = generation.current;
              try {
                const preview =
                  p.initial?.path === p.files[0].path
                    ? p.initial
                    : await api.ReadAssetFile(p.assetID, p.files[0].path);
                if (run === generation.current) setSimpleData(preview.data);
              } catch (error) {
                if (run === generation.current) fail(error);
              }
            }}
          >
            Open simple player
          </Button>
          {simpleData && (
            <audio
              ref={simplePlayer}
              aria-label="Single file audio player"
              controls
              src={simpleData}
              onPlay={() => {
                engine.current?.stop();
                setPlaying(false);
              }}
            />
          )}
        </div>
      )}
      {!!errors.length && (
        <FeedbackFlyout title="Audio Studio">
          {errors.map((error, i) => (
            <p key={i}>{error}</p>
          ))}
          <Button
            variant="ghost"
            disabled={loading}
            onClick={() => void load(p.files, false)}
          >
            Reload selection
          </Button>
        </FeedbackFlyout>
      )}
      <div className="audio-timeline-scroll" ref={timeline}>
        <div className="audio-timeline" style={{ width: width + 230 }}>
          <div className="audio-ruler">
            <span>Sound tracks</span>
            <div
              style={{ width }}
              onPointerDown={(e) => {
                const rect = e.currentTarget.getBoundingClientRect();
                seek((e.clientX - rect.left) / zoom);
              }}
            >
              {Array.from({ length: ticks + 1 }, (_, i) => (
                <span key={i} style={{ left: i * step * zoom }}>
                  {audioTime(i * step)}
                </span>
              ))}
            </div>
          </div>
          {clips.map((clip, index) => (
            <div
              className="audio-track"
              key={clip.id}
              style={
                {
                  "--track-color": palette[index % palette.length],
                } as React.CSSProperties
              }
            >
              <div className="audio-track-controls">
                <strong title={clip.path}>{clip.path.split("/").at(-1)}</strong>
                <small>
                  {clip.channels} ch · {Math.round(clip.sampleRate / 100) / 10}{" "}
                  kHz · {audioTime(clip.duration)}
                </small>
                <div>
                  <button
                    aria-label={`Mute ${clip.path}`}
                    aria-pressed={clip.mute}
                    onClick={() => patch(clip.id, { mute: !clip.mute })}
                  >
                    M
                  </button>
                  <button
                    aria-label={`Solo ${clip.path}`}
                    aria-pressed={clip.solo}
                    onClick={() => patch(clip.id, { solo: !clip.solo })}
                  >
                    S
                  </button>
                  <label>
                    Gain
                    <input
                      aria-label={`Gain ${clip.path}`}
                      type="range"
                      min={-60}
                      max={6}
                      step={0.5}
                      value={clip.gain}
                      onChange={(e) =>
                        patch(clip.id, { gain: +e.target.value })
                      }
                    />
                    <span>{clip.gain} dB</span>
                  </label>
                  <button
                    aria-label={`Remove ${clip.path}`}
                    disabled={loading}
                    onClick={() => {
                      buffers.current.delete(clip.id);
                      setClips((old) => old.filter((c) => c.id !== clip.id));
                    }}
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
                <label>
                  Pan
                  <input
                    aria-label={`Pan ${clip.path}`}
                    type="range"
                    min={-1}
                    max={1}
                    step={0.05}
                    value={clip.pan}
                    onChange={(e) => patch(clip.id, { pan: +e.target.value })}
                  />
                </label>
              </div>
              <div
                className="audio-track-lane"
                style={{ width, backgroundSize: `${step * zoom}px 100%` }}
                onPointerDown={(e) => {
                  if ((e.target as HTMLElement).closest("input,button")) return;
                  const rect = e.currentTarget.getBoundingClientRect();
                  seek((e.clientX - rect.left) / zoom);
                }}
              >
                <div
                  className={`audio-clip ${clip.mute ? "muted" : ""}`}
                  style={{
                    left: clip.start * zoom,
                    width: Math.max(2, (clip.trimEnd - clip.trimStart) * zoom),
                  }}
                >
                  <div className="audio-clip-name">
                    {index + 1}. {clip.path.split("/").at(-1)}
                    <button
                      className="audio-clip-drag"
                      aria-label={`Move clip ${clip.path}`}
                      title="Drag to move; arrow keys nudge"
                      onPointerDown={(e) => {
                        if (e.button !== 0) return;
                        e.preventDefault();
                        e.stopPropagation();
                        pause();
                        drag.current = {
                          id: clip.id,
                          x: e.clientX,
                          start: clip.start,
                          pointer: e.pointerId,
                          zoom,
                        };
                        e.currentTarget.setPointerCapture(e.pointerId);
                      }}
                      onPointerUp={(e) => {
                        if (e.currentTarget.hasPointerCapture(e.pointerId))
                          e.currentTarget.releasePointerCapture(e.pointerId);
                      }}
                      onKeyDown={(e) => {
                        if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
                          e.preventDefault();
                          patch(clip.id, {
                            start: Math.max(
                              0,
                              Math.min(
                                7200,
                                clip.start +
                                  (e.key === "ArrowLeft" ? -1 : 1) *
                                    (e.shiftKey ? 1 : 0.1),
                              ),
                            ),
                          });
                        }
                      }}
                    >
                      ↔
                    </button>
                  </div>
                  <Waveform
                    clip={clip}
                    color={palette[index % palette.length]}
                    width={(clip.trimEnd - clip.trimStart) * zoom}
                  />
                </div>
                <div
                  className="audio-playhead"
                  style={{ left: position * zoom }}
                />
              </div>
              <div className="audio-clip-edit">
                <label>
                  Start (s)
                  <input
                    aria-label={`Start ${clip.path}`}
                    type="number"
                    min={0}
                    max={7200}
                    step={0.1}
                    value={clip.start}
                    onChange={(e) =>
                      patch(clip.id, {
                        start: Math.max(0, Math.min(7200, +e.target.value)),
                      })
                    }
                  />
                </label>
                <label>
                  Trim in
                  <input
                    aria-label={`Trim in ${clip.path}`}
                    type="number"
                    min={0}
                    max={clip.trimEnd - 0.01}
                    step={0.01}
                    value={clip.trimStart}
                    onChange={(e) =>
                      patch(clip.id, {
                        trimStart: Math.min(
                          clip.trimEnd - 0.01,
                          Math.max(0, +e.target.value),
                        ),
                      })
                    }
                  />
                </label>
                <label>
                  Trim out
                  <input
                    aria-label={`Trim out ${clip.path}`}
                    type="number"
                    min={clip.trimStart + 0.01}
                    max={clip.duration}
                    step={0.01}
                    value={clip.trimEnd}
                    onChange={(e) =>
                      patch(clip.id, {
                        trimEnd: Math.max(
                          clip.trimStart + 0.01,
                          Math.min(clip.duration, +e.target.value),
                        ),
                      })
                    }
                  />
                </label>
              </div>
            </div>
          ))}
        </div>
      </div>
      <footer>
        <select
          aria-label="Add sound to session"
          value={addPath}
          onChange={(e) => setAddPath(e.target.value)}
        >
          <option value="">Add another sound…</option>
          {p.available
            .filter((f) => f.kind === "sound")
            .slice(0, 3000)
            .map((f) => (
              <option key={f.path} value={f.path}>
                {f.path}
              </option>
            ))}
        </select>
        <Button
          variant="outline"
          disabled={!addPath || loading || clips.length >= 64}
          onClick={() => {
            const f = p.available.find((f) => f.path === addPath);
            if (f) void load([f], true);
          }}
        >
          Add track
        </Button>
        <small>Click a waveform to seek. Speed also changes pitch.</small>
      </footer>
    </section>
  );
}

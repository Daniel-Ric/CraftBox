import { useRef } from "react";
import type { Sequence } from "./cinematic";
import {
  audibleTrack,
  clipDuration,
  dbGain,
  editAudioClipTiming,
  type AudioAsset,
  type AudioClip,
} from "./cinematic-audio";

function waveform(asset: AudioAsset, clip: AudioClip, gain: number) {
  const paths: string[] = [];
  for (let ch = 0; ch < asset.channels; ch++) {
    const peaks = asset.peaks[ch],
      bins = peaks.length / 2,
      height = 36 / asset.channels,
      center = 7 + height * (ch + 0.5);
    let path = "";
    for (let i = 0; i < 512; i++) {
      const time = clip.in + ((clip.out - clip.in) * i) / 511,
        index =
          Math.min(
            bins - 1,
            Math.max(0, Math.floor((time / asset.duration) * bins)),
          ) * 2;
      const low = Math.max(-1, (peaks[index] ?? 0) * gain),
        high = Math.min(1, (peaks[index + 1] ?? 0) * gain);
      path += `M${(i / 511) * 1000},${center - low * height * 0.45}L${(i / 511) * 1000},${center - high * height * 0.45}`;
    }
    paths.push(path);
  }
  return paths;
}
export default function CinematicAudioTimeline(p: {
  project: Sequence;
  time: number;
  selected?: string;
  zoom: number;
  snap: boolean;
  disabled: boolean;
  onSelect: (id: string) => void;
  onScrub: (time: number) => void;
  onChange: (project: Sequence, before?: Sequence) => void;
  onPreview: (project: Sequence) => void;
}) {
  const drag = useRef<{
      before: Sequence;
      next: Sequence;
      id: string;
      mode: "move" | "in" | "out";
      x: number;
      width: number;
    } | null>(null),
    audio = p.project.audio;
  return (
    <div className="cinematic-audio-timeline" aria-label="Audio timeline">
      {!audio?.tracks.length ? (
        <p>Import audio to view waveforms.</p>
      ) : (
        <div style={{ minWidth: `${p.zoom * 100}%` }}>
          <div className="animation-ruler">
            <span>Audio · sequence time</span>
            <div>
              {Array.from({ length: 9 }, (_, i) => (
                <button
                  key={i}
                  style={{ left: `${(i / 8) * 100}%` }}
                  onClick={() => p.onScrub((i / 8) * p.project.duration)}
                >
                  {((i / 8) * p.project.duration).toFixed(2)}s
                </button>
              ))}
            </div>
          </div>
          {audio.tracks.map((track) => (
            <div
              className={`cinematic-audio-row ${audibleTrack(audio, track) ? "" : "muted"}`}
              key={track.id}
            >
              <span>
                {track.name}
                <small>
                  {track.gainDb} dB ·{" "}
                  {track.solo ? "Solo" : track.muted ? "Mute" : "Stereo"}
                </small>
              </span>
              <div
                className="cinematic-audio-row-body"
                onPointerDown={(e) => {
                  if (e.target !== e.currentTarget || p.disabled) return;
                  const r = e.currentTarget.getBoundingClientRect();
                  p.onScrub(
                    ((e.clientX - r.left) / r.width) * p.project.duration,
                  );
                }}
              >
                {audio.clips
                  .filter((c) => c.trackId === track.id)
                  .map((clip) => {
                    const asset = audio.assets.find(
                      (a) => a.id === clip.assetId,
                    )!;
                    return (
                      <div
                        key={clip.id}
                        className={`cinematic-audio-clip ${p.selected === clip.id ? "selected" : ""} ${clip.muted ? "muted" : ""}`}
                        style={{
                          left: `${(clip.start / p.project.duration) * 100}%`,
                          width: `${(clipDuration(clip) / p.project.duration) * 100}%`,
                        }}
                      >
                        <button
                          aria-label={`Audio clip ${clip.name}`}
                          disabled={p.disabled}
                          className="cinematic-audio-clip-body"
                          onClick={() => p.onSelect(clip.id)}
                          onKeyDown={(e) => {
                            if (!["ArrowLeft", "ArrowRight"].includes(e.key))
                              return;
                            e.preventDefault();
                            e.stopPropagation();
                            p.onChange({
                              ...p.project,
                              audio: editAudioClipTiming(
                                audio,
                                clip.id,
                                "move",
                                (e.key === "ArrowLeft" ? -1 : 1) /
                                  p.project.fps,
                                p.project.fps,
                                p.snap,
                              ),
                            });
                          }}
                          onPointerDown={(e) => {
                            e.stopPropagation();
                            e.currentTarget.setPointerCapture(e.pointerId);
                            p.onSelect(clip.id);
                            drag.current = {
                              before: p.project,
                              next: p.project,
                              id: clip.id,
                              mode: "move",
                              x: e.clientX,
                              width:
                                e.currentTarget.parentElement!.parentElement!.getBoundingClientRect()
                                  .width,
                            };
                          }}
                          onPointerMove={(e) => {
                            const a = drag.current;
                            if (!a || a.id !== clip.id) return;
                            a.next = {
                              ...a.before,
                              audio: editAudioClipTiming(
                                a.before.audio!,
                                a.id,
                                a.mode,
                                ((e.clientX - a.x) / a.width) *
                                  p.project.duration,
                                p.project.fps,
                                p.snap,
                              ),
                            };
                            p.onPreview(a.next);
                          }}
                          onPointerUp={() => {
                            const a = drag.current;
                            drag.current = null;
                            if (a && a.next !== a.before)
                              p.onChange(a.next, a.before);
                          }}
                          onPointerCancel={() => {
                            if (drag.current) p.onPreview(drag.current.before);
                            drag.current = null;
                          }}
                        >
                          <svg
                            viewBox="0 0 1000 50"
                            preserveAspectRatio="none"
                            aria-label={`Waveform ${clip.name}`}
                          >
                            {waveform(
                              asset,
                              clip,
                              dbGain(
                                clip.gainDb + track.gainDb + audio.masterGainDb,
                              ),
                            ).map((path, i) => (
                              <path d={path} key={i} />
                            ))}
                            <path
                              className="audio-fade"
                              d={`M0,50L${Math.min(1, clip.fadeIn / clipDuration(clip)) * 1000},2M${1000 - Math.min(1, clip.fadeOut / clipDuration(clip)) * 1000},2L1000,50`}
                            />
                          </svg>
                          <span>{clip.name}</span>
                        </button>
                        {(["in", "out"] as const).map((mode) => (
                          <button
                            className={`audio-trim ${mode}`}
                            aria-label={`${clip.name} ${mode === "in" ? "Start" : "End"} trimmen`}
                            disabled={p.disabled}
                            key={mode}
                            onPointerDown={(e) => {
                              e.stopPropagation();
                              e.currentTarget.setPointerCapture(e.pointerId);
                              p.onSelect(clip.id);
                              drag.current = {
                                before: p.project,
                                next: p.project,
                                id: clip.id,
                                mode,
                                x: e.clientX,
                                width:
                                  e.currentTarget.parentElement!.parentElement!.getBoundingClientRect()
                                    .width,
                              };
                            }}
                            onPointerMove={(e) => {
                              const a = drag.current;
                              if (!a || a.id !== clip.id) return;
                              a.next = {
                                ...a.before,
                                audio: editAudioClipTiming(
                                  a.before.audio!,
                                  a.id,
                                  a.mode,
                                  ((e.clientX - a.x) / a.width) *
                                    p.project.duration,
                                  p.project.fps,
                                  p.snap,
                                ),
                              };
                              p.onPreview(a.next);
                            }}
                            onPointerUp={() => {
                              const a = drag.current;
                              drag.current = null;
                              if (a && a.next !== a.before)
                                p.onChange(a.next, a.before);
                            }}
                            onPointerCancel={() => {
                              if (drag.current)
                                p.onPreview(drag.current.before);
                              drag.current = null;
                            }}
                          />
                        ))}
                      </div>
                    );
                  })}
                <i
                  className="animation-playhead"
                  style={{ left: `${(p.time / p.project.duration) * 100}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

import StudioInput from "@/StudioInput";
import { useRef } from "react";
import {
  audioDefaults,
  clipDuration,
  parseAudioTimeline,
  type AudioClip,
  type AudioTimeline,
} from "./cinematic-audio";

export default function CinematicAudio(p: {
  audio?: AudioTimeline;
  selected?: string;
  importing: boolean;
  disabled: boolean;
  time: number;
  peak: number;
  onImport: (file: File) => void;
  onSelect: (id: string) => void;
  onChange: (audio: AudioTimeline) => void;
  onTimeline: () => void;
}) {
  const file = useRef<HTMLInputElement>(null),
    audio = p.audio;
  const clip = audio?.clips.find((c) => c.id === p.selected) ?? audio?.clips[0],
    asset = audio?.assets.find((a) => a.id === clip?.assetId);
  function editClip(change: Partial<AudioClip>) {
    if (!audio || !clip) return;
    const changed = { ...clip, ...change },
      duration = clipDuration(changed);
    changed.fadeIn = Math.min(changed.fadeIn, duration);
    changed.fadeOut = Math.min(changed.fadeOut, duration);
    const next = {
      ...audio,
      clips: audio.clips.map((c) => (c.id === clip.id ? changed : c)),
    };
    try {
      parseAudioTimeline(next);
      p.onChange(next);
    } catch {}
  }
  return (
    <section data-camera-panel="audio">
      <h3>Audio &amp; mixing</h3>
      <StudioInput
        type="file"
        ref={file}
        hidden
        aria-label="Import audio file"
        accept="audio/*,.wav,.mp3,.ogg,.m4a,.flac,.aac"
        onChange={(e) => {
          const f = e.target.files?.[0];
          e.target.value = "";
          if (f) p.onImport(f);
        }}
      />
      <button
        disabled={p.disabled || p.importing}
        onClick={() => file.current?.click()}
      >
        {p.importing ? "Decoding audio …" : "Add audio file"}
      </button>
      <button disabled={p.disabled} onClick={p.onTimeline}>
        Open audio timeline
      </button>
      {audio && (
        <>
          <label>
            Master · dB
            <StudioInput
              aria-label="Audio master level"
              type="number"
              min={-60}
              max={12}
              increment={0.5}
              value={audio.masterGainDb}
              disabled={p.disabled}
              onChange={(e) => {
                if (e.target.value && Number.isFinite(+e.target.value))
                  p.onChange({
                    ...audio,
                    masterGainDb: Math.max(-60, Math.min(12, +e.target.value)),
                  });
              }}
            />
          </label>
          <div
            className={`cinematic-audio-meter ${p.peak > 1 ? "clipping" : ""}`}
          >
            <meter
              aria-label="Audio level"
              min={0}
              max={1}
              value={Math.min(1, p.peak)}
            />
            <output>
              {p.peak > 1
                ? "Clipping"
                : p.peak > 0.00001
                  ? `${(20 * Math.log10(p.peak)).toFixed(1)} dBFS`
                  : "−∞ dBFS"}
            </output>
          </div>
          <h3>Mix tracks</h3>
          {audio.tracks.map((track, i) => (
            <div className="cinematic-mixer-track" key={track.id}>
              <label>
                Track
                <StudioInput
                  aria-label={`Audio track ${i + 1} Name`}
                  value={track.name}
                  maxLength={200}
                  disabled={p.disabled}
                  onChange={(e) =>
                    p.onChange({
                      ...audio,
                      tracks: audio.tracks.map((t) =>
                        t.id === track.id ? { ...t, name: e.target.value } : t,
                      ),
                    })
                  }
                />
              </label>
              <div className="cinema-pair">
                {(
                  [
                    {
                      field: "gainDb",
                      name: "Level",
                      min: -60,
                      max: 12,
                      step: 0.5,
                    },
                    {
                      field: "pan",
                      name: "Pan",
                      min: -1,
                      max: 1,
                      step: 0.05,
                    },
                  ] as const
                ).map((c) => (
                  <label key={c.field}>
                    {c.name}
                    <StudioInput
                      aria-label={`Audio track ${i + 1} ${c.name}`}
                      type="number"
                      disabled={p.disabled}
                      value={track[c.field]}
                      min={c.min}
                      max={c.max}
                      increment={c.step}
                      onChange={(e) => {
                        if (e.target.value && Number.isFinite(+e.target.value))
                          p.onChange({
                            ...audio,
                            tracks: audio.tracks.map((t) =>
                              t.id === track.id
                                ? {
                                    ...t,
                                    [c.field]: Math.max(
                                      c.min,
                                      Math.min(c.max, +e.target.value),
                                    ),
                                  }
                                : t,
                            ),
                          });
                      }}
                    />
                  </label>
                ))}
              </div>
              <div className="cinema-pair">
                {(["muted", "solo"] as const).map((field) => (
                  <label key={field}>
                    <StudioInput
                      type="checkbox"
                      aria-label={`Audio track ${i + 1} ${field === "muted" ? "Mute" : "Solo"}`}
                      checked={track[field]}
                      disabled={p.disabled}
                      onChange={(e) =>
                        p.onChange({
                          ...audio,
                          tracks: audio.tracks.map((t) =>
                            t.id === track.id
                              ? { ...t, [field]: e.target.checked }
                              : t,
                          ),
                        })
                      }
                    />
                    {field === "muted" ? "Mute" : "Solo"}
                  </label>
                ))}
              </div>
              <button
                disabled={p.disabled}
                onClick={() =>
                  p.onChange({
                    ...audio,
                    tracks: audio.tracks.filter((t) => t.id !== track.id),
                    clips: audio.clips.filter((c) => c.trackId !== track.id),
                  })
                }
              >
                Delete track
              </button>
            </div>
          ))}
          <button
            disabled={p.disabled || audio.tracks.length >= 16}
            onClick={() =>
              p.onChange({
                ...audio,
                tracks: [
                  ...audio.tracks,
                  {
                    ...audioDefaults().tracks[0],
                    name: `A${audio.tracks.length + 1}`,
                  },
                ],
              })
            }
          >
            Add audio track
          </button>
          <h3>Audio clips</h3>
          {clip && asset ? (
            <>
              <label>
                Clip
                <select
                  aria-label="Audio clip"
                  value={clip.id}
                  disabled={p.disabled}
                  onChange={(e) => p.onSelect(e.target.value)}
                >
                  {audio.clips.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Name
                <StudioInput
                  aria-label="Audio clip name"
                  value={clip.name}
                  maxLength={200}
                  disabled={p.disabled}
                  onChange={(e) => editClip({ name: e.target.value })}
                />
              </label>
              <label>
                Mix track
                <select
                  aria-label="Audio clip mix track"
                  value={clip.trackId}
                  disabled={p.disabled}
                  onChange={(e) => editClip({ trackId: e.target.value })}
                >
                  {audio.tracks.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name}
                    </option>
                  ))}
                </select>
              </label>
              {(
                [
                  {
                    field: "start",
                    label: "Clip position",
                    min: -3600,
                    max: 3600,
                    step: 0.01,
                  },
                  {
                    field: "in",
                    label: "Audio source in",
                    min: 0,
                    max: asset.duration,
                    step: 0.01,
                  },
                  {
                    field: "out",
                    label: "Audio source out",
                    min: 0,
                    max: asset.duration,
                    step: 0.01,
                  },
                  {
                    field: "rate",
                    label: "Audio speed",
                    min: 0.25,
                    max: 4,
                    step: 0.05,
                  },
                  {
                    field: "gainDb",
                    label: "Clip level",
                    min: -60,
                    max: 12,
                    step: 0.5,
                  },
                  {
                    field: "pan",
                    label: "Clip pan",
                    min: -1,
                    max: 1,
                    step: 0.05,
                  },
                  {
                    field: "fadeIn",
                    label: "Fade in",
                    min: 0,
                    max: clipDuration(clip),
                    step: 0.01,
                  },
                  {
                    field: "fadeOut",
                    label: "Fade out",
                    min: 0,
                    max: clipDuration(clip),
                    step: 0.01,
                  },
                ] as const
              ).map((c) => (
                <label key={c.field}>
                  {c.label}
                  <StudioInput
                    aria-label={c.label}
                    type="number"
                    disabled={p.disabled}
                    min={c.min}
                    max={c.max}
                    increment={c.step}
                    value={Number(clip[c.field].toFixed(5))}
                    onChange={(e) => {
                      if (e.target.value && Number.isFinite(+e.target.value))
                        editClip({ [c.field]: +e.target.value });
                    }}
                  />
                </label>
              ))}
              <label>
                <StudioInput
                  aria-label="Mute audio clip"
                  type="checkbox"
                  checked={clip.muted}
                  disabled={p.disabled}
                  onChange={(e) => editClip({ muted: e.target.checked })}
                />
                Mute clip
              </label>
              <div className="cinema-pair">
                <button
                  disabled={p.disabled || audio.clips.length >= 128}
                  onClick={() => {
                    const next = {
                      ...clip,
                      id: crypto.randomUUID(),
                      start: Math.min(3600, clip.start + clipDuration(clip)),
                      name: `${clip.name.slice(0, 190)} Copy`,
                    };
                    p.onChange({ ...audio, clips: [...audio.clips, next] });
                    p.onSelect(next.id);
                  }}
                >
                  Duplicate audio clip
                </button>
                <button
                  disabled={p.disabled}
                  onClick={() =>
                    p.onChange({
                      ...audio,
                      clips: audio.clips.filter((c) => c.id !== clip.id),
                    })
                  }
                >
                  Delete audio clip
                </button>
              </div>
            </>
          ) : (
            <p>Import audio or add a library asset.</p>
          )}
          <h3>Asset library</h3>
          {audio.assets.map((a) => (
            <div className="cinematic-audio-asset" key={a.id}>
              <span>
                {a.name} · {a.duration.toFixed(2)}s
              </span>
              <button
                disabled={p.disabled || audio.clips.length >= 128}
                onClick={() => {
                  const tracks = audio.tracks.length
                      ? audio.tracks
                      : audioDefaults().tracks,
                    c: AudioClip = {
                      id: crypto.randomUUID(),
                      name: a.name,
                      assetId: a.id,
                      trackId: tracks[0].id,
                      start: p.time,
                      in: 0,
                      out: a.duration,
                      rate: 1,
                      gainDb: 0,
                      pan: 0,
                      fadeIn: 0,
                      fadeOut: 0,
                      muted: false,
                    };
                  p.onChange({ ...audio, tracks, clips: [...audio.clips, c] });
                  p.onSelect(c.id);
                }}
              >
                Insert at playhead
              </button>
            </div>
          ))}
          <button
            disabled={p.disabled}
            onClick={() =>
              p.onChange({
                ...audio,
                assets: audio.assets.filter((a) =>
                  audio.clips.some((c) => c.assetId === a.id),
                ),
              })
            }
          >
            Remove unused assets
          </button>
        </>
      )}
      <p>MP4: AAC. WebM: Opus. GIF/PNG: silent. Speed affects pitch.</p>
    </section>
  );
}

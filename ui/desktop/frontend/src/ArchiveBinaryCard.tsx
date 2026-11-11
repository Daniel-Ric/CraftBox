import { useState } from "react";
import type { LabPreview } from "./bedrock-lab-model";
import { labBytes } from "./bedrock-lab-model";
import { api } from "./bridge";

export default function ArchiveBinaryCard({
  preview,
  other,
  label,
}: {
  preview: LabPreview | null;
  other: LabPreview | null;
  label: string;
}) {
  const [hex, setHex] = useState(false),
    [imageScale, setImageScale] = useState(false);
  const [saveMessage, setSaveMessage] = useState(""),
    [saving, setSaving] = useState(false);
  if (!preview)
    return (
      <section className="lab-binary-card">
        <h4>{label}</h4>
        <p className="lab-placeholder">Not present in this version.</p>
      </section>
    );
  const info = preview.binary,
    oldFields = new Map(
      other?.binary?.fields.map((f) => [f.name, f.value]) || [],
    );
  const sound = info?.samples?.[0],
    oldSound = other?.binary?.samples?.[0];
  const samePCM =
    sound?.pcmHash &&
    sound.pcmHash === oldSound?.pcmHash &&
    sound.rate === oldSound.rate &&
    sound.channels === oldSound.channels;
  return (
    <section className="lab-binary-card">
      <header>
        <h4>{label}</h4>
        <b>{preview.image ? "Image" : info?.format || preview.file.kind}</b>
        <small>{labBytes(preview.file.bytes)}</small>
      </header>
      {preview.audio && info?.format === "FMOD FSB5" && (
        <p className="lab-binary-verdict">
          {samePCM
            ? "Decoded samples, sample rate and channels are identical; the stored bank representation differs."
            : oldSound?.pcmHash
              ? "Decoded audio samples or playback timing differ. Use the players to hear both versions."
              : "Audio reconstructed to WAV for playback. The compressed original is retained."}
          {info.samples.length > 1
            ? " Playback and WAV comparison use the first stream."
            : ""}
        </p>
      )}
      {preview.image ? (
        <>
          <button onClick={() => setImageScale(!imageScale)}>
            {imageScale ? "Fit image" : "Actual pixels"}
          </button>
          <div className={`lab-image ${imageScale ? "actual-pixels" : ""}`}>
            <img src={preview.image} alt={preview.file.path} />
          </div>
        </>
      ) : (
        <>
          {info && (
            <dl className="lab-binary-metadata">
              {info.fields.map((f) => (
                <div
                  key={f.name}
                  data-changed={
                    oldFields.has(f.name) && oldFields.get(f.name) !== f.value
                  }
                >
                  <dt>{f.name}</dt>
                  <dd>
                    {f.value}
                    {oldFields.has(f.name) &&
                      oldFields.get(f.name) !== f.value && (
                        <small> differs</small>
                      )}
                  </dd>
                </div>
              ))}
            </dl>
          )}
          {info && info.samples.length > 0 && (
            <div className="lab-sound-samples">
              <h4>Audio streams</h4>
              {info.samples.map((s, i) => (
                <article key={i}>
                  <b>{s.name}</b>
                  <span>
                    {s.duration.toFixed(3)} s · {s.rate.toLocaleString()} Hz ·{" "}
                    {s.channels} {s.channels === 1 ? "channel" : "channels"}
                  </span>
                  <small>
                    {s.frames.toLocaleString()} samples ·{" "}
                    {labBytes(s.dataBytes)} encoded
                    {s.codebook ? ` · codebook ${s.codebook}` : ""}
                  </small>
                </article>
              ))}
            </div>
          )}
          {!!preview.waveform?.length && (
            <div className="lab-waveform">
              <svg
                viewBox="0 0 600 90"
                role="img"
                aria-label="Decoded audio waveform"
              >
                <line x1="0" y1="45" x2="600" y2="45" />
                <polygon
                  points={[
                    ...preview.waveform.map(
                      (v, i) =>
                        `${(i * 600) / (preview.waveform!.length - 1)},${45 - v * 42}`,
                    ),
                    ...preview.waveform
                      .map(
                        (v, i) =>
                          `${(i * 600) / (preview.waveform!.length - 1)},${45 + v * 42}`,
                      )
                      .reverse(),
                  ].join(" ")}
                />
              </svg>
              <small>
                0 s <span>{sound?.duration.toFixed(3)} s</span>
              </small>
            </div>
          )}
          {preview.audio && (
            <audio
              controls
              preload="metadata"
              src={preview.audio}
              aria-label={`${label} audio preview`}
              onPlay={(e) => {
                const active = e.currentTarget;
                document
                  .querySelectorAll<HTMLAudioElement>(".lab-diff audio")
                  .forEach((a) => {
                    if (a !== active) a.pause();
                  });
              }}
            />
          )}
          {preview.audio?.startsWith("data:audio/wav") && preview.snapshot && (
            <button
              disabled={saving}
              onClick={async () => {
                setSaving(true);
                setSaveMessage("");
                try {
                  const path = await api.ExportBedrockAudio(
                    preview.snapshot!,
                    preview.file.path,
                  );
                  if (path) setSaveMessage(`Saved ${path}`);
                } catch (e) {
                  setSaveMessage(e instanceof Error ? e.message : String(e));
                } finally {
                  setSaving(false);
                }
              }}
            >
              {saving ? "Saving…" : "Save decoded WAV"}
            </button>
          )}
          {saveMessage && <p role="status">{saveMessage}</p>}
          {!preview.audio && info?.payloadHash && (
            <p className="lab-binary-verdict">
              {other?.binary?.payloadHash === info.payloadHash
                ? "Encoded audio payload is identical; container metadata differs."
                : other?.binary?.payloadHash
                  ? "Encoded audio payload differs. Metadata cannot establish whether the difference is audible."
                  : "Stream metadata has been decoded. This FSB codec is not playable in the embedded browser."}
            </p>
          )}
          {(preview.file.issue || info?.issue) && (
            <p className="lab-warning">{preview.file.issue || info?.issue}</p>
          )}
          {info?.limited && (
            <p className="lab-warning">Metadata preview is limited.</p>
          )}
          <button aria-expanded={hex} onClick={() => setHex(!hex)}>
            {hex ? "Hide hexadecimal details" : "Show hexadecimal details"}
          </button>
          {hex && <pre className="lab-binary-hex">{preview.text}</pre>}
        </>
      )}
      <details className="lab-binary-provenance">
        <summary>File identity and origin</summary>
        <dl>
          <dt>Path</dt>
          <dd>{preview.file.path}</dd>
          <dt>SHA-256</dt>
          <dd>{preview.file.hash}</dd>
          <dt>Origin</dt>
          <dd>{preview.file.origin || "Loose source file"}</dd>
        </dl>
      </details>
    </section>
  );
}

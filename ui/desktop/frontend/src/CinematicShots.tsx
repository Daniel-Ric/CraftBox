import { useState } from "react";
import type { Sequence } from "./cinematic";
import { addCamera, enableCameras, removeCamera, resolveShot, splitShot, type CameraShot } from "./cinematic-shots";

export default function CinematicShots(p: { project: Sequence; cameraId?: string; time: number; disabled: boolean; onChange: (project: Sequence) => void; onCamera: (id: string) => void; onScrub: (time: number) => void }) {
  const [selection, setSelection] = useState<string>();
  const project = p.project, cameras = project.cameras ?? [], shots = project.shots ?? [];
  const camera = cameras.find((c) => c.id === p.cameraId) ?? cameras[0];
  const shot = shots.find((s) => s.id === selection) ?? resolveShot(project, p.time)?.shot;
  function update(change: Partial<CameraShot>) {
    if (!shot) return;
    const next = { ...shot, ...change };
    if (next.start < 0 || next.end > project.duration || next.start >= next.end || next.sourceIn < 0 || next.sourceIn > (cameras.find((c) => c.id === next.cameraId)?.duration ?? 0) || next.rate < 0.01 || next.rate > 100 || shots.some((s) => s.id !== shot.id && next.start < s.end && next.end > s.start)) return;
    p.onChange({ ...project, shots: shots.map((s) => s.id === shot.id ? next : s).sort((a, b) => a.start - b.start) });
  }
  return <section data-camera-panel="shots" className="cinematic-shots">
    <h3>Cameras &amp; shots</h3>
    {!cameras.length ? <><p>Each camera has its own animation tracks. Shots cut between cameras and use a chosen section of their animation.</p><button disabled={p.disabled} onClick={() => { const next = enableCameras(project); p.onChange(next); p.onCamera(next.cameras![0].id); }}>Create cameras and shots</button></> : <>
      <label>Edit camera<select aria-label="Edit camera" disabled={p.disabled} value={camera.id} onChange={(e) => p.onCamera(e.target.value)}>{cameras.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></label>
      <label>Camera name<input aria-label="Camera name" value={camera.name} maxLength={200} disabled={p.disabled} onChange={(e) => p.onChange({ ...project, cameras: cameras.map((c) => c.id === camera.id ? { ...c, name: e.target.value } : c) })} /></label>
      <div className="cinema-pair"><button disabled={p.disabled || cameras.length >= 16} onClick={() => { const next = addCamera(project, camera.id); p.onChange(next); p.onCamera(next.cameras!.at(-1)!.id); }}>Duplicate camera</button><button disabled={p.disabled || cameras.length < 2 || shots.some((s) => s.cameraId === camera.id)} onClick={() => p.onChange(removeCamera(project, camera.id))}>Delete camera</button></div>
      <h3>Shot sequence</h3>
      <div className="cinematic-shot-list">{shots.map((s, i) => <button key={s.id} aria-pressed={shot?.id === s.id} disabled={p.disabled} onClick={() => { setSelection(s.id); p.onScrub(s.start); }}>{i + 1} · {s.name}<small>{s.start.toFixed(2)}–{s.end.toFixed(2)}s · {cameras.find((c) => c.id === s.cameraId)?.name}</small></button>)}</div>
      {shot && <>
        <label>Shot name<input aria-label="Shot name" value={shot.name} maxLength={200} disabled={p.disabled} onChange={(e) => update({ name: e.target.value })} /></label>
        <label>Shot camera<select aria-label="Shot camera" value={shot.cameraId} disabled={p.disabled} onChange={(e) => update({ cameraId: e.target.value, sourceIn: Math.min(shot.sourceIn, cameras.find((c) => c.id === e.target.value)!.duration) })}>{cameras.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></label>
        {([{ field: "start", label: "Shot start", min: 0, max: project.duration }, { field: "end", label: "Shot end", min: 0, max: project.duration }, { field: "sourceIn", label: "Source in", min: 0, max: cameras.find((c) => c.id === shot.cameraId)!.duration }, { field: "rate", label: "Shot speed", min: 0.01, max: 100 }] as const).map((c) => <label key={c.field}>{c.label}<input aria-label={c.label} type="number" value={Number(shot[c.field].toFixed(5))} min={c.min} max={c.max} step={c.field === "rate" ? 0.05 : 1 / project.fps} disabled={p.disabled} onChange={(e) => { if (e.target.value && Number.isFinite(+e.target.value)) update({ [c.field]: +e.target.value }); }} /></label>)}
        <button disabled={p.disabled || p.time <= shot.start || p.time >= shot.end} onClick={() => { const next = splitShot(project, shot.id, p.time); if (next !== project) { p.onChange(next); setSelection(next.shots!.find((s) => s.start >= p.time - 1 / project.fps / 2)!.id); } }}>Split shot at playhead</button>
        <button disabled={p.disabled || shots.length < 2} onClick={() => { p.onChange({ ...project, shots: shots.filter((s) => s.id !== shot.id) }); setSelection(undefined); }}>Delete shot</button>
        <p>Source in and speed determine local camera time. Gaps and time beyond a camera animation hold the last frame.</p>
      </>}
    </>}
  </section>;
}

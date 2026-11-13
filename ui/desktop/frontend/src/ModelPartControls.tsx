import { useState, type ReactNode } from "react";
import { BoxSelect, Download, Search, Scan, ArrowLeft, Pencil, X } from "lucide-react";
import { Button } from "./components/ui/button";
import StudioInput from "./StudioInput";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "./components/ui/dialog";
import type { ModelPart, PartMode } from "./model-parts";
import { selectionProjectName } from "./model-part-names";
import "./model-part-controls.css";

type Props = {
  enabled: boolean;
  mode: PartMode;
  parts: ModelPart[];
  selected: string[];
  busy: boolean;
  name: string;
  preview: ReactNode;
  onFit: () => void;
  isolated?: boolean;
  onIsolated?: (isolated: boolean) => void;
  onEnabled: (enabled: boolean) => void;
  onMode: (mode: PartMode) => void;
  onSelect: (ids: string[]) => void;
  onRename?: (id: string, name: string) => void;
  onExtract: (name: string, open: boolean) => Promise<boolean>;
};

export default function ModelPartControls(p: Props) {
  const [query, setQuery] = useState(""), [stage, setStage] = useState<"select" | "export">("select");
  const [name, setName] = useState(""), [openBlockbench, setOpenBlockbench] = useState(true);
  const [renaming, setRenaming] = useState<{ id: string; value: string }>();
  const selected = new Set(p.selected), chosen = p.parts.filter(part => selected.has(part.id));
  const visible = p.parts.filter(part => `${part.name} ${part.sourceName ?? ""} ${part.bones.join(" ")}`.toLowerCase().includes(query.toLowerCase()));
  const elements = new Set(chosen.flatMap(part => part.elements.map(element => JSON.stringify(element))));
  const beginExport = () => {
    setName(selectionProjectName(chosen, p.name));
    setStage("export");
  };
  const commitName = () => {
    if (renaming) p.onRename?.(renaming.id, renaming.value.trim());
    setRenaming(undefined);
  };
  return <>
    <div className="model-part-launcher"><Button variant="outline" onClick={() => { setStage("select"); p.onEnabled(true); }}><BoxSelect size={14} />Extract model parts…</Button></div>
    <Dialog open={p.enabled} onOpenChange={value => { if (!p.busy) p.onEnabled(value); }}>
      <DialogContent className="model-selection-dialog">
        <DialogHeader><DialogTitle>{stage === "select" ? "Extract model parts" : "Save extracted model"}</DialogTitle><DialogDescription>{p.name}</DialogDescription></DialogHeader>
        <div className="studio-dialog-body model-selection-body">
          <section className="model-selection-preview" aria-label="Model selection preview">
            <div className="model-selection-preview-toolbar"><span>{p.name}</span><Button variant="ghost" onClick={p.onFit}><Scan size={13} />Fit view</Button></div>
            <div className="model-selection-canvas">{p.preview}</div>
            <p className="model-selection-hint">Click to select · Drag to orbit · Scroll to zoom</p>
          </section>
          <aside className="model-selection-sidebar">
            {stage === "select" ? <>
              <label className="model-selection-field"><span>Select by</span><select aria-label="Model selection granularity" value={p.mode} onChange={event => { p.onSelect([]); setRenaming(undefined); p.onMode(event.target.value as PartMode); }}><option value="parts">Automatic parts</option><option value="hierarchy">Hierarchy branches</option><option value="islands">Connected geometry</option><option value="bones">Bone groups</option><option value="elements">Individual elements</option></select></label>
              <label className="model-selection-search"><Search size={13} /><StudioInput aria-label="Find model parts" value={query} placeholder="Find a part…" onChange={event => setQuery(event.target.value)} /></label>
              <div className="model-selection-list-actions"><span>{visible.length} available</span><Button variant="ghost" disabled={!visible.length} onClick={() => p.onSelect([...new Set([...p.selected, ...visible.map(part => part.id)])])}>Select all</Button><Button variant="ghost" disabled={!chosen.length} onClick={() => p.onSelect([])}><X size={11} />Clear</Button></div>
              <div className="model-part-items" aria-label="Model parts">{visible.slice(0, 200).map(part => <div className="model-part-row" data-selected={selected.has(part.id)} key={part.id}>
                {renaming?.id === part.id ? <StudioInput aria-label="Part name" maxLength={80} value={renaming.value} autoFocus onChange={event => setRenaming({ id: part.id, value: event.target.value })} onBlur={commitName} onKeyDown={event => { if (event.key === "Enter") { event.preventDefault(); event.stopPropagation(); commitName(); } else if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); setRenaming(undefined); } }} /> : <>
                  <label title={part.sourceName ? `Source: ${part.sourceName}` : undefined}><input type="checkbox" checked={selected.has(part.id)} onChange={() => p.onSelect(selected.has(part.id) ? p.selected.filter(id => id !== part.id) : [...p.selected, part.id])} /><span>{part.name}<small>{part.elements.length} elements · {part.bones.length} bones</small>{part.sourceName && part.sourceName.toLowerCase() !== part.name.toLowerCase() && <small className="model-part-source">Source: {part.sourceName}</small>}</span></label>
                  {p.onRename && <Button variant="ghost" className="model-part-rename" aria-label={`Rename ${part.name}`} title="Name for extraction" onClick={() => setRenaming({ id: part.id, value: part.name })}><Pencil size={12} /></Button>}
                </>}
              </div>)}{!visible.length && <p className="model-selection-note">{p.parts.length ? "No matching parts." : "Preparing selectable parts…"}</p>}</div>
              {visible.length > 200 && <p className="model-selection-note">Showing 200 results. Refine the search to find more parts.</p>}
              <p className="model-selection-note">{p.mode === "parts" ? "Combines rig structure and contact between rotated shapes." : p.mode === "hierarchy" ? "Keeps authored branches separate, including touching objects." : p.mode === "islands" ? "Finds connected surfaces independently of the bone hierarchy." : "Uses the authored model structure."} Click again to remove a part.</p>
            </> : <>
              <h3>Blockbench project</h3>
              <label className="model-selection-field"><span>Project name</span><StudioInput aria-label="Project name" value={name} maxLength={120} onChange={event => setName(event.target.value)} autoFocus /></label>
              <dl className="model-selection-summary"><div><dt>Selected parts</dt><dd>{chosen.length}</dd></div><div><dt>Elements</dt><dd>{elements.size}</dd></div><div><dt>Format</dt><dd>.bbmodel</dd></div></dl>
              <p className="model-selection-note">Includes embedded textures, UVs and animation tracks. The extracted model is centered for Blockbench, with its bone hierarchy preserved.</p>
              <label className="model-selection-check"><input type="checkbox" checked={openBlockbench} onChange={event => setOpenBlockbench(event.target.checked)} />Open in Blockbench after saving</label>
            </>}
            {p.onIsolated && <label className="model-selection-check model-selection-isolate"><input type="checkbox" checked={!!p.isolated} disabled={!chosen.length} onChange={event => p.onIsolated?.(event.target.checked)} />Show selection only</label>}
          </aside>
        </div>
        <DialogFooter>
          <span className="model-selection-count" role="status" aria-live="polite">{chosen.length} selected · {elements.size} elements</span>
          {stage === "export" && <Button variant="ghost" disabled={p.busy} onClick={() => setStage("select")}><ArrowLeft size={13} />Back</Button>}
          <Button variant="ghost" disabled={p.busy} onClick={() => p.onEnabled(false)}>Cancel</Button>
          {stage === "select" ? <Button disabled={!chosen.length || p.busy} onClick={beginExport}><Download size={13} />Extract selection…</Button> : <Button disabled={p.busy || !name.trim() || !chosen.length} onClick={() => void p.onExtract(name.trim(), openBlockbench).then(saved => { if (saved) p.onEnabled(false); })}>{p.busy ? "Preparing project…" : "Save extracted project"}</Button>}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </>;
}

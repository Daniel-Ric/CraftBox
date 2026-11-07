import { useEffect, useMemo, useState } from "react";
import CollectionToolbar from "./CollectionToolbar";
import {collectionSelection} from "./collection-selection";
import {
  Blocks,
  Box,
  FolderOpen,
  Plus,
  RefreshCw,
  Search,
  Download,
} from "lucide-react";
import { Button } from "./components/ui/button";
import type { Recovery } from "./structure-api";

export default function StructureLibrary({
  items,
  busy,
  onOpen,
  onNew,
  onFile,
  onWorld,
  onRefresh,
}: {
  items: Recovery[];
  busy: boolean;
  onOpen: (id: string) => void | Promise<void>;
  onNew: () => void;
  onFile: () => void;
  onWorld: () => void;
  onRefresh: () => void;
}) {
  const [query, setQuery] = useState("");
  const [multiple,setMultiple]=useState(false),[selected,setSelected]=useState<string[]>([]),[anchor,setAnchor]=useState("");
  const itemIDs=items.map(value=>value.id).join("\0");
  useEffect(()=>{const available=new Set(items.map(value=>value.id));setSelected(old=>{
    const next=old.filter(id=>available.has(id));return next.length===old.length?old:next;
  });},[itemIDs]);
  const shown = useMemo(
    () =>
      items
        .filter((r) =>
          r.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()),
        )
        .sort((a, b) => b.modified - a.modified),
    [items, query],
  );
  return (
    <div className="structure-workarea structure-library-workarea">
      <aside className="studio-sidebar structure-browser" aria-label="Blueprint library">
        <header className="structure-section-heading">
          <strong>Blueprints</strong>
          <span>{items.length}</span>
        </header>
        <label className="structure-search">
          <Search size={14} />
          <input
            type="search"
            aria-label="Search blueprints"
            placeholder="Search name…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <div className="structure-library-tools">
          <span>Newest first</span>
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={busy}
            aria-label="Refresh blueprints"
            title="Refresh blueprints"
            onClick={onRefresh}
          >
            <RefreshCw size={14} />
          </Button>
        </div>
        <CollectionToolbar ids={shown.map(value=>value.id)} selected={selected} multiple={multiple} onMultiple={setMultiple} onSelected={setSelected} busy={busy}
          onOpen={ids=>void(async()=>{for(const id of ids)await onOpen(id);})()}/>
        <div className="structure-library-list">
          {shown.map((r) => (
            <button
              className="structure-library-row"
              key={r.id}
              disabled={busy}
              aria-pressed={selected.includes(r.id)}
              onClick={event=>{
                const next=collectionSelection(shown.map(value=>value.id),selected,anchor,r.id,event,multiple);
                setSelected(next.selected);setAnchor(next.anchor);setMultiple(next.multiple);
                if(!next.multiple)void onOpen(r.id);
              }}
            >
              <Box size={28} />
              <span>
                <strong>{r.name}</strong>
                <small>{new Date(r.modified).toLocaleString()}</small>
                <small>{r.open ? "Open document" : "Saved blueprint"}</small>
              </span>
            </button>
          ))}
          {!shown.length && (
            <div className="structure-library-empty">
              <Box size={32} />
              <strong>
                {query ? "No matching blueprints" : "No saved blueprints"}
              </strong>
              <small>
                {query
                  ? "Try another name."
                  : "Created and captured structures appear here."}
              </small>
            </div>
          )}
        </div>
        <footer className="structure-library-footer">
          <span>
            {shown.length} {shown.length === 1 ? "blueprint" : "blueprints"}
          </span>
          <small>Local library</small>
        </footer>
      </aside>
      <main
        className="structure-center structure-empty-preview"
        aria-label="Structure preview"
      >
        <div data-editor-tab-host />
        <header className="structure-section-heading">
          <strong>Preview</strong>
        </header>
        <div className="document-empty structure-empty">
          <Blocks size={32} />
          <strong>No open structure</strong>
          <span>Select a blueprint or create one to start.</span>
          <div className="document-empty-actions">
            <Button disabled={busy} onClick={onNew}>
              <Plus size={15} />
              New structure
            </Button>
            <Button disabled={busy} variant="outline" onClick={onFile}>
              <FolderOpen size={15} />
              Open .mcstructure / .craftbox
            </Button>
            <Button disabled={busy} variant="ghost" onClick={onWorld}>
              <Download size={15} />
              Extract from world…
            </Button>
          </div>
        </div>
        <details className="structure-capture-help">
          <summary>Capture a structure in Minecraft</summary>
          <p>Select both corners, then save an independent blueprint.</p>
          <code>/cb structure pos1</code>
          <code>/cb structure pos2</code>
          <code>/cb structure save My house</code>
        </details>
      </main>
      <aside className="structure-inspector" aria-label="Structure properties">
        <header className="structure-section-heading">
          <strong>Properties</strong>
        </header>
        <div className="structure-properties-empty">
          <span>Select a structure to see its properties.</span>
        </div>
      </aside>
    </div>
  );
}

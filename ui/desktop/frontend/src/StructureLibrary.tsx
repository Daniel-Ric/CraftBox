import { formatLocale } from "./i18n";
import { useEffect, useMemo, useState } from "react";
import CollectionToolbar from "./CollectionToolbar";
import { collectionSelection } from "./collection-selection";
import { Box, FolderOpen, Plus, RefreshCw, Download } from "lucide-react";
import { Button } from "./components/ui/button";
import type { Recovery } from "./structure-api";

type Props = {
  items: Recovery[];
  query: string;
  view: "grid" | "list";
  busy: boolean;
  onOpen: (id: string) => void | Promise<void>;
  onNew: () => void;
  onFile: () => void;
  onWorld: () => void;
  onRefresh: () => void;
  onDelete: (ids: string[]) => void;
};
export default function StructureLibrary(p: Props) {
  const [multiple, setMultiple] = useState(false),
    [selected, setSelected] = useState<string[]>([]),
    [anchor, setAnchor] = useState("");
  const itemIDs = p.items.map((value) => value.id).join("\0");
  useEffect(() => {
    const available = new Set(p.items.map((value) => value.id));
    setSelected((old) => {
      const next = old.filter((id) => available.has(id));
      return next.length === old.length ? old : next;
    });
  }, [itemIDs]);
  const shown = useMemo(
    () =>
      p.items
        .filter((item) =>
          item.name
            .toLocaleLowerCase()
            .includes(p.query.trim().toLocaleLowerCase()),
        )
        .sort((a, b) => b.modified - a.modified),
    [p.items, p.query],
  );
  return (
    <section
      className="structure-library-page visual-library"
      aria-label="Blueprint library"
    >
      <div data-editor-tab-host />
      <div className="collection-view-tools">
        <span>
          {shown.length} {shown.length === 1 ? "blueprint" : "blueprints"}
        </span>
        <span>Newest first</span>
        <Button
          variant="ghost"
          size="icon-sm"
          disabled={p.busy}
          aria-label="Refresh blueprints"
          data-tooltip="Refresh blueprints"
          onClick={p.onRefresh}
        >
          <RefreshCw size={14} />
        </Button>
        <details className="blueprint-capture-help">
          <summary>Capture in Minecraft</summary>
          <div>
            <p>Select both corners, then save an independent blueprint.</p>
            <code>/cb structure pos1</code>
            <code>/cb structure pos2</code>
            <code>/cb structure save My house</code>
          </div>
        </details>
        <CollectionToolbar
          ids={shown.map((item) => item.id)}
          selected={selected}
          multiple={multiple}
          onMultiple={setMultiple}
          onSelected={setSelected}
          busy={p.busy}
          onDelete={p.onDelete}
          onOpen={(ids) =>
            void (async () => {
              for (const id of ids) await p.onOpen(id);
            })()
          }
        />
      </div>
      <div
        className={`blueprint-collection collection-items${!shown.length ? " is-empty" : ""}`}
        data-layout={p.view}
        data-content="blueprints"
      >
        {shown.map((item) => (
          <div
            className="collection-item blueprint-card"
            key={item.id}
            data-selected={selected.includes(item.id)}
          >
            <button
              className="collection-select"
              aria-label={item.name}
              disabled={p.busy}
              aria-pressed={selected.includes(item.id)}
              onClick={(event) => {
                const next = collectionSelection(
                  shown.map((value) => value.id),
                  selected,
                  anchor,
                  item.id,
                  event,
                  multiple,
                );
                setSelected(next.selected);
                setAnchor(next.anchor);
                setMultiple(next.multiple);
                if (!next.multiple) void p.onOpen(item.id);
              }}
            >
              <span className="collection-media">
                <Box size={42} />
              </span>
              <span className="collection-copy">
                <strong>{item.name}</strong>
                <small>{new Date(item.modified).toLocaleString(formatLocale())}</small>
                <small>{item.open ? "Open document" : "Saved blueprint"}</small>
              </span>
            </button>
          </div>
        ))}
        {!shown.length && (
          <div className="document-empty structure-library-empty">
            <Box size={36} />
            <strong>
              {p.query ? "No matching blueprints" : "Your blueprint library"}
            </strong>
            <span>
              {p.query
                ? "Try another name."
                : "Create a structure, open a file or extract a region from a captured world."}
            </span>
            <div className="document-empty-actions">
              <Button disabled={p.busy} onClick={p.onNew}>
                <Plus size={15} />
                New structure
              </Button>
              <Button disabled={p.busy} variant="outline" onClick={p.onFile}>
                <FolderOpen size={15} />
                Open .mcstructure / .craftbox
              </Button>
              <Button disabled={p.busy} variant="ghost" onClick={p.onWorld}>
                <Download size={15} />
                Extract from world...
              </Button>
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

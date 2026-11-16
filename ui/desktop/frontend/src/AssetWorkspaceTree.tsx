import { useMemo, useState } from "react";
import type { MouseEvent } from "react";
import { ChevronRight, File, FileCode, Folder, Image, Layers, Shirt } from "lucide-react";
import type { AssetFile, LibraryAsset } from "./bridge";

type Node = { name: string; path: string; children: Map<string, Node>; file?: AssetFile };
function fileNodes(files: AssetFile[]) {
  const root = new Map<string, Node>();
  for (const file of files) {
    const parts = file.path.replaceAll("\\", "/").split("/").filter(Boolean);
    let children = root;
    parts.forEach((name, index) => {
      let node = children.get(name);
      if (!node) {
        node = { name, path: parts.slice(0, index + 1).join("/"), children: new Map() };
        children.set(name, node);
      }
      if (index === parts.length - 1) node.file = file;
      children = node.children;
    });
  }
  return root;
}
export default function AssetWorkspaceTree(p: {
  items: LibraryAsset[]; active: string; selected: string[]; files: AssetFile[];
  file: string; loading: boolean; loadingFiles?: boolean; filtering?: boolean; multiple?: boolean;
  onAsset: (asset: LibraryAsset, event: MouseEvent<HTMLButtonElement>) => void;
  onFile: (path: string) => void;
  onAudio?: (files: AssetFile[]) => void;
}) {
  const roots = useMemo(() => fileNodes(p.files), [p.files]);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [collapsedAssets, setCollapsedAssets] = useState<Set<string>>(new Set());
  const nodes = (children: Map<string, Node>, depth: number) =>
    [...children.values()].sort((a, b) => Number(!a.children.size) - Number(!b.children.size) || a.name.localeCompare(b.name)).map(node => {
      const open = p.filtering || (expanded[node.path] ?? p.file.startsWith(node.path + "/"));
      if (node.children.size) return <details key={node.path} className="asset-tree-folder" open={!!open}>
        <summary role="treeitem" aria-expanded={!!open} onClick={event => {
          event.preventDefault();
          if (!p.filtering) setExpanded(old => ({...old, [node.path]: !open}));
        }} style={{ paddingLeft: 12 + depth * 16 }}><ChevronRight className="tree-chevron"/><Folder/><span>{node.name}</span><small>{node.children.size}</small></summary>
        {open && <div role="group">{p.onAudio && p.files.some(file => file.kind === "sound" && file.path.startsWith(node.path + "/")) && <button className="asset-tree-sequence" style={{paddingLeft: 28 + depth * 16}} onClick={() => p.onAudio?.(p.files.filter(file => file.kind === "sound" && file.path.startsWith(node.path + "/")))}>Open as audio sequence</button>}{nodes(node.children, depth + 1)}</div>}
      </details>;
      const Icon = node.file?.kind === "texture" ? Image : ["json", "model", "animation"].includes(node.file?.kind ?? "") ? FileCode : File;
      return <button key={node.path} role="treeitem" aria-selected={p.file === node.path} title={node.path} className="asset-tree-file" style={{ paddingLeft: 28 + depth * 16 }} onClick={() => p.onFile(node.file!.path)}><Icon/><span>{node.name}</span></button>;
    });
  return <section className="asset-workspace-tree" aria-label="Library explorer">
    <header><strong>Library</strong><span>{p.items.length}</span></header>
    <div role="tree" aria-label="Library files" aria-multiselectable={p.multiple || undefined} onKeyDown={event => {
      if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
        const target = document.activeElement as HTMLElement;
        if (!event.currentTarget.contains(target)) return;
        const folder = target.tagName === "SUMMARY" ? target.parentElement as HTMLDetailsElement : undefined;
        if (event.key === "ArrowRight") {
          if (folder && !folder.open) target.click();
          else if (folder) folder.querySelector<HTMLElement>('[role="group"] [role="treeitem"]')?.focus();
          else if (target.getAttribute("aria-expanded") === "false") target.click();
          else target.nextElementSibling?.querySelector<HTMLElement>('[role="treeitem"]')?.focus();
        } else if (folder?.open) target.click();
        else if (target.classList.contains("asset-tree-asset") && target.getAttribute("aria-expanded") === "true") target.click();
        else {
          const parent = folder ? folder.parentElement?.closest("details") : target.parentElement?.closest("details");
          if (parent) parent.querySelector("summary")?.focus();
          else target.closest(".asset-tree-root")?.querySelector<HTMLElement>("button")?.focus();
        }
        event.preventDefault(); return;
      }
      if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
      const items = [...event.currentTarget.querySelectorAll<HTMLElement>('[role="treeitem"]')].filter(item => item.getClientRects().length);
      const current = items.indexOf(document.activeElement as HTMLElement);
      const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : Math.max(0, Math.min(items.length - 1, current + (event.key === "ArrowDown" ? 1 : -1)));
      items[next]?.focus(); event.preventDefault();
    }}>
      {p.loading && <p role="status">Loading library…</p>}
      {!p.loading && !p.items.length && <p>No matching items. Import files or change your search.</p>}
      {p.items.map(asset => {
        const active = asset.id === p.active;
        const open = active && (p.filtering || !collapsedAssets.has(asset.id));
        const Icon = asset.kind === "skins" ? Shirt : Layers;
        return <div key={asset.id} className="asset-tree-root">
          <button role="treeitem" aria-expanded={open} aria-selected={p.selected.includes(asset.id)} className="asset-tree-asset" title={asset.name} onClick={event => {
            if (event.detail >= 2) {
              setCollapsedAssets(old => { const next = new Set(old); next.delete(asset.id); return next; });
              p.onAsset(asset, event);
            }
            else if (active && !p.multiple && !event.ctrlKey && !event.metaKey && !event.shiftKey) setCollapsedAssets(old => { const next = new Set(old); if (next.has(asset.id)) next.delete(asset.id); else next.add(asset.id); return next; });
            else p.onAsset(asset, event);
          }}><ChevronRight className={`tree-chevron ${open ? "is-open" : ""}`}/><Icon/><span>{asset.name}</span></button>
          {open && <div role="group">{p.loadingFiles ? <p role="status">Loading files…</p> : p.files.length ? nodes(roots, 1) : <p className="asset-tree-note">{p.filtering ? "No matching files" : "No files to display"}</p>}</div>}
        </div>;
      })}
    </div>
  </section>;
}

import { useEffect, useMemo, useRef, useState } from "react";
import {
  Archive,
  ArrowLeftRight,
  Check,
  ChevronLeft,
  ChevronRight,
  Copy,
  Download,
  FileCode,
  FolderOpen,
  Hash,
  Plus,
  Info,
  LoaderCircle,
  RefreshCw,
  Search,
  ShieldCheck,
  Square,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import { api, type MinecraftInstallation } from "./bridge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  fileChanges,
  identifierChanges,
  labBytes,
  preferredInstallation,
  type LabChange,
  type LabFile,
  type LabPage,
  type LabPreview,
  type LabProgress,
  type LabSchema,
  type LabSnapshot,
  type LabString,
} from "./bedrock-lab-model";
import "./archive.css";
import HeaderActionIcon from "./HeaderActionIcon";
import { bedrockLabNotices } from "./bedrock-lab-notices";
import { useWorkspaceActions } from "./workspace-actions";

type Tab = "files" | "changes" | "identifiers" | "strings" | "validation";
const failure = (e: unknown) => (e instanceof Error ? e.message : String(e));
const blankPage: LabPage = { files: [], total: 0, truncated: false };
const tabs: { id: Tab; label: string }[] = [
  { id: "files", label: "Files" },
  { id: "changes", label: "Changes" },
  { id: "identifiers", label: "Identifiers" },
  { id: "strings", label: "Executable strings" },
  { id: "validation", label: "Validation" },
];

function FileView({
  preview,
  label,
}: {
  preview: LabPreview | null;
  label: string;
}) {
  return (
    <section className="lab-file-preview" aria-label={label}>
      <header>
        <b>{label}</b>
        {preview && (
          <small>
            {preview.file.kind} · {labBytes(preview.file.bytes)}
          </small>
        )}
      </header>
      {!preview ? (
        <p className="lab-placeholder">
          Select a file to inspect its contents.
        </p>
      ) : (
        <>
          {preview.file.issue && (
            <p className="lab-warning">
              <TriangleAlert size={13} />
              {preview.file.issue}
            </p>
          )}
          {preview.image ? (
            <div className="lab-image">
              <img src={preview.image} alt={preview.file.path} />
            </div>
          ) : (
            <pre tabIndex={0}>{preview.text || "Empty file"}</pre>
          )}
          {preview.truncated && (
            <p className="lab-warning">
              Preview is limited. The export contains the complete file.
            </p>
          )}
        </>
      )}
    </section>
  );
}

export default function ArchiveStudio() {
  const [snapshotOpen, setSnapshotOpen] = useState(false),
    [propertiesOpen, setPropertiesOpen] = useState(false),
    [snapshotSearch, setSnapshotSearch] = useState("");
  const [snapshots, setSnapshots] = useState<LabSnapshot[]>([]),
    [installations, setInstallations] = useState<MinecraftInstallation[]>([]),
    [installation, setInstallation] = useState(""),
    [source, setSource] = useState(
      () => localStorage.getItem("craftbox:lab-source") || "",
    ),
    [name, setName] = useState(""),
    [version, setVersion] = useState(""),
    [schemaVersion, setSchemaVersion] = useState(""),
    [channel, setChannel] = useState("preview"),
    [schemas, setSchemas] = useState<LabSchema[]>([]),
    [schemaPath, setSchemaPath] = useState(
      () => localStorage.getItem("craftbox:lab-schema") || "",
    ),
    [schemaOpen, setSchemaOpen] = useState(false);
  const [noticesOpen, setNoticesOpen] = useState(false);
  const [selected, setSelected] = useState(
      () => localStorage.getItem("craftbox:lab-snapshot") || "",
    ),
    [baseline, setBaseline] = useState(""),
    [tab, setTab] = useState<Tab>("files"),
    [query, setQuery] = useState(""),
    [kind, setKind] = useState(""),
    [content, setContent] = useState(false),
    [offset, setOffset] = useState(0),
    [page, setPage] = useState<LabPage>(blankPage),
    [changes, setChanges] = useState<LabChange[]>([]),
    [formatting, setFormatting] = useState(false),
    [selectedFile, setSelectedFile] = useState(""),
    [preview, setPreview] = useState<LabPreview | null>(null),
    [before, setBefore] = useState<LabPreview | null>(null);
  const [progress, setProgress] = useState<LabProgress>({
      running: false,
      phase: "Ready",
      path: "",
      done: 0,
      total: 0,
      snapshot: "",
      error: "",
    }),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(false),
    [validation, setValidation] = useState<{
      checked: number;
      issues: string[];
    } | null>(null),
    [strings, setStrings] = useState<LabString[]>([]),
    [binary, setBinary] = useState(""),
    [deleteID, setDeleteID] = useState(""),
    [ids, setIDs] = useState<{ added: string[]; removed: string[] }>({
      added: [],
      removed: [],
    });
  const previousProgress = useRef(false),
    querySequence = useRef(0),
    previewSequence = useRef(0);
  const snapshot = snapshots.find((s) => s.id === selected),
    base = snapshots.find((s) => s.id === baseline),
    changeMap = useMemo(
      () => new Map(changes.map((c) => [c.path, c])),
      [changes],
    ),
    selectedChange = changeMap.get(selectedFile),
    file = preview?.file || selectedChange?.after || selectedChange?.before;
  async function refresh() {
    const [items, bundles] = await Promise.all([
      api.GetBedrockSnapshots(),
      api.GetBedrockSchemas(),
    ]);
    setSnapshots(items);
    setSchemas(bundles);
    setSelected((current) =>
      items.some((s) => s.id === current) ? current : items[0]?.id || "",
    );
    setBaseline((current) =>
      items.some((s) => s.id === current) ? current : items[1]?.id || "",
    );
  }
  async function run(action: () => Promise<unknown>, success = "") {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await action();
      if (success) setNotice(success);
    } catch (e) {
      setError(failure(e));
    } finally {
      setBusy(false);
    }
  }
  async function detect() {
    const result = await api.GetMinecraftInstallations();
    setInstallations(result.installations || []);
    setInstallation((current) =>
      result.installations.some((i) => i.id === current && i.location)
        ? current
        : preferredInstallation(result.installations || []),
    );
    if (result.warning) setNotice(result.warning);
  }
  useEffect(() => {
    void run(async () => {
      await Promise.all([refresh(), detect()]);
    });
    let active = true;
    const poll = async () => {
      try {
        const p = await api.GetBedrockLabProgress();
        if (!active) return;
        setProgress(p);
        if (previousProgress.current && !p.running) {
          await refresh();
          if (p.snapshot) setSelected(p.snapshot);
          if (p.error) setError(p.error);
        }
        previousProgress.current = p.running;
      } catch (e) {
        if (active) setError(failure(e));
      }
    };
    void poll();
    const timer = setInterval(() => void poll(), 700);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, []);
  useEffect(() => {
    localStorage.setItem("craftbox:lab-snapshot", selected);
    setSelectedFile("");
    setValidation(null);
    setStrings([]);
  }, [selected]);
  useEffect(() => {
    localStorage.setItem("craftbox:lab-source", source);
    localStorage.setItem("craftbox:lab-schema", schemaPath);
  }, [source, schemaPath]);
  useWorkspaceActions("bedrock", (action) => {
    if (action === "export" && selected && !working)
      void run(() => api.ExportBedrockSnapshot(selected));
    if (action === "import-folder") void run(() => chooseFolder("source"));
    if (action === "refresh") void run(refresh);
    if (action === "settings") setSchemaOpen(true);
    if (action === "search")
      document.querySelector<HTMLInputElement>(".lab-search input")?.focus();
  });
  useEffect(() => {
    setOffset(0);
    setSelectedFile("");
  }, [selected, baseline, tab, query, kind, content]);
  useEffect(() => {
    let active = true;
    const sequence = ++querySequence.current;
    if (!selected || !["files", "identifiers", "validation"].includes(tab)) {
      setPage(blankPage);
      return;
    }
    setLoading(true);
    const timer = setTimeout(() => {
      api
        .QueryBedrockFiles({
          snapshot: selected,
          search: query,
          kind,
          mode:
            tab === "validation"
              ? "issues"
              : tab === "identifiers"
                ? "identifiers"
                : content
                  ? "content"
                  : "paths",
          offset,
          limit: 200,
        })
        .then((result) => {
          if (active && querySequence.current === sequence) setPage(result);
        })
        .catch((e) => {
          if (active) setError(failure(e));
        })
        .finally(() => {
          if (active) setLoading(false);
        });
    }, 180);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [selected, tab, query, kind, content, offset, progress.snapshot]);
  useEffect(() => {
    let active = true;
    setChanges([]);
    if (!selected || !baseline || selected === baseline) return;
    api
      .CompareBedrockSnapshots(baseline, selected)
      .then((c) => {
        if (active) setChanges(c);
      })
      .catch((e) => {
        if (active) setError(failure(e));
      });
    return () => {
      active = false;
    };
  }, [selected, baseline]);
  useEffect(() => {
    let active = true;
    if (tab !== "identifiers" || !selected) return;
    Promise.all([
      api.GetBedrockSnapshot(selected),
      baseline && baseline !== selected
        ? api.GetBedrockSnapshot(baseline)
        : Promise.resolve(null),
    ])
      .then(([a, b]) => {
        if (active)
          setIDs(
            identifierChanges(
              (b?.files || []).flatMap((f) => f.identifiers || []),
              (a.files || []).flatMap((f) => f.identifiers || []),
            ),
          );
      })
      .catch((e) => {
        if (active) setError(failure(e));
      });
    return () => {
      active = false;
    };
  }, [tab, selected, baseline]);
  useEffect(() => {
    const sequence = ++previewSequence.current;
    setPreview(null);
    setBefore(null);
    if (!selectedFile || !selected) return;
    const change = changeMap.get(selectedFile);
    const load = async () => {
      const current =
        tab === "changes" && change?.status === "removed"
          ? null
          : await api.PreviewBedrockFile(selected, selectedFile);
      const old =
        tab === "changes" && baseline && change?.before
          ? await api.PreviewBedrockFile(baseline, selectedFile)
          : null;
      if (sequence === previewSequence.current) {
        setPreview(current);
        setBefore(old);
      }
    };
    void load().catch((e) => {
      if (sequence === previewSequence.current) setError(failure(e));
    });
  }, [selectedFile, selected, baseline, tab, changes]);
  const visibleChanges = useMemo(
    () => fileChanges(changes, query, formatting),
    [changes, query, formatting],
  );
  const rows: LabFile[] =
    tab === "changes"
      ? visibleChanges
          .slice(offset, offset + 200)
          .map((c) => c.after || c.before!)
      : page.files;
  const totalRows = tab === "changes" ? visibleChanges.length : page.total;
  const working = busy || progress.running;
  async function capture() {
    await api.CaptureBedrockSnapshot({
      installation: source ? "" : installation,
      source,
      name,
      version,
      channel,
      schemas: schemaPath,
    });
    previousProgress.current = true;
    setSnapshotOpen(false);
    setProgress((p) => ({ ...p, running: true, phase: "Starting", error: "" }));
  }
  async function chooseFolder(purpose: string) {
    const p = await api.SelectBedrockFolder(purpose);
    if (p) {
      if (purpose === "schemas") setSchemaPath(p);
      else setSource(p);
    }
  }
  return (
    <section
      className={`lab-workspace${propertiesOpen ? " properties-open" : ""}`}
      aria-label="Archive"
    >
      <header className="lab-heading studio-view-heading">
        <div>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            aria-label="Properties"
            title="Properties"
            aria-pressed={propertiesOpen}
            onClick={() => setPropertiesOpen((open) => !open)}
          >
            <HeaderActionIcon label="Properties" />
            <span className="toolbar-action-label">Properties</span>
          </Button>
          <Archive size={16} />
          <strong>Archive</strong>
          <span>
            {snapshot
              ? `${snapshot.count.toLocaleString()} files · ${snapshot.version || snapshot.channel || "Local snapshot"}`
              : "Preview archives and version research"}
          </span>
        </div>
        <div>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            aria-label="New snapshot"
            title="New snapshot"
            disabled={working}
            onClick={() => setSnapshotOpen(true)}
          >
            <Plus size={15} />
            <span className="toolbar-action-label">New snapshot</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            aria-label="Choose game data folder"
            title="Choose game data folder"
            disabled={working}
            onClick={() =>
              void run(async () => {
                await chooseFolder("source");
                setSnapshotOpen(true);
              })
            }
          >
            <FolderOpen size={15} />
            <span className="toolbar-action-label">
              Choose game data folder
            </span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            aria-label="Refresh Archive"
            title="Refresh library and installations"
            disabled={working}
            onClick={() =>
              void run(async () => {
                await Promise.all([refresh(), detect()]);
              })
            }
          >
            <RefreshCw size={14} />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            aria-label="Export snapshot"
            title="Export snapshot"
            disabled={!selected || working}
            onClick={() =>
              void run(async () => {
                const p = await api.ExportBedrockSnapshot(selected);
                if (p) setNotice(`Saved ${p}`);
              })
            }
          >
            <Download size={14} />
            <span className="toolbar-action-label">Export snapshot</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            title="Delete snapshot"
            disabled={!selected || working}
            aria-label="Delete snapshot"
            onClick={() => setDeleteID(selected)}
          >
            <Trash2 size={14} />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="toolbar-icon-action"
            aria-label="Sources and licenses"
            title="Sources and licenses"
            onClick={() => setNoticesOpen(true)}
          >
            <Info size={15} />
            <span className="toolbar-action-label">Sources and licenses</span>
          </Button>
        </div>
      </header>
      {(error || notice) && (
        <div
          className={error ? "lab-banner lab-warning" : "lab-banner"}
          role={error ? "alert" : "status"}
        >
          <span>{error || notice}</span>
          <button
            onClick={() => {
              setError("");
              setNotice("");
            }}
          >
            Dismiss
          </button>
        </div>
      )}
      <div className="lab-body">
        <aside
          className="studio-sidebar lab-sidebar"
          aria-label="Snapshot library"
        >
          <h3>
            Snapshot library <small>{snapshots.length}</small>
          </h3>
          <div className="lab-snapshot-search">
            <Search size={15} />
            <input
              aria-label="Search snapshots"
              placeholder="Search snapshots"
              value={snapshotSearch}
              onChange={(e) => setSnapshotSearch(e.target.value)}
            />
          </div>
          <Dialog open={snapshotOpen} onOpenChange={setSnapshotOpen}>
            <DialogContent className="lab-snapshot-dialog">
              <DialogHeader>
                <DialogTitle>New snapshot</DialogTitle>
                <DialogDescription>
                  Choose an installed Minecraft version or a local game data
                  folder to inspect.
                </DialogDescription>
              </DialogHeader>
              <div className="studio-dialog-body lab-source">
                <DialogDescription>
                  Choose an installed Minecraft version or a local game data
                  folder to inspect.
                </DialogDescription>
                <label>
                  Source
                  <select
                    aria-label="Minecraft source"
                    value={source ? "folder" : installation}
                    onChange={(e) => {
                      if (e.target.value === "folder") {
                        void run(() => chooseFolder("source"));
                      } else {
                        setSource("");
                        setInstallation(e.target.value);
                      }
                    }}
                  >
                    <option value="">Choose Minecraft installation</option>
                    {installations
                      .filter((i) => i.location)
                      .map((i) => (
                        <option key={i.id} value={i.id}>
                          {i.name} · {i.gameVersion || i.version}
                        </option>
                      ))}
                    <option value="folder">Local game data folder…</option>
                  </select>
                </label>
                {source && (
                  <div className="lab-source-path">
                    <code title={source}>{source}</code>
                    <Button
                      variant="outline"
                      size="icon-sm"
                      title="Choose another folder"
                      onClick={() => void run(() => chooseFolder("source"))}
                    >
                      <FolderOpen size={13} />
                    </Button>
                  </div>
                )}
                <label>
                  Snapshot name
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Automatic from source"
                  />
                </label>
                <div className="lab-pair">
                  <label>
                    Version
                    <input
                      value={version}
                      onChange={(e) => setVersion(e.target.value)}
                      placeholder={
                        installations.find((i) => i.id === installation)
                          ?.gameVersion || "Optional"
                      }
                    />
                  </label>
                  <label>
                    Channel
                    <select
                      value={channel}
                      onChange={(e) => setChannel(e.target.value)}
                    >
                      <option value="preview">Preview</option>
                      <option value="stable">Stable</option>
                    </select>
                  </label>
                </div>
                <Button
                  variant="outline"
                  className="lab-schema-button"
                  onClick={() => setSchemaOpen(true)}
                >
                  <FileCode size={14} />
                  {schemaPath ? "Schemas selected" : "Configure MCB schemas"}
                </Button>
                <small>
                  Game files are read without changing the installation.
                </small>
                {error && (
                  <p className="lab-warning" role="alert">
                    {error}
                  </p>
                )}
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setSnapshotOpen(false)}
                >
                  Close
                </Button>
                <Button
                  disabled={working || (!source && !installation)}
                  onClick={() => void run(capture)}
                >
                  <Archive size={14} />
                  Create snapshot
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <nav className="lab-snapshots" aria-label="Snapshots">
            {snapshots
              .filter((s) =>
                `${s.name} ${s.version} ${s.channel}`
                  .toLowerCase()
                  .includes(snapshotSearch.toLowerCase()),
              )
              .map((s) => (
                <button
                  key={s.id}
                  aria-current={s.id === selected ? "page" : undefined}
                  data-selected={s.id === selected}
                  onClick={() => setSelected(s.id)}
                >
                  <Archive size={15} />
                  <span>
                    <b>{s.name}</b>
                    <small>
                      {s.version || s.channel || "Local"} ·{" "}
                      {s.count.toLocaleString()} files
                    </small>
                    <small>{new Date(s.created).toLocaleString()}</small>
                  </span>
                </button>
              ))}
            {!snapshots.length && (
              <div className="lab-placeholder">
                <p>No snapshots yet.</p>
                <button onClick={() => setSnapshotOpen(true)}>
                  <Plus size={14} />
                  Create first snapshot
                </button>
              </div>
            )}
          </nav>
          <footer>
            <span>Local snapshots</span>
            <span>{snapshots.length}</span>
          </footer>
        </aside>
        <main className="lab-main">
          <header className="lab-document-heading">
            <div>
              <b>{snapshot?.name || "Inspect Bedrock versions"}</b>
              <small>
                {snapshot
                  ? new Date(snapshot.created).toLocaleString()
                  : "Unpack BRArchives, restore compiled JSON and compare changes."}
              </small>
            </div>
            <label className="lab-baseline">
              <ArrowLeftRight size={14} />
              Compare with
              <select
                aria-label="Baseline snapshot"
                value={baseline}
                onChange={(e) => setBaseline(e.target.value)}
              >
                <option value="">No baseline</option>
                {snapshots
                  .filter((s) => s.id !== selected)
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
              </select>
            </label>
          </header>
          <div
            className="lab-tabs"
            role="tablist"
            aria-label="Bedrock analysis"
          >
            {tabs.map((t) => (
              <button
                key={t.id}
                role="tab"
                aria-selected={t.id === tab}
                onClick={() => {
                  setTab(t.id);
                  setSelectedFile("");
                }}
              >
                {t.label}
                {t.id === "changes" && changes.length > 0 && (
                  <small>
                    {changes.filter((c) => c.status !== "formatting").length}
                  </small>
                )}
              </button>
            ))}
          </div>
          <div className="lab-tools">
            <label className="lab-search">
              <Search size={14} />
              <input
                aria-label="Search Archive"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={
                  tab === "strings"
                    ? "Find strings in the installed executable…"
                    : tab === "identifiers"
                      ? "Search namespaced identifiers…"
                      : "Search paths or file contents…"
                }
              />
            </label>
            {tab === "files" && (
              <>
                <select
                  aria-label="File kind"
                  value={kind}
                  onChange={(e) => setKind(e.target.value)}
                >
                  <option value="">All file types</option>
                  {["json", "mcb", "image", "text", "archive", "binary"].map(
                    (k) => (
                      <option key={k}>{k}</option>
                    ),
                  )}
                </select>
                <label>
                  <input
                    type="checkbox"
                    checked={content}
                    onChange={(e) => setContent(e.target.checked)}
                  />
                  File contents
                </label>
              </>
            )}
            {tab === "changes" && (
              <label>
                <input
                  type="checkbox"
                  checked={formatting}
                  onChange={(e) => setFormatting(e.target.checked)}
                />
                Include formatting
              </label>
            )}
            {tab === "strings" && (
              <>
                <button
                  disabled={working}
                  onClick={() =>
                    void run(async () => {
                      const p = await api.SelectBedrockExecutable();
                      if (p) setBinary(p);
                    })
                  }
                >
                  <FolderOpen size={13} />
                  Open readable binary
                </button>
                <button
                  disabled={working || !installation}
                  onClick={() =>
                    void run(async () => {
                      setBinary(
                        await api.PrepareBedrockExecutable(installation),
                      );
                      setNotice(
                        "Readable executable copied into the local library",
                      );
                    })
                  }
                >
                  Create readable copy
                </button>
                <button
                  disabled={
                    busy ||
                    (!installation && !binary) ||
                    query.trim().length < 2
                  }
                  onClick={() =>
                    void run(async () =>
                      setStrings(
                        binary
                          ? await api.SearchBedrockBinary(binary, query)
                          : await api.SearchBedrockExecutable(
                              installation,
                              query,
                            ),
                      ),
                    )
                  }
                >
                  <Search size={13} />
                  Search executable
                </button>
                {binary && (
                  <button
                    title={binary}
                    onClick={() => {
                      setBinary("");
                      setStrings([]);
                    }}
                  >
                    Use installed executable
                  </button>
                )}
              </>
            )}
            {tab === "validation" && (
              <button
                disabled={!selected || working}
                onClick={() =>
                  void run(async () =>
                    setValidation(await api.ValidateBedrockSnapshot(selected)),
                  )
                }
              >
                <ShieldCheck size={14} />
                Verify stored files
              </button>
            )}
          </div>
          {!snapshot && tab !== "strings" ? (
            <div className="lab-empty">
              <Archive size={32} />
              <h2>A library for Bedrock discoveries</h2>
              <p>
                Create a snapshot from Minecraft Preview or Stable. Packed files
                are expanded automatically. Add matching BDS schemas to restore
                MCB files as readable JSON.
              </p>
              <button onClick={() => setSnapshotOpen(true)}>
                <FolderOpen size={14} />
                Choose game data folder
              </button>
            </div>
          ) : tab === "strings" ? (
            <div className="lab-string-results">
              <p className="lab-hint">
                Searches printable ASCII and UTF-16LE ASCII strings with byte
                offsets, up to 1,000 matches. If Windows denies access, create a
                readable copy in the installed game's package context or open an
                existing binary.
              </p>
              {binary && (
                <p className="lab-hint">
                  <code>{binary}</code>
                </p>
              )}
              {strings.length ? (
                <table>
                  <thead>
                    <tr>
                      <th>Offset</th>
                      <th>Encoding</th>
                      <th>Text</th>
                    </tr>
                  </thead>
                  <tbody>
                    {strings.map((s, i) => (
                      <tr key={i}>
                        <td>
                          <code>0x{s.offset.toString(16)}</code>
                        </td>
                        <td>{s.encoding}</td>
                        <td>
                          <code>{s.text}</code>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="lab-placeholder">
                  Enter at least two characters and search the executable.
                </p>
              )}
            </div>
          ) : (
            <>
              {tab === "identifiers" && (
                <div className="lab-identifier-summary">
                  <span>
                    <Hash size={14} />
                    {base
                      ? `Identifiers compared with ${base.name}`
                      : "Identifiers in this snapshot"}
                  </span>
                  <details>
                    <summary>
                      {ids.added.length} {base ? "added" : "found"} ·{" "}
                      {ids.removed.length} removed
                    </summary>
                    <div>
                      {ids.added.slice(0, 500).map((id) => (
                        <code className="lab-added" key={id}>
                          + {id}
                        </code>
                      ))}
                      {ids.removed.slice(0, 500).map((id) => (
                        <code className="lab-removed" key={id}>
                          − {id}
                        </code>
                      ))}
                    </div>
                  </details>
                </div>
              )}
              {tab === "validation" && (
                <div className="lab-validation-summary">
                  {validation ? (
                    <>
                      <Check size={15} />
                      {validation.checked.toLocaleString()} files verified ·{" "}
                      {validation.issues.length} findings
                      <details>
                        <summary>Validation report</summary>
                        <pre>
                          {validation.issues.join("\n") ||
                            "All stored file hashes and sizes match."}
                        </pre>
                      </details>
                    </>
                  ) : (
                    <p>
                      Review decoding and file issues below. Use Verify stored
                      files to check every saved hash and size.
                    </p>
                  )}
                  {(snapshot?.issues || []).map((issue, i) => (
                    <p key={i}>{issue}</p>
                  ))}
                </div>
              )}
              {tab === "changes" && (!baseline || baseline === selected) ? (
                <div className="lab-empty">
                  <ArrowLeftRight size={28} />
                  <h2>Choose a baseline</h2>
                  <p>
                    Select another snapshot above to compare added, removed,
                    modified and reformatted files.
                  </p>
                </div>
              ) : (
                <div
                  className={`lab-inspection ${tab === "changes" ? "lab-comparison" : ""}`}
                >
                  <div className="lab-file-list">
                    <header>
                      <span>
                        {tab === "changes"
                          ? `${visibleChanges.length} changes`
                          : `${page.total.toLocaleString()} files`}
                      </span>
                      {loading && <LoaderCircle size={13} className="spin" />}
                    </header>
                    <div role="list" aria-label="Snapshot files">
                      {rows.map((f) => (
                        <button
                          role="listitem"
                          key={f.path}
                          aria-current={
                            selectedFile === f.path ? "true" : undefined
                          }
                          onClick={() => setSelectedFile(f.path)}
                        >
                          <FileCode size={14} />
                          <span>
                            <b title={f.path}>{f.path}</b>
                            <small>
                              {tab === "changes"
                                ? changeMap.get(f.path)?.status
                                : f.kind}{" "}
                              · {labBytes(f.bytes)}
                              {f.origin ? " · unpacked" : ""}
                            </small>
                          </span>
                          {f.issue && <TriangleAlert size={13} />}
                        </button>
                      ))}
                      {!rows.length && !loading && (
                        <p className="lab-placeholder">
                          {tab === "changes"
                            ? "No matching changes."
                            : "No matching files."}
                        </p>
                      )}
                    </div>
                    {
                      <footer>
                        <button
                          aria-label="Previous files"
                          disabled={offset === 0}
                          onClick={() => setOffset(Math.max(0, offset - 200))}
                        >
                          <ChevronLeft size={14} />
                        </button>
                        <small>
                          {totalRows
                            ? `${Math.min(offset + 1, totalRows)}–${Math.min(offset + 200, totalRows)} of ${totalRows}`
                            : "0 files"}
                        </small>
                        <button
                          aria-label="Next files"
                          disabled={offset + 200 >= totalRows}
                          onClick={() => setOffset(offset + 200)}
                        >
                          <ChevronRight size={14} />
                        </button>
                      </footer>
                    }
                    {page.truncated && (
                      <p className="lab-warning">
                        Search reached its content limit. Narrow the file type
                        or search term.
                      </p>
                    )}
                  </div>
                  <div className="lab-preview-grid">
                    {tab === "changes" && (
                      <FileView
                        preview={before}
                        label={base?.name || "Before"}
                      />
                    )}
                    <FileView
                      preview={preview}
                      label={
                        tab === "changes"
                          ? snapshot?.name || "After"
                          : "File preview"
                      }
                    />
                  </div>
                </div>
              )}
            </>
          )}
          <footer className="lab-status">
            <span>
              {progress.running ? (
                <LoaderCircle className="spin" size={13} />
              ) : (
                <Check size={13} />
              )}{" "}
              {progress.running
                ? `${progress.phase}${progress.total ? ` · ${progress.done}/${progress.total}` : ""}`
                : busy
                  ? "Working…"
                  : "Local version library"}
              <small title={progress.path}>
                {progress.running ? progress.path : snapshot?.source}
              </small>
            </span>
            {progress.running && (
              <button onClick={() => void run(() => api.CancelBedrockLab())}>
                <Square size={12} />
                Cancel
              </button>
            )}
          </footer>
        </main>
        <aside className="lab-properties" aria-label="Selection properties">
          <h3>Properties</h3>
          {file ? (
            <>
              <h4>Selected file</h4>
              <dl>
                <dt>Path</dt>
                <dd>{file.path}</dd>
                <dt>Type</dt>
                <dd>{file.kind}</dd>
                <dt>Size</dt>
                <dd>{labBytes(file.bytes)}</dd>
                <dt>Origin</dt>
                <dd>{file.origin || "Source file"}</dd>
                <dt>SHA-256</dt>
                <dd>
                  <code>{file.hash}</code>
                </dd>
                {file.rawHash && (
                  <>
                    <dt>Compiled original</dt>
                    <dd>
                      <code>{file.rawHash}</code>
                    </dd>
                  </>
                )}
              </dl>
              {file.issue && <p className="lab-warning">{file.issue}</p>}
              <button
                disabled={!preview?.text}
                onClick={() =>
                  void run(
                    () => navigator.clipboard.writeText(preview?.text || ""),
                    "File contents copied",
                  )
                }
              >
                <Copy size={13} />
                Copy contents
              </button>
              <h4>Identifiers</h4>
              <button
                disabled={working || !preview}
                onClick={() =>
                  void run(async () => {
                    const items = await api.ImportBedrockPack(
                      selected,
                      file.path,
                    );
                    if (!items.length)
                      throw Error("The pack could not be imported");
                    window.dispatchEvent(
                      new CustomEvent("craftbox:open-content", {
                        detail: {
                          kind: "packs",
                          id: items[0].id,
                          name: items[0].name,
                        },
                      }),
                    );
                  })
                }
              >
                <Archive size={13} />
                Open pack in CraftBox
              </button>
              <div className="lab-identifiers">
                {(file.identifiers || []).slice(0, 300).map((id) => (
                  <button
                    key={id}
                    onClick={() => {
                      setTab("identifiers");
                      setQuery(id);
                    }}
                  >
                    <code>{id}</code>
                  </button>
                ))}
              </div>
            </>
          ) : snapshot ? (
            <>
              <h4>{snapshot.name}</h4>
              <dl>
                <dt>Version</dt>
                <dd>{snapshot.version || "Unspecified"}</dd>
                <dt>Channel</dt>
                <dd>{snapshot.channel || "Local"}</dd>
                <dt>Files</dt>
                <dd>{snapshot.count.toLocaleString()}</dd>
                <dt>Size</dt>
                <dd>{labBytes(snapshot.bytes)}</dd>
                <dt>BRArchives</dt>
                <dd>{snapshot.archives}</dd>
                <dt>MCB restored</dt>
                <dd>{snapshot.decoded}</dd>
                <dt>MCB unresolved</dt>
                <dd>{snapshot.unresolved}</dd>
                <dt>Schemas</dt>
                <dd>{snapshot.schemaSource || "None selected"}</dd>
              </dl>
              <p className="lab-hint">
                Snapshots preserve compiled originals alongside restored JSON.
                Export includes a provenance manifest.
              </p>
            </>
          ) : (
            <p className="lab-placeholder">
              Select a snapshot or file to inspect its properties.
            </p>
          )}
        </aside>
      </div>
      <Dialog open={noticesOpen} onOpenChange={setNoticesOpen}>
        <DialogContent className="lab-notices-dialog">
          <DialogHeader>
            <DialogTitle>Archive sources and licenses</DialogTitle>
            <DialogDescription>
              Archive format references and notices for the native MCB
              implementation.
            </DialogDescription>
          </DialogHeader>
          <div className="studio-dialog-body lab-notices-body">
            <DialogDescription>
              Archive format references and notices for the native MCB
              implementation.
            </DialogDescription>
            <pre>{bedrockLabNotices}</pre>
          </div>
          <DialogFooter>
            <Button onClick={() => setNoticesOpen(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={schemaOpen} onOpenChange={setSchemaOpen}>
        <DialogContent className="lab-schema-dialog">
          <DialogHeader>
            <DialogTitle>MCB schemas</DialogTitle>
            <DialogDescription>
              Use schemas generated by the BDS version matching your game build.
              CraftBox can download the official Windows server and generate
              them in an isolated folder.
            </DialogDescription>
          </DialogHeader>
          <div className="studio-dialog-body lab-dialog-fields">
            <DialogDescription>
              Use schemas generated by the BDS version matching your game build.
              CraftBox can download the official Windows server and generate
              them in an isolated folder.
            </DialogDescription>
            <label>
              Exact BDS version
              <input
                aria-label="BDS version"
                value={schemaVersion}
                onChange={(e) => setSchemaVersion(e.target.value)}
                placeholder="1.26.52.3"
              />
            </label>
            <label>
              Release channel
              <select
                value={channel}
                onChange={(e) => setChannel(e.target.value)}
              >
                <option value="preview">Preview</option>
                <option value="stable">Stable</option>
              </select>
            </label>
            <Button
              variant="outline"
              disabled={working}
              onClick={() =>
                void run(async () => {
                  const releases = await api.GetBedrockServerReleases();
                  const release = releases.find((r) => r.channel === channel);
                  if (!release)
                    throw Error(
                      "No official BDS release is listed for this channel",
                    );
                  setSchemaVersion(release.version);
                  setNotice(
                    `Latest official ${channel} BDS: ${release.version}. Check that it matches the game data before decoding.`,
                  );
                })
              }
            >
              <RefreshCw size={14} />
              Find latest official version
            </Button>
            <Button
              variant="outline"
              disabled={working || !/^\d+\.\d+\.\d+\.\d+$/.test(schemaVersion)}
              onClick={() =>
                void run(async () => {
                  await api.PrepareBedrockSchemas({
                    version: schemaVersion,
                    channel,
                  });
                  previousProgress.current = true;
                  setProgress((p) => ({
                    ...p,
                    running: true,
                    phase: "Starting BDS download",
                  }));
                })
              }
            >
              <Download size={14} />
              Download BDS and generate schemas
            </Button>
            <small>
              The helper runs without a visible console and stops after
              generation. Logs and download provenance stay in the local
              library.
            </small>
            <h4>Available schema bundles</h4>
            <select
              aria-label="MCB schema bundle"
              value={schemaPath}
              onChange={(e) => setSchemaPath(e.target.value)}
            >
              <option value="">Keep MCB files compiled</option>
              {schemas.map((s) => (
                <option key={s.path} value={s.path}>
                  {s.channel} {s.version} · {s.count} schemas
                </option>
              ))}
              {schemaPath && !schemas.some((s) => s.path === schemaPath) && (
                <option value={schemaPath}>Imported schema folder</option>
              )}
            </select>
            <code>{schemaPath}</code>
            <Button
              variant="outline"
              disabled={working}
              onClick={() => void run(() => chooseFolder("schemas"))}
            >
              <FolderOpen size={14} />
              Use existing schema folder
            </Button>
            {progress.running && <p role="status">{progress.phase}</p>}
          </div>
          <DialogFooter>
            <Button onClick={() => setSchemaOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={!!deleteID}
        onOpenChange={(open) => {
          if (!open) setDeleteID("");
        }}
      >
        <DialogContent tone="confirmation" resizable={false}>
          <DialogHeader>
            <DialogTitle>Delete snapshot?</DialogTitle>
            <DialogDescription>
              {snapshots.find((s) => s.id === deleteID)?.name} will be removed
              from the library. Shared file objects remain available to other
              snapshots.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteID("")}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={working}
              onClick={() =>
                void run(async () => {
                  await api.DeleteBedrockSnapshot(deleteID);
                  setDeleteID("");
                  await refresh();
                })
              }
            >
              Delete snapshot
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}

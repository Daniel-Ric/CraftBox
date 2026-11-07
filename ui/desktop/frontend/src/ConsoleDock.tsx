import {notifyAction,notifyFailure} from "./StudioFlyouts";
import ExportTasks from "./ExportTasks";
import ConsoleCommandInput from "./ConsoleCommandInput";
import CraftBoxCLI from "./CraftBoxCLI";
import {parseCommand} from "./craftbox-cli";
import StudioInput from "@/StudioInput";
import StudioPathInput from "./StudioPathInput";
import {
  lazy,
  Suspense,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  ArrowDown,
  ArrowUp,
  Bookmark,
  CaseSensitive,
  ChevronDown,
  Copy,
  Download,
  FileText,
  Filter,
  Maximize2,
  Minimize2,
  Minus,
  Pause,
  Play,
  Plus,
  Search,
  Terminal,
  Trash2,
  WrapText,
  X,
  Clock,
  ListOrdered,
  ListCollapse,
} from "lucide-react";
import type { Log } from "./bridge";
import PanelResizeHandle from "./PanelResizeHandle";
import {
  consolePurposes,
  createConsoleTab,
  formatConsoleLog,
  normalizeLogLevel,
  parseConsoleTabs,
  selectConsoleLogs,
  type ConsolePurpose,
  type ConsoleTab,
} from "./console-model";
import "./console-dock.css";
const ShellTerminal = lazy(() => import("./ShellTerminal"));
const key = "CraftBox-console-tabs-v1";
type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  logs: Log[];
  sessionName: string;
  running: boolean;
  onNavigate: (page: string) => void;
  onExport: (name: string, text: string) => Promise<unknown>;
  diagnostics?: string;
  onStop?: () => Promise<void>;
};
const tool = (
  label: string,
  icon: ReactNode,
  run: () => void,
  pressed?: boolean,
  disabled = false,
) => (
  <button
    type="button"
    aria-label={label}
    data-tooltip={label}
    aria-pressed={pressed}
    onClick={run}
    disabled={disabled}
  >
    {icon}
  </button>
);

export default function ConsoleDock(p: Props) {
  const [tabs, setTabs] = useState(() =>
      parseConsoleTabs(localStorage.getItem(key)),
    ),
    [selected, select] = useState("");
  const [purpose, setPurpose] = useState<ConsolePurpose>("all"),
    [height, setHeight] = useState(
      () => Number(localStorage.getItem("CraftBox-console-height")) || 280,
    ),
    [maximized, maximize] = useState(false),
    [available, setAvailable] = useState(window.innerHeight);
  const [renaming, rename] = useState<string>(),
    [paused, setPaused] = useState<Record<string, Log[]>>({}),
    [newCwd, setNewCwd] = useState(""),
    [cwdValid, setCwdValid] = useState(true),
    [settings, setSettings] = useState(false);
  const host = useRef<HTMLElement>(null);
  const active = tabs.find((t) => t.id === selected) ?? tabs[0];
  useEffect(() => {
    try {
      localStorage.setItem(
        key,
        JSON.stringify(tabs.filter((t) => t.kind === "console")),
      );
      localStorage.setItem("CraftBox-console-height", String(height));
    } catch {
      /* Storage limits must not interrupt a capture. */
    }
  }, [tabs, height]);
  useEffect(() => {
    const update = () =>
      setAvailable(
        Math.max(
          180,
          host.current?.parentElement?.clientHeight || window.innerHeight - 130,
        ),
      );
    const observer =
      typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(update)
        : undefined;
    if (host.current?.parentElement)
      observer?.observe(host.current.parentElement);
    window.addEventListener("resize", update);
    update();
    return () => {
      window.removeEventListener("resize", update);
      observer?.disconnect();
    };
  }, []);
  function update(changes: Partial<ConsoleTab>, id = active.id) {
    setTabs((previous) =>
      previous.map((t) => (t.id === id ? { ...t, ...changes } : t)),
    );
  }
  function add(kind: "console" | "powershell") {
    if (tabs.length >= 16 || (kind === "powershell" && !cwdValid)) return;
    const tab = createConsoleTab(purpose);
    tab.kind = kind;
    if (kind === "powershell") tab.cwd = newCwd;
    const base =
      kind === "powershell" ? "PowerShell" : consolePurposes[purpose];
    let n = kind === "powershell" ? 1 : 0;
    let name = n ? `${base} ${n}` : base;
    while (tabs.some((t) => t.name === name)) {
      n++;
      name = `${base} ${n}`;
    }
    tab.name = name;
    setTabs((previous) => [...previous, tab]);
    select(tab.id);
    p.onOpenChange(true);
  }
  function close(id: string) {
    const next = tabs.filter((t) => t.id !== id);
    if (!next.length) next.push(createConsoleTab());
    setTabs(next);
    if (active.id === id)
      select(
        next[Math.max(0, tabs.findIndex((t) => t.id === id) - 1)]?.id ??
          next[0].id,
      );
    setPaused((v) => {
      const copy = { ...v };
      delete copy[id];
      return copy;
    });
  }
  const shellCount = tabs.filter((t) => t.kind === "powershell").length;
  return (
    <section
      ref={host}
      className={`console-dock ${p.open ? "expanded" : ""} ${maximized && p.open ? "console-maximized" : ""}`}
      style={
        p.open && !maximized
          ? { height: Math.min(height, available) }
          : undefined
      }
      aria-label="Console tool window"
      onKeyDown={(e) => {
        if (e.shiftKey && e.key === "Escape") {
          e.preventDefault();
          p.onOpenChange(false);
        }
        if (e.ctrlKey && e.shiftKey && e.key === "F12") {
          e.preventDefault();
          maximize((v) => !v);
        }
      }}
    >
      {p.open && !maximized && (
        <PanelResizeHandle
          label="Console height"
          value={Math.min(height, available)}
          min={180}
          max={available}
          onChange={setHeight}
        />
      )}
      <header className="console-dock-header">
        <button
          type="button"
          className="console-toggle"
          onClick={() => p.onOpenChange(!p.open)}
          aria-expanded={p.open}
        >
          <Terminal size={14} />
          <strong>Console</strong>
          <span>{p.logs.length}</span>
          <ChevronDown size={13} className={p.open ? "rotate" : ""} />
        </button>
        {p.open && (
          <>
            <div
              className="console-tabs"
              role="tablist"
              aria-label="Console sessions"
            >
              {tabs.map((tab) => (
                <div
                  className={`console-tab ${active.id === tab.id ? "selected" : ""}`}
                  key={tab.id}
                >
                  {renaming === tab.id ? (
                    <StudioInput
                      autoFocus
                      aria-label="Console tab name"
                      value={tab.name}
                      maxLength={60}
                      onChange={(e) =>
                        setTabs((v) =>
                          v.map((t) =>
                            t.id === tab.id
                              ? { ...t, name: e.target.value }
                              : t,
                          ),
                        )
                      }
                      onBlur={() => rename(undefined)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === "Escape")
                          rename(undefined);
                      }}
                    />
                  ) : (
                    <button
                      type="button"
                      role="tab"
                      aria-selected={active.id === tab.id}
                      aria-controls={`console-pane-${tab.id}`}
                      onClick={() => select(tab.id)}
                      onDoubleClick={() => rename(tab.id)}
                      data-tooltip="Double-click to rename"
                    >
                      <Terminal size={12} />
                      <span>{tab.name}</span>
                      {tab.kind === "console" && p.running && (
                        <i className="console-dot running" />
                      )}
                    </button>
                  )}
                  <button
                    type="button"
                    className="console-close-tab"
                    aria-label={`Close ${tab.name}`}
                    data-tooltip={
                      tab.kind === "powershell"
                        ? "Close terminal and end its processes"
                        : "Close console view; the capture keeps running"
                    }
                    onClick={() => close(tab.id)}
                  >
                    <X size={12} />
                  </button>
                </div>
              ))}
            </div>
            <div className="console-window-actions">
              {tool(
                "New CraftBox console",
                <Plus size={15} />,
                () => add("console"),
                undefined,
                tabs.length >= 16,
              )}
              {tool(
                "New PowerShell terminal",
                <Terminal size={15} />,
                () => add("powershell"),
                undefined,
                tabs.length >= 16 || shellCount >= 8 || !cwdValid,
              )}
              {tool(
                "Console tab options",
                <Filter size={14} />,
                () => setSettings((v) => !v),
                settings,
              )}
              {tool(
                maximized ? "Restore console size" : "Maximize console",
                maximized ? <Minimize2 size={14} /> : <Maximize2 size={14} />,
                () => maximize((v) => !v),
              )}
              {tool("Hide console (Shift+Escape)", <Minus size={15} />, () =>
                p.onOpenChange(false),
              )}
            </div>
          </>
        )}
        <ExportTasks/>
      </header>
      {p.open && (
        <div
          className={`console-create-options ${settings ? "show-options" : ""}`}
        >
          <label>
            New console purpose
            <select
              aria-label="New console purpose"
              value={purpose}
              onChange={(e) => setPurpose(e.target.value as ConsolePurpose)}
            >
              {Object.entries(consolePurposes).map(([id, label]) => (
                <option key={id} value={id}>
                  {label}
                </option>
              ))}
            </select>
          </label>
          <label className="console-cwd">
            New PowerShell directory
            <StudioPathInput
              aria-label="New PowerShell working directory"
              kind="directory"
              onValidityChange={setCwdValid}
              placeholder="Application folder"
              value={newCwd}
              onChange={(e) => {setNewCwd(e.target.value);setCwdValid(!e.target.value)}}
            />
          </label>
          <span>Double-click a tab to rename it.</span>
        </div>
      )}
      <div className="console-panes" hidden={!p.open}>
        {tabs
          .filter((t) => t.kind === "console")
          .map((t) => (
            <ConsoleView
              key={t.id}
              active={p.open && active.id === t.id}
              tab={t}
              logs={paused[t.id] ?? p.logs}
              liveLogs={p.logs}
              paused={!!paused[t.id]}
              onPause={() =>
                setPaused((v) => {
                  const next = { ...v };
                  if (next[t.id]) delete next[t.id];
                  else next[t.id] = p.logs.slice();
                  return next;
                })
              }
              onChange={(changes) => update(changes, t.id)}
              onNavigate={p.onNavigate}
              onExport={p.onExport}
              diagnostics={p.diagnostics}
              onStop={p.onStop}
              running={p.running}
              sessionName={p.sessionName}
            />
          ))}
        {tabs
          .filter((t) => t.kind === "powershell")
          .map((t) => (
            <div
              className="console-pane"
              role="tabpanel"
              id={`console-pane-${t.id}`}
              key={t.id}
              hidden={active.id !== t.id}
            >
              <Suspense
                fallback={<p className="console-empty">Loading terminal …</p>}
              >
                <ShellTerminal
                  active={p.open && active.id === t.id}
                  name={t.name}
                  cwd={t.cwd ?? ""}
                  onExport={p.onExport}
                />
              </Suspense>
            </div>
          ))}
      </div>
    </section>
  );
}

function ConsoleView({
  active,
  tab,
  logs,
  liveLogs,
  paused,
  onPause,
  onChange,
  onNavigate,
  onExport,
  diagnostics,
  sessionName,
  onStop,
  running,
}: {
  active: boolean;
  tab: ConsoleTab;
  logs: Log[];
  liveLogs: Log[];
  paused: boolean;
  onPause: () => void;
  onChange: (v: Partial<ConsoleTab>) => void;
  onNavigate: (page: string) => void;
  onExport: Props["onExport"];
  diagnostics?: string;
  sessionName: string;
  onStop?: () => Promise<void>;
  running: boolean;
}) {
  const setNotice=(message:string)=>notifyAction(message,"Console");
  const [cli,setCLI]=useState<{initial:string}>(),
    [help, setHelp] = useState(false),
    [selected, setSelected] = useState<number>(),
    [details, showDetails] = useState(false),
    [bookmarksOnly, onlyBookmarks] = useState(false),
    [filters, showFilters] = useState(false);
  const scroll = useRef<HTMLDivElement>(null),
    lastScroll = useRef(0),
    search = useRef<HTMLInputElement>(null),
    rowsRef = useRef(new Map<number, HTMLDivElement>()),
    expectedScroll = useRef(false);
  const visible = useMemo(
    () =>
      (active ? selectConsoleLogs(logs, tab) : []).filter(
        (l) => !bookmarksOnly || tab.bookmarks.includes(l.id),
      ),
    [logs, tab, bookmarksOnly, active],
  );
  const rows = useMemo(() => {
    const out: { log: Log; count: number }[] = [];
    for (const log of visible) {
      const previous = out.at(-1);
      if (
        tab.fold &&
        previous &&
        previous.log.source === log.source &&
        previous.log.level === log.level &&
        previous.log.message === log.message
      )
        previous.count++;
      else out.push({ log, count: 1 });
    }
    return out;
  }, [visible, tab.fold]);
  const chosen = logs.find((l) => l.id === selected),
    pending = Math.max(0, (liveLogs.at(-1)?.id ?? 0) - (logs.at(-1)?.id ?? 0));
  useEffect(() => {
    if (active && tab.follow && !paused && scroll.current) {
      expectedScroll.current = true;
      scroll.current.scrollTop = scroll.current.scrollHeight;
      requestAnimationFrame(() => (expectedScroll.current = false));
    }
  }, [rows, tab.follow, paused, active]);
  useLayoutEffect(() => {
    if (active && scroll.current && !tab.follow)
      scroll.current.scrollTop = lastScroll.current;
  }, [active]);
  useEffect(() => {
    setSelected(undefined);
  }, [tab.query, tab.source, tab.level]);
  async function copy(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setNotice("Copied to clipboard");
    } catch (e) {
      notifyFailure(e,"Console action failed");
    }
  }
  function follow(value: boolean) {
    if (!value && scroll.current) {
      lastScroll.current = scroll.current.scrollTop;
      expectedScroll.current = false;
    }
    onChange({ follow: value });
  }
  function jump(direction: number) {
    if (!rows.length) return;
    const i = rows.findIndex((r) => r.log.id === selected);
    const next =
      rows[
        i < 0
          ? direction < 0
            ? rows.length - 1
            : 0
          : (i + direction + rows.length) % rows.length
      ].log.id;
    setSelected(next);
    follow(false);
    rowsRef.current.get(next)?.scrollIntoView?.({ block: "nearest" });
  }
  function bookmark(id: number) {
    onChange({
      bookmarks: tab.bookmarks.includes(id)
        ? tab.bookmarks.filter((v) => v !== id)
        : [...tab.bookmarks.slice(-255), id],
    });
  }
  function execute(line:string) {
    const raw = line.trim();
    if (!raw) return;
    let tokens:string[];
    try {tokens=parseCommand(raw)}catch(error){setNotice(String(error));return}
    const verb=tokens[0]?.toLowerCase(),value=verb==="/cli"?raw.slice(4).trim():tokens.slice(1).join(" ");
    switch (verb) {
      case "/cli":
        setCLI({initial:value});
        break;
      case "/help":
        setHelp((v) => !v);
        break;
      case "/clear":
        onChange({ clearedThrough: liveLogs.at(-1)?.id ?? 0 });
        setNotice("Current view cleared");
        break;
      case "/find":
        onChange({ query: value });
        break;
      case "/exclude":
        onChange({ exclude: value });
        break;
      case "/source":
        onChange({ source: value || "all" });
        break;
      case "/level": {
        const level = normalizeLogLevel(value);
        if (
          ["all", "error", "warning", "info", "debug", "trace"].includes(level)
        )
          onChange({ level });
        else setNotice("Use /level all, error, warning, info, debug or trace");
        break;
      }
      case "/purpose":
        if (Object.hasOwn(consolePurposes, value))
          onChange({ purpose: value as ConsolePurpose });
        else setNotice("Purposes: all, problems, network, assets, render");
        break;
      case "/pause":
        if (!paused) onPause();
        break;
      case "/resume":
        if (paused) onPause();
        break;
      case "/follow":
        follow(value !== "off");
        break;
      case "/wrap":
        onChange({ wrap: value !== "off" });
        break;
      case "/copy":
        void copy(visible.map(formatConsoleLog).join("\n"));
        break;
      case "/export":
        void onExport(tab.name, visible.map(formatConsoleLog).join("\n"))
          .then((path) => {
            if (path) setNotice(`Saved: ${path}`);
          })
          .catch((e) => notifyFailure(e,"Console action failed"));
        break;
      case "/bookmark":
        if (selected !== undefined) bookmark(selected);
        else setNotice("Select an entry first");
        break;
      case "/open":
        if (["worlds", "packs", "skins", "render", "cinematic", "packets", "settings"].includes(value))
          onNavigate(value);
        else setNotice("Open: worlds, packs, skins, render or cinematic");
        break;
      case "/stop":
        if (onStop && running)
          void onStop()
            .then(() => setNotice("Session stopped"))
            .catch((e) => notifyFailure(e,"Console action failed"));
        else setNotice("No running CraftBox session");
        break;
      default:
        setNotice(
          "Unknown command. Use /help; shell commands need PowerShell.",
        );
    }
  }
  const sources = [
    ...new Set(logs.map((l) => l.source).filter(Boolean)),
  ].sort();
  const mark = (text: string) => {
    if (!tab.query) return text;
    const hay = tab.matchCase ? text : text.toLowerCase(),
      needle = tab.matchCase ? tab.query : tab.query.toLowerCase();
    const out: ReactNode[] = [];
    let offset = 0;
    for (let i = hay.indexOf(needle); i >= 0; i = hay.indexOf(needle, offset)) {
      out.push(
        text.slice(offset, i),
        <mark key={i}>{text.slice(i, i + needle.length)}</mark>,
      );
      offset = i + needle.length;
    }
    out.push(text.slice(offset));
    return out;
  };
  if(cli)return <div className="console-pane" role="tabpanel" id={`console-pane-${tab.id}`} hidden={!active}><CraftBoxCLI logs={liveLogs} history={tab.history} onHistory={history=>onChange({history})} onExit={()=>setCLI(undefined)} onNavigate={onNavigate} onExport={onExport} onStop={onStop} initialCommand={cli.initial} fontSize={tab.fontSize}/></div>;
  if (!active)
    return <div hidden role="tabpanel" id={`console-pane-${tab.id}`} />;
  return (
    <div
      className="console-view console-pane"
      role="tabpanel"
      id={`console-pane-${tab.id}`}
      onKeyDown={(e) => {
        if (e.ctrlKey && e.key.toLowerCase() === "f") {
          e.preventDefault();
          search.current?.focus();
        }
        if (e.key === "F3") {
          e.preventDefault();
          jump(e.shiftKey ? -1 : 1);
        }
        if (e.ctrlKey && e.key.toLowerCase() === "l") {
          e.preventDefault();
          onChange({ clearedThrough: liveLogs.at(-1)?.id ?? 0 });
        }
      }}
    >
      <div
        className="console-toolbar"
        role="toolbar"
        aria-label="CraftBox console actions"
      >
        {tool(
          paused ? "Resume output" : "Pause output",
          paused ? <Play size={14} /> : <Pause size={14} />,
          onPause,
          paused,
        )}
        {tool(
          "Follow latest output",
          <ArrowDown size={14} />,
          () => follow(!tab.follow),
          tab.follow,
        )}
        {tool(
          "Soft wrap",
          <WrapText size={14} />,
          () => onChange({ wrap: !tab.wrap }),
          tab.wrap,
        )}
        {tool(
          "Fold repeated messages",
          <ListCollapse size={14} />,
          () => onChange({ fold: !tab.fold }),
          tab.fold,
        )}
        <span className="console-tool-divider" />
        {tool(
          "Copy logs",
          <Copy size={14} />,
          () => void copy(visible.map(formatConsoleLog).join("\n")),
        )}
        {tool(
          "Copy selected entry",
          <FileText size={14} />,
          () => {
            if (chosen) void copy(formatConsoleLog(chosen));
          },
          undefined,
          !chosen,
        )}
        {tool("Export current console", <Download size={14} />, () => {
          void onExport(tab.name, visible.map(formatConsoleLog).join("\n"))
            .then((path) => {
              if (path) setNotice(`Saved: ${path}`);
            })
            .catch((e) => notifyFailure(e,"Console action failed"));
        })}
        {tool("Clear current console", <Trash2 size={14} />, () => {
          onChange({ clearedThrough: liveLogs.at(-1)?.id ?? 0 });
          setSelected(undefined);
        })}
        <span className="console-tool-divider" />
        {tool(
          "Show timestamps",
          <Clock size={14} />,
          () => onChange({ timestamps: !tab.timestamps }),
          tab.timestamps,
        )}
        {tool(
          "Show line numbers",
          <ListOrdered size={14} />,
          () => onChange({ lineNumbers: !tab.lineNumbers }),
          tab.lineNumbers,
        )}
        {tool(
          "Show bookmarked entries",
          <Bookmark size={14} />,
          () => onlyBookmarks((v) => !v),
          bookmarksOnly,
        )}
        {tool(
          "Entry details",
          <FileText size={14} />,
          () => showDetails((v) => !v),
          details,
        )}
        {tool(
          "More console filters",
          <Filter size={14} />,
          () => showFilters((v) => !v),
          filters,
        )}
        <span className="console-toolbar-spacer" />
        <select
          aria-label="Current console purpose"
          value={tab.purpose}
          onChange={(e) =>
            onChange({ purpose: e.target.value as ConsolePurpose })
          }
        >
          {Object.entries(consolePurposes).map(([id, label]) => (
            <option key={id} value={id}>
              {label}
            </option>
          ))}
        </select>
        <button
          onClick={() =>
            onNavigate(
              tab.purpose === "assets"
                ? "packs"
                : tab.purpose === "render"
                  ? "render"
                  : "worlds",
            )
          }
        >
          Open{" "}
          {tab.purpose === "assets"
            ? "resources"
            : tab.purpose === "render"
              ? "renderer"
              : "session"}
        </button>
      </div>
      <div className="console-searchbar">
        <Search size={14} />
        <StudioInput
          ref={search}
          type="search"
          aria-label="Search logs"
          placeholder="Find / filter logs (Ctrl+F)"
          value={tab.query}
          maxLength={256}
          onChange={(e) => onChange({ query: e.target.value })}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              jump(e.shiftKey ? -1 : 1);
            }
            if (e.key === "Escape") onChange({ query: "" });
          }}
        />
        {tool(
          "Match case",
          <CaseSensitive size={15} />,
          () => onChange({ matchCase: !tab.matchCase }),
          tab.matchCase,
        )}
        {tool(
          "Previous matching entry (Shift+F3)",
          <ArrowUp size={14} />,
          () => jump(-1),
          undefined,
          !rows.length,
        )}
        {tool(
          "Next matching entry (F3)",
          <ArrowDown size={14} />,
          () => jump(1),
          undefined,
          !rows.length,
        )}
        <select
          aria-label="Log level"
          value={tab.level}
          onChange={(e) => onChange({ level: e.target.value })}
        >
          <option value="all">All levels</option>
          {["error", "warning", "info", "debug", "trace"].map((v) => (
            <option key={v} value={v}>
              {v}
            </option>
          ))}
        </select>
        <span className="console-match-count">
          {visible.length} / {logs.length}
        </span>
      </div>
      {filters && (
        <div className="console-filterbar">
          <label>
            Source
            <select
              aria-label="Log source"
              value={tab.source}
              onChange={(e) => onChange({ source: e.target.value })}
            >
              <option value="all">All sources</option>
              {sources.map((v) => (
                <option key={v}>{v}</option>
              ))}
            </select>
          </label>
          <label>
            Exclude
            <StudioInput
              aria-label="Exclude log text"
              value={tab.exclude}
              maxLength={256}
              onChange={(e) => onChange({ exclude: e.target.value })}
            />
          </label>
          <label>
            Font
            <select
              aria-label="Console font size"
              value={tab.fontSize}
              onChange={(e) => onChange({ fontSize: +e.target.value })}
            >
              {[10, 11, 12, 13, 14, 16, 18, 20].map((n) => (
                <option key={n} value={n}>
                  {n}px
                </option>
              ))}
            </select>
          </label>
          <button
            onClick={() => {
              void copy(
                diagnostics ||
                  `CraftBox · ${sessionName}\n${logs.length} buffered logs`,
              );
            }}
          >
            Copy diagnostics
          </button>
          <button
            onClick={() =>
              onChange({
                query: "",
                exclude: "",
                source: "all",
                level: "all",
                matchCase: false,
              })
            }
          >
            Reset filters
          </button>
        </div>
      )}
      <div className="console-output-layout">
        <div
          ref={scroll}
          className={`console-log-output ${tab.wrap ? "soft-wrap" : "no-wrap"}`}
          tabIndex={0}
          aria-label="Console output"
          style={{ fontSize: tab.fontSize }}
          onScroll={(e) => {
            if (!active) return;
            if (!expectedScroll.current)
              lastScroll.current = e.currentTarget.scrollTop;
            if (expectedScroll.current || !tab.follow) return;
            const el = e.currentTarget;
            if (el.scrollHeight - el.clientHeight - el.scrollTop > 40)
              onChange({ follow: false });
          }}
        >
          {rows.length ? (
            rows.map(({ log, count }, index) => (
              <div
                className={`console-log-row level-${normalizeLogLevel(log.level)} ${selected === log.id ? "selected" : ""}`}
                key={log.id}
                ref={(node) => {
                  if (node) rowsRef.current.set(log.id, node);
                  else rowsRef.current.delete(log.id);
                }}
                onClick={() => setSelected(log.id)}
                onDoubleClick={() => {
                  setSelected(log.id);
                  showDetails(true);
                }}
                onContextMenu={(e) => {
                  e.preventDefault();
                  setSelected(log.id);
                  showDetails(true);
                }}
              >
                <span className="console-log-gutter">
                <button
                  className="console-bookmark"
                  data-tooltip={
                    tab.bookmarks.includes(log.id)
                      ? "Remove bookmark"
                      : "Bookmark entry"
                  }
                  aria-label={`Bookmark log ${log.id}`}
                  aria-pressed={tab.bookmarks.includes(log.id)}
                  onClick={(e) => {
                    e.stopPropagation();
                    bookmark(log.id);
                  }}
                >
                  <Bookmark size={11} fill={tab.bookmarks.includes(log.id) ? "currentColor" : "none"} />
                </button>
                {tab.lineNumbers && <span className="console-bookmark-number">{index + 1}</span>}
                </span>
                {tab.timestamps && (
                  <time data-tooltip={log.time}>
                    {Number.isNaN(Date.parse(log.time))
                      ? log.time
                      : new Date(log.time).toLocaleTimeString("en-GB", {
                          hour12: false,
                        })}
                  </time>
                )}
                <span className="console-log-level">{log.level}</span>
                <span className="console-log-message">
                  {log.source && (
                    <b className="console-log-source">[{mark(log.source)}] </b>
                  )}
                  {mark(log.message)}
                  {count > 1 && (
                    <span className="console-repeat-count">×{count}</span>
                  )}
                </span>
              </div>
            ))
          ) : (
            <div className="console-empty">
              <Terminal size={25} />
              <strong>
                {logs.length
                  ? "No entries match this console"
                  : "Waiting for CraftBox output"}
              </strong>
              <span>
                {logs.length
                  ? "Change the purpose or reset the filters to see more entries."
                  : "Start a session or use PowerShell for local commands."}
              </span>
              {logs.length > 0 && (
                <button
                  onClick={() => {
                    onChange({
                      query: "",
                      exclude: "",
                      level: "all",
                      source: "all",
                      clearedThrough: 0,
                    });
                    onlyBookmarks(false);
                  }}
                >
                  Show buffered entries
                </button>
              )}
            </div>
          )}
        </div>
        {details && (
          <aside className="console-details" aria-label="Log entry details">
            <header>
              <strong>Entry details</strong>
              {tool("Close entry details", <X size={13} />, () =>
                showDetails(false),
              )}
            </header>
            {chosen ? (
              <>
                <dl>
                  <dt>Time</dt>
                  <dd>{chosen.time}</dd>
                  <dt>Level</dt>
                  <dd>{chosen.level}</dd>
                  <dt>Source</dt>
                  <dd>{chosen.source || "CraftBox"}</dd>
                </dl>
                <pre>{chosen.message}</pre>
                <button
                  onClick={() => void copy(JSON.stringify(chosen, null, 2))}
                >
                  Copy as JSON
                </button>
                <button
                  onClick={() => onChange({ source: chosen.source || "all" })}
                >
                  Filter this source
                </button>
                <button onClick={() => bookmark(chosen.id)}>
                  Toggle bookmark
                </button>
              </>
            ) : (
              <p>Double-click or right-click an entry.</p>
            )}
          </aside>
        )}
      </div>
      {help && (
        <div className="console-command-help">
          <strong>CraftBox commands</strong>
          <span><button type="button" onClick={()=>setCLI({initial:"help"})}>/cli</button> opens the full command-line workspace.</span>
          <span>
            /find text · /exclude text · /source name · /level error · /purpose
            problems · /pause · /resume · /follow on|off · /wrap on|off · /clear
            · /copy · /export · /bookmark · /open render · /stop
          </span>
          <span>↑/↓: history. Shell commands: PowerShell.</span>
        </div>
      )}
      <ConsoleCommandInput mode="console" history={tab.history} onHistory={history=>onChange({history})} onRun={execute} onHelp={()=>setHelp(v=>!v)} onClear={()=>onChange({clearedThrough:liveLogs.at(-1)?.id??0})} context={{sources}}/>
      <footer className="console-status">
        <span>
          {paused
            ? `Paused${pending ? ` · ${pending} new entries` : ""}`
            : `${sessionName} · ${tab.follow ? "Following output" : "Scroll paused"}`}{" "}
          · {visible.length} entries
        </span>
        <div>
          {!tab.follow && (
            <button onClick={() => onChange({ follow: true })}>
              Jump to latest
            </button>
          )}
          <span data-tooltip="The application retains the most recent 2,000 log entries">
            Buffer: 2,000
          </span>
        </div>
      </footer>
    </div>
  );
}

import {notifyAction,notifySaved} from "./StudioFlyouts";
import StudioInput from "@/StudioInput";
import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import {
  Copy,
  Download,
  Search,
  ArrowDown,
  ArrowUp,
  RotateCcw,
  Square,
  X,
  ClipboardPaste,
  Minus,
  Plus,
} from "lucide-react";
import { api, type TerminalInfo } from "./bridge";
import { terminalInputChunks } from "./terminal-input";
import "@xterm/xterm/css/xterm.css";

export default function ShellTerminal({
  active,
  name,
  cwd,
  onExport,
}: {
  active: boolean;
  name: string;
  cwd?: string;
  onExport: (name: string, text: string) => Promise<unknown>;
}) {
  const host = useRef<HTMLDivElement>(null),
    instance = useRef<Terminal | undefined>(undefined),
    fit = useRef<FitAddon | undefined>(undefined),
    searchAddon = useRef<SearchAddon | undefined>(undefined),
    id = useRef<string | undefined>(undefined);
  const [generation, restart] = useState(0),
    [info, setInfo] = useState<TerminalInfo>(),
    [status, setStatus] = useState("Starting PowerShell …"),
    [error, setError] = useState(""),
    [exited, setExited] = useState(false),
    [query, setQuery] = useState(""),
    [searchOpen, setSearchOpen] = useState(false),
    [font, setFont] = useState(12);
  const activeRef = useRef(active);
  activeRef.current = active;
  const resize = useRef<() => void>(() => {});
  const text = () => {
    const t = instance.current;
    if (!t) return "";
    const buffer = t.buffer.active;
    return Array.from(
      { length: buffer.length },
      (_, i) => buffer.getLine(i)?.translateToString(true) ?? "",
    ).join("\n");
  };
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(
        instance.current?.getSelection() || text(),
      );
      notifyAction("Copied terminal output","PowerShell");
    } catch (e) {
      setError(String(e));
    }
  };
  useEffect(() => {
    let disposed = false,
      timer: ReturnType<typeof setTimeout> | undefined,
      resizeTimer: ReturnType<typeof setTimeout> | undefined,
      session: string | undefined;
    const terminal = new Terminal({
      fontFamily: 'Consolas, "Cascadia Mono", monospace',
      fontSize: font,
      lineHeight: 1.3,
      minimumContrastRatio: 4.5,
      scrollback: 5000,
      cursorBlink: true,
      allowProposedApi: false,
      convertEol: false,
      disableStdin: true,
      theme: {
        background:
          getComputedStyle(host.current!).getPropertyValue("--well").trim() ||
          "#1e1f22",
        foreground: getComputedStyle(host.current!).color,
        cursor: getComputedStyle(host.current!).color,
        selectionBackground:
          (getComputedStyle(host.current!)
            .getPropertyValue("--adobe-blue")
            .trim() || "#1473e6") + "66",
      },
    });
    const fitter = new FitAddon(),
      finder = new SearchAddon();
    terminal.loadAddon(fitter);
    terminal.loadAddon(finder);
    terminal.open(host.current!);
    const themeObserver = new MutationObserver(() => {
      if (!host.current || disposed) return;
      const style = getComputedStyle(host.current);
      terminal.options.theme = {
        background: style.getPropertyValue("--well").trim() || "#1e1f22",
        foreground: style.color,
        cursor: style.color,
        selectionBackground:
          (style.getPropertyValue("--adobe-blue").trim() || "#1473e6") + "66",
      };
    });
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "data-theme", "style"],
    });
    instance.current = terminal;
    fit.current = fitter;
    searchAddon.current = finder;
    let writing = Promise.resolve();
    const input = terminal.onData((data) => {
      writing = writing
        .then(async () => {
          if (session && !disposed)
            for (const chunk of terminalInputChunks(data)) {
              if (disposed) break;
              await api.WriteTerminal(session, chunk);
            }
        })
        .catch((e) => {
          if (!disposed) setError(String(e));
        });
    });
    const resizeNow = () => {
      if (
        disposed ||
        !activeRef.current ||
        !host.current?.clientWidth ||
        !host.current.clientHeight
      )
        return;
      fitter.fit();
      const cols = Math.min(500, Math.max(2, terminal.cols)),
        rows = Math.min(300, Math.max(2, terminal.rows));
      if (session)
        void api.ResizeTerminal(session, cols, rows).catch((e) => {
          if (!disposed) setError(String(e));
        });
    };
    resize.current = () => {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(resizeNow, 60);
    };
    const observer = new ResizeObserver(resize.current);
    observer.observe(host.current!);
    terminal.attachCustomKeyEventHandler((e) => {
      if (
        e.type === "keydown" &&
        e.ctrlKey &&
        e.shiftKey &&
        e.code === "KeyC"
      ) {
        void copy();
        return false;
      }
      if (e.type === "keydown" && e.ctrlKey && e.code === "KeyF") {
        setSearchOpen(true);
        return false;
      }
      return true;
    });
    async function poll() {
      if (disposed || !session) return;
      try {
        const out = await api.ReadTerminal(session);
        if (disposed) return;
        if (out.data) {
          const bytes = Uint8Array.from(atob(out.data), (c) => c.charCodeAt(0));
          await new Promise<void>((resolve) => terminal.write(bytes, resolve));
        }
        if (disposed) return;
        if (out.exited) {
          terminal.options.disableStdin = true;
          setExited(true);
          setStatus(`Process exited · code ${out.exitCode}`);
          return;
        }
        timer = setTimeout(() => void poll(), out.data ? 0 : 80);
      } catch (e) {
        if (!disposed) {
          terminal.options.disableStdin = true;
          setError(String(e));
          setExited(true);
        }
      }
    }
    setExited(false);
    setError("");
    setInfo(undefined);
    setStatus("Starting PowerShell …");
    const startTimer = setTimeout(() => {
      void api
        .CreateTerminal({ cols: 80, rows: 24, cwd: cwd || "" })
        .then((created) => {
          if (disposed) {
            void api.CloseTerminal(created.id).catch(() => {});
            return;
          }
          session = created.id;
          id.current = session;
          setInfo(created);
          setStatus("PowerShell ready");
          terminal.options.disableStdin = false;
          resizeNow();
          if (activeRef.current) terminal.focus();
          void poll();
        })
        .catch((e) => {
          if (!disposed) {
            setError(String(e));
            setExited(true);
          }
        });
    }, 0);
    return () => {
      disposed = true;
      clearTimeout(startTimer);
      clearTimeout(timer);
      clearTimeout(resizeTimer);
      observer.disconnect();
      themeObserver.disconnect();
      input.dispose();
      terminal.dispose();
      instance.current = undefined;
      id.current = undefined;
      if (session) void api.CloseTerminal(session).catch(() => {});
    };
    // Shell sessions persist across tab switches. Restart is an explicit action.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [generation, cwd]);
  useEffect(() => {
    if (instance.current) {
      instance.current.options.fontSize = font;
      resize.current();
    }
  }, [font]);
  useEffect(() => {
    if (active) {
      resize.current();
      instance.current?.focus();
    }
  }, [active]);
  const icon = (
    label: string,
    child: React.ReactNode,
    run: () => void,
    disabled = false,
  ) => (
    <button
      type="button"
      data-tooltip={label}
      aria-label={label}
      onClick={run}
      disabled={disabled}
    >
      {child}
    </button>
  );
  return (
    <div className="shell-terminal">
      <div
        className="console-toolbar"
        role="toolbar"
        aria-label="PowerShell actions"
      >
        <span className="console-kind">
          <span className={`console-dot ${!exited ? "running" : ""}`} />
          PowerShell
        </span>
        {icon(
          "Interrupt PowerShell (Ctrl+C)",
          <Square size={14} />,
          () => {
            if (id.current)
              void api
                .WriteTerminal(id.current, "\x03")
                .catch((e) => setError(String(e)));
          },
          exited,
        )}
        {icon("Restart PowerShell", <RotateCcw size={14} />, () =>
          restart((v) => v + 1),
        )}
        <span className="console-tool-divider" />
        {icon("Find in terminal (Ctrl+F)", <Search size={14} />, () =>
          setSearchOpen((v) => !v),
        )}
        {icon(
          "Copy terminal selection or output",
          <Copy size={14} />,
          () => void copy(),
        )}
        {icon(
          "Paste into PowerShell",
          <ClipboardPaste size={14} />,
          () => {
            void navigator.clipboard
              .readText()
              .then((value) => instance.current?.paste(value))
              .catch((e) => setError(String(e)));
          },
          exited,
        )}
        {icon("Export terminal output", <Download size={14} />, () => {
          void onExport(name, text())
            .then((path) => {
              if (typeof path==="string" && path) notifySaved(path);
            })
            .catch((e) => setError(String(e)));
        })}
        <span className="console-tool-divider" />
        {icon(
          "Decrease terminal font",
          <Minus size={14} />,
          () => setFont((v) => Math.max(10, v - 1)),
          font <= 10,
        )}
        {icon(
          "Increase terminal font",
          <Plus size={14} />,
          () => setFont((v) => Math.min(20, v + 1)),
          font >= 20,
        )}
        <span className="console-toolbar-caption" data-tooltip={info?.cwd}>
          {info?.cwd || "Local shell"}
        </span>
      </div>
      {searchOpen && (
        <div className="console-searchbar">
          <Search size={14} />
          <StudioInput
            autoFocus
            aria-label="Find in PowerShell"
            placeholder="Find in terminal"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              searchAddon.current?.findNext(e.target.value);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                if (e.shiftKey) searchAddon.current?.findPrevious(query);
                else searchAddon.current?.findNext(query);
              }
              if (e.key === "Escape") setSearchOpen(false);
            }}
          />
          {icon("Previous terminal match", <ArrowUp size={14} />, () => {
            searchAddon.current?.findPrevious(query);
          })}
          {icon("Next terminal match", <ArrowDown size={14} />, () => {
            searchAddon.current?.findNext(query);
          })}
          {icon("Close terminal search", <X size={14} />, () => {
            setSearchOpen(false);
            instance.current?.focus();
          })}
        </div>
      )}
      <div
        className="shell-terminal-screen"
        ref={host}
        aria-label={`${name} terminal`}
      />
      <div className="console-status">
        <span>{error || status}</span>
        <span>{info ? `PID ${info.pid} · ${font}px` : font + "px"}</span>
      </div>
    </div>
  );
}

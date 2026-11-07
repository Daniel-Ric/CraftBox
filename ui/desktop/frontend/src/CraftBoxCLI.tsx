import { useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  ArrowDown,
  ArrowLeft,
  Copy,
  Download,
  Terminal,
  Trash2,
} from "lucide-react";
import { api, type Log } from "./bridge";
import ConsoleCommandInput from "./ConsoleCommandInput";
import { runCLI, type CompletionContext } from "./craftbox-cli";
import "./craftbox-cli.css";

type Entry = {
  id: number;
  command: string;
  text: string;
  error?: boolean;
  pending?: boolean;
  elapsed?: number;
};
export default function CraftBoxCLI({
  logs,
  history,
  onHistory,
  onExit,
  onNavigate,
  onExport,
  onStop,
  initialCommand = "",
  fontSize = 12,
}: {
  logs: Log[];
  history: string[];
  onHistory: (lines: string[]) => void;
  onExit: () => void;
  onNavigate: (page: string) => void;
  onExport: (name: string, text: string) => Promise<unknown>;
  onStop?: () => Promise<void>;
  initialCommand?: string;
  fontSize?: number;
}) {
  const [entries, setEntries] = useState<Entry[]>([
      {
        id: 0,
        command: "",
        text: "CraftBox CLI\nManage captures, worlds, resources and session logs.\nType help for the reference, or use Tab to explore commands.",
      },
    ]),
    [busy, setBusy] = useState(false),
    [context, setContext] = useState<CompletionContext>({}),
    [following, setFollowing] = useState(true),
    [notice, setNotice] = useState("");
  const scroll = useRef<HTMLDivElement>(null),
    sequence = useRef(0),
    pending = useRef(false),
    mounted = useRef(true),
    started = useRef(false),
    latest = useRef({ logs, history, onStop });
  latest.current = { logs, history, onStop };
  useEffect(() => {
    mounted.current = true;
    void api
      .GetBootstrap()
      .then((b) => {
        if (mounted.current)
          setContext({ worlds: b.state.worlds, commands: b.commands });
      })
      .catch(() => {});
    return () => {
      mounted.current = false;
    };
  }, []);
  useLayoutEffect(() => {
    if (following && scroll.current)
      scroll.current.scrollTop = scroll.current.scrollHeight;
  }, [entries, following, notice]);
  useEffect(() => {
    if (!following || !scroll.current || typeof ResizeObserver === "undefined")
      return;
    const observer = new ResizeObserver(() => {
      if (scroll.current)
        scroll.current.scrollTop = scroll.current.scrollHeight;
    });
    observer.observe(scroll.current);
    return () => observer.disconnect();
  }, [following]);
  async function execute(command: string) {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    setNotice("");
    const id = ++sequence.current,
      start = performance.now();
    setEntries((old) => [
      ...old.slice(-199),
      { id, command, text: "", pending: true },
    ]);
    try {
      const result = await runCLI(command, {
        backend: api,
        logs: latest.current.logs,
        history: latest.current.history,
        navigate: onNavigate,
        exportText: onExport,
        stop: latest.current.onStop,
      });
      if (!mounted.current) return;
      if (result.exit) {
        onExit();
        return;
      }
      if (result.clear) {
        setEntries([]);
        return;
      }
      if (result.worlds || result.commands)
        setContext((old) => ({
          ...old,
          ...(result.worlds ? { worlds: result.worlds } : {}),
          ...(result.commands ? { commands: result.commands } : {}),
        }));
      const text =
        result.text.length > 65536
          ? result.text.slice(0, 65536) +
            "\n[Output limited to 64 KiB. Narrow the query or export the logs.]"
          : result.text;
      setEntries((old) =>
        old.map((e) =>
          e.id === id
            ? { ...e, text, pending: false, elapsed: performance.now() - start }
            : e,
        ),
      );
    } catch (error) {
      if (mounted.current)
        setEntries((old) =>
          old.map((e) =>
            e.id === id
              ? {
                  ...e,
                  text: String(error instanceof Error ? error.message : error),
                  error: true,
                  pending: false,
                  elapsed: performance.now() - start,
                }
              : e,
          ),
        );
    } finally {
      pending.current = false;
      if (mounted.current) setBusy(false);
    }
  }
  useEffect(() => {
    if (!started.current && initialCommand.trim()) {
      started.current = true;
      void execute(initialCommand);
    }
  }, []);
  const transcript = () =>
    entries
      .map((e) => (e.command ? `craftbox › ${e.command}\n` : "") + e.text)
      .join("\n\n");
  const clearTranscript = () =>
    setEntries((old) => old.filter((e) => e.pending));
  async function copy() {
    try {
      await navigator.clipboard.writeText(transcript());
      setNotice("Transcript copied");
    } catch (error) {
      setNotice(String(error));
    }
  }
  async function save() {
    try {
      const path = await onExport("CraftBox CLI", transcript());
      if (path) setNotice(`Saved: ${path}`);
    } catch (error) {
      setNotice(String(error));
    }
  }
  return (
    <div
      className="craftbox-cli console-view console-pane"
      style={{ "--cli-font-size": `${fontSize}px` } as React.CSSProperties}
    >
      <header className="cli-toolbar">
        <div>
          <Terminal size={14} />
          <strong>CraftBox CLI</strong>
          <span>{busy ? "Running command" : "Ready"}</span>
        </div>
        <nav aria-label="CLI actions">
          <button
            onClick={() => void copy()}
            data-tooltip="Copy transcript"
            aria-label="Copy CLI transcript"
          >
            <Copy size={14} />
          </button>
          <button
            onClick={() => void save()}
            data-tooltip="Export transcript"
            aria-label="Export CLI transcript"
          >
            <Download size={14} />
          </button>
          <button
            onClick={clearTranscript}
            data-tooltip="Clear transcript"
            aria-label="Clear CLI transcript"
          >
            <Trash2 size={14} />
          </button>
          <button onClick={onExit} disabled={busy}>
            <ArrowLeft size={13} />
            Application logs
          </button>
        </nav>
      </header>
      <div
        className="cli-transcript"
        ref={scroll}
        role="log"
        aria-busy={busy}
        aria-label="CraftBox CLI output"
        aria-live="polite"
        aria-relevant="additions text"
        onScroll={(e) => {
          const el = e.currentTarget;
          setFollowing(el.scrollHeight - el.scrollTop - el.clientHeight < 36);
        }}
      >
        {entries.map((e) => (
          <section
            className="cli-entry"
            key={e.id}
            data-error={e.error || undefined}
          >
            {e.command && (
              <div className="cli-entry-command">
                <span>›</span>
                <code>{e.command}</code>
                <small>
                  {e.pending
                    ? "Running…"
                    : e.elapsed !== undefined
                      ? `${Math.round(e.elapsed)} ms`
                      : ""}
                </small>
              </div>
            )}
            <pre>{e.pending ? "Working…" : e.text}</pre>
          </section>
        ))}
        {!entries.length && (
          <p className="cli-empty">
            Output cleared. Type help to see commands.
          </p>
        )}
      </div>
      {(!following || notice) && (
        <div className="cli-notice" role="status">
          <span>{notice}</span>
          {!following && (
            <button
              onClick={() => {
                setFollowing(true);
                scroll.current?.scrollTo({ top: scroll.current.scrollHeight });
              }}
            >
              <ArrowDown size={12} />
              Latest output
            </button>
          )}
        </div>
      )}
      <ConsoleCommandInput
        mode="cli"
        autoFocus
        history={history}
        onHistory={onHistory}
        onRun={(value) => void execute(value)}
        onClear={clearTranscript}
        onHelp={() => void execute("help")}
        context={context}
        busy={busy}
      />
    </div>
  );
}

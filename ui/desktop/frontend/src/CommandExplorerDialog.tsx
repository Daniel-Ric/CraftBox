import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Check,
  Copy,
  Filter,
  History,
  RefreshCw,
  Search,
  Star,
  Terminal,
  Trash2,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import { Button } from "./components/ui/button";
import { api } from "./bridge";
import {
  buildCommand,
  cleanCommandText,
  commandExplorer,
  commandSyntax,
  compareCommands,
  filterCommands,
  permissionNames,
  type CommandDefinition,
  type CommandDocument,
  type CommandFilter,
  type CommandSession,
} from "./command-explorer-model";
import "./command-explorer.css";

const date = (value: number) =>
  value ? new Date(value).toLocaleString() : "Live";
const favouritesKey = "craftbox-command-favourites";
function readFavourites(): Set<string> {
  try {
    const value = JSON.parse(localStorage.getItem(favouritesKey) ?? "[]");
    return new Set(
      Array.isArray(value)
        ? value.filter((v) => typeof v === "string").slice(0, 5000)
        : [],
    );
  } catch {
    return new Set();
  }
}
function CommandBuilder({
  command,
  onError,
}: {
  command: CommandDefinition;
  onError: (message: string) => void;
}) {
  const [variant, setVariant] = useState(0),
    [drafts, setDrafts] = useState<Record<number, string[]>>({}),
    [copied, setCopied] = useState("");
  const activeVariant = Math.min(
    variant,
    Math.max(0, command.overloads.length - 1),
  );
  const values = drafts[activeVariant] ?? [],
    parameters = command.overloads[activeVariant] ?? [],
    result = buildCommand(command, activeVariant, values);
  const copy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(value);
    } catch (error) {
      onError(String(error));
    }
  };
  const setValue = (index: number, value: string) =>
    setDrafts((old) => {
      const next = [...(old[activeVariant] ?? [])];
      next[index] = value;
      return { ...old, [activeVariant]: next };
    });
  return (
    <div className="ce-builder">
      <div className="ce-variant-heading">
        <label htmlFor="ce-variant">Syntax variant</label>
        <span>{command.overloads.length || 1} variants</span>
      </div>
      <select
        id="ce-variant"
        value={activeVariant}
        onChange={(e) => setVariant(Number(e.target.value))}
      >
        {(command.overloads.length ? command.overloads : [[]]).map(
          (_, index) => (
            <option value={index} key={index}>
              {commandSyntax(command, index)}
            </option>
          ),
        )}
      </select>
      <div className="ce-arguments">
        {parameters.map((parameter, index) => (
          <div className="ce-argument" key={variant + ":" + index}>
            <label htmlFor={`ce-argument-${index}`}>
              {parameter.name}
              <small>
                {parameter.kind} ·{" "}
                {parameter.optional ? "optional" : "required"}
                {parameter.dynamic ? " · live suggestions" : ""}
              </small>
            </label>
            {parameter.kind === "enum" &&
            !parameter.dynamic &&
            parameter.values.length <= 5000 ? (
              <select
                id={`ce-argument-${index}`}
                value={values[index] ?? ""}
                onChange={(e) => setValue(index, e.target.value)}
              >
                <option value="">Choose {parameter.name}</option>
                {parameter.values.slice(0, 5000).map((value) => (
                  <option key={value} value={value}>
                    {value}
                  </option>
                ))}
              </select>
            ) : (
              <input
                id={`ce-argument-${index}`}
                value={values[index] ?? ""}
                list={
                  parameter.values.length ? `ce-options-${index}` : undefined
                }
                placeholder={
                  parameter.kind === "position"
                    ? "~ ~ ~"
                    : parameter.kind === "target"
                      ? "@s"
                      : parameter.name
                }
                onChange={(e) => setValue(index, e.target.value)}
              />
            )}
            {parameter.values.length > 0 &&
              (parameter.dynamic || parameter.values.length > 5000) && (
                <datalist id={`ce-options-${index}`}>
                  {parameter.values
                    .filter((value) =>
                      value
                        .toLowerCase()
                        .includes((values[index] ?? "").toLowerCase()),
                    )
                    .slice(0, 1000)
                    .map((value) => (
                      <option key={value} value={value} />
                    ))}
                </datalist>
              )}
            {parameter.enum && (
              <small>
                {parameter.enum} · {parameter.values.length.toLocaleString()}{" "}
                advertised values
              </small>
            )}
          </div>
        ))}
      </div>
      {!parameters.length && (
        <p className="ce-muted">This variant takes no arguments.</p>
      )}
      <div className="ce-result">
        <code>{result.line}</code>
        <Button
          disabled={result.errors.length > 0}
          onClick={() => void copy(result.line)}
        >
          {copied === result.line ? <Check size={14} /> : <Copy size={14} />}
          {copied === result.line ? "Copied" : "Copy command"}
        </Button>
      </div>
      {result.errors.length > 0 && (
        <ul className="ce-validation" aria-label="Argument validation">
          {result.errors.map((error) => (
            <li key={error}>{error}</li>
          ))}
        </ul>
      )}
      <p className="ce-muted">
        Copy the command into Minecraft. The server checks your permissions when
        you run it.
      </p>
      <details className="ce-reference">
        <summary>All syntax variants</summary>
        {(command.overloads.length ? command.overloads : [[]]).map(
          (_, index) => (
            <div key={index}>
              <code>{commandSyntax(command, index)}</code>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={`Copy syntax variant ${index + 1}`}
                onClick={() => void copy(commandSyntax(command, index))}
              >
                <Copy size={13} />
              </Button>
            </div>
          ),
        )}
      </details>
    </div>
  );
}

export default function CommandExplorerDialog({
  open,
  onOpenChange,
  initialSession = "",
  connected = false,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialSession?: string;
  connected?: boolean;
}) {
  const [sessions, setSessions] = useState<CommandSession[]>([]),
    [session, setSession] = useState(""),
    [document, setDocument] = useState<CommandDocument>(),
    [query, setQuery] = useState(""),
    [filter, setFilter] = useState<CommandFilter>("all"),
    [selected, setSelected] = useState(""),
    [favourites, setFavourites] = useState(readFavourites),
    [tab, setTab] = useState("Build"),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(false),
    [busy, setBusy] = useState(false),
    [limit, setLimit] = useState(150),
    [baseline, setBaseline] = useState(""),
    [previous, setPrevious] = useState<CommandDocument>(),
    [capture, setCapture] = useState(""),
    [captures, setCaptures] = useState<{ id: string; name: string }[]>([]),
    [exportFormat, setExportFormat] = useState("json"),
    [deleteTarget, setDeleteTarget] = useState("");
  const search = useRef<HTMLInputElement>(null),
    request = useRef(0);
  const [refreshVersion, setRefreshVersion] = useState(0);
  const reportError = useCallback((message: string) => setError(message), []);
  const refresh = useCallback(async () => {
    const next = await commandExplorer.GetCommandExplorerSessions();
    setSessions(next);
    setSession((current) =>
      next.some((s) => s.id === current) ? current : (next[0]?.id ?? ""),
    );
  }, []);
  useEffect(() => {
    if (!open) return;
    let alive = true,
      pending = false;
    const update = async () => {
      if (pending) return;
      pending = true;
      try {
        const next = await commandExplorer.GetCommandExplorerSessions();
        if (alive) {
          setSessions(next);
          setSession((current) =>
            next.some((s) => s.id === current) ? current : (next[0]?.id ?? ""),
          );
        }
      } catch (e) {
        if (alive) setError(String(e));
      } finally {
        pending = false;
      }
    };
    void update();
    const timer = window.setInterval(() => void update(), 1500);
    void api
      .GetPacketInspector()
      .then((value) => {
        if (alive)
          setCaptures(value.captures.map((c) => ({ id: c.id, name: c.name })));
      })
      .catch(() => {});
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [open]);
  useEffect(() => {
    if (open && initialSession) setSession(initialSession);
  }, [open, initialSession]);
  const revision = sessions.find((s) => s.id === session)?.revision;
  useEffect(() => {
    if (!open || !session) {
      setDocument(undefined);
      return;
    }
    const token = ++request.current;
    setLoading(true);
    if (document?.session.id !== session) setDocument(undefined);
    void commandExplorer
      .GetCommandExplorer(session)
      .then((value) => {
        if (token === request.current) {
          setDocument(value);
          setError("");
        }
      })
      .catch((e) => {
        if (token === request.current) setError(String(e));
      })
      .finally(() => {
        if (token === request.current) setLoading(false);
      });
    return () => {
      request.current++;
    };
  }, [open, session, revision, refreshVersion]);
  useEffect(() => {
    setPrevious(undefined);
    if (!open || !baseline) return;
    let alive = true;
    void commandExplorer
      .GetCommandExplorer(baseline)
      .then((value) => {
        if (alive) setPrevious(value);
      })
      .catch((e) => {
        if (alive) setError(String(e));
      });
    return () => {
      alive = false;
    };
  }, [open, baseline]);
  useEffect(() => {
    setLimit(150);
  }, [query, filter, session]);
  const commands = useMemo(
    () => filterCommands(document?.commands ?? [], query, filter, favourites),
    [document, query, filter, favourites],
  );
  const command = commands.find((c) => c.name === selected) ?? commands[0];
  const changes = useMemo(
    () => compareCommands(previous?.commands ?? [], document?.commands ?? []),
    [previous, document],
  );
  const run = async (operation: () => Promise<void>) => {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await operation();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const toggleFavourite = (name: string) =>
    setFavourites((old) => {
      const next = new Set(old);
      next.has(name) ? next.delete(name) : next.add(name);
      try {
        localStorage.setItem(
          favouritesKey,
          JSON.stringify([...next].slice(0, 5000)),
        );
      } catch {}
      return next;
    });
  const selectRelative = (direction: number) => {
    if (!commands.length) return;
    const index = Math.max(
        0,
        commands.findIndex((c) => c.name === command?.name),
      ),
      target = (index + direction + commands.length) % commands.length,
      next = commands[target];
    setSelected(next.name);
    setLimit((n) => Math.max(n, target + 1));
    window.requestAnimationFrame(() =>
      window.document
        .querySelector(`[data-command-name="${CSS.escape(next.name)}"]`)
        ?.scrollIntoView?.({ block: "nearest" }),
    );
  };
  const current = sessions.find((s) => s.id === session);
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="ce-dialog"
        onKeyDown={(e) => {
          if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "f") {
            e.preventDefault();
            search.current?.focus();
          } else if (
            (e.key === "ArrowDown" || e.key === "ArrowUp") &&
            ((e.target as HTMLElement) === search.current ||
              !(e.target as HTMLElement).closest(
                "input,textarea,select,button",
              ))
          ) {
            e.preventDefault();
            selectRelative(e.key === "ArrowDown" ? 1 : -1);
          }
        }}
      >
        <DialogHeader>
          <DialogTitle>
            <Terminal size={17} />
            Command Explorer
          </DialogTitle>
          <DialogDescription>
            Browse advertised server commands, inspect arguments and build
            commands to copy.
          </DialogDescription>
        </DialogHeader>
        <div className="ce-toolbar">
          <label className="ce-session">
            Server session
            <select
              aria-label="Command server session"
              value={session}
              onChange={(e) => {
                setSession(e.target.value);
                setSelected("");
              }}
            >
              {!sessions.length && (
                <option value="">No recorded sessions</option>
              )}
              {sessions.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} · {s.commands} commands · {date(s.started)}
                </option>
              ))}
            </select>
          </label>
          <span className={`ce-connection ${connected ? "is-live" : ""}`}>
            {connected ? "Connected" : "Offline library"}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Refresh commands"
            disabled={busy || loading}
            onClick={() =>
              void run(async () => {
                await refresh();
                setRefreshVersion((n) => n + 1);
              })
            }
          >
            <RefreshCw size={15} />
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() =>
              void run(async () => {
                const imported = await commandExplorer.ImportCommandExplorer();
                if (imported.id) {
                  await refresh();
                  setSession(imported.id);
                }
              })
            }
          >
            <ArrowUpFromLine size={14} />
            Import
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Delete recorded command session"
            disabled={!session || !current?.ended || busy}
            onClick={() => setDeleteTarget(session)}
          >
            <Trash2 size={14} />
          </Button>
        </div>
        {error && (
          <div className="ce-error" role="alert">
            {error}
            <Button variant="ghost" size="sm" onClick={() => setError("")}>
              Dismiss
            </Button>
          </div>
        )}
        <div className="ce-workbench">
          <aside className="ce-library">
            <div className="ce-search">
              <Search size={14} />
              <input
                ref={search}
                aria-label="Search server commands"
                value={query}
                placeholder="Name, alias, argument or value…"
                onChange={(e) => setQuery(e.target.value)}
              />
            </div>
            <div className="ce-filter">
              <Filter size={13} />
              <select
                aria-label="Filter server commands"
                value={filter}
                onChange={(e) => setFilter(e.target.value as CommandFilter)}
              >
                <option value="all">All commands</option>
                <option value="favourites">Favourites</option>
                <option value="restricted">Requires permission</option>
                <option value="dynamic">Live suggestions</option>
              </select>
              <span>{commands.length}</span>
            </div>
            <div
              className="ce-command-list"
              aria-label="Server commands"
              aria-busy={loading}
            >
              {commands.slice(0, limit).map((c) => (
                <div
                  className={`ce-command-row ${c.name === command?.name ? "is-selected" : ""}`}
                  key={c.name}
                >
                  <button
                    data-command-name={c.name}
                    aria-pressed={c.name === command?.name}
                    onClick={() => setSelected(c.name)}
                  >
                    <strong>/{c.name}</strong>
                    <small>
                      {cleanCommandText(c.description) || "No description"}
                    </small>
                  </button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className={favourites.has(c.name) ? "is-favourite" : ""}
                    aria-label={`${favourites.has(c.name) ? "Unfavourite" : "Favourite"} /${c.name}`}
                    aria-pressed={favourites.has(c.name)}
                    onClick={() => toggleFavourite(c.name)}
                  >
                    <Star size={13} />
                  </Button>
                </div>
              ))}
              {commands.length > limit && (
                <Button
                  variant="ghost"
                  onClick={() => setLimit((n) => n + 150)}
                >
                  Show next {Math.min(150, commands.length - limit)}
                </Button>
              )}
              {!commands.length && (
                <div className="ce-empty">
                  <Terminal size={27} />
                  <strong>
                    {loading
                      ? "Loading commands…"
                      : document?.commands.length
                        ? "No matching commands"
                        : "No advertised commands yet"}
                  </strong>
                  <p>
                    {document?.commands.length
                      ? "Change your search or filter."
                      : "Connect through CraftBox or extract a packet recording."}
                  </p>
                  {query && (
                    <Button
                      variant="outline"
                      onClick={() => {
                        setQuery("");
                        setFilter("all");
                      }}
                    >
                      Clear filters
                    </Button>
                  )}
                </div>
              )}
            </div>
          </aside>
          <main className="ce-detail">
            {document ? (
              <>
                <header className="ce-command-heading">
                  <div>
                    <h2>
                      {command ? "/" + command.name : "Command catalogue"}
                    </h2>
                    <p>
                      {cleanCommandText(command?.description ?? "") ||
                        "No description was advertised."}
                    </p>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label="Toggle favourite command"
                    disabled={!command}
                    aria-pressed={
                      command ? favourites.has(command.name) : false
                    }
                    onClick={() => {
                      if (command) toggleFavourite(command.name);
                    }}
                  >
                    <Star
                      size={16}
                      fill={
                        command && favourites.has(command.name)
                          ? "currentColor"
                          : "none"
                      }
                    />
                  </Button>
                </header>
                <div className="ce-meta">
                  <span>
                    {command
                      ? (permissionNames[command.permission] ??
                        `Permission ${command.permission}`)
                      : "No matching command selected"}
                  </span>
                  {command?.aliases.map((alias) => (
                    <button key={alias} onClick={() => setQuery(alias)}>
                      /{alias}
                    </button>
                  ))}
                </div>
                <nav className="ce-tabs" aria-label="Command details">
                  {["Build", "Definition", "Activity", "Compare"].map(
                    (name) => (
                      <button
                        key={name}
                        aria-current={tab === name ? "page" : undefined}
                        onClick={() => setTab(name)}
                      >
                        {name}
                      </button>
                    ),
                  )}
                </nav>
                <div className="ce-detail-scroll">
                  {(tab === "Build" || tab === "Definition") && !command ? (
                    <p className="ce-muted">
                      Choose a command or inspect this session's activity and
                      comparison.
                    </p>
                  ) : tab === "Build" && command ? (
                    <CommandBuilder
                      key={session + ":" + command.name}
                      command={command}
                      onError={reportError}
                    />
                  ) : tab === "Definition" && command ? (
                    <>
                      <dl className="ce-properties">
                        <dt>Permission</dt>
                        <dd>
                          {command.permission} ·{" "}
                          {permissionNames[command.permission]}
                        </dd>
                        <dt>Flags</dt>
                        <dd>
                          {command.flags} · 0x{command.flags.toString(16)}
                        </dd>
                        <dt>Received</dt>
                        <dd>{date(command.time)}</dd>
                        <dt>Chained subcommands</dt>
                        <dd>{command.chained.join(", ") || "None"}</dd>
                      </dl>
                      <pre className="ce-json">
                        {JSON.stringify(command, null, 2)}
                      </pre>
                    </>
                  ) : tab === "Activity" ? (
                    <>
                      <p className="ce-muted">
                        Observed command requests, results and definition
                        updates.
                      </p>
                      {document?.events
                        .filter((event) =>
                          ["commands", "command", "output", "enum"].includes(
                            event.kind,
                          ),
                        )
                        .slice(-200)
                        .reverse()
                        .map((event) => (
                          <details className="ce-event" key={event.id}>
                            <summary>
                              <History size={13} />
                              <span>{cleanCommandText(event.summary)}</span>
                              <time>{date(event.time)}</time>
                            </summary>
                            <pre className="ce-json">
                              {JSON.stringify(event.data, null, 2)}
                            </pre>
                          </details>
                        ))}
                    </>
                  ) : (
                    <>
                      <label className="ce-field">
                        Compare definitions with
                        <select
                          aria-label="Compare command session"
                          value={baseline}
                          onChange={(e) => setBaseline(e.target.value)}
                        >
                          <option value="">Choose a previous session</option>
                          {sessions
                            .filter((s) => s.id !== session)
                            .map((s) => (
                              <option key={s.id} value={s.id}>
                                {s.name} · {date(s.started)}
                              </option>
                            ))}
                        </select>
                      </label>
                      {previous && (
                        <>
                          <p className="ce-muted">
                            {changes.length} changes in advertised commands
                          </p>
                          {changes.map((change) => (
                            <details className="ce-event" key={change.name}>
                              <summary>
                                <span>{change.change}</span>
                                <code>/{change.name}</code>
                              </summary>
                              <pre className="ce-json">
                                {JSON.stringify(
                                  {
                                    before: change.before,
                                    after: change.after,
                                  },
                                  null,
                                  2,
                                )}
                              </pre>
                            </details>
                          ))}
                          {!changes.length && <p>No definition changes.</p>}
                        </>
                      )}
                    </>
                  )}
                </div>
              </>
            ) : (
              <div className="ce-empty">
                <Terminal size={32} />
                <strong>Select a command</strong>
                <p>
                  Server command definitions appear here as they are received.
                </p>
              </div>
            )}
          </main>
        </div>
        <footer className="ce-footer">
          <div>
            <span>{current?.name ?? "Command library"}</span>
            <small>
              {current
                ? `Protocol ${current.protocol} · ${document?.commands.length ?? 0} commands`
                : "Ctrl+F to search · ↑ ↓ to browse"}
            </small>
          </div>
          <div className="ce-export">
            <select
              aria-label="Command export format"
              value={exportFormat}
              onChange={(e) => setExportFormat(e.target.value)}
            >
              <option value="json">JSON archive</option>
              <option value="markdown">Markdown</option>
              <option value="html">HTML reference</option>
            </select>
            <Button
              variant="outline"
              size="sm"
              disabled={!session || !commands.length || busy}
              onClick={() =>
                void run(async () => {
                  await commandExplorer.ExportCommandExplorer({
                    session,
                    format: exportFormat,
                    ids: commands.map((c) => c.name),
                  });
                })
              }
            >
              <ArrowDownToLine size={14} />
              Export{" "}
              {commands.length === document?.commands.length
                ? "commands"
                : "filtered"}
            </Button>
          </div>
        </footer>
        <details className="ce-recording">
          <summary>Extract from a packet recording</summary>
          <div>
            <select
              aria-label="Command source recording"
              value={capture}
              onChange={(e) => setCapture(e.target.value)}
            >
              <option value="">Choose a recording</option>
              {captures.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            <Button
              variant="outline"
              size="sm"
              disabled={!capture || busy}
              onClick={() =>
                void run(async () => {
                  const extracted =
                    await commandExplorer.ExtractCommandExplorer(capture);
                  await refresh();
                  setSession(extracted.id);
                })
              }
            >
              Extract commands
            </Button>
          </div>
        </details>
        {deleteTarget && (
          <div className="ce-delete-confirm" role="alert">
            <span>
              Delete recorded session:{" "}
              {sessions.find((s) => s.id === deleteTarget)?.name ??
                "Selected session"}
              ?
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => setDeleteTarget("")}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={busy}
              onClick={() =>
                void run(async () => {
                  await commandExplorer.DeleteCommandExplorerSessions([
                    deleteTarget,
                  ]);
                  setDeleteTarget("");
                  setSession((current) =>
                    current === deleteTarget ? "" : current,
                  );
                  await refresh();
                })
              }
            >
              Delete session
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

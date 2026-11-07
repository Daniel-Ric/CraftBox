import FeedbackFlyout from "./FeedbackFlyout";
﻿import { useEffect, useId, useRef, useState } from "react";
import {
  Activity,
  Bookmark,
  ChevronLeft,
  ChevronRight,
  LoaderCircle,
  Search,
  X,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import { api } from "./bridge";
import {
  packetBytes,
  type PacketPage,
  type PacketQuery,
  type PacketTypeStat,
} from "./packet-inspector-model";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  capture: string;
  types: PacketTypeStat[];
  initial: PacketQuery;
  onApply: (q: PacketQuery, id?: string) => void;
};
const examples = [
  "type:Move*",
  "dir:in bytes:>1024",
  "is:partial | is:blocked",
  "field:Position.1>=64",
  "has:FormData",
  "raw:0102 -stage:sent",
];
export default function PacketSearchDialog({
  open,
  onOpenChange,
  capture,
  types,
  initial,
  onApply,
}: Props) {
  const [draft, setDraft] = useState<PacketQuery>({ capture }),
    [query, setQuery] = useState<PacketQuery>({ capture }),
    [typeSearch, setTypeSearch] = useState(""),
    [page, setPage] = useState<PacketPage>(),
    [loading, setLoading] = useState(false),
    [active, setActive] = useState(0),
    [error, setError] = useState("");
  const list = useRef<HTMLDivElement>(null),
    id = useId(),
    request = useRef(0);
  // A slow archive scan is allowed to finish before another starts.
  const scanning = useRef(false),
    pending = useRef<{ q: PacketQuery; version: number } | undefined>(
      undefined,
    );
  useEffect(() => {
    if (open) {
      setDraft({ ...initial, capture, offset: 0 });
      setPage(undefined);
      setTypeSearch("");
      setError("");
    }
  }, [open, capture]);
  useEffect(() => {
    if (!open) return;
    request.current++;
    pending.current = undefined;
    setLoading(true);
    const timer = setTimeout(() => {
      setQuery({ ...draft, capture, limit: 200, reverse: true });
    }, 350);
    return () => clearTimeout(timer);
  }, [draft, open, capture]);
  useEffect(() => {
    if (!open || !capture || query.capture !== capture) return;
    const version = ++request.current;
    pending.current = { q: query, version };
    setLoading(true);
    const drain = async () => {
      if (scanning.current) return;
      scanning.current = true;
      try {
        while (pending.current) {
          const task = pending.current;
          pending.current = undefined;
          try {
            const value = await api.QueryPackets(task.q);
            if (task.version === request.current) {
              setPage(value);
              setActive(0);
              setError("");
              setLoading(false);
            }
          } catch (e) {
            if (task.version === request.current) {
              setPage(undefined);
              setError(String(e));
              setLoading(false);
            }
          }
        }
      } finally {
        scanning.current = false;
      }
    };
    void drain();
    return () => {
      request.current++;
      pending.current = undefined;
    };
  }, [query, open, capture]);
  useEffect(() => {
    list.current
      ?.querySelector(`[data-packet-index="${active}"]`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);
  const draftKey = JSON.stringify({
    ...draft,
    capture,
    limit: 200,
    reverse: true,
  });
  const resultsCurrent = draftKey === JSON.stringify(query) && !loading;
  const choose = (packet?: string) => {
    if (packet && !resultsCurrent) return;
    onApply({ ...draft, capture }, packet);
    onOpenChange(false);
  };
  const change = (v: Partial<PacketQuery>) => {
    setDraft((old) => ({ ...old, ...v, offset: 0 }));
    setActive(0);
  };
  const visibleTypes = types.filter(
    (t) => !t.other && t.name.toLowerCase().includes(typeSearch.toLowerCase()),
  );
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="pi-dialog pi-search-dialog">
        <DialogHeader>
          <DialogTitle>
            <Search size={17} /> Find packets
          </DialogTitle>
          <DialogDescription>
            Search the complete recording, inspect a result or apply filters to
            the packet list.
          </DialogDescription>
        </DialogHeader>
        <div className="pi-search-controls">
          <div className="pi-search-input">
            <Search size={16} />
            <input
              autoFocus
              aria-label="Packet search query"
              placeholder="Packet name, content, field:FormID=7, type:Move* ..."
              value={draft.search || ""}
              onChange={(e) => change({ search: e.target.value })}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  list.current?.focus();
                } else if (e.key === "Enter") {
                  e.preventDefault();
                  choose();
                }
              }}
            />
            <button
              aria-label="Clear packet search"
              onClick={() => change({ search: "" })}
            >
              <X size={14} />
            </button>
          </div>
          <div className="pi-search-filter-row">
            <label>
              Direction
              <select
                aria-label="Search direction"
                value={draft.direction || ""}
                onChange={(e) => change({ direction: e.target.value })}
              >
                <option value="">Both directions</option>
                <option value="clientbound">To client</option>
                <option value="serverbound">To server</option>
              </select>
            </label>
            <label>
              Observation
              <select
                aria-label="Search observation"
                value={draft.stage || ""}
                onChange={(e) => change({ stage: e.target.value })}
              >
                <option value="">All observations</option>
                {[
                  "received",
                  "sent",
                  "decision",
                  "injected",
                  "imported",
                  "gap",
                ].map((s) => (
                  <option key={s}>{s}</option>
                ))}
              </select>
            </label>
            <button
              aria-pressed={!!draft.bookmarked}
              onClick={() => change({ bookmarked: !draft.bookmarked })}
            >
              <Bookmark size={14} />
              Bookmarks
            </button>
            <button
              onClick={() =>
                setDraft({
                  capture,
                  search: "",
                  name: "",
                  direction: "",
                  stage: "",
                  bookmarked: false,
                  offset: 0,
                })
              }
            >
              Reset filters
            </button>
          </div>
        </div>
        <div className="pi-search-body">
          <aside>
            <input
              aria-label="Find packet type"
              placeholder="Find a packet type"
              value={typeSearch}
              onChange={(e) => setTypeSearch(e.target.value)}
            />
            <div className="pi-search-type-list">
              <button
                className={!draft.name ? "selected" : ""}
                onClick={() => change({ name: "" })}
              >
                All packet types
                <span>
                  {types.reduce((n, t) => n + t.count, 0).toLocaleString()}
                </span>
              </button>
              {visibleTypes.map((t) => (
                <button
                  key={t.name}
                  className={draft.name === t.name ? "selected" : ""}
                  onClick={() => change({ name: t.name })}
                >
                  <span>{t.name}</span>
                  <small>{t.count.toLocaleString()}</small>
                </button>
              ))}
            </div>
          </aside>
          <section className="pi-search-results">
            <header>
              <span>
                {loading
                  ? "Searching recording..."
                  : `${(page?.total || 0).toLocaleString()} matching observations`}
              </span>
              {draft.name && (
                <button onClick={() => change({ name: "" })}>
                  {draft.name}
                  <X size={12} />
                </button>
              )}
            </header>
            {error && (
              <FeedbackFlyout title="Packet search" tone="warning">
                {error}
              </FeedbackFlyout>
            )}
            <div
              ref={list}
              className="pi-search-result-list"
              role="listbox"
              aria-label="Matching packets"
              aria-busy={!resultsCurrent}
              tabIndex={0}
              aria-activedescendant={
                resultsCurrent && page?.rows[active]
                  ? `${id}-${active}`
                  : undefined
              }
              onKeyDown={(e) => {
                const count = page?.rows.length || 0;
                if (!count || !resultsCurrent) return;
                let next = active;
                if (e.key === "ArrowDown")
                  next = Math.min(count - 1, active + 1);
                else if (e.key === "ArrowUp") next = Math.max(0, active - 1);
                else if (e.key === "Home") next = 0;
                else if (e.key === "End") next = count - 1;
                else if (e.key === "PageDown")
                  next = Math.min(count - 1, active + 10);
                else if (e.key === "PageUp") next = Math.max(0, active - 10);
                else if (e.key === "Enter") {
                  e.preventDefault();
                  choose(page?.rows[active]?.id);
                  return;
                } else return;
                e.preventDefault();
                setActive(next);
              }}
            >
              {!resultsCurrent ? (
                <div className="pi-empty">
                  <Activity size={30} />
                  <LoaderCircle className="pi-spin" size={18} />
                  <p>Searching all recorded packets</p>
                </div>
              ) : page?.rows.length ? (
                page.rows.map((p, i) => (
                  <button
                    role="option"
                    aria-selected={active === i}
                    id={`${id}-${i}`}
                    data-packet-index={i}
                    key={p.id}
                    className={active === i ? "selected" : ""}
                    onFocus={() => setActive(i)}
                    onClick={() => choose(p.id)}
                  >
                    <div>
                      <strong>{p.name}</strong>
                      <span>
                        #{p.id} · {packetBytes(p.bytes)}
                      </span>
                    </div>
                    <small>
                      {new Date(p.time).toLocaleTimeString([], {
                        hour12: false,
                      })}{" "}
                      · {p.direction} · {p.stage}
                      {p.partial ? " · Partial decode" : ""}
                    </small>
                    {p.summary && <p>{p.summary}</p>}
                  </button>
                ))
              ) : (
                <div className="pi-empty">
                  <Search size={30} />
                  <h3>
                    {capture ? "No matching packets" : "Select a recording"}
                  </h3>
                  <p>
                    {capture
                      ? "Change the query or reset the filters."
                      : "Choose a saved recording or start a capture."}
                  </p>
                </div>
              )}
            </div>
            <footer>
              <button
                aria-label="Previous search results"
                disabled={loading || !draft.offset}
                onClick={() =>
                  setDraft((d) => ({
                    ...d,
                    offset: Math.max(0, (d.offset || 0) - 200),
                  }))
                }
              >
                <ChevronLeft size={14} />
              </button>
              <span>
                {page?.total ? (draft.offset || 0) + 1 : 0}–
                {Math.min((draft.offset || 0) + 200, page?.total || 0)} of{" "}
                {(page?.total || 0).toLocaleString()}
              </span>
              <button
                aria-label="Next search results"
                disabled={
                  loading || (draft.offset || 0) + 200 >= (page?.total || 0)
                }
                onClick={() =>
                  setDraft((d) => ({ ...d, offset: (d.offset || 0) + 200 }))
                }
              >
                <ChevronRight size={14} />
              </button>
            </footer>
          </section>
        </div>
        <details className="pi-query-help">
          <summary>Search syntax & examples</summary>
          <p>
            Space = AND · | = OR · - excludes · * matches a name pattern. Field
            comparisons preserve full numeric IDs. Content filters use the same
            privacy settings as the packet list.
          </p>
          <div>
            {examples.map((s) => (
              <button key={s} onClick={() => change({ search: s })}>
                <code>{s}</code>
              </button>
            ))}
          </div>
        </details>
        <DialogFooter>
          <button onClick={() => onOpenChange(false)}>Cancel</button>
          <button className="pi-primary" onClick={() => choose()}>
            Apply filters
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

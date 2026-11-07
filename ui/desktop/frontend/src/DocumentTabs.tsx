import FeedbackFlyout from "./FeedbackFlyout";
import { ContextMenu } from "@base-ui/react/context-menu";
import { notify, notifyFailure } from "./StudioFlyouts";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import {createPortal} from "react-dom";
import {
  ChevronDown,
  Pin,
  X,
  LoaderCircle,
  Files,
  History,
  Check,
  CircleAlert,
} from "lucide-react";
import {
  activateDocument,
  closeDocuments,
  documentsLocked,
  emptyDocuments,
  keepDocument,
  markDocument,
  openDocument,
  pinDocument,
  reorderDocument,
  type DocumentTab,
  type DocumentTabsState,
} from "./document-tabs-model";
import type { StudioSession } from "./studio-document";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./components/ui/dialog";
import { Button } from "./components/ui/button";
import "./document-tabs.css";
export type AreaKind = "render" | "cinematic" | "packs" | "skins" | "packets" | "structures";
export type CloseEntry = {
  id: string;
  area: AreaKind;
  name: string;
  session: StudioSession;
};
export type CloseRequest = (
  entries: CloseEntry[],
  done: () => void,
  quit?: boolean,
) => void;
export type AreaController = {
  open: (
    source: string,
    name: string,
    payload: unknown,
    permanent?: boolean,
    id?: string,
  ) => void;
  save: (saveAs?: boolean) => Promise<boolean>;
  locked: () => boolean;
  activeId: () => string;
  findSource: (source: string) => string | undefined;
};
export function useCloseDocuments(
  resolve?: (id: string) => CloseEntry | undefined,
) {
  const [pending, setPending] = useState<{
    entries: CloseEntry[];
    done: () => void;
    quit: boolean;
    visibleIds: string[];
  }>();
  const [selected, setSelected] = useState<string[]>([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const request: CloseRequest = (entries, done, quit = false) => {
    if (entries.some((e) => e.session.busy)) {
      setError("");
      setPending({
        entries,
        done,
        quit,
        visibleIds: entries
          .filter((e) => e.session.dirty || e.session.busy)
          .map((e) => e.id),
      });
      setSelected(entries.filter((e) => e.session.dirty).map((e) => e.id));
      return;
    }
    const dirty = entries.filter((e) => e.session.dirty);
    if (!dirty.length) {
      done();
      return;
    }
    setSelected(dirty.map((e) => e.id));
    setError("");
    setPending({
      entries: quit ? entries : dirty,
      done,
      quit,
      visibleIds: dirty.map((e) => e.id),
    });
  };
  const entries =
    pending?.entries.map(
      (e) =>
        resolve?.(e.id) ??
        (e.id.startsWith("browser:")
          ? {
              ...e,
              session: {
                ...e.session,
                dirty: false,
                busy: false,
                save: async () => true,
              },
            }
          : e),
    ) || [];
  const dialog = (
    <Dialog
      open={!!pending}
      onOpenChange={(open) => {
        if (!open && !busy) setPending(undefined);
      }}
    >
      <DialogContent resizable={false} className="document-close-dialog" showCloseButton={!busy}>
        <DialogHeader>
          <DialogTitle>
            {pending?.quit ? "Save projects before quitting?" : "Save changes?"}
          </DialogTitle>
        </DialogHeader>
        <div className="studio-dialog-body document-close-body">
          <DialogDescription>
            {pending?.quit
              ? "Choose which projects to save. Unselected changes will be discarded."
              : "Save your project before closing it."}
          </DialogDescription>
          {entries.some((e) => e.session.busy) && (
            <div className="document-close-status" role="status">
              <LoaderCircle size={16} className="animate-spin" />
              <span>
                A project is still finishing a task. You can continue once it
                finishes, or cancel to keep working.
              </span>
            </div>
          )}
          <div className="document-close-list">
            {entries
              .filter(
                (e) =>
                  e.session.dirty ||
                  e.session.busy ||
                  pending?.visibleIds.includes(e.id),
              )
              .map((e) => (
                <label key={e.id}>
                  <input
                    type="checkbox"
                    checked={e.session.dirty && selected.includes(e.id)}
                    disabled={busy || e.session.busy || !e.session.dirty}
                    onChange={(ev) =>
                      setSelected((ids) =>
                        ev.target.checked
                          ? [...ids, e.id]
                          : ids.filter((id) => id !== e.id),
                      )
                    }
                  />
                  <span>
                    <strong>{e.name || "Untitled"}</strong>
                    <small>
                      {e.area}
                      {e.session.busy
                        ? " · Task running"
                        : e.session.dirty
                          ? " · Unsaved changes"
                          : " · Saved"}
                    </small>
                  </span>
                </label>
              ))}
          </div>
          {error && (
            <FeedbackFlyout title="Project changes">
              <CircleAlert size={16} />
              <span>{error}</span>
            </FeedbackFlyout>
          )}
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            disabled={busy}
            onClick={() => setPending(undefined)}
          >
            Cancel
          </Button>
          <Button
            variant="outline"
            disabled={busy || entries.some((e) => e.session.busy)}
            onClick={() => {
              if (entries.some((e) => (resolve?.(e.id) ?? e).session.busy)) {
                setError(
                  "A project is still finishing a task. Wait for it to finish before closing.",
                );
                return;
              }
              const p = pending;
              setPending(undefined);
              p?.done();
            }}
          >
            Discard {pending?.quit ? "all" : ""}
          </Button>
          <Button
            disabled={busy || entries.some((e) => e.session.busy)}
            onClick={async () => {
              if (!pending) return;
              setBusy(true);
              setError("");
              try {
                if (entries.some((e) => (resolve?.(e.id) ?? e).session.busy))
                  throw Error(
                    "A project is still finishing a task. Wait for it to finish before saving.",
                  );
                const dirtyAtStart = new Set(
                  entries
                    .filter((e) => (resolve?.(e.id) ?? e).session.dirty)
                    .map((e) => e.id),
                );
                for (const entry of entries) {
                  const e = resolve?.(entry.id) ?? entry;
                  if (
                    e.session.dirty &&
                    selected.includes(e.id) &&
                    !(await e.session.save())
                  )
                    throw Error(
                      `“${e.name}” was not saved. Nothing has been closed.`,
                    );
                }
                const latest = entries.map((e) => resolve?.(e.id) ?? e);
                if (latest.some((e) => e.session.busy))
                  throw Error("Wait for running tasks to finish.");
                const changed = latest.find(
                  (e) =>
                    e.session.dirty &&
                    (selected.includes(e.id) || !dirtyAtStart.has(e.id)),
                );
                if (changed)
                  throw Error(
                    `“${changed.name || "Untitled"}” changed while saving. Review its changes before closing.`,
                  );
                const p = pending;
                setPending(undefined);
                p.done();
              } catch (e) {
                const message = e instanceof Error ? e.message : String(e);
                setError(message);
                notifyFailure(
                  message,
                  "Project could not be saved",
                  "document-close-error",
                );
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy
              ? "Saving…"
              : !entries.some((e) => e.session.dirty && selected.includes(e.id))
                ? pending?.quit
                  ? "Quit"
                  : "Close"
                : pending?.quit
                  ? "Save selected and quit"
                  : "Save and close"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
  return { request, dialog };
}
export function DocumentArea({
  kind,
  active,
  controller,
  children,
  requestClose,
  onSessions,
  onEmptyAction,
}: {
  kind: AreaKind;
  active: boolean;
  controller: (value: AreaController) => void;
  children: (
    tab: DocumentTab | undefined,
    payload: unknown,
    register: (value: StudioSession | undefined) => void,
    open: AreaController["open"],
    visible: boolean,
  ) => React.ReactNode;
  requestClose: CloseRequest;
  onSessions: (area: AreaKind, entries: CloseEntry[]) => void;
  onEmptyAction?: () => void;
}) {
  const [state, setState] = useState<DocumentTabsState>(emptyDocuments),
    [overview, setOverview] = useState(false);
  const [tabHost]=useState(()=>document.createElement("div"));
  const browserSession = useRef<StudioSession | undefined>(undefined);
  const [browserRevision, setBrowserRevision] = useState(0);
  const deferredOpen = useRef<Parameters<AreaController["open"]> | undefined>(
    undefined,
  );
  const disposals = useRef(new Map<string, () => void>());
  const latest = useRef(state);
  latest.current = state;
  const previousRetained = useRef(new Set<string>());
  const data = useRef(new Map<string, unknown>()),
    sessions = useRef(new Map<string, StudioSession>()),
    root = useRef<HTMLDivElement>(null),
    drag = useRef("");
  useLayoutEffect(()=>{
    const area=root.current;if(!area)return;
    tabHost.className="editor-document-tabs";
    const place=()=>{
      const panel=area.querySelector<HTMLElement>(':scope > .document-panel:not([hidden])');
      const target=panel?.querySelector<HTMLElement>('[data-editor-tab-host]');
      if(target){if(tabHost.parentElement!==target)target.appendChild(tabHost);}
      else if(tabHost.parentElement!==area)area.prepend(tabHost);
    };
    place();const observer=new MutationObserver(place);
    observer.observe(area,{childList:true,subtree:true,attributes:true,attributeFilter:['hidden']});
    return()=>{observer.disconnect();tabHost.remove();};
  },[tabHost]);
  const publish = () =>
    onSessions(kind, [
      ...latest.current.tabs.map((t) => ({
        id: t.id,
        area: kind,
        name: t.name,
        session: sessions.current.get(t.id) ?? {
          name: t.name,
          dirty: t.dirty,
          busy: t.busy,
          save: async () => false,
        },
      })),
      ...(!latest.current.tabs.length && browserSession.current?.busy
        ? [
            {
              id: "browser:" + kind,
              area: kind,
              name: kind + " task",
              session: browserSession.current,
            },
          ]
        : []),
    ]);
  const register = useRef(
    new Map<string, (s: StudioSession | undefined) => void>(),
  );
  function registration(id: string) {
    let f = register.current.get(id);
    if (!f) {
      f = (s) => {
        if (id === "browser:" + kind) {
          browserSession.current = s;
          setBrowserRevision((n) => n + 1);
          queueMicrotask(publish);
          return;
        }
        if (s) {
          sessions.current.set(id, s);
          if (s.dispose) disposals.current.set(id, s.dispose);
        } else sessions.current.delete(id);
        setState((old) =>
          markDocument(old, id, {
            dirty: s?.dirty ?? false,
            busy: s?.busy ?? false,
            ...(s?.name ? { name: s.name } : {}),
            path: s?.path,
            ...(s?.path ? { source: "file:" + s.path } : {}),
          }),
        );
        queueMicrotask(publish);
      };
      register.current.set(id, f);
    }
    return f;
  }
  const open: AreaController["open"] = (
    source,
    name,
    payload,
    permanent = false,
    id = crypto.randomUUID(),
  ) => {
    if (documentsLocked(latest.current) || browserSession.current?.busy) {
      if ((payload as { deferUntilIdle?: boolean })?.deferUntilIdle)
        deferredOpen.current = [source, name, payload, permanent, id];
      else
        notify({
          tone: "info",
          title: "Task running",
          description:
            "Wait for this workspace task to finish before opening another tab.",
        });
      return;
    }
    const structureID=kind==="structures"?(payload as {sourceId?:string})?.sourceId:undefined;
    const exists = latest.current.tabs.find((t) => t.source === source || (!!structureID&&(data.current.get(t.id) as {sourceId?:string})?.sourceId===structureID));
    if(structureID&&exists){setState(old=>activateDocument(old,exists.id));return;}
    if (!exists) data.current.set(id, payload);
    setState((old) => openDocument(old, { id, source, name }, permanent));
  };
  const close = (ids: string[]) => {
    if (documentsLocked(latest.current)) return;
    requestClose(
      ids.map((id) => {
        const t = latest.current.tabs.find((t) => t.id === id)!;
        return {
          id,
          area: kind,
          name: t.name,
          session: sessions.current.get(id) ?? {
            name: t.name,
            dirty: t.dirty,
            busy: t.busy,
            save: async () => false,
          },
        };
      }),
      () =>
        setState((old) =>
          closeDocuments(
            {
              ...old,
              tabs: old.tabs.map((t) =>
                ids.includes(t.id) ? { ...t, busy: false } : t,
              ),
            },
            ids,
          ),
        ),
    );
  };
  controller({
    open,
    save: async (saveAs = false) =>
      (await sessions.current.get(latest.current.active)?.save(saveAs)) ??
      false,
    locked: () =>
      documentsLocked(latest.current) || !!browserSession.current?.busy,
    activeId: () => latest.current.active,
    findSource: (source) =>
      latest.current.tabs.find((t) => t.source === source)?.id,
  });
  useEffect(() => {
    const retained = new Set([...state.tabs, ...state.closed].map((t) => t.id));
    for (const [id, dispose] of disposals.current) {
      if (!retained.has(id)) {
        dispose();
        disposals.current.delete(id);
        data.current.delete(id);
        register.current.delete(id);
      }
    }
    for (const id of previousRetained.current)
      if (!retained.has(id)) {
        data.current.delete(id);
        register.current.delete(id);
      }
    previousRetained.current = retained;
    if (
      !documentsLocked(state) &&
      !browserSession.current?.busy &&
      deferredOpen.current
    ) {
      const args = deferredOpen.current;
      deferredOpen.current = undefined;
      open(...args);
    }
    publish();
    const current = root.current?.querySelector<HTMLElement>(
      '.document-tab > [role="tab"][aria-selected="true"]',
    );
    current?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [state, browserRevision]);
  useEffect(() => {
    if (!active) return;
    const key = (e: KeyboardEvent) => {
      if (
        !(e.ctrlKey || e.metaKey) ||
        e.altKey ||
        document.querySelector('[role="dialog"][data-open]')
      )
        return;
      if (e.defaultPrevented) return;
      const s = latest.current;
      const k = e.key.toLowerCase();
      if (k === "w") {
        e.preventDefault();
        if (s.active) close([s.active]);
      } else if (k === "tab") {
        e.preventDefault();
        const i = s.tabs.findIndex((t) => t.id === s.active),
          next =
            s.tabs[(i + (e.shiftKey ? -1 : 1) + s.tabs.length) % s.tabs.length];
        if (next) setState((old) => activateDocument(old, next.id));
      } else if (k === "t" && e.shiftKey) {
        e.preventDefault();
        const tab = s.closed[0];
        if (tab)
          open(tab.source, tab.name, data.current.get(tab.id), true, tab.id);
      } else if (k === "s" && !["render", "cinematic"].includes(kind)) {
        e.preventDefault();
        void sessions.current
          .get(s.active)
          ?.save()
          .catch((err) => console.error(err));
      }
    };
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, [active, state]);
  useEffect(
    () => () => {
      for (const dispose of disposals.current.values()) dispose();
    },
    [],
  );
  const locked = documentsLocked(state) || !!browserSession.current?.busy;
  const actions = (id: string): [string, () => void][] => [
    [
      "Save project",
      () =>
        void sessions.current
          .get(id)
          ?.save()
          .catch((e) =>
            notify({
              tone: "error",
              title: "Project could not be saved",
              description: String(e),
            }),
          ),
    ],
    [
      "Save project as…",
      () =>
        void sessions.current
          .get(id)
          ?.save(true)
          .catch((e) =>
            notify({
              tone: "error",
              title: "Project could not be saved",
              description: String(e),
            }),
          ),
    ],
    ["Keep open", () => setState((old) => keepDocument(old, id))],
    [
      state.tabs.find((t) => t.id === id)?.pinned ? "Unpin tab" : "Pin tab",
      () => setState((old) => pinDocument(old, id)),
    ],
    ["Close tab", () => close([id])],
    [
      "Close other tabs",
      () => close(state.tabs.filter((t) => t.id !== id).map((t) => t.id)),
    ],
    [
      "Close tabs to the right",
      () =>
        close(
          state.tabs
            .slice(state.tabs.findIndex((t) => t.id === id) + 1)
            .map((t) => t.id),
        ),
    ],
    ["Close all tabs", () => close(state.tabs.map((t) => t.id))],
  ];
  return (
    <div
      className="document-area"
      hidden={!active}
      ref={root}
      data-document-area={kind}
    >
      {!!state.tabs.length && createPortal(<div className="document-tabbar">
        <span className="document-dock-label">
          <Files size={14} />
          Projects
        </span>
        <div
          className="document-tab-scroll"
          role="tablist"
          aria-label={`${kind} projects`}
        >
          {!state.tabs.length && (
            <span className="document-dock-placeholder">No open projects</span>
          )}
          {state.tabs.map((tab) => (
            <ContextMenu.Root key={tab.id}>
              <ContextMenu.Trigger
                tabIndex={-1}
                className={`document-tab${tab.preview ? " is-preview" : ""}${tab.pinned ? " is-pinned" : ""}`}
                data-active={state.active === tab.id}
                draggable={!locked}
                onDragStart={() => {
                  drag.current = tab.id;
                }}
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => {
                  e.preventDefault();
                  setState((old) => reorderDocument(old, drag.current, tab.id));
                }}
                onAuxClick={(e) => {
                  if (e.button === 1) {
                    e.preventDefault();
                    close([tab.id]);
                  }
                }}
              >
                <button
                  role="tab"
                  aria-selected={state.active === tab.id}
                  aria-controls={`document-panel-${tab.id}`}
                  tabIndex={state.active === tab.id ? 0 : -1}
                  title={`${tab.name}${tab.path ? "\n" + tab.path : ""}${tab.dirty ? " · Unsaved changes" : ""}${tab.preview ? " · Preview" : ""}`}
                  disabled={locked && state.active !== tab.id}
                  onClick={() =>
                    setState((old) => activateDocument(old, tab.id))
                  }
                  onDoubleClick={() =>
                    setState((old) => keepDocument(old, tab.id))
                  }
                  onKeyDown={(e) => {
                    if (
                      ["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)
                    ) {
                      e.preventDefault();
                      const i = state.tabs.findIndex((t) => t.id === tab.id);
                      const n =
                        e.key === "Home"
                          ? 0
                          : e.key === "End"
                            ? state.tabs.length - 1
                            : (i +
                                (e.key === "ArrowRight" ? 1 : -1) +
                                state.tabs.length) %
                              state.tabs.length;
                      setState((old) =>
                        activateDocument(old, state.tabs[n].id),
                      );
                      requestAnimationFrame(() =>
                        root.current
                          ?.querySelectorAll<HTMLElement>('.document-tab > [role="tab"]')
                          [n]?.focus(),
                      );
                    }
                  }}
                >
                  {tab.busy ? (
                    <LoaderCircle size={12} className="spin" />
                  ) : tab.pinned ? (
                    <Pin size={12} />
                  ) : null}
                  <span>{tab.name || "Untitled"}</span>
                  {tab.dirty && <i aria-label="Unsaved changes">●</i>}
                </button>
                <button
                  className="document-tab-close"
                  aria-label={`Close ${tab.name}`}
                  title="Close tab (Ctrl+W)"
                  disabled={locked}
                  onClick={() => close([tab.id])}
                >
                  <X size={12} />
                </button>
              </ContextMenu.Trigger>
              <ContextMenu.Portal>
                <ContextMenu.Positioner className="document-context-positioner">
                  <ContextMenu.Popup className="document-context-menu">
                    {actions(tab.id).map(([label, action]) => (
                      <ContextMenu.Item
                        key={label}
                        disabled={locked}
                        onClick={action}
                      >
                        {label}
                      </ContextMenu.Item>
                    ))}
                  </ContextMenu.Popup>
                </ContextMenu.Positioner>
              </ContextMenu.Portal>
            </ContextMenu.Root>
          ))}
        </div>
        {locked && (
          <span className="document-task" role="status">
            <LoaderCircle size={12} className="spin" />
            Task running
          </span>
        )}
        <button
          className="document-overview-trigger"
          aria-label="Show all open tabs"
          title="Open tabs"
          onClick={() => setOverview(true)}
        >
          <ChevronDown size={15} />
        </button>
      </div>,tabHost)}
      {state.tabs.map((tab) => (
        <div
          id={`document-panel-${tab.id}`}
          role="tabpanel"
          key={tab.id}
          className="document-panel"
          hidden={state.active !== tab.id}
        >
          {children(
            tab,
            data.current.get(tab.id),
            registration(tab.id),
            open,
            active && state.active === tab.id,
          )}
        </div>
      ))}
      {!state.tabs.length && (
        <div className="document-panel">
          {children(
            undefined,
            undefined,
            registration("browser:" + kind),
            open,
            active,
          )}
          {onEmptyAction && (
            <button onClick={onEmptyAction}>Open a project</button>
          )}
        </div>
      )}
      <Dialog
        open={overview}
        onOpenChange={(v) => {
          if (!v) {
            setOverview(false);
          }
        }}
      >
        <DialogContent className="document-overview">
          <DialogHeader>
            <DialogTitle>Open projects</DialogTitle>
          </DialogHeader>
          <div className="studio-dialog-body document-overview-body">
            <DialogDescription>
              {state.tabs.length
                ? `${state.tabs.length} open ${state.tabs.length === 1 ? "project" : "projects"} in this workspace.`
                : "Opened projects appear here. Select content in this workspace to begin."}
            </DialogDescription>
            <div className="document-overview-list">
              {state.tabs.length ? (
                state.tabs.map((t) => (
                  <Button
                    key={t.id}
                    variant="ghost"
                    className="document-overview-row"
                    data-active={t.id === state.active}
                    title={t.path || t.name}
                    disabled={locked && t.id !== state.active}
                    onClick={() => {
                      setState((old) => activateDocument(old, t.id));
                      setOverview(false);
                    }}
                  >
                    <Files size={18} className="document-overview-icon" />
                    <span className="document-overview-caption">
                      <strong>{t.name}</strong>
                      <small>
                        {t.path ||
                          (t.preview ? "Preview" : "Project kept open")}
                      </small>
                    </span>
                    {t.pinned && <Pin size={13} aria-label="Pinned" />}
                    {t.dirty && (
                      <span className="document-overview-status">Unsaved</span>
                    )}
                    {t.id === state.active && (
                      <Check size={15} aria-label="Active project" />
                    )}
                  </Button>
                ))
              ) : (
                <div className="document-overview-empty">
                  <Files size={30} />
                  <strong>No open projects</strong>
                  <p>Open a project or select an item from the library.</p>
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            {state.closed.length > 0 && (
              <Button
                disabled={locked}
                variant="outline"
                className="document-reopen-action"
                title={state.closed[0].name}
                onClick={() => {
                  const t = state.closed[0];
                  open(t.source, t.name, data.current.get(t.id), true, t.id);
                  setOverview(false);
                }}
              >
                <History size={14} />
                Reopen last closed tab
              </Button>
            )}
            <Button variant="outline" onClick={() => setOverview(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

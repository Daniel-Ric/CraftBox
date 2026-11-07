import { consumePacketNavigation } from "./packet-navigation";
import { newProjectDefaults, withProjectSource } from "./start-project";
import {
  forwardRef,
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { FolderOpen,FilePlus,Globe } from "lucide-react";
import { api, on, type State, type RenderWorld } from "./bridge";
import type { StudioProject } from "./start-project";
import type { StudioDocument, StudioSession } from "./studio-document";
import {
  DocumentScope,
  parseWorkspaceDocument,
  type WorkspaceDocument,
} from "./document-scope";
import {
  DocumentArea,
  useCloseDocuments,
  type AreaController,
  type AreaKind,
  type CloseEntry,
  type CloseRequest,
} from "./DocumentTabs";
import type { DocumentTab } from "./document-tabs-model";
import type { EditorStatus } from "./editor-commands";
import { Button } from "./components/ui/button";
const RenderEditor = lazy(() => import("./RenderEditor"));
const AssetStudio = lazy(() => import("./AssetStudio"));
const PacketInspector = lazy(() => import("./PacketInspector"));
const StructureStudio = lazy(() => import("./StructureStudio"));
const areas: AreaKind[] = ["render", "cinematic", "packs", "skins", "packets", "structures"];
type Loaded = { world: RenderWorld; document: StudioDocument; path: string };
type Payload = {
  project?: StudioProject;
  loaded?: Loaded;
  workspace?: WorkspaceDocument;
  sourceId?: string;
  path?: string;
};
export type DocumentWorkspacesHandle = {
  openStructure:(id:string,name:string)=>void;
  save: (saveAs?: boolean) => Promise<boolean>;
  quit: (done: () => void) => void;
  openWorkspace: (raw: string, id: string, path: string) => void;
  locked: (area: string) => boolean;
  openWorld: (kind: "archive" | "folder") => void;
};
type Props = {
  page: string;
  state: State;
  project?: StudioProject;
  opened?: Loaded;
  generation: number;
  canStart: boolean;
  source: string;
  onSource: () => void;
  onStart: () => void;
  onStop: () => void;
  onOpen: () => void;
  onNew: () => void;
  onNavigate: (area: string) => void;
  onStatus: (status: EditorStatus) => void;
  onProjectChanged?: (project: StudioProject) => void;
};
export default forwardRef<DocumentWorkspacesHandle, Props>(
  function DocumentWorkspaces(p, ref) {
    const controls = useRef(new Map<AreaKind, AreaController>()),
      sessions = useRef(new Map<AreaKind, CloseEntry[]>()),
      handled = useRef("");
    const writes = useRef<Promise<unknown>>(Promise.resolve());
    const gateWrites = useRef<Promise<unknown>>(Promise.resolve());
    useEffect(() => {
      gateWrites.current = gateWrites.current
        .catch(() => {})
        .then(() => api.SetDocumentCloseManaged(true));
      return () => {
        gateWrites.current = gateWrites.current
          .catch(() => {})
          .then(() => api.SetDocumentCloseManaged(false))
          .catch(() => {});
        writes.current = writes.current
          .catch(() => {})
          .then(() => api.SetStudioProjectDirty(false))
          .catch(() => {});
      };
    }, []);
    const latest = useRef(p);
    latest.current = p;
    useEffect(()=>on<{id:string;name:string;open:boolean}>("structure-created",event=>{if(!event.open)return;latest.current.onNavigate("structures");if(event.id)controls.current.get("structures")?.open(event.id,event.name,{sourceId:event.id},true)}),[]);
    const [, setSessionRevision] = useState(0);
    const close = useCloseDocuments((id) =>
      [...sessions.current.values()].flat().find((e) => e.id === id),
    );
    const closeRef = useRef(close.request);
    closeRef.current = close.request;
    const lastDirty = useRef<boolean | undefined>(undefined);
    const onSessions = useCallback((area: AreaKind, entries: CloseEntry[]) => {
      const summary = (items: CloseEntry[]) =>
        JSON.stringify(
          items.map((e) => [
            e.id,
            e.name,
            e.session.dirty,
            e.session.busy,
            e.session.ready,
            e.session.path,
          ]),
        );
      const changed =
        summary(sessions.current.get(area) || []) !== summary(entries);
      sessions.current.set(area, entries);
      if (changed) setSessionRevision((n) => n + 1);
      if (
        changed &&
        area === latest.current.page &&
        !["render", "cinematic", "structures"].includes(area)
      ) {
        const current = entries.find(
          (e) => e.id === controls.current.get(area)?.activeId(),
        );
        latest.current.onStatus({
          ready: !!current && (current.session.ready ?? true),
          busy: entries.some((e) => e.session.busy),
          undo: false,
          redo: false,
        });
      }
      const dirty = [...sessions.current.values()]
        .flat()
        .some((e) => e.session.dirty || e.session.busy);
      if (lastDirty.current !== dirty) {
        lastDirty.current = dirty;
        writes.current = writes.current
          .catch(() => {})
          .then(() => api.SetStudioProjectDirty(dirty))
          .catch(() => {});
      }
    }, []);
    useEffect(() => {
      if (!["render", "cinematic", "structures"].includes(p.page)) {
        const entries = sessions.current.get(p.page as AreaKind) || [];
        const current = entries.find(
          (e) => e.id === controls.current.get(p.page as AreaKind)?.activeId(),
        );
        p.onStatus({
          ready: !!current && (current.session.ready ?? true),
          busy: entries.some((e) => e.session.busy),
          undo: false,
          redo: false,
        });
      }
    }, [p.page]);
    const requestClose = useCallback<CloseRequest>(
      (...args) => closeRef.current(...args),
      [],
    );
    useImperativeHandle(ref, () => ({
      openStructure:(id,name)=>{controls.current.get("structures")?.open(id,name,{sourceId:id},true);latest.current.onNavigate("structures")},
      save: async (saveAs = false) =>
        (await controls.current
          .get(latest.current.page as AreaKind)
          ?.save(saveAs)) ?? false,
      quit: (done) =>
        closeRef.current([...sessions.current.values()].flat(), done, true),
      locked: (area) =>
        controls.current.get(area as AreaKind)?.locked() ?? false,
      openWorld: (kind) => {
        const area: AreaKind =
          latest.current.page === "cinematic" ? "cinematic" : "render";
        if (controls.current.get(area)?.locked()) return;
        const id = crypto.randomUUID();
        controls.current.get(area)?.open(
          "project:" + id,
          "Untitled",
          {
            project: {
              ...newProjectDefaults(),
              id,
              updated: Date.now(),
              page: area,
              worldId: "",
              openKind: kind,
            },
          },
          true,
          id,
        );
        latest.current.onNavigate(area);
      },
      openWorkspace: (raw, id, path) => {
        let workspace: WorkspaceDocument;
        try {
          workspace = parseWorkspaceDocument(raw);
        } catch (e) {
          void api.ReleaseDocumentTab(id).catch(() => {});
          throw e;
        }
        const owner = controls.current.get(workspace.kind);
        const existing = owner?.findSource("file:" + path);
        if (owner?.locked()) {
          void api.ReleaseDocumentTab(id).catch(() => {});
          return;
        }
        if (existing && existing !== id)
          void api.ReleaseDocumentTab(id).catch(() => {});
        controls.current
          .get(workspace.kind)
          ?.open(
            "file:" + path,
            workspace.source.name,
            { workspace, sourceId: workspace.source.id, path },
            true,
            id,
          );
        p.onNavigate(workspace.kind);
      },
    }));
    useEffect(() => {
      const content = (e: Event) => {
        const target = (e as CustomEvent<{ kind: string; id: string }>).detail;
        if (!target || !["packs", "skins"].includes(target.kind)) return;
        controls.current
          .get(target.kind as AreaKind)
          ?.open(
            target.id,
            target.kind === "packs" ? "Resource pack" : "Player skin",
            { sourceId: target.id },
            true,
          );
        sessionStorage.removeItem("craftbox:packet-content");
      };
      window.addEventListener("craftbox:open-content", content);
      return () => window.removeEventListener("craftbox:open-content", content);
    }, []);
    useEffect(() => {
      const navigate = () => {
        const nav = consumePacketNavigation();
        if (!nav.capture) return;
        const workspace: WorkspaceDocument = {
          format: "craftbox-workspace",
          version: 1,
          kind: "packets",
          source: { id: nav.capture, name: "Recording" },
          view: { search: nav.search || "", selected: nav.packet || "" },
        };
        controls.current
          .get("packets")
          ?.open(
            nav.capture,
            "Recording",
            { workspace, sourceId: nav.capture },
            true,
          );
        window.dispatchEvent(
          new CustomEvent("craftbox:packet-focus", { detail: nav }),
        );
      };
      window.addEventListener("craftbox:open-packets", navigate);
      return () =>
        window.removeEventListener("craftbox:open-packets", navigate);
    }, []);
    useEffect(
      () =>
        on("studio:close-requested", () =>
          closeRef.current(
            [...sessions.current.values()].flat(),
            () => void api.ConfirmStudioProjectClose(),
            true,
          ),
        ),
      [],
    );
    useEffect(() => {
      const key = `${p.generation}:${p.project?.id || ""}:${p.opened?.path || ""}`;
      if (key === handled.current) return;
      handled.current = key;
      if (p.opened) {
        const area = p.opened.document.editor;
        const existing = controls.current
          .get(area)
          ?.findSource("file:" + p.opened.path);
        if (
          (existing && existing !== p.opened.world.id) ||
          controls.current.get(area)?.locked()
        ) {
          void api
            .CloseRenderWorld(p.opened.world.id)
            .then(() => api.ReleaseDocumentTab(p.opened!.world.id))
            .catch(() => {});
          if (existing)
            controls.current
              .get(area)
              ?.open("file:" + p.opened.path, p.opened.document.name, {}, true);
          return;
        }
        controls.current
          .get(area)
          ?.open(
            "file:" + p.opened.path,
            p.opened.document.name,
            { loaded: p.opened },
            true,
            p.opened.world.id,
          );
      } else if (
        p.project &&
        (p.project.page === "render" || p.project.page === "cinematic")
      ) {
        controls.current
          .get(p.project.page)
          ?.open(
            "project:" + p.project.id,
            p.project.name || "Untitled",
            { project: p.project },
            true,
            p.project.id,
          );
      }
    }, [p.project, p.opened, p.generation]);
    return (
      <>
        {close.dialog}
        {areas.map((kind) => (
          <DocumentArea
            key={kind}
            kind={kind}
            active={p.page === kind}
            controller={(c) => {
              controls.current.set(kind, c);
            }}
            requestClose={requestClose}
            onSessions={onSessions}
          >
            {(tab, payload, register, open, visible) => (
              <DocumentSlot
                kind={kind}
                tab={tab}
                payload={payload as Payload | undefined}
                register={register}
                open={open}
                visible={visible}
                requestClose={requestClose}
                props={p}
              />
            )}
          </DocumentArea>
        ))}
      </>
    );
  },
);
function DocumentSlot({
  kind,
  tab,
  payload,
  register,
  open,
  visible,
  requestClose,
  props: p,
}: {
  kind: AreaKind;
  tab?: DocumentTab;
  payload?: Payload;
  register: (session: StudioSession | undefined) => void;
  open: AreaController["open"];
  visible: boolean;
  requestClose: CloseRequest;
  props: Props;
}) {
  const latest = useRef({ visible, p, tab });
  latest.current = { visible, p, tab };
  const session = useRef<StudioSession | undefined>(undefined);
  const registered = useCallback(
    (value: StudioSession | undefined) => {
      session.current = value;
      register(value);
    },
    [register],
  );
  const status = useRef<EditorStatus>({
    ready: false,
    busy: false,
    undo: false,
    redo: false,
  });
  const onStatus = useCallback((next: EditorStatus) => {
    status.current = next;
    if (latest.current.visible) latest.current.p.onStatus(next);
  }, []);
  useEffect(() => {
    if (visible && ["render", "cinematic", "structures"].includes(kind))
      p.onStatus(status.current);
  }, [visible, kind]);
  const onOpen = useCallback(
    (id: string, name: string, permanent = false, deferUntilIdle = false) =>
      open(id, name, { sourceId: id, deferUntilIdle }, permanent),
    [open],
  );
  const scope = {
    id: tab?.id || "",
    sourceId: payload?.sourceId,
    path: payload?.path,
    onSaved: (document: WorkspaceDocument, path: string) => {
      if (payload) {
        payload.workspace = document;
        payload.path = path;
      }
    },
    active: visible,
    busy: tab?.busy,
    initial: payload?.workspace,
    register: registered,
    onOpen: ["packs", "skins", "packets", "structures"].includes(kind) ? onOpen : undefined,
  };
  return (
    <DocumentScope.Provider value={scope}>
      <Suspense
        fallback={
          <div className="document-empty" role="status">
            Loading workspace…
          </div>
        }
      >
        {kind === "structures" ? <StructureStudio worlds={p.state.worlds} onStatus={onStatus}/> : kind === "packs" || kind === "skins" ? (
          <AssetStudio kind={kind} onSource={p.onSource} />
        ) : kind === "packets" ? (
          <PacketInspector
            running={p.state.running}
            canStart={p.canStart}
            source={p.source}
            onSource={p.onSource}
            onStart={p.onStart}
            onStop={p.onStop}
          />
        ) : tab ? (
          <RenderEditor
            project={payload?.project}
            loadedProject={payload?.loaded}
            cinematic={kind === "cinematic"}
            worlds={p.state.worlds}
            activeWorldId={p.state.activeWorldId}
            onStudioSession={registered}
            onStudioSaved={(value) => {
              if (payload) payload.loaded = value;
            }}
            onProjectSourceChange={(worldId, sourceKind, sourceName) => {
              if (!payload) return;
              const base = payload.project || {
                ...newProjectDefaults(),
                id: tab.id,
                updated: Date.now(),
                name: payload.loaded?.document.name || "Untitled",
                page: kind as "render" | "cinematic",
              };
              const next = withProjectSource(
                base,
                worldId,
                sourceName,
                sourceKind,
              );
              payload.project = { ...next, openKind: undefined };
              if (!payload.loaded) p.onProjectChanged?.(payload.project);
            }}
            onEditorStateChange={onStatus}
            onStudioOpen={p.onOpen}
            requestTransition={(action) => {
              const s = session.current;
              if (s)
                requestClose(
                  [{ id: tab.id, area: kind, name: s.name, session: s }],
                  action,
                );
              else action();
            }}
          />
        ) : (
          <EmptyProjects onNew={p.onNew} onOpen={p.onOpen} onWorld={()=>open("untitled:"+crypto.randomUUID(),"Untitled",{},true)}/>
        )}
      </Suspense>
    </DocumentScope.Provider>
  );
}

export function EmptyProjects({onNew,onOpen,onWorld}:{onNew:()=>void;onOpen:()=>void;onWorld:()=>void}){return (
          <div className="document-empty">
            <FolderOpen size={32} />
            <strong>No open projects</strong>
            <span>Open a project or create one to start.</span>
            <div className="document-empty-actions">
              <Button onClick={onNew}>
                <FilePlus size={15}/>New project
              </Button>
              <Button variant="outline" onClick={onOpen}>
                <FolderOpen size={15}/>Open project…
              </Button>
              <Button
                variant="ghost"
                onClick={onWorld}
              >
                <Globe size={15}/>Open world…
              </Button>
            </div>
          </div>
);}

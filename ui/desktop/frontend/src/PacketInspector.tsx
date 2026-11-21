import { formatLocale } from "./i18n";
import { useScopedShortcuts } from "./shortcuts";
import ViewSearch from "./ViewSearch";
import FeedbackFlyout from "./FeedbackFlyout";
import CollectionToolbar from "./CollectionToolbar";
import { collectionSelection } from "./collection-selection";
import { useDocumentScope } from "./document-scope";
import { useWorkspaceDocument } from "./use-workspace-document";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Activity,
  Bookmark,
  ChevronLeft,
  ChevronRight,
  Download,
  FileInput,
  FolderOpen,
  Pause,
  Play,
  Radio,
  RefreshCw,
  Send,
  Settings2,
  PanelRight,
  ArrowLeft,
  ListFilter,
  Pin,
  X,
} from "lucide-react";
import { api } from "./bridge";
import {
  cachedPacketStatus,
  refreshPacketStatus,
  subscribePacketStatus,
} from "./packet-status";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import { useWorkspaceActions } from "./workspace-actions";
import PacketSearchDialog from "./PacketSearchDialog";
import PacketDetail from "./PacketDetail";
import PacketAnalysisView from "./PacketAnalysis";
import PacketRules from "./PacketRules";
import {
  consumePacketNavigation,
  type PacketNavigation,
} from "./packet-navigation";
import {
  packetBytes,
  type PacketAnalysis,
  type PacketTypeStat,
  type PacketCapture,
  type PacketPage,
  type PacketQuery,
  type PacketRecord,
  type PacketSend,
  type PacketStatus,
} from "./packet-inspector-model";
import PacketStreamTable, {
  PacketColumnOptions,
  readStreamOptions,
} from "./PacketStreamTable";
import PacketTraffic from "./PacketTraffic";
import {
  PacketCaptureLibrary,
  PacketTypeCatalog,
} from "./PacketCaptureLibrary";
import "./packet-inspector.css";
import "./packet-workbench.css";

const recordingDeleteEvent = "craftbox:recording-delete";
type RecordingDelete = { id: string; phase: "start" | "finished" | "failed" };
function announceRecordingDelete(detail: RecordingDelete) {
  window.dispatchEvent(new CustomEvent(recordingDeleteEvent, { detail }));
}
const views = [
  "Packets",
  "Traffic",
  "Timeline",
  "Entities",
  "Movement",
  "Transactions",
  "Chunks",
  "Changes",
  "Rules",
];
const emptyPage: PacketPage = {
  rows: [],
  total: 0,
  capture: {} as PacketCapture,
};
const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e));
export default function PacketInspector({
  source,
  onSource,
}: {
  source: string;
  onSource: () => void;
}) {
  const scope = useDocumentScope();
  const [recordingsOpen, setRecordingsOpen] = useState(false);
  const [catalogOpen, setCatalogOpen] = useState(false);
  const [streamOptions, setStreamOptions] = useState(readStreamOptions);
  const [reference, setReference] = useState<PacketRecord>();
  const pinRequest = useRef(0);
  const [detailWidth, setDetailWidth] = useState(() => {
    try {
      return Math.max(
        340,
        Math.min(
          800,
          Number(localStorage.getItem("craftbox-packet-detail-width")) || 460,
        ),
      );
    } catch {
      return 460;
    }
  });
  const splitStart = useRef({ x: 0, width: 460 });
  const [searchQuery, setSearchQuery] = useState("");
  const [detailOpen, setDetailOpen] = useState(true);
  const [multiple, setMultiple] = useState(false),
    [recordingSelection, setRecordingSelection] = useState<string[]>([]),
    [recordingAnchor, setRecordingAnchor] = useState("");
  const [deleteTargets, setDeleteTargets] = useState<string[]>([]);
  const blockedReads = useRef(new Set<string>());
  const deletedRecordings = useRef(new Set<string>());
  const readEpochs = useRef(new Map<string, number>());
  const canRead = (id: string) => !!id && !blockedReads.current.has(id);
  const currentRead = (id: string, epoch: number) =>
    canRead(id) && (readEpochs.current.get(id) ?? 0) === epoch;
  const v = scope.initial?.view;
  const initial = useRef<PacketNavigation>(
    scope.onOpen
      ? {
          capture: scope.initial?.source.id || scope.sourceId,
          search: typeof v?.search === "string" ? v.search : "",
          packet: typeof v?.selected === "string" ? v.selected : "",
        }
      : consumePacketNavigation(),
  );
  const [status, setStatus] = useState<PacketStatus | undefined>(
      cachedPacketStatus(),
    ),
    [capture, setCapture] = useState(initial.current.capture || ""),
    [page, setPage] = useState<PacketPage>(emptyPage),
    [pageScope, setPageScope] = useState(""),
    [search, setSearch] = useState(initial.current.search || ""),
    [name, setName] = useState((v?.name as string) ?? ""),
    [direction, setDirection] = useState((v?.direction as string) ?? ""),
    [stage, setStage] = useState((v?.stage as string) ?? ""),
    [bookmarks, setBookmarks] = useState((v?.bookmarks as boolean) ?? false),
    [offset, setOffset] = useState((v?.offset as number) ?? 0),
    [follow, setFollow] = useState(
      (v?.follow as boolean) ?? !initial.current.capture,
    ),
    [tab, setTab] = useState((v?.tab as string) ?? "Packets"),
    [selected, setSelected] = useState(initial.current.packet || ""),
    [packet, setPacket] = useState<PacketRecord>(),
    [analysis, setAnalysis] = useState<PacketAnalysis>(),
    [types, setTypes] = useState<PacketTypeStat[]>([]),
    [reveal, setReveal] = useState(false),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(""),
    [tick, setTick] = useState(0),
    [searchOpen, setSearchOpen] = useState(false),
    [settingsOpen, setSettingsOpen] = useState(false),
    [deleteOpen, setDeleteOpen] = useState(false),
    [metadataOpen, setMetadataOpen] = useState(false),
    [captureName, setCaptureName] = useState(""),
    [notes, setNotes] = useState(""),
    [playback, setPlayback] = useState((v?.playback as boolean) ?? false),
    [cursor, setCursor] = useState((v?.cursor as number) ?? 1),
    [playing, setPlaying] = useState(false),
    [speed, setSpeed] = useState((v?.speed as number) ?? 1),
    [sendOpen, setSendOpen] = useState(false),
    [send, setSend] = useState<PacketSend>(),
    [sendPreview, setSendPreview] = useState<PacketRecord>(),
    [exportOpen, setExportOpen] = useState(false),
    [exportSecrets, setExportSecrets] = useState(false),
    [loading, setLoading] = useState(false);
  const [detailTab, setDetailTab] = useState(
    typeof v?.detailTab === "string" ? v.detailTab : "Overview",
  );
  const settingsEditing = useRef(false);
  settingsEditing.current = settingsOpen;
  const selectedCapture = status?.captures.find((c) => c.id === capture);
  useEffect(() => {
    if (!multiple) {
      setRecordingSelection(capture ? [capture] : []);
      setRecordingAnchor(capture);
    }
  }, [capture, multiple]);
  const live =
    status?.connected &&
    (status.active === capture || status.recordingIDs?.includes(capture));
  useWorkspaceDocument(
    "packets",
    { id: capture, name: selectedCapture?.name || "Recording" },
    {
      search,
      name,
      direction,
      stage,
      bookmarks,
      offset,
      follow,
      tab,
      detailTab,
      selected,
      playback,
      cursor,
      speed,
    },
    !!busy || !!live,
    !!status && !!capture,
  );
  const scopes = useRef({
    capture,
    reveal,
    analysis: `${capture}:${playback ? cursor : 0}`,
    send,
    sendOpen,
    generation: status?.generation,
  });
  scopes.current = {
    capture,
    reveal,
    analysis: `${capture}:${playback ? cursor : 0}`,
    send,
    sendOpen,
    generation: status?.generation,
  };
  const analysisRequest = useRef(0);
  useEffect(() => {
    const timer = setTimeout(() => setSearchQuery(search), 180);
    return () => clearTimeout(timer);
  }, [search]);
  useEffect(() => {
    try {
      localStorage.setItem(
        "craftbox-packet-stream-options",
        JSON.stringify(streamOptions),
      );
    } catch {}
  }, [streamOptions]);
  useEffect(() => {
    try {
      localStorage.setItem("craftbox-packet-detail-width", String(detailWidth));
    } catch {}
  }, [detailWidth]);
  const query = useMemo<PacketQuery>(
    () => ({
      capture,
      search: searchQuery,
      name,
      direction,
      stage,
      bookmarked: bookmarks,
      offset,
      limit: 200,
      reverse: true,
      before: playback ? cursor : 0,
    }),
    [
      capture,
      searchQuery,
      name,
      direction,
      stage,
      bookmarks,
      offset,
      playback,
      cursor,
    ],
  );
  const acceptStatus = useCallback((incoming: PacketStatus) => {
    const value = {
      ...incoming,
      captures: incoming.captures.filter(
        (c) => !deletedRecordings.current.has(c.id),
      ),
    };
    setStatus((old) =>
      settingsEditing.current && old
        ? { ...value, settings: old.settings }
        : value,
    );
    if (!scope.onOpen)
      setCapture((old) =>
        value.captures.some((c) => c.id === old)
          ? old
          : value.active || value.captures[0]?.id || "",
      );
    return value;
  }, []);
  const refresh = refreshPacketStatus;
  const fail = useCallback((e: unknown) => setError(errorText(e)), []);
  const run = async (label: string, job: () => Promise<unknown>) => {
    setBusy(label);
    setError("");
    try {
      await job();
    } catch (e) {
      fail(e);
    } finally {
      setBusy("");
    }
  };
  useEffect(
    () => subscribePacketStatus(acceptStatus, fail),
    [acceptStatus, fail],
  );
  const liveCount = follow && live ? selectedCapture?.count : 0;
  useEffect(() => {
    if (!canRead(capture) || !scope.active) {
      setTypes([]);
      return;
    }
    let cancelled = false;
    const epoch = readEpochs.current.get(capture) ?? 0;
    void api
      .GetPacketTypes(capture)
      .then((value) => {
        if (!cancelled && currentRead(capture, epoch)) setTypes(value);
      })
      .catch((e) => {
        if (!cancelled && currentRead(capture, epoch)) fail(e);
      });
    return () => {
      cancelled = true;
    };
  }, [capture, liveCount, tick, fail, scope.active]);
  const pageRequest = useRef<{
    busy: boolean;
    mounted: boolean;
    pending?: { query: PacketQuery; key: string; epoch: number };
  }>({ busy: false, mounted: true });
  const queryKey = JSON.stringify(query),
    currentQueryKey = useRef(queryKey);
  currentQueryKey.current = queryKey;
  const shownPage = pageScope === queryKey ? page : emptyPage;
  useEffect(() => {
    const changed = (event: Event) => {
      const { id, phase } = (event as CustomEvent<RecordingDelete>).detail;
      if (phase === "failed") {
        blockedReads.current.delete(id);
        setTick((t) => t + 1);
        return;
      }
      blockedReads.current.add(id);
      if (phase === "start")
        readEpochs.current.set(id, (readEpochs.current.get(id) ?? 0) + 1);
      if (pageRequest.current.pending?.query.capture === id)
        pageRequest.current.pending = undefined;
      if (phase === "finished") {
        deletedRecordings.current.add(id);
        setRecordingSelection((old) => old.filter((value) => value !== id));
        setDeleteTargets((old) => old.filter((value) => value !== id));
        setStatus(
          (old) =>
            old && {
              ...old,
              captures: old.captures.filter((c) => c.id !== id),
            },
        );
        if (capture === id) {
          setCapture("");
          setSelected("");
          setPage(emptyPage);
          setPacket(undefined);
          setTypes([]);
          setAnalysis(undefined);
          setLoading(false);
          setError("");
        }
      }
    };
    window.addEventListener(recordingDeleteEvent, changed);
    return () => window.removeEventListener(recordingDeleteEvent, changed);
  }, [capture]);
  useEffect(() => {
    pageRequest.current.mounted = true;
    return () => {
      pageRequest.current.mounted = false;
      pageRequest.current.pending = undefined;
    };
  }, []);
  useEffect(() => {
    if (!canRead(capture) || !scope.active) {
      setPage(emptyPage);
      setLoading(false);
      pageRequest.current.pending = undefined;
      return;
    }
    const state = pageRequest.current;
    state.pending = {
      query,
      key: queryKey,
      epoch: readEpochs.current.get(capture) ?? 0,
    };
    setLoading(true);
    const drain = async () => {
      if (state.busy) return;
      state.busy = true;
      try {
        while (state.pending && state.mounted) {
          const task = state.pending;
          state.pending = undefined;
          try {
            const value = await api.QueryPackets(task.query);
            if (
              state.mounted &&
              currentQueryKey.current === task.key &&
              currentRead(task.query.capture, task.epoch)
            ) {
              setPage(value);
              setPageScope(task.key);
              setLoading(false);
            }
          } catch (e) {
            if (
              state.mounted &&
              currentQueryKey.current === task.key &&
              currentRead(task.query.capture, task.epoch)
            ) {
              fail(e);
              setPage(emptyPage);
              setPageScope(task.key);
              setLoading(false);
            }
          }
        }
      } finally {
        state.busy = false;
      }
    };
    void drain();
  }, [query, queryKey, tick, liveCount, fail, capture, scope.active]);
  useEffect(() => {
    setSelected(
      initial.current.capture === capture ? initial.current.packet || "" : "",
    );
    initial.current = {};
    setPacket(undefined);
    pinRequest.current++;
    setReference(undefined);
    setAnalysis(undefined);
    setName("");
    setOffset(0);
    setReveal(false);
    setPlaying(false);
    setPlayback(false);
  }, [capture]);
  useEffect(() => {
    if (!selected || !canRead(capture) || !scope.active) {
      setPacket(undefined);
      return;
    }
    setPacket(undefined);
    let dead = false;
    const epoch = readEpochs.current.get(capture) ?? 0;
    void api
      .GetPacketDetail(capture, selected, reveal)
      .then((value) => {
        if (!dead && currentRead(capture, epoch)) setPacket(value);
      })
      .catch((e) => {
        if (!dead && currentRead(capture, epoch)) fail(e);
      });
    return () => {
      dead = true;
    };
  }, [capture, selected, reveal, tick, fail, scope.active]);
  useEffect(() => {
    if (!playing || !playback || !scope.active) return;
    const timer = setInterval(
      () =>
        setCursor((old) => {
          const max = selectedCapture?.count || 1;
          if (old >= max) {
            setPlaying(false);
            return max;
          }
          return Math.min(max, old + 1);
        }),
      Math.max(25, 200 / speed),
    );
    return () => clearInterval(timer);
  }, [playing, playback, speed, selectedCapture?.count, scope.active]);
  useScopedShortcuts("packets", scope.active, id => { if (id === "packets.find") setSearchOpen(true); else { setFollow(false); setPlaying(false); } });
  const analyse = () =>
    run("Analysing capture", async () => {
      const scope = scopes.current.analysis,
        request = ++analysisRequest.current;
      const value = await api.GetPacketAnalysis({
        capture,
        before: playback ? cursor : 0,
      });
      if (
        scopes.current.analysis === scope &&
        analysisRequest.current === request
      )
        setAnalysis(value);
    });
  useEffect(() => {
    let cancelled = false;
    if (!canRead(capture) || !scope.active) return;
    const epoch = readEpochs.current.get(capture) ?? 0;
    if (playback) {
      setAnalysis(undefined);
      if (!["Packets", "Rules"].includes(tab))
        void api
          .GetPacketAnalysis({ capture, before: cursor })
          .then((value) => {
            if (!cancelled && currentRead(capture, epoch)) setAnalysis(value);
          })
          .catch((e) => {
            if (!cancelled && currentRead(capture, epoch)) fail(e);
          });
    }
    return () => {
      cancelled = true;
    };
  }, [playback, cursor, capture, scope.active]);
  useEffect(() => {
    if (scope.onOpen) return;
    const navigate = () => {
      if (!scope.active) return;
      const value = consumePacketNavigation();
      initial.current = { ...value, capture: value.capture || capture };
      if (value.capture && scope.onOpen) {
        scope.onOpen(
          value.capture,
          status?.captures.find((c) => c.id === value.capture)?.name ||
            "Recording",
          true,
        );
        return;
      }
      if (value.capture) setCapture(value.capture);
      setSearch(value.search || "");
      setSelected(value.packet || "");
      if (value.packet) setDetailOpen(true);
      setOffset(0);
      setFollow(false);
      setPlayback(false);
      setTab("Packets");
    };
    window.addEventListener("craftbox:open-packets", navigate);
    return () => window.removeEventListener("craftbox:open-packets", navigate);
  }, [capture, scope.active]);
  useEffect(() => {
    const focus = (e: Event) => {
      const value = (e as CustomEvent<PacketNavigation>).detail;
      if (value.capture !== capture) return;
      setSearch(value.search || "");
      setSelected(value.packet || "");
      if (value.packet) setDetailOpen(true);
      setOffset(0);
      setFollow(false);
      setTab("Packets");
    };
    window.addEventListener("craftbox:packet-focus", focus);
    return () => window.removeEventListener("craftbox:packet-focus", focus);
  }, [capture]);
  const chooseTab = (value: string) => {
    setTab(value);
    if (!["Packets", "Rules"].includes(value) && !analysis) void analyse();
  };
  function importCapture() {
    void run("Importing capture", async () => {
      const imported = await api.ImportPacketCapture("");
      if (!imported.id) return;
      await refresh();
      setRecordingsOpen(false);
      if (scope.onOpen) scope.onOpen(imported.id, imported.name, true, true);
      else {
        setCapture(imported.id);
        setFollow(false);
      }
    });
  }
  const choosePacket = (id: string) => {
    setFollow(false);
    setSelected(id);
    setDetailOpen(true);
  };
  const jump = (id: string) => {
    setPlayback(false);
    setPlaying(false);
    setTab("Packets");
    setSearch("");
    setName("");
    setDirection("");
    setStage("");
    setBookmarks(false);
    setOffset(
      Math.floor(
        Math.max(0, (selectedCapture?.count || 0) - Number(id)) / 200,
      ) * 200,
    );
    choosePacket(id);
  };
  useWorkspaceActions("packets", (action) => {
    if (busy) {
      setError("Wait for the current inspector operation to finish.");
      return;
    }
    if (action === "search") setSearchOpen(true);
    else if (action === "settings") setSettingsOpen(true);
    else if (action === "rules") setTab("Rules");
    else if (action === "export") {
      if (capture) setExportOpen(true);
      else setError("Select a recording to export.");
    } else if (action === "refresh") setTick((t) => t + 1);
    else if (action === "import-files")
      void run("Importing capture", async () => {
        const c = await api.ImportPacketCapture("");
        if (c.id) {
          await refresh();
          if (scope.onOpen) {
            scope.onOpen(c.id, c.name, true, true);
            return;
          }
          setCapture(c.id);
          setFollow(false);
        }
      });
  });
  const prepareSend = (p: PacketRecord) => {
    if (p.redacted) {
      setError("Reveal sensitive data before preparing this packet.");
      return;
    }
    setSend({
      generation: status?.generation || "",
      direction: p.direction,
      packetId: p.packetId,
      payload: p.payload,
      raw: p.raw || "",
      useRaw: false,
    });
    setSendPreview(undefined);
    setSendOpen(true);
  };
  const filterType = (value: string) => {
    setFollow(false);
    setName(value);
    setOffset(0);
    setTab("Packets");
  };
  const resetFilters = (type = "") => {
    setName(type);
    setSearch("");
    setDirection("");
    setStage("");
    setBookmarks(false);
    setPlayback(false);
    setPlaying(false);
    setFollow(false);
    setOffset(0);
  };
  const selectedType = types.find((t) => t.name === name);
  const directionExcluded =
    !!selectedType &&
    !!direction &&
    (direction === "serverbound"
      ? selectedType.serverbound === 0
      : selectedType.clientbound === 0);
  const hasFilters = !!(
    search ||
    name ||
    direction ||
    stage ||
    bookmarks ||
    playback
  );
  const selectedKeys = new Set(shownPage.rows.map((r) => r.id));
  const openCapture = (item: PacketCapture) => {
    if (scope.onOpen) scope.onOpen(item.id, item.name, true);
    else {
      setCapture(item.id);
      setFollow(false);
    }
  };
  const pinPacket = (value: PacketRecord) =>
    void run("Loading reference packet", async () => {
      const epoch = readEpochs.current.get(capture) ?? 0;
      const requestCapture = capture;
      const requestReveal = reveal;
      const request = ++pinRequest.current;
      const loaded = await api.GetPacketDetail(capture, value.id, reveal);
      if (
        scopes.current.capture === requestCapture &&
        scopes.current.reveal === requestReveal &&
        pinRequest.current === request &&
        currentRead(requestCapture, epoch)
      ) {
        setReference(loaded);
        setDetailTab("Compare");
        choosePacket(value.id);
      }
    });
  const bookmarkPacket = (value: PacketRecord) =>
    void run("Updating bookmark", async () => {
      await api.UpdatePacketCapture({
        capture,
        packet: value.id,
        annotation: { note: value.note || "", bookmarked: !value.bookmarked },
      });
      setTick((t) => t + 1);
    });
  const excludeType = (value: string) => {
    setFollow(false);
    setSearch((old) => `${old} -name:${value}`.trim());
    setOffset(0);
    if (name === value) setName("");
  };
  return (
    <section
      className="packet-inspector packet-workbench"
      aria-label="Packet Inspector"
    >
      <header className="pi-main-toolbar studio-view-heading has-view-search">
        <div className="view-heading-title">
          {capture && scope.onLibrary && (
            <button aria-label="Back to recordings" onClick={scope.onLibrary}>
              <ArrowLeft size={14} />
              Recordings
            </button>
          )}
          <Activity size={17} />
          <strong>Packet Inspector</strong>
          <span
            className={`pi-connection ${status?.connected ? "connected" : ""}`}
          >
            <i />
            {status?.connected ? "NetherNet connected" : "Offline"}
          </span>
        </div>
        <ViewSearch
          label={capture ? "Search packets" : "Search recordings"}
          placeholder={
            capture
              ? "Search packets · name:, field:, hex:, time:"
              : "Search recordings or sources"
          }
          value={search}
          onChange={(value) => {
            setFollow(false);
            setSearch(value);
            setOffset(0);
          }}
          onFilters={() => setSearchOpen(true)}
        />
        <div className="view-heading-actions">
          <button
            className="toolbar-icon-action"
            aria-label="Import packet capture"
            title="Import packet capture"
            disabled={!!busy}
            onClick={importCapture}
          >
            <FileInput size={15} />
          </button>

          <button
            className="toolbar-icon-action"
            aria-label="Packet details"
            data-tooltip="Packet details"
            aria-pressed={!!packet && detailOpen}
            disabled={!packet}
            onClick={() => setDetailOpen((value) => !value)}
          >
            <PanelRight size={15} />
          </button>
          <button
            title="Select source"
            aria-label="Select source"
            className="toolbar-icon-action"
            onClick={onSource}
          >
            <FolderOpen size={14} />
            <span className="toolbar-action-label">
              {source || "Select source"}
            </span>
          </button>
          <button
            title={
              status?.settings.recording ? "Pause recording" : "Record packets"
            }
            aria-label={
              status?.settings.recording ? "Pause recording" : "Record packets"
            }
            className="toolbar-icon-action"
            disabled={!status || !!busy}
            onClick={() =>
              void run("Changing recorder", async () => {
                await api.ConfigurePacketInspector({
                  ...status!.settings,
                  recording: !status!.settings.recording,
                });
                await refresh();
              })
            }
          >
            {status?.settings.recording ? (
              <Pause size={14} />
            ) : (
              <Radio size={14} />
            )}
            <span className="toolbar-action-label"> </span>
            <span className="toolbar-action-label">
              {status?.settings.recording
                ? "Pause recording"
                : "Record packets"}
            </span>
          </button>
          <button
            title="Inspector settings"
            aria-label="Inspector settings"
            onClick={() => setSettingsOpen(true)}
          >
            <Settings2 size={16} />
          </button>
        </div>
      </header>
      {(error || status?.error || selectedCapture?.error) && (
        <FeedbackFlyout title="Packet Inspector">
          <span>{error || status?.error || selectedCapture?.error}</span>
          <button aria-label="Dismiss error" onClick={() => setError("")}>
            <X size={13} />
          </button>
        </FeedbackFlyout>
      )}
      {(status?.queueDropped || selectedCapture?.dropped || 0) > 0 && (
        <FeedbackFlyout title="Packet Inspector" tone="warning">
          Capture gaps: {selectedCapture?.dropped || status?.queueDropped}{" "}
          observations could not be retained.
        </FeedbackFlyout>
      )}
      <PacketSearchDialog
        open={searchOpen}
        onOpenChange={setSearchOpen}
        capture={capture}
        types={types}
        initial={{
          capture,
          search,
          name,
          direction,
          stage,
          bookmarked: bookmarks,
        }}
        onApply={(q, id) => {
          setFollow(false);
          setSearch(q.search || "");
          setName(q.name || "");
          setDirection(q.direction || "");
          setStage(q.stage || "");
          setBookmarks(!!q.bookmarked);
          setOffset(q.offset || 0);
          setTab("Packets");
          if (id) choosePacket(id);
        }}
      />
      {capture && (
        <div className="pi-session-strip">
          <label>
            <span>Recording</span>
            <select
              aria-label="Current recording"
              value={capture}
              onChange={(event) => {
                const item = status?.captures.find(
                  (value) => value.id === event.target.value,
                );
                if (item) openCapture(item);
              }}
            >
              <option value="">Choose recording</option>
              {status?.captures.map((item) => (
                <option key={item.id} value={item.id} translate="no">
                  {item.name}
                </option>
              ))}
            </select>
          </label>
          <span>
            {selectedCapture?.count.toLocaleString(formatLocale()) ?? 0} observations
          </span>
          <span>{packetBytes(selectedCapture?.bytes || 0)}</span>
          <span>Protocol {selectedCapture?.protocol || "unknown"}</span>
          <span className="pi-session-spacer" />
          <span
            className={status?.settings.intervention ? "pi-intervention" : ""}
          >
            {status?.settings.intervention
              ? "Intervention enabled"
              : "Passive inspection"}
          </span>
          <button
            aria-label="Manage recordings"
            onClick={() => setRecordingsOpen(true)}
          >
            <FolderOpen size={14} />
            Recordings
          </button>
          <button
            aria-label="Export current recording"
            disabled={!!busy}
            onClick={() => {
              setExportSecrets(false);
              setExportOpen(true);
            }}
          >
            <Download size={14} />
            Export
          </button>
        </div>
      )}
      <PacketTypeCatalog
        open={catalogOpen}
        onOpenChange={setCatalogOpen}
        types={types}
        onType={filterType}
      />
      <Dialog open={recordingsOpen} onOpenChange={setRecordingsOpen}>
        <DialogContent className="collection-picker-dialog">
          <DialogHeader>
            <DialogTitle>Recordings</DialogTitle>
            <DialogDescription>
              Open, import, export or manage captured sessions.
            </DialogDescription>
          </DialogHeader>
          <div className="packet-recording-picker">
            <div className="collection-view-tools">
              <button
                disabled={!!busy}
                onClick={importCapture}
                aria-label="Import packet capture"
              >
                <FileInput size={14} />
                Import capture
              </button>
            </div>

            <CollectionToolbar
              ids={status?.captures.map((c) => c.id) ?? []}
              selected={recordingSelection}
              multiple={multiple}
              onMultiple={setMultiple}
              onSelected={setRecordingSelection}
              busy={!!busy}
              deleteDisabled={
                !!status?.connected &&
                recordingSelection.some(
                  (id) =>
                    id === status.active || status.recordingIDs?.includes(id),
                )
              }
              onDelete={(ids) => {
                setDeleteTargets(ids);
                setDeleteOpen(true);
              }}
            />
            <div className="pi-capture-list">
              {status?.captures.map((c) => (
                <button
                  className={capture === c.id ? "selected" : ""}
                  aria-pressed={recordingSelection.includes(c.id)}
                  key={c.id}
                  onDoubleClick={() => {
                    if (!multiple) scope.onOpen?.(c.id, c.name, true);
                  }}
                  onClick={(e) => {
                    const next = collectionSelection(
                      status?.captures.map((value) => value.id) ?? [],
                      recordingSelection,
                      recordingAnchor,
                      c.id,
                      e,
                      multiple,
                    );
                    setRecordingSelection(next.selected);
                    setRecordingAnchor(next.anchor);
                    setMultiple(next.multiple);
                    if (next.multiple) return;
                    setRecordingsOpen(false);
                    if (scope.onOpen) {
                      scope.onOpen(c.id, c.name, e.detail >= 2);
                      return;
                    }
                    setCapture(c.id);
                    setFollow(c.id === status.active && status.connected);
                  }}
                >
                  <span>
                    <strong>{c.name}</strong>
                    {c.id === status.active && status.connected && (
                      <i className="pi-live-dot" />
                    )}
                  </span>
                  <small>{new Date(c.started).toLocaleString(formatLocale())}</small>
                  <small>
                    {c.count.toLocaleString(formatLocale())} packets ·{" "}
                    {packetBytes(c.diskBytes)}
                    {c.dropped ? " · Gaps" : ""}
                  </small>
                </button>
              ))}
              {!status?.captures.length && (
                <p>No recordings yet. Start NetherNet or import a capture.</p>
              )}
            </div>
            {selectedCapture && (
              <div className="pi-capture-actions">
                <button
                  onClick={() => {
                    setCaptureName(selectedCapture.name);
                    setNotes(selectedCapture.notes);
                    setMetadataOpen(true);
                  }}
                >
                  Rename & notes
                </button>
                <button
                  disabled={!!busy}
                  onClick={() => {
                    setExportSecrets(false);
                    setExportOpen(true);
                  }}
                >
                  <Download size={13} />
                  Export
                </button>
              </div>
            )}
          </div>{" "}
        </DialogContent>
      </Dialog>
      <div
        className={`pi-workspace${packet && detailOpen ? " has-detail" : ""}`}
        style={
          { "--pi-detail-width": `${detailWidth}px` } as React.CSSProperties
        }
      >
        <main className="pi-center">
          <div data-editor-tab-host />
          {!capture ? (
            <PacketCaptureLibrary
              captures={status?.captures || []}
              active={status?.connected ? status.active : ""}
              query={search}
              busy={!!busy}
              onOpen={openCapture}
              onImport={importCapture}
              onManage={() => setRecordingsOpen(true)}
              onSource={onSource}
            />
          ) : (
            <>
              <nav
                className="pi-tabs pi-workspace-tabs"
                aria-label="Inspector workspaces"
              >
                {views.map((v) => (
                  <button
                    className={tab === v ? "active" : ""}
                    aria-pressed={tab === v}
                    key={v}
                    onClick={() => chooseTab(v)}
                  >
                    {v}
                  </button>
                ))}
              </nav>
              {tab === "Packets" && (
                <div className="pi-filterbar">
                  <button
                    aria-label="Browse packet types"
                    onClick={() => setCatalogOpen(true)}
                  >
                    <ListFilter size={14} />
                    Types <span>{types.length}</span>
                  </button>
                  <select
                    aria-label="Packet type"
                    value={name}
                    onChange={(event) => filterType(event.target.value)}
                  >
                    <option value="">All packet types</option>
                    {types.map((item) => (
                      <option
                        key={item.name}
                        value={item.name}
                        disabled={item.other}
                      >
                        {item.name} ({item.count})
                      </option>
                    ))}
                  </select>
                  {hasFilters && (
                    <button onClick={() => resetFilters()}>
                      Reset filters
                      <X size={12} />
                    </button>
                  )}
                  <select
                    aria-label="Packet direction"
                    value={direction}
                    onChange={(e) => {
                      setFollow(false);
                      setDirection(e.target.value);
                      setOffset(0);
                    }}
                  >
                    <option value="">Both directions</option>
                    <option value="serverbound">Serverbound</option>
                    <option value="clientbound">Clientbound</option>
                  </select>
                  <select
                    aria-label="Observation stage"
                    value={stage}
                    onChange={(e) => {
                      setFollow(false);
                      setStage(e.target.value);
                      setOffset(0);
                    }}
                  >
                    <option value="">All observations</option>
                    {[
                      "received",
                      "sent",
                      "decision",
                      "injected",
                      "imported",
                    ].map((s) => (
                      <option key={s}>{s}</option>
                    ))}
                  </select>
                  <button
                    aria-label="Bookmarked only"
                    aria-pressed={bookmarks}
                    className={bookmarks ? "selected" : ""}
                    onClick={() => {
                      setFollow(false);
                      setBookmarks(!bookmarks);
                      setOffset(0);
                    }}
                  >
                    <Bookmark size={14} />
                  </button>
                  <span className="pi-filter-spacer" />
                  {reference && (
                    <button
                      title={`Reference #${reference.id}`}
                      onClick={() => {
                        setDetailOpen(true);
                        setDetailTab("Compare");
                      }}
                    >
                      <Pin size={12} />
                      <span>#{reference.id}</span>
                    </button>
                  )}
                  <PacketColumnOptions
                    value={streamOptions}
                    onChange={setStreamOptions}
                  />
                </div>
              )}
              {tab === "Packets" ? (
                <>
                  <PacketStreamTable
                    rows={shownPage.rows}
                    selected={selected}
                    reference={reference?.id}
                    options={streamOptions}
                    onSelect={choosePacket}
                    onType={filterType}
                    onExclude={excludeType}
                    onPin={pinPacket}
                    onBookmark={bookmarkPacket}
                  >
                    {!shownPage.rows.length && loading && (
                      <div className="pi-empty">
                        <Activity size={30} />
                        <RefreshCw className="pi-spin" size={18} />
                        <p>Loading recorded packets</p>
                      </div>
                    )}
                    {!shownPage.rows.length && !loading && (
                      <div className="pi-empty">
                        <Activity size={30} />
                        <h3>
                          {capture
                            ? "No matching packets"
                            : "Ready to inspect NetherNet"}
                        </h3>
                        <p>
                          {capture
                            ? directionExcluded
                              ? `No ${name} observations were recorded in the ${direction} direction.`
                              : selectedType
                                ? `${selectedType.count.toLocaleString(formatLocale())} ${name} observations are recorded. The active filters exclude them.`
                                : "Change your filters or resume recording."
                            : "Select a source and start a connection, or import a saved recording."}
                        </p>
                        {capture && hasFilters && (
                          <button onClick={() => resetFilters(name)}>
                            {name
                              ? `Show all ${name}`
                              : "Show all recorded packets"}
                          </button>
                        )}
                      </div>
                    )}
                  </PacketStreamTable>
                  <footer className="pi-table-footer">
                    <span>
                      {loading
                        ? "Loading…"
                        : `${shownPage.total.toLocaleString(formatLocale())} matches`}
                      {selected && !selectedKeys.has(selected)
                        ? ` · Inspecting #${selected}`
                        : ""}
                    </span>
                    <div>
                      <button
                        aria-label="Previous packet page"
                        disabled={!offset}
                        onClick={() => {
                          setFollow(false);
                          setOffset(Math.max(0, offset - 200));
                        }}
                      >
                        <ChevronLeft size={14} />
                      </button>
                      <span>
                        {shownPage.total ? offset + 1 : 0}–
                        {Math.min(offset + 200, shownPage.total)}
                      </span>
                      <button
                        aria-label="Next packet page"
                        disabled={offset + 200 >= shownPage.total}
                        onClick={() => {
                          setFollow(false);
                          setOffset(offset + 200);
                        }}
                      >
                        <ChevronRight size={14} />
                      </button>
                      <button
                        className={follow ? "selected" : ""}
                        aria-pressed={follow}
                        disabled={!live}
                        onClick={() => {
                          setFollow(!follow);
                          setOffset(0);
                          setTick((t) => t + 1);
                        }}
                      >
                        {follow ? "Following live" : "Follow live"}
                      </button>
                      <button
                        title="Refresh packet page"
                        aria-label="Refresh packet page"
                        onClick={() => setTick((t) => t + 1)}
                      >
                        <RefreshCw size={13} />
                      </button>
                    </div>
                  </footer>
                </>
              ) : tab === "Traffic" ? (
                <PacketTraffic
                  analysis={analysis}
                  onType={filterType}
                  onError={fail}
                  onWindow={(value) => {
                    resetFilters();
                    setSearch(value);
                    setTab("Packets");
                  }}
                />
              ) : tab === "Rules" && status ? (
                <PacketRules
                  key={capture || "rules"}
                  status={status}
                  onChanged={() => void refresh()}
                  onError={fail}
                />
              ) : (
                <PacketAnalysisView
                  analysis={analysis}
                  tab={tab}
                  onPacket={jump}
                  onType={filterType}
                />
              )}
              {tab !== "Packets" && tab !== "Rules" && (
                <div className="pi-analysis-controls">
                  <span>
                    {analysis
                      ? `${analysis.capture.count.toLocaleString(formatLocale())} analysed observations`
                      : "Capture-wide analysis"}
                  </span>
                  <button
                    aria-label="Refresh analysis"
                    disabled={!!busy || !capture}
                    onClick={() => void analyse()}
                  >
                    <RefreshCw size={13} />
                    Refresh analysis
                  </button>
                </div>
              )}
              <div className="pi-playback">
                <label>
                  <input
                    type="checkbox"
                    checked={playback}
                    disabled={!capture}
                    onChange={(e) => {
                      setPlayback(e.target.checked);
                      setPlaying(false);
                      setFollow(false);
                      setOffset(0);
                      setCursor(Math.max(1, selectedCapture?.count || 1));
                    }}
                  />
                  Offline playback
                </label>
                {playback && (
                  <>
                    <button
                      aria-label={playing ? "Pause playback" : "Play capture"}
                      onClick={() => setPlaying(!playing)}
                    >
                      {playing ? <Pause size={13} /> : <Play size={13} />}
                    </button>
                    <button
                      aria-label="Step back"
                      onClick={() => setCursor((c) => Math.max(1, c - 1))}
                    >
                      <ChevronLeft size={13} />
                    </button>
                    <input
                      aria-label="Playback position"
                      type="range"
                      min={1}
                      max={Math.max(1, selectedCapture?.count || 1)}
                      value={cursor}
                      onChange={(e) => {
                        setCursor(Number(e.target.value));
                        setPlaying(false);
                      }}
                    />
                    <button
                      aria-label="Step forward"
                      onClick={() =>
                        setCursor((c) =>
                          Math.min(selectedCapture?.count || 1, c + 1),
                        )
                      }
                    >
                      <ChevronRight size={13} />
                    </button>
                    <span>#{cursor}</span>
                    <select
                      aria-label="Playback speed"
                      value={speed}
                      onChange={(e) => setSpeed(Number(e.target.value))}
                    >
                      {[0.25, 0.5, 1, 2, 4, 10].map((n) => (
                        <option value={n} key={n}>
                          {n}×
                        </option>
                      ))}
                    </select>
                  </>
                )}
                {!playback && (
                  <span>Offline playback never sends packets.</span>
                )}
              </div>
            </>
          )}
        </main>
        {packet && detailOpen && (
          <>
            <div
              className="pi-detail-splitter"
              role="separator"
              aria-label="Resize packet details"
              aria-orientation="vertical"
              tabIndex={0}
              aria-valuenow={detailWidth}
              aria-valuemin={340}
              aria-valuemax={800}
              onPointerDown={(event) => {
                splitStart.current = { x: event.clientX, width: detailWidth };
                event.currentTarget.setPointerCapture(event.pointerId);
              }}
              onPointerMove={(event) => {
                if (event.currentTarget.hasPointerCapture(event.pointerId))
                  setDetailWidth(
                    Math.max(
                      340,
                      Math.min(
                        800,
                        splitStart.current.width +
                          splitStart.current.x -
                          event.clientX,
                      ),
                    ),
                  );
              }}
              onPointerUp={(event) => {
                if (event.currentTarget.hasPointerCapture(event.pointerId))
                  event.currentTarget.releasePointerCapture(event.pointerId);
              }}
              onKeyDown={(event) => {
                if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
                  event.preventDefault();
                  setDetailWidth((old) =>
                    Math.max(
                      340,
                      Math.min(
                        800,
                        old + (event.key === "ArrowLeft" ? 24 : -24),
                      ),
                    ),
                  );
                }
              }}
              onDoubleClick={() => setDetailWidth(460)}
            />
            <PacketDetail
              reference={reference}
              onPin={() => packet && pinPacket(packet)}
              onClearReference={() => {
                pinRequest.current++;
                setReference(undefined);
              }}
              onClose={() => setDetailOpen(false)}
              initialTab={detailTab}
              onTabChange={setDetailTab}
              capture={capture}
              packet={packet}
              reveal={reveal}
              onReveal={(value) => {
                pinRequest.current++;
                setReference(undefined);
                setReveal(value);
                setSendOpen(false);
                setSend(undefined);
                setSendPreview(undefined);
              }}
              onError={fail}
              onChanged={() => {
                setTick((t) => t + 1);
                void refresh();
              }}
              onSend={prepareSend}
            />
          </>
        )}
      </div>
      {busy && (
        <div className="pi-busy" role="status">
          {busy}…
        </div>
      )}
      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
        <DialogContent className="pi-dialog">
          <DialogHeader>
            <DialogTitle>Packet Inspector settings</DialogTitle>
            <DialogDescription>
              One NetherNet connection shared with CraftBox capture modules.
            </DialogDescription>
          </DialogHeader>
          {status && (
            <>
              <div className="studio-dialog-body pi-dialog-body">
                <label className="pi-check">
                  <input
                    type="checkbox"
                    checked={status.settings.recording}
                    onChange={(e) =>
                      setStatus({
                        ...status,
                        settings: {
                          ...status.settings,
                          recording: e.target.checked,
                        },
                      })
                    }
                  />
                  Record packet observations
                </label>
                <label className="pi-check">
                  <input
                    type="checkbox"
                    checked={status.settings.intervention}
                    onChange={(e) =>
                      setStatus({
                        ...status,
                        settings: {
                          ...status.settings,
                          intervention: e.target.checked,
                        },
                      })
                    }
                  />
                  Enable intervention: edit, block, delay and send
                </label>
                <label>
                  Archive limit per recording (MiB)
                  <input
                    type="number"
                    min={16}
                    max={1048576}
                    value={status.settings.maxDiskMB}
                    onChange={(e) =>
                      setStatus({
                        ...status,
                        settings: {
                          ...status.settings,
                          maxDiskMB: Number(e.target.value),
                        },
                      })
                    }
                  />
                </label>
                <h4>Additional capture modules</h4>
                {(
                  ["captureWorlds", "capturePacks", "captureSkins"] as const
                ).map((key, i) => (
                  <label className="pi-check" key={key}>
                    <input
                      type="checkbox"
                      checked={status.settings[key]}
                      onChange={(e) =>
                        setStatus({
                          ...status,
                          settings: {
                            ...status.settings,
                            [key]: e.target.checked,
                          },
                        })
                      }
                    />
                    {["Worlds", "Resource packs", "Player skins"][i]}
                  </label>
                ))}
                <p>
                  Module selection applies when starting the next connection.
                  The current command continues its own capture.
                </p>
              </div>
              <DialogFooter>
                <button onClick={() => setSettingsOpen(false)}>Cancel</button>
                <button
                  className="pi-primary"
                  onClick={() =>
                    void run("Saving inspector settings", async () => {
                      await api.ConfigurePacketInspector(status.settings);
                      setSettingsOpen(false);
                      await refresh();
                    })
                  }
                >
                  Save settings
                </button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
      <Dialog open={metadataOpen} onOpenChange={setMetadataOpen}>
        <DialogContent className="pi-dialog">
          <DialogHeader>
            <DialogTitle>Recording details</DialogTitle>
            <DialogDescription>
              Rename this recording and keep investigation notes.
            </DialogDescription>
          </DialogHeader>
          <div className="studio-dialog-body pi-dialog-body">
            <label>
              Name
              <input
                value={captureName}
                maxLength={160}
                onChange={(e) => setCaptureName(e.target.value)}
              />
            </label>
            <label>
              Notes
              <textarea
                value={notes}
                maxLength={8192}
                rows={6}
                onChange={(e) => setNotes(e.target.value)}
              />
            </label>
          </div>
          <DialogFooter>
            <button onClick={() => setMetadataOpen(false)}>Cancel</button>
            <button
              className="pi-primary"
              onClick={() =>
                void run("Saving recording", async () => {
                  await api.UpdatePacketCapture({
                    capture,
                    name: captureName,
                    notes,
                  });
                  await refresh();
                  setMetadataOpen(false);
                })
              }
            >
              Save
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        lockWhileBusy={!!busy}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
      >
        <DialogContent
          resizable={false}
          className="pi-dialog"
          tone="confirmation"
        >
          <DialogHeader>
            <DialogTitle>
              {deleteTargets.length > 1
                ? `Delete ${deleteTargets.length} recordings?`
                : "Delete recording?"}
            </DialogTitle>
            <DialogDescription>
              Deletes{" "}
              {(deleteTargets.length ? deleteTargets : [capture])
                .map(
                  (id) => status?.captures.find((c) => c.id === id)?.name ?? id,
                )
                .join(", ")}{" "}
              and its saved packet data. Imported source files are kept.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <button disabled={!!busy} onClick={() => setDeleteOpen(false)}>
              Cancel
            </button>
            <button
              className="pi-danger"
              disabled={!!busy}
              onClick={() =>
                void run("Deleting recording", async () => {
                  const targets = deleteTargets.length
                    ? deleteTargets
                    : [capture];
                  for (let index = 0; index < targets.length; index++) {
                    const id = targets[index];
                    announceRecordingDelete({ id, phase: "start" });
                    try {
                      await api.DeletePacketCapture(id);
                    } catch (error) {
                      announceRecordingDelete({ id, phase: "failed" });
                      setDeleteTargets(targets.slice(index));
                      await refresh();
                      throw error;
                    }
                    announceRecordingDelete({ id, phase: "finished" });
                  }
                  setDeleteTargets([]);
                  setRecordingSelection([]);
                  setDeleteOpen(false);
                  setCapture("");
                  await refresh();
                })
              }
            >
              {busy === "Deleting recording" ? "Deleting…" : "Delete recording"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        lockWhileBusy={!!busy}
        open={exportOpen}
        onOpenChange={setExportOpen}
      >
        <DialogContent className="pi-dialog">
          <DialogHeader>
            <DialogTitle>Export filtered recording</DialogTitle>
            <DialogDescription>
              Exports every matching packet, including rows outside this
              shownPage.
            </DialogDescription>
          </DialogHeader>
          <div className="studio-dialog-body pi-dialog-body">
            <label className="pi-check">
              <input
                type="checkbox"
                checked={exportSecrets}
                onChange={(e) => setExportSecrets(e.target.checked)}
              />
              Include sensitive data in this export
            </label>
            <p>
              {exportSecrets
                ? "The export may contain authentication material and pack keys."
                : "Tokens, keys and associated wire bytes are omitted."}
            </p>
          </div>
          <DialogFooter>
            <button onClick={() => setExportOpen(false)}>Cancel</button>
            <button
              className="pi-primary"
              onClick={() =>
                void run("Exporting recording", async () => {
                  await api.ExportPacketCapture(
                    { ...query, offset: 0 },
                    exportSecrets,
                  );
                  setExportOpen(false);
                })
              }
            >
              Export JSON
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={sendOpen} onOpenChange={setSendOpen}>
        <DialogContent className="pi-dialog pi-send-dialog">
          <DialogHeader>
            <DialogTitle>Send a packet</DialogTitle>
            <DialogDescription>
              Targets the active NetherNet connection. Preview the exact packet
              before sending.
            </DialogDescription>
          </DialogHeader>
          {send && (
            <>
              <div className="studio-dialog-body pi-dialog-body">
                <div className="pi-inline-actions">
                  <label>
                    Packet ID
                    <input
                      type="number"
                      min={0}
                      value={send.packetId}
                      onChange={(e) => {
                        setSend({ ...send, packetId: Number(e.target.value) });
                        setSendPreview(undefined);
                      }}
                    />
                  </label>
                  <label>
                    Destination
                    <select
                      value={send.direction}
                      onChange={(e) => {
                        setSend({ ...send, direction: e.target.value });
                        setSendPreview(undefined);
                      }}
                    >
                      <option value="serverbound">Server</option>
                      <option value="clientbound">Minecraft client</option>
                    </select>
                  </label>
                  <label className="pi-check">
                    <input
                      type="checkbox"
                      checked={send.useRaw}
                      onChange={(e) => {
                        setSend({ ...send, useRaw: e.target.checked });
                        setSendPreview(undefined);
                      }}
                    />
                    Raw bytes
                  </label>
                </div>
                <label>
                  {send.useRaw ? "Base64 packet body" : "Complete decoded JSON"}
                  <textarea
                    rows={13}
                    value={send.useRaw ? send.raw : send.payload}
                    onChange={(e) => {
                      setSend({
                        ...send,
                        [send.useRaw ? "raw" : "payload"]: e.target.value,
                      });
                      setSendPreview(undefined);
                    }}
                  />
                </label>
                {sendPreview && (
                  <div className="pi-send-preview">
                    <strong>
                      {sendPreview.name} · {packetBytes(sendPreview.bytes)} ·{" "}
                      {sendPreview.direction}
                    </strong>
                    <pre>{sendPreview.payload}</pre>
                  </div>
                )}
                {!status?.settings.intervention && (
                  <p>
                    Enable intervention in Inspector settings to send. Preview
                    remains available.
                  </p>
                )}
              </div>
              <DialogFooter>
                <button
                  onClick={() =>
                    void run("Preparing packet", async () => {
                      const generation = status?.generation;
                      const value = await api.PreviewPacketSend({
                        ...send,
                        generation: status?.generation || "",
                      });
                      if (
                        scopes.current.send === send &&
                        scopes.current.sendOpen &&
                        scopes.current.generation === generation
                      )
                        setSendPreview(value);
                    })
                  }
                >
                  Preview packet
                </button>
                <button
                  className="pi-primary"
                  disabled={
                    !sendPreview ||
                    !status?.connected ||
                    !status.settings.intervention ||
                    !!busy
                  }
                  onClick={() =>
                    void run("Sending packet", async () => {
                      await api.SendInspectorPacket({
                        ...send,
                        previewToken: sendPreview?.previewToken || "",
                        generation: status?.generation || "",
                      });
                      setSendOpen(false);
                      setTick((t) => t + 1);
                    })
                  }
                >
                  <Send size={13} />
                  Send once
                </button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </section>
  );
}

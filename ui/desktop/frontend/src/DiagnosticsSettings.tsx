import FeedbackFlyout from "./FeedbackFlyout";
import { useCallback, useEffect, useState } from "react";
import {
  RefreshCw,
  FolderOpen,
  FileText,
  Trash2,
  Activity,
} from "lucide-react";
import {
  api,
  type LogStorage,
  type LogPolicy,
  type SavedLogPreview,
} from "./bridge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import "./diagnostics-settings.css";
const defaults: LogPolicy = {
  maxBytes: 100 * 1048576,
  maxAgeDays: 14,
  maxSessions: 20,
  level: "debug",
};
const size = (n: number) => `${(n / 1048576).toFixed(1)} MiB`;
export default function DiagnosticsSettings() {
  const [storage, setStorage] = useState<LogStorage>(),
    [policy, setPolicy] = useState(defaults),
    [selected, setSelected] = useState(""),
    [file, setFile] = useState(""),
    [preview, setPreview] = useState<SavedLogPreview>(),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [deleting, setDeleting] = useState<string>(),
    [filter, setFilter] = useState("");
  const load = useCallback(async () => {
    const next = await api.GetLogStorage();
    setStorage(next);
    setPolicy(next.policy);
    setSelected((old) =>
      next.sessions.some((s) => s.id === old)
        ? old
        : next.sessions[0]?.id || "",
    );
  }, []);
  useEffect(() => {
    let mounted = true;
    void load().catch((e) => {
      if (mounted) setError(String(e));
    });
    return () => {
      mounted = false;
    };
  }, [load]);
  const session = storage?.sessions.find((s) => s.id === selected);
  useEffect(() => {
    setFile(
      session?.files.find((f) => f.name === "crash.log")?.name ||
        session?.files.filter((f) => f.name.endsWith(".jsonl")).at(-1)?.name ||
        session?.files[0]?.name ||
        "",
    );
  }, [selected, session?.files.map((f) => f.name).join("|")]);
  useEffect(() => {
    let active = true;
    setPreview(undefined);
    if (selected && file)
      void api
        .ReadSavedLog(selected, file)
        .then((p) => {
          if (active) setPreview(p);
        })
        .catch((e) => {
          if (active) setError(String(e));
        });
    return () => {
      active = false;
    };
  }, [selected, file, storage]);
  const run = async (action: () => Promise<unknown>, reload = true) => {
    setBusy(true);
    setError("");
    try {
      await action();
      if (reload) await load();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="diagnostic-settings">
      <p>
        Logs record app activity, memory use and crash details. Old logs are deleted automatically. Logs from running instances are kept.
      </p>
      {error && (
        <FeedbackFlyout title="Diagnostics & logs">
          {error}
        </FeedbackFlyout>
      )}
      {storage?.warning && <FeedbackFlyout title="Log storage" tone="warning">{storage.warning}</FeedbackFlyout>}
      <form
        className="diagnostic-policy"
        onSubmit={(e) => {
          e.preventDefault();
          void run(() => api.SetLogPolicy(policy));
        }}
      >
        <label>
          Maximum log storage (MiB)
          <input
            aria-label="Maximum log storage (MiB)"
            type="number"
            min={32}
            max={1024}
            step={1}
            required
            value={policy.maxBytes / 1048576}
            onChange={(e) =>
              setPolicy((p) => ({
                ...p,
                maxBytes: Number(e.target.value) * 1048576,
              }))
            }
          />
        </label>
        <label>
          Keep logs for (days)
          <input
            type="number"
            min={1}
            max={90}
            required
            value={policy.maxAgeDays}
            onChange={(e) =>
              setPolicy((p) => ({ ...p, maxAgeDays: Number(e.target.value) }))
            }
          />
        </label>
        <label>
          Maximum saved sessions
          <input
            type="number"
            min={1}
            max={100}
            required
            value={policy.maxSessions}
            onChange={(e) =>
              setPolicy((p) => ({ ...p, maxSessions: Number(e.target.value) }))
            }
          />
        </label>
        <label>
          Log detail
          <select
            value={policy.level}
            onChange={(e) =>
              setPolicy((p) => ({ ...p, level: e.target.value }))
            }
          >
            <option value="info">Standard</option>
            <option value="debug">Detailed</option>
            <option value="trace">Trace (temporary troubleshooting)</option>
          </select>
        </label>
        <div className="diagnostic-actions">
          <button
            className="studio-button primary"
            disabled={busy}
            type="submit"
          >
            Save log settings
          </button>
          <button
            className="studio-button"
            type="button"
            disabled={busy}
            onClick={() => setPolicy(defaults)}
          >
            Restore log defaults
          </button>
        </div>
      </form>
      <p className="preference-note">
        3 rotating event files of up to 4 MiB and a crash report of up to 2 MiB
        per session. Password and token fields are redacted. Packet payloads and
        resource-pack keys are not recorded by this system. Trace captures more
        detail; it does not remove storage limits.
      </p>
      <div className="diagnostic-actions">
        <button
          className="studio-button"
          disabled={busy}
          onClick={() => void run(load, false)}
        >
          <RefreshCw size={14} />
          Refresh logs
        </button>
        <button
          className="studio-button"
          disabled={busy || !storage}
          onClick={() => void run(() => api.OpenLogFolder(), false)}
        >
          <FolderOpen size={14} />
          Open log folder
        </button>
        <button
          className="studio-button"
          disabled={busy || !storage}
          onClick={() => void run(() => api.CreateDiagnosticSnapshot())}
        >
          <Activity size={14} />
          Record diagnostic report
        </button>
        <button
          className="studio-button"
          disabled={busy || !storage}
          onClick={() => void run(() => api.CleanupSavedLogs())}
        >
          Clean up now
        </button>
        <button
          className="studio-button"
          disabled={busy || !storage?.sessions.some((s) => !s.active)}
          onClick={() => setDeleting("all")}
        >
          <Trash2 size={14} />
          Delete logs from previous launches
        </button>
      </div>
      {storage && (
        <>
          <p className="diagnostic-summary">
            {size(storage.bytes)} used of {size(storage.policy.maxBytes)} ·{" "}
            {storage.sessions.length} sessions · {size(storage.reservedBytes)}{" "}
            reserved for active sessions
          </p>
          <code className="diagnostic-directory">{storage.directory}</code>
        </>
      )}
      <label className="diagnostic-filter">
        Filter saved sessions
        <input
          type="search"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Session, version or status"
        />
      </label>
      <div className="diagnostic-browser">
        <div className="diagnostic-sessions" aria-label="Saved log sessions">
          {storage?.sessions
            .filter((s) =>
              `${s.id} ${s.version} ${s.status} ${s.role}`
                .toLowerCase()
                .includes(filter.toLowerCase()),
            )
            .map((s) => (
              <button
                className="diagnostic-session"
                key={s.id}
                aria-pressed={selected === s.id}
                onClick={() => setSelected(s.id)}
              >
                <strong>{new Date(s.started).toLocaleString()}</strong>
                <span>
                  {s.status} · {s.role || "main"} · {size(s.bytes)}
                </span>
                <small>{s.id}</small>
              </button>
            ))}
          {storage && !storage.sessions.length && <p>No saved sessions.</p>}
        </div>
        <div className="diagnostic-detail">
          {session ? (
            <>
              <p>
                {session.version} · PID {session.pid} ·{" "}
                {session.active ? "Recording now" : session.status}
              </p>
              <label>
                Log file
                <select value={file} onChange={(e) => setFile(e.target.value)}>
                  {session.files.map((f) => (
                    <option key={f.name} value={f.name}>
                      {f.name} ({size(f.bytes)})
                    </option>
                  ))}
                </select>
              </label>
              <div className="diagnostic-actions">
                <button
                  className="studio-button"
                  disabled={busy || !file}
                  onClick={() =>
                    void run(() => api.OpenSavedLog(selected, file), false)
                  }
                >
                  <FileText size={14} />
                  Open in editor
                </button>
                <button
                  className="studio-button"
                  disabled={busy || session.active}
                  onClick={() => setDeleting(selected)}
                >
                  Delete session
                </button>
              </div>
              {preview?.truncated && (
                <p role="status">
                  Showing the last 128 KiB. Open in editor for the full file.
                </p>
              )}
              <pre aria-label="Log preview" tabIndex={0}>
                {preview?.text || "Loading log…"}
              </pre>
            </>
          ) : (
            <p>Select a launch to view its logs.</p>
          )}
        </div>
      </div>
      <Dialog
        open={deleting !== undefined}
        onOpenChange={(open) => {
          if (!open) setDeleting(undefined);
        }}
      >
        <DialogContent resizable={false} tone="warning">
          <DialogHeader>
            <DialogTitle>Delete saved logs?</DialogTitle>
            <DialogDescription>
              {deleting === "all"
                ? "Delete all closed log sessions? Active sessions are kept."
                : "Delete this saved session and its crash report?"}{" "}
              This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <button
              className="studio-button"
              onClick={() => setDeleting(undefined)}
            >
              Cancel
            </button>
            <button
              className="studio-button primary"
              disabled={busy}
              onClick={() => {
                const id = deleting;
                setDeleting(undefined);
                void run(() =>
                  id === "all"
                    ? api.DeleteClosedLogSessions()
                    : api.DeleteLogSession(id!),
                );
              }}
            >
              Confirm deletion
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

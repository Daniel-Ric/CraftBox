import { useEffect, useState } from "react";
import { ExternalLink } from "lucide-react";
import { api, type CrashReportSession, type StartupRecovery } from "./bridge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { createCrashReport, crashDescriptionLimit } from "./crash-report";
import { useLocale } from "./i18n";
import "./crash-recovery.css";

export default function CrashReportDialog({
  ids,
  recovery,
  onClose,
}: {
  ids: string[];
  recovery?: StartupRecovery;
  onClose: () => void;
}) {
  useLocale();
  const [review, setReview] = useState(!recovery);
  const [sessions, setSessions] = useState<CrashReportSession[]>();
  const [includeEvents, setIncludeEvents] = useState(true);
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);
  useEffect(() => {
    if (!review) return;
    let current = true;
    void api
      .GetCrashReport(ids)
      .then((value) => {
        if (current) setSessions(value);
      })
      .catch(() => {
        if (current) setError("The crash report could not be loaded.");
      });
    return () => {
      current = false;
    };
  }, [review, ids, loadAttempt]);

  const finish = (disposition: "later" | "dismissed") => {
    if (!recovery || !ids.length) {
      onClose();
      return;
    }
    void run(async () => {
      await api.SetCrashNotice(ids, disposition);
      onClose();
    }, "The crash notice could not be saved.");
  };

  async function run(action: () => Promise<void>, failure: string) {
    if (busy) return;
    setBusy(true);
    setError("");
    setStatus("");
    try {
      await action();
    } catch {
      setError(failure);
    } finally {
      setBusy(false);
    }
  }

  const report = sessions?.length
    ? createCrashReport(sessions, description, includeEvents)
    : undefined;
  const title = review
    ? "Review crash report"
    : !ids.length
      ? "CraftBox had trouble starting"
      : recovery?.mapOnly
        ? "CraftBox map window closed unexpectedly"
        : recovery?.confirmed
          ? "CraftBox crashed"
          : "CraftBox closed unexpectedly";

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) finish("later");
      }}
    >
      <DialogContent
        className="crash-recovery-dialog"
        resizable={false}
        showCloseButton={false}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {review
              ? "Review the details before opening a public issue on GitHub. You can also save the report as a ZIP."
              : ids.length
                ? "A report from the previous session is available."
                : "CraftBox stopped unexpectedly during two consecutive starts."}
          </DialogDescription>
        </DialogHeader>
        {!review && ids.length > 1 && !recovery?.mapOnly && (
          <p>The report also includes the separate map window.</p>
        )}
        {!review && recovery?.safeStart && (
          <p>
            CraftBox opened the Start screen without restoring the previous
            workspace.
          </p>
        )}
        {review && report && (
          <div className="crash-report-fields">
            <label>
              What were you doing? (optional)
              <textarea
                rows={3}
                maxLength={crashDescriptionLimit}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </label>
            {sessions?.some((session) => session.events) && (
              <label className="crash-report-option">
                <input
                  type="checkbox"
                  checked={includeEvents}
                  onChange={(event) => setIncludeEvents(event.target.checked)}
                />
                Include diagnostic events
              </label>
            )}
            <div className="crash-report-preview">
              <strong>GitHub issue preview</strong>
              <pre tabIndex={0} translate="no">
                {report.title}
                {"\n\n"}
                {report.issueBody}
              </pre>
            </div>
            {report.fullBody !== report.issueBody && (
              <details className="crash-report-preview">
                <summary>Full ZIP report</summary>
                <pre tabIndex={0} translate="no">
                  {report.fullBody}
                </pre>
              </details>
            )}
          </div>
        )}
        {review && !report && !error && (
          <p role="status">Loading crash report…</p>
        )}
        {error && <p role="alert">{error}</p>}
        {status && <p role="status">{status}</p>}
        <DialogFooter className="crash-recovery-actions">
          {recovery && ids.length > 0 && (
            <Button
              variant="ghost"
              disabled={busy}
              onClick={() => finish("dismissed")}
            >
              Dismiss crash notice
            </Button>
          )}
          <Button
            variant="outline"
            disabled={busy}
            onClick={() => finish("later")}
          >
            {!recovery
              ? "Close"
              : recovery.safeStart
                ? "Continue without restoring"
                : "Review later"}
          </Button>
          {error === "The crash notice could not be saved." && (
            <Button variant="ghost" onClick={onClose}>
              Close without saving the notice
            </Button>
          )}
          {!review && ids.length > 0 && (
            <Button
              disabled={busy}
              onClick={() => {
                setError("");
                setReview(true);
              }}
            >
              Review report
            </Button>
          )}
          {review && !report && error && (
            <Button
              onClick={() => {
                setError("");
                setLoadAttempt((value) => value + 1);
              }}
            >
              Retry loading the crash report
            </Button>
          )}
          {review && report && (
            <>
              <Button
                variant="outline"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    if (
                      await api.ExportCrashReport(
                        `# ${report.title}\n\n${report.fullBody}\n`,
                      )
                    )
                      setStatus("Crash report ZIP saved.");
                  }, "The crash report ZIP could not be saved.")
                }
              >
                Save ZIP
              </Button>
              <Button
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await api.OpenCrashIssue(report.url);
                    setStatus("Submit the issue in GitHub to publish it.");
                  }, "GitHub could not be opened. You can still save the report as a ZIP.")
                }
              >
                Open GitHub issue <ExternalLink size={14} />
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

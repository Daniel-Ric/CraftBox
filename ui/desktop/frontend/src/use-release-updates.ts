import { useCallback, useEffect, useRef, useState } from "react";
import { api, type ReleaseCheck } from "./bridge";
import { useWorkspacePreferences } from "./workspace-preferences";
import { notify } from "./StudioFlyouts";
const lastKey = "craftbox-update-last-check";
export function useReleaseUpdates(ready: boolean) {
  const prefs = useWorkspacePreferences();
  const [info, setInfo] = useState<ReleaseCheck>(),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const pending = useRef<string | undefined>(undefined);
  const sequence = useRef(0);
  const announced = useRef("");
  useEffect(() => {
    if (
      info?.available &&
      info.release &&
      info.channel === prefs.updateChannel &&
      announced.current !== info.release.version
    ) {
      const release = info.release;
      announced.current = release.version;
      notify({
        id: "craftbox-update",
        tone: "info",
        title: "CraftBox update available",
        description: "Version " + release.version + " is available on GitHub.",
        action: {
          label: "View release",
          onClick: () => api.OpenURL(release.releaseURL),
        },
      });
    }
  }, [info, prefs.updateChannel]);
  const check = useCallback(async () => {
    if (pending.current === prefs.updateChannel) return;
    const request = ++sequence.current;
    pending.current = prefs.updateChannel;
    setBusy(true);
    setError("");
    try {
      const result = await api.CheckUpdateDetails(prefs.updateChannel);
      if (request !== sequence.current) return;
      setInfo(result);
      try {
        localStorage.setItem(
          lastKey,
          JSON.stringify({ channel: prefs.updateChannel, time: Date.now() }),
        );
      } catch {}
    } catch (e) {
      if (request !== sequence.current) return;
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (request === sequence.current) {
        pending.current = undefined;
        setBusy(false);
      }
    }
  }, [prefs.updateChannel]);
  useEffect(() => {
    sequence.current++;
    pending.current = undefined;
    setBusy(false);
    setInfo(undefined);
    setError("");
  }, [prefs.updateChannel]);
  useEffect(() => {
    if (!ready || !prefs.automaticUpdates) return;
    let nextAttempt = 0;
    const due = () => {
      if (Date.now() < nextAttempt) return;
      try {
        const last = JSON.parse(localStorage.getItem(lastKey) || "null");
        if (
          last?.channel === prefs.updateChannel &&
          Number.isFinite(last.time) &&
          Date.now() - last.time < prefs.updateIntervalHours * 3600000
        )
          return;
      } catch {}
      nextAttempt = Date.now() + 15 * 60000;
      void check();
    };
    due();
    const timer = window.setInterval(due, 60000);
    return () => window.clearInterval(timer);
  }, [
    ready,
    prefs.automaticUpdates,
    prefs.updateIntervalHours,
    prefs.updateChannel,
    check,
  ]);
  return {
    info: info?.channel === prefs.updateChannel ? info : undefined,
    error,
    busy,
    check,
  };
}

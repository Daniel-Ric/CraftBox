import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./bridge";
import { exportTasks, type ExportTaskHandle } from "./export-tasks";
import { notify } from "./StudioFlyouts";

export function useUpdateInstaller(progress: number, stage: string) {
  const [busy, setBusy] = useState(false);
  const [ready, setReady] = useState(false);
  const [promptOpen, setPromptOpen] = useState(false);
  const task = useRef<ExportTaskHandle | undefined>(undefined);
  useEffect(() => {
    if (!busy) return;
    if (stage === "installing")
      task.current?.update("Installing update. CraftBox will restart…");
    else if (progress >= 100) task.current?.update("Verifying download…");
    else if (stage === "downloading")
      task.current?.update(
        "Downloading update…",
        progress >= 0 ? progress : undefined,
        100,
      );
  }, [progress, stage, busy]);
  const install = useCallback(async () => {
    if (task.current) return;
    const current = exportTasks.begin("Update CraftBox");
    task.current = current;
    current.update("Connecting to GitHub…");
    setBusy(true);
    try {
      await api.InstallUpdate();
      current.complete();
      current.dismiss();
      setReady(true);
      setPromptOpen(true);
      notify({
        id: "craftbox-update-ready",
        tone: "info",
        title: "Update installed",
        description: "Restart CraftBox when you are ready.",
        action: {
          label: "Restart options",
          onClick: () => setPromptOpen(true),
        },
      });
    } catch (error) {
      current.fail(error);
      throw error;
    } finally {
      task.current = undefined;
      setBusy(false);
    }
  }, []);
  return { busy, ready, promptOpen, setPromptOpen, install };
}

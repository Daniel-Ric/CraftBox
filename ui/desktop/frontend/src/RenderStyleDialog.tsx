import { useEffect, useRef, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  loadRenderStyles,
  saveRenderStyle,
  deleteRenderStyle,
  encodeRenderStyle,
  decodeRenderStyle,
  type RenderStyle,
} from "./render-style-library";
import { saveCinematic } from "./cinematic-export";
import type { RenderSettings } from "./render-settings";
export default function RenderStyleDialog({
  open,
  onOpenChange,
  settings,
  skyFile,
  onApply,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  settings: RenderSettings;
  skyFile?: File;
  onApply: (settings: RenderSettings, file?: File) => void;
}) {
  const [styles, setStyles] = useState<RenderStyle[]>([]),
    [selected, setSelected] = useState(""),
    [name, setName] = useState("My scene style"),
    [message, setMessage] = useState(""),
    [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (open) {
      setMessage("");
      setSelected("");
      void loadRenderStyles()
        .then(setStyles)
        .catch((e) => setMessage(String(e)));
    }
  }, [open]);
  const chosen = styles.find((s) => s.id === selected);
  const run = async (action: () => Promise<void>) => {
    setBusy(true);
    setMessage("");
    try {
      await action();
    } catch (e) {
      setMessage(String(e));
    } finally {
      setBusy(false);
    }
  };
  const current = async () => {
    let asset: RenderStyle["customSky"];
    if (settings.sky === "custom") {
      if (!skyFile)
        throw new Error(
          "Select the custom HDR skybox again so it can be saved with the preset.",
        );
      if (skyFile.size > 16 * 1024 * 1024)
        throw new Error("Custom HDR skyboxes up to 16 MiB are supported.");
      const bytes = new Uint8Array(await skyFile.arrayBuffer());
      let binary = "";
      for (let i = 0; i < bytes.length; i += 32768)
        binary += String.fromCharCode(...bytes.subarray(i, i + 32768));
      asset = { name: skyFile.name, data: btoa(binary) };
    }
    return decodeRenderStyle(encodeRenderStyle(name, settings, asset));
  };
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="render-style-window">
        <DialogHeader>
          <DialogTitle>Render presets</DialogTitle>
          <DialogDescription>
            Save your complete scene styling and apply it to other worlds.
          </DialogDescription>
        </DialogHeader>
        <div className="studio-dialog-body render-style-body">
          <nav aria-label="Saved render presets">
            <strong>My presets</strong>
            <button aria-pressed={!selected} onClick={() => setSelected("")}>
              Current scene style
            </button>
            {styles.map((s) => (
              <button
                key={s.id}
                aria-pressed={selected === s.id}
                onClick={() => setSelected(s.id)}
              >
                {s.name}
              </button>
            ))}
            {!styles.length && (
              <p>No saved presets yet.</p>
            )}
          </nav>
          <section>
            <h3>{chosen?.name || "Current scene style"}</h3>
            <p>
              Lighting, color, sky, quality, output format and all cloud and fog settings are applied together.
            </p>
            <label>
              Preset name
              <input
                aria-label="Preset name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={120}
              />
            </label>
            <button
              className="studio-button primary"
              disabled={busy || !name.trim()}
              onClick={() =>
                void run(async () => {
                  const s = await current();
                  await saveRenderStyle(s);
                  setStyles(await loadRenderStyles());
                  setSelected(s.id);
                  setMessage("Current scene style saved.");
                })
              }
            >
              Save current style
            </button>
            <dl>
              <dt>Sky</dt>
              <dd>{(chosen?.settings || settings).sky}</dd>
              <dt>Exposure</dt>
              <dd>{(chosen?.settings || settings).exposure}</dd>
              <dt>Clouds / Fog</dt>
              <dd>
                {(chosen?.settings || settings).atmosphere.clouds
                  ? "On"
                  : "Off"}{" "}
                / {(chosen?.settings || settings).atmosphere.fog ? "On" : "Off"}
              </dd>
            </dl>
            <div className="render-style-actions">
              <button
                className="studio-button"
                disabled={busy}
                onClick={() => input.current?.click()}
              >
                Import JSON …
              </button>
              <button
                className="studio-button"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    const s = chosen || (await current());
                    const path = await saveCinematic(
                      new Blob([JSON.stringify(s, null, 2)], {
                        type: "application/json",
                      }),
                      s.name + ".render-style",
                      "json",
                    );
                    setMessage(
                      path ? "Saved: " + path : "Export canceled.",
                    );
                  })
                }
              >
                Export JSON …
              </button>
              {chosen && (
                <button
                  className="studio-button"
                  disabled={busy}
                  onClick={() =>
                    void run(async () => {
                      await deleteRenderStyle(chosen.id);
                      setStyles(await loadRenderStyles());
                      setSelected("");
                    })
                  }
                >
                  Remove
                </button>
              )}
            </div>
            <input
              hidden
              ref={input}
              type="file"
              accept=".json"
              aria-label="Import render preset"
              onChange={(e) => {
                const file = e.target.files?.[0];
                e.target.value = "";
                if (file)
                  void run(async () => {
                    if (file.size > 48 * 1024 * 1024)
                      throw new Error("The preset exceeds 48 MiB.");
                    const s = decodeRenderStyle(await file.text());
                    await saveRenderStyle(s);
                    setStyles(await loadRenderStyles());
                    setSelected(s.id);
                    setMessage(
                      "Imported. Select “Apply” to use it for this world.",
                    );
                  });
              }}
            />
            {message && <p role="status">{message}</p>}
            <p>
              Camera position and selected world are preserved. Relative heights adapt to the target world's dimensions. Custom HDR files are included.
            </p>
          </section>
        </div>
        <DialogFooter>
          <button className="studio-button" onClick={() => onOpenChange(false)}>
            Close
          </button>
          <button
            className="studio-button primary"
            disabled={!chosen || busy}
            onClick={() => {
              if (!chosen) return;
              const asset = chosen.customSky;
              const file = asset
                ? new File(
                    [Uint8Array.from(atob(asset.data), (c) => c.charCodeAt(0))],
                    asset.name,
                  )
                : undefined;
              onApply(chosen.settings, file);
              onOpenChange(false);
            }}
          >
            Apply
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

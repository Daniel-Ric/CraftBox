import { useState } from "react";
import ShortcutKeys, { ShortcutBindingField } from "./ShortcutKeys";
import { Keyboard, RotateCcw, Search, Check } from "lucide-react";
import {
  defaultShortcutProfile,
  displayBinding,
  eventStroke,
  loadShortcutProfile,
  parseShortcutProfile,
  saveShortcutProfile,
  shortcutActions,
  shortcutScopeLabels,
  validateShortcutProfile,
  type ShortcutProfile,
} from "./shortcuts";
import "./shortcuts.css";

export default function ShortcutSettings() {
  const [saved, setSaved] = useState(loadShortcutProfile);
  const [draft, setDraft] = useState(saved);
  const [query, setQuery] = useState("");
  const [scope, setScope] = useState("all");
  const [modified, setModified] = useState(false);
  const [error, setError] = useState("");
  const [raw, setRaw] = useState<Record<string, string>>({});
  const [recording, setRecording] = useState<{
    id: string;
    strokes: string[];
  } | null>(null);
  const defaults = defaultShortcutProfile();
  const persist = (next: ShortcutProfile) => {
    setError("");
    if (validateShortcutProfile(next).length) return;
    try {
      if (JSON.stringify(next) !== JSON.stringify(saved))
        saveShortcutProfile(next);
      setSaved(next);
    } catch {
      setError(
        "Changes could not be saved. Check available storage and try again.",
      );
    }
  };
  const update = (next: ShortcutProfile) => {
    setDraft(next);
    setRaw({});
    persist(next);
  };
  const bind = (id: string, values: string[]) => {
    const next = { ...draft, bindings: { ...draft.bindings, [id]: values } };
    setDraft(next);
    setRaw((old) => {
      const value = { ...old };
      delete value[id];
      return value;
    });
    persist(next);
  };
  const errors = validateShortcutProfile(draft);
  const rows = shortcutActions.filter(
    (a) =>
      (scope === "all" || scope === a.scope) &&
      (!modified ||
        JSON.stringify(draft.bindings[a.id]) !==
          JSON.stringify(defaults.bindings[a.id])) &&
      `${a.label} ${shortcutScopeLabels[a.scope]} ${a.id} ${draft.bindings[a.id]?.map(displayBinding).join(" ")}`
        .toLowerCase()
        .includes(query.trim().toLowerCase()),
  );
  const finishRecording = (replace: boolean) => {
    if (!recording?.strokes.length) return;
    bind(recording.id, [
      ...(replace ? [] : draft.bindings[recording.id]),
      recording.strokes.join(" > "),
    ]);
    setRecording(null);
    document.getElementById(`record-${recording.id}`)?.focus();
  };
  const cancelRecording = () => {
    const id = recording?.id;
    setRecording(null);
    if (id) document.getElementById(`record-${id}`)?.focus();
  };
  return (
    <div className="shortcut-settings">
      <div className="shortcut-toolbar">
        <div className="shortcut-search">
          <Search size={14} aria-hidden="true" />
          <input
            type="search"
            aria-label="Search shortcuts"
            placeholder="Search actions or keys"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setRecording(null);
            }}
          />
        </div>
        <select
          aria-label="Shortcut context"
          value={scope}
          onChange={(e) => {
            setScope(e.target.value);
            setRecording(null);
          }}
        >
          <option value="all">All contexts</option>
          {[...new Set(shortcutActions.map((a) => a.scope))].map((s) => (
            <option key={s} value={s}>
              {shortcutScopeLabels[s]}
            </option>
          ))}
        </select>
        <label className="shortcut-modified">
          <input
            type="checkbox"
            checked={modified}
            onChange={(e) => {
              setModified(e.target.checked);
              setRecording(null);
            }}
          />{" "}
          Modified only
        </label>
      </div>
      <div className="shortcut-summary">
        <span>
          {rows.length === shortcutActions.length
            ? `${rows.length} actions`
            : `${rows.length} of ${shortcutActions.length} actions`}
        </span>
        <span
          role="status"
          className={
            error || errors.length ? "shortcut-unsaved" : "shortcut-saved"
          }
        >
          {error || errors.length ? (
            "Changes not saved"
          ) : (
            <>
              <Check size={12} aria-hidden="true" /> Saved automatically
            </>
          )}
        </span>
      </div>
      <details className="shortcut-advanced">
        <summary>Advanced &amp; profiles</summary>
        <div className="shortcut-advanced-body">
          <p>
            Record a combination beside an action, or edit its field. Clear a
            field to disable it. Ctrl also accepts Command on macOS.
          </p>
          <p>
            Use <kbd>Ctrl+KeyS</kbd> for a combination, <kbd>;</kbd> for
            alternatives and <kbd> &gt; </kbd> for sequences, for example{" "}
            <kbd>Ctrl+KeyG &gt; KeyF</kbd>. Camera movement uses one combination
            held down.
          </p>
          <label className="shortcut-timeout">
            Sequence timeout (ms)
            <input
              type="number"
              min={250}
              max={5000}
              step={50}
              value={draft.timeout}
              onChange={(e) =>
                update({ ...draft, timeout: Number(e.target.value) })
              }
            />
          </label>
          <div className="shortcut-profile-actions">
            <label className="studio-button">
              Import profile
              <input
                type="file"
                accept=".json,application/json"
                aria-label="Import shortcut profile"
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  e.target.value = "";
                  if (!file) return;
                  try {
                    if (file.size > 128 * 1024)
                      throw new Error("Profile is too large (maximum 128 KB).");
                    update(parseShortcutProfile(await file.text()));
                  } catch (err) {
                    setError(String(err));
                  }
                }}
              />
            </label>
            <button
              className="studio-button"
              disabled={errors.length > 0}
              onClick={() => {
                const url = URL.createObjectURL(
                  new Blob([JSON.stringify(draft, null, 2)], {
                    type: "application/json",
                  }),
                );
                const link = document.createElement("a");
                link.href = url;
                link.download = "craftbox-shortcuts.json";
                link.click();
                setTimeout(() => URL.revokeObjectURL(url), 1000);
              }}
            >
              Export profile
            </button>
            <button
              className="studio-button"
              onClick={() => update(defaultShortcutProfile())}
            >
              Restore all defaults
            </button>
          </div>
        </div>
      </details>
      {(error || errors.length > 0) && (
        <div role="alert" className="shortcut-errors" id="shortcut-errors">
          {error || errors.join("\n")}
          <div className="shortcut-error-actions">
            <button
              className="studio-button"
              onClick={() => {
                update(loadShortcutProfile());
                setRecording(null);
              }}
            >
              Revert unsaved changes
            </button>
            {error && !errors.length && (
              <button className="studio-button" onClick={() => persist(draft)}>
                Retry saving
              </button>
            )}
          </div>
        </div>
      )}
      <div className="shortcut-rows">
        {rows.map((a) => (
          <div
            className={`shortcut-row${recording?.id === a.id ? " is-recording" : ""}`}
            key={a.id}
          >
            <label htmlFor={`binding-${a.id}`}>
              <strong>{a.label}</strong>
              <span>{shortcutScopeLabels[a.scope]}</span>
            </label>
            <ShortcutBindingField
              id={`binding-${a.id}`}
              label={`${a.label} bindings`}
              invalid={errors.some(
                (error) =>
                  error.startsWith(a.label + ":") ||
                  error.startsWith(a.label + " conflicts"),
              )}
              value={raw[a.id] ?? draft.bindings[a.id].join("; ")}
              bindings={draft.bindings[a.id]}
              onChange={(value) => {
                bind(
                  a.id,
                  value
                    .split(";")
                    .map((s) => s.trim())
                    .filter(Boolean),
                );
                setRaw((old) => ({ ...old, [a.id]: value }));
              }}
            />
            <button
              className="studio-button shortcut-record-button"
              id={`record-${a.id}`}
              aria-label={`Record ${a.label}`}
              onClick={() => setRecording({ id: a.id, strokes: [] })}
            >
              <Keyboard size={14} aria-hidden="true" /> Record
            </button>
            <button
              className="studio-button shortcut-reset-button"
              aria-label={`Reset ${a.label}`}
              title="Restore default binding"
              onClick={() => bind(a.id, defaults.bindings[a.id])}
            >
              <RotateCcw size={14} aria-hidden="true" />
            </button>
            {recording?.id === a.id && (
              <div
                className="shortcut-recording"
                role="group"
                aria-label="Record shortcut"
                tabIndex={0}
                ref={(el) => {
                  if (el && !el.contains(document.activeElement)) {
                    el.focus({ preventScroll: true });
                    el.scrollIntoView?.({ block: "nearest" });
                  }
                }}
                onKeyDown={(e) => {
                  if (
                    e.target !== e.currentTarget ||
                    (e.code === "Tab" && !e.ctrlKey && !e.altKey && !e.metaKey)
                  )
                    return;
                  e.preventDefault();
                  e.stopPropagation();
                  if (
                    e.code === "Escape" &&
                    !e.ctrlKey &&
                    !e.altKey &&
                    !e.metaKey &&
                    !e.shiftKey
                  ) {
                    cancelRecording();
                    return;
                  }
                  if (e.repeat || e.nativeEvent.isComposing) return;
                  const stroke = eventStroke(e.nativeEvent);
                  if (!stroke) return;
                  const camera = recording.id.startsWith("camera.");
                  if (!camera && ["Shift", "Alt"].includes(stroke)) return;
                  setRecording({
                    ...recording,
                    strokes: camera
                      ? [stroke]
                      : [...recording.strokes, stroke].slice(0, 4),
                  });
                }}
              >
                <div className="shortcut-recording-prompt">
                  <strong>
                    {recording.strokes.length
                      ? "Recorded keys"
                      : "Press your keys"}
                  </strong>
                  <span>
                    Up to{" "}
                    {a.scope === "camera" ? "one combination" : "four steps"}.
                    Esc cancels.
                  </span>
                </div>
                <div className="shortcut-recorded-keys">
                  {recording.strokes.length ? (
                    <ShortcutKeys bindings={[recording.strokes.join(" > ")]} />
                  ) : (
                    "Waiting for keys…"
                  )}
                </div>
                <div className="shortcut-recording-actions">
                  <button
                    className="studio-button primary"
                    disabled={!recording.strokes.length}
                    onClick={() => finishRecording(true)}
                  >
                    Replace binding
                  </button>
                  <button
                    className="studio-button"
                    disabled={!recording.strokes.length}
                    onClick={() => finishRecording(false)}
                  >
                    Add alternative
                  </button>
                  <button className="studio-button" onClick={cancelRecording}>
                    Cancel
                  </button>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
      {!rows.length && (
        <div className="shortcut-empty">
          <Search size={22} aria-hidden="true" />
          <strong>No matching shortcuts</strong>
          <span>Try a different action, key or context.</span>
          <button
            className="studio-button"
            onClick={() => {
              setQuery("");
              setScope("all");
              setModified(false);
            }}
          >
            Clear filters
          </button>
        </div>
      )}
    </div>
  );
}

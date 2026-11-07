import { useId, useMemo, useState } from "react";
import { Copy, Image as ImageIcon, Maximize2, RotateCcw } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "./components/ui/dialog";
import {
  cleanFormText,
  formElements,
  formList,
  formObject,
  formRange,
  formattedFormText,
  initialFormValues,
  plainFormText,
  type FormObject,
  type PacketForm,
  type FormTextRun,
} from "./packet-form-model";
import "./packet-form-preview.css";
function FormText({ runs }: { runs: FormTextRun[] }) {
  return (
    <span className="pi-form-text">
      {runs.map((run, i) => (
        <span
          key={i}
          style={{
            color: run.color,
            fontWeight: run.bold ? 700 : undefined,
            fontStyle: run.italic ? "italic" : undefined,
          }}
          title={run.obfuscated ? "Obfuscated text in Minecraft" : undefined}
        >
          {run.text}
        </span>
      ))}
    </span>
  );
}
function FormDropdown({
  field,
  id,
  value,
  onChange,
}: {
  field: FormObject;
  id: string;
  value: unknown;
  onChange: (v: number) => void;
}) {
  const [query, setQuery] = useState("");
  const options = formList(field.options),
    label = plainFormText(field.text).slice(0, 160) || "dropdown";
  const matches = useMemo(() => {
    const rows: { index: number; label: string }[] = [];
    const q = query.toLowerCase();
    for (let i = 0; i < options.length; i++) {
      const full = plainFormText(options[i]);
      if (!q || full.toLowerCase().includes(q)) {
        rows.push({ index: i, label: full.slice(0, 240) });
        if (rows.length > 200) break;
      }
    }
    return rows;
  }, [options, query]);
  const rows = matches.slice(0, 200),
    selected =
      typeof value === "number" && value >= 0 && value < options.length
        ? value
        : undefined;
  return (
    <div className="pi-form-dropdown">
      {options.length > 200 && (
        <input
          type="search"
          aria-label={`Search ${label} options`}
          placeholder={`Search all ${options.length.toLocaleString()} options`}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      )}
      <select
        id={id}
        value={selected ?? ""}
        disabled={!options.length}
        onChange={(e) => onChange(Number(e.target.value))}
      >
        {!options.length && <option value="">No options</option>}
        {rows.map((row) => (
          <option key={row.index} value={row.index}>
            {row.label}
          </option>
        ))}
        {selected !== undefined &&
          !rows.some((row) => row.index === selected) && (
            <option value={selected}>
              {plainFormText(options[selected]).slice(0, 240)} (current
              selection)
            </option>
          )}
      </select>
      {matches.length > 200 && (
        <small>
          Showing the first 200 matches. Refine the search to find another
          option.
        </small>
      )}
      {query && !matches.length && (
        <small>No matching options. The current selection is unchanged.</small>
      )}
    </div>
  );
}
function FormSurface(p: {
  form: PacketForm;
  showGlyphs: boolean;
  values: unknown[];
  onValue: (i: number, v: unknown) => void;
  onResponse: (v: unknown) => void;
}) {
  const prefix = useId(),
    elements = useMemo(() => formElements(p.form), [p.form]),
    [limit, setLimit] = useState(100);
  let buttonIndex = 0;
  let textChars = 30000,
    textRuns = 1400;
  const text = (value: unknown) => {
    const runs = formattedFormText(
      value,
      p.showGlyphs,
      Math.min(6000, textChars),
      Math.min(256, textRuns),
    );
    for (const run of runs)
      if (!run.shortened) {
        textChars = Math.max(0, textChars - run.text.length);
        textRuns = Math.max(0, textRuns - 1);
      }
    return <FormText runs={runs} />;
  };
  const passive = (field: FormObject, key: number) =>
    field.type === "divider" ? (
      <hr key={key} />
    ) : field.type === "header" ? (
      <h3 key={key}>{text(field.text)}</h3>
    ) : field.type === "label" ? (
      <p key={key}>{text(field.text)}</p>
    ) : null;
  return (
    <div className="pi-form-sheet">
      <header>
        <h2>
          {plainFormText(p.form.data.title)
            ? text(p.form.data.title)
            : "Untitled form"}
        </h2>
      </header>
      <div className="pi-form-sheet-body">
        {p.form.type !== "custom_form" &&
          cleanFormText(p.form.data.content, p.showGlyphs) && (
            <p>{text(p.form.data.content)}</p>
          )}
        {p.form.type === "modal" ? (
          <div className="pi-form-message-buttons">
            <button type="button" onClick={() => p.onResponse(true)}>
              {text(p.form.data.button1)}
            </button>
            <button type="button" onClick={() => p.onResponse(false)}>
              {text(p.form.data.button2)}
            </button>
          </div>
        ) : (
          elements.slice(0, limit).map((field, i) => {
            if (
              ["label", "header", "divider"].includes(
                String(field.type).slice(0, 120),
              )
            )
              return passive(field, i);
            if (p.form.type === "form") {
              if (field.type && field.type !== "button")
                return (
                  <p className="pi-form-unsupported" key={i}>
                    Unsupported element: {String(field.type)}
                  </p>
                );
              const index = buttonIndex++,
                image = formObject(field.image),
                imagePath =
                  typeof image?.data === "string" ? image.data : undefined;
              return (
                <button
                  className="pi-form-action"
                  key={i}
                  type="button"
                  onClick={() => p.onResponse(index)}
                >
                  {imagePath && (
                    <span className="pi-form-image" title={imagePath}>
                      <ImageIcon size={20} />
                    </span>
                  )}
                  <span>
                    {plainFormText(field.text)
                      ? text(field.text)
                      : `Button ${index + 1}`}
                  </span>
                </button>
              );
            }
            const id = `${prefix}-${i}`,
              label = plainFormText(field.text) || `Control ${i + 1}`,
              value = p.values[i],
              tooltip = plainFormText(field.tooltip);
            let control;
            switch (field.type) {
              case "input":
                control = (
                  <input
                    id={id}
                    value={typeof value === "string" ? value : ""}
                    placeholder={plainFormText(field.placeholder)}
                    onChange={(e) => p.onValue(i, e.target.value)}
                  />
                );
                break;
              case "toggle":
                control = (
                  <input
                    id={id}
                    type="checkbox"
                    checked={value === true}
                    onChange={(e) => p.onValue(i, e.target.checked)}
                  />
                );
                break;
              case "dropdown": {
                control = (
                  <FormDropdown
                    field={field}
                    id={id}
                    value={value}
                    onChange={(v) => p.onValue(i, v)}
                  />
                );
                break;
              }
              case "slider": {
                const range = formRange(field);
                control = (
                  <>
                    <input
                      id={id}
                      type="range"
                      min={range.min}
                      max={range.max}
                      step={range.step}
                      value={typeof value === "number" ? value : range.value}
                      onChange={(e) => p.onValue(i, Number(e.target.value))}
                    />
                    <output htmlFor={id}>{String(value)}</output>
                  </>
                );
                break;
              }
              case "step_slider": {
                const options = formList(field.steps);
                control = (
                  <>
                    <input
                      id={id}
                      type="range"
                      min={0}
                      max={Math.max(0, options.length - 1)}
                      step={1}
                      disabled={!options.length}
                      value={typeof value === "number" ? value : 0}
                      onChange={(e) => p.onValue(i, Number(e.target.value))}
                    />
                    <output htmlFor={id}>
                      {plainFormText(
                        options[typeof value === "number" ? value : 0],
                      ) || "No options"}
                    </output>
                  </>
                );
                break;
              }
              default:
                return (
                  <p key={i} className="pi-form-unsupported">
                    Unsupported control:{" "}
                    {String(field.type ?? "unknown").slice(0, 120)}. Inspect its
                    JSON for details.
                  </p>
                );
            }
            return (
              <div
                className={`pi-form-control${field.type === "toggle" ? " is-toggle" : ""}`}
                key={i}
              >
                <label htmlFor={id}>
                  {plainFormText(field.text) ? text(field.text) : label}
                </label>
                {control}
                {tooltip && <small>{text(field.tooltip)}</small>}
              </div>
            );
          })
        )}
        {elements.length > limit && (
          <button type="button" onClick={() => setLimit((n) => n + 100)}>
            Show {Math.min(100, elements.length - limit)} more of{" "}
            {elements.length - limit} remaining elements
          </button>
        )}
        {p.form.type === "custom_form" && (
          <button
            type="button"
            className="pi-form-submit"
            onClick={() => p.onResponse(p.values)}
          >
            Preview response
          </button>
        )}
      </div>
    </div>
  );
}
export default function PacketFormPreview({ form }: { form: PacketForm }) {
  const [expanded, setExpanded] = useState(false),
    [showGlyphs, setShowGlyphs] = useState(false),
    [values, setValues] = useState(() => initialFormValues(form)),
    [response, setResponse] = useState<unknown>(),
    [copyState, setCopyState] = useState("");
  const hasGlyphs =
    /[\uE000-\uF8FF\u{F0000}-\u{FFFFD}\u{100000}-\u{10FFFD}]/u.test(
      JSON.stringify(form.data),
    );
  const update = (i: number, v: unknown) => {
    setValues((old) => old.map((x, n) => (n === i ? v : x)));
    setResponse(undefined);
  };
  const surface = () => (
    <FormSurface
      form={form}
      showGlyphs={showGlyphs}
      values={values}
      onValue={update}
      onResponse={setResponse}
    />
  );
  const reset = () => {
    setValues(initialFormValues(form));
    setResponse(undefined);
    setCopyState("");
  };
  const responseJSON = JSON.stringify(
    { FormID: form.id, ResponseData: response },
    null,
    2,
  );
  const result = () =>
    response !== undefined && (
      <details open className="pi-form-response">
        <summary>Local response JSON</summary>
        <pre>{responseJSON}</pre>
        <button
          type="button"
          onClick={() => {
            void navigator.clipboard
              .writeText(responseJSON)
              .then(() => setCopyState("Copied"))
              .catch(() => setCopyState("Clipboard unavailable"));
          }}
        >
          <Copy size={13} />
          Copy response
        </button>
        {copyState && <span role="status">{copyState}</span>}
      </details>
    );
  const options = () =>
    hasGlyphs && (
      <label className="pi-form-glyph-option">
        <input
          type="checkbox"
          checked={showGlyphs}
          onChange={(e) => setShowGlyphs(e.target.checked)}
        />
        Show server font glyphs
      </label>
    );
  return (
    <div className="pi-form-preview">
      <div className="pi-form-tools">
        <span>
          {form.type === "form"
            ? "Button menu"
            : form.type === "modal"
              ? "Confirmation dialog"
              : "Custom form"}
          {form.id !== undefined ? ` · Form ${String(form.id)}` : ""}
        </span>
        <button
          type="button"
          title="Reset preview"
          aria-label="Reset form preview"
          onClick={reset}
        >
          <RotateCcw size={14} />
        </button>
        <button
          type="button"
          title="Expand form preview"
          aria-label="Expand form preview"
          onClick={() => setExpanded(true)}
        >
          <Maximize2 size={14} />
        </button>
      </div>
      <p className="pi-form-preview-note">
        Interactive preview · no response is sent to the server.
      </p>
      {options()}
      {hasGlyphs && (
        <p className="pi-form-preview-note">
          Custom glyphs need the server's font. They are hidden by default.
        </p>
      )}
      {surface()}
      {result()}
      {formElements(form).some((f) => formObject(f.image)) && (
        <details className="pi-form-assets">
          <summary>Referenced images</summary>
          <p>Pack icons require the server resource pack.</p>
          {formElements(form)
            .filter((f) => formObject(f.image))
            .map((f, i) => (
              <code key={i}>{String(formObject(f.image)?.data ?? "")}</code>
            ))}
        </details>
      )}
      <Dialog open={expanded} onOpenChange={setExpanded}>
        <DialogContent className="pi-form-dialog">
          <DialogHeader>
            <DialogTitle>
              {plainFormText(form.data.title) || "Form preview"}
            </DialogTitle>
            <DialogDescription>
              Interactive local preview. No packets are sent.
            </DialogDescription>
          </DialogHeader>
          <div className="pi-form-dialog-body">
            <p className="pi-form-preview-note">
              Local preview · no response is sent to the server.
            </p>
            {options()}
            {surface()}
            {result()}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

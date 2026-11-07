import {
  useEffect,
  useLayoutEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { ArrowRight, Search, Terminal } from "lucide-react";
import StudioInput from "./StudioInput";
import CommandCaret from "./CommandCaret";
import { completeCommand, type CompletionContext } from "./craftbox-cli";
import "./craftbox-cli.css";

export default function ConsoleCommandInput({
  mode,
  history,
  onHistory,
  onRun,
  onClear,
  onHelp,
  busy = false,
  context = {},
  autoFocus = false,
}: {
  mode: "console" | "cli";
  history: string[];
  onHistory: (history: string[]) => void;
  onRun: (value: string) => void;
  onClear?: () => void;
  onHelp?: () => void;
  busy?: boolean;
  context?: CompletionContext;
  autoFocus?: boolean;
}) {
  const [value, setValue] = useState(""),
    [focused, setFocused] = useState(false),
    [open, setOpen] = useState(true),
    [selected, setSelected] = useState(0),
    [historySearch, setHistorySearch] = useState(false),
    [atEnd, setAtEnd] = useState(true),
    [popupSpace, setPopupSpace] = useState(250);
  const input = useRef<HTMLInputElement>(null),
    root = useRef<HTMLDivElement>(null),
    list = useRef<HTMLDivElement>(null),
    cursor = useRef(-1),
    draft = useRef(""),
    searchDraft = useRef(""),
    explicitSelection = useRef(false);
  const id = useId();
  const suggestions = useMemo(
    () =>
      historySearch
        ? [...new Set(history)]
            .reverse()
            .filter((h) => h.toLowerCase().includes(value.toLowerCase()))
            .slice(0, 20)
            .map((h) => ({ value: h, label: h, description: "History" }))
        : completeCommand(value, mode, context),
    [value, mode, context, historySearch, history],
  );
  const expanded =
    focused &&
    open &&
    atEnd &&
    (value.length > 0 || historySearch) &&
    suggestions.length > 0;
  const index = Math.min(selected, Math.max(0, suggestions.length - 1));
  useLayoutEffect(() => {
    if (!expanded || !root.current) return;
    const pane = root.current.closest<HTMLElement>(".console-pane");
    const measure = () => {
      if (!root.current) return;
      const bounds = root.current.getBoundingClientRect(),
        scale = bounds.width / root.current.offsetWidth || 1;
      setPopupSpace(
        Math.max(
          24,
          Math.min(
            250,
            (bounds.top - (pane?.getBoundingClientRect().top ?? 0)) / scale - 8,
          ),
        ),
      );
    };
    measure();
    const observer =
      typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(measure)
        : undefined;
    if (pane) observer?.observe(pane);
    window.addEventListener("resize", measure);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", measure);
    };
  }, [expanded]);
  useEffect(() => {
    if (expanded)
      list.current
        ?.querySelector('[aria-selected="true"]')
        ?.scrollIntoView?.({ block: "nearest" });
  }, [index, expanded]);
  function accept() {
    const next = suggestions[index];
    if (!next) return;
    setValue(next.value);
    setHistorySearch(false);
    setOpen(false);
    cursor.current = -1;
    input.current?.focus();
    requestAnimationFrame(() =>
      input.current?.setSelectionRange(next.value.length, next.value.length),
    );
  }
  function run() {
    if (busy || !value.trim()) return;
    const raw = value.trim();
    onHistory(
      [
        ...history.filter(
          (_, i) => i !== history.length - 1 || history[i] !== raw,
        ),
        raw,
      ].slice(-100),
    );
    setValue("");
    setHistorySearch(false);
    setOpen(false);
    cursor.current = -1;
    draft.current = "";
    onRun(raw);
    input.current?.focus();
  }
  return (
    <div className="console-command-entry" ref={root}>
      <CommandCaret input={input} root={root} focused={focused} value={value}/>
      {expanded && (
        <div className="command-completions">
          <div className="command-completions-caption" hidden={popupSpace < 70}>
            {historySearch ? "Search command history" : "Suggestions"}
            <span>↑ ↓ select · Tab insert · Enter run</span>
          </div>
          <div
            ref={list}
            style={{
              maxHeight: Math.max(
                22,
                Math.min(210, popupSpace - (popupSpace < 70 ? 2 : 32)),
              ),
            }}
            role="listbox"
            id={id}
            aria-label={
              historySearch ? "Command history" : "Command suggestions"
            }
          >
            {suggestions.map((s, i) => (
              <button
                type="button"
                tabIndex={-1}
                role="option"
                aria-selected={index === i}
                id={`${id}-${i}`}
                key={s.value + "-" + i}
                onMouseDown={(e) => e.preventDefault()}
                onMouseEnter={() => setSelected(i)}
                onClick={() => {
                  const next = suggestions[i];
                  setValue(next.value);
                  setHistorySearch(false);
                  setOpen(false);
                  cursor.current = -1;
                  input.current?.focus();
                }}
              >
                <code>{s.label}</code>
                <span>{s.description}</span>
              </button>
            ))}
          </div>
        </div>
      )}
      <form
        className="console-command-bar"
        onSubmit={(e) => {
          e.preventDefault();
          run();
        }}
      >
        {historySearch ? <Search size={13} /> : <Terminal size={13} />}
        {mode === "cli" && <span className="command-prompt">craftbox ›</span>}
        <StudioInput
          ref={input}
          autoFocus={autoFocus}
          role="combobox"
          aria-label={
            mode === "cli" ? "CraftBox CLI command" : "CraftBox console command"
          }
          aria-autocomplete="list"
          aria-expanded={expanded}
          aria-controls={expanded ? id : undefined}
          aria-activedescendant={expanded ? `${id}-${index}` : undefined}
          autoComplete="off"
          spellCheck={false}
          placeholder={
            historySearch
              ? "Search history…"
              : mode === "cli"
                ? "Type a command · help"
                : "CraftBox command · /help · /cli"
          }
          value={value}
          maxLength={4096}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onSelect={(e) =>
            setAtEnd(
              e.currentTarget.selectionEnd === e.currentTarget.value.length,
            )
          }
          onChange={(e) => {
            setValue(e.target.value);
            setAtEnd(e.target.selectionEnd === e.target.value.length);
            setSelected(0);
            setOpen(true);
            cursor.current = -1;
            explicitSelection.current = false;
          }}
          onKeyDown={(e) => {
            if (e.nativeEvent.isComposing) return;
            if (e.ctrlKey && e.key.toLowerCase() === "r") {
              e.preventDefault();
              searchDraft.current = value;
              setHistorySearch(true);
              setOpen(true);
              setSelected(0);
              return;
            }
            if (e.ctrlKey && e.key.toLowerCase() === "l") {
              e.preventDefault();
              onClear?.();
              return;
            }
            if (
              e.ctrlKey &&
              e.key.toLowerCase() === "c" &&
              e.currentTarget.selectionStart === e.currentTarget.selectionEnd
            ) {
              e.preventDefault();
              setValue("");
              setOpen(false);
              setHistorySearch(false);
              cursor.current = -1;
              return;
            }
            if (e.key === "Escape") {
              e.preventDefault();
              if (historySearch) setValue(searchDraft.current);
              setHistorySearch(false);
              setOpen(false);
              return;
            }
            if (
              expanded &&
              (e.key === "Tab" ||
                (e.key === "Enter" &&
                  (historySearch || explicitSelection.current)))
            ) {
              e.preventDefault();
              explicitSelection.current = false;
              accept();
              return;
            }
            if (expanded && (e.key === "ArrowDown" || e.key === "ArrowUp")) {
              e.preventDefault();
              explicitSelection.current = true;
              setSelected(
                (index +
                  (e.key === "ArrowDown" ? 1 : -1) +
                  suggestions.length) %
                  suggestions.length,
              );
              return;
            }
            if (e.key === "ArrowUp" || e.key === "ArrowDown") {
              e.preventDefault();
              if (!history.length) return;
              if (cursor.current === -1) draft.current = value;
              cursor.current =
                e.key === "ArrowUp"
                  ? Math.min(history.length - 1, cursor.current + 1)
                  : Math.max(-1, cursor.current - 1);
              setValue(
                cursor.current < 0
                  ? draft.current
                  : history[history.length - 1 - cursor.current],
              );
              setOpen(false);
              return;
            }
          }}
        />
        <button
          type="submit"
          aria-label="Run command"
          disabled={busy || !value.trim()}
        >
          <ArrowRight size={13} />
          {busy ? "Running…" : "Run"}
        </button>
        {onHelp && (
          <button
            type="button"
            aria-label="Command help"
            disabled={busy}
            onClick={onHelp}
          >
            {mode === "cli" ? "help" : "/help"}
          </button>
        )}
      </form>
      <div className="command-shortcuts">
        <span>
          {historySearch
            ? `${suggestions.length} history matches`
            : busy
              ? "Waiting for command to finish"
              : "Tab autocomplete · ↑ ↓ history · Ctrl+R search"}
        </span>
        <span>Ctrl+L clear</span>
      </div>
    </div>
  );
}

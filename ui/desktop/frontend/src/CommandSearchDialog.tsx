import { useEffect, useId, useRef, useState } from "react";
import { Search, X, ArrowUp, ArrowDown, CornerDownLeft } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { findCommands, type CommandAction } from "./command-search";

export default function CommandSearchDialog({
  open,
  onOpenChange,
  commands,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  commands: CommandAction[];
}) {
  const [query, setQuery] = useState("");
  const [showUnavailable, setShowUnavailable] = useState(false);
  const [selected, setSelected] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);
  const id = useId();
  const results = findCommands(commands, query, showUnavailable);
  const available = results.filter((command) => command.enabled);
  const active =
    available.find((command) => command.label === selected) ?? available[0];
  const activeIndex = results.indexOf(active);
  useEffect(() => {
    if (open) {
      setQuery("");
      setSelected("");
      setShowUnavailable(false);
    }
  }, [open]);
  useEffect(() => {
    list.current
      ?.querySelector('[aria-selected="true"]')
      ?.scrollIntoView?.({ block: "nearest" });
  }, [active?.label, query, showUnavailable]);
  function run(command: CommandAction) {
    if (!command.enabled) return;
    onOpenChange(false);
    command.run();
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="command-palette" resizable={false}>
        <DialogHeader>
          <DialogTitle>Find command</DialogTitle>
          <DialogDescription>
            Search actions and switch workspaces.
          </DialogDescription>
        </DialogHeader>
        <div className="command-palette-search">
          <Search size={17} aria-hidden="true" />
          <input
            ref={input}
            autoFocus
            role="combobox"
            aria-label="Search commands"
            aria-expanded={true}
            aria-controls={`${id}-results`}
            aria-autocomplete="list"
            aria-activedescendant={active ? `${id}-${activeIndex}` : undefined}
            placeholder="Search commands or workspaces…"
            value={query}
            autoComplete="off"
            spellCheck={false}
            onChange={(event) => {
              setQuery(event.target.value);
              setSelected("");
            }}
            onKeyDown={(event) => {
              if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
                if (
                  (event.key === "Home" || event.key === "End") &&
                  !event.ctrlKey &&
                  !event.metaKey
                )
                  return;
                event.preventDefault();
                const at = available.indexOf(active);
                const next =
                  event.key === "Home"
                    ? 0
                    : event.key === "End"
                      ? available.length - 1
                      : (at +
                          (event.key === "ArrowDown" ? 1 : -1) +
                          available.length) %
                        available.length;
                setSelected(available[next]?.label ?? "");
              } else if (event.key === "Enter") {
                event.preventDefault();
                if (active) run(active);
              }
            }}
          />
          {query && (
            <button
              aria-label="Clear command search"
              onClick={() => {
                setQuery("");
                setSelected("");
                input.current?.focus();
              }}
            >
              <X size={14} />
            </button>
          )}
        </div>
        <div
          className="command-palette-results"
          ref={list}
          id={`${id}-results`}
          role="listbox"
          aria-label="Commands"
        >
          {Array.from(new Set(results.map((command) => command.group))).map(
            (group) => (
              <div role="group" aria-label={group} key={group}>
                <div className="command-palette-group" aria-hidden="true">
                  {group}
                </div>
                {results
                  .filter((command) => command.group === group)
                  .map((command) => {
                    const index = results.indexOf(command);
                    return (
                      <div
                        key={command.label}
                        id={`${id}-${index}`}
                        role="option"
                        aria-selected={active === command}
                        aria-disabled={!command.enabled}
                        className="command-palette-option"
                        onMouseMove={() => {
                          if (command.enabled) setSelected(command.label);
                        }}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => run(command)}
                      >
                        <span>
                          {command.label}
                          {!command.enabled && (
                            <small>Unavailable in this workspace</small>
                          )}
                        </span>
                        {command.shortcut && <kbd>{command.shortcut}</kbd>}
                      </div>
                    );
                  })}
              </div>
            ),
          )}
          {!results.length && (
            <div className="command-palette-empty">
              <Search size={24} />
              <strong>No matching commands</strong>
              <span>Try a workspace name, action or export format.</span>
            </div>
          )}
        </div>
        <footer className="command-palette-footer">
          <span>
            <ArrowUp size={12} />
            <ArrowDown size={12} /> Navigate <CornerDownLeft size={12} /> Run{" "}
            <kbd>Esc</kbd> Close
          </span>
          <label>
            <input
              type="checkbox"
              checked={showUnavailable}
              onChange={(event) => setShowUnavailable(event.target.checked)}
            />{" "}
            Show unavailable
          </label>
        </footer>
      </DialogContent>
    </Dialog>
  );
}

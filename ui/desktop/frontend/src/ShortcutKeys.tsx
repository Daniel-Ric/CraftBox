import { Fragment, useState } from "react";
import {
  Keyboard,
  Calculator,
  ArrowBigUp,
  Option,
  Command,
  CornerDownLeft,
  CornerUpLeft,
  ArrowRightToLine,
  Space,
  Delete,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  ArrowDown,
  House,
  ChevronRight,
  ChevronsUp,
  ChevronsDown,
  type LucideIcon,
} from "lucide-react";
import { displayBinding, validStroke } from "./shortcuts";
import "./shortcut-keys.css";

const namedKeys: Record<string, { label: string; icon: LucideIcon }> = {
  Shift: { label: "Shift", icon: ArrowBigUp },
  Alt: { label: "Alt", icon: Option },
  Meta: { label: "Cmd", icon: Command },
  Enter: { label: "Enter", icon: CornerDownLeft },
  NumpadEnter: { label: "Enter", icon: Calculator },
  Escape: { label: "Esc", icon: CornerUpLeft },
  Tab: { label: "Tab", icon: ArrowRightToLine },
  Space: { label: "Space", icon: Space },
  Backspace: { label: "Backspace", icon: Delete },
  Delete: { label: "Del", icon: Delete },
  ArrowLeft: { label: "Left", icon: ArrowLeft },
  ArrowRight: { label: "Right", icon: ArrowRight },
  ArrowUp: { label: "Up", icon: ArrowUp },
  ArrowDown: { label: "Down", icon: ArrowDown },
  Home: { label: "Home", icon: House },
  End: { label: "End", icon: ArrowRightToLine },
  PageUp: { label: "PgUp", icon: ChevronsUp },
  PageDown: { label: "PgDn", icon: ChevronsDown },
};
const punctuation: Record<string, string> = {
  Comma: ",",
  Period: ".",
  Slash: "/",
  Backslash: "\\",
  Semicolon: ";",
  Quote: "'",
  Backquote: "`",
  BracketLeft: "[",
  BracketRight: "]",
  Minus: "−",
  Equal: "=",
  NumpadAdd: "+",
  NumpadSubtract: "−",
  NumpadMultiply: "×",
  NumpadDivide: "÷",
  NumpadDecimal: ".",
};
function KeyToken({ code }: { code: string }) {
  if (code === "Ctrl") return <kbd className="shortcut-key">Ctrl</kbd>;
  const named = namedKeys[code];
  const Icon =
    named?.icon ?? (code.startsWith("Numpad") ? Calculator : Keyboard);
  const label =
    named?.label ??
    punctuation[code] ??
    code.replace(/^(Key|Digit|Numpad)(?=[A-Z0-9]$)/, "");
  return (
    <kbd className="shortcut-key">
      <Icon className="shortcut-key-icon" size={12} aria-hidden="true" />
      <span>{label}</span>
    </kbd>
  );
}
export default function ShortcutKeys({ bindings }: { bindings: string[] }) {
  if (!bindings.length)
    return <span className="shortcut-unassigned">Unassigned</span>;
  return (
    <span
      className="shortcut-keys"
      role="group"
      aria-label={bindings.map(displayBinding).join(" / ")}
    >
      {bindings.map((binding, index) => (
        <Fragment key={`${index}:${binding}`}>
          {index > 0 && (
            <span className="shortcut-key-divider" aria-hidden="true">
              /
            </span>
          )}
          <span className="shortcut-key-sequence" aria-hidden="true">
            {binding.split(" > ").map((stroke, step) => (
              <Fragment key={`${step}:${stroke}`}>
                {step > 0 && (
                  <ChevronRight className="shortcut-key-divider" size={11} />
                )}
                {validStroke(stroke) ? (
                  <span className="shortcut-key-chord">
                    {stroke.split("+").map((code, part) => (
                      <Fragment key={`${part}:${code}`}>
                        {part > 0 && (
                          <span className="shortcut-key-plus">+</span>
                        )}
                        <KeyToken code={code} />
                      </Fragment>
                    ))}
                  </span>
                ) : (
                  <span>{stroke}</span>
                )}
              </Fragment>
            ))}
          </span>
        </Fragment>
      ))}
    </span>
  );
}

export function ShortcutBindingField(p: {
  id: string;
  label: string;
  value: string;
  bindings: string[];
  invalid: boolean;
  onChange: (value: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  return (
    <div className={`shortcut-binding-field${editing ? "" : " is-preview"}`}>
      <input
        id={p.id}
        aria-label={p.label}
        aria-invalid={p.invalid || undefined}
        spellCheck={false}
        value={p.value}
        placeholder={editing ? "Unassigned" : ""}
        onFocus={() => setEditing(true)}
        onBlur={() => setEditing(false)}
        onChange={(e) => p.onChange(e.target.value)}
      />
      {!editing && (
        <span className="shortcut-binding-preview" aria-hidden="true">
          <ShortcutKeys bindings={p.bindings} />
        </span>
      )}
    </div>
  );
}

import { useRef } from "react";
import { Search, X, SlidersHorizontal, List, LayoutGrid } from "lucide-react";

export default function ViewSearch({
  label,
  inputId,
  placeholder,
  value,
  onChange,
  onSubmit,
  onFilters,
  disabled = false,
}: {
  label: string;
  inputId?: string;
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
  onSubmit?: () => void;
  onFilters?: () => void;
  disabled?: boolean;
}) {
  const input = useRef<HTMLInputElement>(null);
  return (
    <form
      className="view-heading-search"
      role="search"
      aria-label={label}
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit?.();
      }}
    >
      <Search size={15} aria-hidden="true" />
      <input
        id={inputId}
        ref={input}
        type="search"
        aria-label={label}
        placeholder={placeholder}
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
      />
      {value && (
        <button
          type="button"
          aria-label="Clear search"
          data-tooltip="Clear search"
          disabled={disabled}
          onClick={() => {
            onChange("");
            input.current?.focus();
          }}
        >
          <X size={14} />
        </button>
      )}
      {onFilters && (
        <button
          type="button"
          aria-label="Search filters"
          data-tooltip="Search filters"
          disabled={disabled}
          onClick={onFilters}
        >
          <SlidersHorizontal size={15} />
        </button>
      )}
    </form>
  );
}

export function ViewModeToggle({
  value,
  onChange,
  listLabel = "List view",
  gridLabel = "Grid view",
}: {
  value: string;
  onChange: (value: "list" | "grid") => void;
  listLabel?: string;
  gridLabel?: string;
}) {
  return (
    <div className="view-heading-modes" role="group" aria-label="Library view">
      <button
        type="button"
        className="toolbar-icon-action"
        aria-label={listLabel}
        data-tooltip={listLabel}
        aria-pressed={value === "list"}
        onClick={() => onChange("list")}
      >
        <List size={15} />
      </button>
      <button
        type="button"
        className="toolbar-icon-action"
        aria-label={gridLabel}
        data-tooltip={gridLabel}
        aria-pressed={value === "grid"}
        onClick={() => onChange("grid")}
      >
        <LayoutGrid size={15} />
      </button>
    </div>
  );
}

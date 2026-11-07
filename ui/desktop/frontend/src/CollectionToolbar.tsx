import { CheckCheck, ListChecks, Trash2, X, FolderOpen } from "lucide-react";
import { Button } from "./components/ui/button";
import "./studio-sidebars.css";
export default function CollectionToolbar({
  ids,
  selected,
  multiple,
  onMultiple,
  onSelected,
  onDelete,
  onOpen,
  busy = false,
  deleteDisabled = false,
}: {
  ids: string[];
  selected: string[];
  multiple: boolean;
  onMultiple: (value: boolean) => void;
  onSelected: (ids: string[]) => void;
  onDelete?: (ids: string[]) => void;
  onOpen?: (ids: string[]) => void;
  busy?: boolean;
  deleteDisabled?: boolean;
}) {
  return (
    <div
      className="collection-selection-toolbar"
      role="group"
      aria-label="Collection selection"
    >
      <Button
        variant={multiple ? "secondary" : "ghost"}
        size="sm"
        disabled={busy || !ids.length}
        aria-pressed={multiple}
        onClick={() => {
          onMultiple(!multiple);
          if (multiple) onSelected([]);
        }}
      >
        <ListChecks size={14} />
        {multiple ? "Done" : "Select"}
      </Button>
      {multiple && (
        <>
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={busy || !ids.length}
            aria-label="Select all"
            title="Select all"
            onClick={() => onSelected([...new Set([...selected, ...ids])])}
          >
            <CheckCheck size={14} />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={busy || !selected.length}
            aria-label="Clear selection"
            title="Clear selection"
            onClick={() => onSelected([])}
          >
            <X size={14} />
          </Button>
          <span className="collection-selection-count" role="status">
            {selected.length} selected
          </span>
        </>
      )}
      {onOpen && multiple && (
        <Button
          variant="ghost"
          size="icon-sm"
          disabled={busy || !selected.length}
          aria-label={`Open selected (${selected.length})`}
          title="Open selected"
          onClick={() => onOpen(selected)}
        >
          <FolderOpen size={14} />
        </Button>
      )}
      {onDelete && (
        <Button
          variant="ghost"
          size="icon-sm"
          disabled={busy || deleteDisabled || !selected.length}
          aria-label={`Delete selected (${selected.length})`}
          title={
            deleteDisabled
              ? "An active recording cannot be deleted"
              : "Delete selected"
          }
          onClick={() => onDelete(selected)}
        >
          <Trash2 size={14} />
        </Button>
      )}
    </div>
  );
}

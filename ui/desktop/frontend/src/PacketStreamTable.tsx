import { useEffect, useState, type PointerEvent } from "react";
import {
  ArrowDown,
  ArrowUp,
  Bookmark,
  Columns3,
  Filter,
  GitCompareArrows,
  MoreHorizontal,
  Pin,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { packetBytes, type PacketRecord } from "./packet-inspector-model";
import { packetTime } from "./packet-workbench";
export const streamColumns = [
  { id: "id", label: "#", width: 70 },
  { id: "time", label: "Time", width: 118 },
  { id: "direction", label: "Direction", width: 100 },
  { id: "name", label: "Packet", width: 270 },
  { id: "bytes", label: "Bytes", width: 80 },
  { id: "stage", label: "Observation", width: 100 },
  { id: "connection", label: "Connection", width: 125 },
];
export type StreamOptions = { compact: boolean; hidden: string[] };
export function readStreamOptions(): StreamOptions {
  try {
    const value = JSON.parse(
      localStorage.getItem("craftbox-packet-stream-options") || "{}",
    );
    return {
      compact: value.compact === true,
      hidden: Array.isArray(value.hidden)
        ? value.hidden.filter(
            (id: unknown) =>
              typeof id === "string" &&
              id !== "name" &&
              streamColumns.some((c) => c.id === id),
          )
        : [],
    };
  } catch {
    return { compact: false, hidden: [] };
  }
}
export function PacketColumnOptions({
  value,
  onChange,
}: {
  value: StreamOptions;
  onChange: (value: StreamOptions) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            aria-label="Packet list options"
            title="Columns and row density"
          >
            <Columns3 size={15} />
          </button>
        }
      />
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          closeOnClick={false}
          onClick={() => onChange({ ...value, compact: !value.compact })}
        >
          {value.compact ? "✓ " : ""}Compact rows
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        {streamColumns
          .filter((c) => c.id !== "name")
          .map((c) => (
            <DropdownMenuItem
              key={c.id}
              closeOnClick={false}
              onClick={() =>
                onChange({
                  ...value,
                  hidden: value.hidden.includes(c.id)
                    ? value.hidden.filter((id) => id !== c.id)
                    : [...value.hidden, c.id],
                })
              }
            >
              {value.hidden.includes(c.id) ? "" : "✓ "}
              {c.label}
            </DropdownMenuItem>
          ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={() => onChange({ compact: false, hidden: [] })}
        >
          Reset visible columns
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
export default function PacketStreamTable({
  rows,
  selected,
  reference,
  options,
  onSelect,
  onType,
  onExclude,
  onPin,
  onBookmark,
  children,
}: {
  rows: PacketRecord[];
  selected: string;
  reference?: string;
  options: StreamOptions;
  onSelect: (id: string) => void;
  onType: (name: string) => void;
  onExclude: (name: string) => void;
  onPin: (packet: PacketRecord) => void;
  onBookmark: (packet: PacketRecord) => void;
  children?: React.ReactNode;
}) {
  const [widths, setWidths] = useState<Record<string, number>>(() => {
    try {
      const stored = JSON.parse(
        localStorage.getItem("craftbox-packet-column-widths") || "{}",
      );
      return Object.fromEntries(
        streamColumns.flatMap((column) => {
          const width = stored?.[column.id];
          return typeof width === "number" && Number.isFinite(width)
            ? [[column.id, Math.max(55, Math.min(650, width))]]
            : [];
        }),
      );
    } catch {
      return {};
    }
  });
  useEffect(() => {
    try {
      localStorage.setItem(
        "craftbox-packet-column-widths",
        JSON.stringify(widths),
      );
    } catch {}
  }, [widths]);
  const columns = streamColumns.filter((c) => !options.hidden.includes(c.id));
  function resize(
    event: PointerEvent<HTMLSpanElement>,
    id: string,
    initial: number,
  ) {
    event.preventDefault();
    const element = event.currentTarget;
    const start = event.clientX;
    element.setPointerCapture(event.pointerId);
    const move = (e: globalThis.PointerEvent) =>
      setWidths((old) => {
        const value = {
          ...old,
          [id]: Math.max(55, Math.min(650, initial + e.clientX - start)),
        };
        try {
          localStorage.setItem(
            "craftbox-packet-column-widths",
            JSON.stringify(value),
          );
        } catch {}
        return value;
      });
    const stop = () => {
      element.removeEventListener("pointermove", move);
      element.removeEventListener("lostpointercapture", stop);
    };
    element.addEventListener("pointermove", move);
    element.addEventListener("lostpointercapture", stop);
  }
  function cell(p: PacketRecord, id: string) {
    if (id === "id")
      return (
        <span className="pi-sequence">
          {p.bookmarked && <Bookmark size={11} />} {p.id}
        </span>
      );
    if (id === "time") return packetTime(p.time);
    if (id === "direction")
      return (
        <span className={`pi-direction ${p.direction}`}>
          {p.direction === "serverbound" ? (
            <ArrowUp size={12} />
          ) : (
            <ArrowDown size={12} />
          )}{" "}
          {p.direction === "serverbound" ? "Server" : "Client"}
        </span>
      );
    if (id === "name")
      return (
        <>
          <strong className="pi-packet-name">
            {p.name}
            {reference === p.id && <Pin size={12} />}
          </strong>
          {p.summary && (
            <small className="pi-packet-summary" title={p.summary}>
              {p.summary}
            </small>
          )}
          {p.partial && <span className="pi-partial">Partial</span>}
          {p.action && <span className="pi-decision">{p.action}</span>}
        </>
      );
    if (id === "bytes") return packetBytes(p.bytes);
    if (id === "stage") return p.stage;
    return p.connectionLeg || "Legacy";
  }
  return (
    <div
      className={`pi-packet-table pi-stream${options.compact ? " is-compact" : ""}${!rows.length ? " is-empty" : ""}`}
      tabIndex={0}
      role="region"
      aria-label="Recorded packets"
      onKeyDown={(e) => {
        if ((e.target as HTMLElement).closest("button,input,select")) return;
        const at = rows.findIndex((p) => p.id === selected);
        const index =
          e.key === "ArrowDown"
            ? Math.min(rows.length - 1, at + 1)
            : e.key === "ArrowUp"
              ? Math.max(0, at - 1)
              : e.key === "Home"
                ? 0
                : e.key === "End"
                  ? rows.length - 1
                  : -1;
        if (index >= 0 && rows[index]) {
          e.preventDefault();
          onSelect(rows[index].id);
          e.currentTarget
            .querySelector(`[data-packet-id="${rows[index].id}"]`)
            ?.scrollIntoView({ block: "nearest" });
        }
      }}
    >
      <table aria-label="Packet observations">
        <colgroup>
          {columns.map((c) => (
            <col key={c.id} style={{ width: widths[c.id] || c.width }} />
          ))}
          <col style={{ width: 34 }} />
        </colgroup>
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.id} scope="col">
                {c.label}
                <span
                  className="pi-column-grip"
                  role="separator"
                  aria-orientation="vertical"
                  aria-label={`Resize ${c.label} column`}
                  tabIndex={0}
                  onPointerDown={(event) =>
                    resize(event, c.id, widths[c.id] || c.width)
                  }
                  onDoubleClick={() =>
                    setWidths((old) => ({ ...old, [c.id]: c.width }))
                  }
                  onKeyDown={(event) => {
                    if (
                      event.key === "ArrowLeft" ||
                      event.key === "ArrowRight"
                    ) {
                      event.preventDefault();
                      setWidths((old) => ({
                        ...old,
                        [c.id]: Math.max(
                          55,
                          Math.min(
                            650,
                            (old[c.id] || c.width) +
                              (event.key === "ArrowRight" ? 16 : -16),
                          ),
                        ),
                      }));
                    }
                  }}
                />
              </th>
            ))}
            <th aria-label="Packet actions" />
          </tr>
        </thead>
        <tbody>
          {rows.map((p) => (
            <tr
              key={p.id}
              data-packet-id={p.id}
              aria-selected={selected === p.id}
              className={`${selected === p.id ? "selected " : ""}${p.direction}`}
              onClick={() => onSelect(p.id)}
            >
              {columns.map((c) => (
                <td
                  key={c.id}
                  title={
                    c.id === "connection"
                      ? `${p.sourceAddress || "Not recorded"} → ${p.destinationAddress || "Not recorded"}`
                      : undefined
                  }
                >
                  {cell(p, c.id)}
                </td>
              ))}
              <td
                className="pi-row-actions"
                onClick={(e) => e.stopPropagation()}
              >
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <button aria-label={`Actions for packet ${p.id}`}>
                        <MoreHorizontal size={14} />
                      </button>
                    }
                  />
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => onType(p.name)}>
                      <Filter size={13} />
                      Only {p.name}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => onExclude(p.name)}>
                      Exclude this packet type
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => onPin(p)}>
                      <GitCompareArrows size={13} />
                      Use as comparison reference
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => onBookmark(p)}>
                      <Bookmark size={13} />
                      {p.bookmarked ? "Remove bookmark" : "Bookmark packet"}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {children}
    </div>
  );
}

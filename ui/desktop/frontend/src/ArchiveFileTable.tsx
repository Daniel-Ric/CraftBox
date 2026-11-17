import { Archive, Binary, FileCode, Image, TriangleAlert } from "lucide-react";
import { labBytes, type LabChange, type LabFile } from "./bedrock-lab-model";
export default function ArchiveFileTable({
  rows,
  selected,
  onSelect,
  changes,
  comparing,
}: {
  rows: LabFile[];
  selected: string;
  onSelect: (path: string, button: HTMLButtonElement) => void;
  changes: Map<string, LabChange>;
  comparing: boolean;
}) {
  return (
    <table
      className="lab-files-table archive-file-table"
      aria-label="Snapshot files"
      onKeyDown={(event) => {
        if (
          event.key !== "ArrowDown" &&
          event.key !== "ArrowUp" &&
          event.key !== "Home" &&
          event.key !== "End"
        )
          return;
        const buttons = [
          ...event.currentTarget.querySelectorAll<HTMLButtonElement>(
            "button[data-file-open]",
          ),
        ];
        const at = buttons.indexOf(document.activeElement as HTMLButtonElement);
        if (at < 0) return;
        const next =
          event.key === "Home"
            ? 0
            : event.key === "End"
              ? buttons.length - 1
              : Math.max(
                  0,
                  Math.min(
                    buttons.length - 1,
                    at + (event.key === "ArrowDown" ? 1 : -1),
                  ),
                );
        if (buttons[next] && rows[next]) {
          event.preventDefault();
          buttons[next].focus();
          buttons[next].scrollIntoView?.({ block: "nearest" });
          onSelect(rows[next].path, buttons[next]);
        }
      }}
    >
      <colgroup>
        <col />
        <col style={{ width: 100 }} />
        <col style={{ width: 85 }} />
        <col className="archive-detail-column" style={{ width: 180 }} />
      </colgroup>
      <thead>
        <tr>
          <th scope="col">File</th>
          <th scope="col">{comparing ? "Change" : "Type"}</th>
          <th scope="col">Size</th>
          <th scope="col" className="archive-file-detail">
            Details
          </th>
        </tr>
      </thead>
      <tbody>
        {rows.map((file) => {
          const slash = file.path.lastIndexOf("/"),
            name = file.path.slice(slash + 1),
            parent = slash < 0 ? "" : file.path.slice(0, slash);
          const change = changes.get(file.path);
          const status = comparing ? change?.status : undefined;
          const Icon =
            file.kind === "image"
              ? Image
              : file.kind === "archive"
                ? Archive
                : file.kind === "binary"
                  ? Binary
                  : FileCode;
          return (
            <tr
              key={file.path}
              aria-selected={selected === file.path}
              data-status={status}
            >
              <td>
                <button
                  data-file-open
                  aria-label={file.path}
                  title={file.path}
                  onClick={(event) => onSelect(file.path, event.currentTarget)}
                >
                  <Icon size={15} />
                  <span className="archive-file-name">
                    <strong>
                      {name}
                      {file.issue && (
                        <TriangleAlert size={11} aria-label="File issue" />
                      )}
                    </strong>
                    <small>{parent || "Snapshot root"}</small>
                  </span>
                </button>
              </td>
              <td>
                <span
                  className={`archive-file-kind ${status ? "is-" + status : ""}`}
                >
                  {status || file.kind}
                </span>
              </td>
              <td>{labBytes(file.bytes)}</td>
              <td className="archive-file-detail">
                {file.issue && (
                  <TriangleAlert size={13} aria-label="File issue" />
                )}
                <span title={file.issue || change?.summary || file.origin}>
                  {comparing
                    ? change?.summary ||
                      (status === "renamed"
                        ? `From ${change?.before?.path}`
                        : "")
                    : file.origin
                      ? "Unpacked"
                      : ""}
                </span>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

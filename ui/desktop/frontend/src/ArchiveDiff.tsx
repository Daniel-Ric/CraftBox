import { useEffect, useMemo, useState } from "react";
import { api } from "./bridge";
import type { LabJSONAnalysis } from "./bedrock-lab-model";
import type { LabPreview } from "./bedrock-lab-model";
import { lineDiff, diffHunks, type DiffLine } from "./archive-diff";
import ArchiveChangeSummary from "./ArchiveChangeSummary";
import ArchiveBinaryCard from "./ArchiveBinaryCard";

type DiffPair = { old?: DiffLine; next?: DiffLine };
function splitRows(rows: DiffLine[]): DiffPair[] {
  const pairs: DiffPair[] = [];
  for (let i = 0; i < rows.length;) {
    const row = rows[i];
    if (row.kind === "context") {
      pairs.push({ old: row, next: row });
      i++;
      continue;
    }
    const removed: DiffLine[] = [],
      added: DiffLine[] = [];
    while (i < rows.length && rows[i].kind !== "context") {
      const r = rows[i++];
      (r.kind === "removed" ? removed : added).push(r);
    }
    for (let j = 0; j < Math.max(removed.length, added.length); j++)
      pairs.push({ old: removed[j], next: added[j] });
  }
  return pairs;
}

export default function ArchiveDiff({
  before,
  after,
  beforeLabel,
  afterLabel,
  status,
}: {
  before: LabPreview | null;
  after: LabPreview | null;
  beforeLabel: string;
  afterLabel: string;
  status: string;
}) {
  const [requestedMode, setMode] = useState("summary"),
    [full, setFull] = useState(false),
    [source, setSource] = useState(false),
    [page, setPage] = useState(0);
  const json =
    [before, after].some(Boolean) &&
    [before, after].every((p) => !p || p.file.kind === "json");
  const mode = requestedMode === "summary" && !json ? "split" : requestedMode;
  const left = before
      ? source
        ? (before.original ?? before.text)
        : (before.canonical ?? before.text)
      : "",
    right = after
      ? source
        ? (after.original ?? after.text)
        : (after.canonical ?? after.text)
      : "";
  const diff = useMemo(() => lineDiff(left, right), [left, right]);
  const hunks = useMemo(
    () => (full ? [{ start: 0, end: diff.rows.length }] : diffHunks(diff.rows)),
    [diff.rows, full],
  );
  const preparedHunks = useMemo(
    () =>
      hunks.map((h) => ({
        ...h,
        pairs:
          mode === "split"
            ? splitRows(diff.rows.slice(h.start, h.end))
            : undefined,
      })),
    [hunks, mode, diff.rows],
  );
  const totalRows = preparedHunks.reduce(
      (n, h) => n + (h.pairs?.length ?? h.end - h.start),
      0,
    ),
    currentPage = Math.min(page, Math.max(0, Math.ceil(totalRows / 1000) - 1));
  const visibleHunks = useMemo(() => {
    let skip = currentPage * 1000,
      remaining = 1000;
    const visible: { start: number; end: number; pairs?: DiffPair[] }[] = [];
    for (const h of preparedHunks) {
      const size = h.pairs?.length ?? h.end - h.start;
      if (skip >= size) {
        skip -= size;
        continue;
      }
      const count = Math.min(size - skip, remaining);
      if (h.pairs)
        visible.push({
          start: h.start,
          end: h.end,
          pairs: h.pairs.slice(skip, skip + count),
        });
      else visible.push({ start: h.start + skip, end: h.start + skip + count });
      remaining -= count;
      skip = 0;
      if (!remaining) break;
    }
    return visible;
  }, [preparedHunks, currentPage]);
  const textual = [before, after]
    .filter(Boolean)
    .every((p) => p!.file.kind === "json" || p!.file.kind === "text");
  const partial =
    textual && (before?.truncated || after?.truncated || diff.limited);
  const [analysis, setAnalysis] = useState<LabJSONAnalysis | null>(null);
  const [analysisError, setAnalysisError] = useState("");
  useEffect(() => {
    let active = true;
    setAnalysis(null);
    setAnalysisError("");
    if (partial || [before, after].some((p) => p && p.file.kind !== "json"))
      return;
    void api
      .AnalyzeBedrockJSON(
        before?.canonical ?? before?.original ?? before?.text ?? "",
        after?.canonical ?? after?.original ?? after?.text ?? "",
      )
      .then((result) => {
        if (active) setAnalysis(result);
      })
      .catch((e) => {
        if (active)
          setAnalysisError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      active = false;
    };
  }, [before, after, partial]);
  if (!before && !after)
    return (
      <div className="lab-empty">
        <p>Select a changed file to inspect its diff.</p>
      </div>
    );
  return (
    <section className="lab-diff" aria-label="Line differences">
      <header className="lab-diff-toolbar">
        <b>{status}</b>
        {textual && (
          <>
            <span className="lab-added">+{diff.added}</span>
            <span className="lab-removed">−{diff.removed}</span>
            <div className="lab-diff-modes" aria-label="Diff layout">
              {(json
                ? ["summary", "split", "unified"]
                : ["split", "unified"]
              ).map((m) => (
                <button
                  key={m}
                  aria-pressed={mode === m}
                  onClick={() => setMode(m)}
                >
                  {m === "summary"
                    ? "Readable summary"
                    : m === "split"
                      ? "Split"
                      : "Unified"}
                </button>
              ))}
            </div>
            <label>
              <input
                type="checkbox"
                checked={full}
                onChange={(e) => setFull(e.target.checked)}
              />
              All lines
            </label>
            <label>
              <input
                type="checkbox"
                checked={source}
                onChange={(e) => setSource(e.target.checked)}
              />
              Original formatting
            </label>
          </>
        )}
      </header>
      {partial && (
        <p className="lab-warning">
          Partial comparison: only the previewed content and at most 20,000
          lines per file are compared. Counts cover that portion. Export the
          snapshot for complete files.
        </p>
      )}
      {diff.coarse && textual && (
        <p className="lab-warning">
          Large differing regions are shown as replacement blocks to keep the
          comparison responsive.
        </p>
      )}
      {analysisError && (
        <p className="lab-warning">
          Field analysis unavailable: {analysisError}. Split and Unified remain
          available.
        </p>
      )}
      {mode !== "summary" && analysis && analysis.fields.length > 0 && (
        <details className="lab-field-analysis">
          <summary>
            {analysis.versionOnly
              ? "Only format_version changed; the remaining JSON values are identical."
              : `${analysis.fields.length}${analysis.truncated ? "+" : ""} JSON field changes`}
          </summary>
          <div>
            {analysis.fields.map((f) => (
              <p key={f.path}>
                <code>{f.path || "/"}</code>
                <span className="lab-removed">
                  {f.status !== "added" && `− ${f.before}`}
                </span>
                <span className="lab-added">
                  {f.status !== "removed" && `+ ${f.after}`}
                </span>
              </p>
            ))}
            {analysis.truncated && (
              <p className="lab-hint">
                Field list limited to 256 changes. Inspect the line diff for
                further changes.
              </p>
            )}
          </div>
        </details>
      )}
      {textual && mode === "summary" ? (
        <>
          <div className="lab-diff-labels">
            <b>Before · {beforeLabel}</b>
            <b>After · {afterLabel}</b>
          </div>
          {[before, after].some((p) => p?.file.rawHash) && (
            <p className="lab-decoding-notice">
              MCB reconstructed automatically with schema{" "}
              {after?.decoding?.version ||
                before?.decoding?.version ||
                "documents stored in the snapshot"}
              . Compiled originals are retained.
            </p>
          )}
          {partial ? (
            <p className="lab-warning">
              Field analysis is unavailable for a partial preview. Choose Split
              or Unified to inspect the available content.
            </p>
          ) : analysisError ? (
            <p className="lab-placeholder">
              Choose Split or Unified to inspect the available file contents.
            </p>
          ) : (
            <ArchiveChangeSummary analysis={analysis} />
          )}
        </>
      ) : textual ? (
        <div className="lab-diff-scroll" tabIndex={0}>
          <div className="lab-diff-labels">
            <b>Before · {beforeLabel}</b>
            <b>After · {afterLabel}</b>
          </div>
          {!before && <p className="lab-hint">Not present in {beforeLabel}</p>}
          {!after && <p className="lab-hint">Not present in {afterLabel}</p>}
          {!diff.added && !diff.removed && (
            <p className="lab-placeholder">
              {status === "renamed"
                ? "File moved; contents are identical."
                : "No text differences in this view. Enable Original formatting to inspect source formatting."}
            </p>
          )}
          <table
            className={`lab-diff-table ${mode}`}
            aria-label={mode === "split" ? "Split diff" : "Unified diff"}
          >
            <tbody>
              {visibleHunks.map(({ start, end, pairs }, h) => (
                <DiffHunk
                  key={h}
                  start={start}
                  end={end}
                  rows={diff.rows}
                  mode={mode}
                  pairs={pairs}
                />
              ))}
            </tbody>
          </table>
          {totalRows > 1000 && (
            <nav className="lab-diff-pagination" aria-label="Diff pages">
              <button
                disabled={currentPage === 0}
                onClick={() => setPage(currentPage - 1)}
              >
                Previous lines
              </button>
              <span>
                {currentPage * 1000 + 1}–
                {Math.min(totalRows, (currentPage + 1) * 1000)} of{" "}
                {totalRows.toLocaleString()} displayed lines
              </span>
              <button
                disabled={(currentPage + 1) * 1000 >= totalRows}
                onClick={() => setPage(currentPage + 1)}
              >
                Next lines
              </button>
            </nav>
          )}
        </div>
      ) : (
        <div className="lab-diff-binary">
          <ArchiveBinaryCard
            preview={before}
            other={after}
            label={`Before · ${beforeLabel}`}
          />
          <ArchiveBinaryCard
            preview={after}
            other={before}
            label={`After · ${afterLabel}`}
          />
        </div>
      )}
    </section>
  );
}
function DiffHunk({
  start,
  end,
  rows,
  mode,
  pairs,
}: {
  start: number;
  end: number;
  rows: DiffLine[];
  mode: string;
  pairs?: DiffPair[];
}) {
  const slice = pairs
      ? pairs.flatMap((p) =>
          p.old === p.next
            ? [p.old!]
            : [p.old, p.next].filter((r): r is DiffLine => !!r),
        )
      : rows.slice(start, end),
    firstOld = slice.find((r) => r.oldLine)?.oldLine ?? 0,
    firstNew = slice.find((r) => r.newLine)?.newLine ?? 0;
  return (
    <>
      <tr className="lab-diff-hunk">
        <td colSpan={mode === "split" ? 6 : 4}>
          @@ −{firstOld},{slice.filter((r) => r.oldLine).length} +{firstNew},
          {slice.filter((r) => r.newLine).length} @@
          {start > 0 && ` · ${start} preceding lines hidden`}
        </td>
      </tr>
      {mode === "unified"
        ? slice.map((r, i) => (
            <tr key={i} data-kind={r.kind}>
              <td className="lab-line-number">{r.oldLine}</td>
              <td className="lab-line-number">{r.newLine}</td>
              <td className="lab-line-sign">
                {r.kind === "added" ? "+" : r.kind === "removed" ? "−" : " "}
              </td>
              <td className="lab-line-text">
                <code>{r.text || " "}</code>
              </td>
            </tr>
          ))
        : (pairs ?? []).map((r, i) => (
            <tr key={i}>
              <td className="lab-line-number" data-kind={r.old?.kind}>
                {r.old?.oldLine}
              </td>
              <td className="lab-line-sign" data-kind={r.old?.kind}>
                {r.old?.kind === "removed" ? "−" : " "}
              </td>
              <td className="lab-line-text" data-kind={r.old?.kind ?? "empty"}>
                <code>{r.old?.text || " "}</code>
              </td>
              <td className="lab-line-number" data-kind={r.next?.kind}>
                {r.next?.newLine}
              </td>
              <td className="lab-line-sign" data-kind={r.next?.kind}>
                {r.next?.kind === "added" ? "+" : " "}
              </td>
              <td className="lab-line-text" data-kind={r.next?.kind ?? "empty"}>
                <code>{r.next?.text || " "}</code>
              </td>
            </tr>
          ))}
    </>
  );
}

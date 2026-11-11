import { useMemo, useState } from "react";
import type { LabJSONAnalysis } from "./bedrock-lab-model";

function fieldTitle(path: string) {
  const parts = path
    .split("/")
    .filter(Boolean)
    .map((s) => s.replace(/~1/g, "/").replace(/~0/g, "~"));
  if (path === "/format_version") return "Document format version";
  const component = parts.indexOf("components");
  if (component >= 0 && parts[component + 1])
    return `${parts[component + 1]}${parts.length > component + 2 ? ` / ${parts.slice(component + 2).join(" / ")}` : " component"}`;
  const final = parts.at(-1);
  return final ? final.replaceAll("_", " ") : "Document";
}
export default function ArchiveChangeSummary({
  analysis,
}: {
  analysis: LabJSONAnalysis | null;
}) {
  const [filter, setFilter] = useState("");
  const groups = useMemo(
    () =>
      analysis?.fields.filter((f) =>
        `${f.path} ${f.before} ${f.after}`
          .toLowerCase()
          .includes(filter.toLowerCase()),
      ) ?? [],
    [analysis, filter],
  );
  if (!analysis)
    return (
      <div className="lab-readable-summary" role="status">
        Analysing JSON field changes…
      </div>
    );
  return (
    <section
      className="lab-readable-summary"
      aria-label="Readable change summary"
    >
      <header>
        <h3>
          {analysis.versionOnly
            ? "Format version update"
            : analysis.fields.length
              ? `${analysis.fields.length}${analysis.truncated ? "+" : ""} changed JSON fields`
              : "The JSON values are identical"}
        </h3>
        <p>
          {analysis.versionOnly
            ? "No other JSON values changed. A format version update alone does not establish a gameplay change."
            : analysis.fields.length
              ? "Read the old and new values below. Paths identify the exact fields; the code diff provides surrounding context."
              : "Only formatting, property order or the file location may differ."}
        </p>
      </header>
      {analysis.fields.length > 6 && (
        <input
          aria-label="Search changed fields"
          placeholder="Find a component, field or value…"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      )}
      <div className="lab-field-cards">
        {groups.map((f) => (
          <article key={f.path} data-change={f.status}>
            <header>
              <span>
                {f.status === "added"
                  ? "Added"
                  : f.status === "removed"
                    ? "Removed"
                    : "Changed"}
              </span>
              <b>{fieldTitle(f.path)}</b>
            </header>
            <code className="lab-field-path">{f.path || "/"}</code>
            <div className="lab-field-values">
              <section>
                <small>Before</small>
                <pre>{f.status === "added" ? "Not present" : f.before}</pre>
              </section>
              <section>
                <small>After</small>
                <pre>{f.status === "removed" ? "Not present" : f.after}</pre>
              </section>
            </div>
          </article>
        ))}
      </div>
      {analysis.truncated && (
        <p className="lab-warning">
          Showing the first 256 field changes. Use the line diff or export the
          snapshot for complete content.
        </p>
      )}
    </section>
  );
}

import { useMemo, useState } from "react";
import { GitCompareArrows, Pin, X } from "lucide-react";
import { comparePacketFields } from "./packet-workbench";
import type { PacketRecord } from "./packet-inspector-model";
export default function PacketComparison({
  baseline,
  packet,
  onClear,
  onPin,
}: {
  baseline?: PacketRecord;
  packet?: PacketRecord;
  onClear: () => void;
  onPin: () => void;
}) {
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(100);
  const diff = useMemo(
    () =>
      baseline && packet ? comparePacketFields(baseline, packet) : undefined,
    [baseline, packet],
  );
  const rows =
    diff?.rows.filter((row) =>
      `${row.path} ${row.before} ${row.after}`
        .toLowerCase()
        .includes(query.toLowerCase()),
    ) || [];
  if (!baseline)
    return (
      <div className="pi-comparison-empty">
        <GitCompareArrows size={26} />
        <h3>Compare two observations</h3>
        <p>
          Pin this packet as a reference, then select another packet. CraftBox
          compares decoded fields and preserves full 64-bit identifiers.
        </p>
        <button disabled={!packet} onClick={onPin}>
          <Pin size={14} />
          Use as reference
        </button>
      </div>
    );
  return (
    <section className="pi-comparison" aria-label="Packet comparison">
      <div className="pi-reference">
        <Pin size={14} />
        <span>
          Reference{" "}
          <strong>
            #{baseline.id} {baseline.name}
          </strong>
        </span>
        <button aria-label="Clear reference packet" onClick={onClear}>
          <X size={14} />
        </button>
      </div>
      {baseline.id === packet?.id ? (
        <p>Select a different packet in the list to see its changes.</p>
      ) : (
        <>
          <p className="pi-comparison-caption">
            #{baseline.id} → #{packet?.id} ·{" "}
            {diff?.rows.length.toLocaleString()} differing fields
            {baseline.name !== packet?.name ? " · Different packet types" : ""}
          </p>
          {diff?.incomplete && (
            <p className="pi-comparison-warning" role="status">
              Partial comparison: data is redacted, incomplete or exceeds 4,000
              fields. A missing field does not prove a network change.
            </p>
          )}
          {diff?.error ? (
            <p role="status">{diff.error}</p>
          ) : (
            <>
              <input
                type="search"
                aria-label="Find comparison field"
                placeholder="Find a field or value"
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setLimit(100);
                }}
              />
              <div className="pi-comparison-table">
                <table>
                  <thead>
                    <tr>
                      <th>Field</th>
                      <th>Reference</th>
                      <th>Selected</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.slice(0, limit).map((row) => (
                      <tr key={row.path} data-change={row.kind}>
                        <th>
                          <span>{row.kind}</span>
                          {row.path || "(root)"}
                        </th>
                        <td>{row.before ?? "Not present"}</td>
                        <td>{row.after ?? "Not present"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {!rows.length && (
                <p>
                  {query
                    ? "No matching differences."
                    : "No differences in the available decoded fields."}
                </p>
              )}
              {rows.length > limit && (
                <button onClick={() => setLimit((value) => value + 100)}>
                  Show next {Math.min(100, rows.length - limit)} fields
                </button>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
}

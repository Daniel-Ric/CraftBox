import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, BarChart3, Download } from "lucide-react";
import { packetBytes, type PacketAnalysis } from "./packet-inspector-model";
import {
  trafficSummary,
  trafficWindowQuery,
  fillTrafficBuckets,
} from "./packet-workbench";
export default function PacketTraffic({
  analysis,
  onType,
  onWindow,
  onError,
}: {
  analysis?: PacketAnalysis;
  onType: (name: string) => void;
  onWindow: (search: string) => void;
  onError: (e: unknown) => void;
}) {
  const [metric, setMetric] = useState("packets");
  const [sort, setSort] = useState("count");
  const [query, setQuery] = useState("");
  const totals = useMemo(
    () => trafficSummary(analysis?.types || []),
    [analysis],
  );
  const bucketWidth = analysis?.bucketMS || totals.bucketWidth;
  const buckets = useMemo(
    () => fillTrafficBuckets(analysis?.buckets || [], bucketWidth),
    [analysis, bucketWidth],
  );
  const types = useMemo(
    () =>
      [...(analysis?.types || [])]
        .filter((t) => t.name.toLowerCase().includes(query.toLowerCase()))
        .sort((a, b) =>
          sort === "name"
            ? a.name.localeCompare(b.name)
            : sort === "bytes"
              ? b.bytes - a.bytes
              : b.count - a.count,
        ),
    [analysis, sort, query],
  );
  if (!analysis)
    return (
      <div className="pi-empty">
        <BarChart3 size={30} />
        <h3>Traffic analysis</h3>
        <p>
          Refresh analysis to inspect recorded traffic, packet distribution and
          directional volume.
        </p>
      </div>
    );
  const max = Math.max(
    1,
    ...analysis.buckets.map((b) =>
      metric === "bytes" ? b.bytes : b.serverbound + b.clientbound,
    ),
  );
  async function exportCSV() {
    try {
      const rows = [
        ["Packet", "ID", "Count", "Bytes", "Serverbound", "Clientbound"],
        ...types.map((t) => [
          t.name,
          t.packetId,
          t.count,
          t.bytes,
          t.serverbound,
          t.clientbound,
        ]),
      ];
      const data = rows
        .map((row) =>
          row
            .map(
              (value) =>
                '"' +
                (typeof value === "string" && /^[=+\-@\t\r]/.test(value)
                  ? "'" + value
                  : String(value)
                ).replace(/"/g, '""') +
                '"',
            )
            .join(","),
        )
        .join("\r\n");
      const url = URL.createObjectURL(
        new Blob([data], { type: "text/csv;charset=utf-8" }),
      );
      const link = document.createElement("a");
      link.href = url;
      link.download = "craftbox-packet-statistics.csv";
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      onError(error);
    }
  }
  return (
    <section className="pi-traffic" aria-label="Traffic analysis">
      <div className="pi-traffic-summary">
        <div>
          <strong>{totals.count.toLocaleString()}</strong>
          <span>Observations</span>
        </div>
        <div>
          <strong>{packetBytes(totals.bytes)}</strong>
          <span>Payload volume</span>
        </div>
        <div>
          <strong>
            {totals.rate === null ? "—" : totals.rate.toFixed(1) + "/s"}
          </strong>
          <span>Observed mean rate</span>
        </div>
        <div>
          <strong>
            <ArrowUp size={13} />
            {totals.serverbound.toLocaleString()} <ArrowDown size={13} />
            {totals.clientbound.toLocaleString()}
          </strong>
          <span>Serverbound / clientbound</span>
        </div>
      </div>
      <div className="pi-chart-heading">
        <h3>Recorded traffic</h3>
        <div>
          <button
            aria-pressed={metric === "packets"}
            onClick={() => setMetric("packets")}
          >
            Packets
          </button>
          <button
            aria-pressed={metric === "bytes"}
            onClick={() => setMetric("bytes")}
          >
            Bytes
          </button>
        </div>
      </div>
      <div
        className="pi-traffic-chart"
        role="group"
        aria-label="Traffic time buckets"
      >
        {buckets.map((b, i) => (
          <button
            key={b.time}
            title={`${new Date(b.time).toLocaleTimeString()} · ${b.serverbound} serverbound · ${b.clientbound} clientbound · ${packetBytes(b.bytes)}. Inspect this time window.`}
            aria-label={`Inspect traffic at ${new Date(b.time).toLocaleTimeString()} (${i + 1})`}
            onClick={() => onWindow(trafficWindowQuery(b.time, bucketWidth))}
            style={{
              height: `${Math.max(3, (100 * (metric === "bytes" ? b.bytes : b.serverbound + b.clientbound)) / max)}%`,
            }}
          >
            {metric === "packets" && (
              <>
                <span style={{ flex: b.serverbound || 0 }} />
                <i style={{ flex: b.clientbound || 0 }} />
              </>
            )}
          </button>
        ))}
      </div>
      <div className="pi-chart-caption">
        <span>
          {totals.first
            ? new Date(totals.first).toLocaleTimeString()
            : "No timestamps"}
        </span>
        <span>
          {metric === "packets" ? (
            <>
              <i />
              Serverbound <b />
              Clientbound
            </>
          ) : (
            "Payload bytes"
          )}{" "}
          · Select a bar to inspect its packets
        </span>
        <span>
          {totals.last ? new Date(totals.last).toLocaleTimeString() : ""}
        </span>
      </div>
      <div className="pi-chart-heading">
        <h3>Packet distribution</h3>
        <div>
          <input
            type="search"
            aria-label="Search traffic types"
            placeholder="Find packet type"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <select
            aria-label="Sort traffic types"
            value={sort}
            onChange={(e) => setSort(e.target.value)}
          >
            <option value="count">Most packets</option>
            <option value="bytes">Most bytes</option>
            <option value="name">Name</option>
          </select>
          <button onClick={() => void exportCSV()}>
            <Download size={13} />
            CSV
          </button>
        </div>
      </div>
      <table className="pi-traffic-types">
        <thead>
          <tr>
            <th>Packet type</th>
            <th>Count</th>
            <th>Share</th>
            <th>Payload</th>
            <th>↑ Server</th>
            <th>↓ Client</th>
          </tr>
        </thead>
        <tbody>
          {types.map((t) => (
            <tr key={t.name}>
              <td>
                <button onClick={() => onType(t.name)}>{t.name}</button>
              </td>
              <td>{t.count.toLocaleString()}</td>
              <td>
                <span className="pi-type-share">
                  <i
                    style={{
                      width: `${(100 * t.count) / Math.max(1, totals.count)}%`,
                    }}
                  />
                  {((100 * t.count) / Math.max(1, totals.count)).toFixed(1)}%
                </span>
              </td>
              <td>{packetBytes(t.bytes)}</td>
              <td>{t.serverbound.toLocaleString()}</td>
              <td>{t.clientbound.toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="pi-traffic-note">
        Capture-wide recorded observations, including distinct received and sent
        observations. Rate measures observed arrivals, not network latency.
        Transport overhead is excluded.
      </p>
    </section>
  );
}

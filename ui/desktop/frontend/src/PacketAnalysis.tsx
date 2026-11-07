import FeedbackFlyout from "./FeedbackFlyout";
import { useMemo, useState } from "react";
import {
  type PacketAnalysis,
  type PacketEvidence,
} from "./packet-inspector-model";
export default function PacketAnalysisView({
  analysis,
  tab,
  onPacket,
}: {
  analysis?: PacketAnalysis;
  tab: string;
  onPacket: (id: string) => void;
  onType: (name: string) => void;
}) {
  const [actor, setActor] = useState("");
  const entity =
    analysis?.entities.find((e) => (e.key || e.id) === actor) ??
    analysis?.entities[0];
  const route = useMemo(() => {
    const pts = entity?.positions ?? [];
    if (!pts.length) return "";
    const xs = pts.map((p) => p[0]),
      zs = pts.map((p) => p[2]);
    const minX = Math.min(...xs),
      minZ = Math.min(...zs),
      range = Math.max(1, Math.max(...xs) - minX, Math.max(...zs) - minZ);
    return pts
      .map(
        (p) =>
          `${20 + ((p[0] - minX) / range) * 560},${280 - ((p[2] - minZ) / range) * 260}`,
      )
      .join(" ");
  }, [entity]);
  if (!analysis)
    return (
      <div className="pi-empty">
        <h3>Analyse this capture</h3>
        <p>
          Choose Refresh analysis to build evidence indexes from the recording.
        </p>
      </div>
    );
  const evidence = (rows: PacketEvidence[]) =>
    rows.length ? (
      <table className="pi-evidence">
        <thead>
          <tr>
            <th>Time</th>
            <th>Evidence</th>
            <th>Direction</th>
            <th>Packet</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr
              key={r.id}
              tabIndex={0}
              onClick={() => onPacket(r.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter") onPacket(r.id);
              }}
            >
              <td>{new Date(r.time).toLocaleTimeString()}</td>
              <td>{r.summary}</td>
              <td>{r.direction}</td>
              <td>#{r.id}</td>
            </tr>
          ))}
        </tbody>
      </table>
    ) : (
      <p className="pi-empty">No matching evidence in this capture.</p>
    );
  return (
    <div className="pi-analysis">
      {analysis.partial && (
        <FeedbackFlyout title="Packet analysis" tone="warning">
          Some evidence/history lists reached their display limits. Original
          recorded packets remain searchable.
        </FeedbackFlyout>
      )}
      {tab === "Timeline" && evidence(analysis.timeline)}
      {tab === "Transactions" && (
        <>
          <p>
            Observed request/form identifiers and packet evidence. Missing
            replies and truncated source logs remain incomplete.
          </p>
          {evidence(analysis.transactions)}
        </>
      )}
      {tab === "Chunks" && (
        <>
          <p>
            Chunk delivery and client-cache evidence. Select an event to inspect
            coordinates, hashes and payload.
          </p>
          {evidence(analysis.chunks)}
        </>
      )}
      {tab === "Changes" && (
        <>
          <p>
            Recorded interception decisions and explicit sends. “Received” and
            “sent” packet rows expose the endpoint observations.
          </p>
          {evidence(analysis.changes)}
        </>
      )}
      {(tab === "Entities" || tab === "Movement") && (
        <>
          <label>
            Actor
            <select
              aria-label="Observed actor"
              value={entity?.key || entity?.id || ""}
              onChange={(e) => setActor(e.target.value)}
            >
              {analysis.entities.map((e) => (
                <option key={e.key || e.id} value={e.key || e.id}>
                  {e.name || e.type || "Actor"} · {e.id}
                </option>
              ))}
            </select>
          </label>
          {entity ? (
            <>
              <dl className="pi-properties">
                <dt>Runtime/observed ID</dt>
                <dd>{entity.id}</dd>
                <dt>Type</dt>
                <dd>{entity.type || "Unknown"}</dd>
                <dt>Position</dt>
                <dd>{entity.position.map((n) => n.toFixed(3)).join(", ")}</dd>
                <dt>Lifecycle</dt>
                <dd>
                  {entity.partial
                    ? "Spawn not observed"
                    : entity.removed
                      ? "Removed"
                      : "Last observed"}
                </dd>
              </dl>
              <svg
                className="pi-route"
                viewBox="0 0 600 310"
                role="img"
                aria-label="Observed actor movement on X Z plane"
              >
                <path d="M20 20V280H580" className="pi-chart-grid" />
                <polyline
                  points={route}
                  fill="none"
                  stroke="var(--pi-blue)"
                  strokeWidth="2.5"
                />
              </svg>
              <p>
                Last {entity.positions.length} observed positions on the X/Z
                plane.
              </p>
              <div className="pi-inline-actions">
                {entity.evidence.map((id) => (
                  <button key={id} onClick={() => onPacket(id)}>
                    #{id}
                  </button>
                ))}
              </div>
            </>
          ) : (
            <p>
              No actor identity or positions were present in the decoded
              packets.
            </p>
          )}
        </>
      )}
    </div>
  );
}

import { formatLocale } from "./i18n";
import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useMemo, useState, useRef } from "react";
import {
  Copy,
  X,
  Bookmark,
  Send,
  Search,
  ArrowUpRight,
  GitCompareArrows,
} from "lucide-react";
import PacketFormPreview from "./PacketFormPreview";
import PacketBytes from "./PacketBytes";
import PacketComparison from "./PacketComparison";
import { packetForm } from "./packet-form-model";
import { api } from "./bridge";
import {
  packetBytes,
  packetFields,
  parseExactJSON,
  rawBytes,
  type PacketLink,
  type PacketRecord,
} from "./packet-inspector-model";
import { openPacketContent } from "./packet-navigation";

function JSONTree({
  value,
  name = "root",
  depth = 0,
}: {
  value: unknown;
  name?: string;
  depth?: number;
}) {
  const [expanded, setExpanded] = useState(depth < 2);
  const [limit, setLimit] = useState(100);
  if (value && typeof value === "object" && depth < 16) {
    const total = Array.isArray(value)
      ? value.length
      : Object.keys(value).length;
    const entries = Array.isArray(value)
      ? value.slice(0, limit).map((v, i) => [String(i), v] as [string, unknown])
      : Object.entries(value).slice(0, limit);
    return (
      <details
        open={expanded}
        onToggle={(e) => setExpanded(e.currentTarget.open)}
      >
        <summary>
          <b translate={depth > 0 ? "no" : undefined}>{name}</b>
          <span>{Array.isArray(value) ? `[${total}]` : `{${total}}`}</span>
        </summary>
        {expanded && (
          <div className="pi-json-children">
            {entries.map(([k, v]) => (
              <JSONTree key={k} name={k} value={v} depth={depth + 1} />
            ))}
            {total > limit && (
              <button onClick={() => setLimit((n) => n + 100)}>
                Show next {Math.min(100, total - limit)} of {total - limit}{" "}
                remaining fields
              </button>
            )}
          </div>
        )}
      </details>
    );
  }
  return (
    <div className="pi-json-leaf">
      <b translate={depth > 0 ? "no" : undefined}>{name}</b>:{" "}
      <span className={typeof value === "string" ? "pi-string" : "pi-number"}>
        <span translate="no">{(JSON.stringify(value) ?? "null").slice(0, 1000)}</span>
        {(JSON.stringify(value)?.length ?? 0) > 1000 && (
          <button translate={expanded ? "no" : undefined} onClick={() => setExpanded(!expanded)}>
            {expanded ? JSON.stringify(value) : "Show full value"}
          </button>
        )}
      </span>
    </div>
  );
}
export default function PacketDetail({
  capture,
  packet,
  reveal,
  onReveal,
  onError,
  onChanged,
  onSend,
  initialTab,
  onTabChange,
  onClose,
  reference,
  onPin,
  onClearReference,
}: {
  reference?: PacketRecord;
  onPin?: () => void;
  onClearReference?: () => void;
  onClose?: () => void;
  capture: string;
  packet?: PacketRecord;
  reveal: boolean;
  onReveal: (v: boolean) => void;
  onError: (e: unknown) => void;
  onChanged: () => void;
  onSend: (p: PacketRecord) => void;
  initialTab?: string;
  onTabChange?: (tab: string) => void;
}) {
  const [tab, setTab] = useState(initialTab || "Overview"),
    [query, setQuery] = useState(""),
    [fieldPage, setFieldPage] = useState(0),
    [links, setLinks] = useState<PacketLink[]>([]),
    [note, setNote] = useState("");
  useEffect(() => onTabChange?.(tab), [tab, onTabChange]);
  useEffect(() => {
    if (initialTab) setTab(initialTab);
  }, [initialTab]);
  const scope = useRef(`${capture}:${packet?.id}:${reveal}`);
  scope.current = `${capture}:${packet?.id}:${reveal}`;
  const loadLinks = async (extract = false) => {
    if (!packet) return;
    const request = scope.current;
    const value = extract
      ? await api.ExtractPacketContent(capture, packet.id)
      : await api.GetPacketContentLinks(capture, packet.id);
    if (scope.current === request) setLinks(value);
  };
  useEffect(() => {
    setNote(packet?.note ?? "");
    setLinks([]);
    setFieldPage(0);
    setQuery("");
  }, [capture, packet?.id]);
  const decoded = useMemo(() => {
    try {
      return parseExactJSON(packet?.decoded || packet?.payload || "{}");
    } catch {
      return packet?.payload;
    }
  }, [packet?.payload, packet?.decoded]);
  const form = useMemo(() => packetForm(decoded), [decoded]);
  const activeTab = tab === "Form" && !form ? "Overview" : tab;
  const bytes = useMemo(() => {
    try {
      return rawBytes(packet?.raw);
    } catch {
      return new Uint8Array();
    }
  }, [packet?.raw]);
  const fields = useMemo(
    () => packetFields(decoded, Number.MAX_SAFE_INTEGER),
    [decoded],
  );
  const filteredFields = useMemo(
    () =>
      fields.filter((f) =>
        `${f.path} ${f.value}`.toLowerCase().includes(query.toLowerCase()),
      ),
    [fields, query],
  );
  if (!packet)
    return (
      <aside className="pi-detail pi-empty">
        <Search size={27} />
        <h3>Inspect a packet</h3>
        <p>
          Select a row to explore decoded fields, wire bytes and related
          content.
        </p>
      </aside>
    );
  const run = async (fn: () => Promise<unknown>) => {
    try {
      await fn();
    } catch (e) {
      onError(e);
    }
  };
  const copy = () =>
    run(() => navigator.clipboard.writeText(JSON.stringify(decoded, null, 2)));
  return (
    <aside className="pi-detail" aria-label="Packet details">
      <header>
        {onClose && (
          <button
            aria-label="Close packet details"
            title="Close details"
            onClick={onClose}
          >
            <X size={15} />
          </button>
        )}
        <div>
          <strong>{packet.name}</strong>
          <span>
            #{packet.id} · 0x{packet.packetId.toString(16)} ·{" "}
            {packetBytes(packet.bytes)}
          </span>
        </div>
        <button
          aria-label="Copy packet JSON"
          title="Copy packet JSON"
          onClick={copy}
        >
          <Copy size={15} />
        </button>
        <button
          aria-label="Bookmark packet"
          aria-pressed={packet.bookmarked}
          title="Bookmark packet"
          onClick={() =>
            run(async () => {
              await api.UpdatePacketCapture({
                capture,
                packet: packet.id,
                annotation: {
                  note: packet.note || "",
                  bookmarked: !packet.bookmarked,
                },
              });
              onChanged();
            })
          }
        >
          <Bookmark
            size={15}
            fill={packet.bookmarked ? "currentColor" : "none"}
          />
        </button>
        <button
          aria-label="Prepare packet send"
          title="Prepare packet send"
          onClick={() => onSend(packet)}
        >
          <Send size={15} />
        </button>
        {onPin && (
          <button
            aria-label="Use packet as comparison reference"
            title="Pin as comparison reference"
            onClick={onPin}
          >
            <GitCompareArrows size={15} />
          </button>
        )}
      </header>
      <div className="pi-detail-privacy">
        <label>
          <input
            type="checkbox"
            checked={reveal}
            onChange={(e) => onReveal(e.target.checked)}
          />
          Show sensitive data
        </label>
        {packet.redacted && <span>Secrets hidden</span>}
      </div>
      {packet.partial && (
        <FeedbackFlyout title="Packet details" tone="warning">
          Partial decode:{" "}
          {packet.reason || "The source did not retain the complete payload."}
        </FeedbackFlyout>
      )}
      <nav className="pi-tabs" aria-label="Packet detail views">
        {[
          "Overview",
          ...(form ? ["Form"] : []),
          "Fields",
          "JSON",
          "Hex",
          "Compare",
          ...(packet.before ? ["Changes"] : []),
          "Content",
          "Notes",
        ].map((t) => (
          <button
            key={t}
            className={activeTab === t ? "active" : ""}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </nav>
      <div className="pi-detail-body" key={packet.id + activeTab}>
        {activeTab === "Compare" && (
          <PacketComparison
            baseline={reference}
            packet={packet}
            onPin={() => onPin?.()}
            onClear={() => onClearReference?.()}
          />
        )}
        {activeTab === "Form" && form && (
          <PacketFormPreview
            key={`${capture}:${packet.id}:${reveal}`}
            form={form}
          />
        )}
        {activeTab === "Overview" && (
          <>
            <dl className="pi-properties">
              <dt>Direction</dt>
              <dd>{packet.direction}</dd>
              <dt>Observation</dt>
              <dd>{packet.stage}</dd>
              <dt>Connection leg</dt>
              <dd>{packet.connectionLeg || "Not recorded in this archive"}</dd>
              <dt>Source</dt>
              <dd>{packet.sourceAddress || "Not recorded"}</dd>
              <dt>Destination</dt>
              <dd>{packet.destinationAddress || "Not recorded"}</dd>
              <dt>Packet header</dt>
              <dd>
                {[...rawBytes(packet.header)]
                  .map((b) => b.toString(16).padStart(2, "0"))
                  .join(" ") || "Not recorded"}
              </dd>
              <dt>Time</dt>
              <dd>{new Date(packet.time).toISOString()}</dd>
              <dt>Action</dt>
              <dd>{packet.action || "Observed"}</dd>
              <dt>Wire bytes</dt>
              <dd>
                {bytes.length
                  ? packetBytes(bytes.length)
                  : packet.redacted
                    ? "Hidden"
                    : "Unavailable"}
              </dd>
            </dl>
            {packet.reason && <p>{packet.reason}</p>}
            {form && (
              <button className="pi-form-open" onClick={() => setTab("Form")}>
                Preview form
              </button>
            )}
            <h4>Decoded payload</h4>
            <JSONTree value={decoded} />
          </>
        )}
        {activeTab === "Fields" && (
          <>
            <input
              aria-label="Find decoded field"
              placeholder="Find field or value"
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setFieldPage(0);
              }}
            />
            <div className="pi-field-list">
              {filteredFields
                .slice(fieldPage * 150, (fieldPage + 1) * 150)
                .map((f) => (
                  <button
                    title="Copy field path and value"
                    key={f.path}
                    onClick={() =>
                      run(() =>
                        navigator.clipboard.writeText(`${f.path} = ${f.value}`),
                      )
                    }
                  >
                    <b translate="no">{f.path}</b>
                    <span translate="no">{f.value}</span>
                  </button>
                ))}
            </div>
            <div className="pi-field-pagination">
              <button
                disabled={!fieldPage}
                onClick={() => setFieldPage((n) => n - 1)}
              >
                Previous
              </button>
              <span>
                {filteredFields.length.toLocaleString(formatLocale())} fields · Page{" "}
                {fieldPage + 1} of{" "}
                {Math.max(1, Math.ceil(filteredFields.length / 150))}
              </span>
              <button
                disabled={(fieldPage + 1) * 150 >= filteredFields.length}
                onClick={() => setFieldPage((n) => n + 1)}
              >
                Next
              </button>
            </div>
          </>
        )}
        {activeTab === "JSON" && (
          <>
            <div className="pi-inline-actions">
              <button onClick={copy}>
                <Copy size={13} />
                Copy JSON
              </button>
            </div>
            <pre className="pi-json-source">
              {JSON.stringify(decoded, null, 2)}
            </pre>
          </>
        )}
        {activeTab === "Hex" && (
          <PacketBytes
            key={`${capture}:${packet.id}:${reveal}`}
            bytes={bytes}
            packet={packet}
            onError={onError}
          />
        )}
        {activeTab === "Changes" && (
          <>
            {packet.before && (
              <>
                <h4>Before rule edit</h4>
                <pre>{packet.before}</pre>
              </>
            )}
          </>
        )}
        {activeTab === "Content" && (
          <>
            <button onClick={() => run(() => loadLinks())}>
              Find linked content
            </button>
            {["playerskin", "playerlist"].includes(
              packet.name.toLowerCase(),
            ) && (
              <button onClick={() => run(() => loadLinks(true))}>
                Extract captured skins
              </button>
            )}
            {links.map((link) => (
              <button
                className="pi-content-link"
                key={link.kind + link.id}
                onClick={() => openPacketContent(link)}
              >
                <div>
                  <strong>{link.name}</strong>
                  <small>
                    {link.kind} · {link.relation}
                  </small>
                </div>
                <ArrowUpRight size={15} />
              </button>
            ))}
            {!links.length && (
              <p>
                Find matching skins and resource packs, or open advertised
                command definitions in Command Explorer.
              </p>
            )}
          </>
        )}
        {activeTab === "Notes" && (
          <>
            <textarea
              aria-label="Packet note"
              rows={8}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Add investigation notes…"
            />
            <button
              onClick={() =>
                run(async () => {
                  await api.UpdatePacketCapture({
                    capture,
                    packet: packet.id,
                    annotation: {
                      note,
                      bookmarked: packet.bookmarked,
                    },
                  });
                  onChanged();
                })
              }
            >
              Save note
            </button>
          </>
        )}
      </div>
    </aside>
  );
}

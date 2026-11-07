import FeedbackFlyout from "./FeedbackFlyout";
import { useEffect, useMemo, useState, useRef } from "react";
import {
  Copy,
  Bookmark,
  Send,
  Search,
  ArrowUpRight,
  Download,
} from "lucide-react";
import PacketFormPreview from "./PacketFormPreview";
import { packetForm } from "./packet-form-model";
import { api } from "./bridge";
import {
  numericBytes,
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
          <b>{name}</b>
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
      <b>{name}</b>:{" "}
      <span className={typeof value === "string" ? "pi-string" : "pi-number"}>
        {(JSON.stringify(value) ?? "null").slice(0, 1000)}
        {(JSON.stringify(value)?.length ?? 0) > 1000 && (
          <button onClick={() => setExpanded(!expanded)}>
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
}: {
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
    [offset, setOffset] = useState(0),
    [fieldPage, setFieldPage] = useState(0),
    [hexSearch, setHexSearch] = useState(""),
    [hexMessage, setHexMessage] = useState(""),
    [little, setLittle] = useState(true),
    [links, setLinks] = useState<PacketLink[]>([]),
    [note, setNote] = useState("");
  useEffect(() => onTabChange?.(tab), [tab, onTabChange]);
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
    setOffset(0);
    setFieldPage(0);
    setHexSearch("");
    setHexMessage("");
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
              <dt>Header</dt>
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
                    <b>{f.path}</b>
                    <span>{f.value}</span>
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
                {filteredFields.length.toLocaleString()} fields · Page{" "}
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
          <>
            <div className="pi-inline-actions">
              <label>
                Offset
                <input
                  aria-label="Byte offset"
                  type="number"
                  value={offset}
                  min={0}
                  max={Math.max(0, bytes.length - 1)}
                  onChange={(e) =>
                    setOffset(Math.max(0, Number(e.target.value)))
                  }
                />
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={little}
                  onChange={(e) => setLittle(e.target.checked)}
                />
                Little endian
              </label>
              <button
                disabled={!bytes.length}
                onClick={() => {
                  const url = URL.createObjectURL(
                    new Blob([new Uint8Array(bytes)]),
                  );
                  const a = document.createElement("a");
                  a.href = url;
                  a.download = `packet-${packet.id}.bin`;
                  a.click();
                  setTimeout(() => URL.revokeObjectURL(url), 1000);
                }}
              >
                <Download size={13} />
                Bytes
              </button>
            </div>
            <form
              className="pi-hex-search"
              onSubmit={(e) => {
                e.preventDefault();
                const query = hexSearch.replace(/\s/g, "");
                if (!query || !/^(?:[0-9a-f]{2})+$/i.test(query)) {
                  setHexMessage("Enter complete hex bytes, e.g. 01 02 ff");
                  return;
                }
                const needle = Uint8Array.from(query.match(/../g)!, (h) =>
                  parseInt(h, 16),
                );
                let found = -1;
                for (let n = 0; n < bytes.length; n++) {
                  const at = (offset + 1 + n) % bytes.length;
                  if (
                    at + needle.length <= bytes.length &&
                    needle.every((b, i) => bytes[at + i] === b)
                  ) {
                    found = at;
                    break;
                  }
                }
                if (found < 0) setHexMessage("Byte sequence not found");
                else {
                  setOffset(found);
                  setHexMessage(`Found at 0x${found.toString(16)}`);
                }
              }}
            >
              <input
                aria-label="Find hex bytes"
                placeholder="Find bytes: 01 02 ff"
                value={hexSearch}
                onChange={(e) => setHexSearch(e.target.value)}
              />
              <button type="submit" disabled={!bytes.length}>
                Find next
              </button>
              <span role="status">{hexMessage}</span>
            </form>
            <div className="pi-field-pagination">
              <button
                disabled={offset < 4096}
                onClick={() => setOffset((n) => Math.max(0, n - 4096))}
              >
                Previous bytes
              </button>
              <span>{bytes.length.toLocaleString()} body bytes retained</span>
              <button
                disabled={Math.floor(offset / 16) * 16 + 4096 >= bytes.length}
                onClick={() => setOffset((n) => Math.floor(n / 16) * 16 + 4096)}
              >
                Next bytes
              </button>
            </div>
            <dl className="pi-byte-numbers">
              {Object.entries(numericBytes(bytes, offset, little)).map(
                ([k, v]) => (
                  <div key={k}>
                    <dt>{k}</dt>
                    <dd>{v}</dd>
                  </div>
                ),
              )}
            </dl>
            {!bytes.length ? (
              <p>
                {packet.redacted
                  ? "Enable sensitive data to reveal these bytes."
                  : "No original wire bytes were supplied by this source."}
              </p>
            ) : (
              <div className="pi-hex">
                <div className="pi-hex-head">
                  Offset 00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F
                </div>
                {Array.from(
                  {
                    length: Math.min(
                      256,
                      Math.ceil(
                        (bytes.length - Math.floor(offset / 16) * 16) / 16,
                      ),
                    ),
                  },
                  (_, row) => {
                    const start = Math.floor(offset / 16) * 16 + row * 16;
                    const chunk = bytes.slice(start, start + 16);
                    return (
                      <div className="pi-hex-line" key={start}>
                        <b>{start.toString(16).padStart(8, "0")}</b>
                        <span>
                          {[...chunk].map((b, i) => (
                            <button
                              key={i}
                              className={start + i === offset ? "selected" : ""}
                              onClick={() => setOffset(start + i)}
                            >
                              {b.toString(16).padStart(2, "0")}
                            </button>
                          ))}
                        </span>
                        <code>
                          {[...chunk]
                            .map((b) =>
                              b >= 32 && b < 127 ? String.fromCharCode(b) : ".",
                            )
                            .join("")}
                        </code>
                      </div>
                    );
                  },
                )}
              </div>
            )}
            <p>
              Showing up to 4,096 bytes from the selected offset. Use Offset or
              export to inspect the full body.
            </p>
          </>
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
                Find matching identifiers in the skin and pack libraries. Worlds
                and resource packs are assembled by their live capture modules.
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

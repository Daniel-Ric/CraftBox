import { formatLocale } from "./i18n";
import { useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, Copy, Download } from "lucide-react";
import { numericBytes, type PacketRecord } from "./packet-inspector-model";
import { packetVarInt, findPacketBytes } from "./packet-workbench";
export default function PacketBytes({
  bytes,
  packet,
  onError,
}: {
  bytes: Uint8Array;
  packet: PacketRecord;
  onError: (error: unknown) => void;
}) {
  const [anchor, setAnchor] = useState(0),
    [end, setEnd] = useState(0),
    [page, setPage] = useState(0),
    [little, setLittle] = useState(true),
    [query, setQuery] = useState(""),
    [mode, setMode] = useState("hex"),
    [message, setMessage] = useState("");
  const start = Math.min(anchor, end),
    stop = Math.max(anchor, end),
    selected = bytes.subarray(start, stop + 1);
  const readings = useMemo(() => {
    const data = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    const base = numericBytes(bytes, start, little);
    const variable = packetVarInt(bytes, start);
    return {
      ...base,
      uint16:
        start + 2 <= bytes.length ? String(data.getUint16(start, little)) : "—",
      int32:
        start + 4 <= bytes.length ? String(data.getInt32(start, little)) : "—",
      uint64:
        start + 8 <= bytes.length
          ? String(data.getBigUint64(start, little))
          : "—",
      VarUInt64: variable?.unsigned || "—",
      ZigZag64: variable?.signed || "—",
      VarIntBytes: variable?.bytes.toString() || "—",
    };
  }, [bytes, start, little]);
  const select = (offset: number, extend = false) => {
    const at = Math.max(0, Math.min(bytes.length - 1, Math.trunc(offset)));
    if (!extend) setAnchor(at);
    setEnd(at);
    setPage(Math.floor(at / 256));
  };
  function find(reverse = false) {
    let needle: Uint8Array;
    if (mode === "hex") {
      const clean = query.replace(/\s/g, "");
      if (!/^(?:[0-9a-f]{2})+$/i.test(clean)) {
        setMessage("Enter complete hex bytes, e.g. 01 02 ff.");
        return;
      }
      needle = Uint8Array.from(clean.match(/../g)!, (v) => parseInt(v, 16));
    } else needle = new TextEncoder().encode(query);
    if (!needle.length) {
      setMessage("Enter text or bytes to find.");
      return;
    }
    const at = findPacketBytes(bytes, needle, start, reverse);
    if (at >= 0) {
      setAnchor(at);
      setEnd(at + needle.length - 1);
      setPage(Math.floor(at / 256));
      setMessage(`Match at 0x${at.toString(16)} · ${needle.length} bytes`);
      return;
    }
    setMessage("No matching bytes in the retained payload.");
  }
  async function copySelection() {
    if (selected.length > 262144) {
      setMessage(
        "Selection is too large to copy. Export selected bytes instead.",
      );
      return;
    }
    try {
      await navigator.clipboard.writeText(
        [...selected].map((b) => b.toString(16).padStart(2, "0")).join(" "),
      );
      setMessage(`${selected.length} bytes copied`);
    } catch (error) {
      onError(error);
    }
  }
  function download(selection: boolean) {
    try {
      const data = selection ? selected : bytes;
      const url = URL.createObjectURL(new Blob([new Uint8Array(data)]));
      const link = document.createElement("a");
      link.href = url;
      link.download = `packet-${packet.id}${selection ? `-${start}-${stop}` : ""}.bin`;
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      onError(error);
    }
  }
  if (!bytes.length)
    return (
      <p>
        {packet.redacted
          ? "Enable sensitive data to reveal these bytes."
          : "No original wire bytes were supplied by this source."}
      </p>
    );
  return (
    <section className="pi-bytes-workbench" aria-label="Packet byte inspector">
      <div className="pi-byte-tools">
        <label>
          Offset
          <input
            type="number"
            aria-label="Byte offset"
            min={0}
            max={bytes.length - 1}
            value={start}
            onChange={(e) => select(Number(e.target.value) || 0)}
          />
        </label>
        <select
          aria-label="Byte order"
          value={little ? "little" : "big"}
          onChange={(e) => setLittle(e.target.value === "little")}
        >
          <option value="little">Little endian</option>
          <option value="big">Big endian</option>
        </select>
        <button
          aria-label="Export packet bytes"
          title="Export all retained body bytes"
          onClick={() => download(false)}
        >
          <Download size={14} />
        </button>
      </div>
      <form
        className="pi-byte-search"
        onSubmit={(e) => {
          e.preventDefault();
          find();
        }}
      >
        <select
          aria-label="Byte search mode"
          value={mode}
          onChange={(e) => {
            setMode(e.target.value);
            setMessage("");
          }}
        >
          <option value="hex">Hex</option>
          <option value="text">UTF-8</option>
        </select>
        <input
          type="search"
          aria-label="Find packet bytes"
          placeholder={mode === "hex" ? "01 02 ff" : "Find text in bytes"}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button
          type="button"
          aria-label="Previous byte match"
          onClick={() => find(true)}
        >
          <ChevronLeft size={13} />
        </button>
        <button type="submit" aria-label="Next byte match">
          <ChevronRight size={13} />
        </button>
      </form>
      <div className="pi-byte-selection">
        <span>
          0x{start.toString(16)}–0x{stop.toString(16)} ·{" "}
          {selected.length.toLocaleString(formatLocale())} selected
        </span>
        <button
          title="Copy selected bytes as hex"
          aria-label="Copy selected bytes"
          onClick={() => void copySelection()}
        >
          <Copy size={13} />
        </button>
        <button
          title="Export selected bytes"
          aria-label="Export selected bytes"
          onClick={() => download(true)}
        >
          <Download size={13} />
        </button>
      </div>
      <div
        className="pi-byte-grid"
        role="group"
        aria-label="Selectable packet bytes"
      >
        {Array.from(
          { length: Math.min(16, Math.ceil((bytes.length - page * 256) / 16)) },
          (_, row) => {
            const offset = page * 256 + row * 16;
            return (
              <div className="pi-byte-line" key={offset}>
                <code>{offset.toString(16).padStart(6, "0")}</code>
                <span>
                  {[...bytes.subarray(offset, offset + 16)].map((b, i) => (
                    <button
                      key={i}
                      title={`Offset ${offset + i} · decimal ${b}`}
                      aria-label={`Byte ${offset + i}: ${b.toString(16).padStart(2, "0")}`}
                      aria-pressed={offset + i >= start && offset + i <= stop}
                      onClick={(e) => select(offset + i, e.shiftKey)}
                    >
                      {b.toString(16).padStart(2, "0")}
                    </button>
                  ))}
                </span>
                <small>
                  {[...bytes.subarray(offset, offset + 16)]
                    .map((b) =>
                      b >= 32 && b < 127 ? String.fromCharCode(b) : "·",
                    )
                    .join("")}
                </small>
              </div>
            );
          },
        )}
      </div>
      <div className="pi-byte-pagination">
        <button disabled={!page} onClick={() => setPage((old) => old - 1)}>
          <ChevronLeft size={12} />
          Previous
        </button>
        <span>
          {page * 256 + 1}–{Math.min((page + 1) * 256, bytes.length)} /{" "}
          {bytes.length.toLocaleString(formatLocale())} bytes
        </span>
        <button
          disabled={(page + 1) * 256 >= bytes.length}
          onClick={() => setPage((old) => old + 1)}
        >
          Next
          <ChevronRight size={12} />
        </button>
      </div>
      <small className="pi-byte-hint">
        Shift-click to select a range. Readings begin at the first selected
        byte.
      </small>
      <dl className="pi-byte-readings">
        {Object.entries(readings).map(([key, value]) => (
          <div key={key}>
            <dt>{key}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      <label className="pi-byte-text">
        Selected UTF-8
        <pre>
          {new TextDecoder().decode(selected.subarray(0, 256))}
          {selected.length > 256 ? "…" : ""}
        </pre>
      </label>
      <small className="pi-byte-hint">
        {bytes.length < packet.bytes
          ? "Only part of the observed payload is retained. "
          : ""}
        Body bytes exclude the Bedrock header. VarInt readings are unsigned /
        ZigZag candidates, not a field-schema mapping.
      </small>
      {message && (
        <p role="status" className="pi-byte-hint">
          {message}
        </p>
      )}
    </section>
  );
}

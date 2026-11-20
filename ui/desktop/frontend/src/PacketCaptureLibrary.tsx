import { formatLocale } from "./i18n";
import { useMemo, useState } from "react";
import {
  Activity,
  ArrowUpRight,
  FileInput,
  FolderOpen,
  Radio,
  Search,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  packetBytes,
  type PacketCapture,
  type PacketTypeStat,
} from "./packet-inspector-model";
export function PacketTypeCatalog({
  open,
  onOpenChange,
  types,
  onType,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  types: PacketTypeStat[];
  onType: (name: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState("count");
  const filtered = useMemo(
    () =>
      [...types]
        .filter((t) =>
          `${t.name} ${t.packetId} 0x${t.packetId.toString(16)}`
            .toLowerCase()
            .includes(query.toLowerCase()),
        )
        .sort((a, b) =>
          sort === "name"
            ? a.name.localeCompare(b.name)
            : sort === "bytes"
              ? b.bytes - a.bytes
              : b.count - a.count,
        ),
    [types, query, sort],
  );
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="pi-dialog pi-type-catalog">
        <DialogHeader>
          <DialogTitle>Observed packet types</DialogTitle>
          <DialogDescription>
            Type counts cover the recorded capture. Select a type to inspect its
            packets.
          </DialogDescription>
        </DialogHeader>
        <div className="pi-catalog-controls">
          <Search size={14} />
          <input
            type="search"
            aria-label="Find observed packet type"
            placeholder="Packet name, ID or hexadecimal ID"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <select
            aria-label="Sort observed packet types"
            value={sort}
            onChange={(e) => setSort(e.target.value)}
          >
            <option value="count">Most packets</option>
            <option value="bytes">Most bytes</option>
            <option value="name">Name</option>
          </select>
        </div>
        <div className="pi-catalog-list">
          <table>
            <thead>
              <tr>
                <th>Packet</th>
                <th>ID</th>
                <th>Count</th>
                <th>Volume</th>
                <th>↑ Server</th>
                <th>↓ Client</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((t) => (
                <tr key={t.name}>
                  <td>
                    <button
                      disabled={t.other}
                      onClick={() => {
                        onType(t.name);
                        onOpenChange(false);
                      }}
                    >
                      {t.name}
                      <ArrowUpRight size={12} />
                    </button>
                  </td>
                  <td>0x{t.packetId.toString(16)}</td>
                  <td>{t.count.toLocaleString(formatLocale())}</td>
                  <td>{packetBytes(t.bytes)}</td>
                  <td>{t.serverbound.toLocaleString(formatLocale())}</td>
                  <td>{t.clientbound.toLocaleString(formatLocale())}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!filtered.length && (
            <p className="pi-empty">
              {types.length
                ? "No matching packet types."
                : "No packet types recorded yet."}
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
export function PacketCaptureLibrary({
  captures,
  active,
  query,
  busy,
  onOpen,
  onImport,
  onManage,
  onSource,
}: {
  captures: PacketCapture[];
  active: string;
  query: string;
  busy: boolean;
  onOpen: (capture: PacketCapture) => void;
  onImport: () => void;
  onManage: () => void;
  onSource: () => void;
}) {
  const rows = [...captures]
    .filter((c) =>
      `${c.name} ${c.source} ${c.notes}`
        .toLowerCase()
        .includes(query.toLowerCase()),
    )
    .sort((a, b) => b.started - a.started);
  return (
    <section
      className="pi-capture-library"
      aria-label="Packet recording library"
    >
      <div className="pi-library-intro">
        <Activity size={24} />
        <div>
          <h2>Inspect a connection. Understand the traffic.</h2>
          <p>Open a recording or capture your next Minecraft session.</p>
        </div>
        <button disabled={busy} onClick={onImport}>
          <FileInput size={14} />
          Import recording
        </button>
        <button onClick={onSource}>
          <Radio size={14} />
          Choose capture source
        </button>
      </div>
      <div className="pi-library-heading">
        <h3>
          Recordings <span>{captures.length}</span>
        </h3>
        <button onClick={onManage}>
          <FolderOpen size={14} />
          Manage recordings
        </button>
      </div>
      {rows.length ? (
        <div className="pi-recordings-table">
          <table>
            <thead>
              <tr>
                <th>Recording</th>
                <th>Started</th>
                <th>Observations</th>
                <th>Payload</th>
                <th>Protocol</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => (
                <tr key={c.id}>
                  <td>
                    <button onClick={() => onOpen(c)}>
                      <span
                        className={
                          active === c.id
                            ? "pi-recording-symbol live"
                            : "pi-recording-symbol"
                        }
                      >
                        <Activity size={17} />
                      </span>
                      <span>
                        <strong>{c.name}</strong>
                        <small>
                          {c.source || "Imported recording"}
                          {c.dropped
                            ? ` · ${c.dropped.toLocaleString(formatLocale())} capture gaps`
                            : ""}
                        </small>
                      </span>
                      <ArrowUpRight size={13} />
                    </button>
                  </td>
                  <td>{new Date(c.started).toLocaleString(formatLocale())}</td>
                  <td>{c.count.toLocaleString(formatLocale())}</td>
                  <td>{packetBytes(c.bytes)}</td>
                  <td>{c.protocol || "Unknown"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="pi-empty">
          <FolderOpen size={30} />
          <h3>{query ? "No matching recordings" : "No recordings yet"}</h3>
          <p>
            {query
              ? "Try another name or source."
              : "Import a CraftBox capture, .pcap2, ProxyPass log or ProxyWeb JSON. Live recordings appear here automatically."}
          </p>
        </div>
      )}
    </section>
  );
}

export type PacketRecord = {
  decoded?: string;
  summary?: string;
  sourceAddress?: string;
  destinationAddress?: string;
  connectionLeg?: string;
  previewToken?: string;
  header?: string;
  id: string;
  time: number;
  packetId: number;
  name: string;
  direction: string;
  stage: string;
  bytes: number;
  raw?: string;
  payload: string;
  before?: string;
  action?: string;
  reason?: string;
  redacted: boolean;
  partial: boolean;
  note?: string;
  bookmarked: boolean;
};
export type PacketCapture = {
  id: string;
  name: string;
  source: string;
  protocol: number;
  started: number;
  ended: number;
  count: number;
  bytes: number;
  diskBytes: number;
  dropped: number;
  notes: string;
  error?: string;
};
export type PacketSettings = {
  recording: boolean;
  intervention: boolean;
  maxDiskMB: number;
  captureWorlds: boolean;
  captureSkins: boolean;
  capturePacks: boolean;
};
export type PacketRule = {
  preMS?: number;
  postMS?: number;
  cooldownMS?: number;
  id: string;
  label: string;
  enabled: boolean;
  name: string;
  direction: string;
  search: string;
  action: string;
  payload: string;
  delayMS: number;
  hits: number;
};
export type PacketStatus = {
  recordingIDs?: string[];
  settings: PacketSettings;
  captures: PacketCapture[];
  active: string;
  connected: boolean;
  generation: string;
  queueDropped: number;
  error: string;
  rules: PacketRule[];
};
export type PacketQuery = {
  capture: string;
  search?: string;
  name?: string;
  direction?: string;
  stage?: string;
  action?: string;
  offset?: number;
  limit?: number;
  before?: number;
  after?: number;
  bookmarked?: boolean;
  reverse?: boolean;
};
export type PacketPage = {
  rows: PacketRecord[];
  total: number;
  capture: PacketCapture;
};
export type PacketEvidence = {
  id: string;
  time: number;
  name: string;
  direction: string;
  summary: string;
};
export type PacketEntity = {
  key?: string;
  session?: string;
  id: string;
  type: string;
  name: string;
  removed: boolean;
  partial: boolean;
  position: [number, number, number];
  positions: [number, number, number][];
  evidence: string[];
};
export type PacketTypeStat = {
  other?: boolean;
  name: string;
  packetId: number;
  count: number;
  bytes: number;
  serverbound: number;
  clientbound: number;
  first: number;
  last: number;
};
export type PacketAnalysis = {
  capture: PacketCapture;
  types: PacketTypeStat[];
  buckets: {
    time: number;
    serverbound: number;
    clientbound: number;
    bytes: number;
  }[];
  timeline: PacketEvidence[];
  entities: PacketEntity[];
  transactions: PacketEvidence[];
  chunks: PacketEvidence[];
  changes: PacketEvidence[];
  partial: boolean;
};
export type PacketLink = {
  kind: string;
  id: string;
  name: string;
  search: string;
  relation: string;
};
export type PacketSend = {
  previewToken?: string;
  generation: string;
  direction: string;
  packetId: number;
  payload: string;
  raw: string;
  useRaw: boolean;
};
export const packetBytes = (n: number) =>
  n >= 1073741824
    ? `${(n / 1073741824).toFixed(2)} GB`
    : n >= 1048576
      ? `${(n / 1048576).toFixed(1)} MB`
      : n >= 1024
        ? `${(n / 1024).toFixed(1)} KB`
        : `${n} B`;
export function parseExactJSON(text: string): unknown {
  return JSON.parse(
    text.replace(
      /"(?:\\.|[^"\\])*"|-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/g,
      (token) =>
        token[0] !== '"' &&
        !/[.eE]/.test(token) &&
        !Number.isSafeInteger(Number(token))
          ? JSON.stringify(token)
          : token,
    ),
  );
}
export function packetFields(value: unknown, max = 2000) {
  const rows: { path: string; value: string }[] = [];
  const visit = (v: unknown, path: string, depth: number) => {
    if (rows.length >= max) return;
    if (v && typeof v === "object" && depth < 64) {
      const entries = Object.entries(v);
      if (!entries.length)
        rows.push({ path, value: Array.isArray(v) ? "[]" : "{}" });
      for (const [k, x] of entries)
        visit(x, path ? `${path}.${k}` : k, depth + 1);
    } else
      rows.push({
        path,
        value: typeof v === "string" ? v : (JSON.stringify(v) ?? String(v)),
      });
  };
  visit(value, "", 0);
  return rows;
}
export function rawBytes(base64 = "") {
  return Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
}
export function numericBytes(bytes: Uint8Array, offset: number, little = true) {
  const data = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  const read = (size: number, fn: () => unknown) =>
    offset + size <= bytes.length ? String(fn()) : "—";
  return {
    uint8: read(1, () => data.getUint8(offset)),
    int16: read(2, () => data.getInt16(offset, little)),
    uint32: read(4, () => data.getUint32(offset, little)),
    int64: read(8, () => data.getBigInt64(offset, little)),
    float32: read(4, () => data.getFloat32(offset, little)),
    float64: read(8, () => data.getFloat64(offset, little)),
  };
}

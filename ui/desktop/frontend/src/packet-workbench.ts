import { formatLocale } from "./i18n";
import {
  type PacketRecord,
  type PacketTypeStat,
} from "./packet-inspector-model";

export type FieldChange = {
  path: string;
  before?: string;
  after?: string;
  kind: "added" | "removed" | "changed";
};
function comparisonFields(text: string) {
  if (text.length > 4 * 1024 * 1024)
    throw new Error(
      "Decoded payload exceeds the 4 MiB comparison limit. Inspect it in JSON or Hex.",
    );
  const transformed = text.replace(
    /"(?:\\.|[^"\\])*"|-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/g,
    (token, offset: number) => {
      let next = offset + token.length;
      while (next < text.length && /\s/.test(text[next])) next++;
      if (token[0] === '"' && text[next] === ":") return token;
      return JSON.stringify({
        scalarType: token[0] === '"' ? "string" : "number",
        scalarText: token[0] === '"' ? JSON.parse(token) : token,
      });
    },
  );
  const value: unknown = JSON.parse(transformed);
  const fields = new Map<string, { value: string; signature: string }>();
  let incomplete = false;
  function visit(node: unknown, path: string, depth: number) {
    if (fields.size >= 4000 || depth > 64) {
      incomplete = true;
      return;
    }
    if (node && typeof node === "object") {
      const scalar = node as { scalarType?: unknown; scalarText?: unknown };
      if (
        (scalar.scalarType === "number" || scalar.scalarType === "string") &&
        typeof scalar.scalarText === "string"
      ) {
        fields.set(path, {
          value:
            scalar.scalarType === "string"
              ? JSON.stringify(scalar.scalarText)
              : scalar.scalarText,
          signature: scalar.scalarType + ":" + scalar.scalarText,
        });
        return;
      }
      const array = Array.isArray(node);
      fields.set(path, {
        value: array ? "[array]" : "{object}",
        signature: array ? "array" : "object",
      });
      for (const [key, child] of Object.entries(node)) {
        if (fields.size >= 4000) {
          incomplete = true;
          break;
        }
        const part = array
          ? `[${key}]`
          : /^[A-Za-z_$][\w$]*$/.test(key)
            ? `${path ? "." : ""}${key}`
            : `[${JSON.stringify(key)}]`;
        visit(child, path + part, depth + 1);
      }
    } else
      fields.set(path, {
        value: JSON.stringify(node),
        signature: typeof node + ":" + JSON.stringify(node),
      });
  }
  visit(value, "", 0);
  return { fields, incomplete };
}
export function comparePacketFields(before: PacketRecord, after: PacketRecord) {
  const incomplete =
    before.partial || after.partial || before.redacted || after.redacted;
  try {
    const left = comparisonFields(before.decoded || before.payload);
    const right = comparisonFields(after.decoded || after.payload);
    const a = left.fields,
      b = right.fields;
    const rows: FieldChange[] = [];
    for (const path of [...new Set([...a.keys(), ...b.keys()])].sort()) {
      if (
        a.has(path) &&
        b.has(path) &&
        a.get(path)!.signature === b.get(path)!.signature
      )
        continue;
      rows.push({
        path,
        before: a.get(path)?.value,
        after: b.get(path)?.value,
        kind: !a.has(path) ? "added" : !b.has(path) ? "removed" : "changed",
      });
    }
    return {
      rows,
      incomplete: incomplete || left.incomplete || right.incomplete,
      error: "",
    };
  } catch (error) {
    return {
      rows: [] as FieldChange[],
      incomplete: true,
      error:
        error instanceof Error && error.message.includes("comparison limit")
          ? error.message
          : "Decoded JSON is unavailable or invalid. Inspect the source payload in JSON or Hex.",
    };
  }
}
export function trafficSummary(types: PacketTypeStat[]) {
  const count = types.reduce((sum, t) => sum + t.count, 0);
  const first = types.length ? Math.min(...types.map((t) => t.first)) : 0;
  const last = types.length ? Math.max(...types.map((t) => t.last)) : 0;
  return {
    count,
    first,
    last,
    bytes: types.reduce((sum, t) => sum + t.bytes, 0),
    serverbound: types.reduce((sum, t) => sum + t.serverbound, 0),
    clientbound: types.reduce((sum, t) => sum + t.clientbound, 0),
    rate: last > first ? ((count - 1) * 1000) / (last - first) : null,
    bucketWidth: Math.max(1000, Math.floor((last - first) / 240)),
  };
}
export const trafficWindowQuery = (start: number, width: number) =>
  `time:>=${Math.floor(start)} time:<${Math.floor(start + width)}`;
export const packetTime = (time: number) =>
  `${new Date(time).toLocaleTimeString(formatLocale(), { hour12: false })}.${String(time % 1000).padStart(3, "0")}`;
export function packetVarInt(bytes: Uint8Array, offset: number) {
  let value = 0n;
  for (let i = 0; i < 10 && offset + i < bytes.length; i++) {
    const byte = bytes[offset + i];
    if (i === 9 && byte > 1) return null;
    value |= BigInt(byte & 0x7f) << BigInt(i * 7);
    if (!(byte & 0x80))
      return {
        unsigned: String(value),
        signed: String((value >> 1n) ^ -(value & 1n)),
        bytes: i + 1,
      };
  }
  return null;
}
export function fillTrafficBuckets(
  buckets: {
    time: number;
    serverbound: number;
    clientbound: number;
    bytes: number;
  }[],
  width: number,
) {
  if (!buckets.length || !Number.isFinite(width) || width <= 0) return [];
  const sorted = [...buckets].sort((a, b) => a.time - b.time);
  const start = sorted[0].time,
    end = sorted[sorted.length - 1].time;
  const indexed = new Map(sorted.map((bucket) => [bucket.time, bucket]));
  return Array.from(
    { length: Math.min(241, Math.round((end - start) / width) + 1) },
    (_, i) => {
      const time = start + i * width;
      return (
        indexed.get(time) || { time, serverbound: 0, clientbound: 0, bytes: 0 }
      );
    },
  );
}
export function findPacketBytes(
  bytes: Uint8Array,
  needle: Uint8Array,
  offset: number,
  reverse = false,
) {
  if (!needle.length || needle.length > bytes.length) return -1;
  const prefix = new Uint32Array(needle.length);
  for (let i = 1, j = 0; i < needle.length; i++) {
    while (j && needle[i] !== needle[j]) j = prefix[j - 1];
    if (needle[i] === needle[j]) j++;
    prefix[i] = j;
  }
  let first = -1,
    last = -1,
    before = -1;
  for (let i = 0, j = 0; i < bytes.length; i++) {
    while (j && bytes[i] !== needle[j]) j = prefix[j - 1];
    if (bytes[i] === needle[j]) j++;
    if (j === needle.length) {
      const at = i - needle.length + 1;
      if (first < 0) first = at;
      last = at;
      if (at < offset) before = at;
      if (!reverse && at > offset) return at;
      j = prefix[j - 1];
    }
  }
  return reverse ? (before >= 0 ? before : last) : first;
}

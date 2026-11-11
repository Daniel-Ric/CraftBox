export type LabFile = {
  path: string;
  hash: string;
  rawHash?: string;
  semantic: string;
  bytes: number;
  kind: string;
  origin: string;
  issue?: string;
  identifiers: string[];
};
export type LabSnapshot = {
  parentId?: string;
  schemaVersion?: string;
  schemaHash?: string;
  schemaObjects?: LabFile[];
  id: string;
  name: string;
  version: string;
  channel: string;
  source: string;
  created: string;
  schemaSource: string;
  files: LabFile[] | null;
  count: number;
  bytes: number;
  archives: number;
  decoded: number;
  unresolved: number;
  issues: string[];
};
export type LabProgress = {
  running: boolean;
  phase: string;
  path: string;
  done: number;
  total: number;
  snapshot: string;
  error: string;
};
export type LabPreview = {
  snapshot?: string;
  decoding?: LabDecoding;
  binary?: LabBinary;
  audio?: string;
  waveform?: number[];
  original?: string;
  canonical?: string;
  file: LabFile;
  text: string;
  image: string;
  truncated: boolean;
};
export type LabChange = {
  category?: string;
  summary?: string;
  fieldCount?: number;
  path: string;
  status: string;
  before?: LabFile;
  after?: LabFile;
};
export type LabDecoding = {
  restored: number;
  unresolved: number;
  schema: string;
  version: string;
  note: string;
};
export type LabSound = {
  name: string;
  rate: number;
  channels: number;
  frames: number;
  duration: number;
  dataOffset: number;
  dataBytes: number;
  codebook?: string;
  pcmHash?: string;
};
export type LabBinary = {
  format: string;
  fields: { name: string; value: string }[];
  samples: LabSound[];
  payloadHash?: string;
  issue?: string;
  limited: boolean;
};
export type LabComparison = {
  before: LabSnapshot;
  after: LabSnapshot;
  beforeDecoding: LabDecoding;
  afterDecoding: LabDecoding;
  changes: LabChange[];
};
export type LabJSONAnalysis = {
  fields: { path: string; status: string; before: string; after: string }[];
  versionOnly: boolean;
  truncated: boolean;
};
export type LabSchema = {
  path: string;
  version: string;
  channel: string;
  count: number;
  source: string;
  hash: string;
};
export type LabQuery = {
  snapshot: string;
  search: string;
  kind: string;
  mode: string;
  offset: number;
  limit: number;
};
export type LabPage = { files: LabFile[]; total: number; truncated: boolean };
export type LabCapture = {
  installation: string;
  source: string;
  name: string;
  version: string;
  channel: string;
  schemas: string;
};
export type LabString = { offset: number; text: string; encoding: string };
export function preferredInstallation(
  items: { id: string; channel: string; location: string }[],
) {
  const available = items.filter((i) => i.location);
  return (
    available.find((i) => i.channel === "preview")?.id ?? available[0]?.id ?? ""
  );
}
export function identifierChanges(before: string[], after: string[]) {
  const a = new Set(before),
    b = new Set(after);
  return {
    added: [...b].filter((x) => !a.has(x)).sort(),
    removed: [...a].filter((x) => !b.has(x)).sort(),
  };
}
export function fileChanges<T extends { path: string; status: string }>(
  changes: T[],
  query: string,
  formatting: boolean,
) {
  const needle = query.toLowerCase();
  return changes
    .filter(
      (c) =>
        (formatting || c.status !== "formatting") &&
        c.path.toLowerCase().includes(needle),
    )
    .sort((a, b) => a.path.localeCompare(b.path));
}
export function labBytes(n: number) {
  if (n < 1024) return `${n} B`;
  if (n < 1048576) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1048576).toFixed(1)} MB`;
}
export function snapshotLabel(s: LabSnapshot) {
  const build = s.source.match(/_(\d+\.\d+\.\d+\.\d+)_/)?.[1];
  return `${s.name}${build && !s.name.includes(build) ? ` · ${build}` : ""}`;
}
export function comparisonOrder(
  selected: LabSnapshot | undefined,
  baseline: LabSnapshot | undefined,
  reverse = false,
) {
  if (!selected || !baseline || selected.id === baseline.id)
    return { before: undefined, after: undefined };
  const numbers = (s: LabSnapshot) => {
    const packed = s.source.match(/_(\d+\.\d+\.\d+\.\d+)_/)?.[1];
    const version = (packed || s.version).match(/^\d+(?:\.\d+)+/)?.[0];
    if (!version) return [];
    const n = version.split(".").map(Number);
    if (packed && n[3] === 0 && n[2] >= 100)
      return [n[0], n[1], Math.floor(n[2] / 100), n[2] % 100];
    return n;
  };
  const a = numbers(baseline),
    b = numbers(selected);
  let order = 0;
  if (a.length && b.length)
    for (let i = 0; i < Math.max(a.length, b.length); i++) {
      order = (a[i] || 0) - (b[i] || 0);
      if (order) break;
    }
  const chronological =
    order < 0 || (order === 0 && baseline.created <= selected.created)
      ? [baseline, selected]
      : [selected, baseline];
  return {
    before: chronological[reverse ? 1 : 0],
    after: chronological[reverse ? 0 : 1],
  };
}
export function snapshotBuild(s: LabSnapshot) {
  return s.source.match(/_(\d+\.\d+\.\d+\.\d+)_/)?.[1] || s.version || s.name;
}

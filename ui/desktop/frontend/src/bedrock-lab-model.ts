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
  file: LabFile;
  text: string;
  image: string;
  truncated: boolean;
};
export type LabChange = {
  path: string;
  status: string;
  before?: LabFile;
  after?: LabFile;
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

export type LocalWorld = {
  id: string;
  name: string;
  folder: string;
  installation: string;
  modified: number;
  icon?: string;
  template: boolean;
  marketplace: boolean;
};
export type LocalWorldIssue = {
  severity: "error" | "warning";
  code: string;
  message: string;
  packId?: string;
};
export type LocalWorldOptions = { includeGlobal: boolean; backupOnly: boolean };
export type LocalWorldPlan = {
  world: LocalWorld;
  packs: {
    id: string;
    name: string;
    version: string;
    kind: string;
    status: "included" | "missing" | "ambiguous" | "protected";
    requiredBy: string;
    embedded: boolean;
  }[];
  issues: LocalWorldIssue[];
  portable: boolean;
  files: number;
  bytes: number;
  fingerprint: string;
};
export type LocalWorldCatalog = { worlds: LocalWorld[]; warning?: string };
export type LocalWorldRequest = {
  worldId: string;
  jobId?: string;
  fingerprint?: string;
  options: LocalWorldOptions;
};
export type LocalWorldResult = {
  path: string;
  bytes: number;
  files: number;
  packs: number;
  portable: boolean;
  issues: LocalWorldIssue[];
};
export type LocalWorldProgress = {
  stage: string;
  completed: number;
  total: number;
  files: number;
  running: boolean;
  jobId: string;
};

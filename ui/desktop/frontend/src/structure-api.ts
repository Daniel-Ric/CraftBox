import { api, type ScenePage } from "./bridge";
export type Vec3 = [number, number, number];
export type Box = { min: Vec3; max: Vec3 };
export type BlockState = {
  name: string;
  states: Record<string, string | number | boolean>;
  version?: number;
  types?: Record<string, string>;
};
export type Region = { id: string; name: string; bounds: Box };
export type StructureDocument = {
  id: string;
  name: string;
  size: Vec3;
  origin: Vec3;
  anchor: Vec3;
  revision: number;
  dirty: boolean;
  undo: boolean;
  redo: boolean;
  path: string;
  palette: BlockState[];
  regions: Region[];
  packs: {
    id: string;
    name: string;
    uuid: string;
    kind: string;
    missing: boolean;
  }[];
  view: Record<string, unknown>;
  volume: number;
  memoryBytes: number;
  blockEntities: number;
  entities: number;
};
export type StructureEdit = {
  kind: string;
  name?: string;
  bounds?: Box[];
  block?: BlockState;
  from?: string;
  matchStates?: boolean;
  destination?: Vec3;
  turns?: number;
  mirror?: string;
  anchor?: Vec3;
  regions?: Region[];
  allowUnsupported?: boolean;
};
export type ExportOptions = {
  entities: boolean;
  inventories: boolean;
  blockData: boolean;
  omitUnknown: boolean;
  packs: boolean;
};
export type Materials = {
  rows: {
    name: string;
    blocks: number;
    items: number;
    stacks: number;
    remainder: number;
    estimated: boolean;
  }[];
  blocks: number;
  unknown: number;
  void: number;
};
export type Difference = {
  position: Vec3;
  kind: string;
  expected: string;
  actual: string;
};
export type Comparison = {
  correct: number;
  missing: number;
  extra: number;
  wrong: number;
  orientation: number;
  unknown: number;
  samples: Difference[];
  truncated: boolean;
};
export type Recovery = {
  id: string;
  name: string;
  modified: number;
  open: boolean;
};
export type Job = { busy: boolean; stage: string; done: number; total: number };
export type Layer = {
  y: number;
  x: number;
  z: number;
  width: number;
  depth: number;
  cells: { layers: [number, number]; known: boolean }[];
};
export type Placement = {
  id: string;
  source: string;
  anchor: Vec3;
  mode: string;
  dimension: number;
  entities: boolean;
  packs: boolean;
  inventories: boolean;
  blockData: boolean;
};
interface StructureAPI {
  NewStructure(name: string, size: Vec3): Promise<StructureDocument>;
  GetStructure(id: string): Promise<StructureDocument>;
  GetStructureBlockCatalog(query: string): Promise<BlockState[]>;
  EditStructure(id: string, e: StructureEdit): Promise<StructureDocument>;
  StructureHistory(id: string, redo: boolean): Promise<StructureDocument>;
  SetStructureView(
    id: string,
    view: Record<string, unknown>,
  ): Promise<StructureDocument>;
  CloseStructure(id: string): Promise<void>;
  GetStructureJob(id: string): Promise<Job>;
  CancelStructureJob(id: string): Promise<void>;
  CancelStructurePreview(id: string, viewer: string): Promise<void>;
  ChooseStructureSource(
    kind: "folder" | "world" | "structure" | "pack" | "pack-folder",
  ): Promise<string>;
  OpenStructurePath(path: string): Promise<StructureDocument>;
  AttachStructurePack(
    id: string,
    path: string,
    replaceID: string,
  ): Promise<StructureDocument>;
  OpenStructureFile(path: string, jobID: string): Promise<StructureDocument>;
  ImportStructureWorld(r: {
    jobId?: string;
    name: string;
    worldId: string;
    path: string;
    bounds: Box;
    dimension: number;
  }): Promise<StructureDocument>;
  ListStructureRecovery(): Promise<Recovery[]>;
  RecoverStructure(id: string): Promise<StructureDocument>;
  SaveStructureDialog(
    id: string,
    project: boolean,
    saveAs: boolean,
    options: ExportOptions,
  ): Promise<StructureDocument>;
  GetStructureScenePage(
    id: string,
    r: Record<string, unknown>,
  ): Promise<ScenePage>;
  GetStructureLayer(
    id: string,
    y: number,
    x: number,
    z: number,
    width: number,
    depth: number,
  ): Promise<Layer>;
  GetStructureBlock(id: string, p: Vec3): Promise<BlockState>;
  GetStructureMaterials(id: string, boxes: Box[]): Promise<Materials>;
  ExtractStructure(
    id: string,
    name: string,
    boxes: Box[],
  ): Promise<StructureDocument>;
  ExportStructureMaterials(id: string, boxes: Box[]): Promise<string>;
  CompareStructures(
    source: string,
    target: string,
    offset: Vec3,
  ): Promise<Comparison>;
  PlaceStructureDocument(
    source: string,
    target: string,
    anchor: Vec3,
    mode: string,
    entities: boolean,
  ): Promise<StructureDocument>;
  ExportStructureWorldDialog(r: Placement): Promise<string>;
}
export const structures = api as unknown as StructureAPI;
export const exportDefaults: ExportOptions = {
  entities: false,
  inventories: true,
  blockData: true,
  omitUnknown: false,
  packs: true,
};
export function normalized(b: Box): Box {
  return {
    min: b.min.map((n, i) => Math.min(n, b.max[i])) as Vec3,
    max: b.min.map((n, i) => Math.max(n, b.max[i])) as Vec3,
  };
}
export function volume(b: Box) {
  const n = normalized(b);
  return n.min.reduce((v, a, i) => v * (n.max[i] - a + 1), 1);
}
export function blockColor(name: string) {
  let h = 2166136261;
  for (const c of name) {
    h = Math.imul(h ^ c.charCodeAt(0), 16777619);
  }
  return `hsl(${(h >>> 0) % 360} 32% 53%)`;
}

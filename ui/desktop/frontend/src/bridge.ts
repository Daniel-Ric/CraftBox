export type Option = {
  name: string;
  description: string;
  flag: string;
  default: string;
  type: string;
  fileType: string;
};
export type CommandInfo = {
  name: string;
  description: string;
  options: Option[];
};
export type World = {
  heightRange?: [number, number];
  spawn?: [number, number, number];
  parentId?: string;
  id: string;
  imagePath: string;
  player: [number, number, number];
  hasPlayer: boolean;
  name: string;
  path: string;
  chunks: number;
  entities: number;
  state: string;
};
export type Pack = {
  id: string;
  name: string;
  version: string;
  path: string;
  size: number;
  received: number;
  icon: string;
  error: string;
  done: boolean;
};
export type Skin = { name: string; image: string };
export type Tile = { worldId?: string; x: number; z: number; image: string };
export type Log = {
  id: number;
  time: string;
  level: string;
  message: string;
  source: string;
};
export type State = {
  activeWorldId?: string;
  revision: number;
  command: string;
  running: boolean;
  stopping: boolean;
  connection: number;
  listenAddr: string;
  localIP: string;
  worlds: World[];
  packs: Pack[];
  skins: Skin[];
  player: [number, number, number];
  hasPlayer: boolean;
  account: string;
  loggedIn: boolean;
  authCode: string;
  authURI: string;
  authenticating: boolean;
  updateVersion: string;
  updateProgress: number;
  error: string;
};
export type Bootstrap = {
  mapWindow?: boolean;
  theme?: string;
  version: string;
  minecraft: string;
  commands: CommandInfo[];
  state: State;
};
export type Source = {
  id: string;
  name: string;
  description: string;
  address: string;
  available: boolean;
};
export type ChunkBounds = {
  minX: number;
  minZ: number;
  maxX: number;
  maxZ: number;
};
export type CreateWorldRegion = {
  worldId: string;
  name: string;
  bounds: ChunkBounds;
  move: boolean;
};
export type WorldRegion = {
  id: string;
  parentId: string;
  name: string;
  bounds: ChunkBounds;
  move: boolean;
};
export interface Backend {
  SaveStudioProject(
    worldId: string,
    name: string,
    document: string,
    saveAs: boolean,
  ): Promise<string>;
  OpenStudioProject(): Promise<OpenedStudioProject>;
  SetStudioProjectDirty(dirty: boolean): Promise<void>;
  ConfirmStudioProjectClose(): Promise<void>;
  OpenRenderWorld(kind: "archive" | "folder"): Promise<RenderWorld>;
  CancelRenderImport(): Promise<void>;
  CloseRenderWorld(id: string): Promise<void>;
  GetRenderScenePage(id: string, page: ScenePageRequest): Promise<ScenePage>;
  SaveRenderPNG(name: string, data: string): Promise<string>;
  SaveCinematicFile(name: string, kind: string, data: string): Promise<string>;
  GetScenePage(worldId: string, page: ScenePageRequest): Promise<ScenePage>;
  GetScene(worldId: string, after: number): Promise<SceneSnapshot>;
  ExportScene(worldId: string): Promise<string>;
  EditWorld(request: {
    worldId: string;
    name?: string;
    spawn?: [number, number, number];
  }): Promise<void>;
  GetBootstrap(): Promise<Bootstrap>;
  Snapshot(): Promise<State>;
  GetMapTiles(): Promise<Tile[]>;
  GetLogs(): Promise<Log[]>;
  StartCommand(request: {
    command: string;
    target: string;
    values: Record<string, string>;
  }): Promise<void>;
  StopCommand(): Promise<void>;
  ListSources(kind: string): Promise<Source[]>;
  Login(): Promise<void>;
  CancelLogin(): Promise<void>;
  Logout(): Promise<void>;
  SwitchAccount(name: string): Promise<void>;
  PickFile(): Promise<string>;
  OpenURL(url: string): Promise<void>;
  OpenMinecraft(): Promise<void>;
  OpenMapWindow(theme: string): Promise<void>;
  CreateWorldRegion(request: CreateWorldRegion): Promise<WorldRegion>;
  OpenResultFolder(path: string): Promise<void>;
  DismissError(): Promise<void>;
  CheckUpdate(): Promise<string>;
  InstallUpdate(): Promise<void>;
}
export type ScenePart = {
  mode: "opaque" | "mask" | "blend" | "unlit" | "billboard" | "unlit-blend";
  positions: number[];
  normals: number[];
  uvs: number[];
  indices: number[];
};
export type SceneChunk = {
  lights?: SceneLight[];
  x: number;
  z: number;
  parts: ScenePart[];
  missingTextures: number;
  approximateBlocks: number;
};
export type SceneLight = {
  position: [number, number, number];
  level: number;
  color: [number, number, number];
};
export type SceneSnapshot = {
  sequence: number;
  reset: boolean;
  chunks: SceneChunk[];
  atlas: string;
  atlasRevision: number;
  source: string;
};
export type ScenePageRequest = {
  terrainOnly?: boolean;
  after: number;
  target: number;
  cursor: [number, number] | null;
  focus: [number, number];
  limit: number;
  atlasRevision: number;
  scope: number;
};
export type RenderWorld = { id: string; name: string; chunks: number };
export type OpenedStudioProject = {
  world: RenderWorld;
  document: string;
  path: string;
};
export type PackedPart = {
  mode: ScenePart["mode"];
  vertices: number;
  indices: number;
  data: string;
};
export type PackedChunk = Omit<SceneChunk, "parts"> & { parts: PackedPart[] };
export type AtlasPatch = { x: number; y: number; size: number; data: string };
export type ScenePage = {
  sequence: number;
  scope: number;
  reset: boolean;
  complete: boolean;
  next: [number, number] | null;
  total: number;
  chunks: PackedChunk[];
  patches: AtlasPatch[];
  atlasSize: number;
  atlasRevision: number;
  source: string;
};
declare global {
  interface Window {
    go?: { desktop: { App?: Backend; MapWindowApp?: Partial<Backend> } };
    runtime?: {
      EventsOn: (
        name: string,
        callback: (value: unknown) => void,
      ) => () => void;
    };
  }
}
export const api = new Proxy({} as Backend, {
  get:
    (_target, key) =>
    (...args: unknown[]) => {
      const backend = window.go?.desktop.MapWindowApp ?? window.go?.desktop.App;
      if (!backend)
        return Promise.reject(
          new Error(
            "The desktop connection is not ready. Launch this interface using the CraftBox executable.",
          ),
        );
      const method = backend[key as keyof Backend] as (
        ...params: unknown[]
      ) => Promise<unknown>;
      return method(...args);
    },
});
export function on<T>(name: string, callback: (data: T) => void) {
  return (
    window.runtime?.EventsOn(name, (value) => callback(value as T)) ??
    (() => {})
  );
}

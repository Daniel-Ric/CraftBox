import type * as PI from "./packet-inspector-model";
import {isExportMethod,trackBackendExport} from "./export-tasks";
import type * as MP from "./marketplace-model";
import type * as BL from "./bedrock-lab-model";
import type {IngameInterfaceSettings} from "./ingame-interface";
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
export type WorldExportSettings={
 name?:string;spawn?:[number,number,number];gameMode?:number;difficulty?:number;time?:number;weather?:"clear"|"rain"|"thunder";
 rules:Partial<Record<"daylightCycle"|"weatherCycle"|"mobSpawning"|"mobGriefing"|"fireTick"|"keepInventory"|"tileDrops"|"entityDrops"|"showCoordinates"|"commandBlocks"|"commands",boolean>>&{randomTickSpeed?:number};
};
export type WorldExportOptions={profile:string;settings:WorldExportSettings;compact?:boolean;dimensions?:WorldDimensionRule[];biomes?:{source:number;target:number}[];playerDimension?:number};
export type WorldDimensionRule={source:number;target:number|null;bounds?:{minX:number;minZ:number;maxX:number;maxZ:number}};
export type WorldExportPlan={dimensions:{id:number;name:string;chunks:number;kept:number}[];chunks:{x:number;z:number;dimension:number;included:boolean}[];keptChunks:number;excludedChunks:number;previewTruncated:boolean;records:number;biomes:{id:number;name:string}[]};
export type WorldCopyRequest={worldId:string;options:WorldExportOptions};
export type WorldCopyResult={path:string;folder:string;chunks:number};
export type WorldExportProgress={stage:string;completed:number;total:number;running:boolean};
export type WorldExportConfiguration={worldId:string;name:string;capturing:boolean;heightRange:[number,number];profiles:{id:string;name:string;description:string}[];options:WorldExportOptions};
export type CapturePoint = {id:string;kind:"mark"|"marker"|"note"|"camera";name:string;icon:string;position:[number,number,number];yaw:number;pitch:number;created:string;elapsed:number};
export type World = {
	 capturePoints?: CapturePoint[];
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
export type LibraryAsset={playerUUID?:string;captureType?:string;id:string;kind:"skins"|"packs";name:string;version:string;uuid:string;minimumVersion:string;source:string;created:string;bytes:number;files:number;favourite:boolean;tags:string[];image:string;texturePath:string;encrypted:boolean;armModel:string;hasGeometry:boolean;hasCape:boolean;hasAnimations:boolean;persona:boolean;width:number;height:number};
export type AssetLibraryState={items:LibraryAsset[];bytes:number;directory:string;equipmentRevision?:string;blockRevision?:string;blockRevisions?:Record<string,string>};
export type PackObject={textureVariantObjects?:Record<string,string>;materialModes?:Record<string,string>;transformation?:Record<string,unknown>;visibility?:{pattern:string;visible:boolean}[];boneMaterials?:{pattern:string;material:string}[]|null;layers?:string[];id:string;name:string;kind:string;definition:string;geometryId:string;geometryFile:string;textures:Record<string,string>;textureOptions:Record<string,string>;material:string;animations:Record<string,string>;warnings:string[]};
export type MotionRuntime={visibility?:{pattern:string;expression:unknown;condition?:unknown}[];previewQueries?:Record<string,number>;aliases:Record<string,string>;animations:Record<string,any>;controllers:Record<string,any>;scripts:Record<string,any>;particles:Record<string,{definition:any;image:string}>;hideCape:boolean};
export type PackObjectPreview={motion?:MotionRuntime;textureAnimation?:{frames:string[];expression:string;scripts:unknown};initialPose?:{name:string;animation:unknown}[];capturedSlot?:string;object:PackObject;geometry:unknown;geometryId:string;textures:Record<string,string>;image:string;warnings:string[]};
export type AssetFile={path:string;bytes:number;hash:string;kind:string};
export type AssetIssue={severity:"info"|"warning"|"error";code:string;path:string;message:string;reference?:string};
export type AssetDetail={partial?:boolean;item:LibraryAsset;files:AssetFile[];issues:AssetIssue[];metadata:Record<string,unknown>};
export type AssetFilePreview={path:string;kind:string;data:string;truncated:boolean};
export type AssetChange={path:string;status:"added"|"removed"|"modified"};
export type AssetStack={conflicts:{path:string;winner:string;packs:string[]}[];issues:AssetIssue[];resolved:Record<string,string>};
export type MinecraftInstallation={id:string;name:string;channel:string;layout:string;storage:string;version:string;gameVersion:string;location:string;application:string;storageAvailable:boolean};
export type Tile = {
  worldId?: string;
  x: number;
  z: number;
  image: string;
  heights?: number[];
  biomes?: number[];
  biomeNames?: Record<string, string>;
  blocks?: string[];
  version?: number;
};
export type MapPageRequest = {
  omitCoverage?: boolean;
  after?: number;
  target?: number;
  cursor?: [number, number] | null;
  pixels?: number;
  y?: number;
  limit?: number;
  bounds?: [number, number, number, number];
};
export type MapTilePage = {
  sequence: number;
  complete: boolean;
  next: [number, number] | null;
  total: number;
  coverage?: [number, number][];
  tiles: Tile[];
};
export type WorldPlacement = {
  worldId: string;
  offset: [number, number, number];
  bounds?: ChunkBounds;
};
export type WorldCompositionPreview = {
  id: string;
  name: string;
  conflicts: number;
  examples: [number, number, number][];
  blocks: number;
  chunks: number;
  blockEntities: number;
  entities: number;
  spawn: [number, number, number];
};
export type Log = {
  id: number;
  time: string;
  level: string;
  message: string;
  source: string;
};
export type TerminalInfo = {
  id: string;
  pid: number;
  shell: string;
  cwd: string;
};
export type TerminalOutput = {
  data: string;
  exited: boolean;
  exitCode: number;
};
export type State = {
  errorConnection?: import("./action-error").ConnectionFailure;
  logSessionID?: string;
  activeWorldId?: string;
  revision: number;
  command: string;
  running: boolean;
  stopping: boolean;
  connection: number;
  listenAddr: string;
	 captureID?:number;
	 minecraftJoinReady?:boolean;
	 expectedJoinAddress?:string;
	 minecraftJoinAddress?:string;
	 joinHostnameMatches?:boolean;
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
  installationId?: string;
  sceneBudgetBytes?: number;
  mapWindow?: boolean;
  theme?: string;
  version: string;
  minecraft: string;
  commands: CommandInfo[];
  state: State;
};
export type Source = {
  profileId?: string;
  profileName?: string;
  onDemand?: boolean;
  id: string;
  name: string;
  description: string;
  address: string;
  available: boolean;
  version?: string;
  protocol?: number;
  players?: number;
  maxPlayers?: number;
  playerCountError?: string;
  host?: string;
  joinability?: string;
  route?: string;
  hardcore?: boolean;
  editorWorld?: boolean;
  crossPlayDisabled?: boolean;
  startsAt?: string;
  endsAt?: string;
  probe?: SourceProbe;
};
export type MinecraftSourceProfile = {
  id: string;
  name: string;
  channel: string;
  path: string;
  count: number;
  skipped: number;
  warning?: string;
};
export type SourceProbe = {
  discovery?: "http" | "session";
  networkProtocol?: string;
  endpoint: string;
  reachable: boolean;
  name: string;
  protocol?: number;
  version?: string;
  level?: string;
  players?: number;
  maxPlayers?: number;
  gameType?: number;
  onlineAuth?: boolean;
  selfSignedAuth?: boolean;
  editor?: boolean;
  hardcore?: boolean;
  transportLayer?: number;
  connection?: number;
  dataVersion?: number;
  compatible?: boolean;
  latencyMs: number;
  transport: "not-tested" | "connected" | "failed";
  transportError?: string;
  error?: string;
  checkedAt: string;
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
  GetBedrockSnapshots():Promise<BL.LabSnapshot[]>;
  GetBedrockSnapshot(id:string):Promise<BL.LabSnapshot>;
  QueryBedrockFiles(query:BL.LabQuery):Promise<BL.LabPage>;
  PreviewBedrockFile(id:string,path:string):Promise<BL.LabPreview>;
  ExportBedrockAudio(id:string,path:string):Promise<string>;
  AnalyzeBedrockJSON(before:string,after:string):Promise<BL.LabJSONAnalysis>;
  CompareBedrockSnapshots(before:string,after:string):Promise<BL.LabChange[]>;
  CompareBedrockVersions(before:string,after:string):Promise<BL.LabComparison>;
  ClearBedrockAnalysis():Promise<void>;
  GetBedrockIdentifierReport(before:string,after:string):Promise<{added:string[];removed:string[];referencesAdded:string[];referencesRemoved:string[];unresolved:number}>;
  RedecodeBedrockSnapshot(id:string,schemas:string):Promise<void>;
  GetBedrockLabProgress():Promise<BL.LabProgress>;
  CaptureBedrockSnapshot(request:BL.LabCapture):Promise<void>;
  CancelBedrockLab():Promise<void>;
  SelectBedrockFolder(purpose:string):Promise<string>;
  DeleteBedrockSnapshot(id:string):Promise<void>;
  GetBedrockSchemas():Promise<BL.LabSchema[]>;
  GetBedrockServerReleases():Promise<{channel:string;version:string;url:string}[]>;
  PrepareBedrockSchemas(request:{version:string;channel:string}):Promise<void>;
  ValidateBedrockSnapshot(id:string):Promise<{checked:number;issues:string[]}>;
  ExportBedrockSnapshot(id:string):Promise<string>;
  ImportBedrockPack(id:string,path:string):Promise<LibraryAsset[]>;
  SearchBedrockExecutable(installation:string,query:string):Promise<BL.LabString[]>;
  PrepareBedrockExecutable(installation:string):Promise<string>;
  SelectBedrockExecutable():Promise<string>;
  SearchBedrockBinary(path:string,query:string):Promise<BL.LabString[]>;
  PreviewPickedBlock(id: string, object: string, states: Record<string,unknown>): Promise<PackObjectPreview>;
  GetPreparedAssetObjects(id: string): Promise<PackObject[]>;
  GetPreparedAssetDetail(id: string): Promise<AssetDetail>;
  GetIngameHotbarMessages(): Promise<boolean>;
  GetIngameInterfaceSettings():Promise<IngameInterfaceSettings>;
  SetIngameInterfaceSettings(settings:IngameInterfaceSettings):Promise<void>;
  SetIngameHotbarMessages(enabled: boolean): Promise<void>;
  ListMarketplaceSources(environment:string):Promise<MP.MarketDiscovery>;
  GetMarketplaceCreatorImages(environment:string):Promise<Record<string,{url:string;collectionId:string}>>;
  SearchMarketplace(query:MP.MarketQuery):Promise<MP.MarketPage>;
  GetMarketplaceItem(source:string,id:string,environment:string):Promise<MP.MarketItem>;
  ResolveMarketplaceReference(ref:MP.MarketFavorite):Promise<MP.MarketItem>;
  GetMarketplaceFavorites(environment:string):Promise<MP.MarketFavorite[]>;
  SetMarketplaceFavorite(source:string,id:string,enabled:boolean,environment:string):Promise<void>;
  GetMarketplaceFavoriteItems(environment:string):Promise<MP.MarketPage>;
  GetMarketplaceAssetMatches(asset:string):Promise<MP.MarketFavorite[]>;
  GetMarketplacePlayerCounts(environment:string):Promise<Record<string,number>>;
  OpenMarketplaceItem(source:string,id:string,environment:string):Promise<void>;
  JoinMarketplaceSource(source:string,environment:string):Promise<void>;
  ExportMarketplaceItems(selection:MP.MarketFavorite[],format:"json"|"csv"):Promise<string>;
  GetPacketInspector():Promise<PI.PacketStatus>;
  ConfigurePacketInspector(settings:PI.PacketSettings):Promise<void>;
  QueryPackets(query:PI.PacketQuery):Promise<PI.PacketPage>;
  GetPacketDetail(capture:string,id:string,revealSecrets:boolean):Promise<PI.PacketRecord>;
  GetPacketAnalysis(query:PI.PacketQuery):Promise<PI.PacketAnalysis>;
  GetPacketTypes(capture:string):Promise<PI.PacketTypeStat[]>;
  ImportPacketCapture(path:string):Promise<PI.PacketCapture>;
  ExportPacketCapture(query:PI.PacketQuery,revealSecrets:boolean):Promise<string>;
  UpdatePacketCapture(update:{capture:string;name?:string;notes?:string;packet?:string;annotation?:{note:string;bookmarked:boolean;color?:string}}):Promise<void>;
  DeletePacketCapture(id:string):Promise<void>;
  SetPacketRules(rules:PI.PacketRule[]):Promise<void>;
  GetPacketRules():Promise<PI.PacketRule[]>;
  PreviewPacketSend(request:PI.PacketSend):Promise<PI.PacketRecord>;
  SendInspectorPacket(request:PI.PacketSend):Promise<PI.PacketRecord>;
  GetPacketContentLinks(capture:string,id:string):Promise<PI.PacketLink[]>;
  ExtractPacketContent(capture:string,id:string):Promise<PI.PacketLink[]>;
  GetAssetLibrary():Promise<AssetLibraryState>;
  ImportAssets(kind:string,paths:string[]):Promise<LibraryAsset[]>;
  ImportAssetFolder(kind:string):Promise<LibraryAsset[]>;
  ImportAssetBytes(kind:string,name:string,data:string):Promise<LibraryAsset[]>;
  SaveAssetTexture(id:string,path:string,name:string,data:string):Promise<LibraryAsset[]>;
  SetAssetArmModel(id:string,model:string):Promise<void>;
  GetAssetDetail(id:string):Promise<AssetDetail>;
GetSkinAppearance(id:string):Promise<{bodyId:string;variants:LibraryAsset[];attachments:PackObjectPreview[];warnings:string[]}>;
GetAssetObjects(id:string):Promise<PackObject[]>;
PreviewAssetObject(id:string,objectID:string,variant:string):Promise<PackObjectPreview>;
  RecoverCapturedAssets():Promise<{recovered:number;remaining:number;warnings:string[]}>;
  ReadAssetFile(id:string,path:string):Promise<AssetFilePreview>;
  UpdateAsset(id:string,name:string,favourite:boolean,tags:string[]):Promise<void>;
  DeleteAssets(ids:string[]):Promise<void>;
  CompareAssets(left:string,right:string):Promise<AssetChange[]>;
  InspectAssetStack(ids:string[]):Promise<AssetStack>;
  GetMinecraftInstallations():Promise<{installations:MinecraftInstallation[];warning:string}>;
  ImportMinecraftAssets(id:string):Promise<LibraryAsset[]>;
  InstallAssetsInMinecraft(ids:string[],installation:string):Promise<string>;
  OpenMinecraftInstallation(id:string):Promise<void>;
  OpenAssetFolder(id:string):Promise<void>;
  ExportAssets(ids:string[],name:string,format:string):Promise<string>;
  SaveAssetModel(name:string,data:string,format:string):Promise<string>;
  GetAssetModelSources(id:string,objectID:string):Promise<{documents:Record<string,unknown>;animations:Record<string,unknown>;controllers:Record<string,any>;aliases:Record<string,string>;scripts:Record<string,any>;warnings?:string[]}>;
  ApplyAssetStack(worldID:string,ids:string[]):Promise<void>;
  TraceSceneResource(worldID:string,position:[number,number,number],face:number):Promise<{position:[number,number,number];block:string;states:Record<string,unknown>;face:string;texture:string;pack:string;found:boolean;note:string}>;
  CreateRenderRegion(request:{worldId:string;name:string;bounds:ChunkBounds}):Promise<{world:RenderWorld;path:string}>;
    ExportModel(request: {worldId:string;name:string;format:string;scale:number;origin:string;upAxis:string;textures:boolean;bounds?:ChunkBounds}):Promise<string>;
  SaveImage(request:{name:string;format:string;data:string;quality:number;background:string}):Promise<string>;
  ExportRenderGLB(worldId: string): Promise<string>;
  GetModelExportProgress(): Promise<{
    running: boolean;
    done: number;
    total: number;
    bytes: number;
    stage: string;
  }>;
  CancelGLBExport(): Promise<void>;
  SaveStudioProjectForTab(tabID:string,worldId:string,name:string,document:string,saveAs:boolean):Promise<string>;
  SetDocumentCloseManaged(managed:boolean):Promise<void>;
  ReleaseDocumentTab(tabID:string):Promise<void>;
  SaveWorkspaceProject(tabID:string,name:string,document:string,saveAs:boolean):Promise<string>;
  SaveStudioProject(
    worldId: string,
    name: string,
    document: string,
    saveAs: boolean,
  ): Promise<string>;
  OpenStudioProject(): Promise<OpenedStudioProject>;
  SetStudioProjectDirty(dirty: boolean): Promise<void>;
  ConfirmStudioProjectClose(): Promise<void>;
  ConfirmSessionClose(): Promise<void>;
  OpenRenderWorldForTab(tabID:string,kind:"archive"|"folder"):Promise<RenderWorld>;
  CancelRenderImportForTab(tabID:string):Promise<void>;
  SwitchRenderDimensionForTab(tabID:string,id:string,dimension:number):Promise<RenderWorld>;
  OpenRenderWorld(kind: "archive" | "folder"): Promise<RenderWorld>;
  SwitchRenderDimension(id:string,dimension:number):Promise<RenderWorld>;
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
    export?:WorldExportOptions;
  }): Promise<void>;
  GetWorldExportConfig(worldId:string):Promise<WorldExportConfiguration>;
  PreviewWorldExport(request:WorldCopyRequest):Promise<WorldExportPlan>;
  GetWorldExportCatalog(worldId:string):Promise<WorldExportPlan>;
  ExportWorldCopy(request:WorldCopyRequest):Promise<WorldCopyResult>;
  GetLocalWorlds():Promise<import('./local-worlds-model').LocalWorldCatalog>;
  PickLocalWorldFolder():Promise<import('./local-worlds-model').LocalWorldCatalog>;
  PickLocalWorldPackFolder(id:string):Promise<void>;
  OpenLocalWorldFolder(id:string):Promise<void>;
  InspectLocalWorld(request:import('./local-worlds-model').LocalWorldRequest):Promise<import('./local-worlds-model').LocalWorldPlan>;
  ExportLocalWorld(request:import('./local-worlds-model').LocalWorldRequest):Promise<import('./local-worlds-model').LocalWorldResult|null>;
  GetLocalWorldExportProgress():Promise<import('./local-worlds-model').LocalWorldProgress>;
  CancelLocalWorldExportJob(jobID:string):Promise<void>;
  GetWorldExportProgress():Promise<WorldExportProgress>;
  CancelWorldExport():Promise<void>;
  DeleteCapturedWorld(worldId: string): Promise<void>;
  GetBootstrap(): Promise<Bootstrap>;
  Snapshot(): Promise<State>;
  GetMapTiles(): Promise<Tile[]>;
  GetMapPage(id: string, request: MapPageRequest): Promise<MapTilePage>;
  SaveMapPNG(name: string, data: string): Promise<string>;
  PrepareWorldComposition(request: {
    name: string;
    parts: WorldPlacement[];
  }): Promise<WorldCompositionPreview>;
  SaveWorldComposition(id: string, allowConflicts: boolean): Promise<World>;
  CancelComposition(): Promise<void>;
  DiscardComposition(id: string): Promise<void>;
  GetCompositionProgress(): Promise<{
    stage?: string;
    running: boolean;
    done: number;
    total: number;
  }>;
  GetLogs(): Promise<Log[]>;
  GetLogStorage(): Promise<LogStorage>;
  ReadSavedLog(id: string, name: string): Promise<SavedLogPreview>;
  SetLogPolicy(policy: LogPolicy): Promise<void>;
  DeleteLogSession(id: string): Promise<void>;
  DeleteClosedLogSessions(): Promise<void>;
  CleanupSavedLogs(): Promise<void>;
  CreateDiagnosticSnapshot(): Promise<void>;
  OpenSavedLog(id: string, name: string): Promise<void>;
  OpenSessionLog(id: string): Promise<void>;
  OpenLogFolder(): Promise<void>;
  ReportFrontendError(message: string, stack: string): Promise<void>;
  CreateTerminal(request: {
    cols: number;
    rows: number;
    cwd: string;
  }): Promise<TerminalInfo>;
  ValidateLocalPath(path: string, kind: "file" | "directory"): Promise<void>;
  ReadTerminal(id: string): Promise<TerminalOutput>;
  WriteTerminal(id: string, data: string): Promise<void>;
  ResizeTerminal(id: string, cols: number, rows: number): Promise<void>;
  CloseTerminal(id: string): Promise<void>;
  SaveConsoleLog(name: string, text: string): Promise<string>;
  StartCommand(request: {
    command: string;
    target: string;
    values: Record<string, string>;
  }): Promise<void>;
  StopCommand(): Promise<void>;
  ListSources(kind: string): Promise<Source[]>;
  GetMinecraftServerSources(): Promise<{profiles: MinecraftSourceProfile[]; sources: Source[]}>;
  InspectSource(target: string, connection: boolean): Promise<SourceProbe>;
  GetAccountProfile(): Promise<AccountProfile>;
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
  CheckUpdateDetails(channel: "stable" | "preview"): Promise<ReleaseCheck>;
  InstallUpdate(): Promise<void>;
}
export type ReleaseCheck = {
 currentVersion:string; channel:"stable"|"preview"; checkedAt:string; available:boolean;
 release?:{version:string;name:string;notes:string;releaseURL:string;publishedAt:string;prerelease:boolean;assetName:string;size:number;installable:boolean;unavailableReason?:string};
};
export type ScenePart = {
  mode:
    | "opaque"
    | "mask"
    | "blend"
    | "water"
    | "unlit"
    | "billboard"
    | "unlit-blend";
  positions: number[];
  normals: number[];
  uvs: number[];
  indices: number[];
};
export type SceneChunk = {
  signals?: Record<string,number>;
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
	previewStep?:0|4;
  bounds?:ChunkBounds;
  terrainOnly?: boolean;
  after: number;
  target: number;
  cursor: [number, number] | null;
  focus: [number, number];
  limit: number;
  atlasRevision: number;
  scope: number;
};
export type RenderWorld = { capturePoints?:CapturePoint[];id: string; name: string; chunks: number;dimension?:number;dimensions?:{id:number;name:string;chunks:number;kept:number}[] };
export type OpenedStudioProject = {
 structure?:import("./structure-api").StructureDocument;
 workspaceId?:string;
  world: RenderWorld;
  document: string;
  path: string;
};
export type PackedPart = {
  blockLight?: string;
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
export type LogPolicy = { maxBytes: number; maxAgeDays: number; maxSessions: number; level: string };
export type SavedLogFile = { name: string; bytes: number; modified: string };
export type SavedLogSession = { id: string; started: string; ended?: string; version: string; role: string; pid: number; clean: boolean; status: string; active: boolean; bytes: number; files: SavedLogFile[] };
export type LogStorage = { directory: string; policy: LogPolicy; bytes: number; reservedBytes: number; sessions: SavedLogSession[]; warning?: string };
export type SavedLogPreview = { text: string; truncated: boolean };
export type AccountProfile = {
  gamertag: string;
  classicGamertag?: string;
  xuid: string;
};
export const api = new Proxy({} as Backend, {
  get:
    (_target, key) =>
    (...args: unknown[]) => {
      const backend = window.go?.desktop.MapWindowApp ?? window.go?.desktop.App;
      if (!backend)
        return Promise.reject(new Error("Desktop unavailable. Open CraftBox."));
      const method = backend[key as keyof Backend] as (
        ...params: unknown[]
      ) => Promise<unknown>;
      if(typeof method!=="function")return Promise.reject(new Error("This operation is unavailable in this window."));
      if(isExportMethod(String(key))){
        const jobID=["ExportModel","ExportWorldCopy","ExportLocalWorld"].includes(String(key))?crypto.randomUUID():undefined;
        const params=jobID?[{...(args[0] as object),jobId:jobID},...args.slice(1)]:args;
        return trackBackendExport(String(key),()=>method.apply(backend,params),backend as unknown as Record<string,((...args:unknown[])=>Promise<unknown>)|undefined>,jobID);
      }
      return method.apply(backend,args);
    },
});
export function on<T>(name: string, callback: (data: T) => void) {
  return (
    window.runtime?.EventsOn(name, (value) => callback(value as T)) ??
    (() => {})
  );
}

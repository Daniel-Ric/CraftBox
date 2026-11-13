import { api, type AssetLibraryState } from "./bridge";
import { exportTasks } from "./export-tasks";

type LibraryKind = "packs" | "skins";
const cached = new Map<LibraryKind, AssetLibraryState>();
const pending = new Map<LibraryKind, Promise<AssetLibraryState>>();
const visible = new Set<LibraryKind>();
let revision = 0;

export function cachedAssetLibrary(kind: LibraryKind) {
  return cached.get(kind);
}

export function refreshAssetLibrary(kind: LibraryKind, showProgress = true) {
  const active = pending.get(kind);
  if (active) {
    if (showProgress) showLibraryProgress(kind, active);
    return active;
  }
  const load = async () => {
    let result: AssetLibraryState;
    let requested: number;
    do {
      requested = revision;
      result = await api.GetAssetLibraryForKind(kind);
    } while (requested !== revision);
    cached.set(kind, result);
    return result;
  };
  const request = load().finally(() => {
    pending.delete(kind);
    visible.delete(kind);
  });
  pending.set(kind, request);
  if (showProgress) showLibraryProgress(kind, request);
  return request;
}

function showLibraryProgress(kind: LibraryKind, request: Promise<AssetLibraryState>) {
  if (visible.has(kind)) return;
  visible.add(kind);
  const task = exportTasks.begin(kind === "packs" ? "Load resource library" : "Load player skin library");
  task.update(kind === "packs" ? "Reading resource packs…" : "Reading player skins…");
  void request.then(() => task.complete(), error => task.fail(error));
}

export function invalidateAssetLibrary() {
  revision++;
}

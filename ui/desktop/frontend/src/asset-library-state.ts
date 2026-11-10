import { api, type AssetLibraryState } from "./bridge";

let cached: AssetLibraryState | undefined;
let pending: Promise<AssetLibraryState> | undefined;
let revision = 0;

export function cachedAssetLibrary() {
  return cached;
}

export function refreshAssetLibrary() {
  if (pending) return pending;
  pending = (async () => {
    let result: AssetLibraryState;
    let requested: number;
    do {
      requested = revision;
      result = await api.GetAssetLibrary();
    } while (requested !== revision);
    cached = result;
    return result;
  })().finally(() => {
    pending = undefined;
  });
  return pending;
}

export function invalidateAssetLibrary() {
  revision++;
}

import { parseSequence, type Sequence } from "./cinematic";
const format = "craftbox-cinematic-draft";
export function isAudioDraft(text: string | null) {
  try {
    return !!text && JSON.parse(text)?.format === format;
  } catch {
    return false;
  }
}
export function hasAudioDraft(key: string) {
  try {
    return isAudioDraft(localStorage.getItem(key));
  } catch {
    return false;
  }
}
function database(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (typeof indexedDB === "undefined") {
      reject(
        new Error(
          "Local audio storage is unavailable. Save the project to a file.",
        ),
      );
      return;
    }
    const request = indexedDB.open("craftbox-cinematic", 1);
    request.onupgradeneeded = () => request.result.createObjectStore("drafts");
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}
export async function saveAudioDraft(
  key: string,
  project: Sequence,
  isCurrent = () => true,
) {
  const db = await database();
  try {
    if (!isCurrent()) return;
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction("drafts", "readwrite");
      tx.objectStore("drafts").put(JSON.stringify(project), key);
      tx.oncomplete = () => resolve();
      tx.onerror = tx.onabort = () => reject(tx.error);
    });
    if (isCurrent())
      localStorage.setItem(key, JSON.stringify({ format, version: 1 }));
  } finally {
    db.close();
  }
}
export async function loadAudioDraft(key: string): Promise<Sequence> {
  const db = await database();
  try {
    const text = await new Promise<string>((resolve, reject) => {
      const request = db.transaction("drafts").objectStore("drafts").get(key);
      request.onsuccess = () =>
        typeof request.result === "string"
          ? resolve(request.result)
          : reject(new Error("The audio draft is missing from local storage."));
      request.onerror = () => reject(request.error);
    });
    return parseSequence(text);
  } finally {
    db.close();
  }
}

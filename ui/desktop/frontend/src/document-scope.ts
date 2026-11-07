import { createContext, useContext } from "react";
import type { StudioSession } from "./studio-document";
export type WorkspaceDocument = {
  format: "craftbox-workspace";
  version: 1;
  kind: "packs" | "skins" | "packets";
  source: { id: string; name: string };
  view: Record<string, unknown>;
};
export type DocumentScopeValue = {
  id: string;
  sourceId?: string;
  path?: string;
  onSaved?: (document: WorkspaceDocument, path: string) => void;
  active: boolean;
  busy?: boolean;
  initial?: WorkspaceDocument;
  onOpen?: (
    id: string,
    name: string,
    permanent?: boolean,
    deferUntilIdle?: boolean,
  ) => void;
  register?: (session: StudioSession | undefined) => void;
};
export const DocumentScope = createContext<DocumentScopeValue>({
  id: "",
  active: true,
});
export const useDocumentScope = () => useContext(DocumentScope);
export function parseWorkspaceDocument(raw: string): WorkspaceDocument {
  if (raw.length > 32 * 1024 * 1024)
    throw Error("Workspace project exceeds 32 MiB.");
  const d = JSON.parse(raw);
  if (
    d?.format !== "craftbox-workspace" ||
    d.version !== 1 ||
    !["packs", "skins", "packets"].includes(d.kind) ||
    typeof d.source?.id !== "string" ||
    !d.source.id ||
    typeof d.source?.name !== "string" ||
    !d.view ||
    typeof d.view !== "object" ||
    Array.isArray(d.view)
  )
    throw Error("Invalid CraftBox workspace project.");
  if (d.source.id.length > 512 || d.source.name.length > 512)
    throw Error("Invalid project source.");
  const v = d.view as Record<string, unknown>;
  for (const key of [
    "outer",
    "showCape",
    "rotate",
    "walking",
    "wireframe",
    "useGeometry",
    "effects",
    "manualPreview",
    "groupSounds",
    "bookmarks",
    "follow",
    "playback",
  ])
    if (v[key] !== undefined && typeof v[key] !== "boolean")
      throw Error("Invalid project setting: " + key);
  for (const key of [
    "search",
    "name",
    "direction",
    "stage",
    "tab",
    "selected",
    "arm",
    "file",
    "object",
    "variant",
    "fileQuery",
    "fileKind",
    "shape",
    "animationName",
    "detailTab",
  ])
    if (
      v[key] !== undefined &&
      (typeof v[key] !== "string" || (v[key] as string).length > 8192)
    )
      throw Error("Invalid project setting: " + key);
  for (const [key, min, max] of [
    ["light", 0, 20],
    ["offset", 0, 1e12],
    ["cursor", 0, 1e18],
    ["speed", 0.01, 100],
    ["textureZoom", 0.01, 1000],
  ] as const)
    if (
      v[key] !== undefined &&
      (typeof v[key] !== "number" ||
        !Number.isFinite(v[key]) ||
        (v[key] as number) < min ||
        (v[key] as number) > max)
    )
      throw Error("Invalid project setting: " + key);
  const vectors = (m: unknown) =>
    !!m &&
    typeof m === "object" &&
    !Array.isArray(m) &&
    Object.values(m).length <= 4096 &&
    Object.values(m).every(
      (a) =>
        Array.isArray(a) &&
        a.length === 3 &&
        a.every(
          (n) =>
            typeof n === "number" && Number.isFinite(n) && Math.abs(n) <= 3e7,
        ),
    );
  if (v.camera !== undefined) {
    const c = v.camera as Record<string, unknown>;
    if (!vectors({ position: c?.position, target: c?.target }))
      throw Error("Invalid saved camera.");
  }
  if (v.pose !== undefined) {
    const pose = v.pose as Record<string, unknown>;
    if (
      !pose ||
      typeof pose !== "object" ||
      ["head", "leftArm", "rightArm", "leftLeg", "rightLeg"].some(
        (k) =>
          typeof pose[k] !== "number" ||
          !Number.isFinite(pose[k]) ||
          Math.abs(pose[k] as number) > 1e6,
      )
    )
      throw Error("Invalid saved pose.");
    for (const key of ["axes", "positions", "targets", "poles"])
      if (pose[key] !== undefined && !vectors(pose[key]))
        throw Error("Invalid saved pose vectors.");
  }
  if (v.wireframeOptions !== undefined) {
    const w = v.wireframeOptions as Record<string, unknown>;
    if (
      !w ||
      typeof w !== "object" ||
      Array.isArray(w) ||
      Object.keys(w).length > 32
    )
      throw Error("Invalid wireframe settings.");
    for (const [key, value] of Object.entries(w)) {
      if (
        typeof value === "number" &&
        (!Number.isFinite(value) || Math.abs(value) > 1e6)
      )
        throw Error("Invalid wireframe setting: " + key);
      if (
        ["color", "hiddenColor", "vertexColor"].includes(key) &&
        (typeof value !== "string" || !/^#[a-f0-9]{6}$/i.test(value))
      )
        throw Error("Invalid wireframe color.");
    }
  }
  if (v.textureDrafts !== undefined) {
    const drafts = v.textureDrafts as Record<string, unknown>;
    if (
      !drafts ||
      typeof drafts !== "object" ||
      Array.isArray(drafts) ||
      Object.keys(drafts).length > 128 ||
      Object.entries(drafts).some(
        ([path, png]) =>
          path.length > 2048 ||
          typeof png !== "string" ||
          !png.startsWith("data:image/png;base64,"),
      )
    )
      throw Error("Invalid texture drafts.");
  }
  if (
    v.audioSelection !== undefined &&
    (!Array.isArray(v.audioSelection) ||
      v.audioSelection.length > 64 ||
      v.audioSelection.some(
        (path) => typeof path !== "string" || path.length > 2048,
      ))
  )
    throw Error("Invalid audio selection.");
  if (v.audioStates !== undefined) {
    const states = v.audioStates as Record<string, any>;
    const inRange = (n: unknown, min: number, max: number) =>
      typeof n === "number" && Number.isFinite(n) && n >= min && n <= max;
    if (
      !states ||
      typeof states !== "object" ||
      Array.isArray(states) ||
      Object.keys(states).length > 128
    )
      throw Error("Invalid audio states.");
    for (const state of Object.values(states)) {
      if (
        !state ||
        typeof state !== "object" ||
        !Array.isArray(state.clips) ||
        state.clips.length > 64 ||
        typeof state.loop !== "boolean" ||
        !inRange(state.speed, 0.25, 2) ||
        !inRange(state.master, 0, 1) ||
        !inRange(state.zoom, 1, 240) ||
        ["position", "rangeStart", "rangeEnd"].some(
          (k) => !inRange(state[k], 0, 1e7),
        )
      )
        throw Error("Invalid audio preview settings.");
      for (const clip of state.clips)
        if (
          !clip ||
          typeof clip.path !== "string" ||
          clip.path.length > 2048 ||
          !inRange(clip.start, 0, 7200) ||
          !inRange(clip.trimStart, 0, 601) ||
          !inRange(clip.trimEnd, clip.trimStart, 601) ||
          !inRange(clip.gain, -60, 6) ||
          !inRange(clip.pan, -1, 1) ||
          typeof clip.mute !== "boolean" ||
          typeof clip.solo !== "boolean"
        )
          throw Error("Invalid audio clip.");
    }
  }
  return d;
}

export function documentLabel(name: string, path?: string) {
  return (
    path
      ?.split(/[\\/]/)
      .pop()
      ?.replace(/\.craftbox(?:-workspace)?$/i, "") ||
    name ||
    "Untitled"
  );
}

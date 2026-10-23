export type StudioProject = {
  id: string;
  name: string;
  page: "render" | "cinematic" | "worlds" | "packs" | "skins";
  worldId: string;
  openKind?: "archive" | "folder";
  sourceKind?: "archive" | "folder";
  edge: number;
  aspect: number;
  look: string;
  duration: number;
  fps: number;
  address: string;
  updated: number;
};
export const projectDefaults = {
  name: "Untitled",
  page: "render",
  worldId: "",
  edge: 1920,
  aspect: 16 / 9,
  look: "island",
  duration: 12,
  fps: 24,
  address: "",
} as const;
export const recentKey = "craftbox-recent-projects";
export function loadRecent(): StudioProject[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(recentKey) || "[]");
    return Array.isArray(value)
      ? value
          .filter(
            (v): v is StudioProject =>
              !!v &&
              typeof v.id === "string" &&
              typeof v.name === "string" &&
              ["render", "cinematic", "worlds", "packs", "skins"].includes(
                v.page,
              ) &&
              [v.edge, v.aspect, v.duration, v.fps, v.updated].every(
                Number.isFinite,
              ),
          )
          .slice(0, 12)
      : [];
  } catch {
    return [];
  }
}
export function rememberProject(project: StudioProject) {
  const recent = [
    { ...project, openKind: undefined, updated: Date.now() },
    ...loadRecent().filter((p) => p.id !== project.id),
  ].slice(0, 12);
  try {
    localStorage.setItem(recentKey, JSON.stringify(recent));
  } catch {

  }
  return recent;
}

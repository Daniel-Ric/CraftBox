import {loadWorkspacePreferences} from "./workspace-preferences";
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
  nameMode?: "auto" | "custom";
  sourceName?: string;
  pinned?: boolean;
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
export function newProjectDefaults(){const p=loadWorkspacePreferences();const [w,h]=p.outputAspect.split(":").map(Number);return {...projectDefaults,edge:p.outputEdge,aspect:w/h}}
export const projectLabels = {
  render: "Still render",
  cinematic: "Camera sequence",
  worlds: "World capture",
  packs: "Resource packs",
  skins: "Player skins",
};
export type RecentProjectAction =
  | { type: "rename"; name: string }
  | { type: "pin" | "duplicate" | "remove" }
  | { type: "automatic"; sourceName?: string };
const cleanName = (name: string) =>
  name
    .replace(/§[0-9a-fk-or]/gi, "")
    .replace(/[\u0000-\u001f\u007f]/g, " ")
    .trim()
    .slice(0, 100);
const genericProjectName = (name: string) =>
  /^(?:untitled|new project|world-\d+)?$/i.test(name.trim());
export function automaticProjectName(
  project: Pick<StudioProject, "name" | "nameMode">,
) {
  return (
    project.nameMode === "auto" ||
    (project.nameMode !== "custom" && genericProjectName(project.name))
  );
}
export function suggestProjectName(
  project: Pick<StudioProject, "page" | "address" | "updated"> & {
    sourceName?: string;
  },
) {
  let source = cleanName(project.sourceName || "");
  const file = project.address?.match(
    /(?:^|[\\/])([^\\/]+)\.(?:pcap2|mcworld)$/i,
  );
  if (!source && file) source = cleanName(file[1]);
  if (!source && project.address) {
    try {
      source = new URL(
        project.address.includes("://")
          ? project.address
          : `http://${project.address}`,
      ).hostname;
    } catch {
      source = cleanName(project.address);
    }
  }
  const type = projectLabels[project.page];
  if (source) return cleanName(`${source} – ${type}`);
  const date = new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(project.updated));
  return `${type} – ${date}`;
}
export function uniqueProjectName(name: string, projects: StudioProject[]) {
  const names = new Set(projects.map((p) => p.name.toLocaleLowerCase()));
  if (!names.has(name.toLocaleLowerCase())) return name;
  for (let n = 2; ; n++) {
    const next = `${name.slice(0, 94)} (${n})`;
    if (!names.has(next.toLocaleLowerCase())) return next;
  }
}
export function withProjectSource(
  project: StudioProject,
  worldId: string,
  sourceName?: string,
  sourceKind?: "archive" | "folder",
): StudioProject {
  const next = {
    ...project,
    worldId,
    sourceName: cleanName(sourceName || project.sourceName || ""),
    sourceKind: sourceKind ?? project.sourceKind,
  };
  if (
    automaticProjectName(project) &&
    (genericProjectName(project.name) ||
      next.sourceName !== project.sourceName ||
      worldId !== project.worldId)
  ) {
    next.name = suggestProjectName(next);
    next.nameMode = "auto";
  }
  return next;
}

export function withProjectAddress(
  project: StudioProject,
  address: string,
  sourceName?: string,
): StudioProject {
  const changed = address.trim() !== project.address;
  const next = {
    ...project,
    address: address.trim(),
    sourceName: cleanName(
      sourceName || (changed ? "" : project.sourceName) || "",
    ),
    worldId: changed ? "" : project.worldId,
  };
  if (automaticProjectName(project)) next.name = suggestProjectName(next);
  return next;
}
function normalizeRecent(projects: StudioProject[]) {
  const result: StudioProject[] = [];
  for (const project of projects) {
    if (result.some((p) => p.id === project.id)) continue;
    const next = { ...project, openKind: undefined };
    if (automaticProjectName(next) && genericProjectName(next.name)) {
      next.name = uniqueProjectName(suggestProjectName(next), result);
      next.nameMode = "auto";
    } else if (next.nameMode === "auto")
      next.name = uniqueProjectName(next.name, result);
    result.push(next);
  }
  return result
    .sort(
      (a, b) =>
        Number(!!b.pinned) - Number(!!a.pinned) || b.updated - a.updated,
    )
    .slice(0, loadWorkspacePreferences().recentLimit);
}
function saveRecent(projects: StudioProject[]) {
  const recent = normalizeRecent(projects);
  try {
    localStorage.setItem(recentKey, JSON.stringify(recent));
  } catch {}
  return recent;
}
export function loadRecent(): StudioProject[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(recentKey) || "[]");
    return Array.isArray(value)
      ? normalizeRecent(
          value.filter(
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
          ),
        )
      : [];
  } catch {
    return [];
  }
}
export function rememberProject(project: StudioProject) {
  const others = loadRecent().filter((p) => p.id !== project.id);
  const name = automaticProjectName(project)
    ? uniqueProjectName(
        genericProjectName(project.name)
          ? suggestProjectName(project)
          : project.name,
        others,
      )
    : project.name;
  const recent = [
    { ...project, name, openKind: undefined, updated: Date.now() },
    ...others,
  ];
  return saveRecent(recent);
}

export function manageRecentProject(id: string, action: RecentProjectAction) {
  const recent = loadRecent(),
    project = recent.find((p) => p.id === id);
  if (!project) return recent;
  if (action.type === "remove")
    return saveRecent(recent.filter((p) => p.id !== id));
  if (action.type === "duplicate") {
    const copy: StudioProject = {
      ...project,
      id: crypto.randomUUID(),
      name: uniqueProjectName(`${project.name.slice(0, 90)} copy`, recent),
      nameMode: "custom",
      pinned: false,
      updated: Date.now(),
    };
    const settings = localStorage.getItem(
      `craftbox-render-project:${project.id}`,
    );
    if (settings)
      localStorage.setItem(`craftbox-render-project:${copy.id}`, settings);
    return saveRecent([copy, ...recent]);
  }
  let next = project;
  if (action.type === "rename") {
    const name = cleanName(action.name);
    if (!name) throw new Error("Enter a project name.");
    next = { ...project, name, nameMode: "custom" };
  } else if (action.type === "pin")
    next = { ...project, pinned: !project.pinned };
  else if (action.type === "automatic")
    next = withProjectSource(
      { ...project, name: "", nameMode: "auto" },
      project.worldId,
      action.sourceName,
    );
  return saveRecent(recent.map((p) => (p.id === id ? next : p)));
}

export function syncRecentProjectSources(
  sources: { id: string; name: string }[],
) {
  const recent = loadRecent();
  let changed = false;
  const next = recent.map((p) => {
    const source = sources.find((w) => w.id === p.worldId);
    if (!source || source.name === p.sourceName) return p;
    changed = true;
    return withProjectSource(p, p.worldId, source.name);
  });
  return changed ? saveRecent(next) : recent;
}

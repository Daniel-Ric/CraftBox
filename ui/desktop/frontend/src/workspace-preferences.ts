import { useEffect, useState } from "react";

export const preferenceDefaults = {
  discordEnabled: false,
  discordDetails: "Using CraftBox",
  discordState: "In the workspace",
  discordAutomatic: true,
  discordShowElapsed: true,
  discordLargeText: "CraftBox",
  discordSmallImage: "",
  discordSmallText: "",
  discordGitHubButton: true,
  discordButton1Label: "",
  discordButton1Url: "",
  destinationHostname: false,
  autoOpenMinecraft: true,
  ingameHotbarMessages: true,
  ingameConfirmDisconnect: true,
  ingameModelPickFeedback: true,
  ingameModelPickFocus: true,
  ingameMenuColors: true,
  autoOpenCaptureMap: true,
  lanDiscovery: true,
  localConnectionPort: 19132,
  showWelcome: true,
  density: "compact" as "compact" | "comfortable",
  reducedMotion: false,
  uiScale: 100,
  tooltips: true,
  tooltipDelay: 600,
  draggableDialogs: true,
  startupPage: "worlds" as
    | "worlds"
    | "packs"
    | "skins"
    | "packets"
    | "marketplace"
    | "render"
    | "cinematic"
    | "structures"
    | "bedrock",
  rememberWorkspace: false,
  recentLimit: 48,
  renderQuality: "balanced" as "fast" | "balanced" | "fine",
  outputEdge: 1920,
  outputAspect: "16:9" as "16:9" | "1:1" | "4:3" | "9:16",
  transparentOutput: false,
  sceneBudgetMB: 512,
  flightSpeed: 6,
  flightFov: 70,
  flightSensitivity: 0.0015,
  flightSmoothing: 0.18,
  flightLevel: false,
  consoleFontSize: 12,
  consoleWrap: true,
  consoleTimestamps: true,
  consoleLineNumbers: true,
  consoleFollow: true,
  automaticUpdates: true,
  updateChannel: "stable" as "stable" | "preview",
  updateIntervalHours: 24,
};
export type WorkspacePreferences = typeof preferenceDefaults;
export type PreferenceKey = keyof WorkspacePreferences;
export const preferenceCategories = [
  {
    id: "appearance",
    name: "Appearance",
    description: "Interface brightness, size and feedback",
  },
  {
    id: "connection",
    name: "Connection",
    description: "Minecraft connection and local hostname",
  },
  {
    id: "ingame",
    name: "Minecraft interface",
    description: "CraftBox notifications and ingame menus",
  },
  {
    id: "workspace",
    name: "Workspace",
    description: "Startup, recent projects and dialogs",
  },
  {
    id: "editor",
    name: "3D editor & export",
    description: "Defaults for new projects and scene memory",
  },
  {
    id: "navigation",
    name: "Navigation",
    description: "First-person camera defaults",
  },
  {
    id: "console",
    name: "Console",
    description: "Defaults for new console tabs",
  },
  {
    id: "diagnostics",
    name: "Diagnostics & logs",
    description: "Saved sessions, crash reports and storage limits",
  },
  {
    id: "version",
    name: "Version & updates",
    description: "GitHub releases and update checks",
  },
  {
    id: "discord",
    name: "Discord",
    description: "Your CraftBox activity, on your Discord profile",
  },
] as const;
export type PreferenceCategory = (typeof preferenceCategories)[number]["id"];
type Setting = {
  key: PreferenceKey;
  category: PreferenceCategory;
  label: string;
  description: string;
  min?: number;
  max?: number;
  maxLength?: number;
  step?: number;
  choices?: readonly (readonly [string | number, string])[];
};
export const preferenceSettings: Setting[] = [
  {
    key: "discordEnabled",
    category: "discord",
    label: "Discord Rich Presence",
    description:
      "Share your activity while CraftBox and Discord desktop are open. Ready to use with the built-in CraftBox connection.",
  },
  {
    key: "discordDetails",
    category: "discord",
    label: "Details",
    description: "First status line on your profile (up to 128 characters).",
  },
  {
    key: "discordAutomatic",
    category: "discord",
    label: "Automatic activity",
    description:
      "Show the current capture mode. Server addresses, world names and account names are never shared.",
  },
  {
    key: "discordState",
    category: "discord",
    label: "Custom status",
    description:
      "Second status line when automatic activity is off (up to 128 characters).",
  },
  {
    key: "discordShowElapsed",
    category: "discord",
    label: "Show elapsed time",
    description: "Display how long CraftBox has been open.",
  },
  {
    key: "discordLargeText",
    category: "discord",
    label: "Large image hover text",
    description: "Text shown when someone hovers over the large image.",
  },
  {
    key: "discordSmallImage",
    category: "discord",
    label: "Status badge URL",
    description:
      "Optional public image link shown as a small badge on the CraftBox logo. Leave empty to hide it.",
    maxLength: 300,
  },
  {
    key: "discordSmallText",
    category: "discord",
    label: "Small image hover text",
    description: "Text shown when someone hovers over the small badge.",
  },
  {
    key: "discordGitHubButton",
    category: "discord",
    label: "Show CraftBox GitHub button",
    description:
      "Show a fixed CraftBox on GitHub button linking to github.com/Daniel-Ric/CraftBox.",
  },
  {
    key: "discordButton1Label",
    category: "discord",
    label: "Custom button label",
    description:
      "Caption for your one custom button, for example Community. Leave both custom fields empty to hide it.",
    maxLength: 32,
  },
  {
    key: "discordButton1Url",
    category: "discord",
    label: "Custom button URL",
    description: "Public http(s) link opened by your custom button.",
    maxLength: 512,
  },
  {
    key: "ingameHotbarMessages",
    category: "ingame",
    label: "CraftBox hotbar messages",
    description:
      "Show CraftBox notifications above the inventory. Also available in /cb menu > Interface settings.",
  },
  {
    key: "ingameConfirmDisconnect",
    category: "ingame",
    label: "Confirm disconnecting from the in-game menu",
    description:
      "Ask before ending the Minecraft connection from the CraftBox in-game menu.",
  },
  {
    key: "ingameModelPickFeedback",
    category: "ingame",
    label: "F8 model-picking hints",
    description:
      "Show feedback when F8 cannot identify a model or block. Picking and opening models continues when hints are hidden.",
  },
  {
    key: "ingameModelPickFocus",
    category: "ingame",
    label: "Bring CraftBox to the front after F8",
    description:
      "Show and restore CraftBox after picking a model. Disable to keep playing while the model opens in the background.",
  },
  {
    key: "ingameMenuColors",
    category: "ingame",
    label: "Colored menu text",
    description:
      "Use colors and formatting in CraftBox menus. Saved names and form input remain unchanged.",
  },
  {
    key: "destinationHostname",
    category: "connection",
    label: "Destination hostname",
    description:
      "Use the destination hostname for new connections. Requires Windows permission.",
  },
  {
    key: "autoOpenMinecraft",
    category: "connection",
    label: "Open Minecraft automatically",
    description: "Open the connection when the local server is ready.",
  },
  {
    key: "autoOpenCaptureMap",
    category: "connection",
    label: "Open capture map automatically",
    description: "Open the 2D map when a world capture starts.",
  },
  {
    key: "lanDiscovery",
    category: "connection",
    label: "Show on local network",
    description:
      "Advertise local connections on LAN. Unavailable with destination hostname.",
  },
  {
    key: "localConnectionPort",
    category: "connection",
    label: "Local connection port",
    description:
      "Default port for new connections. A custom listen address takes priority.",
    min: 1,
    max: 65535,
    step: 1,
  },
  {
    key: "density",
    category: "appearance",
    label: "Controls",
    description: "Spacing in interface controls.",
    choices: [
      ["compact", "Compact"],
      ["comfortable", "Comfortable"],
    ],
  },
  {
    key: "uiScale",
    category: "appearance",
    label: "Interface scale",
    description: "Scale the complete workspace, including dialogs.",
    choices: [
      [90, "90%"],
      [100, "100%"],
      [110, "110%"],
      [125, "125%"],
    ],
  },
  {
    key: "reducedMotion",
    category: "appearance",
    label: "Reduce motion",
    description: "Disable interface animations and transitions.",
  },
  {
    key: "tooltips",
    category: "appearance",
    label: "Show tooltips",
    description: "Show descriptions for toolbar icons.",
  },
  {
    key: "tooltipDelay",
    category: "appearance",
    label: "Tooltip delay (ms)",
    description: "Wait before showing a toolbar tooltip.",
    min: 100,
    max: 2000,
    step: 50,
  },
  {
    key: "showWelcome",
    category: "workspace",
    label: "Show welcome dialog on startup",
    description: "Show recent projects and options for starting a project.",
  },
  {
    key: "startupPage",
    category: "workspace",
    label: "Startup workspace",
    description: "Workspace shown behind the welcome dialog.",
    choices: [
      ["worlds", "Worlds"],
      ["packs", "Resource packs"],
      ["skins", "Player skins"],
      ["packets", "Packet Inspector"],
      ["marketplace", "Marketplace"],
      ["bedrock", "Archive"],
      ["render", "Render editor"],
      ["cinematic", "Cinematic editor"],
      ["structures", "Structure Studio"],
    ],
  },
  {
    key: "rememberWorkspace",
    category: "workspace",
    label: "Remember last workspace",
    description: "Open the workspace you used last time.",
  },
  {
    key: "recentLimit",
    category: "workspace",
    label: "Recent project limit",
    description: "How many recent projects to keep, including pinned projects.",
    min: 8,
    max: 48,
    step: 1,
  },
  {
    key: "draggableDialogs",
    category: "workspace",
    label: "Move dialogs by title bar",
    description: "Drag dialog headers; double-click to center them.",
  },
  {
    key: "renderQuality",
    category: "editor",
    label: "Preview quality",
    description:
      "Initial quality for new render projects; saved projects keep their settings.",
    choices: [
      ["fast", "Fast"],
      ["balanced", "Balanced"],
      ["fine", "Fine"],
    ],
  },
  {
    key: "outputEdge",
    category: "editor",
    label: "Output long edge (px)",
    description: "Initial image size for new projects.",
    choices: [
      [1280, "1280"],
      [1920, "1920"],
      [2560, "2560"],
      [3840, "3840"],
    ],
  },
  {
    key: "outputAspect",
    category: "editor",
    label: "Output aspect ratio",
    description: "Initial frame shape for new projects.",
    choices: [
      ["16:9", "16:9"],
      ["1:1", "Square"],
      ["4:3", "4:3"],
      ["9:16", "Portrait 9:16"],
    ],
  },
  {
    key: "transparentOutput",
    category: "editor",
    label: "Transparent background",
    description: "Start new render projects with transparent image output.",
  },
  {
    key: "sceneBudgetMB",
    category: "editor",
    label: "3D geometry memory (MB)",
    description:
      "Geometry budget for subsequently loaded scenes. Larger scenes need more memory.",
    choices: [
      [128, "128 MB"],
      [256, "256 MB"],
      [512, "512 MB"],
      [768, "768 MB"],
      [1024, "1024 MB"],
    ],
  },
  {
    key: "flightSpeed",
    category: "navigation",
    label: "Flight speed",
    description: "Initial movement speed in blocks per second.",
    min: 1,
    max: 100,
    step: 1,
  },
  {
    key: "flightFov",
    category: "navigation",
    label: "Field of view",
    description: "Initial first-person camera field of view in degrees.",
    min: 30,
    max: 110,
    step: 1,
  },
  {
    key: "flightSensitivity",
    category: "navigation",
    label: "Mouse sensitivity",
    description: "Initial mouse-look sensitivity.",
    min: 0.0001,
    max: 0.01,
    step: 0.0001,
  },
  {
    key: "flightSmoothing",
    category: "navigation",
    label: "Camera smoothing",
    description: "Initial first-person movement smoothing.",
    min: 0,
    max: 0.95,
    step: 0.01,
  },
  {
    key: "flightLevel",
    category: "navigation",
    label: "Level movement",
    description: "Move along the horizontal plane while flying.",
  },
  {
    key: "consoleFontSize",
    category: "console",
    label: "Console font size",
    description: "Text size in newly created console tabs.",
    min: 10,
    max: 20,
    step: 1,
  },
  {
    key: "consoleWrap",
    category: "console",
    label: "Wrap long lines",
    description: "Wrap console messages to fit the panel width.",
  },
  {
    key: "consoleTimestamps",
    category: "console",
    label: "Show timestamps",
    description: "Include message times in new console tabs.",
  },
  {
    key: "consoleLineNumbers",
    category: "console",
    label: "Show line numbers",
    description: "Display line numbers in new console tabs.",
  },
  {
    key: "consoleFollow",
    category: "console",
    label: "Follow new messages",
    description: "Automatically scroll new console tabs to the latest message.",
  },
  {
    key: "automaticUpdates",
    category: "version",
    label: "Check for updates automatically",
    description:
      "Check GitHub when CraftBox starts and periodically while open. Installation always requires your action.",
  },
  {
    key: "updateChannel",
    category: "version",
    label: "Update channel",
    description: "Stable releases, or stable releases plus previews.",
    choices: [
      ["stable", "Stable"],
      ["preview", "Include previews"],
    ],
  },
  {
    key: "updateIntervalHours",
    category: "version",
    label: "Check interval",
    description: "Minimum time between automatic GitHub checks.",
    choices: [
      [6, "Every 6 hours"],
      [24, "Daily"],
      [168, "Weekly"],
    ],
  },
];
const storageKey = "craftbox-workspace-preferences";
export function normalizePreferences(value: unknown): WorkspacePreferences {
  const source =
    value && typeof value === "object"
      ? (value as Record<string, unknown>)
      : {};
  const result = { ...preferenceDefaults };
  for (const setting of preferenceSettings) {
    const key = setting.key;
    const v = source[key];
    const d = preferenceDefaults[key];
    if (typeof v !== typeof d) continue;
    if (
      key === "discordSmallImage" &&
      typeof v === "string" &&
      v.trim() &&
      !/^https?:\/\/[^\s/]+(?:\/[^\s]*)?$/.test(v.trim())
    )
      continue;
    if (key === "localConnectionPort" && !Number.isInteger(v)) continue;
    if (setting.choices && !setting.choices.some((c) => c[0] === v)) continue;
    if (
      typeof v === "number" &&
      (!Number.isFinite(v) ||
        v < (setting.min ?? -Infinity) ||
        v > (setting.max ?? Infinity))
    )
      continue;
    Object.assign(result, { [key]: v });
  }
  if (
    !result.discordButton1Label.trim() &&
    !result.discordButton1Url.trim() &&
    typeof source.discordButton2Label === "string" &&
    typeof source.discordButton2Url === "string"
  ) {
    result.discordButton1Label = source.discordButton2Label;
    result.discordButton1Url = source.discordButton2Url;
  }
  return result;
}
export function loadWorkspacePreferences(): WorkspacePreferences {
  try {
    return normalizePreferences(
      JSON.parse(localStorage.getItem(storageKey) || "null"),
    );
  } catch {
    return { ...preferenceDefaults };
  }
}
export function applyWorkspacePreferences(p: WorkspacePreferences) {
  document.documentElement.dataset.density = p.density;
  document.documentElement.dataset.motion = p.reducedMotion
    ? "reduced"
    : "system";
  document.documentElement.style.setProperty("zoom", String(p.uiScale / 100));
  document.documentElement.style.setProperty(
    "--workspace-scale",
    String(p.uiScale / 100),
  );
  document.documentElement.dataset.dialogDragging = String(p.draggableDialogs);
}
export function saveWorkspacePreferences(p: WorkspacePreferences) {
  const normalized = normalizePreferences(p);
  localStorage.setItem(storageKey, JSON.stringify(normalized));
  applyWorkspacePreferences(normalized);
  window.dispatchEvent(new Event("craftbox-preferences-changed"));
  return normalized;
}
export function useWorkspacePreferences() {
  const [value, setValue] = useState(loadWorkspacePreferences);
  useEffect(() => {
    const changed = () => setValue(loadWorkspacePreferences());
    window.addEventListener("craftbox-preferences-changed", changed);
    return () =>
      window.removeEventListener("craftbox-preferences-changed", changed);
  }, []);
  return value;
}

import { useEffect, useRef, useState } from "react";

export type ShortcutScope =
  | "app"
  | "editor"
  | "cinematic"
  | "camera"
  | "structure"
  | "rig"
  | "packets"
  | "documents"
  | "audio"
  | "welcome"
  | "console"
  | "texture";
export type ShortcutAction = {
  id: string;
  label: string;
  scope: ShortcutScope;
  defaults: string[];
  repeat?: boolean;
  inputs?: boolean;
};
export const shortcutScopeLabels: Record<ShortcutScope, string> = {
  app: "Application",
  editor: "Scene editor",
  cinematic: "Cinematic editor",
  camera: "Free camera",
  structure: "Structure Studio",
  rig: "Skin rig",
  packets: "Packet Inspector",
  texture: "Texture preview",
  documents: "Document tabs",
  audio: "Audio studio",
  welcome: "Start screen",
  console: "Console",
};
const action = (
  id: string,
  label: string,
  scope: ShortcutScope,
  defaults: string[] = [],
  extra: Partial<ShortcutAction> = {},
): ShortcutAction => ({ id, label, scope, defaults, ...extra });
export const shortcutActions: ShortcutAction[] = [
  action("search", "Find command", "app", ["Ctrl+KeyK"], { inputs: true }),
  action("new", "New project", "app", ["Ctrl+KeyN"], { inputs: true }),
  action("settings", "Preferences", "app", ["Ctrl+Comma"], { inputs: true }),
  action("help", "Controls & shortcuts", "app", ["F1"]),
  action("logs", "Toggle console", "app", ["Ctrl+Backquote"], { inputs: true }),
  action("explorer", "Command Explorer", "app", ["Ctrl+Shift+KeyJ"]),
  action("home", "Start screen", "app", ["Ctrl+Shift+KeyH"]),
  action("account", "Microsoft account", "app", ["Ctrl+Shift+KeyA"]),
  action("theme", "Toggle light / dark theme", "app", ["Ctrl+Alt+KeyT"]),
  action("log-folder", "Open log folder", "app"),
  action("about", "About CraftBox", "app"),
  action("updates", "Check for updates", "app"),
  ...[
    ["packs", "import-files", "Import resource pack files"],
    ["packs", "import-folder", "Import resource pack folder"],
    ["skins", "import-files", "Import player skin files"],
    ["packs", "browse-minecraft", "Browse installed Minecraft packs"],
    ["skins", "browse-minecraft", "Browse installed Minecraft skins"],
    ["packets", "import-files", "Import packet recording"],
  ].map(([page, command, label]) =>
    action(`library.${page}.${command}`, label, "app"),
  ),
  ...[
    "worlds",
    "packs",
    "skins",
    "packets",
    "structures",
    "bedrock",
    "marketplace",
    "render",
    "cinematic",
  ].map((id, i) =>
    action(
      `workspace.${id}`,
      [
        "Worlds",
        "Resource packs",
        "Player skins",
        "Packet Inspector",
        "Structure Studio",
        "Archive",
        "Marketplace",
        "Render editor",
        "Cinematic editor",
      ][i],
      "app",
      [`Ctrl+Shift+Digit${i + 1}`],
    ),
  ),
  ...[
    ["open-archive", "Open world file", "Ctrl+KeyO"],
    ["open-folder", "Open world folder", "Ctrl+Shift+KeyO"],
    ["open-project", "Open studio project", "Ctrl+Alt+KeyO"],
    ["save-project", "Save project", "Ctrl+KeyS"],
    ["save-project-as", "Save project as", "Ctrl+Shift+KeyS"],
    ["reload", "Reload render data", "Ctrl+KeyR"],
    ["export-png", "Export image", "Ctrl+Shift+KeyE"],
    ["export-glb", "Export 3D model", "Ctrl+Alt+KeyE"],
    ["fit", "Fit world to view", "KeyF"],
    ["isometric", "Isometric view", "Numpad0"],
    ["top", "Top view", "Numpad7"],
    ["inspect-view", "Inspect view", "KeyV"],
    ["reset-look", "Reset scene styling", ""],
  ].map(([id, label, binding]) =>
    action(id, label, "editor", binding ? [binding] : [], {
      inputs: id.startsWith("save-"),
    }),
  ),
  ...[
    ["undo", "Undo", "Ctrl+KeyZ"],
    ["redo", "Redo", "Ctrl+Shift+KeyZ"],
    ["play", "Play / pause", "Space"],
    ["add-key", "Add keyframe", "KeyK"],
    ["add-marker", "Add sequence marker", "KeyM"],
    ["toggle-path", "Toggle camera path", "KeyP"],
    ["image-panel", "Toggle scene styling", "KeyI"],
    ["previous-frame", "Previous frame", "ArrowLeft"],
    ["next-frame", "Next frame", "ArrowRight"],
    ["sequence-start", "Go to sequence start", "Home"],
    ["sequence-end", "Go to sequence end", "End"],
    ["export-mp4", "Export MP4 video", ""],
    ["export-webm", "Export WebM video", ""],
    ["export-gif", "Export GIF animation", ""],
  ].map(([id, label, binding]) =>
    action(id, label, "cinematic", binding ? [binding] : [], {
      repeat: id.endsWith("-frame"),
    }),
  ),
  ...(
    [
      ["forward", "Move forward", ["KeyW"]],
      ["back", "Move backward", ["KeyS"]],
      ["left", "Move left", ["KeyA"]],
      ["right", "Move right", ["KeyD"]],
      ["up", "Move up", ["KeyE", "Space"]],
      ["down", "Move down", ["KeyQ", "KeyC"]],
      ["boost", "Move faster", ["Shift"]],
      ["slow", "Move slower", ["Alt"]],
      ["lock", "Toggle pointer lock", ["Tab"]],
      ["exit", "Exit free camera", ["Escape"]],
    ] as [string, string, string[]][]
  ).map(([id, label, bindings]) =>
    action(`camera.${id}`, label, "camera", bindings),
  ),
  ...[
    ["copy", "Copy selection", "Ctrl+KeyC"],
    ["cut", "Cut selection", "Ctrl+KeyX"],
    ["paste", "Paste selection", "Ctrl+KeyV"],
    ["undo", "Undo structure edit", "Ctrl+KeyZ"],
    ["redo", "Redo structure edit", "Ctrl+Shift+KeyZ"],
    ["erase", "Erase selection", "Delete"],
  ].map(([id, label, binding]) =>
    action(
      `structure.${id}`,
      label,
      "structure",
      id === "redo" ? [binding, "Ctrl+KeyY"] : [binding],
    ),
  ),
  ...[
    ["undo", "Undo pose", "Ctrl+KeyZ"],
    ["redo", "Redo pose", "Ctrl+Shift+KeyZ"],
    ["translate", "Translate rig", "KeyW"],
    ["rotate", "Rotate rig", "KeyE"],
    ["focus", "Focus selected rig part", "KeyF"],
  ].map(([id, label, binding]) => action(`rig.${id}`, label, "rig", [binding])),
  action("packets.stop", "Stop playback & following", "packets", ["Escape"]),
  action("packets.find", "Find packets", "packets", ["Ctrl+KeyF"], {
    inputs: true,
  }),
  action(
    "documents.close",
    "Close active document",
    "documents",
    ["Ctrl+KeyW"],
    { inputs: true },
  ),
  action("documents.next", "Next document", "documents", ["Ctrl+Tab"], {
    inputs: true,
  }),
  action(
    "documents.previous",
    "Previous document",
    "documents",
    ["Ctrl+Shift+Tab"],
    { inputs: true },
  ),
  action(
    "documents.reopen",
    "Reopen closed document",
    "documents",
    ["Ctrl+Shift+KeyT"],
    { inputs: true },
  ),
  action("audio.play", "Play / pause audio", "audio", ["Space"]),
  action("welcome.find", "Find recent projects", "welcome", ["Ctrl+KeyF"], {
    inputs: true,
  }),
  action("console.close", "Hide console", "console", ["Shift+Escape"], {
    inputs: true,
  }),
  action(
    "console.maximize",
    "Maximize / restore console",
    "console",
    ["Ctrl+Shift+F12"],
    { inputs: true },
  ),
  action("console.find", "Find / filter logs", "console", ["Ctrl+KeyF"], {
    inputs: true,
  }),
  action("console.next", "Next log search result", "console", ["F3"], {
    inputs: true,
  }),
  action(
    "console.previous",
    "Previous log search result",
    "console",
    ["Shift+F3"],
    { inputs: true },
  ),
  action("console.clear", "Clear console", "console", ["Ctrl+KeyL"], {
    inputs: true,
  }),
  action("texture.zoom-in", "Zoom texture in", "texture", [
    "Equal",
    "Shift+Equal",
    "NumpadAdd",
  ]),
  action("texture.zoom-out", "Zoom texture out", "texture", [
    "Minus",
    "NumpadSubtract",
  ]),
  action("texture.reset", "Reset texture zoom", "texture", [
    "Digit0",
    "Numpad0",
  ]),
];
export type ShortcutProfile = {
  version: 1;
  timeout: number;
  bindings: Record<string, string[]>;
};
const storageKey = "craftbox.shortcuts.v1";
const changed = "craftbox:shortcuts-changed";
export function defaultShortcutProfile(): ShortcutProfile {
  return {
    version: 1,
    timeout: 1000,
    bindings: Object.fromEntries(
      shortcutActions.map((a) => [a.id, [...a.defaults]]),
    ),
  };
}
const codePattern =
  /^(Key[A-Z]|Digit[0-9]|Numpad[0-9]|F(?:[1-9]|1[0-9]|2[0-4])|Arrow(?:Left|Right|Up|Down)|Space|Enter|Tab|Escape|Backspace|Delete|Insert|Home|End|PageUp|PageDown|Comma|Period|Slash|Backslash|Semicolon|Quote|BracketLeft|BracketRight|Minus|Equal|Backquote|NumpadAdd|NumpadSubtract|NumpadMultiply|NumpadDivide|NumpadDecimal|NumpadEnter|Shift|Alt)$/;
export function validStroke(stroke: string) {
  const parts = stroke.split("+");
  const code = parts.pop() ?? "";
  const order = ["Ctrl", "Alt", "Shift"];
  return (
    codePattern.test(code) &&
    (!parts.length || !["Shift", "Alt"].includes(code)) &&
    parts.every((p) => order.includes(p)) &&
    parts.every(
      (p, i) => i === 0 || order.indexOf(parts[i - 1]) < order.indexOf(p),
    ) &&
    !parts.includes(code)
  );
}
function overlapping(a: ShortcutScope, b: ShortcutScope) {
  if (a === b || a === "app" || b === "app") return true;
  if (a === "documents" || b === "documents") return true;
  return (
    (a === "editor" && b === "cinematic") ||
    (b === "editor" && a === "cinematic")
  );
}
export function validateShortcutProfile(profile: ShortcutProfile): string[] {
  const errors: string[] = [];
  if (
    profile.version !== 1 ||
    !Number.isFinite(profile.timeout) ||
    profile.timeout < 250 ||
    profile.timeout > 5000
  )
    errors.push("Sequence timeout must be between 250 and 5000 ms.");
  const used: { action: ShortcutAction; binding: string }[] = [];
  for (const id of Object.keys(profile.bindings))
    if (!shortcutActions.some((a) => a.id === id))
      errors.push(`Unknown action: ${id}`);
  for (const a of shortcutActions) {
    const bindings = profile.bindings[a.id];
    if (!Array.isArray(bindings) || bindings.length > 8) {
      errors.push(`${a.label}: use up to 8 bindings.`);
      continue;
    }
    for (const binding of bindings) {
      if (
        typeof binding !== "string" ||
        binding.length > 160 ||
        !binding.split(" > ").every(validStroke) ||
        binding.split(" > ").length > 4
      ) {
        errors.push(`${a.label}: invalid binding ${String(binding)}.`);
        continue;
      }
      if (a.scope === "camera" && binding.includes(" > "))
        errors.push(
          `${a.label}: camera controls require a single key combination.`,
        );
      if (
        !a.id.startsWith("camera.") &&
        binding.split(" > ").some((stroke) => ["Shift", "Alt"].includes(stroke))
      )
        errors.push(`${a.label}: modifiers need a main key.`);
      for (const old of used)
        if (
          overlapping(a.scope, old.action.scope) &&
          (binding === old.binding ||
            binding.startsWith(old.binding + " > ") ||
            old.binding.startsWith(binding + " > "))
        )
          errors.push(
            `${a.label} conflicts with ${old.action.label}: ${displayBinding(binding)}`,
          );
      used.push({ action: a, binding });
    }
  }
  return errors;
}
export function parseShortcutProfile(json: string): ShortcutProfile {
  const value = JSON.parse(json);
  if (
    !value ||
    value.version !== 1 ||
    !value.bindings ||
    typeof value.bindings !== "object" ||
    Array.isArray(value.bindings)
  )
    throw new Error("Invalid CraftBox shortcut profile.");
  const profile = {
    ...defaultShortcutProfile(),
    ...value,
    bindings: { ...defaultShortcutProfile().bindings, ...value.bindings },
  } as ShortcutProfile;
  const errors = validateShortcutProfile(profile);
  if (errors.length) throw new Error(errors.join("\n"));
  return profile;
}
export function loadShortcutProfile(): ShortcutProfile {
  try {
    const raw = localStorage.getItem(storageKey);
    return raw ? parseShortcutProfile(raw) : defaultShortcutProfile();
  } catch {
    return defaultShortcutProfile();
  }
}
export function saveShortcutProfile(profile: ShortcutProfile) {
  const errors = validateShortcutProfile(profile);
  if (errors.length) throw new Error(errors.join("\n"));
  localStorage.setItem(storageKey, JSON.stringify(profile));
  window.dispatchEvent(new Event(changed));
}
export function useShortcutProfile() {
  const [profile, setProfile] = useState(loadShortcutProfile);
  useEffect(() => {
    const update = () => setProfile(loadShortcutProfile());
    window.addEventListener(changed, update);
    window.addEventListener("storage", update);
    return () => {
      window.removeEventListener(changed, update);
      window.removeEventListener("storage", update);
    };
  }, []);
  return profile;
}
export function eventStroke(
  e: Pick<
    KeyboardEvent,
    "code" | "key" | "ctrlKey" | "metaKey" | "altKey" | "shiftKey"
  >,
) {
  let code = e.code;
  if (/^(Shift|Alt|Control|Meta)(Left|Right)$/.test(code))
    return code.startsWith("Shift")
      ? "Shift"
      : code.startsWith("Alt")
        ? "Alt"
        : "";
  if (!code)
    code = /^[a-z]$/i.test(e.key)
      ? `Key${e.key.toUpperCase()}`
      : /^[0-9]$/.test(e.key)
        ? `Digit${e.key}`
        : ((
            { ",": "Comma", " ": "Space", "`": "Backquote" } as Record<
              string,
              string
            >
          )[e.key] ?? e.key);
  return [
    e.ctrlKey || e.metaKey ? "Ctrl" : "",
    e.altKey ? "Alt" : "",
    e.shiftKey ? "Shift" : "",
    code,
  ]
    .filter(Boolean)
    .join("+");
}
export function displayBinding(binding: string) {
  return binding
    .replace(/Key([A-Z])/g, "$1")
    .replace(/Digit([0-9])/g, "$1")
    .replace(/Comma/g, ",")
    .replace(/Backquote/g, "`")
    .replace(/Arrow/g, "");
}
export function shortcutLabel(profile: ShortcutProfile, id: string) {
  return (profile.bindings[id] ?? []).map(displayBinding).join(" / ");
}
export function shortcutBlocked(e: KeyboardEvent) {
  return (
    e.defaultPrevented ||
    e.isComposing ||
    !!(e.target as HTMLElement)?.closest?.(".shell-terminal") ||
    !!document.querySelector(
      '[role="dialog"][data-open], [role="alertdialog"][data-open], [data-slot="dropdown-menu-content"][data-open]',
    )
  );
}
export function useScopedShortcuts(
  scope: ShortcutScope,
  enabled: boolean,
  run: (id: string) => void,
) {
  const profile = useShortcutProfile();
  const current = useRef(run);
  current.current = run;
  useEffect(() => {
    if (!enabled) return;
    const matcher = new ShortcutMatcher();
    const key = (e: KeyboardEvent) => {
      if (shortcutBlocked(e)) {
        matcher.reset();
        return;
      }
      const id = matcher.match(e, profile, [scope]);
      if (id !== undefined) {
        e.preventDefault();
        if (id) current.current(id);
      }
    };
    const reset = () => matcher.reset();
    window.addEventListener("keydown", key);
    window.addEventListener("blur", reset);
    return () => {
      window.removeEventListener("keydown", key);
      window.removeEventListener("blur", reset);
    };
  }, [scope, enabled, profile]);
}
export function cameraHeld(
  profile: ShortcutProfile,
  keys: Set<string>,
  id: string,
  vertical = true,
  eventCode?: string,
) {
  const has = (part: string) =>
    part === "Ctrl"
      ? keys.has("ControlLeft") ||
        keys.has("ControlRight") ||
        keys.has("MetaLeft") ||
        keys.has("MetaRight")
      : part === "Shift"
        ? keys.has("ShiftLeft") || keys.has("ShiftRight")
        : part === "Alt"
          ? keys.has("AltLeft") || keys.has("AltRight")
          : keys.has(part);
  return (
    profile.bindings[`camera.${id}`]?.some((binding) => {
      const parts = binding.split("+");
      if (eventCode) {
        const part = eventCode.startsWith("Shift")
          ? "Shift"
          : eventCode.startsWith("Alt")
            ? "Alt"
            : /^(Control|Meta)/.test(eventCode)
              ? "Ctrl"
              : eventCode;
        if (!parts.includes(part)) return false;
      }
      if (!vertical && ["Space", "KeyC"].includes(parts[parts.length - 1]))
        return false;
      return (!has("Ctrl") || parts.includes("Ctrl")) && parts.every(has);
    }) ?? false
  );
}
export function cameraConsumesKey(
  profile: ShortcutProfile,
  keys: Set<string>,
  code: string,
  vertical: boolean,
) {
  return [
    "forward",
    "back",
    "left",
    "right",
    "up",
    "down",
    "boost",
    "slow",
  ].some((id) => cameraHeld(profile, keys, id, vertical, code));
}
export class ShortcutMatcher {
  private pending = "";
  private at = 0;
  private context = "";
  reset() {
    this.pending = "";
    this.context = "";
  }
  match(
    e: KeyboardEvent,
    profile: ShortcutProfile,
    scopes: ShortcutScope[],
    now = Date.now(),
    enabled: (id: string) => boolean = () => true,
  ): string | null | undefined {
    if (e.isComposing || e.getModifierState?.("AltGraph")) {
      this.reset();
      return;
    }
    const stroke = eventStroke(e);
    if (
      !stroke ||
      (["Shift", "Alt"].includes(stroke) && !scopes.includes("camera"))
    )
      return;
    const typing = !!(e.target as HTMLElement)?.closest?.(
      "input,textarea,select,[contenteditable]:not([contenteditable=false]),[role=textbox]",
    );
    const context = scopes.join(",") + JSON.stringify(profile) + String(typing);
    if (now - this.at > profile.timeout || this.context !== context)
      this.reset();
    const actions = shortcutActions.filter(
      (a) =>
        scopes.includes(a.scope) &&
        enabled(a.id) &&
        (!e.repeat || a.repeat) &&
        (!typing ||
          (a.inputs &&
            (e.ctrlKey ||
              e.metaKey ||
              /^F\d+$/.test(e.code) ||
              e.code === "Escape"))),
    );
    const find = (candidate: string) => {
      const exact = actions.find((a) =>
        profile.bindings[a.id]?.includes(candidate),
      );
      if (exact) {
        this.reset();
        return exact.id;
      }
      if (
        actions.some((a) =>
          profile.bindings[a.id]?.some((b) => b.startsWith(candidate + " > ")),
        )
      ) {
        this.pending = candidate;
        this.at = now;
        this.context = context;
        return null;
      }
      return undefined;
    };
    const candidate = this.pending ? `${this.pending} > ${stroke}` : stroke;
    const result = find(candidate);
    if (result !== undefined) return result;
    this.reset();
    return candidate !== stroke ? find(stroke) : undefined;
  }
}

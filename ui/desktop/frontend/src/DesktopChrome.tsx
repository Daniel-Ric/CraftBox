import type { WorkspaceAction } from "./workspace-actions";
import StudioInput from "@/StudioInput";
import { useEffect, useState } from "react";
import { Menubar } from "@base-ui/react/menubar";
import {
  Settings2,
  Check,
  Terminal,
  House,
  Search,
  Activity,
  Store,
  Blocks,
  Archive,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
} from "@/components/ui/dropdown-menu";
import type { EditorCommand, EditorStatus } from "./editor-commands";
import {
  WorldIcon,
  PackIcon,
  SkinIcon,
  RenderIcon,
  CinematicIcon,
  AccountIcon,
} from "./studio-icons";
const workspaces = [
  { id: "worlds", label: "Worlds", icon: WorldIcon },
  { id: "packs", label: "Resource packs", icon: PackIcon },
  { id: "skins", label: "Player skins", icon: SkinIcon },
  { id: "packets", label: "Packet Inspector", icon: Activity },
  { id: "structures", label: "Structure Studio", icon: Blocks },
  { id: "bedrock", label: "Archive", icon: Archive },
  { id: "marketplace", label: "Marketplace", icon: Store },
  { id: "render", label: "Render editor", icon: RenderIcon },
  { id: "cinematic", label: "Cinematic editor", icon: CinematicIcon },
];
type Props = {
  onNew?: () => void;
  onWorkspaceAction?: (page: string, action: WorkspaceAction) => void;
  page: string;
  onPage: (page: string) => void;
  onCommand: (command: EditorCommand) => void;
  onAccount: () => void;
  onSettings: () => void;
  onLogs: () => void;
  onCommandExplorer: () => void;
  onHelp: () => void;
  onAbout?: () => void;
  onUpdates?: () => void;
  onTheme: (theme: string) => void;
  theme: string;
  account: string;
  running: boolean;
  status: string;
  logsOpen: boolean;
  editor: EditorStatus;
};
export default function DesktopChrome(p: Props) {
  const [searchOpen, setSearchOpen] = useState(false),
    [query, setQuery] = useState("");
  const editorPage = p.page === "render" || p.page === "cinematic",
    cinematic = p.page === "cinematic",
    editable = editorPage && !p.editor.busy,
    ready = editable && p.editor.ready;
  const documentReady =
    ["render", "cinematic", "packs", "skins", "packets", "structures"].includes(
      p.page,
    ) &&
    p.editor.ready &&
    !p.editor.busy;
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if ((e.target as HTMLElement)?.closest?.(".shell-terminal")) return;
      if (e.defaultPrevented) return;
      const mod = e.ctrlKey || e.metaKey;
      const dialog = document.querySelector('[role="dialog"][data-open]');
      if (dialog) {
        if (
          mod &&
          e.key.toLowerCase() === "n" &&
          dialog.classList.contains("welcome-dialog")
        ) {
          e.preventDefault();
          p.onNew?.();
        }
        return;
      }
      if (mod && e.shiftKey && e.key.toLowerCase() === "j") {
        e.preventDefault();
        p.onCommandExplorer();
        return;
      }
      if (mod && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setQuery("");
        setSearchOpen(true);
        return;
      }
      if (e.key === "F1") {
        e.preventDefault();
        p.onHelp();
        return;
      }
      if (mod && e.shiftKey && /^Digit[0-9]$/.test(e.code)) {
        e.preventDefault();
        const target = workspaces[(+e.code.slice(-1) + 9) % 10];
        if (target) p.onPage(target.id);
        return;
      }
      if (!mod) return;
      if (e.key.toLowerCase() === "n") {
        e.preventDefault();
        p.onNew?.();
        return;
      }
      if (e.key === ",") {
        e.preventDefault();
        p.onSettings();
        return;
      }
      const typing = (e.target as HTMLElement)?.matches?.(
        "input,textarea,[contenteditable=true]",
      );
      if (editable && e.key.toLowerCase() === "o") {
        e.preventDefault();
        p.onCommand(e.shiftKey ? "open-folder" : "open-archive");
      } else if (ready && e.shiftKey && e.key.toLowerCase() === "e") {
        e.preventDefault();
        p.onCommand("export-png");
      } else if (documentReady && e.key.toLowerCase() === "s") {
        e.preventDefault();
        p.onCommand(e.shiftKey ? "save-project-as" : "save-project");
      } else if (
        cinematic &&
        editable &&
        !typing &&
        e.key.toLowerCase() === "z"
      ) {
        e.preventDefault();
        p.onCommand(e.shiftKey ? "redo" : "undo");
      }
    };
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, [p, editable, ready, cinematic]);
  const item = (
    label: string,
    command: EditorCommand,
    enabled: boolean,
    shortcut?: string,
  ) => (
    <DropdownMenuItem disabled={!enabled} onClick={() => p.onCommand(command)}>
      {label}
      {shortcut && <DropdownMenuShortcut>{shortcut}</DropdownMenuShortcut>}
    </DropdownMenuItem>
  );
  const commands: {
    label: string;
    group: string;
    enabled: boolean;
    run: () => void;
    shortcut?: string;
    keywords?: string;
  }[] = [
    {
      label: "Command Explorer",
      group: "View",
      enabled: true,
      run: p.onCommandExplorer,
      shortcut: "Ctrl+Shift+J",
    },
    {
      label: "New project",
      group: "File",
      enabled: true,
      run: () => p.onNew?.(),
      shortcut: "Ctrl+N",
    },
    {
      label: "Open world file",
      group: "File",
      enabled: editable,
      run: () => p.onCommand("open-archive"),
      shortcut: "Ctrl+O",
    },
    {
      label: "Open world folder",
      group: "File",
      enabled: editable,
      run: () => p.onCommand("open-folder"),
    },
    {
      label: "Export image",
      keywords: "PNG JPEG JPG WebP TIFF BMP",
      group: "File",
      enabled: ready,
      run: () => p.onCommand("export-png"),
      shortcut: "Ctrl+Shift+E",
    },
    {
      label: "Export 3D model",
      keywords: "GLB glTF OBJ FBX STL Blender",
      group: "File",
      enabled: ready,
      run: () => p.onCommand("export-glb"),
    },
    ...(["mp4", "webm", "gif"] as const).map((format) => ({
      label: `${format.toUpperCase()} export`,
      group: "File",
      enabled: cinematic && ready,
      run: () => p.onCommand(`export-${format}`),
    })),
    {
      label: "Save current project",
      group: "File",
      enabled: documentReady,
      run: () => p.onCommand("save-project"),
      shortcut: "Ctrl+S",
    },
    {
      label: "Fit world to view",
      group: "Scene",
      enabled: ready,
      run: () => p.onCommand("fit"),
    },
    {
      label: "Isometric view",
      group: "Scene",
      enabled: ready,
      run: () => p.onCommand("isometric"),
    },
    {
      label: "Top view",
      group: "Scene",
      enabled: ready,
      run: () => p.onCommand("top"),
    },
    {
      label: "Play / pause",
      group: "Sequence",
      enabled: cinematic && ready,
      run: () => p.onCommand("play"),
    },
    {
      label: "Preferences",
      group: "Edit",
      enabled: true,
      run: p.onSettings,
      shortcut: "Ctrl+,",
    },
    {
      label: "Toggle console",
      group: "Window",
      enabled: true,
      run: p.onLogs,
    },
    {
      label: "Controls & shortcuts",
      group: "Help",
      enabled: true,
      run: p.onHelp,
      shortcut: "F1",
    },
    {
      label: "Import resource pack files",
      group: "Import & libraries",
      enabled: !!p.onWorkspaceAction,
      run: () => p.onWorkspaceAction?.("packs", "import-files"),
    },
    {
      label: "Import resource pack folder",
      group: "Import & libraries",
      enabled: !!p.onWorkspaceAction,
      run: () => p.onWorkspaceAction?.("packs", "import-folder"),
    },
    {
      label: "Import player skin files",
      group: "Import & libraries",
      enabled: !!p.onWorkspaceAction,
      run: () => p.onWorkspaceAction?.("skins", "import-files"),
    },
    {
      label: "Browse installed Minecraft packs",
      group: "Import & libraries",
      enabled: !!p.onWorkspaceAction,
      run: () => p.onWorkspaceAction?.("packs", "browse-minecraft"),
    },
    {
      label: "Browse installed Minecraft skins",
      group: "Import & libraries",
      enabled: !!p.onWorkspaceAction,
      run: () => p.onWorkspaceAction?.("skins", "browse-minecraft"),
    },
    {
      label: "Import packet recording",
      group: "Import & libraries",
      enabled: !!p.onWorkspaceAction,
      run: () => p.onWorkspaceAction?.("packets", "import-files"),
    },
    ...workspaces.map((w) => ({
      label: w.label,
      group: "Workspaces",
      enabled: true,
      run: () => p.onPage(w.id),
    })),
  ];
  return (
    <>
      <header className="desktop-chrome">
        <div className="application-menu-row">
          <Menubar
            className="application-menubar"
            aria-label="Application menu"
          >
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                File
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                <DropdownMenuItem onClick={p.onNew}>
                  Create new …
                  <DropdownMenuShortcut>Ctrl+N</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => p.onPage("home")}>
                  Go to start screen
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => p.onPage("render")}>
                  Render workspace
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => p.onPage("cinematic")}>
                  Cinematic workspace
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {item("Open world file …", "open-archive", editable, "Ctrl+O")}
                {item(
                  "Open world folder …",
                  "open-folder",
                  editable,
                  "Ctrl+Shift+O",
                )}
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    Import & browse
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent className="application-menu-popup">
                    <DropdownMenuItem
                      disabled={!p.onWorkspaceAction}
                      onClick={() =>
                        p.onWorkspaceAction?.("packs", "import-files")
                      }
                    >
                      Import resource pack files
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={!p.onWorkspaceAction}
                      onClick={() =>
                        p.onWorkspaceAction?.("packs", "import-folder")
                      }
                    >
                      Import resource pack folder
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={!p.onWorkspaceAction}
                      onClick={() =>
                        p.onWorkspaceAction?.("skins", "import-files")
                      }
                    >
                      Import player skin files
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={!p.onWorkspaceAction}
                      onClick={() =>
                        p.onWorkspaceAction?.("packs", "browse-minecraft")
                      }
                    >
                      Browse installed Minecraft packs
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={!p.onWorkspaceAction}
                      onClick={() =>
                        p.onWorkspaceAction?.("skins", "browse-minecraft")
                      }
                    >
                      Browse installed Minecraft skins
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={!p.onWorkspaceAction}
                      onClick={() =>
                        p.onWorkspaceAction?.("packets", "import-files")
                      }
                    >
                      Import packet recording
                    </DropdownMenuItem>
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
                <DropdownMenuSeparator />
                {item("Reload render data", "reload", editable)}
                <DropdownMenuSeparator />
                {item("Open studio project …", "open-project", !p.editor.busy)}
                {item(
                  "Save project …",
                  "save-project",
                  documentReady,
                  "Ctrl+S",
                )}
                {item(
                  "Save project as …",
                  "save-project-as",
                  documentReady,
                  "Ctrl+Shift+S",
                )}
                <DropdownMenuSeparator />
                {[
                  "packs",
                  "skins",
                  "packets",
                  "structures",
                  "bedrock",
                ].includes(p.page) && (
                  <DropdownMenuItem
                    onClick={() => p.onWorkspaceAction?.(p.page, "export")}
                  >
                    {p.page === "bedrock"
                      ? "Export Bedrock snapshot…"
                      : p.page === "structures"
                        ? "Export blueprint…"
                        : p.page === "packets"
                          ? "Export packet recording"
                          : "Export library selection"}
                  </DropdownMenuItem>
                )}
                {item("Export image …", "export-png", ready, "Ctrl+Shift+E")}
                {item("Export 3D model …", "export-glb", ready)}
                {item(
                  "Export video as MP4 …",
                  "export-mp4",
                  cinematic && ready,
                )}
                {item(
                  "Export video as WebM …",
                  "export-webm",
                  cinematic && ready,
                )}
                {item(
                  "Export animation as GIF …",
                  "export-gif",
                  cinematic && ready,
                )}
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Edit
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                {item(
                  "Undo",
                  "undo",
                  cinematic && editable && p.editor.undo,
                  "Ctrl+Z",
                )}
                {item(
                  "Redo",
                  "redo",
                  cinematic && editable && p.editor.redo,
                  "Ctrl+Shift+Z",
                )}
                <DropdownMenuSeparator />
                {item("Reset scene styling", "reset-look", editable)}
                <DropdownMenuItem onClick={p.onSettings}>
                  Settings …<DropdownMenuShortcut>Ctrl+,</DropdownMenuShortcut>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Library
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() => p.onWorkspaceAction?.("packs", "files")}
                >
                  Resource packs
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() => p.onWorkspaceAction?.("skins", "files")}
                >
                  Player skins
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() => p.onWorkspaceAction?.("packets", "settings")}
                >
                  Packet recording settings
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() => p.onWorkspaceAction?.("packets", "search")}
                >
                  Packet search
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() => p.onWorkspaceAction?.("packets", "rules")}
                >
                  Packet rules & watchpoints
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() =>
                    p.onWorkspaceAction?.("marketplace", "settings")
                  }
                >
                  Marketplace filters
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() =>
                    p.onWorkspaceAction?.("marketplace", "collections")
                  }
                >
                  Marketplace collections
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={!p.onWorkspaceAction}
                  onClick={() =>
                    p.onWorkspaceAction?.("marketplace", "favorites")
                  }
                >
                  Marketplace favorites
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={p.onAccount}>
                  Account & linked services
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Scene
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                {item("Fit world to view", "fit", ready)}
                {item("Isometric view", "isometric", ready)}
                {item("Top view", "top", ready)}
                <DropdownMenuSeparator />
                {item("Reload render data", "reload", editable)}
                {item("Reset scene styling", "reset-look", editable)}
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Sequence
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                {item("Play / pause", "play", cinematic && ready, "Space")}
                {item("Add keyframe", "add-key", cinematic && ready)}
                {item("Toggle camera path", "toggle-path", cinematic && ready)}
                <DropdownMenuSeparator />
                {item("Open studio project …", "open-project", editable)}
                {item(
                  "Save studio project …",
                  "save-project",
                  editable && ready,
                  "Ctrl+S",
                )}
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                View
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                {["packs", "skins", "marketplace"].includes(p.page) && (
                  <>
                    <DropdownMenuItem
                      onClick={() => p.onWorkspaceAction?.(p.page, "list")}
                    >
                      List view
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={() => p.onWorkspaceAction?.(p.page, "grid")}
                    >
                      Tile view
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                  </>
                )}{" "}
                {item("Fit world to view", "fit", ready)}
                {item("Isometric view", "isometric", ready)}
                {item("Top view", "top", ready)}
                {item(
                  "Toggle scene styling",
                  "image-panel",
                  cinematic && editable,
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={p.onCommandExplorer}>
                  <Terminal size={13} />
                  Command Explorer
                  <DropdownMenuShortcut>Ctrl+Shift+J</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={p.onLogs}>
                  {p.logsOpen ? (
                    <Check size={13} />
                  ) : (
                    <span className="menu-check-space" />
                  )}
                  Logs
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                {[
                  ["dark", "Dark interface"],
                  ["graphite", "Charcoal"],
                  ["gray", "Mid-gray"],
                  ["light", "Light interface"],
                  ["system", "System theme"],
                ].map(([id, label]) => (
                  <DropdownMenuItem key={id} onClick={() => p.onTheme(id)}>
                    {p.theme === id ? (
                      <Check size={13} />
                    ) : (
                      <span className="menu-check-space" />
                    )}
                    {label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Window
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                {workspaces.map((w, i) => (
                  <DropdownMenuItem key={w.id} onClick={() => p.onPage(w.id)}>
                    {p.page === w.id ? (
                      <Check size={13} />
                    ) : (
                      <span className="menu-check-space" />
                    )}
                    {w.label}
                    <DropdownMenuShortcut>
                      Ctrl+Shift+{i + 1}
                    </DropdownMenuShortcut>
                  </DropdownMenuItem>
                ))}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={p.onAccount}>
                  Microsoft account …
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Help
              </DropdownMenuTrigger>
              <DropdownMenuContent className="application-menu-popup">
                <DropdownMenuItem onClick={p.onHelp}>
                  Controls &amp; shortcuts
                  <DropdownMenuShortcut>F1</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={p.onUpdates || p.onSettings}>
                  Version &amp; updates
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={p.onAbout || p.onHelp}>
                  About CraftBox
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </Menubar>
          <button
            className="chrome-command-search"
            onClick={() => {
              setQuery("");
              setSearchOpen(true);
            }}
          >
            <Search size={13} />
            <span>Find command</span>
            <kbd>Ctrl+K</kbd>
          </button>
          <span className="application-document-name">
            {workspaces.find((w) => w.id === p.page)?.label ??
              (p.page === "home" ? "Start" : "Settings")}
          </span>
          <button
            className="chrome-account"
            onClick={p.onAccount}
            aria-label={
              p.account
                ? `Manage account: ${p.account}`
                : "Not signed in – Sign in with Microsoft"
            }
          >
            <AccountIcon size={14} />
            <span>{p.account || "Sign in"}</span>
          </button>
        </div>
        <div className="workspace-bar">
          <nav className="workspace-tabs" aria-label="Workspaces">
            <button
              aria-label="Start"
              data-tooltip="Start"
              className={`workspace-home${p.page === "home" ? " active" : ""}`}
              aria-current={p.page === "home" ? "page" : undefined}
              onClick={() => p.onPage("home")}
            >
              <House size={14} />
            </button>
            {workspaces.map((w) => (
              <button
                key={w.id}
                className={`${p.page === w.id ? "active" : ""}${w.id === "render" ? " workspace-editor-start" : ""}`}
                data-tooltip={`${w.label} · Ctrl+${workspaces.indexOf(w) + 1}`}
                aria-current={p.page === w.id ? "page" : undefined}
                onClick={() => p.onPage(w.id)}
              >
                <w.icon size={16} />
                <span>{w.label}</span>
                {(w.id === "packs" ||
                  w.id === "skins" ||
                  w.id === "packets") && (
                  <span className="workspace-beta-badge">
                    {w.id === "packets" ? "BETA" : "3D BETA"}
                  </span>
                )}
              </button>
            ))}
          </nav>
          <div className="workspace-bar-tools">
            <span className={`chrome-session ${p.running ? "running" : ""}`}>
              <i />
              {p.status}
            </span>
            <button
              aria-label="Open logs"
              aria-pressed={p.logsOpen}
              onClick={p.onLogs}
            >
              <Terminal size={15} />
            </button>
            <button
              aria-label="Settings"
              aria-current={p.page === "settings" ? "page" : undefined}
              onClick={p.onSettings}
            >
              <Settings2 size={15} />
            </button>
          </div>
        </div>
      </header>
      <Dialog open={searchOpen} onOpenChange={setSearchOpen}>
        <DialogContent className="command-search-dialog">
          <DialogHeader>
            <DialogTitle>Find command</DialogTitle>
            <DialogDescription>
              Search menu commands and workspaces.
            </DialogDescription>
          </DialogHeader>
          <div className="command-search-field">
            <Search size={16} />
            <StudioInput
              autoFocus
              aria-label="Search commands"
              placeholder="Enter a command or workspace"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  document
                    .querySelector<HTMLButtonElement>(
                      ".command-results button:not(:disabled)",
                    )
                    ?.focus();
                }
                if (e.key === "Enter") {
                  e.preventDefault();
                  const c = commands.find(
                    (c) =>
                      c.enabled &&
                      `${c.label} ${c.group} ${c.keywords || ""}`
                        .toLocaleLowerCase()
                        .includes(query.toLocaleLowerCase()),
                  );
                  if (c) {
                    setSearchOpen(false);
                    c.run();
                  }
                }
              }}
            />
          </div>
          <div
            className="command-results"
            onKeyDown={(e) => {
              if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
              const buttons = Array.from(
                e.currentTarget.querySelectorAll<HTMLButtonElement>(
                  "button:not(:disabled)",
                ),
              );
              const at = buttons.indexOf(
                document.activeElement as HTMLButtonElement,
              );
              e.preventDefault();
              buttons[
                (at + (e.key === "ArrowDown" ? 1 : -1) + buttons.length) %
                  buttons.length
              ]?.focus();
            }}
          >
            {commands
              .filter((c) =>
                `${c.label} ${c.group} ${c.keywords || ""}`
                  .toLocaleLowerCase()
                  .includes(query.toLocaleLowerCase()),
              )
              .map((c) => (
                <button
                  key={c.label}
                  disabled={!c.enabled}
                  onClick={() => {
                    setSearchOpen(false);
                    c.run();
                  }}
                >
                  <span>
                    <small>{c.group}</small>
                    {c.label}
                  </span>
                  {c.shortcut && <kbd>{c.shortcut}</kbd>}
                </button>
              ))}
            {!commands.some((c) =>
              `${c.label} ${c.group} ${c.keywords || ""}`
                .toLocaleLowerCase()
                .includes(query.toLocaleLowerCase()),
            ) && <p>No matching commands.</p>}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

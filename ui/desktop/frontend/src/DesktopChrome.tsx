import {
  ShortcutMatcher,
  shortcutActions,
  shortcutBlocked,
  shortcutLabel,
  useShortcutProfile,
} from "./shortcuts";
import type { WorkspaceAction } from "./workspace-actions";
import CommandSearchDialog from "./CommandSearchDialog";
import FolderShortcuts, { craftBoxFolders } from "./FolderShortcuts";
import type { CraftBoxFolder } from "./bridge";
import type { CommandAction } from "./command-search";
import { useEffect, useRef, useState, type ReactNode } from "react";
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

function ApplicationMenuContent({ children }: { children: ReactNode }) {
  return (
    <DropdownMenuContent className="application-menu-popup" sideOffset={0}>
      {children}
    </DropdownMenuContent>
  );
}

type Props = {
  onNew?: () => void;
  onWorkspaceAction?: (page: string, action: WorkspaceAction) => void;
  page: string;
  onPage: (page: string) => void;
  onCommand: (command: EditorCommand) => void;
  onAccount: () => void;
  onSettings: () => void;
  onLogs: () => void;
  onOpenLogFolder: () => void;
  onOpenCraftBoxFolder: (folder: CraftBoxFolder) => void;
  onOpenResultFolder: (path: string) => void;
  onCommandExplorer: () => void;
  onHelp: () => void;
  onOpenGuide: () => void;
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
  const [searchOpen, setSearchOpen] = useState(false);
  const moduleNavigation = useRef<HTMLElement>(null);
  useEffect(() => {
    const navigation = moduleNavigation.current;
    if (!navigation) return;
    const revealActive = () => {
      navigation
        .querySelector<HTMLElement>('[aria-current="page"]')
        ?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
    };
    revealActive();
    const observer = new ResizeObserver(revealActive);
    observer.observe(navigation);
    return () => observer.disconnect();
  }, [p.page]);
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
  const profile = useShortcutProfile();
  const matcher = useRef(new ShortcutMatcher());
  const label = (id: string) => shortcutLabel(profile, id);
  const available = (id: string) => {
    if (id.startsWith("library.")) return !!p.onWorkspaceAction;
    if (id === "new") return !!p.onNew;
    if (["save-project", "save-project-as"].includes(id)) return documentReady;
    if (id === "open-project") return !p.editor.busy;
    if (["open-archive", "open-folder", "reload", "reset-look"].includes(id))
      return editable;
    if (id === "undo") return cinematic && editable && p.editor.undo;
    if (id === "redo") return cinematic && editable && p.editor.redo;
    const action = shortcutActions.find((a) => a.id === id);
    if (action?.scope === "editor") return ready;
    if (action?.scope === "cinematic") return cinematic && ready;
    return true;
  };
  const runShortcut = (id: string) => {
    if (id.startsWith("library.")) {
      const [, page, action] = id.split(".");
      p.onWorkspaceAction?.(page, action as WorkspaceAction);
      return;
    }
    if (id.startsWith("workspace.")) {
      p.onPage(id.slice(10));
      return;
    }
    const handlers: Record<string, () => void> = {
      search: () => setSearchOpen(true),
      new: () => p.onNew?.(),
      settings: p.onSettings,
      help: p.onHelp,
      logs: p.onLogs,
      explorer: p.onCommandExplorer,
      home: () => p.onPage("home"),
      account: p.onAccount,
      theme: () => p.onTheme(p.theme === "dark" ? "light" : "dark"),
      "log-folder": p.onOpenLogFolder,
      about: p.onAbout || p.onHelp,
      updates: p.onUpdates || p.onSettings,
    };
    if (handlers[id]) handlers[id]();
    else p.onCommand(id as EditorCommand);
  };
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (shortcutBlocked(e)) {
        const welcome = document.querySelector(
          '[role="dialog"][data-open].welcome-dialog',
        );
        if (!e.defaultPrevented && welcome && p.onNew) {
          const id = matcher.current.match(
            e,
            profile,
            ["app"],
            undefined,
            (id) => id === "new",
          );
          if (id !== undefined) {
            e.preventDefault();
            if (id) p.onNew();
          }
        } else matcher.current.reset();
        return;
      }
      const id = matcher.current.match(
        e,
        profile,
        editorPage
          ? cinematic
            ? ["app", "editor", "cinematic"]
            : ["app", "editor"]
          : ["app", "editor"],
        undefined,
        (id) => {
          if (
            !editorPage &&
            !["save-project", "save-project-as", "open-project"].includes(id) &&
            shortcutActions.find((a) => a.id === id)?.scope !== "app"
          )
            return false;
          return available(id);
        },
      );
      if (id !== undefined) {
        e.preventDefault();
        if (id) runShortcut(id);
      }
    };
    const reset = () => matcher.current.reset();
    window.addEventListener("keydown", key);
    window.addEventListener("blur", reset);
    return () => {
      window.removeEventListener("keydown", key);
      window.removeEventListener("blur", reset);
    };
  }, [p, profile, editable, ready, cinematic, documentReady]);
  const item = (
    text: string,
    command: EditorCommand,
    enabled: boolean,
    _shortcut?: string,
  ) => (
    <DropdownMenuItem disabled={!enabled} onClick={() => p.onCommand(command)}>
      {text}
      <DropdownMenuShortcut bindings={profile.bindings[command]} />
    </DropdownMenuItem>
  );
  const commands: CommandAction[] = shortcutActions
    .filter((a) => ["app", "editor", "cinematic"].includes(a.scope))
    .map((a) => ({
      label: a.label,
      group: a.id.startsWith("workspace.")
        ? "Workspaces"
        : a.scope === "app"
          ? "Application"
          : a.scope === "cinematic"
            ? "Sequence"
            : "Editor",
      enabled: available(a.id),
      run: () => runShortcut(a.id),
      shortcut: label(a.id),
      keywords:
        a.id +
        (a.id === "export-png"
          ? " PNG JPEG JPG WebP TIFF BMP"
          : a.id === "export-glb"
            ? " GLB glTF OBJ FBX STL Blender"
            : ""),
    }));
  commands.push(...craftBoxFolders.map(({ id, label }) => ({
    label,
    group: "Storage locations",
    enabled: true,
    run: () => p.onOpenCraftBoxFolder(id),
    keywords: "open folder " + id,
  })));
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
              <ApplicationMenuContent>
                <DropdownMenuItem onClick={p.onNew}>
                  Create new …
                  <DropdownMenuShortcut bindings={profile.bindings.new} />
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
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Edit
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
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
                  Settings …
                  <DropdownMenuShortcut bindings={profile.bindings.settings} />
                </DropdownMenuItem>
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Library
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
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
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Scene
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
                {item("Fit world to view", "fit", ready)}
                {item("Isometric view", "isometric", ready)}
                {item("Top view", "top", ready)}
                <DropdownMenuSeparator />
                {item("Reload render data", "reload", editable)}
                {item("Reset scene styling", "reset-look", editable)}
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Sequence
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
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
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                View
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
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
                  <DropdownMenuShortcut bindings={profile.bindings.explorer} />
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
                  ["light", "Light interface"],
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
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Window
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
                {workspaces.map((w) => (
                  <DropdownMenuItem key={w.id} onClick={() => p.onPage(w.id)}>
                    {p.page === w.id ? (
                      <Check size={13} />
                    ) : (
                      <span className="menu-check-space" />
                    )}
                    {w.label}
                    <DropdownMenuShortcut
                      bindings={profile.bindings[`workspace.${w.id}`]}
                    />
                  </DropdownMenuItem>
                ))}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={p.onAccount}>
                  Microsoft account …
                </DropdownMenuItem>
              </ApplicationMenuContent>
            </DropdownMenu>
            <DropdownMenu>
              <DropdownMenuTrigger className="application-menu-trigger">
                Help
              </DropdownMenuTrigger>
              <ApplicationMenuContent>
                <DropdownMenuItem onClick={p.onHelp}>
                  Controls &amp; shortcuts
                  <DropdownMenuShortcut bindings={profile.bindings.help} />
                </DropdownMenuItem>
                <DropdownMenuItem onClick={p.onOpenGuide}>
                  Online documentation
                </DropdownMenuItem>
                <DropdownMenuItem onClick={p.onUpdates || p.onSettings}>
                  Version &amp; updates
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={p.onAbout || p.onHelp}>
                  About CraftBox
                </DropdownMenuItem>
              </ApplicationMenuContent>
            </DropdownMenu>
          </Menubar>
          <button
            className="chrome-command-search"
            aria-label="Find command"
            onClick={() => {
              setSearchOpen(true);
            }}
          >
            <Search size={13} />
            <span>Find command</span>
            <kbd>{label("search")}</kbd>
          </button>
          <span className="application-document-name">
            {workspaces.find((w) => w.id === p.page)?.label ??
              (p.page === "home" ? "Start" : "Settings")}
          </span>
          <span
            className={`chrome-session ${p.running ? "running" : ""}`}
            role="status"
          >
            <i />
            {p.status}
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
            <span translate={p.account ? "no" : undefined}>{p.account || "Sign in"}</span>
          </button>
        </div>
      </header>
      <aside className="module-sidebar" aria-label="Workspace sidebar">
        <nav
          ref={moduleNavigation}
          className="module-navigation"
          aria-label="Workspaces"
        >
          <button
            className="module-start"
            aria-label="Start"
            data-tooltip="Start"
            aria-current={p.page === "home" ? "page" : undefined}
            onClick={() => p.onPage("home")}
          >
            <span className="module-icon">
              <House size={20} />
              <House className="module-icon-accent" size={20} />
            </span>
            <span>Start</span>
          </button>
          {workspaces.map((w) => (
            <button
              key={w.id}
              className={w.id === "render" ? "module-editor-start" : undefined}
              aria-label={w.label}
              data-tooltip={`${w.label}${["packs", "skins", "packets"].includes(w.id) ? " · Beta" : ""}`}
              data-tooltip-shortcut={label(`workspace.${w.id}`)}
              aria-current={p.page === w.id ? "page" : undefined}
              onClick={() => p.onPage(w.id)}
            >
              <span className="module-icon">
                <w.icon size={21} />
                <w.icon className="module-icon-accent" size={21} />
              </span>
              <span>
                {(
                  {
                    packs: "Packs",
                    skins: "Skins",
                    packets: "Packets",
                    structures: "Structures",
                    render: "Render",
                    cinematic: "Cinematic",
                    marketplace: "Market",
                  } as Record<string, string>
                )[w.id] ?? w.label}
              </span>
            </button>
          ))}
        </nav>
        <div className="module-sidebar-utilities">
          <FolderShortcuts
            key={p.account}
            signedIn={!!p.account}
            onOpenCraftBoxFolder={p.onOpenCraftBoxFolder}
            onOpenLogFolder={p.onOpenLogFolder}
            onOpenResultFolder={p.onOpenResultFolder}
          />
          <button
            aria-label="Settings"
            data-tooltip="Settings"
            data-tooltip-shortcut={label("settings")}
            onClick={p.onSettings}
          >
            <Settings2 size={19} />
            <span>Settings</span>
          </button>
        </div>
      </aside>
      <CommandSearchDialog
        open={searchOpen}
        onOpenChange={setSearchOpen}
        commands={commands}
      />
    </>
  );
}

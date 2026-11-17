import {
  ShortcutMatcher,
  shortcutBlocked,
  useShortcutProfile,
} from "./shortcuts";
import { useRef, useState } from "react";
import {
  ChevronDown,
  FolderOpen,
  MoreHorizontal,
  Pin,
  Plus,
  Search,
  SlidersHorizontal,
  Trash2,
  Info,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import RecentProjectDetails from "./RecentProjectDetails";
import { CaptureIcon, WorkspaceIcon } from "./studio-icons";
import {
  projectLabels,
  type RecentProjectAction,
  type StudioProject,
} from "./start-project";
import type { World } from "./bridge";
import "./welcome-workspace.css";

export default function WelcomeHome({
  recent,
  worlds,
  onCapture,
  onNew,
  onOpen,
  onOpenProject,
  onResume,
  onRecentAction,
}: {
  recent: StudioProject[];
  worlds: World[];
  onCapture: () => void;
  onNew: () => void;
  onOpen: (kind: "archive" | "folder") => void;
  onOpenProject: () => void;
  onResume: (project: StudioProject) => void;
  onRecentAction: (id: string, action: RecentProjectAction) => void;
}) {
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState("all");
  const [sort, setSort] = useState("recent");
  const [details, setDetails] = useState("");
  const search = useRef<HTMLInputElement>(null);
  const projects = recent
    .filter(
      (project) =>
        `${project.name} ${project.sourceName ?? ""} ${project.address}`
          .toLocaleLowerCase()
          .includes(query.trim().toLocaleLowerCase()) &&
        (filter === "all" ||
          (filter === "pinned"
            ? project.pinned
            : filter === "capture"
              ? !["render", "cinematic"].includes(project.page)
              : project.page === filter)),
    )
    .sort(
      (a, b) =>
        Number(!!b.pinned) - Number(!!a.pinned) ||
        (sort === "name"
          ? a.name.localeCompare(b.name)
          : b.updated - a.updated),
    );
  const shortcutProfile = useShortcutProfile();
  const shortcutMatcher = useRef(new ShortcutMatcher());
  return (
    <section
      className={`welcome-home${recent.length ? "" : " welcome-home-empty"}`}
      aria-labelledby="welcome-heading"
      onKeyDown={(event) => {
        if (shortcutBlocked(event.nativeEvent) || !recent.length) return;
        const id = shortcutMatcher.current.match(
          event.nativeEvent,
          shortcutProfile,
          ["welcome"],
        );
        if (id !== undefined) {
          event.preventDefault();
          if (id) search.current?.focus();
        }
      }}
    >
      <div className="welcome-hero" aria-hidden="true">
        <img
          src="/studio/welcome-pixel-landscape.png"
          alt=""
          decoding="async"
          fetchPriority="high"
        />
      </div>
      <div className="welcome-home-inner">
        <header className="welcome-brand">
          <img
            src="/craft-box-icon.png"
            alt="CraftBox logo"
            width={76}
            height={76}
          />
          <div>
            <div className="welcome-title">
              <h1 id="welcome-heading">CraftBox</h1>
              <span
                className="welcome-credit"
                title="With contributions from CraftBox and bedrocktool contributors"
              >
                by Daniel-Ric and contributors
              </span>
            </div>
            <p>Capture a world. Create something with it.</p>
          </div>
        </header>
        <div className="welcome-quick-actions" aria-label="Get started">
          <Button className="welcome-capture" onClick={onCapture}>
            <CaptureIcon size={19} />
            <span>Start capture</span>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="secondary" />}>
              <FolderOpen size={19} />
              <span>Open …</span>
              <ChevronDown size={14} />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem onClick={() => onOpen("archive")}>
                Open world file …
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onOpen("folder")}>
                Open world folder …
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onOpenProject}>
                Open CraftBox project …
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button variant="secondary" onClick={onNew}>
            <Plus size={19} />
            <span>New …</span>
          </Button>
        </div>
        {recent.length ? (
          <section
            className="welcome-recents"
            aria-labelledby="welcome-recent-heading"
          >
            <div className="welcome-recent-heading">
              <h2 id="welcome-recent-heading">
                Recently opened{" "}
                <span className="welcome-recent-count">{recent.length}</span>
              </h2>
              <div
                className="welcome-search"
                role="search"
                aria-label="Find recent projects"
              >
                <Search size={15} />
                <input
                  ref={search}
                  type="search"
                  autoComplete="off"
                  spellCheck={false}
                  aria-label="Search recent projects"
                  placeholder="Search projects"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Escape" && query) {
                      event.preventDefault();
                      event.stopPropagation();
                      setQuery("");
                    }
                  }}
                />
                {query && (
                  <button
                    type="button"
                    className="welcome-search-clear"
                    aria-label="Clear project search"
                    onClick={() => {
                      setQuery("");
                      search.current?.focus();
                    }}
                  >
                    <X size={13} />
                  </button>
                )}
              </div>
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label="Recent project options"
                    />
                  }
                >
                  <SlidersHorizontal size={16} />
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="end"
                  className="welcome-options-menu"
                >
                  <DropdownMenuRadioGroup
                    value={filter}
                    onValueChange={(value) => setFilter(String(value))}
                    aria-label="Show projects"
                  >
                    <DropdownMenuLabel>Show</DropdownMenuLabel>
                    {[
                      ["all", "All projects"],
                      ["pinned", "Pinned"],
                      ["render", "Still renders"],
                      ["cinematic", "Camera sequences"],
                      ["capture", "Captures"],
                    ].map(([value, label]) => (
                      <DropdownMenuRadioItem key={value} value={value}>
                        {label}
                      </DropdownMenuRadioItem>
                    ))}
                  </DropdownMenuRadioGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuRadioGroup
                    value={sort}
                    onValueChange={(value) => setSort(String(value))}
                    aria-label="Sort projects"
                  >
                    <DropdownMenuLabel>Sort by</DropdownMenuLabel>
                    <DropdownMenuRadioItem value="recent">
                      Last opened
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem value="name">
                      Name
                    </DropdownMenuRadioItem>
                  </DropdownMenuRadioGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            {filter !== "all" && (
              <button
                className="welcome-active-filter"
                onClick={() => setFilter("all")}
              >
                {filter === "pinned"
                  ? "Pinned projects"
                  : filter === "capture"
                    ? "Captures"
                    : filter === "render"
                      ? "Still renders"
                      : "Camera sequences"}
                <span>Clear ×</span>
              </button>
            )}
            <div className="welcome-project-list">
              {projects.map((project) => (
                <div className="welcome-project" key={project.id}>
                  <button
                    className="welcome-project-open"
                    onClick={() => onResume(project)}
                  >
                    <span className="welcome-project-icon">
                      <WorkspaceIcon page={project.page} size={23} />
                    </span>
                    <span className="welcome-project-name">
                      <strong>{project.name}</strong>
                      <small>
                        {projectLabels[project.page]}
                        {project.sourceName ? ` · ${project.sourceName}` : ""}
                      </small>
                    </span>
                    {project.pinned && <Pin size={13} aria-label="Pinned" />}
                    <time dateTime={new Date(project.updated).toISOString()}>
                      {new Date(project.updated).toLocaleDateString(undefined, {
                        month: "short",
                        day: "numeric",
                      })}
                    </time>
                  </button>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          variant="ghost"
                          size="icon"
                          className="welcome-project-more"
                          aria-label={`Actions for ${project.name}`}
                        />
                      }
                    >
                      <MoreHorizontal size={17} />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      align="end"
                      className="welcome-project-menu"
                    >
                      <DropdownMenuItem
                        onClick={() =>
                          onRecentAction(project.id, { type: "pin" })
                        }
                      >
                        <Pin size={14} />
                        {project.pinned ? "Unpin" : "Pin project"}
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        onClick={() =>
                          setDetails(details === project.id ? "" : project.id)
                        }
                      >
                        <Info size={14} />
                        {details === project.id
                          ? "Hide details"
                          : "Project details"}
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        onClick={() =>
                          onRecentAction(project.id, { type: "remove" })
                        }
                      >
                        <Trash2 size={14} />
                        Remove from recent
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  {details === project.id && (
                    <div className="welcome-project-details">
                      <RecentProjectDetails
                        project={project}
                        worlds={worlds}
                        onAction={onRecentAction}
                      />
                    </div>
                  )}
                </div>
              ))}
              {!projects.length && (
                <div className="welcome-no-matches">
                  <p>No matching projects.</p>
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setQuery("");
                      setFilter("all");
                    }}
                  >
                    Clear search and filters
                  </Button>
                </div>
              )}
            </div>
          </section>
        ) : (
          <div className="welcome-first-project">
            <h2>Recently opened</h2>
            <p>
              Start a capture or open a world to begin. Your recent projects
              will be kept here so you can pick up where you left off.
            </p>
          </div>
        )}
      </div>
    </section>
  );
}

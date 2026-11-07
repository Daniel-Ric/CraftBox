import { useEffect, useState } from "react";
import StudioInput from "./StudioInput";
import { Pin, Copy, Trash2, WandSparkles } from "lucide-react";
import {
  projectLabels,
  type StudioProject,
  type RecentProjectAction,
} from "./start-project";
import type { World } from "./bridge";

export default function RecentProjectDetails({
  project,
  worlds,
  onAction,
}: {
  project: StudioProject;
  worlds: World[];
  onAction?: (id: string, action: RecentProjectAction) => void;
}) {
  const [name, setName] = useState(project.name);
  useEffect(() => setName(project.name), [project.id, project.name]);
  const world = worlds.find((w) => w.id === project.worldId);
  const width = Math.round(
    project.aspect >= 1 ? project.edge : project.edge * project.aspect,
  );
  const height = Math.round(
    project.aspect >= 1 ? project.edge / project.aspect : project.edge,
  );
  return (
    <>
      {onAction ? (
        <form
          className="recent-name-form"
          onSubmit={(e) => {
            e.preventDefault();
            if (name.trim()) onAction(project.id, { type: "rename", name });
          }}
        >
          <label>
            Recent project name
            <StudioInput
              value={name}
              maxLength={100}
              required
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <button
            className="studio-button"
            disabled={!name.trim() || name.trim() === project.name}
          >
            Save name
          </button>
        </form>
      ) : (
        <strong>{project.name}</strong>
      )}
      <dl className="properties-list">
        <dt>Workspace</dt>
        <dd>{projectLabels[project.page]}</dd>
        <dt>Source</dt>
        <dd>
          {world?.name ||
            project.sourceName ||
            project.address ||
            "Choose when opening"}
        </dd>
        <dt>Output</dt>
        <dd>
          {["render", "cinematic"].includes(project.page)
            ? `${width} × ${height} px`
            : "Bedrock capture"}
        </dd>
        {project.page === "cinematic" && (
          <>
            <dt>Sequence</dt>
            <dd>
              {project.duration} s / {project.fps} fps
            </dd>
          </>
        )}
        <dt>Last opened</dt>
        <dd>
          {new Date(project.updated).toLocaleString(undefined, {
            dateStyle: "medium",
            timeStyle: "short",
          })}
        </dd>
      </dl>
      {onAction && (
        <div className="recent-project-actions">
          <button
            className="studio-button"
            aria-label={project.pinned ? "Unpin project" : "Pin project"}
            aria-pressed={!!project.pinned}
            onClick={() => onAction(project.id, { type: "pin" })}
          >
            <Pin size={14} />
            {project.pinned ? "Unpin" : "Pin"}
          </button>
          <button
            className="studio-button"
            aria-label="Duplicate project setup"
            title="Copy the source, output format and render settings into a new project."
            onClick={() => onAction(project.id, { type: "duplicate" })}
          >
            <Copy size={14} />
            Duplicate setup
          </button>
          <button
            className="studio-button"
            onClick={() =>
              onAction(project.id, {
                type: "automatic",
                sourceName: world?.name,
              })
            }
          >
            <WandSparkles size={14} />
            Use automatic name
          </button>
          <button
            className="studio-button"
            onClick={() => onAction(project.id, { type: "remove" })}
          >
            <Trash2 size={14} />
            Remove from recent
          </button>
          <small>
            Removing an entry keeps its world and saved project files.
          </small>
        </div>
      )}
    </>
  );
}

import { expect, it } from "vitest";
import {
  loadRecent,
  rememberProject,
  projectDefaults,
  recentKey,
} from "./start-project";
it("updates recent documents without replaying a one-time native import request", () => {
  localStorage.removeItem(recentKey);
  const project = {
    ...projectDefaults,
    id: "recent-test",
    name: "Insel",
    worldId: "world-1",
    openKind: "archive" as const,
    sourceKind: "archive" as const,
    updated: 1,
  };
  rememberProject(project);
  rememberProject({ ...project, name: "Insel final", worldId: "world-2" });
  expect(loadRecent()).toEqual([
    expect.objectContaining({
      id: "recent-test",
      name: "Insel final",
      worldId: "world-2",
      sourceKind: "archive",
    }),
  ]);
  expect(loadRecent()[0].openKind).toBeUndefined();
  localStorage.removeItem(recentKey);
});

import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { act, cleanup, render as testingRender, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import StudioFlyouts from "../src/StudioFlyouts";
function render(ui: ReactNode) { return testingRender(<>{ui}<StudioFlyouts /></>); }
import ExportTasks from "../src/ExportTasks";
import { exportTasks, runExportTask } from "../src/export-tasks";
import { api } from "../src/bridge";

beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  cleanup();
  exportTasks.clearCompleted();
  delete window.go;
  vi.useRealTimers();
});

it("removes the completed image export dialog and footer indicator", async () => {
  render(<ExportTasks />);
  let task: ReturnType<typeof exportTasks.begin>;
  act(() => { task = exportTasks.begin("Render image"); });
  await act(async () => {});
  expect(screen.queryByRole("dialog")).not.toBeNull();
  act(() => task.complete());
  await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByRole("button", { name: "Show export tasks", hidden: true })).toBeNull();
});

it("shows progress for project saves even when they finish within one render", async () => {
  window.go = { desktop: { App: {
    SaveStudioProjectForTab: async () => "scene.craftbox-studio",
  } as unknown as NonNullable<NonNullable<Window["go"]>["desktop"]["App"]> } };
  render(<ExportTasks />);
  await act(async () => {
    expect(await api.SaveStudioProjectForTab("tab", "world", "Scene", "{}", true)).toBe("scene.craftbox-studio");
  });
  expect(screen.queryByRole("dialog")).not.toBeNull();
  expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("100");
  await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  expect(screen.queryByRole("button", { name: "Show export tasks", hidden: true })).toBeNull();
});

it.each(["SaveStudioProject", "SaveStudioProjectForTab", "SaveWorkspaceProject"])("tracks %s until its file write resolves", async name => {
  let finish!: (path: string) => void;
  const saved = new Promise<string>(resolve => { finish = resolve; });
  window.go = { desktop: { App: { [name]: () => saved } as unknown as NonNullable<NonNullable<Window["go"]>["desktop"]["App"]> } };
  let pending!: Promise<unknown>;
  act(() => { pending = (api[name as keyof typeof api] as (...args: unknown[]) => Promise<unknown>)(); });
  expect(exportTasks.snapshot().some(task => task.status === "running")).toBe(true);
  await act(async () => { finish("scene.project"); await pending; });
  expect(exportTasks.snapshot()[0].status).toBe("succeeded");
});

it("keeps a failed export visible for error inspection", async () => {
  render(<ExportTasks />);
  await act(async () => {
    await runExportTask("Save image", async () => { throw Error("Disk full"); }).catch(() => {});
  });
  await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  expect(screen.queryByRole("dialog")).not.toBeNull();
  expect(screen.getByRole("alert").textContent).toContain("Disk full");
});

it("keeps a second export active when the first export finishes", async () => {
  render(<ExportTasks />);
  let first!: ReturnType<typeof exportTasks.begin>;
  let second!: ReturnType<typeof exportTasks.begin>;
  act(() => {
    first = exportTasks.begin("Render image");
    second = exportTasks.begin("Save project");
    first.complete();
  });
  await act(async () => { await vi.advanceTimersByTimeAsync(3000); });
  expect(screen.queryByRole("dialog")).not.toBeNull();
  expect(exportTasks.snapshot().find(task => task.id === second.id)?.status).toBe("running");
  act(() => second.complete());
  await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  expect(screen.queryByRole("button", { name: "Show export tasks", hidden: true })).toBeNull();
});

it("clears a cancelled project destination without reporting success", async () => {
  window.go = { desktop: { App: {
    SaveWorkspaceProject: async () => "",
  } as unknown as NonNullable<NonNullable<Window["go"]>["desktop"]["App"]> } };
  render(<ExportTasks />);
  await act(async () => { await api.SaveWorkspaceProject("tab", "Scene", "{}", true); });
  expect(exportTasks.snapshot()[0].status).toBe("cancelled");
  await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByRole("button", { name: "Show export tasks", hidden: true })).toBeNull();
});

it("cleans later successful exports while retaining an earlier error", async () => {
  render(<ExportTasks />);
  await act(async () => {
    await runExportTask("Save image", async () => { throw Error("Disk full"); }).catch(() => {});
  });
  for (let i = 0; i < 6; i++) {
    await act(async () => { await runExportTask("Save project", async () => "scene.project"); });
    await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  }
  expect(exportTasks.snapshot().map(task => task.status)).toEqual(["failed"]);
  expect(screen.getByRole("alert").textContent).toContain("Disk full");
});

it("cancels pending dismissal when another export starts", async () => {
  render(<ExportTasks />);
  await act(async () => { await runExportTask("Save image", async () => "scene.png"); });
  await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
  let task!: ReturnType<typeof exportTasks.begin>;
  act(() => { task = exportTasks.begin("Save project"); });
  await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
  expect(screen.queryByRole("dialog")).not.toBeNull();
  expect(screen.getByRole("button", { name: "Show export tasks", hidden: true }).textContent).toContain("Save project");
  act(() => task.complete());
  await act(async () => { await vi.advanceTimersByTimeAsync(2000); });
  expect(screen.queryByRole("button", { name: "Show export tasks", hidden: true })).toBeNull();
});


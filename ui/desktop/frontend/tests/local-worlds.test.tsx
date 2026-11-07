import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import LocalWorldsDialog from "../src/LocalWorldsDialog";
import StudioFlyouts from "../src/StudioFlyouts";
import { exportTasks } from "../src/export-tasks";
const world = {
  id: "world-one",
  name: "My singleplayer world",
  folder: "C:/Minecraft/worlds/one",
  installation: "Minecraft",
  modified: 1,
  template: true,
  marketplace: true,
};
afterEach(() => {
  cleanup();
  delete window.go;
  exportTasks.clearCompleted();
});
it("requires an explicit backup choice when protected packs prevent a complete share", async () => {
  const exportWorld = vi.fn().mockResolvedValue({
    path: "C:/Exports/backup.mcworld",
    portable: false,
    bytes: 100,
    files: 8,
    packs: 1,
    issues: [],
  });
  const inspect = vi.fn().mockResolvedValue({
    world,
    packs: [
      {
        id: "pack",
        name: "Marketplace texture pack",
        version: "1.0.0",
        kind: "resource",
        status: "protected",
        embedded: true,
        requiredBy: "World",
      },
    ],
    issues: [
      {
        severity: "error",
        code: "protected_pack",
        message: "Protected content is kept unchanged.",
      },
    ],
    portable: false,
    files: 8,
    bytes: 100,
    fingerprint: "reviewed-world",
  });
  window.go = {
    desktop: {
      App: {
        GetLocalWorlds: async () => ({ worlds: [world] }),
        InspectLocalWorld: inspect,
        ExportLocalWorld: exportWorld,
        GetLocalWorldExportProgress: async () => ({ running: false }),
      } as any,
    },
  };
  render(
    <>
      <LocalWorldsDialog open onOpenChange={() => {}} />
      <StudioFlyouts />
    </>,
  );
  fireEvent.click(
    await screen.findByRole("button", { name: world.name, exact: true }),
  );
  await screen.findByText("Marketplace texture pack");
  expect(
    (
      screen.getByRole("button", {
        name: "Export .mcworld",
        exact: true,
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  fireEvent.click(screen.getByRole("switch", { name: "Backup only" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Export backup", exact: true }),
  );
  await waitFor(() => expect(exportWorld).toHaveBeenCalled());
  expect(exportWorld.mock.calls[0][0]).toMatchObject({
    worldId: world.id,
    fingerprint: "reviewed-world",
    options: { includeGlobal: false, backupOnly: true },
  });
  expect(exportWorld.mock.calls[0][0].jobId).toMatch(/^[\da-f-]{36}$/);
});
it("rechecks global texture dependencies when the option changes", async () => {
  const inspect = vi.fn().mockResolvedValue({
    world,
    packs: [],
    issues: [],
    portable: true,
    files: 8,
    bytes: 100,
    fingerprint: "world",
  });
  window.go = {
    desktop: {
      App: {
        GetLocalWorlds: async () => ({ worlds: [world] }),
        InspectLocalWorld: inspect,
      } as any,
    },
  };
  render(
    <>
      <LocalWorldsDialog open onOpenChange={() => {}} />
      <StudioFlyouts />
    </>,
  );
  fireEvent.click(
    await screen.findByRole("button", { name: world.name, exact: true }),
  );
  await screen.findByText("All required files found");
  fireEvent.click(
    screen.getByRole("switch", { name: "Include global texture packs" }),
  );
  await waitFor(() =>
    expect(inspect.mock.calls.at(-1)?.[0].options.includeGlobal).toBe(true),
  );
});

it("refreshes the selected world inspection and export fingerprint", async () => {
  const inspect = vi
    .fn()
    .mockResolvedValue({
      world,
      packs: [],
      issues: [],
      portable: true,
      files: 8,
      bytes: 100,
      fingerprint: "first",
    });
  const exportWorld = vi.fn().mockResolvedValue(null);
  window.go = {
    desktop: {
      App: {
        GetLocalWorlds: async () => ({ worlds: [world] }),
        InspectLocalWorld: inspect,
        ExportLocalWorld: exportWorld,
        GetLocalWorldExportProgress: async () => ({ running: false }),
      } as any,
    },
  };
  render(<LocalWorldsDialog open onOpenChange={() => {}} />);
  fireEvent.click(
    await screen.findByRole("button", { name: world.name, exact: true }),
  );
  await screen.findByText("All required files found");
  inspect.mockResolvedValue({
    world,
    packs: [],
    issues: [],
    portable: true,
    files: 9,
    bytes: 200,
    fingerprint: "refreshed",
  });
  fireEvent.click(screen.getByRole("button", { name: "Refresh local worlds" }));
  await waitFor(() => expect(inspect.mock.calls.length).toBeGreaterThan(1));
  await screen.findByText("All required files found");
  fireEvent.click(
    screen.getByRole("button", { name: "Export .mcworld", exact: true }),
  );
  await waitFor(() => expect(exportWorld).toHaveBeenCalled());
  expect(exportWorld.mock.calls[0][0].fingerprint).toBe("refreshed");
});

import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import CraftBoxCLI from "../src/CraftBoxCLI";
afterEach(() => {
  cleanup();
  delete window.go;
});
it("retains pending command results when the transcript is cleared", async () => {
  let finish!: (value: any) => void;
  window.go = {
    desktop: {
      App: {
        GetBootstrap: async () => ({ state: { worlds: [] }, commands: [] }),
        Snapshot: () =>
          new Promise((resolve) => {
            finish = resolve;
          }),
      } as any,
    },
  };
  render(
    <CraftBoxCLI
      logs={[]}
      history={[]}
      onHistory={() => {}}
      onExit={() => {}}
      onNavigate={() => {}}
      onExport={async () => ""}
      initialCommand="status"
    />,
  );
  await screen.findByText("Working…");
  fireEvent.click(screen.getByRole("button", { name: "Clear CLI transcript" }));
  finish({ running: false, worlds: [], packs: [], skins: [], connection: 0 });
  await waitFor(() =>
    expect(screen.getByRole("log").textContent).toContain('"running": false'),
  );
});
it("prints errors, keeps the CLI usable and executes quoted world commands", async () => {
  window.go = {
    desktop: {
      App: {
        GetBootstrap: async () => ({
          state: { worlds: [{ id: "a", name: "Hive Games" }] },
          commands: [],
        }),
        Snapshot: async () => ({
          worlds: [{ id: "a", name: "Hive Games", chunks: 120 }],
        }),
      } as any,
    },
  };
  const exit = vi.fn();
  render(
    <CraftBoxCLI
      logs={[]}
      history={[]}
      onHistory={() => {}}
      onExit={exit}
      onNavigate={() => {}}
      onExport={async () => ""}
    />,
  );
  const input = screen.getByRole("combobox");
  fireEvent.change(input, { target: { value: 'worlds show "Hive Games"' } });
  fireEvent.click(screen.getByRole("button", { name: "Run command" }));
  await waitFor(() =>
    expect(screen.getByRole("log").textContent).toContain('"chunks": 120'),
  );
  fireEvent.change(input, { target: { value: "missing command" } });
  fireEvent.click(screen.getByRole("button", { name: "Run command" }));
  await screen.findByText(/Unknown command: missing command/);
  expect(
    screen
      .getByRole("button", { name: "Application logs" })
      .hasAttribute("disabled"),
  ).toBe(false);
});

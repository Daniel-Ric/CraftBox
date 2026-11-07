import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import PacketInspector from "../src/PacketInspector";
import { DocumentScope } from "../src/document-scope";
import StudioFlyouts from "../src/StudioFlyouts";
afterEach(() => {
  cleanup();
  delete window.go;
});
it("keeps only surviving recordings selected after partial deletion and a cancelled retry", async () => {
  let remaining = ["A", "B"],
    retry = false;
  const remove = vi.fn(async (id: string) => {
    if (id === "B" && !retry) throw Error("File is busy");
    remaining = remaining.filter((value) => value !== id);
  });
  window.go = {
    desktop: {
      App: {
        GetPacketInspector: async () => ({
          captures: remaining.map((id) => ({
            id,
            name: "Recording " + id,
            count: 0,
            started: 0,
            bytes: 0,
            diskBytes: 0,
            dropped: 0,
            notes: "",
          })),
          settings: { recording: false },
          rules: [],
          active: "",
          connected: false,
          generation: "test",
          queueDropped: 0,
          error: "",
        }),
        DeletePacketCapture: remove,
      } as any,
    },
  };
  render(
    <>
      <DocumentScope.Provider
        value={{
          id: "packet-browser-test",
          active: true,
          onOpen: () => {},
          register: () => {},
        }}
      >
        <PacketInspector
          running={false}
          canStart={false}
          source=""
          onSource={() => {}}
          onStart={() => {}}
          onStop={() => {}}
        />
      </DocumentScope.Provider>
      <StudioFlyouts />
    </>,
  );
  await screen.findByRole("button", { name: /^Recording A/ });
  fireEvent.click(screen.getByRole("button", { name: "Select", exact: true }));
  fireEvent.click(screen.getByRole("button", { name: /^Recording A/ }));
  fireEvent.click(screen.getByRole("button", { name: /^Recording B/ }));
  fireEvent.click(screen.getByRole("button", { name: "Delete selected (2)" }));
  fireEvent.click(
    await screen.findByRole("button", {
      name: "Delete recording",
      exact: true,
    }),
  );
  await waitFor(() => expect(remove).toHaveBeenCalledTimes(2));
  await screen.findByRole("alert");
  fireEvent.click(screen.getByRole("button", { name: "Cancel", exact: true }));
  fireEvent.click(
    await screen.findByRole("button", { name: "Delete selected (1)" }),
  );
  const dialog = await screen.findByRole("dialog", {
    name: "Delete recording?",
  });
  expect(dialog.textContent).not.toContain("Recording A");
  expect(dialog.textContent).toContain("Recording B");
  retry = true;
  fireEvent.click(
    screen.getByRole("button", { name: "Delete recording", exact: true }),
  );
  await waitFor(() => expect(remaining).toEqual([]));
  expect(remove.mock.calls.map(([id]) => id)).toEqual(["A", "B", "B"]);
});

import { afterEach, expect, it, vi } from "vitest";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  within,
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
  await act(async () => {
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
          <PacketInspector source="" onSource={() => {}} />
        </DocumentScope.Provider>
        <StudioFlyouts />
      </>,
    );
  });
  fireEvent.click(screen.getByRole("button", { name: "Manage recordings" }));
  const recordings = within(screen.getByRole("dialog", { name: "Recordings" }));
  fireEvent.click(
    recordings.getByRole("button", { name: "Select", exact: true }),
  );
  fireEvent.click(recordings.getByRole("button", { name: /^Recording A/ }));
  fireEvent.click(recordings.getByRole("button", { name: /^Recording B/ }));
  fireEvent.click(
    recordings.getByRole("button", { name: "Delete selected (2)" }),
  );
  const firstConfirmation = within(
    screen.getByRole("dialog", { name: "Delete 2 recordings?" }),
  );
  await act(async () => {
    fireEvent.click(
      firstConfirmation.getByRole("button", {
        name: "Delete recording",
        exact: true,
      }),
    );
  });
  expect(remove.mock.calls.map(([id]) => id)).toEqual(["A", "B"]);
  expect(await firstConfirmation.findByText("File is busy")).toBeTruthy();
  fireEvent.click(
    firstConfirmation.getByRole("button", { name: "Cancel", exact: true }),
  );
  fireEvent.click(
    recordings.getByRole("button", { name: "Delete selected (1)" }),
  );
  const dialog = screen.getByRole("dialog", {
    name: "Delete recording?",
  });
  expect(dialog.textContent).not.toContain("Recording A");
  expect(dialog.textContent).toContain("Recording B");
  retry = true;
  await act(async () => {
    fireEvent.click(
      within(dialog).getByRole("button", {
        name: "Delete recording",
        exact: true,
      }),
    );
  });
  expect(remaining).toEqual([]);
  expect(remove.mock.calls.map(([id]) => id)).toEqual(["A", "B", "B"]);
}, 10_000);

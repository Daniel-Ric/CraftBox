import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useCloseDocuments, type CloseEntry } from "../src/DocumentTabs";
import { requestSessionQuit } from "../src/session-quit";
import { api } from "../src/bridge";
import { Toaster } from "sonner";

vi.mock("../src/bridge", () => ({
  api: { StopCommand: vi.fn(), ConfirmSessionClose: vi.fn() },
}));
afterEach(() => { cleanup(); vi.resetAllMocks(); });

function QuitFlow({ save = true }: { save?: boolean }) {
  const close = useCloseDocuments();
  const entry: CloseEntry = {
    id: "pack-one", area: "packs", name: "Enchanted Hub Pack",
    session: {
      name: "Enchanted Hub Pack",
      dirty: true, busy: false,
      save: async () => {
        if (save) entry.session.dirty = false;
        return save;
      },
    },
  };
  return <>
    <Toaster />
    <button onClick={() => requestSessionQuit(
      (done) => close.request([entry], done, true),
      () => {},
    )}>Quit session</button>
    {close.dialog}
  </>;
}

it("keeps the session running when project closing is cancelled", () => {
  render(<QuitFlow />);
  fireEvent.click(screen.getByRole("button", { name: "Quit session" }));
  expect(screen.getByText("Save projects before quitting?")).toBeTruthy();
  expect(api.StopCommand).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(api.StopCommand).not.toHaveBeenCalled();
  expect(api.ConfirmSessionClose).not.toHaveBeenCalled();
});

it("discards once and stops before confirming the native close", async () => {
  let stopped = false;
  vi.mocked(api.StopCommand).mockImplementation(async () => { stopped = true; });
  vi.mocked(api.ConfirmSessionClose).mockImplementation(async () => {
    expect(stopped).toBe(true);
  });
  render(<QuitFlow />);
  fireEvent.click(screen.getByRole("button", { name: "Quit session" }));
  fireEvent.click(screen.getByRole("button", { name: "Discard all" }));
  await waitFor(() => expect(api.ConfirmSessionClose).toHaveBeenCalledTimes(1));
  expect(api.StopCommand).toHaveBeenCalledTimes(1);
  expect(screen.queryByText("Save projects before quitting?")).toBeNull();
});

it("does not stop or quit when saving the project fails", async () => {
  render(<QuitFlow save={false} />);
  fireEvent.click(screen.getByRole("button", { name: "Quit session" }));
  fireEvent.click(screen.getByRole("button", { name: "Save selected and quit" }));
  await waitFor(() => expect(screen.getByText(/was not saved/)).toBeTruthy());
  expect(api.StopCommand).not.toHaveBeenCalled();
  expect(api.ConfirmSessionClose).not.toHaveBeenCalled();
});

it("saves the selected project and completes one quit", async () => {
  vi.mocked(api.StopCommand).mockResolvedValue(undefined);
  vi.mocked(api.ConfirmSessionClose).mockResolvedValue(undefined);
  render(<QuitFlow />);
  fireEvent.click(screen.getByRole("button", { name: "Quit session" }));
  fireEvent.click(screen.getByRole("button", { name: "Save selected and quit" }));
  await waitFor(() => expect(api.ConfirmSessionClose).toHaveBeenCalledTimes(1));
  expect(api.StopCommand).toHaveBeenCalledTimes(1);
  expect(screen.queryByText("Save projects before quitting?")).toBeNull();
});

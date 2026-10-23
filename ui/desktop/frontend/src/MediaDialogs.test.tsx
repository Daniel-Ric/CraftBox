import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { ImportWorldDialog, MediaExportDialog } from "./MediaDialogs";
it("shows the current scene preview and sends an editable filename to export", async () => {
  const save = vi.fn(),
    user = userEvent.setup();
  render(
    <MediaExportDialog
      open
      onOpenChange={() => {}}
      format="png"
      edge={1920}
      onEdge={() => {}}
      aspect={16 / 9}
      name="Shot"
      onExport={save}
      preview="data:image/png;base64,preview"
    />,
  );
  expect(
    screen.getByRole("img", { name: "Current render view" }),
  ).toHaveAttribute("src", "data:image/png;base64,preview");
  await user.clear(screen.getByLabelText("File name"));
  await user.type(screen.getByLabelText("File name"), "Insel Sunset");
  await user.click(screen.getByRole("button", { name: "Export …" }));
  expect(save).toHaveBeenCalledWith("Insel Sunset");
});
it("selects an import source before invoking the native picker", async () => {
  const open = vi.fn(),
    user = userEvent.setup();
  render(
    <ImportWorldDialog
      open
      onOpenChange={() => {}}
      kind="archive"
      onImport={open}
    />,
  );
  await user.click(screen.getByRole("button", { name: /World folder/ }));
  await user.click(screen.getByRole("button", { name: "Select …" }));
  expect(open).toHaveBeenCalledWith("folder");
});
it("reports the exact encoder size for CinemaScope video", () => {
  render(
    <MediaExportDialog
      open
      onOpenChange={() => {}}
      format="mp4"
      edge={3840}
      onEdge={() => {}}
      aspect={2.39}
      name="Film"
      onExport={() => {}}
    />,
  );
  expect(screen.getByText("3840 × 1606 pixels")).toBeInTheDocument();
});

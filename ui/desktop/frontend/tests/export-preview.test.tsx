import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MediaExportDialog } from "../src/MediaDialogs";

afterEach(cleanup);
const preview = "data:image/svg+xml," + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="1600" height="900"><rect width="1600" height="900" fill="#54765d"/></svg>');
function show(src: string | undefined = preview) {
  const onExport = vi.fn();
  render(<MediaExportDialog open onOpenChange={vi.fn()} format="png" edge={4096} onEdge={vi.fn()} aspect={16/9} name="CrabSMP" title="Export image" preview={src} onExport={onExport} />);
  return onExport;
}

it("opens a large image preview without exporting and returns to the export dialog on Escape", async () => {
  const onExport = show();
  expect(screen.getByRole("button", { name: "Enlarge image preview" })).not.toBeNull();
  fireEvent.click(screen.getByAltText("Current render view"));
  const enlarged = await screen.findByRole("dialog", { name: "Image preview" });
  expect(enlarged.querySelector("img")?.getAttribute("src")).toBe(preview);
  expect(onExport).not.toHaveBeenCalled();
  fireEvent.keyDown(enlarged, { key: "Escape", code: "Escape" });
  await waitFor(() => expect(screen.queryByRole("dialog", { name: "Image preview" })).toBeNull());
  expect(screen.getByRole("dialog", { name: "Export image" })).not.toBeNull();
});

it("offers a keyboard-accessible preview button only when an image is available", () => {
  show("");
  expect(screen.queryByRole("button", { name: "Enlarge image preview" })).toBeNull();
});

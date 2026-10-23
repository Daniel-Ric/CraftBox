import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import DesktopChrome from "./DesktopChrome";
const props = () => ({
  page: "render",
  onPage: vi.fn(),
  onCommand: vi.fn(),
  onAccount: vi.fn(),
  onSettings: vi.fn(),
  onLogs: vi.fn(),
  onHelp: vi.fn(),
  onTheme: vi.fn(),
  theme: "dark",
  account: "",
  running: false,
  status: "Ready",
  logsOpen: false,
  editor: { ready: true, busy: false, undo: false, redo: false },
});
it("opens a real application menu and routes a file action", async () => {
  const p = props(),
    user = userEvent.setup();
  render(<DesktopChrome {...p} />);
  expect(screen.queryByRole("complementary")).not.toBeInTheDocument();
  await user.click(screen.getByRole("menuitem", { name: "File" }));
  await user.click(
    await screen.findByRole("menuitem", { name: /Open world file/ }),
  );
  expect(p.onCommand).toHaveBeenCalledWith("open-archive");
});
it("keeps workspace navigation and keyboard shortcuts accessible without a sidebar", () => {
  const p = props();
  render(<DesktopChrome {...p} />);
  fireEvent.click(screen.getByRole("button", { name: "Cinematic editor" }));
  expect(p.onPage).toHaveBeenCalledWith("cinematic");
  fireEvent.keyDown(window, {
    key: "§",
    code: "Digit3",
    ctrlKey: true,
    shiftKey: true,
  });
  expect(p.onPage).toHaveBeenCalledWith("skins");
});
it("does not run export shortcuts while the editor is busy", () => {
  const p = props();
  p.editor.busy = true;
  render(<DesktopChrome {...p} />);
  fireEvent.keyDown(window, { key: "e", ctrlKey: true, shiftKey: true });
  expect(p.onCommand).not.toHaveBeenCalled();
});
it("searches actual commands and executes the selected result", async () => {
  const p = props(),
    user = userEvent.setup();
  render(<DesktopChrome {...p} />);
  fireEvent.keyDown(window, { key: "k", ctrlKey: true });
  await user.type(screen.getByLabelText("Search commands"), "PNG");
  await user.click(
    screen.getByRole("button", { name: /Export image as PNG/ }),
  );
  expect(p.onCommand).toHaveBeenCalledWith("export-png");
  expect(
    screen.queryByRole("dialog", { name: "Find command" }),
  ).not.toBeInTheDocument();
});
it("does not send editor shortcuts behind an open dialog", () => {
  const p = props();
  render(
    <>
      <DesktopChrome {...p} />
      <div role="dialog" data-open="">
        Export settings
      </div>
    </>,
  );
  fireEvent.keyDown(window, { key: "o", ctrlKey: true });
  expect(p.onCommand).not.toHaveBeenCalled();
});

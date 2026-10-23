import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { it, expect, vi } from "vitest";
import PreferencesDialog, {
  loadWorkspacePreferences,
} from "./PreferencesDialog";
it("renders separate preference categories and persists workspace options", async () => {
  localStorage.clear();
  const user = userEvent.setup(),
    onTheme = vi.fn(),
    check = vi.fn();
  render(
    <PreferencesDialog
      open
      onOpenChange={() => {}}
      theme="dark"
      onTheme={onTheme}
      version="Studio"
      minecraft="1.26"
      updateBusy={false}
      onCheckUpdate={check}
    />,
  );
  await user.click(screen.getByRole("button", { name: "Light" }));
  expect(onTheme).toHaveBeenCalledWith("light");
  await user.click(screen.getByRole("tab", { name: "Workspace" }));
  expect(
    screen.queryByRole("button", { name: "Light" }),
  ).not.toBeInTheDocument();
  await user.click(screen.getByLabelText("Show welcome dialog on startup"));
  expect(loadWorkspacePreferences().showWelcome).toBe(false);
  await user.selectOptions(
    screen.getByLabelText("Controls"),
    "comfortable",
  );
  expect(document.documentElement.dataset.density).toBe("comfortable");
  await user.click(screen.getByRole("tab", { name: "Version & updates" }));
  await user.click(screen.getByRole("button", { name: "Check for updates" }));
  expect(check).toHaveBeenCalledOnce();
});

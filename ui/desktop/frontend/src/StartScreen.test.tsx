import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { NewProjectDialog, StartScreen, StartDialog } from "./StartScreen";
it("rejects unsupported output aspect ratios before creating a project", async () => {
  const user = userEvent.setup(),
    create = vi.fn();
  render(
    <NewProjectDialog
      open
      onOpenChange={() => {}}
      worlds={[]}
      onCreate={create}
    />,
  );
  await user.clear(screen.getByLabelText("Width"));
  await user.type(screen.getByLabelText("Width"), "4096");
  await user.clear(screen.getByLabelText("Height"));
  await user.type(screen.getByLabelText("Height"), "64");
  expect(screen.getByRole("button", { name: "Create" })).toBeDisabled();
  expect(screen.getByRole("alert")).toHaveTextContent("1:4 and 4:1");
});
it("creates custom output dimensions and swaps orientation", async () => {
  const create = vi.fn(),
    user = userEvent.setup();
  render(
    <NewProjectDialog
      open
      onOpenChange={() => {}}
      worlds={[]}
      onCreate={create}
    />,
  );
  await user.clear(screen.getByLabelText("Width"));
  await user.type(screen.getByLabelText("Width"), "2400");
  await user.clear(screen.getByLabelText("Height"));
  await user.type(screen.getByLabelText("Height"), "1600");
  await user.click(
    screen.getByRole("button", { name: "Swap orientation" }),
  );
  expect(screen.getByLabelText("Width")).toHaveValue(1600);
  expect(screen.getByLabelText("Height")).toHaveValue(2400);
  await user.click(screen.getByRole("button", { name: "Create" }));
  expect(create).toHaveBeenCalledWith(
    expect.objectContaining({ edge: 2400, aspect: 2 / 3 }),
  );
});
it("creates a cinematic document with real sequence and output settings", async () => {
  const create = vi.fn(),
    user = userEvent.setup();
  render(
    <NewProjectDialog
      open
      onOpenChange={() => {}}
      worlds={[]}
      onCreate={create}
    />,
  );
  await user.click(screen.getByRole("tab", { name: "Camera sequence" }));
  await user.clear(screen.getByLabelText("Project name"));
  await user.type(screen.getByLabelText("Project name"), "Insel-Film");
  await user.selectOptions(screen.getByLabelText("Format preset"), "vertical");
  await user.clear(screen.getByLabelText("Duration (seconds)"));
  await user.type(screen.getByLabelText("Duration (seconds)"), "30");
  await user.click(screen.getByRole("button", { name: "Create" }));
  expect(create).toHaveBeenCalledWith(
    expect.objectContaining({
      name: "Insel-Film",
      page: "cinematic",
      aspect: 9 / 16,
      duration: 30,
      edge: 1920,
    }),
  );
});
it("shows welcome actions in a dismissible modal and routes folder opening", async () => {
  const close = vi.fn(),
    open = vi.fn(),
    user = userEvent.setup();
  render(
    <StartDialog
      open
      onOpenChange={close}
      recent={[]}
      worlds={[]}
      onCreate={() => {}}
      onOpen={open}
      onResume={() => {}}
      onPage={() => {}}
    />,
  );
  expect(
    screen.getByRole("dialog", { name: "Welcome to CraftBox" }),
  ).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Open world folder" }));
  expect(open).toHaveBeenCalledWith("folder");
  await user.click(
    screen.getByRole("button", { name: "Open workspace" }),
  );
  expect(close).toHaveBeenCalledWith(false);
});
it("offers an empty recent state and routes direct opening without a fake project", async () => {
  const open = vi.fn(),
    user = userEvent.setup();
  render(
    <StartScreen
      recent={[]}
      worlds={[]}
      onCreate={() => {}}
      onOpen={open}
      onResume={() => {}}
      onPage={() => {}}
    />,
  );
  expect(screen.getByText("No projects yet")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Open world file" }));
  expect(open).toHaveBeenCalledWith("archive");
});

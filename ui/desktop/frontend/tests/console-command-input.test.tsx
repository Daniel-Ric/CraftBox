import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import ConsoleCommandInput from "../src/ConsoleCommandInput";
afterEach(cleanup);
it("shows a custom caret only for a focused, collapsed text selection", () => {
  const {container}=render(<ConsoleCommandInput mode="console" history={[]} onHistory={()=>{}} onRun={()=>{}}/>);
  const input=screen.getByRole('combobox') as HTMLInputElement;
  fireEvent.focus(input);
  fireEvent.change(input,{target:{value:'help'}});
  input.setSelectionRange(4,4);fireEvent.select(input);
  expect(container.querySelector('.command-caret')).not.toBeNull();
  input.setSelectionRange(0,4);fireEvent.select(input);
  expect(container.querySelector('.command-caret')).toBeNull();
});
it("Enter runs the typed command without silently selecting a different suggestion", () => {
  render(
    <ConsoleCommandInput
      mode="cli"
      history={[]}
      onHistory={() => {}}
      onRun={() => {}}
    />,
  );
  const input = screen.getByRole("combobox");
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: "help" } });
  expect(fireEvent.keyDown(input, { key: "Enter" })).toBe(true);
});
it("completes with Tab without running the command", () => {
  const run = vi.fn();
  render(
    <ConsoleCommandInput
      mode="console"
      history={[]}
      onHistory={() => {}}
      onRun={run}
    />,
  );
  const input = screen.getByRole("combobox");
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: "/lev" } });
  fireEvent.keyDown(input, { key: "Tab" });
  expect((input as HTMLInputElement).value).toBe("/level ");
  expect(run).not.toHaveBeenCalled();
  fireEvent.change(input, { target: { value: "/level error" } });
  fireEvent.click(screen.getByRole("button", { name: "Run command" }));
  expect(run).toHaveBeenCalledWith("/level error");
});
it("restores the draft after navigating history and supports history search", () => {
  render(
    <ConsoleCommandInput
      mode="console"
      history={["/level error", "/find timeout"]}
      onHistory={() => {}}
      onRun={() => {}}
    />,
  );
  const input = screen.getByRole("combobox") as HTMLInputElement;
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: "unfinished draft" } });
  fireEvent.keyDown(input, { key: "ArrowUp" });
  expect(input.value).toBe("/find timeout");
  fireEvent.keyDown(input, { key: "ArrowDown" });
  expect(input.value).toBe("unfinished draft");
  fireEvent.change(input, { target: { value: "timeout" } });
  fireEvent.keyDown(input, { key: "r", ctrlKey: true });
  expect(screen.getByRole("option").textContent).toContain("/find timeout");
  fireEvent.keyDown(input, { key: "Escape" });
  expect(input.value).toBe("timeout");
});
it("does not submit while busy or composing and clears the view with Ctrl+L", () => {
  const run = vi.fn(),
    clear = vi.fn();
  render(
    <ConsoleCommandInput
      mode="cli"
      history={[]}
      onHistory={() => {}}
      onRun={run}
      onClear={clear}
      busy
    />,
  );
  expect(
    screen
      .getByRole("button", { name: "Run command" })
      .hasAttribute("disabled"),
  ).toBe(true);
  fireEvent.keyDown(screen.getByRole("combobox"), { key: "l", ctrlKey: true });
  expect(clear).toHaveBeenCalled();
  expect(run).not.toHaveBeenCalled();
});

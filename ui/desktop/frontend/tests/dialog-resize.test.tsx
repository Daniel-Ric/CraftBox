import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "../src/components/ui/dialog";
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
it("enlarges dialogs with keyboard controls and never shrinks below their default size", () => {
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    function (this: HTMLElement) {
      return {
        width: parseFloat(this.style.width) || 400,
        height: parseFloat(this.style.height) || 240,
        left: 100,
        top: 100,
        right: 500,
        bottom: 340,
        x: 100,
        y: 100,
        toJSON() {},
      };
    },
  );
  render(
    <Dialog open>
      <DialogContent>
        <DialogTitle>Resize test</DialogTitle>
      </DialogContent>
    </Dialog>,
  );
  const resize = screen.getByRole("button", { name: "Resize dialog" }),
    dialog = screen.getByRole("dialog");
  fireEvent.keyDown(resize, { key: "ArrowRight" });
  fireEvent.keyDown(resize, { key: "ArrowDown" });
  expect(parseFloat(dialog.style.width)).toBeGreaterThan(400);
  expect(parseFloat(dialog.style.height)).toBeGreaterThan(240);
  for (let i = 0; i < 30; i++) {
    fireEvent.keyDown(resize, { key: "ArrowLeft" });
    fireEvent.keyDown(resize, { key: "ArrowUp" });
  }
  expect(parseFloat(dialog.style.width)).toBe(400);
  expect(parseFloat(dialog.style.height)).toBe(240);
});

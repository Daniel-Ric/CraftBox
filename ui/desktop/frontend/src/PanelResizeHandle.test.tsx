import { render, screen, fireEvent } from "@testing-library/react";
import { it, expect, vi } from "vitest";
import PanelResizeHandle from "./PanelResizeHandle";
it("resizes a dock by keyboard while respecting its bounds", () => {
  const change = vi.fn();
  render(
    <PanelResizeHandle
      label="Timeline height"
      value={240}
      min={160}
      max={360}
      onChange={change}
    />,
  );
  const divider = screen.getByRole("separator", { name: "Timeline height" });
  fireEvent.keyDown(divider, { key: "ArrowUp" });
  expect(change).toHaveBeenLastCalledWith(260);
  fireEvent.keyDown(divider, { key: "Home" });
  expect(change).toHaveBeenLastCalledWith(160);
  fireEvent.keyDown(divider, { key: "End" });
  expect(change).toHaveBeenLastCalledWith(360);
});

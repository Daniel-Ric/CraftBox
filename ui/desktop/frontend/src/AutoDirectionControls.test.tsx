import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import AutoDirectionControls from "./AutoDirectionControls";
import { analyseModel } from "./auto-direction";
import { renderDefaults } from "./render-settings";
const analysis = () =>
  analyseModel(
    [
      {
        position: [0, 0, 0],
        normal: [0, 1, 0],
        color: [0.2, 0.6, 0.3],
        area: 1,
        mode: "mask",
      },
      {
        position: [100, 100, 100],
        normal: [1, 0, 0],
        color: [0.8, 0.7, 0.6],
        area: 1,
        mode: "solid",
      },
    ],
    { lights: 2, triangles: 200 },
  );
it("applies a complete analysis in one action and restores the previous settings", async () => {
  const apply = vi.fn();
  render(
    <AutoDirectionControls
      ready
      settings={renderDefaults}
      onApply={apply}
      analyse={async () => analysis()}
    />,
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Auto-style scene" }),
  );
  await waitFor(() => expect(apply).toHaveBeenCalledTimes(1));
  expect(screen.getByText(/Scene looks/)).toBeVisible();
  fireEvent.click(
    screen.getByRole("button", {
      name: "Restore previous settings",
    }),
  );
  expect(apply.mock.calls[1][0]).toEqual(renderDefaults);
});
it("does not apply an analysis after cancellation", async () => {
  let resolve!: (x: ReturnType<typeof analysis>) => void;
  const apply = vi.fn();
  render(
    <AutoDirectionControls
      ready
      settings={renderDefaults}
      onApply={apply}
      analyse={() =>
        new Promise((r) => {
          resolve = r;
        })
      }
    />,
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Auto-style scene" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Cancel analysis" }));
  resolve(analysis());
  await new Promise((r) => setTimeout(r, 0));
  expect(apply).not.toHaveBeenCalled();
});

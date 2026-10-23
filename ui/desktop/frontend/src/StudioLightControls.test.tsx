import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { useState } from "react";
import StudioLightControls from "./StudioLightControls";
import type { StudioLight } from "./studio-lights";

it("creates, selects, edits, duplicates and deletes independent light objects", () => {
  const place = vi.fn();
  function Editor() {
    const [lights, setLights] = useState<StudioLight[]>([]);
    const [selected, select] = useState<string>();
    return (
      <StudioLightControls
        lights={lights}
        onChange={setLights}
        selected={selected}
        onSelect={select}
        onPlace={place}
        disabled={false}
      />
    );
  }
  render(<Editor />);
  fireEvent.click(screen.getByRole("button", { name: "Add spot light" }));
  expect(screen.getByLabelText("Light name")).toHaveValue("Spot light");
  fireEvent.change(screen.getByLabelText("Light name"), {
    target: { value: "Fensterlicht" },
  });
  fireEvent.change(screen.getByLabelText("Light position X"), {
    target: { value: "1.2" },
  });
  expect(screen.getByLabelText("Light position X")).toHaveValue(1.2);
  fireEvent.click(
    screen.getByRole("button", { name: "Place in view" }),
  );
  expect(place).toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Duplicate light" }));
  expect(screen.getByLabelText("Light name")).toHaveValue("Fensterlicht Copy");
  fireEvent.click(screen.getByRole("button", { name: "Delete light" }));
  expect(
    screen.getByRole("button", { name: "Fensterlicht" }),
  ).toBeInTheDocument();
});

it("disables editing while exporting", () => {
  render(
    <StudioLightControls
      lights={[]}
      selected={undefined}
      onSelect={() => {}}
      onChange={() => {}}
      onPlace={() => {}}
      disabled
    />,
  );
  expect(
    screen.getByRole("button", { name: "Add point light" }),
  ).toBeDisabled();
});

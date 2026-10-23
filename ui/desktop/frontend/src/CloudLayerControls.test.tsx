import { useState } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import CloudLayerControls from "./CloudLayerControls";
import { atmosphereDefaults } from "./atmosphere-settings";
import { createCloudLayer } from "./cloud-layers";
it("selects a loaded cloud before starting viewport placement", () => {
  const layer = createCloudLayer("volume"),
    onSelect = vi.fn(),
    onPlace = vi.fn();
  render(
    <CloudLayerControls
      value={{ ...atmosphereDefaults, cloudLayers: [layer] }}
      onChange={() => {}}
      disabled={false}
      onPlace={onPlace}
      editor={{
        placing: false,
        mode: "translate",
        onSelect,
        onTransform: () => {},
      }}
    />,
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Place cloud in view" }),
  );
  expect(onSelect).toHaveBeenCalledWith(layer.id);
  expect(onPlace).toHaveBeenCalledOnce();
});
function Editor() {
  const [value, onChange] = useState(atmosphereDefaults);
  return (
    <CloudLayerControls
      value={value}
      onChange={onChange}
      disabled={false}
      animation={{ time: 1, duration: 4, fps: 24, onScrub: () => {} }}
    />
  );
}
it("authors independent named cloud volumes, transforms and copies without losing the base atmosphere", () => {
  render(<Editor />);
  fireEvent.click(
    screen.getByRole("button", { name: "Add cloud volume" }),
  );
  fireEvent.change(screen.getByLabelText("Cloud region name"), {
    target: { value: "Foreground links" },
  });
  fireEvent.change(screen.getByLabelText("Cloud position X"), {
    target: { value: "-.8" },
  });
  expect(screen.getByLabelText("Cloud position X")).toHaveValue(-0.8);
  fireEvent.click(
    screen.getByRole("button", { name: "Duplicate cloud region" }),
  );
  expect(screen.getByLabelText("Cloud region name")).toHaveValue(
    "Foreground links Copy",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Delete cloud region" }),
  );
  expect(
    screen.getByRole("button", { name: "Foreground links" }),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Foreground links" }),
  ).toHaveAttribute("aria-pressed", "true");
});
it("keys density and exposes actual cloud curves, opening and independent render quality", () => {
  render(<Editor />);
  fireEvent.click(
    screen.getByRole("button", { name: "Add cloud layer" }),
  );
  fireEvent.change(screen.getByLabelText("Cloud animation channel"), {
    target: { value: "density" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Add cloud keyframe" }),
  );
  expect(
    screen.getByRole("button", { name: "Curve keyframe 1" }),
  ).toBeInTheDocument();
  expect(screen.getByLabelText("Value curve Cloud density")).toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Opening strength"), {
    target: { value: "1" },
  });
  fireEvent.change(screen.getByLabelText("Cloud preview quality"), {
    target: { value: "low" },
  });
  fireEvent.change(screen.getByLabelText("Cloud export quality"), {
    target: { value: "ultra" },
  });
  expect(screen.getByLabelText("Opening strength")).toHaveValue(1);
  expect(screen.getByLabelText("Cloud preview quality")).toHaveValue("low");
  expect(screen.getByLabelText("Cloud export quality")).toHaveValue("ultra");
});

import { render, screen, fireEvent } from "@testing-library/react";
import { useState } from "react";
import { expect, it } from "vitest";
import AtmosphereControls from "./AtmosphereControls";
import { atmosphereDefaults } from "./atmosphere-settings";
it("applies cloud and fog library presets independently and accepts signed heights", () => {
  function Editor() {
    const [s, set] = useState(atmosphereDefaults);
    return <AtmosphereControls value={s} onChange={set} disabled={false} />;
  }
  render(<Editor />);
  fireEvent.click(
    screen.getByRole("button", { name: "Open cloud library" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: /Small fair-weather clouds/ }),
  );
  expect(screen.getByLabelText("Volumetric clouds")).toBeChecked();
  expect(screen.getByLabelText("Cloud height reference")).toHaveValue("top");
  fireEvent.change(screen.getByLabelText("Cloud height"), {
    target: { value: "-" },
  });
  expect(screen.getByLabelText("Cloud height")).toHaveValue("-");
  fireEvent.change(screen.getByLabelText("Cloud height"), {
    target: { value: "-48" },
  });
  fireEvent.blur(screen.getByLabelText("Cloud height"));
  expect(screen.getByLabelText("Cloud height")).toHaveValue("-48");
  fireEvent.click(
    screen.getByRole("button", { name: "Open fog library" }),
  );
  fireEvent.click(screen.getByRole("button", { name: /Light morning haze/ }));
  expect(screen.getByLabelText("Volumetric fog")).toBeChecked();
  expect(screen.getByLabelText("Cloud height")).toHaveValue("-48");
});

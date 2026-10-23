import { render, screen, fireEvent } from "@testing-library/react";
import { expect, it } from "vitest";
import RenderEditor from "./RenderEditor";
it("opens all scene looks directly from the cinematic toolbar and applies their atmosphere", () => {
  render(<RenderEditor worlds={[]} cinematic />);
  fireEvent.click(
    screen.getByRole("button", { name: "Edit scene looks" }),
  );
  expect(screen.getByRole("tab", { name: "Looks" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Open look library" }),
  );
  fireEvent.click(screen.getByRole("button", { name: /All scenes/ }));
  fireEvent.change(screen.getByLabelText("Browse library"), {
    target: { value: "Peach morning" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: /Cloud island · Peach morning/ }),
  );
  fireEvent.click(screen.getByRole("tab", { name: "Atmosphere" }));
  expect(screen.getByLabelText("Volumetric clouds")).toBeChecked();
});
it.each([false, true])(
  "sets daytime and keeps manual light edits explicit (cinematic=%s)",
  (cinematic) => {
    render(<RenderEditor worlds={[]} cinematic={cinematic} />);
    fireEvent.click(screen.getByRole("tab", { name: "Lighting" }));
    fireEvent.change(screen.getByLabelText("Time of day", { exact: true }), {
      target: { value: "22:00" },
    });
    expect(screen.getByLabelText("Skybox", { exact: true })).toHaveValue(
      "night",
    );
    expect(screen.getByText("Linked")).toBeVisible();
    fireEvent.change(
      screen.getByLabelText("Sun direction Value", { exact: true }),
      { target: { value: "90" } },
    );
    expect(screen.getByText("Manual lighting")).toBeVisible();
  },
);
it("opens the full cinematic atmosphere and color inspector directly", () => {
  render(<RenderEditor worlds={[]} cinematic />);
  fireEvent.click(
    screen.getByRole("button", { name: "Edit clouds and fog" }),
  );
  expect(screen.getByRole("tab", { name: "Atmosphere" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  expect(screen.getByLabelText("Volumetric clouds")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Edit colors" }));
  expect(screen.getByLabelText("Saturation", { exact: true })).toBeVisible();
  fireEvent.click(
    screen.getByRole("button", { name: "Edit camera and sequence" }),
  );
  expect(
    screen.getByRole("complementary", { name: "Render settings" })
      .parentElement,
  ).not.toHaveClass("cinema-light-open");
});
it.each([false, true])(
  "edits cloud type, height and fog in the shared inspector (cinematic=%s)",
  (cinematic) => {
    render(<RenderEditor worlds={[]} cinematic={cinematic} />);
    fireEvent.click(screen.getByRole("tab", { name: "Atmosphere" }));
    fireEvent.click(screen.getByLabelText("Volumetric clouds"));
    fireEvent.change(screen.getByLabelText("Cloud type"), {
      target: { value: "storm" },
    });
    expect(screen.getByLabelText("Cloud type")).toHaveValue("storm");
    fireEvent.change(screen.getByLabelText("Cloud model"), {
      target: { value: "classic" },
    });
    expect(screen.getByLabelText("Cloud model")).toHaveValue("classic");
    fireEvent.change(screen.getByLabelText("Cloud height"), {
      target: { value: "124" },
    });
    expect(screen.getByLabelText("Cloud height")).toHaveValue("124");
    fireEvent.click(screen.getByLabelText("Volumetric fog"));
    fireEvent.change(screen.getByLabelText("Fog color"), {
      target: { value: "#aabbcc" },
    });
    expect(screen.getByLabelText("Fog color")).toHaveValue("#aabbcc");
  },
);
it("routes the shared image inspector from reference looks to editable color controls", () => {
  render(<RenderEditor worlds={[]} />);
  expect(
    screen.getByRole("button", { name: /Vibrant island/ }),
  ).toBeInTheDocument();
  fireEvent.click(screen.getByRole("tab", { name: "Color" }));
  expect(screen.getByLabelText("Saturation", { exact: true })).toHaveValue(
    "1.08",
  );
  fireEvent.change(screen.getByLabelText("Saturation", { exact: true }), {
    target: { value: "1.3" },
  });
  expect(screen.getByLabelText("Saturation Value", { exact: true })).toHaveValue(
    1.3,
  );
  fireEvent.click(screen.getByRole("tab", { name: "Looks" }));
  fireEvent.click(screen.getByRole("button", { name: /Vibrant island/ }));
  fireEvent.click(screen.getByRole("tab", { name: "Color" }));
  expect(screen.getByLabelText("Saturation", { exact: true })).toHaveValue(
    "1.22",
  );
});

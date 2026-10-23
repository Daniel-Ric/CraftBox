import { render, screen, fireEvent, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import LookLibrary from "./LookLibrary";
it("finds a scene by subject and applies the selected look", () => {
  const pick = vi.fn();
  render(
    <LookLibrary selected="cloudGolden" onSelect={pick} disabled={false} />,
  );
  expect(
    screen.getByRole("button", { name: /Cloud island · Golden backlight/ }),
  ).toHaveAttribute("aria-pressed", "true");
  fireEvent.change(screen.getByLabelText("Search looks"), {
    target: { value: "cyberpunk" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: /Metropolis · Neon contrast/ }),
  );
  expect(pick).toHaveBeenCalledWith("cityNeon");
});
it("opens a full categorized library without losing the selected look", () => {
  render(
    <LookLibrary selected="cloudGolden" onSelect={() => {}} disabled={false} />,
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Open look library" }),
  );
  const dialog = screen.getByRole("dialog");
  fireEvent.click(
    within(dialog).getByRole("button", { name: /Winter & mountains/ }),
  );
  expect(
    within(dialog).getByRole("button", { name: /Winter · Crystal clear/ }),
  ).toBeVisible();
  expect(
    within(dialog).queryByRole("button", { name: /Metropolis · Neon contrast/ }),
  ).toBeNull();
});

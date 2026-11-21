import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { localization } from "../src/i18n";
import ReleaseNotes from "../src/ReleaseNotes";

afterEach(async () => {
  cleanup();
  await localization.setLocale("en");
});

it("preserves the published changelog when the interface language changes", async () => {
  const entry = "Add export frame sampling (cinematic)";
  render(
    <details open>
      <summary>What's new</summary>
      <ReleaseNotes notes={`## New features\n\n- ${entry}`} onOpenURL={() => {}} />
    </details>,
  );

  expect(screen.getByRole("listitem").textContent).toBe(entry);
  await act(() => localization.setLocale("de"));

  expect(screen.getByText("Was ist neu")).toBeTruthy();
  expect(screen.getByRole("listitem").textContent).toBe(entry);
});

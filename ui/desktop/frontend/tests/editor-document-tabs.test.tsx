import { afterEach, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { DocumentArea, type AreaController } from "../src/DocumentTabs";
afterEach(cleanup);
it("places compact project tabs in the active editor slot and moves them with the active document", async () => {
  let control!: AreaController;
  HTMLElement.prototype.scrollIntoView = vi.fn();
  render(
    <DocumentArea
      kind="packs"
      active
      controller={(value) => {
        control = value;
      }}
      requestClose={(_entries, done) => done()}
      onSessions={() => {}}
    >
      {(tab) => (
        <section>
          <header>Resource packs</header>
          <main>
            <div data-editor-tab-host data-testid={tab?.id ?? "browser"} />
          </main>
        </section>
      )}
    </DocumentArea>,
  );
  await act(async () => {
    control.open("pack-one", "Pack one", {}, true);
  });
  const tabs = screen.getByRole("tablist", { name: "packs projects" });
  await waitFor(() =>
    expect(tabs.closest("[data-editor-tab-host]")).not.toBeNull(),
  );
  expect(tabs.closest("[hidden]")).toBeNull();
  await act(async () => {
    control.open("pack-two", "Pack two", {}, true);
  });
  await waitFor(() =>
    expect(
      screen
        .getByRole("tab", { name: "Pack two" })
        .getAttribute("aria-selected"),
    ).toBe("true"),
  );
  expect(tabs.closest("[hidden]")).toBeNull();
});

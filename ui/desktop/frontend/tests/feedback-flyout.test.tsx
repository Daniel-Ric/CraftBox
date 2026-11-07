import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { toast } from "sonner";
import FeedbackFlyout from "../src/FeedbackFlyout";
import StudioFlyouts from "../src/StudioFlyouts";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "../src/components/ui/dialog";
import { notificationHistory } from "../src/notification-history";

afterEach(() => {
  cleanup();
  toast.dismiss();
  notificationHistory.clear();
});
it("moves rich errors and recovery controls into the shared flyout, keeping callbacks current", async () => {
  const first = vi.fn(),
    latest = vi.fn();
  const view = render(
    <>
      <main data-testid="view">
        <FeedbackFlyout title="Resource packs">
          Pack failed<button onClick={first}>Retry loading</button>
        </FeedbackFlyout>
      </main>
      <StudioFlyouts />
    </>,
  );
  expect(screen.getByTestId("view").textContent).toBe("");
  const button = await screen.findByRole("button", { name: "Retry loading" });
  fireEvent.click(button);
  expect(first).toHaveBeenCalledTimes(1);
  view.rerender(
    <>
      <main data-testid="view">
        <FeedbackFlyout title="Resource packs">
          Pack failed<button onClick={latest}>Retry loading</button>
        </FeedbackFlyout>
      </main>
      <StudioFlyouts />
    </>,
  );
  fireEvent.click(screen.getByRole("button", { name: "Retry loading" }));
  expect(latest).toHaveBeenCalledTimes(1);
  expect(notificationHistory.snapshot()).toHaveLength(1);
});
it("does not reopen a dismissed error on unrelated re-renders but shows a new error", async () => {
  const make = (text: string) => (
    <>
      <FeedbackFlyout title="Export">{text}</FeedbackFlyout>
      <StudioFlyouts />
    </>
  );
  const view = render(make("Export failed"));
  fireEvent.click(
    await screen.findByRole("button", { name: "Dismiss notification" }),
  );
  await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
  view.rerender(make("Export failed"));
  expect(screen.queryByRole("alert")).toBeNull();
  view.rerender(make("Output folder missing"));
  expect((await screen.findByRole("alert")).textContent).toContain(
    "Output folder missing",
  );
});
it("keeps field descriptions available to assistive technology without an inline warning", async () => {
  render(
    <>
      <FeedbackFlyout id="field-reason" title="Invalid input">
        Enter a positive value
      </FeedbackFlyout>
      <StudioFlyouts />
    </>,
  );
  expect(document.getElementById("field-reason")?.className).toBe("sr-only");
  expect((await screen.findByRole("alert")).textContent).toContain(
    "Enter a positive value",
  );
});
it("keeps flyout recovery controls inside the modal focus boundary", async () => {
  render(
    <>
      <Dialog open>
        <DialogContent>
          <DialogTitle>Export</DialogTitle>
          <FeedbackFlyout title="Export failed">
            Disk full<button>Choose another folder</button>
          </FeedbackFlyout>
        </DialogContent>
      </Dialog>
      <StudioFlyouts />
    </>,
  );
  const button = await screen.findByRole("button", {
    name: "Choose another folder",
  });
  await waitFor(() =>
    expect(screen.getByRole("dialog").contains(button)).toBe(true),
  );
});

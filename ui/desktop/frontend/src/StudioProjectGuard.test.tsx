import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { useStudioProjectGuard } from "./StudioProjectGuard";
function Harness({
  save,
  next,
}: {
  save: () => Promise<boolean>;
  next: () => void;
}) {
  const guard = useStudioProjectGuard();
  guard.session.current = { name: "Testkarte", dirty: true, busy: false, save };
  return (
    <>
      {guard.dialog}
      <button onClick={() => guard.run(next)}>Switch editor</button>
    </>
  );
}
it("keeps the editor after cancelling the guard or a cancelled file save", async () => {
  const next = vi.fn();
  render(<Harness save={async () => false} next={next} />);
  fireEvent.click(screen.getByText("Switch editor"));
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(next).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText("Switch editor"));
  fireEvent.click(screen.getByRole("button", { name: "Save" }));
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "not saved",
  );
  expect(next).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Discard" }));
  expect(next).toHaveBeenCalledOnce();
});
it("waits for a successful save before continuing the requested action", async () => {
  const next = vi.fn();
  let finish!: (value: boolean) => void;
  render(
    <Harness
      save={() =>
        new Promise((resolve) => {
          finish = resolve;
        })
      }
      next={next}
    />,
  );
  fireEvent.click(screen.getByText("Switch editor"));
  fireEvent.click(screen.getByRole("button", { name: "Save" }));
  expect(next).not.toHaveBeenCalled();
  finish(true);
  await waitFor(() => expect(next).toHaveBeenCalledOnce());
});

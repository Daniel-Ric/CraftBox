import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import RenderEditor from "./RenderEditor";
import { renderDefaults } from "./render-settings";
import { mapDefaults } from "./map-presentation";
import type { StudioSession } from "./studio-document";

vi.mock("./RenderCanvas", () => ({
  default: (p: { onReady: () => void }) => (
    <button onClick={p.onReady}>Finish terrain fixture</button>
  ),
}));

it("keeps settings edited during project loading dirty before and after readiness", async () => {
  let session: StudioSession | undefined;
  render(
    <RenderEditor
      worlds={[]}
      loadedProject={{
        world: { id: "saved", name: "Saved", chunks: 1 },
        path: "Saved.craftbox",
        document: {
          format: "craftbox-studio",
          version: 1,
          name: "Saved",
          editor: "render",
          settings: renderDefaults,
          mapView: mapDefaults,
        },
      }}
      onStudioSession={(s) => {
        session = s;
      }}
    />,
  );
  await screen.findByText("Finish terrain fixture");
  expect(session?.dirty).toBe(false);
  fireEvent.click(screen.getByRole("tab", { name: "Color" }));
  fireEvent.change(screen.getByLabelText("Saturation", { exact: true }), {
    target: { value: "1.6" },
  });
  await waitFor(() => expect(session?.dirty).toBe(true));
  fireEvent.click(screen.getByText("Finish terrain fixture"));
  await waitFor(() => expect(session?.dirty).toBe(true));
});

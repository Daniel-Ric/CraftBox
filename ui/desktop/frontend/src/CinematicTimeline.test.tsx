import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { useState } from "react";
import CinematicTimeline from "./CinematicTimeline";
import { createSequence } from "./cinematic";
import { independentSequence } from "./cinematic-tracks";

it("adds a key only to the active channel, edits it and opens its editable curve", () => {
  const changed = vi.fn();
  function Editor() {
    const [sequence, setSequence] = useState(independentSequence(createSequence()));
    return <CinematicTimeline sequence={sequence} time={2} zoom={1} snap disabled={false} onScrub={() => {}} onChange={(next) => { changed(next); setSequence(next); }} onPreview={setSequence} />;
  }
  render(<Editor />);
  fireEvent.change(screen.getByLabelText("Active animation track"), { target: { value: "fov" } });
  fireEvent.click(screen.getByRole("button", { name: "Add track keyframe" }));
  const sequence = changed.mock.calls.at(-1)![0];
  expect(sequence.tracks.find((t: {channel: string}) => t.channel === "fov").keys).toHaveLength(4);
  expect(sequence.tracks.find((t: {channel: string}) => t.channel === "position.x").keys).toHaveLength(3);
  fireEvent.change(screen.getByLabelText("Track value"), { target: { value: "75" } });
  fireEvent.click(screen.getByRole("button", { name: "Curve editor" }));
  expect(screen.getByLabelText("Value curve Field of view")).toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Outgoing Tangent Value"), { target: { value: "10" } });
  expect(changed.mock.calls.at(-1)![0].tracks.find((t: {channel: string}) => t.channel === "fov").keys.find((k: {time: number}) => k.time === 2).outgoing[1]).toBe(10);
});

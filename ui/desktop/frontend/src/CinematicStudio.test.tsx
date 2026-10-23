import {
  render,
  screen,
  fireEvent,
  waitFor,
  act,
} from "@testing-library/react";
import { expect, it, vi } from "vitest";
vi.mock("./RenderCanvas", async () => {
  const React = await import("react");
  return {
    default: React.forwardRef((props: any, ref) => {
      React.useImperativeHandle(ref, () => ({
        snapshot: () => ({
          position: [4, 2, 3],
          target: [1, 1, 1],
          fov: 35,
          roll: 12,
          exposure: 1.4,
        }),
      }));
      return (
        <div
          data-testid="canvas-state"
          data-navigation={props.shot.navigation}
          data-preview={String(props.shot.preview)}
          data-fov={props.shot.pose.fov}
          data-target={JSON.stringify(props.shot.pose.target)}
        >
          3D-Vorschau
        </div>
      );
    }),
  };
});
import CinematicStudio from "./CinematicStudio";
import { renderDefaults } from "./render-settings";
import { createSequence } from "./cinematic";
import * as audioRuntime from "./cinematic-audio-runtime";
import * as audioDraft from "./cinematic-draft";
it("updates cinematic export resolution when shared art direction changes output", () => {
  const props = {
    worldId: "direction-size",
    settings: renderDefaults,
    onReady: () => {},
    onError: () => {},
    onProgress: () => {},
    onSkyLoading: () => {},
  };
  const { rerender } = render(<CinematicStudio {...props} />);
  fireEvent.click(screen.getByRole("tab", { name: "Export" }));
  expect(screen.getByLabelText("MP4 resolution", { exact: true })).toHaveValue(
    "1920",
  );
  rerender(
    <CinematicStudio {...props} settings={{ ...renderDefaults, edge: 3840 }} />,
  );
  expect(screen.getByLabelText("MP4 resolution", { exact: true })).toHaveValue(
    "3840",
  );
});
it("enters free navigation and applies the view to the selected keyframe with undo", () => {
  render(
    <CinematicStudio
      worldId="free-navigation"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.click(screen.getByRole("button", { name: "Free camera" }));
  expect(screen.getByTestId("canvas-state")).toHaveAttribute(
    "data-navigation",
    "fly",
  );
  expect(screen.getByTestId("canvas-state")).toHaveAttribute(
    "data-preview",
    "false",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Update keyframe" }),
  );
  const saved = JSON.parse(
    localStorage.getItem("craftbox-cinematic:free-navigation")!,
  );
  expect(saved.keys[0]).toMatchObject({
    position: [4, 2, 3],
    target: [1, 1, 1],
    fov: 35,
    roll: 12,
    exposure: 1.4,
  });
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  expect(
    JSON.parse(localStorage.getItem("craftbox-cinematic:free-navigation")!)
      .keys[0].position,
  ).toEqual(createSequence().keys[0].position);
});
it("preserves an occupied key when nudging a camera key with the keyboard", () => {
  const s = createSequence();
  s.keys[1].time = 1 / 24;
  localStorage.setItem("craftbox-cinematic:keyboard", JSON.stringify(s));
  render(
    <CinematicStudio
      worldId="keyboard"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.keyDown(
    screen.getByRole("button", { name: "Position X Keyframe 1" }),
    { key: "ArrowRight" },
  );
  expect(screen.getByText("3 Keyframes")).toBeInTheDocument();
});
it("stores independent animated lens keys and restores channel edits with undo", () => {
  render(
    <CinematicStudio
      worldId="independent-ui"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.change(screen.getByLabelText("Playhead position"), {
    target: { value: "2" },
  });
  fireEvent.change(screen.getByLabelText("Active animation track"), {
    target: { value: "fov" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Add track keyframe" }),
  );
  fireEvent.change(screen.getByLabelText("Track value"), {
    target: { value: "75" },
  });
  const saved = () =>
    JSON.parse(localStorage.getItem("craftbox-cinematic:independent-ui")!);
  expect(saved().version).toBe(2);
  expect(
    saved().tracks.find((t: { channel: string }) => t.channel === "fov").keys,
  ).toHaveLength(4);
  expect(
    saved().tracks.find((t: { channel: string }) => t.channel === "position.x")
      .keys,
  ).toHaveLength(3);
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  expect(
    saved()
      .tracks.find((t: { channel: string }) => t.channel === "fov")
      .keys.find((k: { time: number }) => k.time === 2).value,
  ).not.toBe(75);
});
it("preserves independently edited sibling axes when editing a saved pose coordinate", () => {
  render(
    <CinematicStudio
      worldId="axes-ui"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.change(screen.getByLabelText("Track value"), {
    target: { value: "8" },
  });
  fireEvent.change(screen.getByLabelText("position Y"), {
    target: { value: "3" },
  });
  const saved = JSON.parse(localStorage.getItem("craftbox-cinematic:axes-ui")!);
  expect(
    saved.tracks.find((t: { channel: string }) => t.channel === "position.x")
      .keys[0].value,
  ).toBe(8);
});
it("edits sequence duration, scrubs timeline and adds a camera keyframe", () => {
  render(
    <CinematicStudio
      worldId="test"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.change(screen.getByLabelText("Playhead position"), {
    target: { value: "3" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Add keyframe" }));
  expect(screen.getByText("4 Keyframes")).toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Sequence duration"), {
    target: { value: "20" },
  });
  expect(screen.getByLabelText("Sequence duration")).toHaveValue(20);
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  expect(screen.getByLabelText("Sequence duration")).toHaveValue(12);
});
it("edits separate cameras, cuts the program at the shot boundary and undoes the camera assignment", () => {
  render(
    <CinematicStudio
      worldId="shots-ui"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.click(screen.getByRole("tab", { name: "Shots" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Create cameras and shots" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Duplicate camera" }));
  const saved = () =>
    JSON.parse(localStorage.getItem("craftbox-cinematic:shots-ui")!);
  const second = saved().cameras[1].id;
  fireEvent.change(screen.getByLabelText("Active animation track"), {
    target: { value: "fov" },
  });
  fireEvent.change(screen.getByLabelText("Track value"), {
    target: { value: "100" },
  });
  expect(
    saved().cameras[0].tracks.find(
      (t: { channel: string }) => t.channel === "fov",
    ).keys[0].value,
  ).toBe(45);
  fireEvent.change(screen.getByLabelText("Playhead position"), {
    target: { value: "6" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Split shot at playhead" }),
  );
  fireEvent.change(screen.getByLabelText("Shot camera"), {
    target: { value: second },
  });
  fireEvent.change(screen.getByLabelText("Source in"), {
    target: { value: "0" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Show shot sequence" }),
  );
  expect(screen.getByTestId("canvas-state")).toHaveAttribute("data-fov", "100");
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  expect(saved().shots[1].sourceIn).toBe(6);
});
it("authors target movement independently, follows it, edits its curves and restores it with undo", () => {
  render(
    <CinematicStudio
      worldId="tracking-ui"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  fireEvent.click(screen.getByRole("tab", { name: /^Target$/ }));
  fireEvent.click(
    screen.getByRole("button", { name: "Add tracking target" }),
  );
  fireEvent.change(screen.getByLabelText("Target position X"), {
    target: { value: "-1" },
  });
  fireEvent.change(screen.getByLabelText("Playhead position"), {
    target: { value: "12" },
  });
  fireEvent.change(screen.getByLabelText("Target position X"), {
    target: { value: "1" },
  });
  fireEvent.click(screen.getByLabelText("Enable target tracking"));
  fireEvent.change(screen.getByLabelText("Playhead position"), {
    target: { value: "6" },
  });
  expect(
    JSON.parse(screen.getByTestId("canvas-state").dataset.target!)[0],
  ).toBeCloseTo(0);
  fireEvent.click(
    screen.getByRole("button", {
      name: "Edit target tracks in the curve editor",
    }),
  );
  expect(
    screen.getByLabelText("Active animation track").querySelectorAll("option"),
  ).toHaveLength(3);
  fireEvent.change(screen.getByLabelText("Track value"), {
    target: { value: "-2" },
  });
  const saved = () =>
    JSON.parse(localStorage.getItem("craftbox-cinematic:tracking-ui")!);
  expect(
    saved().targets[0].tracks.find(
      (t: { channel: string }) => t.channel === "position.x",
    ).keys[0].value,
  ).toBe(-2);
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  expect(
    saved().targets[0].tracks.find(
      (t: { channel: string }) => t.channel === "position.x",
    ).keys[0].value,
  ).toBe(-1);
});
it("authors named range markers with keyboard navigation and undo", () => {
  const changed = vi.fn();
  const { container } = render(
    <CinematicStudio
      worldId="markers-ui"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
      onCinematicChange={changed}
    />,
  );
  fireEvent.change(screen.getByLabelText("Playhead position"), {
    target: { value: "2" },
  });
  fireEvent.keyDown(container.querySelector(".cinema-studio")!, { key: "m" });
  fireEvent.change(screen.getByLabelText("Marker name"), {
    target: { value: "Musikeinsatz" },
  });
  fireEvent.change(screen.getByLabelText("Marker duration"), {
    target: { value: "1" },
  });
  expect(changed.mock.calls.at(-1)![0].sequence.markers[0]).toMatchObject({
    name: "Musikeinsatz",
    time: 2,
    duration: 1,
  });
  fireEvent.keyDown(
    screen.getByRole("button", { name: "Marker Musikeinsatz" }),
    { key: "ArrowRight" },
  );
  expect(changed.mock.calls.at(-1)![0].sequence.markers[0].time).toBeCloseTo(
    2 + 1 / 24,
  );
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  expect(changed.mock.calls.at(-1)![0].sequence.markers[0].time).toBe(2);
});
it("cleans duplicate import buffers even when the retained material array is unchanged", async () => {
  const asset = {
    id: "first",
    name: "tone.wav",
    mime: "audio/wav",
    data: "AAAA",
    duration: 1,
    sampleRate: 48000,
    channels: 1,
    peaks: [[0, 0.4]],
  };
  const importer = vi
    .spyOn(audioRuntime, "importAudioAsset")
    .mockResolvedValueOnce(asset)
    .mockResolvedValueOnce({ ...asset, id: "duplicate" });
  const cleanup = vi.spyOn(audioRuntime, "releaseAudioCache");
  const draft = vi.spyOn(audioDraft, "saveAudioDraft").mockResolvedValue();
  const changed = vi.fn();
  const view = render(
    <CinematicStudio
      worldId="duplicate-audio"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
      onCinematicChange={changed}
    />,
  );
  try {
    fireEvent.click(screen.getByRole("tab", { name: "Audio" }));
    const input = screen.getByLabelText("Import audio file");
    const upload = () =>
      fireEvent.change(input, {
        target: {
          files: [new File(["tone"], "tone.wav", { type: "audio/wav" })],
        },
      });
    upload();
    await waitFor(() =>
      expect(changed.mock.calls.at(-1)?.[0].sequence.audio?.clips).toHaveLength(
        1,
      ),
    );
    const assets = changed.mock.calls.at(-1)![0].sequence.audio.assets;
    cleanup.mockClear();
    upload();
    await waitFor(() =>
      expect(changed.mock.calls.at(-1)?.[0].sequence.audio?.clips).toHaveLength(
        2,
      ),
    );
    const audio = changed.mock.calls.at(-1)![0].sequence.audio;
    expect(audio.assets).toBe(assets);
    expect(audio.assets.map((a: { id: string }) => a.id)).toEqual(["first"]);
    expect(cleanup).toHaveBeenCalledWith(audio);
  } finally {
    view.unmount();
    importer.mockRestore();
    cleanup.mockRestore();
    draft.mockRestore();
  }
});
it("discards a decoded import that finishes after the editor unmounts", async () => {
  let finish!: (
    asset: Awaited<ReturnType<typeof audioRuntime.importAudioAsset>>,
  ) => void;
  const importer = vi
    .spyOn(audioRuntime, "importAudioAsset")
    .mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
  const discard = vi.spyOn(audioRuntime, "discardAudioAsset");
  const view = render(
    <CinematicStudio
      worldId="abandoned-audio"
      settings={renderDefaults}
      onReady={() => {}}
      onError={() => {}}
      onProgress={() => {}}
      onSkyLoading={() => {}}
    />,
  );
  try {
    fireEvent.click(screen.getByRole("tab", { name: "Audio" }));
    fireEvent.change(screen.getByLabelText("Import audio file"), {
      target: { files: [new File(["tone"], "tone.wav")] },
    });
    view.unmount();
    await act(async () =>
      finish({
        id: "late-buffer",
        name: "tone.wav",
        mime: "audio/wav",
        data: "AAAA",
        duration: 1,
        sampleRate: 48000,
        channels: 1,
        peaks: [[0, 0.4]],
      }),
    );
    expect(discard).toHaveBeenCalledWith("late-buffer");
  } finally {
    importer.mockRestore();
    discard.mockRestore();
  }
});

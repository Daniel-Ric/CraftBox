import {
  render,
  screen,
  waitFor,
  act,
  within,
  fireEvent,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import App from "./App";
import { api } from "./bridge";

const { start, callbacks } = vi.hoisted(() => ({
  start: vi.fn().mockResolvedValue(undefined),
  callbacks: new Map<string, (data: unknown) => void>(),
}));
vi.mock("./bridge", () => ({
  api: {
    GetBootstrap: vi.fn().mockResolvedValue({
      version: "test-dev",
      minecraft: "1.26.50",
      state: {
        running: false,
        worlds: [],
        packs: [],
        skins: [],
        player: [0, 0, 0],
        loggedIn: false,
        account: "",
      },
      commands: [
        {
          name: "worlds",
          options: [
            {
              flag: "address",
              type: "connectInfo",
              name: "Server",
              default: "",
            },
            { flag: "capture", type: "bool", name: "Capture", default: "true" },
            {
              flag: "save-custom-entities",
              type: "bool",
              name: "Custom Entities",
              default: "false",
            },
            {
              flag: "custom-option",
              type: "string",
              name: "Custom option",
              default: "kept",
            },
          ],
        },
        { name: "packs", options: [] },
        { name: "skins", options: [] },
      ],
    }),
    StartCommand: start,
    GetMapTiles: vi.fn().mockResolvedValue([]),
    GetLogs: vi.fn().mockResolvedValue([]),
    OpenMinecraft: vi.fn().mockResolvedValue(undefined),
    OpenMapWindow: vi.fn().mockResolvedValue(undefined),
  },
  on: (name: string, callback: (data: unknown) => void) => {
    callbacks.set(name, callback);
    return () => callbacks.delete(name);
  },
}));
vi.mock("./WorldMap", () => ({
  default: ({ tiles }: { tiles: Map<string, unknown> }) => (
    <div data-testid="map-tiles">{tiles.size}</div>
  ),
}));
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
async function renderMapWindow() {
  const snapshot = await api.GetBootstrap();
  vi.mocked(api.GetBootstrap).mockResolvedValueOnce({
    ...snapshot,
    mapWindow: true,
  });
  render(<App />);
  fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
  await screen.findByRole("region", { name: "World maps" });
}
describe("desktop workflow", () => {
  it("returns to welcome after repeated New Project shortcuts and cancellation", async () => {
    const user = userEvent.setup();
    render(<App />);
    await screen.findByRole("dialog", { name: "Welcome to CraftBox" });
    await user.click(screen.getByRole("button", { name: "Create new" }));
    await screen.findByRole("dialog", { name: "New project" });
    fireEvent.keyDown(window, { key: "n", code: "KeyN", ctrlKey: true });
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      await screen.findByRole("dialog", { name: "Welcome to CraftBox" }),
    ).toBeVisible();
  });
  it("switches to 3D and returns to 2D for selecting a region", async () => {
    await renderMapWindow();
    act(() =>
      callbacks.get("desktop:state")?.({
        revision: 800,
        command: "worlds",
        running: true,
        connection: 5,
        activeWorldId: "world3d",
        worlds: [
          {
            id: "world3d",
            name: "Terrain",
            state: "Capturing",
            player: [0, 70, 0],
            hasPlayer: true,
          },
        ],
        packs: [],
        skins: [],
        player: [0, 70, 0],
        hasPlayer: true,
      }),
    );
    await userEvent.click(screen.getByRole("button", { name: "3D" }));
    expect(
      await screen.findByRole("region", { name: "3D world Terrain" }),
    ).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: "Select region" }),
    );
    expect(screen.getByRole("button", { name: "2D" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });
  it("renames a world on tab double click and edits its own spawn", async () => {
    api.EditWorld = vi.fn().mockResolvedValue(undefined);
    await renderMapWindow();
    act(() =>
      callbacks.get("desktop:state")?.({
        revision: 300,
        command: "worlds",
        running: false,
        worlds: [
          {
            id: "original",
            name: "Lobby",
            state: "finished",
            player: [8, 70, 8],
            hasPlayer: true,
          },
          {
            id: "island",
            parentId: "original",
            name: "Island",
            state: "finished",
            player: [80, 70, 80],
            hasPlayer: true,
          },
        ],
        packs: [],
        skins: [],
        player: [0, 0, 0],
      }),
    );
    await userEvent.dblClick(screen.getByRole("tab", { name: "Island · 2" }));
    const name = await screen.findByLabelText("World name");
    await userEvent.clear(name);
    await userEvent.type(name, "SkyWars");
    await userEvent.click(
      screen.getByRole("button", { name: "Save name" }),
    );
    expect(api.EditWorld).toHaveBeenCalledWith({
      worldId: "island",
      name: "SkyWars",
    });
    await userEvent.click(screen.getByRole("button", { name: "Spawn point" }));
    const editor = screen.getByRole("complementary", {
      name: "Edit spawn point",
    });
    await userEvent.clear(within(editor).getByLabelText("Y"));
    await userEvent.type(within(editor).getByLabelText("Y"), "90");
    await userEvent.click(
      within(editor).getByRole("button", { name: "Save spawn point" }),
    );
    expect(api.EditWorld).toHaveBeenLastCalledWith({
      worldId: "island",
      spawn: [80, 90, 80],
    });
  });
  it("creates a separately captured world from edited bounds", async () => {
    api.CreateWorldRegion = vi.fn().mockResolvedValue({
      id: "region",
      parentId: "source",
      name: "Island",
      bounds: { minX: -1, minZ: 2, maxX: 0, maxZ: 3 },
      move: true,
    });
    await renderMapWindow();
    act(() =>
      callbacks.get("desktop:state")?.({
        revision: 100,
        command: "worlds",
        running: true,
        connection: 5,
        activeWorldId: "source",
        worlds: [{ id: "source", name: "Lobby", state: "Capturing" }],
        packs: [],
        skins: [],
        player: [0, 70, 0],
      }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Select region" }),
    );
    const editor = screen.getByRole("complementary", {
      name: "Edit world region",
    });
    await userEvent.type(within(editor).getByLabelText("World name"), "Island");
    for (const [name, value] of [
      ["X min", "-16"],
      ["X max", "15"],
      ["Z min", "32"],
      ["Z max", "63"],
    ]) {
      const input = within(editor).getByLabelText(name);
      await userEvent.clear(input);
      await userEvent.type(input, value);
    }
    await userEvent.click(
      within(editor).getByRole("button", {
        name: "Create world from selection",
      }),
    );
    await waitFor(() =>
      expect(api.CreateWorldRegion).toHaveBeenCalledWith({
        worldId: "source",
        name: "Island",
        bounds: { minX: -1, minZ: 2, maxX: 0, maxZ: 3 },
        move: true,
      }),
    );
    expect(
      screen.queryByRole("complementary", { name: "Edit world region" }),
    ).not.toBeInTheDocument();
  });
  it("waits for map-window initialization before handling live listener events", async () => {
    const snapshot = await api.GetBootstrap();
    const pending = deferred<typeof snapshot>();
    vi.mocked(api.GetBootstrap).mockReturnValueOnce(pending.promise);
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    act(() =>
      callbacks.get("desktop:state")?.({
        ...snapshot.state,
        command: "worlds",
        running: true,
        connection: 2,
        listenAddr: "127.0.0.1:19132",
      }),
    );
    expect(api.OpenMinecraft).not.toHaveBeenCalled();
    expect(api.OpenMapWindow).not.toHaveBeenCalled();
    await act(async () => pending.resolve({ ...snapshot, mapWindow: true }));
    expect(
      await screen.findByRole("region", { name: "World maps" }),
    ).toBeVisible();
    expect(api.OpenMinecraft).not.toHaveBeenCalled();
    expect(api.OpenMapWindow).not.toHaveBeenCalled();
  });
  it("keeps same-name visits in separate tabs with separate terrain", async () => {
    await renderMapWindow();
    act(() =>
      callbacks.get("desktop:state")?.({
        revision: 100,
        command: "worlds",
        running: true,
        activeWorldId: "second",
        worlds: [
          { id: "first", name: "Lobby", state: "finished" },
          { id: "second", name: "Lobby", state: "Capturing" },
        ],
        packs: [],
        skins: [],
        player: [0, 0, 0],
      }),
    );
    act(() =>
      callbacks.get("desktop:tiles")?.([
        { worldId: "first", x: 0, z: 0, image: "first" },
        { worldId: "second", x: 0, z: 0, image: "second" },
        { worldId: "first", x: 1, z: 0, image: "first" },
      ]),
    );
    const modal = await screen.findByRole("region", { name: "World maps" });
    expect(
      within(modal).getByRole("tab", { name: "Lobby · 2" }),
    ).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("map-tiles")).toHaveTextContent("1");
    await userEvent.click(
      within(modal).getByRole("tab", { name: "Lobby · 1" }),
    );
    expect(
      within(modal).getByRole("tab", { name: "Lobby · 1" }),
    ).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("map-tiles")).toHaveTextContent("2");
  });
  it("launches the deeplink once when ready and shows waiting inside the modal", async () => {
    vi.mocked(api.OpenMinecraft).mockClear();
    vi.mocked(api.OpenMapWindow).mockClear();
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    await screen.findByLabelText("Server address");
    const session = {
      revision: 100,
      command: "worlds",
      running: true,
      connection: 2,
      listenAddr: "0.0.0.0:19132",
      worlds: [],
      packs: [],
      skins: [],
      player: [0, 0, 0],
    };
    act(() => callbacks.get("desktop:state")?.(session));
    await waitFor(() => expect(api.OpenMapWindow).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await waitFor(() => expect(api.OpenMinecraft).toHaveBeenCalledTimes(1));
    act(() => callbacks.get("desktop:state")?.({ ...session, revision: 101 }));
    expect(api.OpenMinecraft).toHaveBeenCalledTimes(1);
    expect(
      screen.queryByRole("button", { name: "Open Minecraft" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("Your world at a glance"),
    ).not.toBeInTheDocument();
  });
  it("reopens the independent map window without starting another capture", async () => {
    vi.mocked(api.OpenMapWindow).mockClear();
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    await screen.findByLabelText("Server address");
    act(() =>
      callbacks.get("desktop:state")?.({
        revision: 100,
        command: "worlds",
        running: true,
        worlds: [],
        packs: [],
        skins: [],
        player: [0, 0, 0],
      }),
    );
    await waitFor(() => expect(api.OpenMapWindow).toHaveBeenCalledTimes(1));
    await userEvent.click(
      screen.getByRole("button", { name: "Open live map" }),
    );
    expect(api.OpenMapWindow).toHaveBeenCalledTimes(2);
    expect(screen.queryByTestId("map-tiles")).not.toBeInTheDocument();
  });
  it("starts with the entered source and preserves every existing command option", async () => {
    const user = userEvent.setup();
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    const address = await screen.findByLabelText("Server address");
    await user.type(address, "mco.cubecraft.net:19132");
    await user.click(screen.getByRole("button", { name: "Start download" }));
    await waitFor(() =>
      expect(start).toHaveBeenCalledWith({
        command: "worlds",
        target: "mco.cubecraft.net:19132",
        values: {
          capture: "true",
          "custom-option": "kept",
          "save-custom-entities": "false",
        },
      }),
    );
  });
  it("does not start a command with an empty source", async () => {
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    const button = await screen.findByRole("button", {
      name: "Start download",
    });
    expect(button).toBeDisabled();
  });
  it("makes custom entity recording opt-in and exposes the render editor", async () => {
    const user = userEvent.setup();
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    const option = await screen.findByRole("switch", {
      name: "Record custom entities",
    });
    expect(option).toHaveAttribute("aria-checked", "false");
    await user.click(option);
    await user.type(
      screen.getByLabelText("Server address"),
      "mco.cubecraft.net:19132",
    );
    await user.click(screen.getByRole("button", { name: "Start download" }));
    await waitFor(() =>
      expect(start).toHaveBeenCalledWith(
        expect.objectContaining({
          values: expect.objectContaining({ "save-custom-entities": "true" }),
        }),
      ),
    );
    await user.click(screen.getByRole("button", { name: "Render editor" }));
    await act(async () => {
      await vi.dynamicImportSettled();
    });
    expect(
      await screen.findByRole("region", { name: "Render editor" }),
    ).toBeVisible();
  });
  it("keeps a live error delivered while the older bootstrap snapshot is loading", async () => {
    const snapshot = await api.GetBootstrap(),
      pending = deferred<typeof snapshot>();
    vi.mocked(api.GetBootstrap).mockReturnValueOnce(pending.promise);
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    act(() =>
      callbacks.get("desktop:state")?.({
        ...snapshot.state,
        error: "Aktueller Verbindungsfehler",
      }),
    );
    await act(async () => pending.resolve(snapshot));
    expect(
      await screen.findByText("Aktueller Verbindungsfehler"),
    ).toBeVisible();
  });
  it("keeps incoming tiles when an older map snapshot resolves", async () => {
    const snapshot = await api.GetBootstrap(),
      pending = deferred<Awaited<ReturnType<typeof api.GetMapTiles>>>();
    vi.mocked(api.GetBootstrap).mockResolvedValueOnce({
      ...snapshot,
      mapWindow: true,
      state: {
        ...snapshot.state,
        command: "worlds",
        running: true,
        connection: 2,
      },
    });
    vi.mocked(api.GetMapTiles).mockReturnValueOnce(pending.promise);
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    await screen.findByTestId("map-tiles");
    act(() =>
      callbacks.get("desktop:tiles")?.([{ x: 128, z: -32, image: "" }]),
    );
    await act(async () => pending.resolve([]));
    expect(screen.getByTestId("map-tiles")).toHaveTextContent("1");
  });
  it("uses the native replay filename without a protocol prefix", async () => {
    const user = userEvent.setup();
    api.PickFile = vi.fn().mockResolvedValue("C:\\captures\\world.pcap2");
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    await screen.findByLabelText("Server address");
    await user.click(
      screen.getByRole("button", { name: "Open capture" }),
    );
    expect(screen.getByLabelText("Server address")).toHaveValue(
      "C:\\captures\\world.pcap2",
    );
  });
  it("ignores a stale source response after changing source tabs", async () => {
    const user = userEvent.setup(),
      old = deferred<Awaited<ReturnType<typeof api.ListSources>>>();
    api.ListSources = vi
      .fn()
      .mockReturnValueOnce(old.promise)
      .mockResolvedValue([
        {
          id: "realm-1",
          name: "Aktuelles Realm",
          description: "Realm",
          address: "realm:Aktuelles Realm",
          available: true,
        },
      ]);
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Worlds" }));
    await screen.findByLabelText("Server address");
    await user.click(screen.getByRole("button", { name: "Featured servers" }));
    await user.click(
      within(screen.getByRole("dialog")).getByRole("tab", {
        name: "Realms",
      }),
    );
    await screen.findByRole("button", { name: /Aktuelles Realm/ });
    await act(async () =>
      old.resolve([
        {
          id: "old",
          name: "Alter Server",
          description: "alt",
          address: "old:19132",
          available: true,
        },
      ]),
    );
    expect(
      screen.getByRole("button", { name: /Aktuelles Realm/ }),
    ).toBeVisible();
    expect(
      screen.queryByRole("button", { name: /Alter Server/ }),
    ).not.toBeInTheDocument();
  });
});

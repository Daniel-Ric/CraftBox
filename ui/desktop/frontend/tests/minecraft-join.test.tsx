import { afterEach, expect, it, vi } from "vitest";
import {
  act,
  cleanup,
  fireEvent,
  render,
  renderHook,
  screen,
} from "@testing-library/react";
import { useMinecraftJoin } from "../src/use-minecraft-join";
import MinecraftJoinStatus from "../src/MinecraftJoinStatus";
import StudioFlyouts from "../src/StudioFlyouts";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
const listening = {
  running: true,
  stopping: false,
  captureID: 12,
  connection: 2,
  minecraftJoinReady: true,
  listenAddr: "http://zerix.ml:19132",
};

it("does not treat opening Minecraft as a successful join and offers help after 20 seconds", async () => {
  vi.useFakeTimers();
  const open = vi.fn().mockResolvedValue(undefined);
  const { result } = renderHook(() => useMinecraftJoin(listening, true, open));
  expect(result.current.phase).toBe("idle");
  await act(async () => {
    await result.current.join();
  });
  expect(result.current.phase).toBe("waiting");
  act(() => vi.advanceTimersByTime(19999));
  expect(result.current.phase).toBe("waiting");
  act(() => vi.advanceTimersByTime(1));
  expect(result.current.phase).toBe("delayed");
  expect(open).toHaveBeenCalledTimes(1);
  await act(async () => {
    await result.current.join();
  });
  expect(result.current.phase).toBe("waiting");
  expect(open).toHaveBeenCalledTimes(2);
});

it("clears the pending hint as soon as Minecraft connects, before server or resource loading completes", async () => {
  vi.useFakeTimers();
  const { result, rerender } = renderHook(
    ({ state }) =>
      useMinecraftJoin(state, true, vi.fn().mockResolvedValue(undefined)),
    { initialProps: { state: listening } },
  );
  await act(async () => {
    await result.current.join();
  });
  rerender({ state: { ...listening, connection: 3 } });
  act(() => vi.advanceTimersByTime(30000));
  expect(result.current.phase).toBe("idle");
  expect(result.current.canJoin).toBe(false);
});

it("cleans up on capture replacement, stop, endpoint change and in map windows", async () => {
  vi.useFakeTimers();
  const open = vi.fn().mockResolvedValue(undefined);
  const { result, rerender } = renderHook(
    ({ state, enabled }) => useMinecraftJoin(state, enabled, open),
    { initialProps: { state: listening, enabled: true } },
  );
  for (const change of [
    { captureID: 13 },
    { stopping: true },
    { running: false },
    { listenAddr: "http://other.ml:19132" },
  ]) {
    rerender({ state: listening, enabled: true });
    await act(async () => {
      await result.current.join();
    });
    rerender({ state: { ...listening, ...change }, enabled: true });
    act(() => vi.advanceTimersByTime(30000));
    expect(result.current.phase).toBe("idle");
  }
  rerender({ state: listening, enabled: false });
  await act(async () => {
    await result.current.join();
  });
  expect(result.current.canJoin).toBe(false);
  expect(open).toHaveBeenCalledTimes(4);
});

it("recovers from dispatch errors and ignores repeated clicks while the launch request is pending", async () => {
  let reject!: (reason: Error) => void;
  const open = vi.fn(
    () =>
      new Promise<void>((_, fail) => {
        reject = fail;
      }),
  );
  const { result } = renderHook(() => useMinecraftJoin(listening, true, open));
  let pending!: Promise<void>;
  act(() => {
    pending = result.current.join();
  });
  await act(async () => {
    await result.current.join();
  });
  expect(open).toHaveBeenCalledTimes(1);
  await act(async () => {
    reject(Error("launch unavailable"));
    await expect(pending).rejects.toThrow("launch unavailable");
  });
  expect(result.current.phase).toBe("idle");
  expect(result.current.launching).toBe(false);
});

it("shows recovery actions in a dismissible flyout outside the page layout", async () => {
  const retry = vi.fn(),
    copy = vi.fn();
  const { container, rerender } = render(
    <>
      <MinecraftJoinStatus
        phase="delayed"
        address={listening.listenAddr}
        launching={false}
        onRetry={retry}
        onCopy={copy}
      />
      <StudioFlyouts />
    </>,
  );
  expect(container.querySelector(".minecraft-join-status")).toBeNull();
  await screen.findByRole("button", { name: "Dismiss notification" });
  expect(screen.getByRole("status").textContent).toContain("main menu");
  expect(screen.getByRole("status").textContent).toContain("proxy stays ready");
  fireEvent.click(screen.getByRole("button", { name: "Retry join" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Copy connection address" }),
  );
  expect(retry).toHaveBeenCalledTimes(1);
  expect(copy).toHaveBeenCalledWith(listening.listenAddr);
  rerender(
    <>
      <MinecraftJoinStatus
        phase="idle"
        address={listening.listenAddr}
        launching={false}
        onRetry={retry}
        onCopy={copy}
      />
      <StudioFlyouts />
    </>,
  );
});

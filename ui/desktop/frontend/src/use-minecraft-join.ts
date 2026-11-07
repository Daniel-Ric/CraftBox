import { useCallback, useEffect, useRef, useState } from "react";
import type { State } from "./bridge";

export type MinecraftJoinPhase = "idle" | "waiting" | "delayed";
type JoinState = Pick<
  State,
  | "running"
  | "stopping"
  | "captureID"
  | "connection"
  | "minecraftJoinReady"
  | "listenAddr"
>;

export function useMinecraftJoin(
  state: JoinState,
  enabled: boolean,
  openMinecraft: () => Promise<unknown>,
) {
  // ServerConnecting is emitted only after clientConnecting for local proxy sessions.
  // The OS accepting a URI launch is deliberately not considered a connection.
  const canJoin =
    enabled &&
    state.running &&
    !state.stopping &&
    state.connection === 2 &&
    !!state.minecraftJoinReady &&
    !!state.listenAddr;
  const key = `${state.captureID}|${state.listenAddr}`;
  const [attempt, setAttempt] = useState<{
    key: string;
    id: number;
    delayed: boolean;
  }>();
  const [launching, setLaunching] = useState(false);
  const sequence = useRef(0);
  const inFlight = useRef<{ key: string; id: number } | undefined>(undefined);

  useEffect(() => {
    if (!canJoin || (attempt && attempt.key !== key)) {
      setAttempt(undefined);
      inFlight.current = undefined;
      setLaunching(false);
    }
  }, [canJoin, key, attempt?.key]);

  useEffect(() => {
    if (!canJoin || !attempt || attempt.key !== key) return;
    const id = attempt.id;
    const timer = window.setTimeout(() => {
      setAttempt((current) =>
        current?.id === id ? { ...current, delayed: true } : current,
      );
    }, 20000);
    return () => window.clearTimeout(timer);
  }, [canJoin, key, attempt?.id]);

  const join = useCallback(async () => {
    if (!canJoin || inFlight.current?.key === key) return;
    const id = ++sequence.current;
    inFlight.current = { key, id };
    setLaunching(true);
    setAttempt({ key, id, delayed: false });
    try {
      await openMinecraft();
    } catch (error) {
      setAttempt((current) => (current?.id === id ? undefined : current));
      throw error;
    } finally {
      if (inFlight.current?.id === id) {
        inFlight.current = undefined;
        setLaunching(false);
      }
    }
  }, [canJoin, key, openMinecraft]);

  const phase: MinecraftJoinPhase =
    canJoin && attempt?.key === key
      ? attempt.delayed
        ? "delayed"
        : "waiting"
      : "idle";
  return { phase, join, canJoin, launching };
}

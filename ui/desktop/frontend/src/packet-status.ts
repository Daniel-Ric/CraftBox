import { api } from "./bridge";
import type { PacketStatus } from "./packet-inspector-model";

type Listener = {
  update: (value: PacketStatus) => void;
  fail: (error: unknown) => void;
};
const listeners = new Set<Listener>();
let cached: PacketStatus | undefined;
let pending: Promise<PacketStatus> | undefined;
let timer: ReturnType<typeof setTimeout> | undefined;
let polling = false;
let revision = 0;

export function cachedPacketStatus() {
  return cached;
}

export function refreshPacketStatus(): Promise<PacketStatus> {
  revision++;
  return readPacketStatus();
}

function readPacketStatus(): Promise<PacketStatus> {
  if (pending) return pending;
  pending = (async () => {
    let value: PacketStatus;
    let requested: number;
    do {
      requested = revision;
      value = await api.GetPacketInspector();
    } while (requested !== revision);
    cached = value;
    for (const listener of listeners) listener.update(value);
    return value;
  })().finally(() => {
    pending = undefined;
  });
  return pending;
}

async function poll() {
  polling = true;
  timer = undefined;
  try {
    await readPacketStatus();
  } catch (error) {
    for (const listener of listeners) listener.fail(error);
  }
  polling = false;
  if (listeners.size) timer = setTimeout(poll, 1000);
}

export function subscribePacketStatus(
  update: Listener["update"],
  fail: Listener["fail"],
) {
  const listener = { update, fail };
  listeners.add(listener);
  if (cached) update(cached);
  if (listeners.size === 1 && timer === undefined && !polling) void poll();
  return () => {
    listeners.delete(listener);
    if (!listeners.size) {
      clearTimeout(timer);
      timer = undefined;
    }
  };
}

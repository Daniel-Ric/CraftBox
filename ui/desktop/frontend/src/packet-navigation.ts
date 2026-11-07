import type { PacketLink } from "./packet-inspector-model";
export type PacketNavigation = {
  search?: string;
  capture?: string;
  packet?: string;
};
export function openPacketEvidence(value: PacketNavigation) {
  sessionStorage.setItem("craftbox:packet-navigation", JSON.stringify(value));
  window.dispatchEvent(new CustomEvent("craftbox:open-packets"));
}
export function openPacketContent(link: PacketLink) {
  sessionStorage.setItem("craftbox:packet-content", JSON.stringify(link));
  window.dispatchEvent(
    new CustomEvent("craftbox:open-content", { detail: link }),
  );
}
export function consumePacketNavigation(): PacketNavigation {
  try {
    const value = JSON.parse(
      sessionStorage.getItem("craftbox:packet-navigation") || "{}",
    );
    sessionStorage.removeItem("craftbox:packet-navigation");
    return value;
  } catch {
    return {};
  }
}

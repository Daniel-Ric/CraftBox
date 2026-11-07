import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render as testingRender, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import StudioFlyouts from "../src/StudioFlyouts";
function render(ui: ReactNode) { return testingRender(<>{ui}<StudioFlyouts /></>); }
import ActionErrorDialog from "../src/ActionErrorDialog";
import { connectionErrorPresentation } from "../src/action-error";

afterEach(() => { cleanup(); delete window.go; });
const dtls="start transports: start DTLS: DTLS Handshake completed and no SRTP Protection Profile was chosen";

it("explains an incompatible hostname redirect and how to disable it", () => {
  const info=connectionErrorPresentation("hostname alias requires a DNS server name, not an IP address or local alias");
  expect(info.summary).toContain("IP address");
  expect(info.summary).toContain("Preferences > Connection");
  expect(info.summary).toContain("Destination hostname");
  expect(info.summary).toContain("play.example.com");
});

it("marks the upstream leg and opens the log belonging to the displayed error", async () => {
  const open=vi.fn(async(id:string)=>{if(id!=="session-error-origin")throw Error("wrong session")});
  window.go={desktop:{App:{OpenSessionLog:open} as unknown as NonNullable<NonNullable<Window["go"]>["desktop"]["App"]>}};
  render(<ActionErrorDialog message={dtls} connection={{leg:"server",target:"https://zerix.ml:19132",localReady:true}} sessionID="session-error-origin" onCopy={vi.fn()} onClose={vi.fn()}/>);
  expect(screen.getByRole("img").getAttribute("aria-label")).toBe("Minecraft to CraftBox: Connected. CraftBox to zerix.ml: Connection failed.");
  expect(screen.getByText(/required security setting/)).not.toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"Open session log"}));
  await waitFor(()=>expect(open).toHaveBeenCalledWith("session-error-origin"));
});

it("marks client-side failure without claiming that the server connection works",()=>{
  const result=connectionErrorPresentation("read packet minecraft: connection reset",{leg:"client",target:"https://zerix.ml:19132",localReady:true});
  expect(result.local).toBe("failed");expect(result.server).toBe("unknown");
});

it("leaves unverified connection legs neutral",()=>{
  const result=connectionErrorPresentation(dtls);
  expect(result.local).toBe("unknown");expect(result.server).toBe("unknown");
  expect(connectionErrorPresentation("receive minecraft: Error: opaque-reference").summary).toBe("The connection ended with an error. Its technical reference is included below.");
});

it("retains the original error if opening its log fails",async()=>{
  window.go={desktop:{App:{OpenSessionLog:async()=>{throw Error("file missing")}} as unknown as NonNullable<NonNullable<Window["go"]>["desktop"]["App"]>}};
  render(<ActionErrorDialog message={dtls} sessionID="session-original" onCopy={vi.fn()} onClose={vi.fn()}/>);
  fireEvent.click(screen.getByRole("button",{name:"Open session log"}));
  expect((await screen.findByRole("alert")).textContent).toContain("file missing");
  expect(screen.getByRole("dialog")).not.toBeNull();
  expect(screen.getByText(dtls)).not.toBeNull();
});

import { expect, it } from "vitest";
import { actionErrorSummary } from "../src/action-error";

it("explains DTLS profile negotiation without exposing transport jargon as the summary", () => {
  const result = actionErrorSummary("start transports: start DTLS: DTLS Handshake completed and no SRTP Protection Profile was chosen");
  expect(result).toContain("security");
  expect(result).not.toContain("SRTP Protection Profile");
});

it("explains a server kick even when its packet had no message", () => {
 const result=actionErrorSummary("read packet minecraft remote: receive minecraft remote: The server removed you from the world. No reason was provided (disconnect code 55)");
 expect(result).toContain("removed you");
 expect(result).toContain("no reason");
 expect(result).not.toContain("read packet");
});

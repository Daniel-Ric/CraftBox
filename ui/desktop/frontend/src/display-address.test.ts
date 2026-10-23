import { expect, it } from "vitest";
import { displayListenAddress } from "./display-address";
it("shows a connectable loopback address for wildcard listeners", () => {
  expect(displayListenAddress("0.0.0.0:19132")).toBe("127.0.0.1:19132");
  expect(displayListenAddress("[::]:43210")).toBe("127.0.0.1:43210");
  expect(displayListenAddress("192.168.1.10:19132")).toBe("192.168.1.10:19132");
});

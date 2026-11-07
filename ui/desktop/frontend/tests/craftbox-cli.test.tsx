import { expect, it, vi } from "vitest";
import { completeCommand, parseCommand, runCLI } from "../src/craftbox-cli";

it("completes nested commands and quoted world identifiers", () => {
  expect(
    completeCommand("/source P", "console", {
      sources: ["Packet Inspector"],
    })[0]?.value,
  ).toBe('/source "Packet Inspector" ');
  expect(
    completeCommand("/lev", "console").some((x) => x.value === "/level "),
  ).toBe(true);
  expect(
    completeCommand("worlds sh", "cli").some((x) => x.value === "worlds show "),
  ).toBe(true);
  expect(
    completeCommand("worlds show H", "cli", {
      worlds: [{ id: "world-hive", name: "Hive Games" }],
    }).some((x) => x.value === 'worlds show "Hive Games" '),
  ).toBe(true);
  const apostrophe = completeCommand("worlds show D", "cli", {
    worlds: [{ id: "a", name: "Daniel's" }],
  }).find((c) => c.label === "Daniel's")!;
  expect(parseCommand(apostrophe.value)).toEqual([
    "worlds",
    "show",
    "Daniel's",
  ]);
});
it("parses quotes without corrupting Windows paths and rejects incomplete quotes", () => {
  expect(parseCommand('worlds show "Hive Games"')).toEqual([
    "worlds",
    "show",
    "Hive Games",
  ]);
  expect(parseCommand('connection inspect "C:\\data\\world"')[2]).toBe(
    "C:\\data\\world",
  );
  expect(parseCommand('worlds show "\\\\server\\share\\"')[2]).toBe(
    "\\\\server\\share\\",
  );
  expect(parseCommand('worlds show "The ""Hive"" world"')[2]).toBe(
    'The "Hive" world',
  );
  expect(() => parseCommand('worlds show "Hive')).toThrow(/quote/i);
});
it("shows world data and refuses ambiguous world selection", async () => {
  const snapshot = vi.fn(async () => ({
    worlds: [
      { id: "a", name: "Hive Games", chunks: 30, entities: 2, state: "saved" },
      { id: "b", name: "Hive Games", chunks: 5, entities: 1, state: "saved" },
    ],
  }));
  const ctx = {
    backend: { Snapshot: snapshot } as any,
    logs: [],
    history: [],
    navigate: vi.fn(),
    exportText: vi.fn(),
  };
  expect((await runCLI("worlds list", ctx)).text).toContain("Hive Games");
  await expect(runCLI('worlds show "Hive Games"', ctx)).rejects.toThrow(
    /ambiguous/i,
  );
  expect((await runCLI("worlds show a", ctx)).text).toContain('"chunks": 30');
});
it("opens the session belonging to the current application and propagates failures", async () => {
  let opened = "";
  const ctx = {
    backend: {
      Snapshot: async () => ({ logSessionID: "session-original" }),
      OpenSessionLog: async (id: string) => {
        opened = id;
      },
    } as any,
    logs: [],
    history: [],
    navigate: vi.fn(),
    exportText: vi.fn(),
  };
  await runCLI("logs open", ctx);
  expect(opened).toBe("session-original");
  ctx.backend.OpenSessionLog = async () => {
    throw Error("editor unavailable");
  };
  await expect(runCLI("logs open", ctx)).rejects.toThrow("editor unavailable");
});
it("filters logs and validates capture flags before starting a session", async () => {
  let started: any;
  const ctx = {
    backend: {
      GetBootstrap: async () => ({
        commands: [
          {
            name: "worlds",
            options: [
              {
                name: "Entities",
                flag: "entities",
                type: "bool",
                default: "false",
              },
            ],
          },
        ],
      }),
      StartCommand: async (r: any) => {
        started = r;
      },
    } as any,
    logs: [
      {
        id: 1,
        time: "now",
        level: "error",
        source: "Proxy",
        message: "failed",
      },
      { id: 2, time: "now", level: "info", source: "Proxy", message: "ready" },
    ],
    history: [],
    navigate: vi.fn(),
    exportText: vi.fn(),
  };
  expect((await runCLI("logs tail 10 --level error", ctx)).text).not.toContain(
    "ready",
  );
  await runCLI('capture start "play.example.net:19132" --entities true', ctx);
  expect(started).toEqual({
    command: "worlds",
    target: "play.example.net:19132",
    values: { entities: "true" },
  });
  await expect(
    runCLI("capture start example.net --unknown yes", ctx),
  ).rejects.toThrow(/unknown/i);
});

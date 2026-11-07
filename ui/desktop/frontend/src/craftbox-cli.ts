import type { Backend, CommandInfo, Log } from "./bridge";
import { formatConsoleLog, normalizeLogLevel } from "./console-model";

export type Completion = { value: string; label: string; description: string };
export type CompletionContext = {
  worlds?: { id: string; name: string }[];
  sources?: string[];
  commands?: CommandInfo[];
};
export const cliCommands = [
  ["help", "[group]", "Command reference and examples"],
  ["status", "[--json]", "Current application and connection status"],
  ["version", "", "CraftBox and Minecraft versions"],
  ["worlds list", "[search] [--json]", "List captured worlds"],
  ["worlds show", "<id|name>", "World details and coordinates"],
  ["worlds export", "<id|name>", "Export a world model as GLB"],
  ["worlds reveal", "<id|name>", "Open the world folder"],
  ["assets list", "[packs|skins] [--json]", "List saved resources"],
  ["assets show", "<id>", "Inspect a resource"],
  ["capture status", "", "Current capture state"],
  ["capture commands", "[name]", "Capture commands, flags and defaults"],
  [
    "capture start",
    "<target> [--command worlds] [--flag value]",
    "Start a capture with validated options",
  ],
  ["capture stop", "", "Stop the current capture and save its results"],
  ["connection inspect", "<address>", "Probe a server connection"],
  [
    "logs tail",
    "[count] [--level error] [--source name] [--find text]",
    "Read and filter recent session logs",
  ],
  ["logs stats", "", "Entry counts by level and source"],
  ["logs sessions", "", "List saved log sessions"],
  ["logs open", "[session-id]", "Open the current or a saved session log"],
  ["logs folder", "", "Open log storage"],
  ["logs export", "", "Export the current log buffer"],
  ["account status", "", "Show the active account"],
  ["diagnostics snapshot", "", "Save a diagnostic snapshot"],
  ["open", "<page>", "Open a CraftBox workspace"],
  ["history", "", "Recent commands"],
  ["clear", "", "Clear the CLI transcript"],
  ["exit", "", "Return to application logs"],
] as const;
const consoleCommands = [
  ["/cli", "[command]", "Open the interactive CraftBox CLI"],
  ["/help", "", "Show command help"],
  ["/find", "text", "Find log entries"],
  ["/exclude", "text", "Hide matching entries"],
  ["/source", "name", "Filter a log source"],
  ["/level", "level", "Filter by severity"],
  ["/purpose", "purpose", "Change console purpose"],
  ["/pause", "", "Pause displayed logs"],
  ["/resume", "", "Resume displayed logs"],
  ["/follow", "on|off", "Follow new entries"],
  ["/wrap", "on|off", "Wrap long entries"],
  ["/clear", "", "Clear this log view"],
  ["/copy", "", "Copy visible logs"],
  ["/export", "", "Export visible logs"],
  ["/bookmark", "", "Bookmark the selected entry"],
  ["/open", "page", "Open a workspace"],
  ["/stop", "", "Stop the capture"],
] as const;
const pages = [
  "worlds",
  "packs",
  "skins",
  "render",
  "cinematic",
  "packets",
  "settings",
];
const quote = (value: string) =>
  /\s|["'\\]/.test(value) ? '"' + value.replaceAll('"', '""') + '"' : value;
export function completeCommand(
  input: string,
  mode: "console" | "cli",
  context: CompletionContext = {},
): Completion[] {
  const text = input.trimStart(),
    prefix = mode === "console" && text.startsWith("/cli ") ? "/cli " : "";
  if (prefix)
    return completeCommand(text.slice(5), "cli", context).map((c) => ({
      ...c,
      value: prefix + c.value,
      label: prefix + c.label,
    }));
  const specs = mode === "cli" ? cliCommands : consoleCommands;
  const candidates: Completion[] = specs.map(([path, usage, description]) => ({
    value: path + " ",
    label: path + (usage ? " " + usage : ""),
    description,
  }));
  const values: Record<string, string[]> =
    mode === "console"
      ? {
          "/level": ["all", "error", "warning", "info", "debug", "trace"],
          "/purpose": ["all", "problems", "network", "assets", "render"],
          "/follow": ["on", "off"],
          "/wrap": ["on", "off"],
          "/open": pages,
          "/source": ["all", ...(context.sources ?? [])],
        }
      : {
          open: pages,
          "assets list": ["packs", "skins"],
          help: [
            "worlds",
            "assets",
            "capture",
            "connection",
            "logs",
            "account",
            "diagnostics",
          ],
          "capture commands": (context.commands ?? []).map((c) => c.name),
        };
  for (const [path, choices] of Object.entries(values)) {
    if (text.toLowerCase().startsWith(path.toLowerCase() + " ")) {
      const fragment = text
        .slice(path.length + 1)
        .replace(/^"/, "")
        .toLowerCase();
      return choices
        .filter((value) => value.toLowerCase().startsWith(fragment))
        .slice(0, 20)
        .map((value) => ({
          value: `${path} ${quote(value)} `,
          label: `${path} ${value}`,
          description: "Argument",
        }));
    }
  }
  for (const [path, options] of Object.entries(values))
    for (const option of options)
      candidates.push({
        value: `${path} ${quote(option)} `,
        label: `${path} ${option}`,
        description: "Argument",
      });
  for (const path of ["worlds show", "worlds export", "worlds reveal"])
    if (mode === "cli" && text.toLowerCase().startsWith(path + " ")) {
      const fragment = text
        .slice(path.length + 1)
        .replace(/^"/, "")
        .toLowerCase();
      return (context.worlds ?? [])
        .filter(
          (w) =>
            w.id.toLowerCase().startsWith(fragment) ||
            w.name.toLowerCase().startsWith(fragment),
        )
        .flatMap((w) => [
          {
            value: `${path} ${quote(w.name)} `,
            label: w.name,
            description: w.id,
          },
          {
            value: `${path} ${quote(w.id)} `,
            label: w.id,
            description: w.name,
          },
        ])
        .slice(0, 20);
    }
  if (
    mode === "cli" &&
    text.startsWith("capture start ") &&
    /\s--[^\s]*$/.test(text)
  ) {
    const start = text.lastIndexOf("--"),
      fragment = text.slice(start + 2).toLowerCase();
    const selected = text.match(/--command\s+([^\s]+)/)?.[1] ?? "worlds";
    return [
      "command",
      ...(
        (context.commands ?? []).find((c) => c.name === selected)?.options ?? []
      )
        .filter((o) => o.type !== "connectInfo")
        .map((o) => o.flag),
    ]
      .filter((f) => f.startsWith(fragment))
      .map((f) => ({
        value: text.slice(0, start) + "--" + f + " ",
        label: "--" + f,
        description: "Capture option",
      }));
  }
  return candidates
    .filter(
      (c) =>
        c.value.toLowerCase().startsWith(text.toLowerCase()) &&
        c.value.trim() !== text.trim(),
    )
    .slice(0, 20);
}

export function parseCommand(input: string): string[] {
  const tokens: string[] = [];
  let current = "",
    quoted = "",
    started = false;
  for (let i = 0; i < input.length; i++) {
    const ch = input[i];
    if (quoted) {
      if (ch === quoted) {
        if (input[i + 1] === quoted) {
          current += ch;
          i++;
        } else {
          quoted = "";
        }
      } else {
        current += ch;
      }
      started = true;
      continue;
    }
    if (ch === '"' || ch === "'") {
      quoted = ch;
      started = true;
    } else if (/\s/.test(ch)) {
      if (started) {
        tokens.push(current);
        current = "";
        started = false;
      }
    } else {
      current += ch;
      started = true;
    }
  }
  if (quoted)
    throw Error(
      "Unclosed quote. Finish the quoted argument before running the command.",
    );
  if (started) tokens.push(current);
  return tokens;
}
type CLIContext = {
  backend: Backend;
  logs: Log[];
  history: string[];
  navigate: (page: string) => void;
  exportText: (name: string, text: string) => Promise<unknown>;
  stop?: () => Promise<void>;
};
export type CLIResult = {
  text: string;
  clear?: boolean;
  exit?: boolean;
  worlds?: { id: string; name: string }[];
  commands?: CommandInfo[];
};
const json = (value: unknown) => JSON.stringify(value, null, 2);
function table(headers: string[], rows: unknown[][]) {
  if (!rows.length) return "No results.";
  const cells = [
    headers,
    ...rows
      .slice(0, 200)
      .map((row) => row.map((v) => String(v ?? "—").replace(/[\r\n\t]/g, " "))),
  ];
  const widths = headers.map((_, i) =>
    Math.min(60, Math.max(...cells.map((r) => String(r[i] ?? "").length))),
  );
  return (
    cells
      .map(
        (row, index) =>
          row
            .map((v, i) => String(v).slice(0, widths[i]).padEnd(widths[i]))
            .join("  ")
            .trimEnd() +
          (index === 0
            ? "\n" + widths.map((w) => "─".repeat(w)).join("  ")
            : ""),
      )
      .join("\n") +
    (rows.length > 200 ? `\nShowing 200 of ${rows.length} results.` : "")
  );
}
function options(tokens: string[]) {
  const args: string[] = [],
    flags: Record<string, string> = {};
  for (let i = 0; i < tokens.length; i++) {
    const t = tokens[i];
    if (!t.startsWith("--")) {
      args.push(t);
      continue;
    }
    const [key, ...rest] = t.slice(2).split("=");
    if (!key || Object.hasOwn(flags, key))
      throw Error(`Invalid or repeated option: ${t}`);
    flags[key] = rest.length
      ? rest.join("=")
      : key === "json"
        ? "true"
        : tokens[i + 1] && !tokens[i + 1].startsWith("--")
          ? tokens[++i]
          : "true";
  }
  return { args, flags };
}
function allow(flags: Record<string, string>, keys: string[]) {
  for (const k of Object.keys(flags))
    if (!keys.includes(k))
      throw Error(`Unknown option --${k}. Use help for supported options.`);
}
function arity(args: string[], min: number, max: number, usage: string) {
  if (args.length < min || args.length > max) throw Error(`Usage: ${usage}`);
}
function reference(group = "") {
  const entries = cliCommands.filter(
    ([path]) => !group || path === group || path.startsWith(group + " "),
  );
  if (!entries.length) throw Error(`Unknown help topic: ${group}`);
  return (
    table(
      ["Command", "Arguments", "Description"],
      entries.map((row) => [...row]),
    ) +
    (!group
      ? '\nExamples:\n  worlds show "Hive Games"\n  capture commands worlds\n  capture start play.example.net:19132 --entities true\n  logs tail 30 --level error\n\nTab completes commands. ↑/↓ recalls history. Ctrl+R searches history. Ctrl+L clears output.'
      : "")
  );
}
export async function runCLI(
  input: string,
  ctx: CLIContext,
): Promise<CLIResult> {
  const tokens = parseCommand(input.replace(/^\/cli(?:\s+|$)/, ""));
  if (!tokens.length) return { text: reference() };
  const { args, flags } = options(tokens),
    root = args.shift()?.toLowerCase() ?? "";
  let sub = args[0]?.toLowerCase();
  const out = (value: unknown) => ({
    text: typeof value === "string" ? value : json(value),
  });
  if (root === "help") {
    allow(flags, []);
    arity(args, 0, 2, "help [group]");
    return out(reference(args.join(" ")));
  }
  if (root === "clear" || root === "exit") {
    allow(flags, []);
    arity(args, 0, 0, root);
    return { text: "", [root]: true };
  }
  if (root === "history") {
    allow(flags, []);
    arity(args, 0, 0, "history");
    return out(
      ctx.history.map((line, i) => `${i + 1}  ${line}`).join("\n") ||
        "No history yet.",
    );
  }
  if (root === "open") {
    allow(flags, []);
    arity(args, 1, 1, "open <page>");
    if (!pages.includes(args[0])) throw Error(`Pages: ${pages.join(", ")}`);
    ctx.navigate(args[0]);
    return out(`Opened ${args[0]}.`);
  }
  if (root === "version") {
    allow(flags, []);
    arity(args, 0, 0, "version");
    const b = await ctx.backend.GetBootstrap();
    return out(`CraftBox ${b.version}\nMinecraft ${b.minecraft}`);
  }
  if (root === "status" || (root === "capture" && sub === "status")) {
    allow(flags, ["json"]);
    if (root === "capture") args.shift();
    arity(args, 0, 0, "status [--json]");
    const s = await ctx.backend.Snapshot();
    return out({
      running: s.running,
      stopping: s.stopping,
      command: s.command,
      connection: s.connection,
      listenAddress: s.listenAddr,
      account: s.account,
      loggedIn: s.loggedIn,
      worlds: s.worlds?.length ?? 0,
      packs: s.packs?.length ?? 0,
      skins: s.skins?.length ?? 0,
      player: s.hasPlayer ? s.player : undefined,
      logSession: s.logSessionID,
    });
  }
  if (
    [
      "worlds",
      "assets",
      "capture",
      "connection",
      "logs",
      "account",
      "diagnostics",
    ].includes(root) &&
    !sub
  )
    return out(reference(root));
  if (root === "worlds") {
    args.shift();
    allow(flags, sub === "list" ? ["json"] : []);
    const s = await ctx.backend.Snapshot(),
      worlds = s.worlds ?? [];
    if (sub === "list") {
      arity(args, 0, 1, "worlds list [search] [--json]");
      const found = worlds.filter(
        (w) =>
          !args[0] ||
          `${w.id} ${w.name}`.toLowerCase().includes(args[0].toLowerCase()),
      );
      return {
        text: flags.json
          ? json(found)
          : table(
              ["ID", "World", "Chunks", "Entities", "State"],
              found.map((w) => [w.id, w.name, w.chunks, w.entities, w.state]),
            ),
        worlds,
      };
    }
    if (!["show", "export", "reveal"].includes(sub ?? ""))
      throw Error("Use help worlds for available commands.");
    arity(args, 1, 1, `worlds ${sub} <id|name>`);
    const byID = worlds.find((w) => w.id === args[0]),
      found = byID
        ? [byID]
        : worlds.filter((w) => w.name.toLowerCase() === args[0].toLowerCase());
    if (!found.length) throw Error(`World not found: ${args[0]}`);
    if (found.length !== 1)
      throw Error("Ambiguous world name. Use its unique ID from worlds list.");
    const w = found[0];
    if (sub === "show") return out(w);
    if (sub === "reveal") {
      await ctx.backend.OpenResultFolder(w.path);
      return out(`Opened ${w.path}`);
    }
    const path = await ctx.backend.ExportScene(w.id);
    return out(path ? `Exported: ${path}` : "Export cancelled.");
  }
  if (root === "assets") {
    args.shift();
    allow(flags, sub === "list" ? ["json"] : []);
    const library = await ctx.backend.GetAssetLibrary();
    if (sub === "list") {
      arity(args, 0, 1, "assets list [packs|skins] [--json]");
      if (args[0] && !["packs", "skins"].includes(args[0]))
        throw Error("Asset kinds: packs, skins");
      const items = library.items.filter((x) => !args[0] || x.kind === args[0]);
      return out(
        flags.json
          ? json(items.map(({ image, ...x }) => x))
          : table(
              ["ID", "Kind", "Name", "Bytes"],
              items.map((x) => [x.id, x.kind, x.name, x.bytes]),
            ),
      );
    }
    if (sub === "show") {
      arity(args, 1, 1, "assets show <id>");
      const item = library.items.find((x) => x.id === args[0]);
      if (!item) throw Error(`Asset not found: ${args[0]}`);
      const { image, ...metadata } = item;
      return out(metadata);
    }
    throw Error("Use help assets for available commands.");
  }
  if (root === "capture") {
    args.shift();
    if (sub === "commands") {
      allow(flags, []);
      arity(args, 0, 1, "capture commands [name]");
      const b = await ctx.backend.GetBootstrap();
      const commands = b.commands.filter((c) => !args[0] || c.name === args[0]);
      if (!commands.length) throw Error(`Unknown capture command: ${args[0]}`);
      return {
        text: commands
          .map(
            (c) =>
              `${c.name} — ${c.description}\n` +
              table(
                ["Flag", "Type", "Default", "Description"],
                c.options.map((o) => [
                  "--" + o.flag,
                  o.type,
                  o.default,
                  o.description,
                ]),
              ),
          )
          .join("\n\n"),
        commands: b.commands,
      };
    }
    if (sub === "stop") {
      allow(flags, []);
      arity(args, 0, 0, "capture stop");
      await (ctx.stop ? ctx.stop() : ctx.backend.StopCommand());
      return out("Capture stopped.");
    }
    if (sub === "start") {
      arity(
        args,
        1,
        1,
        "capture start <target> [--command worlds] [--flag value]",
      );
      const b = await ctx.backend.GetBootstrap(),
        command = flags.command ?? "worlds",
        spec = b.commands.find((c) => c.name === command);
      if (!spec) throw Error(`Unknown capture command: ${command}`);
      const values: Record<string, string> = {};
      for (const [key, value] of Object.entries(flags)) {
        if (key === "command") continue;
        const option = spec.options.find(
          (o) => o.flag === key && o.type !== "connectInfo",
        );
        if (!option)
          throw Error(
            `Unknown option --${key}. Use capture commands ${command}.`,
          );
        if (option.type === "bool" && !["true", "false"].includes(value))
          throw Error(`--${key} expects true or false.`);
        if (option.type === "int" && !/^-?\d+$/.test(value))
          throw Error(`--${key} expects an integer.`);
        values[key] = value;
      }
      await ctx.backend.StartCommand({ command, target: args[0], values });
      return out(
        `Started ${command} capture for ${args[0]}. Use capture status to inspect it.`,
      );
    }
    throw Error("Use help capture for available commands.");
  }
  if (root === "connection" && sub === "inspect") {
    args.shift();
    allow(flags, []);
    arity(args, 1, 1, "connection inspect <address>");
    return out(await ctx.backend.InspectSource(args[0], true));
  }
  if (root === "account" && sub === "status") {
    args.shift();
    allow(flags, []);
    arity(args, 0, 0, "account status");
    const s = await ctx.backend.Snapshot();
    return out({
      account: s.account,
      loggedIn: s.loggedIn,
      authenticating: s.authenticating,
    });
  }
  if (root === "diagnostics" && sub === "snapshot") {
    args.shift();
    allow(flags, []);
    arity(args, 0, 0, "diagnostics snapshot");
    await ctx.backend.CreateDiagnosticSnapshot();
    return out("Diagnostic snapshot saved in session logs.");
  }
  if (root === "logs") {
    args.shift();
    if (sub === "tail") {
      allow(flags, ["level", "source", "find", "json"]);
      arity(
        args,
        0,
        1,
        "logs tail [count] [--level level] [--source name] [--find text]",
      );
      const count = Number(args[0] ?? 50);
      if (!Number.isInteger(count) || count < 1 || count > 500)
        throw Error("Log count must be between 1 and 500.");
      if (
        flags.level &&
        !["all", "error", "warning", "info", "debug", "trace"].includes(
          normalizeLogLevel(flags.level),
        )
      )
        throw Error("Unknown log level.");
      const rows = ctx.logs
        .filter(
          (l) =>
            (!flags.level ||
              flags.level === "all" ||
              normalizeLogLevel(l.level) === normalizeLogLevel(flags.level)) &&
            (!flags.source ||
              l.source.toLowerCase() === flags.source.toLowerCase()) &&
            (!flags.find ||
              `${l.source} ${l.message}`
                .toLowerCase()
                .includes(flags.find.toLowerCase())),
        )
        .slice(-count);
      return out(
        flags.json
          ? json(rows)
          : rows.map(formatConsoleLog).join("\n") || "No matching log entries.",
      );
    }
    allow(flags, []);
    arity(args, 0, sub === "open" ? 1 : 0, `logs ${sub}`);
    if (sub === "stats") {
      const counts = new Map<string, number>();
      for (const l of ctx.logs) {
        const k = `${normalizeLogLevel(l.level)} / ${l.source || "CraftBox"}`;
        counts.set(k, (counts.get(k) ?? 0) + 1);
      }
      return out(
        table(
          ["Level / source", "Entries"],
          [...counts.entries()].sort((a, b) => b[1] - a[1]),
        ),
      );
    }
    if (sub === "open") {
      const id = args[0] ?? (await ctx.backend.Snapshot()).logSessionID;
      if (!id) throw Error("Session logging is unavailable.");
      await ctx.backend.OpenSessionLog(id);
      return out(`Opened session log: ${id}`);
    }
    if (sub === "folder") {
      await ctx.backend.OpenLogFolder();
      return out("Opened log storage.");
    }
    if (sub === "sessions") {
      const s = await ctx.backend.GetLogStorage();
      return out(
        table(
          ["Session", "Started", "Status", "Bytes"],
          s.sessions.map((x) => [
            x.id,
            x.started,
            x.active ? "Active" : x.status,
            x.bytes,
          ]),
        ),
      );
    }
    if (sub === "export") {
      const path = await ctx.exportText(
        "CraftBox session",
        ctx.logs.map(formatConsoleLog).join("\n"),
      );
      return out(path ? `Saved: ${path}` : "Export cancelled.");
    }
  }
  throw Error(
    `Unknown command: ${tokens.join(" ")}. Use help or Tab to discover commands.`,
  );
}

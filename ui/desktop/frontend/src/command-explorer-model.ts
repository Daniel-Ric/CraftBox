import { api } from "./bridge";

export type CommandParameter = {
  name: string;
  type: number;
  kind: string;
  optional: boolean;
  options: number;
  enum?: string;
  dynamic: boolean;
  values: string[];
};
export type CommandDefinition = {
  name: string;
  description: string;
  permission: number;
  flags: number;
  aliases: string[];
  overloads: CommandParameter[][];
  chained: string[];
  time: number;
  event: string;
};
export type CommandSession = {
  id: string;
  name: string;
  capture: string;
  protocol: number;
  started: number;
  ended: number;
  updated: number;
  revision: number;
  commands: number;
  events: number;
  error?: string;
};
export type CommandEvent = {
  id: string;
  kind: string;
  name: string;
  time: number;
  direction: string;
  summary: string;
  packet?: string;
  data: unknown;
};
export type CommandDocument = {
  session: CommandSession;
  commands: CommandDefinition[];
  events: CommandEvent[];
  warnings: string[];
};
export type CommandFilter = "all" | "favourites" | "restricted" | "dynamic";
export const permissionNames = [
  "Everyone",
  "Game directors",
  "Administrators",
  "Host",
  "Owner",
  "Internal",
];

interface CommandAPI {
  GetCommandExplorerSessions(): Promise<CommandSession[]>;
  GetCommandExplorer(id: string): Promise<CommandDocument>;
  DeleteCommandExplorerSessions(ids: string[]): Promise<void>;
  ImportCommandExplorer(): Promise<CommandSession>;
  ExtractCommandExplorer(capture: string): Promise<CommandSession>;
  ExportCommandExplorer(request: {
    session: string;
    format: string;
    ids: string[];
  }): Promise<string>;
}
export const commandExplorer = api as unknown as CommandAPI;
export const cleanCommandText = (value: string) =>
  value.replace(/§[0-9a-z]/gi, "");
export function commandSyntax(command: CommandDefinition, variant: number) {
  return (
    "/" +
    command.name +
    (command.overloads[variant] ?? [])
      .map(
        (p) =>
          " " + (p.optional ? "[" : "<") + p.name + (p.optional ? "]" : ">"),
      )
      .join("")
  );
}

export function filterCommands(
  commands: CommandDefinition[],
  query: string,
  filter: CommandFilter,
  favourites: Set<string>,
): CommandDefinition[] {
  const terms = query
    .trim()
    .toLowerCase()
    .replace(/^\//, "")
    .split(/\s+/)
    .filter(Boolean);
  return commands
    .flatMap((command) => {
      if (
        (filter === "favourites" && !favourites.has(command.name)) ||
        (filter === "restricted" && command.permission === 0) ||
        (filter === "dynamic" &&
          !command.overloads.some((o) => o.some((p) => p.dynamic)))
      )
        return [];
      const name = command.name.toLowerCase(),
        aliases = command.aliases.map((a) => a.toLowerCase());
      const content = [
        name,
        ...aliases,
        cleanCommandText(command.description),
        ...command.overloads.flatMap((o) =>
          o.flatMap((p) => [p.name, p.kind, p.enum ?? "", ...p.values]),
        ),
      ]
        .join(" ")
        .toLowerCase();
      if (!terms.every((term) => content.includes(term))) return [];
      const score = terms.reduce(
        (sum, term) =>
          sum +
          (name === term
            ? 0
            : name.startsWith(term)
              ? 1
              : aliases.includes(term)
                ? 2
                : name.includes(term)
                  ? 3
                  : 5),
        0,
      );
      return [{ command, score }];
    })
    .sort(
      (a, b) =>
        a.score - b.score || a.command.name.localeCompare(b.command.name),
    )
    .map((row) => row.command);
}

export function buildCommand(
  command: CommandDefinition,
  variant: number,
  values: string[],
) {
  const parts = ["/" + command.name],
    errors: string[] = [];
  let gap = false;
  for (const [index, p] of (command.overloads[variant] ?? []).entries()) {
    const value = (values[index] ?? "").trim();
    if (!value) {
      if (!p.optional) errors.push(`${p.name} is required`);
      gap = true;
      continue;
    }
    if (gap) errors.push(`${p.name} follows an empty argument`);
    if (/[\r\n\0]/.test(value)) errors.push(`${p.name} must be a single line`);
    if (
      p.kind === "integer" &&
      (!/^[+-]?\d+$/.test(value) ||
        Number(value) < -2147483648 ||
        Number(value) > 2147483647)
    )
      errors.push(`${p.name} must be a 32-bit integer`);
    if (
      p.kind === "number" &&
      (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(value) ||
        !Number.isFinite(Number(value)))
    )
      errors.push(`${p.name} must be a number`);
    if (p.kind === "position") {
      const coordinates = value.split(/\s+/),
        local = coordinates.some((c) => c.startsWith("^"));
      if (
        coordinates.length !== 3 ||
        !coordinates.every((c) =>
          (local
            ? /^\^(?:[+-]?(?:\d+(?:\.\d*)?|\.\d+))?$/
            : /^(?:~(?:[+-]?(?:\d+(?:\.\d*)?|\.\d+))?|[+-]?(?:\d+(?:\.\d*)?|\.\d+))$/
          ).test(c),
        )
      )
        errors.push(
          `${p.name} needs three absolute, relative or local coordinates`,
        );
    }
    if (p.kind === "json") {
      try {
        JSON.parse(value);
      } catch {
        errors.push(`${p.name} must contain valid JSON`);
      }
    }
    if (
      p.kind === "enum" &&
      !p.dynamic &&
      p.values.length &&
      !p.values.includes(value)
    )
      errors.push(`${p.name} must match an advertised value`);
    const quote = p.kind === "enum" || p.kind === "string";
    parts.push(
      quote && /\s/.test(value) && !/^".*"$/.test(value)
        ? JSON.stringify(value)
        : value,
    );
  }
  return { line: parts.join(" "), errors };
}

export function compareCommands(
  before: CommandDefinition[],
  after: CommandDefinition[],
) {
  const normalize = (c: CommandDefinition) =>
    JSON.stringify({
      name: c.name,
      description: c.description,
      permission: c.permission,
      flags: c.flags,
      aliases: c.aliases,
      overloads: c.overloads,
      chained: c.chained,
    });
  const previous = new Map(before.map((c) => [c.name, c])),
    current = new Map(after.map((c) => [c.name, c]));
  return [...new Set([...previous.keys(), ...current.keys()])]
    .sort()
    .flatMap((name) => {
      const a = previous.get(name),
        b = current.get(name);
      return a && b && normalize(a) === normalize(b)
        ? []
        : [
            {
              name,
              change: !a ? "added" : !b ? "removed" : "changed",
              before: a,
              after: b,
            },
          ];
    });
}

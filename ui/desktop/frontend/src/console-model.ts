import type { Log } from "./bridge";
import {loadWorkspacePreferences} from "./workspace-preferences";

export const consolePurposes = {
  all: "CraftBox",
  problems: "Problems",
  network: "Connection",
  assets: "Resources",
  render: "Render & export",
} as const;
export type ConsolePurpose = keyof typeof consolePurposes;
export type ConsoleTab = {
  id: string;
  kind: "console" | "powershell";
  cwd?: string;
  history: string[];
  name: string;
  purpose: ConsolePurpose;
  query: string;
  exclude: string;
  source: string;
  level: string;
  matchCase: boolean;
  wrap: boolean;
  timestamps: boolean;
  lineNumbers: boolean;
  follow: boolean;
  fold: boolean;
  fontSize: number;
  clearedThrough: number;
  bookmarks: number[];
};
export function createConsoleTab(purpose: ConsolePurpose = "all"): ConsoleTab {
  const prefs=loadWorkspacePreferences();
  return {
    id: crypto.randomUUID(),
    kind: "console",
    history: [],
    name: consolePurposes[purpose],
    purpose,
    query: "",
    exclude: "",
    source: "all",
    level: "all",
    matchCase: false,
    wrap: prefs.consoleWrap,
    timestamps: prefs.consoleTimestamps,
    lineNumbers: prefs.consoleLineNumbers,
    follow: prefs.consoleFollow,
    fold: false,
    fontSize: prefs.consoleFontSize,
    clearedThrough: 0,
    bookmarks: [],
  };
}
export const normalizeLogLevel = (v: string) =>
  v === "fatal" || v === "panic" ? "error" : v === "warn" ? "warning" : v;
export function purposeMatches(log: Log, purpose: ConsolePurpose) {
  const text = `${log.source} ${log.message}`.toLowerCase();
  if (purpose === "problems")
    return ["error", "warning"].includes(normalizeLogLevel(log.level));
  if (purpose === "network")
    return /proxy|connect|network|listen|login|transfer|packet|raknet|nethernet|gophertunnel/.test(
      text,
    );
  if (purpose === "assets")
    return /pack|skin|texture|resource|asset|download/.test(text);
  if (purpose === "render")
    return /render|scene|mesh|atlas|glb|export|cinematic|chunk/.test(text);
  return true;
}
export function selectConsoleLogs(logs: Log[], tab: ConsoleTab) {
  const convert = (v: string) => (tab.matchCase ? v : v.toLowerCase()),
    query = convert(tab.query),
    exclude = convert(tab.exclude);
  return logs.filter(
    (l) =>
      l.id > tab.clearedThrough &&
      purposeMatches(l, tab.purpose) &&
      (tab.source === "all" || tab.source === l.source) &&
      (tab.level === "all" || normalizeLogLevel(l.level) === tab.level) &&
      (!query || convert(`${l.source} ${l.message}`).includes(query)) &&
      (!exclude || !convert(`${l.source} ${l.message}`).includes(exclude)),
  );
}
export function formatConsoleLog(log: Log) {
  return `${log.time} [${log.level.toUpperCase()}]${log.source ? ` [${log.source}]` : ""} ${log.message}`;
}
export function parseConsoleTabs(text: string | null): ConsoleTab[] {
  try {
    const raw = JSON.parse(text || "[]");
    if (!Array.isArray(raw)) throw Error();
    const tabs: ConsoleTab[] = raw
      .filter(
        (v) =>
          v &&
          v.kind === "console" &&
          Object.hasOwn(consolePurposes, v.purpose),
      )
      .slice(0, 16)
      .map((v) => {
        const t = createConsoleTab(v.purpose);
        if (Array.isArray(v.history))
          t.history = v.history
            .filter((x: unknown) => typeof x === "string")
            .slice(-100)
            .map((x: string) => x.slice(0, 4096));
        t.name = typeof v.name === "string" ? v.name.slice(0, 60) : t.name;
        for (const k of ["query", "exclude", "source"] as const)
          if (typeof v[k] === "string") t[k] = v[k].slice(0, 256);
        if (
          ["all", "info", "warning", "error", "debug", "trace"].includes(
            v.level,
          )
        )
          t.level = v.level;
        for (const k of [
          "matchCase",
          "wrap",
          "timestamps",
          "lineNumbers",
          "follow",
          "fold",
        ] as const)
          if (typeof v[k] === "boolean") t[k] = v[k];
        if (Number.isFinite(v.fontSize))
          t.fontSize = Math.max(10, Math.min(20, v.fontSize));
        return t;
      });
    return tabs.length ? tabs : [createConsoleTab()];
  } catch {
    return [createConsoleTab()];
  }
}

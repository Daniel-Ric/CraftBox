import type { CrashReportSession } from "./bridge";
import { text } from "./i18n";

const issuePage = "https://github.com/Daniel-Ric/CraftBox/issues/new";
const issueURLLimit = 7600;
export const crashDescriptionLimit = 400;

function channel(version: string) {
  if (version === "unknown") return text("Unknown");
  if (version.endsWith("-dev")) return text("Development build");
  return version.includes("-preview")
    ? text("Preview build")
    : text("Stable release");
}

const workspaceNames: Record<string, string> = {
  home: "Start screen",
  worlds: "Worlds",
  packs: "Resource packs",
  skins: "Player skins",
  packets: "Packet Inspector",
  marketplace: "Marketplace",
  bedrock: "Archive",
  structures: "Structure Studio",
  render: "Render editor",
  cinematic: "Cinematic editor",
  map: "Map window",
};

function excerpt(value: string | undefined, count: number) {
  return value?.split("\n").slice(0, count).join("\n");
}

export function createCrashReport(
  sessions: CrashReportSession[],
  description: string,
  includeEvents: boolean,
) {
  const title = sessions.every((session) => session.role === "map-window")
    ? text("CraftBox map window closed unexpectedly")
    : sessions[0]?.status === "crashed"
      ? text("CraftBox crashed")
      : text("CraftBox closed unexpectedly");
  const note = description.slice(0, crashDescriptionLimit).trim();
  const hasDetails = sessions.some(
    (session) => session.stack || (includeEvents && session.events),
  );
  const body = (maxLines: number) => {
    const sections = note
      ? [
          `## ${text("What were you doing?")}\n\n${note
            .split(/\r?\n/)
            .map((line) => `    ${line}`)
            .join("\n")}`,
        ]
      : [];
    for (const session of sessions) {
      const heading =
        session.role === "map-window"
          ? text("Map window")
          : text("Main window");
      const result =
        session.status === "crashed"
          ? text("Confirmed crash")
          : text("Unexpected exit");
      const unknown = text("Unknown");
      const details = [
        `## ${heading} · ${result}`,
        `- ${text("CraftBox version")}: ${session.version === "unknown" ? unknown : session.version}`,
        `- ${text("Release channel")}: ${channel(session.version)}`,
        `- ${text("Platform")}: ${session.platform}`,
        `- ${text("Last recorded activity (UTC)")}: ${session.lastActivity === "unknown" ? unknown : session.lastActivity}`,
        `- ${text("Active workspace")}: ${text(workspaceNames[session.workspace ?? ""] ?? "Unknown")}`,
      ];
      const stack = excerpt(session.stack, maxLines);
      const events = includeEvents
        ? excerpt(session.events, maxLines)
        : undefined;
      if (stack)
        details.push(
          `\n**${text("Crash stack excerpt")}**\n\n\`\`\`text\n${stack}\n\`\`\``,
        );
      if (events)
        details.push(
          `\n**${text("Diagnostic events")}**\n\n\`\`\`text\n${events}\n\`\`\``,
        );
      sections.push(details.join("\n"));
    }
    if (hasDetails && maxLines < 32)
      sections.push(text("Use the ZIP export for the full diagnostic report."));
    return sections.join("\n\n");
  };
  const fullBody = body(32);
  const buildURL = (issueBody: string) =>
    `${issuePage}?${new URLSearchParams({ title, body: issueBody })}`;
  let issueBody = fullBody;
  for (const maxLines of [6, 3, 0]) {
    if (buildURL(issueBody).length <= issueURLLimit) break;
    issueBody = body(maxLines);
  }
  return { title, fullBody, issueBody, url: buildURL(issueBody) };
}

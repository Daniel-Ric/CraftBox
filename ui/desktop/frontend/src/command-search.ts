export type CommandAction = {
  label: string;
  group: string;
  enabled: boolean;
  run: () => void;
  shortcut?: string;
  keywords?: string;
};

export function findCommands(
  commands: CommandAction[],
  query: string,
  showUnavailable: boolean,
) {
  const words = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return commands.filter((command) => {
    if (!words.length && !showUnavailable && !command.enabled) return false;
    const text =
      `${command.label} ${command.group} ${command.keywords ?? ""}`.toLocaleLowerCase();
    return words.every((word) => text.includes(word));
  });
}

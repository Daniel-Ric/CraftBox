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
  translate: (label: string) => string = (label) => label,
) {
  const words = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return commands.filter((command) => {
    if (!words.length && !showUnavailable && !command.enabled) return false;
    const terms = [
      command.label,
      command.group,
      command.keywords ?? "",
      translate(command.label),
      translate(command.group),
    ].join(" ").toLocaleLowerCase();
    return words.every((word) => terms.includes(word));
  });
}

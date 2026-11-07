# CraftBox console and CLI

Enter `/cli` in a CraftBox console to open the interactive CLI in that tab.
Enter `help` for the complete reference or `help logs` for one command group.
`exit` or **Application logs** returns to the log view. PowerShell tabs remain
available for operating-system shell commands.

## Keyboard controls

- **Tab** inserts a command or argument suggestion without executing it.
- **Up/Down** selects suggestions, or recalls history when suggestions are closed.
- **Ctrl+R** searches command history. Enter or Tab inserts the selected result.
- **Escape** dismisses suggestions or leaves history search.
- **Ctrl+L** clears output. A running command and its eventual result remain visible.
- **Ctrl+C** clears an unselected input line. Selected text keeps normal copy behavior.
- **Enter** runs the typed command; after selecting a suggestion with arrow keys,
  Enter inserts that suggestion first.

Command history retains up to 100 entries per console tab. The CLI keeps up to
200 transcript entries in memory. Copy and export buttons save the transcript;
closing its tab or leaving the CLI clears that in-memory transcript.

## Examples

```text
status
version
worlds list
worlds show "Hive Games"
worlds export world-id
worlds reveal world-id
assets list packs
assets show asset-id
capture commands worlds
capture start play.example.net:19132 --entities true
capture status
capture stop
connection inspect play.example.net:19132
logs tail 30 --level error
logs tail 100 --source "Packet Inspector" --find disconnect
logs stats
logs sessions
logs open
logs open session-id
logs folder
logs export
account status
diagnostics snapshot
open render
history
clear
exit
```

`capture commands` reads available capture commands, option names, types and
defaults from the running application. `capture start` accepts `--command name`
and the options shown by `capture commands name`; omitted options use backend
defaults. Unknown options and invalid boolean/integer values are rejected.

`worlds list`, `assets list`, `status` and `logs tail` accept `--json`.
World commands accept an exact ID or name. Duplicate names require the unique ID.
Names containing spaces or apostrophes can be quoted. Backslashes in Windows
paths are literal. Inside a quoted argument, double its quote character:

```text
worlds show "Daniel's world"
worlds show "The ""Hive"" world"
```

`logs open` opens the current application's session log. Model and log exports
use the normal desktop save dialogs. Starting/stopping captures and opening
workspaces use the existing CraftBox operations.

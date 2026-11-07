# Console and terminal contract

The console is a local Wails tool window. No terminal operation is exposed by the
map HTTP bridge. Console views share the bounded CraftBox log stream but keep
their own purpose, filters, bookmarks, pause snapshot and clear position. Removing
a console view does not stop a capture. PowerShell tabs own independent ConPTY
processes; removing a shell tab terminates its process tree.

Native methods (frontend types in `bridge.ts`):

- `CreateTerminal({cols, rows, cwd})` → `{id, pid, shell, cwd}`. Windows only,
  PowerShell 7 when installed, otherwise Windows PowerShell. Empty cwd means the
  application working directory. At most eight shells; dimensions are validated.
- `ReadTerminal(id)` → `{data, exited, exitCode}`. `data` is base64 UTF-8/VT bytes,
  at most 64 KiB per call. Callers feed bytes to xterm and wait for its write
  callback before polling again. A 256 KiB producer queue applies backpressure.
- `WriteTerminal(id, data)` writes UTF-8 keyboard/control input, at most 64 KiB.
- `ResizeTerminal(id, cols, rows)` changes the actual pseudoconsole dimensions.
- `CloseTerminal(id)` is idempotent; shutdown closes all shells. A Windows job
  object owns the shell process tree.
- `SaveConsoleLog(name, text)` displays a native save dialog for a UTF-8 `.log`
  file. Empty result means cancellation; maximum text size is 8 MiB.

Shell creation happens only on an explicit user action. Shell commands, output
and working directories are not persisted or automatically replayed on restart.
Log view settings are persisted locally. Shell input is never inferred from log
text. CraftBox remains a single active capture session; multiple console views
do not create multiple proxies or compete for the listening port.

Verification uses Go session-lifecycle/flow-control tests, a real Windows ConPTY
integration test, React interaction tests and browser resize/scroll tests. There
is no HTTP service for a Postman collection; the native interface is verified
directly against these contracts.

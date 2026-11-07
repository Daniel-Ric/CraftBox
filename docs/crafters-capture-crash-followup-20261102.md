# Second CraftersMC process exit — 2026-11-02

The user confirmed that the entire CraftBox window disappeared again with
`1.63.1-capture-crashfix-wails.20261102-dev`. The fresh log shows a successful
join at 07:30:16, a saved 207-chunk world at 07:30:56, then a new world opened
at 07:31:08. The capture continued through 07:34:24, while the ordinary log's
last entry is 07:31:10. No fatal stack, matching Windows Application crash event,
resource-exhaustion event or CraftBox dump was found in the inspected locations.

Evidence is preserved in `tmp/crafters-crash-073424/`. Capture SHA256:
`3D901C2F702934111C84270787BCBCB2A51ECEE68BD359016987FE077CF23D3B`.

Three offline runs completed without a panic:

- Proxy/world capture replay.
- Desktop App.HandleEvent plus repeated 3D scene-page requests with installed
  vanilla assets.
- The same desktop replay with a sink Minecraft client enabling the MapUI and
  its asynchronous texture-color resolver.

They saved the first world and the final 10-chunk world with 24 entities. A
separate resolver probe decoded all seven resource packs using the capture's
content keys and resolved 77 colors for the 90 custom block definitions.
These tests do not reproduce a native WebView host or real live network timing.
The cause of the user's second full-process exit remains unknown.

A separate logging defect was reproduced: with no valid console stdout handle,
`io.MultiWriter(originalStdout, logWriter)` fails on its first destination and
stops the pipe reader. Runtime fatal output can also be lost when the process
exits before that goroutine drains its pipe. A subprocess regression simulated
the missing GUI console and initially produced an empty log after a panic.

Logging now writes the file first, treats console output as optional, and uses
`debug.SetCrashOutput` for direct runtime fatal stacks. Explicit PrintPanic
reports also write through synchronous logrus hooks before exiting. Tests cover
both runtime panic and handled fatal exit with a missing console. Native host
termination or external process killing may still provide no Go stack.

GUI/production tests passed for cmd/CraftBox, utils, ui/desktop and handlers/worlds.
Independent review found no must-fix logging defect. This is a diagnostic fix;
it does not establish that the live process-exit cause has been fixed.

Diagnostic artifact: `builds/resourcepack-fix/Craft-Box.exe`, version
`1.63.1-capture-crashlog-wails.20261102-dev`, SHA256
`BC5789900832F2BD1EF6C7F4F72E070A35A1D131D76A754F28C9A995080DA0A3`.
GUI subsystem 2 verified; console stays hidden. The installed executable was
not replaced because the installation directory is outside the writable root.
The user needs to replace it and reproduce the live failure for fresh evidence.

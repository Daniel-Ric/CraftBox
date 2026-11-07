# Resource-pack join investigation — 2026-11-02

## Confirmed InPvP join and follow-up review (07:03-07:10)

User confirmed a successful live join. Fresh log independently shows chunk 1
first, progress 100/120 while zero is still missing, then chunk 0 as the final
120th response at 07:03:46 (12,284,503 bytes). The cached smaller pack was reused.
At 07:03:51 the proxy reports Connected to server. The world was saved with
422 chunks and 8 entities. Log preserved under
 tmp/resourcepack-join/inpvp-20261102-success-071018/.

A distinct failure at 07:10:17 is a NetherNet ICE failed state. The MapUI writes
to session.Client, so the failed endpoint in that error is the local Minecraft
connection. The user also supplied a read error for the same endpoint. This
proves transport failure after the join, not its network/suspension cause.
The record does not establish whether Minecraft had just been left/minimized;
a clarification was requested. No change suppresses genuine ICE failures.

The saved world was extracted into a workspace copy and audited read-only:
422 chunks, 12 approximate terrain blocks, zero missing textures. All twelve
were torchflower (3) or torchflower_crop (9), incorrectly caught by the broad
name substring torch heuristic despite existing crossed-plant geometry. Actual
torches already return dedicated models before that heuristic. Removing that
false match makes the repeated full terrain audit report zero approximate
blocks and zero missing textures. This is a classification correction; it
does not claim new entity renderers or models for unrelated worlds.

Review also reproduced and corrected these resource-pack edge cases:
- Packet archives are validated before publishing their cache filename.
- A final upload chunk exactly 128 KiB now finishes without requiring an EOF.
- Client pack names lacking a version return errors instead of panicking.
- Waiting for client pack completion responds to session/server cancellation.
- Completed CDN downloads deliver after client negotiation without blocking
  upstream completion, missing a request or closing the client queue twice.
  The negotiation channel publishes the pending queue; delivery is serialized.
Tests cover cached and uncached client negotiation after CDN completion.

The locatorBar export rule was missing despite existing level.dat support;
it is now copied, with a saved-NBT assertion. Seven affected packages pass
(scene, worlds, worldstate, proxy, resourcepacks, connectinfo, gatherings).
Independent code review found no further must-fix issue in the revised pack
handling. The new tests were first observed failing for their respective bugs.
No race-detector run: gcc is unavailable in this shell.

Latest artifact: builds/resourcepack-fix/Craft-Box.exe,
version 1.63.1-pack-review-shapes-wails.20261102-dev.
SHA256: 1731CF6C3A82577B7AAC635DF5736C839948E4838AAE3C5D467D1A2AC0BE1233.
Built gui,production with -H=windowsgui; PE subsystem 2 checked. The installed
D:/bedrock-tools executable is unchanged; manual replacement is required.

## Refill follows receipt rather than contiguous writing (07:02 follow-up)

The latest live join with the reorder build still stalled. The small 51-chunk
pack completed (5,193,655-byte cache ZIP); the large 120-chunk pack's temporary
file stayed at zero bytes while waiting for chunk zero. Preserved log/dump:
tmp/resourcepack-join/inpvp-20261102-065755/. These logs do not show the indices
of all out-of-order arrivals, so the precise live missing-chunk cause is not
proven by this run.

Native data-handler.log lines 294-302 increment the received count and invoke
145a7eba0(receiver,1) even when the sorted queue lacks the next contiguous chunk.
construct.log shows that callback replenishing the request window via
145a7eea3. The proxy incorrectly tied refill to sequential consumption. A test
requiring request 100 after receiving chunk 1 without chunk 0 failed, and a
full two-archive test withholding zero until the last request timed out.

The worker now writes validated unique chunks directly at checked file offsets,
refills once per received chunk, and hashes the whole file in order after every
chunk has arrived. Only index metadata is retained; out-of-order payloads live
on disk. Outstanding requests remain at most 100 per pack. Progress logs now
include received chunk counts and the first missing index. Both new regression
tests and all four affected packages pass. Independent review found no
must-fix issue. The live effectiveness still requires the user's next test.

Latest version: 1.63.1-pack-receive-window-wails.20261102-dev.
Artifact: builds/resourcepack-fix/Craft-Box.exe (gui,production, -H=windowsgui).
SHA256: C196E56D21AEED5C102B4A3FB7D51204E93E4AE1D48C5A69364F2D82B1ACA094.
PE GUI subsystem verified as 2. Installer hash refreshed; target unchanged.

## Live response ordering and console correction (06:51 follow-up)

The user's InPvP screenshots show both initial request windows queued, followed
by an actual ChunkData response with index 1 before index 0. The ordered-only
receiver rejected this response. Thus the window change obtained data but did
not complete the live join. The worker now buffers only requested chunks within
the bounded window, rejects duplicates and invalid offsets/lengths, then writes
and hashes in index order. The native data-handler.log at 0x145a7d1a0 also shows
a queued/sorted receive path. A completion signal unblocks late chunk callbacks;
worker failures are published before that signal closes to preserve the cause.

The concurrent archive regression test first reproduced the exact index-1 error;
ordered and reordered responses now complete two archives across the 100-chunk
window with matching checksums. Invalid chunks and completed-receiver shutdown
are covered. All four affected packages pass. Independent review identified the
completion and failure-ordering issues, both addressed before final build.

Latest artifact: builds/resourcepack-fix/Craft-Box.exe,
version 1.63.1-pack-reorder-wails.20261102-dev,
SHA256 55F4EAC3CB81F321B8DD50EE258067374E610146A0F3273246C43D792710CDD3.
Built with gui,production and -H=windowsgui. PE subsystem verified as 2 (GUI),
which prevents Windows from automatically opening a console. build.go now uses
that flag for Windows GUI builds. The installer check confirms the older
pack-window executable is still installed; no write to D:/bedrock-tools occurred.
Live InPvP/Mob-Maze join success remains pending the user's manual test.

## Live failure after the wire-name correction: request window

The user's subsequent InPvP test at 06:44:59–06:45:09 still stalled with the
correct wire names. Both packs received DataInfo (120 and 51 chunks, each
102,400 bytes/chunk); both downloads remained at zero bytes. The fresh stall
dump reports **no buffered outbound packets and no deferred inbound packets**.
The network reader waits in NetherNet Receive. This contradicts any claim that
the wire-name change alone fixed the reported live join. It does not establish
that the remote application received the requests: an empty local queue is not
a remote acknowledgment.

Ghidra's `construct.log` shows the Windows client initially submitting
`min(100, chunkCount)` calls to the identified ChunkRequest sender at
`0x145a7f040` (call site `0x145aa9115`). The receive/progress path replenishes
requests through call site `0x145a7eea3`. The proxy instead used a stop-and-wait
loop with only one request outstanding per pack.

The proxy now reproduces this bounded per-pack window. The new regression test
first failed with one request per pack instead of the native 100/51 initial
windows, then passed after the change. It also verifies a request for chunk 100
after consuming chunk zero and rejects requests beyond the initial window.
The concurrent archive test now uses incompressible payloads large enough to
cross the window and proves full completion/checksums for two interleaved packs.
The four affected test packages, including the real local NetherNet handshake,
pass. Independent review found no must-fix issue in this pipeline change.

The new build is `builds/resourcepack-fix/Craft-Box.exe`, version
`1.63.1-pack-window-wails.20261102-dev`, SHA-256
`E0F4BA68D68832A4E7BB45912C4896EFA54A931C20D5BF04D19B45215664EAB2`.
It retains `gui,production`. The InPvP log and dump were copied to
`tmp/resourcepack-join/inpvp-20261102-064509/` before the next attempt.
**Live effectiveness of the window correction is still unverified.** In
particular, the test peer's batching behavior is a controlled hypothesis, not
an observation of the InPvP server implementation.

**Build correction:** the initial binary with SHA-256 `BEDF8AAA...` was incorrectly
compiled with only `gui`; Wails requires `production` for this application.
Successful compilation was insufficient startup verification. The replacement
is `builds/resourcepack-fix/Craft-Box.exe`, version
`1.63.1-resourcepack-wire-name-wails.20261102-dev`, SHA-256
`80504AB5CD672EBE8C2825D15D55CB65B189F7C12F36CAD31524B112BC6AF936`.
It was built with `gui,production` after a successful `npm run build` of the
current frontend. Go's selected Wails sources include `app_production.go` and
exclude `app_default_windows.go`. An isolated startup produced a responding
CraftBox main window; the Computer Use helper could not target that test window,
so rendered UI verification remains user-side. The 700 recorded source-file
hashes were unchanged across the rebuild. No Git reset or checkout appears in
the relevant reflog; this is not proof about a particular unspecified earlier
change. The installation script was updated to reject the earlier broken build.
The historical build entries below describe the original investigation.

Status: protocol and failure-path corrections implemented and locally tested;
successful live joins to both reported targets remain unverified.

Reported targets:

- `https://geo.pvp.inpvp.net:19132`
- Mob Maze, catalog experience `f22e6d68-93c7-44b2-9cc5-716b2e6031cf`

## Evidence and its boundaries

The existing `D:/bedrock-tools/resourcepack-stall.txt`, captured on November 1
at 21:11:01 by `1.63.1-canonical-pack-ids-dev`, contains two pack download
goroutines waiting for chunk replies. The Minecraft-side pack-response handler
waits for the remote stack. No transport writer blocked in `Flush` appears in
that snapshot. This establishes a wait for upstream data at that instant; it
does not establish which request bytes reached the server or identify the
server-side reason for withholding a reply.

The installed desktop application successfully displayed NetherNet status for
the reported HTTPS endpoint, version 1.26.60 / protocol 2225. A shell request
from the restricted execution environment failed to connect. These observations
come from different execution contexts and do not establish a server outage.
The desktop join attempt was stopped during Computer Use before Minecraft joined
the proxy. No successful live join is claimed for the new binary.

## Wire-name correction

`ResourcePackDataInfo.UUID` and `ResourcePackChunkRequest.UUID` are historically
named Go fields. Their wire representation is a **Resource Name string**.
The current official [DataInfo schema](https://mojang.github.io/bedrock-protocol-docs/latest/packets/resource-pack-data-info-packet/)
and [ChunkRequest schema](https://mojang.github.io/bedrock-protocol-docs/latest/packets/resource-pack-chunk-request-packet/)
use this name. The website currently defaults to preview releases; this is not
by itself proof of stable-client behavior. The installed stable Windows client
was inspected independently below.

The proxy parsed that name into a UUID for its internal lookup, then sent
`pack.ID.String()` back to the server. This removed a version suffix and
normalized casing. A server indexing its active uploader by the announced
string cannot resolve that rewritten name. For example:

```text
DataInfo:      AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE_1.0.0
old request:   aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
new request:   AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE_1.0.0
```

Chunk requests now echo `pk.UUID` exactly. Internal maps continue to use parsed
UUIDs, and chunk replies with a bare UUID or a version suffix remain accepted.
The vendored Gophertunnel downloader also preserves the DataInfo string.

`TestChunkRequestsEchoAnnouncedResourceName` failed before the correction for
versioned and uppercase/versioned names, while the bare-name case passed. All
three pass after it. The concurrent full-archive test now acts as an exact-key
uploader and verifies successful completion and checksums. The previous test
assertion that required stripping the suffix was corrected.

This proves the rewritten-name defect and its fix against a strict peer. It
does not prove that either reported live target uses an exact-name lookup.

## Ghidra analysis of the installed Windows client

Target:
`C:/Program Files/WindowsApps/Microsoft.MinecraftUWP_1.26.6005.0_x64__8wekyb3d8bbwe/Minecraft.Windows.exe`

- SHA-256: `988f6a61403b4381aa41e6c66d64cce441b14e561a9b10f647c2758ef3f2672a`
- Ghidra 12.1.4, PE x86-64, image base `0x140000000`.
- Imported project: `tmp/resourcepack-join/ghidra/minecraft-1266005.gpr`.
- Whole-program auto-analysis was disabled. Selected functions were disassembled
  and decompiled, with unwind-table ranges used to identify function boundaries.
- Read-only analysis changes were discarded; the game executable was not patched.

The name getter at `0x1443888f0` references the literal
`ResourcePackChunkRequestPacket` at `0x14f53c99c`. Its vtable slot at
`0x14f1dd1f0` identifies the packet vtable at `0x14f1dd1e0`.
Code references to that vtable locate the request sender at `0x145a7f040`.
That sender calls `0x140f63220` with a stored string at receiver offset `0x38`
and a chunk index, then constructs and sends the identified packet.
The helper copies the entire string and stores the index; it performs no UUID
parsing, suffix removal or case normalization.

This is static evidence about the request-construction path. The complete origin
of the stored string has not been traced, and execution of these functions was
not observed. It must not be presented as a live packet capture or as proof of
the server's implementation.

Reusable local analysis sources and outputs are under
`tmp/resourcepack-join/ghidra/`: `PackBootstrap.java`, `PackInventory.java`,
`PackNative.java`, `inventory.log`, `native.log`, and `construct.log`.
The native logs bind conclusions to the target hash and exact addresses.
An initial Java-script compilation failed because Ghidra tried to use protected
profile/cache paths and subsequently hit a Java archive-close access error.
The completed inspection used a compiled Ghidra launchable with caches in the
workspace, reopening the saved project rather than repeating the import.

## Failure handling and cancellation

Separate failing tests reproduced these defects:

1. Cancelling a join left its HTTP resource-pack request running.
2. An HTTP 503 response was published as a cached archive before parsing failed.
3. A malformed HTTP 200 archive could also poison the next join's cache.
4. Asynchronous download errors were logged without terminating the join.
5. A blocked packet handler replaced the original CDN failure with
   `context canceled`.

HTTP requests now use the download context, require HTTP 200, and publish the
cache file only after archive validation. Worker failures cancel the owning
session with the original error before waking handler waiters. Packet handlers
return `context.Cause`, preserving that error. Standalone handlers without an
owner callback close the server connection on download failure. Normal teardown
does not report secondary worker-cancellation errors as a new failure.

The production session callback is covered by
`TestResourcePackWorkerFailurePreservesSessionCauseWhileClientWaits`, using a
real local HTTP server while the Minecraft-side response is outstanding.
Independent review found and prompted the cancellation-order correction; its
follow-up found no additional must-fix issue in this scope.

## Build and remaining acceptance checks

A Windows GUI build is available at
`tmp/resourcepack-join/CraftBox-gui.exe`, version
`1.63.1-resourcepack-wire-name.20261102-dev`.

- Size: 82,849,280 bytes.
- SHA-256: `BEDF8AAA8BB3787990DEA83A5C09D1266C663A7675B0B745551552877C17BD2D`.
- Built from the current dirty worktree, including unrelated pre-existing work.
- Not installed over `D:/bedrock-tools/Craft-Box.exe` and not live-validated.

The resource-pack suite passed including NetherNet login/spawn, URL pack,
cold/warm/replay caches, concurrent archives, wire names and failure paths.
Some earlier broad runs encountered Windows IPv6 UDP bind access failures;
those failed runs are not evidence that the relevant network scenarios passed.

Completion still requires a successful join through the new build to the HTTPS
server and to Mob Maze, with resource download/stack completion and StartGame
observed. If a target still stalls, the next decisive evidence is its exact
DataInfo resource names, outgoing ChunkRequest names/indexes, chunk responses,
and transport state from that same attempt. Dynamic Ghidra breakpoints have
not been exercised.

### Headless live checks and requested installation

A bounded probe of the real proxy path was built at
`tmp/resourcepack-join/join-probe.exe` (source in the adjacent `probe` directory).
It loads the existing saved account without printing credentials, writes only
to an isolated work directory, and distinguishes pack completion, StartGame and
spawn. It does not record a world or a full packet capture.

On November 2 at 05:34–05:35 UTC the direct server probe failed before the pack
phase: Windows denied the socket to `device.auth.xboxlive.com:443`. The Mob Maze
probe failed during service discovery with the same socket-access restriction.
Results are retained as `live-mineville.jsonl` and `live-mob-maze.jsonl` under
`tmp/resourcepack-join`. Neither run reached resource-pack negotiation.

The user subsequently requested replacing `D:/bedrock-tools/Craft-Box.exe` for
manual testing. A backup was successfully written to
`tmp/resourcepack-join/Craft-Box.before-wire-name.exe`. Replacement was denied
both while the old application was running and after it exited. The target
directory is outside this session's writable workspace. The old executable and
backup both had SHA-256
`3646C02C93D86DCBBD4FB282534F0353D8949A2DF9BF2EF948F737AE40CD8B75`;
the tested replacement remains available at the build path above. Installation
has not succeeded and must not be reported as complete.

`tmp/resourcepack-join/Install-ResourcePackFix.ps1` now provides the exact local
installation step requested by the user. It verifies the pinned source hash,
requires CraftBox to be closed, backs up and verifies the old executable, copies
only the named target, and verifies the installed hash. It deletes no files and
does not change permissions. Its `-CheckOnly` run succeeded with zero changed
files and confirmed that the old executable was still installed. Actual
installation requires the user to run it outside this restricted session.

### User moved the test build to the application directory

Subsequent inspection found `D:/bedrock-tools/CraftBox-gui.exe` with the verified
replacement hash `BEDF8AAA8BB3787990DEA83A5C09D1266C663A7675B0B745551552877C17BD2D`.
The corresponding workspace build file is no longer present. `CraftBox.log`
records startup of `1.63.1-resourcepack-wire-name.20261102-dev` at 06:37:43 local
time, but no subsequent join events were present at inspection. The separate
`D:/bedrock-tools/Craft-Box.exe` still has the old hash. Thus the new build is
available under its alternate filename; replacement of the default filename
and successful live joins are separate checks. The installation script now also
accepts this moved, hash-verified source file.

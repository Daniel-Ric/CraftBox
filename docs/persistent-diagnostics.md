# Persistent diagnostics and first-start account guide

Diagnostics are stored locally in `logs` under CraftBox's data directory. If
that directory is not writable, the logger tries the user's CraftBox cache
directory. Preferences → Diagnostics & logs shows the actual path.

Each launch has a unique session directory and metadata with version, role,
process ID, start/end timestamps and a clean-exit marker. Main and separate map
windows have independent sessions. Structured JSONL records include UTC time,
sequence number, elapsed time, severity, component and context fields. Logging
covers application/capture lifecycle, connection failures, Go callers, stdout/
stderr, JavaScript errors and native WebView framework messages. Runtime health
samples every 30 seconds include heap, goroutines and garbage-collection totals.
Manual snapshots and the first packet-handler panic collect bounded Go stacks.

## Storage policy

Defaults are 100 MiB total, 14 days and 20 sessions. Settings support 32–1024 MiB,
1–90 days, 1–100 sessions and info/debug/trace detail. Policies are stored on disk
independently of browser preferences. Trace does not disable storage limits.

Each process is limited to three 4 MiB event segments and a 2 MiB crash report.
An additional 64 KiB is reserved for session metadata. Before a session starts,
retention reserves its entire worst-case footprint and removes the oldest closed
sessions as necessary. If other live instances reserve the entire budget, the
new logger does not create another growing session; log settings remain usable
to adjust the policy. Active OS locks protect live sessions and crash recorders
from deletion. Active sessions can temporarily exceed a requested session-count
limit, but not the reserved byte budget.

Rotation removes the expired part before accepting a replacement. If an editor
or scanner prevents removal, logging pauses at that boundary and retries; it
does not accumulate unbounded parts. The Settings storage warning explains the
failure. Incomplete session metadata does not hide remaining files from accounting
or cleanup. Policy cleanup happens before the new policy is persisted.

## Crash behavior

A hidden local recorder process receives Go fatal output through a pipe. It
announces readiness only after holding its protective lock, stores a bounded,
redacted report, and drains any remaining output after truncation. It survives
the parent's termination long enough to write the report. Normal shutdown waits
for the recorder, flushes events and writes the clean marker. The fatal sink is
not disabled by a panic-time defer. Explicit native-framework fatal messages are
stored synchronously before exiting and mark the session as crashed.

Unexpected exits without a Go/native fatal report remain classified as
interrupted sessions. This detects missing clean shutdowns; it does not invent a
stack for OS kills, power loss or native failures that bypass the application's
logging paths. This work improves diagnosis of the recent CraftersMC exit; it
does not establish that exit's still-unreproduced cause.

The former `CraftBox.log` becomes a small pointer to the current session, after
importing a bounded 128 KiB tail from an existing legacy log. Previous managed
sessions are retained until policy cleanup or explicit deletion.

## Settings and local bridge contract

The log browser lists sessions, statuses and file sizes; previews the last
128 KiB; opens a full log in the text editor or the log folder; captures a
snapshot; and deletes one or all closed sessions after confirmation. Active
sessions cannot be deleted. Managed IDs and filenames are validated and file
access uses `os.Root` confinement. There is no arbitrary-path read/delete API.

Wails methods and matching TypeScript types live in `ui/desktop/diagnostics.go`
and `ui/desktop/frontend/src/bridge.ts`:

| Method | Behavior |
| --- | --- |
| GetLogStorage | Policy, actual/reserved bytes, warnings and sessions/files |
| ReadSavedLog(session, file) | Bounded preview and truncation flag |
| SetLogPolicy(policy) | Validate, prune, atomically persist and apply detail |
| DeleteLogSession / DeleteClosedLogSessions | Refuse live sessions; delete saved logs |
| CleanupSavedLogs | Apply retention immediately |
| CreateDiagnosticSnapshot | Save bounded goroutine/runtime context |
| OpenSavedLog / OpenLogFolder | Open validated managed paths |
| ReportFrontendError | Store bounded JavaScript error context |

Credential fields, nested credential values, Bearer/JWT text and sensitive URL
query parameters (including encoded names) are redacted before persistence.
Raw packet payload diagnostics are omitted from persistent records. Logs are
local; this feature does not upload them.

## First start

Unsigned-in users see an optional Xbox account guide before the welcome screen.
Linking uses the existing Microsoft device sign-in flow. Skipping shows which
features require an account (new online server/Realm/Experience/friends captures,
including live pack and skin capture) and which remain available offline (world
import, rendering/GLB export, local packs and captured-session replays).

Continuing offline remembers completion locally. Cancelling sign-in returns to
the guide without marking it complete. Existing signed-in users are not prompted.
The account button remains available later. Separate map windows skip onboarding.

## Validation

Regression tests cover rotation, failed deletion/retry, retention, active-session
reservations, interrupted/corrupted metadata, bounded crashes, redaction, safe
read/delete paths and policy persistence. Subprocess tests cover runtime panic,
handled fatal exit, clean shutdown and native-framework fatal persistence. UI
tests cover the log browser, deletion confirmation/protection, policy controls,
first-start skip/link/cancellation and existing capture workflows. The relevant
39 UI tests and eight affected Go packages pass, and the production frontend
build passes. The new logging and onboarding paths were independently reviewed.

The finished Windows GUI executable was also run in its recorder mode using
explicit redirected process handles. A report larger than 2 MiB completed with
the readiness handshake, a 2,097,150-byte stored report, truncation marker and
redacted credential. It exited successfully without opening a window.

Artifact: `builds/resourcepack-fix/Craft-Box.exe`, version
`1.63.1-diagnostics-onboarding-wails.20261102-dev`, SHA256
`222201732F77B37BD62C1E58B4B8E03AC966CBCA0E879B33E43F4679ADB88EE9`.
Built with `gui,production` and `-H=windowsgui`; PE subsystem 2 checked. The
installer source hash is updated; D:/bedrock-tools was not modified.

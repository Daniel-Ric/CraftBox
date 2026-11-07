# Last Hive capture interruption, 2026-11-04

## Evidence

The main application session `session-20261104T202242Z-155156-c811e2b20c1d` used `1.66.0-unified-ui-v32-wails.20261104-dev`, compiled with Go 1.26.0 on Windows/amd64. It failed at 20:26:14 UTC (21:26:14 local) with Windows access violation `0xc0000005`. Its fatal report reaches `reflect.Value.Elem`, `encoding/json.interfaceEncoder`, nested map and interface-slice encoding. The last runtime health entry reports about 452 MB heap, 1.11 GB system memory, and 154 goroutines. No normal capture disconnect/finalization appears. The later application session ended normally and did not run another capture.

The installed EXE SHA256 is `662b70642d28cfedeac9d4029cd08187fdbec9266f170d72ec953e64e3ab6ed3`, matching the released v32 artifact. The bad address `0x10ec974fb7` has an unusual high-32-bit corruption pattern. Using the exact binary's `reflect.Value.Elem` symbol and the reported instruction offset gives image base `0x7ff6eb790000`. Restoring the image-address prefix yields `0x7ff6ec974fb7`: exactly the Kind byte at offset 0x17 in the valid runtime uint8 type descriptor at `0x7ff6ec974fa0`. This is strong evidence of pointer corruption, rather than an ordinary JSON validation error.

The same upper-address corruption pattern is documented in [Go issue 77975](https://github.com/golang/go/issues/77975). This supports a Windows runtime problem. It does not conclusively establish the cause of this particular process failure. Do not call this a proven GC bug: upstream investigation explicitly reproduced its failure without GC.

## Checks and mitigation

All 36,485 stored inspector records and 17,467 raw replay records were decoded again successfully. Both replays were also repeated three times under GOGC=10 on the original compiler. These tests did not reproduce a deterministic malformed-packet failure. They do not rule out live concurrency, native memory corruption, or the runtime issue.

Root `go.mod` and `go.work` now require Go 1.26.10. CI already reads `go.work` through `actions/setup-go`. The new Windows release must be built with that actual compiler and checked using `go version -m`; changing version directives alone is insufficient. The existing environment pins GOROOT to 1.26.0, so local release commands explicitly set GOROOT to the installed 1.26.10 directory as well as invoking its executable.

This is an evidence-backed runtime update and mitigation, pending a successful live capture. No speculative changes were made to packet parsing or the user's saved world. The unfinished world folder and original recordings are left untouched. Tests read the source captures only; temporary tests with absolute personal paths are removed before release, with source hashes and logs retained.

Verification logs and address analysis: `tmp/hostname-prototype/capture-crash-*`, `capture-runtime-*`. The runtime-policy check fails under Go 1.26.0 before the update; patched tests and executable metadata are recorded separately.
## Verified v33 artifact

`builds/unified-ui/Craft-Box.exe` was built successfully with `gui,production` and `-H=windowsgui`. `go version -m` confirms the actual compiler is `go1.26.10`. SHA256: `b99ab8d624557fd13cb71d807b55cfd69ec0b27ab196956e67afb884c19d0dc6`; size 254,261,248 bytes; PE subsystem 2. All 63 existing frontend files match their embedded bytes. Patched-runtime replay/policy checks and targeted GUI vet passed. The temporary replay/policy tests were removed after verification, with hashes retained. Live capture confirmation is still required to establish whether this mitigation resolves the sporadic failure.

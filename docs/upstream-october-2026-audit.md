# Upstream October 2026 integration

Compared the official `bedrock-tool/bedrocktool` master at
`974f87f34780e6a7be0d362ed22c802713416050` against the current CraftBox sources.
Commit dates below are the upstream committer dates, rather than a cached web
history page. Fork changes were integrated into the existing working tree.

| Date | Commit | Result |
| --- | --- | --- |
| October 17 | [`595d7bd`](https://github.com/bedrock-tool/bedrocktool/commit/595d7bd33c41d08bbe8b0d76124560797739cba9) | Dependency updates already present or superseded. Keep the corrected inclusive dimension range `minimum + height - 1`, including negative minima. |
| October 21 | [`8cffe40`](https://github.com/bedrock-tool/bedrocktool/commit/8cffe40c0cbb7cc589459e9228a6b0fd7bd14206) | UUID/version splitting already supported for pack download and upload. The internal `size` field rename does not change the wire protocol; existing size checks remain. |
| October 29 | [`0b87051`](https://github.com/bedrock-tool/bedrocktool/commit/0b8705150f95f421fca796c21160f499c9d49305) | MC 1.26.60 and NetherNet dependency changes already present. Add typed source kinds, Xbox aliases, current catalog search/client integration, an Experiences source list, meaningful session names and safe output filenames. Preserve legacy source values and the authenticated JSONRPC allocation extension. |
| October 29 | [`a695280`](https://github.com/bedrock-tool/bedrocktool/commit/a6952802951911deddedcc483afa0ee4338bfcad) | Trim formatted/sanitized filenames. |
| October 29 | [`8b4b10f`](https://github.com/bedrock-tool/bedrocktool/commit/8b4b10fa8ce2c0527058eeb1a2dffd9f9a8055c9) | Select Realm signaling by advertised protocol; fix nil/ID Realm lookup, retain protocol during source inspection, resolve session metadata before handler initialization, and propagate initialization errors. Keep the Realm-address CLI with protocol-aware output. |
| October 29 | [`974f87f`](https://github.com/bedrock-tool/bedrocktool/commit/974f87f34780e6a7be0d362ed22c802713416050) | Equivalent HTTP signaling listener already implemented by `DirectNetherNet`. Preserve its persistent identity, LAN discovery, real bound address, cleanup and transfer behavior. |

## Compatibility choices

- Bare addresses retain port 19132 and HTTPS/HTTP negotiation. Explicit URLs
  retain their scheme and use the standard HTTP(S) default port. IPv6 and invalid
  ports are parsed consistently. `xbox:` and existing `nethernet:` values select
  friend sessions; raw JSONRPC allocations remain distinct from Xbox handles.
- Featured sources show records with public endpoints; the Experiences tab
  shows only records that require allocation. A valid endpoint takes priority
  even when its record also includes an experience ID. The complete catalog is
  retained for resolving saved experience names. Browsing performs discovery only; session
  allocation still requires an explicit resolve/connect action. Existing
  `gathering:` values and their compatibility API remain available.
- The native Gophertunnel catalog client supplies the current search body,
  headers and renewable token source. The compatibility join adapter retains the
  `netherNetId` response field needed by captured JSONRPC experience responses.
- The proxy retains its newer transfer forwarding, cancellation cleanup, resource
  pack ordering, blob cache and rendering hooks. A failed listener cancels the
  waiting upstream connection rather than leaving its goroutine blocked.

## Dependency comparison

The October 29 pinned submodules are Dragonfly `cf904cb604fd20c46502fa54a2adb7a573d5d146`
and Gophertunnel `c3beb7b9f3bd27121805e8d8f8c3cddf465c151b`. Token-level comparison
against changes since their October 17 pins found 40 Dragonfly files and 33
Gophertunnel files equivalent before this integration; binary palettes and packet
changes are included. Remaining differences were reviewed as fork adaptations:
resource-pack pointer interfaces, registry/biome APIs, capture decoding,
NetherNet-only listener defaults, login metadata and forwarding hooks.

Retain CraftBox's persistent-palette decoder, which handles namespacing and bounded
NetworkLittleEndian palettes, and its corrected chunk light/height behavior.
Also add upstream's missing skin/cape dimension tolerance, bounded to prevent
large inferred allocations and retaining errors for inconsistent non-square data.

## Verification

Regression tests cover Xbox/source parsing, cached/uncached Realm lookup, Realm
protocol selection and inspection routes, handler initialization failures,
formatted filenames, the catalog HTTP request/response contract and skin/cape
repair. Existing real loopback tests cover NetherNet handshakes, resource packs,
LAN discovery, transfers, disconnection and listener cleanup. These checks do not
claim a live authenticated Realm or external game-server login.

Verified on October 30:

- Application suites: `.`, `./cmd/...`, `./handlers/...`, `./internal/...`,
  `./locale/...`, `./subcommands`, `./subcommands/dump-actors`, `./tools/...`,
  `./ui/...` and `./utils/...` passed with `-count=1`.
- Gophertunnel protocol, Dragonfly chunk and MCDB suites passed.
- SourcePicker/App frontend tests: 21 passed; TypeScript and production frontend
  build passed. Focused read-only review found no remaining blockers.

The initial blanket `go test ./...` includes historical scratch programs in
`tmp` and a binary file named `subcommands/resourcepack-d/resourcepack-d.go`;
these prevent blanket package discovery and were left untouched. One initial
loopback test also encountered a Windows UDP/IPv6 bind rejection; the final
application-suite run, including all resource-pack handshake cases, passed.

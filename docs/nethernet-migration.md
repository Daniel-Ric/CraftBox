# NetherNet migration

The application module and release updater use `github.com/Daniel-Ric/CraftBox`.
No Git submodules are required. `third_party/upstream.json` records the official
Gophertunnel and Dragonfly snapshots and the go-nethernet release. The snapshots
include the original licenses plus the compatibility hooks needed for capture,
world export and the resource-pack proxy. Xbox signaling is maintained locally in
`utils/nethernet`; its MIT license is retained.

## Updating the official snapshots

`third_party/dragonfly` and `third_party/gophertunnel` are tracked source copies,
not submodules. They contain local compatibility changes. `go get -u` does not
update these directories: `go.mod` replacements and `go.work` select the local
copies. There is currently no automatic snapshot updater.

For each update:

1. Read the repository URL and current base commit in `third_party/upstream.json`.
2. Clone the official repository into a separate temporary directory and fetch
   the intended new tag or commit. Keep the current base commit available.
3. Compare three versions: the recorded upstream base, the local `third_party`
   copy, and the new upstream version. Merge the upstream changes into the local
   copy, preserving or adapting the compatibility hooks and regression tests.
   Do not replace the entire directory with a fresh download.
4. Update `third_party/upstream.json` to the new upstream base after the merge.
   If versions or requirements changed, update the module files and workspace
   dependency sums as well. Preserve upstream licenses.
5. Run the affected library tests and the application's login, LAN, transfer,
   resource-pack, capture and world-export tests, then build CLI and GUI. Also
   verify a standalone build with `GOWORK=off` and `-mod=readonly`.
6. Commit the merged source and metadata together, then push the update.

For a direct Go module dependency such as go-nethernet, `go get` can select the
new version; it still needs API review and the same relevant integration tests.
Changing the JSON file alone does not update any source code. Updates to the
two libraries should be coordinated when Dragonfly depends on changed
Gophertunnel APIs.

## Connections

- Dedicated servers: NetherNet HTTP/HTTPS signaling. Bare `host:port` addresses
  probe HTTPS first, then HTTP. Explicit schemes are honored.
- Local proxy: HTTP signaling on the configured TCP port and WebRTC on the same
  UDP port, including loopback candidates for Minecraft on the same computer.
  HTTP and LAN share a persistent P-384 identity at `nethernet/listener-key.pem`
  under the application cache directory. Existing keys are never replaced;
  concurrent first starts publish one complete key atomically.
- LAN: official NetherNet discovery on UDP 7551, with protocol/version, world
  status and nonce; joins use WebRTC. If another application occupies UDP 7551,
  a warning is logged and direct connections remain available.
- Xbox sessions: existing Xbox signaling adapters use official go-nethernet.

The former RakNet transport is unavailable. NetherNet therefore requires a
NetherNet-capable client and upstream server. This development branch targets
Minecraft protocol 2193 (1.26.50 and compatible patch versions).

## Validation and limitations

Tests cover Minecraft login/spawn through HTTP and LAN, loopback/fixed UDP port,
HTTPS selection, large WebRTC payloads, multi-chunk resource packs through a proxy
with cold/warm/replay caches, archive checksums, capture decoding, chunk palettes,
item registries, entity storage, client disconnect cleanup and delivery of server
Transfer packets over both HTTP and LAN connections.

A live CubeCraft test confirmed join and export of 275 chunks and 208 entities.
Encrypted resource packs are forwarded to Minecraft and retained in captures, but
cannot be included as readable texture packs in exported worlds by the public
build. The private `packs` extension source is encrypted with git-crypt and is
not verifiable until unlocked; its file is preserved unchanged. The original
public fork also returned `CanRead() == false` for encrypted packs, and the
configured remote has no `REPO_KEY` secret: this limitation predates migration.

Legacy leading-zero JSON numbers are normalized without rewriting strings or
comments; malformed values remain errors. A wildcard `go test ./...` encounters
the locked private extension, so default-build packages must be enumerated
separately when that source remains encrypted.

Windows CLI and GUI are built with `CmdName` set to `CraftBox` and
`CraftBox-gui`, respectively. Local test builds use a version ending in `-dev`
so they do not automatically query published releases. Explicit updates query the
Daniel-Ric repository. Keep official release asset names consistent with these
command names and include GitHub's SHA-256 asset digest.

## Live endpoint checks (2026-10-13)

`mco.cubecraft.net:19132` offers HTTPS NetherNet signaling and was used for the
successful proxy/world-export test. `play.inpvp.net:19132` resets HTTP and HTTPS
status requests (also on HTTPS 443). `mco.lbsg.net:19132` refuses TCP connections;
HTTPS 443 returns 404 for `/v1/join`. Both addresses answer the legacy UDP status
query with protocol 2193 / version 1.26.50. These checks establish no reachable
HTTP NetherNet endpoint at those tested addresses; they do not rule out another
provider endpoint or signaling service. A successful vanilla join can still use
its legacy transport. The application intentionally has no legacy fallback.

The CI native Linux GUI prerequisites must be installed independently of the
optional private extension key. The previous remote build failed because
`xkbcommon` and `wayland-client` development packages were omitted without that
key; the workflow now always installs its listed build dependencies.

The supplied preview trace advertises 1.26.70.20 / protocol 2235; preview compatibility is not claimed by the current official protocol-2193 sources.

## Hive observation (2026-10-14)

The official Hive support page lists `geo.hivebedrock.cloud` for Bedrock:
https://support.playhive.com/how-to-join/ . During local checks, the NetherNet
`/v1/join` probes on HTTPS and HTTP port 19132 at this address timed out.
The user confirmed that a direct Minecraft join works. That confirmation alone
does not establish which transport or signaling address Minecraft selected.
CubeCraft HTTPS signaling continued to work. No unverified alternative Hive
endpoint or legacy transport fallback has been added.

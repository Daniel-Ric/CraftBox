# NetherNet migration

## Hoplite spawn compatibility — 2026-11-06

### Downstream terrain-readiness follow-up

The next attempt, running `1.65.3-hoplite-spawn-fix-dev`, reached the server
successfully. The new recording (`ef48e4bd10a2282fd496255fd2b5b7c7`) exposed
a second ordering fault on the Minecraft-facing connection: StartGame was sent
at 10:59:21.783 UTC, the sole bootstrap LevelChunk at 10:59:22.209, but Minecraft
did not request its chunk radius until 10:59:23.419. Radius/spawn responses were
sent only then. Minecraft never sent SetLocalPlayerAsInitialised; the local
ICE connection failed later. Thus the prior upstream-radius fix did advance
the join, while the downstream loading phase still failed.

`StartGameContextWithReady` now releases queued upstream terrain only after
Minecraft's radius request has been processed and its radius/spawn responses
have been flushed. It still waits for the real client initialisation before
acknowledging that player upstream. The callback is consumed once and bound to
both connection and caller contexts, including cancellation during the flush.
Diagnostics now distinguish `connection.terrain_ready` from the existing
`connection.spawn_ready` completion event.

The ordering test failed before this change. A real local NetherNet test now
checks spawn-before-bootstrap-terrain ordering and delayed final initialisation.
A blocked-flush cancellation test also failed before the context guard and
passes with it. The complete Gophertunnel Minecraft and CraftBox proxy suites
pass. This verifies the corrected order locally; an authenticated live Hoplite
join remains necessary to confirm the Minecraft loading issue is resolved.
Binary: `builds/Craft-Box.exe`, version `1.65.4-hoplite-terrain-order-dev`.

### Upstream radius acknowledgement

The last recorded join to `http://mc.hoplite.gg:19132` completed ICE, DTLS,
SCTP and resource-pack exchange. The packet-inspector recording then showed
StartGame, ItemRegistry, our RequestChunkRadius(16), LevelChunk and
PlayStatus(PlayerSpawn), but no ChunkRadiusUpdated. The client-side handshake
incorrectly required the radius acknowledgement as well as PlayerSpawn, so
the dial never completed and Minecraft eventually closed its local connection.

ItemRegistry now marks the world data as complete. PlayerSpawn is still
required to finish the join. Until a server radius is supplied, the connection
uses the radius it actually requested (16); an earlier server acknowledgement
continues to take precedence. Proxy initialisation remains deferred until the
real Minecraft client finishes loading. Errors writing the radius request are
returned rather than leaving another pending spawn wait.

Regression coverage includes the observed packet ordering through the normal
receive/expected-packet/deferred-queue path, both standalone and proxy modes,
preservation of early terrain and late radius packets, the ordinary acknowledged
radius flow, and rejection of PlayerSpawn without world data. Removing the fix
reproduced the blocked spawn in both wire tests; restoring it passed the full
Gophertunnel Minecraft and CraftBox proxy suites, including the existing deferred
initialisation loopback test. A live authenticated Hoplite rejoin is still a
separate verification step. No endpoint, TLS or authentication bypass was added.

Evidence: session `session-20261106T091756Z-76244-b2c031669207`, inspector capture
`9d34cb6f51090c3f50efe6ecf2741358`. The local read-only evidence helper is
`tmp/hoplite-connect/main.go`; its output contains packet types/timing and
selected spawn/radius fields, not account/login payloads. Binary:
`builds/Craft-Box.exe`, version `1.65.3-hoplite-spawn-fix-dev`.

## Transport migration

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
Minecraft protocol 2225 (1.26.60 and compatible patch versions).

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

The supplied preview trace advertises 1.26.70.20 / protocol 2235; preview compatibility is not claimed by the current official protocol-2225 sources.

## Official dependency update (2026-10-29)

The source snapshots were merged against their previous recorded bases, preserving
the local NetherNet transport, resource-pack, capture and world-export hooks:

- Dragonfly: `v0.11.6`, commit `51dd9c1d615f17b70bbeef6ab2cbcee0545e66c0`.
- Gophertunnel: commit `25acc41eb24af21cfebf27ffa5ace37069838eb3`
  (`v1.63.1-0.20261028113046-25acc41eb24a`, including post-v1.63.0 fixes).
- go-nethernet: commit `74af70df9aa6b58da0d9089dc2c0aecbedfcea46`
  (`v1.0.27-0.20261028141106-74af70df9aa6`, including the post-v1.0.26 SDP fix).

This includes the 1.26.60 packet formats, block/item/recipe registries, offline
NetherNet identity handling and acceptance of the new NetherNet login address.
Skin metadata retains its historical `PlayFabID` JSON field as an empty string:
1.26.60 no longer supplies that field inside the skin packet.
The blob-cache reconstruction preserves the new `ClientBiomeUpdate` flag;
regression tests cover both cached blobs and blobs fetched after a cache miss.
Previous endpoint observations above are historical, not live 1.26.60 validation.

## Hive observation (2026-10-14)

The official Hive support page lists `geo.hivebedrock.cloud` for Bedrock:
https://support.playhive.com/how-to-join/ . During local checks, the NetherNet
`/v1/join` probes on HTTPS and HTTP port 19132 at this address timed out.
The user confirmed that a direct Minecraft join works. That confirmation alone
does not establish which transport or signaling address Minecraft selected.
CubeCraft HTTPS signaling continued to work. No unverified alternative Hive
endpoint or legacy transport fallback has been added.

## Initial Hive spawn placement (2026-11-03)

A read-only audit of the current Hive packet recording reproduced the reported fall. The upstream StartGame used position Y=0. Hive then sent its actual player teleport at Y=45.62, 18.954 seconds after recording began. CraftBox sent downstream StartGame with the stale origin at 19.015 seconds and forwarded the teleport at 19.043 seconds, before downstream PlayerSpawn at 19.450 seconds and real-client initialization at 21.280 seconds. Minecraft started reporting Y=0 and fell to Y=-38.38. Only the later manual spawn/dimension transition recovered it. Initial chunk data was already arriving at 18.929 seconds. The evidence identifies a spawn-ordering defect rather than establishing high network latency as the cause.

The proxy now observes authoritative initial self-player placement separately from the ordered relay queue and uses an already received position in downstream StartGame. Own-player movement received during client loading is retained until downstream StartGame actually completes. Only the latest placement is released, retaining subsequent rotation, and it is flushed before publishing spawn readiness. Terrain and unrelated packets continue to relay while the client loads. Pre-initialization PlayerAuthInput and own-player MovePlayer are discarded, including older buffered movement read after readiness. Clientless capture and ordinary post-spawn movement retain their existing behavior. No guessed coordinates, artificial timeout or automatic spawn command is introduced.

The info diagnostic connection.spawn_ready records whether StartGame position was corrected and whether player movement was held, without logging coordinates or credentials.

Verification reproduced the original failures before the fix, then covered placement arriving before and after StartGame, terrain flow during loading, early input suppression, latest teleport plus rotation, other-player traffic, stale input, normal movement after readiness, and cancellation. The three synthetic test cases passed 100 repetitions. An additional test decoded the actual recorded MovePlayer and PlayerAuthInput wire packets and confirmed the same Y=0 / Y=45.62 failure and corrected placement. Scoped Go vet and read-only review passed. Evidence is in tmp/hive-packet-audit.log, tmp/hive-spawn-verification.log and tmp/hive-recording-verification.log. This remains local protocol verification, not a live Minecraft reconnect test.

## Bedrock 26.62 dependency refresh (2026-11-05)

Mojang lists protocol 2225 with no packet changes for 1.26.62. CraftBox now
announces that compatible patch version while retaining protocol 2225.
`third_party/upstream-lock.json` records exact upstream commits for all five
source snapshots. Gophertunnel and Dragonfly were merged against the recorded
bases using the actual working files, preserving the existing local changes.
The new NetherNet, Xbox and PlayFab snapshots are included in both root module
replacements and the Go workspace. WebRTC resolves to 4.2.26, ICE to 4.4.8 and
DTLS to 3.1.15.

This imports the corrected camera spline, structure-block and video-capture
packet layouts; optional education-pack delivery; vertical-throw NaN fix;
NetherNet close/drain, LAN, HTTP-status and latency improvements; Xbox session
retry/reconnect/token fixes; and current PlayFab custom-ID/signature support.

Direct HTTP NetherNet now uses one endpoint client for the GET join probe and
the subsequent SDP POST, keeping the same network ID. Query metadata is taken
from the dialer's version, protocol and platform, including the existing
Android metadata used by authenticated logins. A successful probe can have an
empty body. Explicit POST-only endpoints remain supported. Supplied HTTP
clients, HTTPS/HTTP selection, identity assertions and authenticated JSONRPC
signaling paths are retained.

Validation includes independently encoded camera/video wire fixtures (failed
before the update), vertical throw regression, TLS/local HTTP endpoint tests,
plain and identity-bearing probe/offer ID checks, client metadata and POST-only
compatibility. The full offline test suites for NetherNet, Xbox and PlayFab
passed. Two imported tests were corrected to reflect current contracts: signed
PlayFab requests do not put their secret in plaintext JSON, and measured
Windows loopback ICE latency may round to zero. All product packages tested
successfully, as did the 29 frontend tests and production frontend build.
Gophertunnel resource tests used a workspace-local APPDATA because the sandbox
cannot write to the normal roaming profile. This does not change production
paths. Live remote-server, friend-world and Realm rejoins remain separate from
these local and loopback checks.

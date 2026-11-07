# NetherNet and experience-count audit — 2026-10-30

## Rate-limit regression — 2026-11-06

### Discovery routing follow-up

The deployed `1.64.0-player-counts-fix-dev` binary was verified by SHA-256.
Session `session-20261106T061033Z-69116-9deaef73bc93` still reports
`TooManyRequests` at 06:10:49, 06:20:52 and 06:32:10 UTC, so the cooldown alone
did not restore the data.

The application had reused the secondary catalog client's default host for
player-count data queries, ignoring the Gatherings `ServiceURI` returned by
Discovery. Player counts now use that discovered service with the supplied
Minecraft token. Catalog search keeps its existing secondary client; the shared
count cache and cooldown remain. Errors now include the count-service host.

A two-server HTTP regression test failed against the previous implementation
(count query sent to the catalog server, receiving 429) and passes after the
change (query sent to the discovered service with its token, receiving the
expected 42 players). Cache and GUI desktop tests also pass. This establishes
correct routing; live recovery remains unverified under the environment's
network restriction.

Build: `1.64.0-player-counts-discovery-fix-dev`, `builds/Craft-Box.exe`.

Session `session-20261106T055034Z-70976-91d57d921718`, event 10 at
05:50:47 UTC, records `minecraft/service: TooManyRequests: "Too many requests." (ServiceRuntime)`.
This is a rejected count query, rather than a missing catalog-ID match.

The account's Gatherings service now shares a synchronized 60-second count cache
between Marketplace and source selection. Concurrent requests coalesce; callers
receive independent map copies. HTTP 429 uses at least a 60-second cooldown and
honors a longer numeric or HTTP-date `Retry-After`. Caller cancellation does not
create a cooldown. Account service initialization returns the atomic winner so
concurrent callers use the same cache. Source details explain rate limiting.

Verified with loopback HTTP tests: concurrent/cache request count, immutable
results, rate-limit cooldown, expiration/refresh, real zero counts and cancellation.
GUI desktop tests, the Gatherings client build, 38 frontend tests, TypeScript,
Vite and Windows production build passed. External live recovery has not been
verified because this execution environment blocks network access.

Build: `1.64.0-player-counts-fix-dev`, `builds/Craft-Box-PlayerCounts-Fix.exe`.

## Original verification

The existing Gophertunnel client supports `GET /api/v2.0/dataquery/playercounts`.
CraftBox had omitted this read-only call. Live verification with the current
account returned HTTP 200 and 14 records; all seven catalog experiences matched.
Records contain `experienceId` and `playerCount`, with no capacity field.

Experience totals now load with the source list, without allocation or game
login. They are labeled `Experience players` / `Across sessions`. Zero remains
zero, missing records stay unknown, and failures do not block source selection.
The counts refresh with the source list; they are not per-match occupancy.

Source: [pinned Gophertunnel client](https://github.com/sandertv/gophertunnel/blob/25acc41eb24af21cfebf27ffa5ace37069838eb3/minecraft/service/gatherings/client.go).

| Route | Player metadata | Verification |
| --- | --- | --- |
| Public HTTP/HTTPS NetherNet | `/v1/join` players/maxPlayers | Live public status and native loopback tests; status and WebRTC results remain separate. |
| LAN NetherNet | Advertised server data | Discovery/network-ID/nonce and Minecraft login tests. |
| Experience | Gatherings read-only GET, grouped by experience ID | Live counts for seven experiences and real native/browser display, without allocation. |
| JSONRPC signaling | No occupancy fields in implemented messages | Source review: TURN credentials, delivery, WebRTC signaling, keepalive. `PongData` is not a count query. |
| WebSocket signaling | No occupancy fields in implemented messages | Source review: credentials, accepted/delivered messages and WebRTC signaling. Realm selection retains the advertised type. |
| Xbox friend world | Optional MPSD MemberCount/MaxMemberCount | Metadata tests and handle/signaling-path review. |
| Realm | Catalog capacity; online-player metadata is a separate Realms API | Lookup, ID, protocol selection and inspection-route tests. |

The full affected suites passed, including NetherNet identity/transport,
Gophertunnel Minecraft login/spawn, proxy packet exchange, pack handshake,
transfers, cancellation, disconnection and listener cleanup. No external Realm
or friend's-world game join was performed. A count never proves connectivity.

Saved addresses now select and check in one click, with selected styling and a
usable checked endpoint. Epoch guards discard stale checks after another
selection. Native/browser checks covered selection, Use source and narrow layout.

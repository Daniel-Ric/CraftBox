# NetherNet source browser

The World, Resource packs and Player skins views share one source browser.
`Choose source` opens the browser; manual addresses are entered in its Custom
address tab. The workspace displays the selected source and a compact status
summary. Captured `.pcap2` sessions remain selectable without a network probe.

The browser has a searchable source list and a separate detail pane, or a stacked
layout in narrow windows. The selected row does not immediately start a download.
Use source confirms it. Up to twelve checked custom addresses can be saved locally
and removed individually. Stored addresses do not cause background connections.

## Discovery and connection are separate

1. **Status:** GET `/v1/join` with the current game version, protocol, Windows
   platform and an ephemeral network ID. Bare addresses prefer HTTPS and then
   HTTP on the supplied port, default 19132. Explicit schemes are respected;
   explicit HTTPS is never downgraded. IPv6 addresses use brackets.
2. **Transport:** Test connection opens the actual NetherNet WebRTC data channels
   and immediately closes them. When an account is available, its multiplayer
   token is bound to a new P-384 key and presented in the SDP identity assertion.
   The status request and SDP exchange use the same network ID.
3. **Game session:** Minecraft login, pack negotiation and gameplay forwarding
   are checked during the actual download. A successful status or transport test
   never claims that these subsequent steps have passed.

HTTP status latency includes DNS/TLS/request overhead. It is not an in-game RTT.
A failed probe means the endpoint was not verified by this route; it does not
establish that the entire server network is offline. An anonymous `401` transport
failure directs the user to Microsoft sign-in.

## Displayed information

| Source | Available details |
| --- | --- |
| HTTP NetherNet status | Server message/MOTD, level, version, protocol compatibility, current/max players, game mode, measured status latency, checked endpoint and timestamp |
| Extended status, when advertised | Online/self-signed authentication support, editor, hardcore, transport layer, connection type, discovery data version |
| Friends' worlds | Host, version/protocol, optional session member counts, joinability, editor/hardcore and cross-play restrictions; authenticated Xbox signaling is negotiated when downloading |
| Realms | Owner, MOTD, open/closed/expired state and advertised capacity; no online count inferred from the membership roster |
| Events | Description, enabled/time-window availability, start and end dates; endpoint status on selection |
| Featured servers | Catalog entries appear immediately; public servers receive bounded status checks. On-demand experiences are resolved explicitly and show that no public player count is supplied |

Unknown numeric data stays absent and displays `?` / Not provided. Explicit zero
and false remain distinguishable from missing fields. Minecraft formatting codes
are removed from HTTP server names and level names. Nonces, tokens and candidate
credentials are not exposed as user-facing server metadata.

## Native interface contract

`ListSources(kind) -> Source[]` preserves existing IDs and cached ConnectInfo
objects. Its network operations have a 25-second context deadline. Metadata is
optional, so older catalogs remain valid. Account changes invalidate results.

`InspectSource(target, connection) -> sourceprobe.Result` accepts either a cached
source ID or a custom endpoint. It resolves a copied ConnectInfo with the current
account and has a 20-second total deadline, including waiting for one of four
global probe slots. Individual status attempts stop after four seconds; responses
are bounded to 64 KiB and redirects are not followed. Only HTTP/HTTPS endpoints
without credentials, paths or queries are accepted (one trailing slash is allowed).

Result contains `reachable`, optional metadata, optional `compatible`,
`latencyMs`, `endpoint`, `checkedAt`, `error`, and `transport`:
`not-tested`, `connected`, or `failed` with `transportError`.
Missing counts are not serialized as zero. Status errors remain distinct from
transport errors. Malformed/empty non-NetherNet JSON is rejected. A catalog error
does not block manually checking a custom address.

The frontend discards stale responses after tab/account/address changes and
deduplicates in-flight checks for each listed source. Automatic featured checks
have at most three concurrent requests, skip on-demand experiences, and never start a Minecraft game login.
These methods are bound through Wails; no new public HTTP API is introduced.

## Research and verification, 2026-10-29

- [Mojang HTTP signaling guide](https://mojang.github.io/bedrock-protocol-docs/guides/nether-net-onboarding-guide/)
  documents discovery metadata, SDP exchange and the distinct ICE/data-channel phase.
- [go-nethernet endpoint implementation](https://github.com/df-mc/go-nethernet/tree/74af70df9aa6/endpoint)
  supplies the existing transport integration. Local code uses the pinned dependency.
- [NetherNet LAN specification](https://github.com/df-mc/nethernet-spec)
  and [discovery package](https://pkg.go.dev/github.com/df-mc/go-nethernet/discovery)
  describe binary LAN advertisements. They are a separate discovery route; HTTP
  endpoint failure is not proof that Xbox/LAN signaling is unavailable.
- API discovery through Orbit returned generic Minecraft status services, without
  the required NetherNet signaling contract. No external status aggregator was added.
  The Postman CLI was unavailable; local real-HTTP fixtures and native transport
  tests exercise the client rather than substituting its HTTP methods.

Live native check: CubeCraft's `mco.cubecraft.net:19132` answered over HTTPS with
MC 1.26.60/protocol 2225, 7,987 / 55,000 players, a 142 ms status request, and
extended authentication/transport metadata. The subsequent anonymous SDP offer
returned `401 Unauthorized`. HTTP returned `426 Upgrade Required` in the separate
direct check. Counts and latency are snapshots, not current promises.

Direct checks of `geo.hivebedrock.network` did not establish a reachable status
endpoint on the tested routes. No claim is made that Hive is offline; its current
authenticated catalog resolver path was not validated by that direct check.

Automated checks cover valid/malformed/missing/zero/extended metadata, cancellation,
version mismatch, address validation, distinct status/transport failures, actual
local NetherNet channels, stale responses, duplicated requests and selection into
the existing download flow. Browser verification uses the real App, a local native
InspectSource bridge, and a real local NetherNet listener: player count, successful
WebRTC check, saved-address selection, wide/narrow layouts and light/dark themes.
The public-server authenticated login path still requires a signed-in runtime.

## Creator-experience correction, 2026-10-29

Live authenticated allocation confirmed that GenWars returns
`networkProtocol: NetherNet_JsonRpc`, `netherNetId`, and destination metadata,
with no IP/port or player counts. The old resolver discarded these fields and
returned `:0`, producing the misleading hostname-validation message. The UUID
is a messaging destination, not an Xbox world handle: treating it as a handle
produced MPSD 404.

`ResolveExperience` now preserves the typed assignment and validates its
transport-specific destination. JSON-RPC targets use a distinct
`nethernet-jsonrpc:<uuid>` route. The probe and capture paths open authenticated
messaging directly and bypass Xbox world-handle joining. Missing destinations or
unsupported allocation protocols fail explicitly instead of becoming `:0`.

Catalog entries without a public address have `onDemand: true`. Browsing them
does not allocate sessions. Experience-wide counts now come from the read-only
player-count GET, with Resolve session and Test connection as separate actions.
Starting a download resolves the original experience again, avoiding reuse of
an expired assignment. The probe result has `discovery: session` and the actual
`networkProtocol`; successful allocation is distinct from transport connectivity.
No HTTP latency is inferred for messaging sessions.

Native live test and a complete browser flow both successfully resolved GenWars,
displayed `Overworld (Matchmaked)`, and established the authenticated NetherNet
WebRTC transport. The browser confirmed that no GenWars request was made before
Resolve session, then selected the original source ID for downloading. No game
login or pack download was started by these checks. The catalog-based HTTP check
also confirmed The Hive's current endpoint `nethernet.playhive.com:19132` works;
the earlier direct failure referred to its old address.

## Fixed-server classification, 2026-10-30

Some catalog records contain both a public URL/port and an experience ID. The
endpoint takes priority: Featured servers now lists only public endpoints, while
Experiences lists entries with an experience ID and no endpoint. Their complete
catalog remains available for resolving previously saved experience names.
Selecting a public source also prefers its endpoint over allocation.

Public featured entries have `onDemand: false`, enabling the existing bounded
automatic status queue. Player counts no longer require resolving an allocation.
True on-demand experiences retain explicit resolution for their assigned session.
Experience-wide totals are fetched separately through the gatherings count GET;
see `nethernet-player-counts-audit.md`.

Live authenticated discovery returned seven public servers and seven on-demand
experiences with no overlapping names. Read-only public status checks returned
Hive, CubeCraft and Lifeboat player counts without allocating a game session.
Regression tests cover records containing both fields, distinct lists, legacy
resolution and automatic frontend player-count requests.

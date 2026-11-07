# Confirmed join disconnect: cache acknowledgement list limit

The next native SoulSteel join produced the previously masked cause:
`decode packet *packet.ClientCacheBlobStatus: slice length was too long: length
of 2054 (max 1024)` at 08:53:45 UTC in session
`session-20261102T085157Z-55016-0689afa18569`.

The strict local Minecraft listener applied the general `protocol.FuncSlice`
limit of 1,024 to each cache MISS/HIT list. Returning clients aggregate cache
acknowledgements across chunks, so their legitimate list exceeded this limit.
Rejecting it closed the client connection. The improved close-cause and transport
logging from the preceding build exposed the exact packet and count.

`ClientCacheBlobStatus` now uses its own 4,095-entry bound for each list, checked
before allocation. The generic 1,024 bound for other packets remains unchanged.
Count/hash ordering, little-endian hash representation, and truncation checks
are preserved. The current Cloudburst v1001 serializer applies 4,095 separately
to NAK and ACK arrays; see
[ClientCacheBlobStatusSerializer_v1001](https://github.com/CloudburstMC/Protocol/blob/3.0/bedrock-codec/src/main/java/org/cloudburstmc/protocol/bedrock/codec/v1001/serializer/ClientCacheBlobStatusSerializer_v1001.java).
The original client-cache design also describes the
[4,095-ID bound and batched acknowledgements](https://gist.github.com/Tomcc/4be79d3eafcd158c5059abd4ab2e8d35).

## Reproduction and verification

- Golden wire cases for both lists: 0, 1,024, 1,025, **2,054**, and 4,095 entries.
  The original decoder fails these valid cases above 1,024.
- Lists of 4,096 and truncated bodies remain rejected. Generic slices above
  1,024 remain rejected before allocation.
- A real localhost NetherNet login/cache test now sends 2,054 client ACK hashes
  through the strict server listener, verifies all hashes, and receives a text
  reply afterward to establish that the connection remains usable.
- A Go build overlay containing the original codec reproduces exactly the native
  2,054/max-1,024 disconnect in that network test. The patched codec passes it,
  including 10 consecutive runs.
- All gophertunnel Minecraft tests and affected CraftBox GUI/proxy/world/logging
  packages pass. Independent scoped review found no must-fix issues.

The native rejection shown in the new log is confirmed and locally reproduced.
The previous cache-ownership/FIFO fixes remain in place. A live native join with
this new build is still the final end-to-end confirmation.

Build: `1.63.1-cache-ack-limit-wails.20261102-dev`, Windows GUI subsystem,
complete portable assets, no extra console window.
Artifact: `builds/resourcepack-fix/Craft-Box.exe`, 250,222,592 bytes.
SHA256: `05F9272172955B2B840E0EEA8D6F2BB07C432C0932D412F244B320A2D94A168C`.

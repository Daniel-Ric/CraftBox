# SoulSteel join disconnect: cache mutation and inbound reordering

## Evidence

The failing session is `session-20261102T075337Z-35208-75e1ac22f530`.
All seven resource packs completed. The upstream connection was established at
07:57:27.435 UTC. At 07:57:29.510 UTC the local-client read loop ended with
`use of closed network connection`. CraftBox stayed alive and saved 30 chunks.
The `playerWaypoints` warning occurred afterward during export and is not the
disconnect trigger.

The isolated failing capture (`tmp/join-disconnect-0857/input.pcap2`) contains
2,240 packets: 1,676 server-to-proxy and 564 proxy-to-server. There is no explicit
Disconnect, Transfer or PacketViolationWarning packet. All captured client-bound
directions and client packet bodies were examined, including strict listener
reader limits; no captured client packet failed that check. The PCAP cannot
prove the exact final client-side decoder error because local inbound traffic
rejected before forwarding is not in that upstream capture.

## Confirmed defects and fixes

1. **Shared cache-entry storage corrupted forwarded packets.** `HandleSubChunk`
   copied the outer packet but reused its `SubChunkEntries` slice. Reconstruction
   replaced `BlobHash` with `RawPayload` in that shared slice. The original
   forwarded packet retained `CacheEnabled=true` while its reference fields had
   changed. Clone the entry slice before reconstruction. Both immediate warm
   cache resolution and later asynchronous cold-cache resolution are covered.
   The optional wrappers and heightmap arrays are value fields; reconstruction
   replaces payload references without changing payload bytes.
2. **Inbound packets could overtake earlier packets.** `Conn.receive` moved old
   channel data into a deferred queue before sending new channel data. A reader
   checked the deferred queue only before entering its channel select. A producer
   could move the earlier packet after that check, causing the reader to receive
   the newer packet first. A real NetherNet login/data test reproduced packet 49
   overtaking packet 48. Use one mutex-protected FIFO with a separate readiness
   signal for all read APIs. Preserve packet ownership, propagate wakeups while
   data remains, and honor cancellation/ready deadlines before dequeuing.
3. **Pending capture dependencies resolved the wrong packet.** After a LevelChunk
   completed, the resolver recursed on the LevelChunk again instead of the waiting
   SubChunk. Resolve the actual dependent waiter.
4. **No-cache clients could receive duplicate reconstructed chunks.** Immediate
   resolution wrote the packet and returned it for normal forwarding. Only
   asynchronous resolutions now write directly; the synchronous path returns its
   packet for one normal send.
5. **Diagnostics hid the original failure.** Packet and batch decoder errors
   could be replaced by `net.ErrClosed` during cleanup. Preserve the original
   close cause. Route `slog` and standard-library logging directly into persistent
   diagnostics instead of relying on a stderr handle captured before GUI startup.
   Grouped fields retain their scope and credentials remain redacted.

The protocol fields were cross-checked against Mojang's
[protocol-2225 SubChunk documentation](https://mojang.github.io/bedrock-protocol-docs/1.26.60/packets/sub-chunk-packet/).

## Verification and limits

- Failing-capture regression: before fix **253/253** cached SubChunk packets
  changed during reconstruction; afterward **0/253** changed.
- Previously successful capture: **0/4,704** cached SubChunk packets changed
  afterward. These tests warm the proxy cache with the actual captured blobs.
- Warm/cold cache, dependent waiter and duplicate-delivery tests failed on the
  defective paths and pass after their corresponding changes.
- Actual localhost NetherNet login plus 100 ordered cached subchunks: initial
  test exposed reordering; after FIFO fix passed **10 consecutive runs**.
- Concurrent arrival and buffer-reuse test: 3,000 ordered packets through each
  of ReadPacket, ReadBytes and Read, plus cancellation/deadline/wakeup checks.
- Decoder-cause tests failed before and pass afterward.
- Closed-console subprocess test previously lost transport logs; now preserves
  transport errors, grouped attributes and stdlib output while redacting secrets.
- All `gophertunnel/minecraft/...` tests and affected CraftBox GUI/backend packages
  passed. Independent review found no remaining regressions in the FIFO/cache fix.

These are confirmed implementation defects and locally reproduced fixes. A new
native Minecraft join against SoulSteel is still required to confirm that the
user's exact live disconnect has disappeared; neither the original masked log
nor an upstream packet capture alone can prove its final client-side cause.

An existing one-shot SetReadDeadline behavior is outside this join fix: repeated
reads after a consumed timeout do not keep observing that timeout until reset.
It was already present before these changes and is not evidence for this failure.

Artifact: `builds/resourcepack-fix/Craft-Box.exe`, version
`1.63.1-join-cache-fifo-wails.20261102-dev`, built with `gui,production` and
`-H=windowsgui`. The complete portable asset bundle and onboarding fixes remain.
SHA256: `8F6D4C71E9EECEB598B0C2987D35AD015E7672EB1E1D1878F4B5980D9162B497`;
size: 250,220,544 bytes; Windows GUI subsystem 2 verified.

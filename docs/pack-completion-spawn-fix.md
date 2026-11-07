# Pack capture completion

The Hive capture in `session-20261103T082531Z-75108-d61e4c4ce0d9` downloaded its final resource pack at 08:28:09 UTC and reported `Done!`. The resource-pack command intentionally closes the server connection after sending `PackResponseCompleted`. The session then attempted `DoSpawnContext` on the closed connection, surfacing `do spawn ... use of closed network connection` as a desktop error despite successful pack capture.

`Session.runGame` now stops before entering the game when a deliberate server disconnect has been requested. If that request arrives during the spawn wait, only its closed-network error is treated as successful completion. An unexpected closed connection, timeout or malformed-packet error still propagates. The cross-goroutine completion marker is an `atomic.Bool`, published before closing the server connection.

Four regression cases were run red/green: completion while waiting to spawn, completion before entering spawn, unexpected spawn failures, and protocol failure concurrent with a deliberate close. The final case includes joined closed-socket/protocol and closed-socket/deadline errors; all non-closure causes remain visible. The tests also ran 100 times. Scoped proxy/resource-pack/command/handler compilation, Go vet and read-only review passed. The race detector is unavailable in this environment (`CGO_ENABLED=0`); this is not a live NetherNet capture test.

The fix changes session completion handling only. The unified UI and Marketplace changes remain included in the v2 executable.

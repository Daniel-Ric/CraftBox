# Friend world connection, 2026-11-05

The Farm It attempt in session `session-20261105T122331Z-54228-d9f3d4db85f5`
logged `unresolved trait minecraft:multi_block` at 12:27:30 UTC. StartGame had
already been marked received, but its block registry and dimension were not
ready. A subsequent cache-miss response delivered a LevelChunk and crashed in
`chunk.New` with a nil registry.

The fix maps the network trait's `multi_block` flag and `part_count` to integer
`minecraft:multi_block_part` states. Behavior-pack export writes the corresponding
enabled state, parts and string direction. Disabled traits are omitted. The pack
format is documented by [Microsoft](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/blockreference/examples/traits/multi_block?view=minecraft-bedrock-stable).
StartGame is marked ready only after initialization, pre-play handler failures
cancel the session with their original cause, and chunk capture rejects missing
initialization with an error instead of dereferencing nil.

Replaying the recorded 2,195 block definitions exposed a second startup panic:
two distinct add-on states share the same 32-bit network hash. Registry
finalization now preserves distinct sequential IDs and marks colliding hashes
ambiguous. Only capture decoding substitutes `minecraft:unknown` for such a
hash; a warning reports this limitation. Original packet payloads, raw captures
and forwarded hashes are unchanged. Unknown hashes without a proven collision
still return an error.

Validation: the recorded definitions now finalize successfully into 92,116
states. Regression tests cover multi-block parts, pack export, disabled flags,
initialization failure, session cancellation, and the real colliding states,
including capture fallback, clone metadata and preservation of wire bytes.
This is an offline verification; a live friend-world rejoin remains to be tried.

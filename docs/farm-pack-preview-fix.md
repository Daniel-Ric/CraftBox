# FarmIt preview and disconnect diagnostics

The capture in session `session-20261105T124315Z-13216-d5357dfc367e` reached
client spawn at 12:50:17 UTC. Its final upstream packet at 12:51:14 UTC was
Disconnect with reason 55 (`Kicked`), both message strings empty and the
disconnect screen flag false. There is no evidence of a local crash or a
specific reason for the host's decision. The transport previously discarded
the reason number and surfaced a blank error. It now retains the code and
provides fallback text; the UI explains a kick with no supplied reason. This
improves diagnosis and does not claim to prevent a remote kick.

The same session logged `Invalid block identifier` when saving preview
bindings. `BlockRenderJSON` returned a bare block body; the consumer expects
the `format_version` and `minecraft:block` document wrapper. Correcting this
contract restores the actual geometry/material associations without texture
filename guesses. Captured visual definitions for the exact FarmIt pack hash
are bundled, so that already captured copies can resolve their models too.

World saving also failed on `entity/livestock/buffalo.entity.json: EOF`.
That file starts with a UTF-8 BOM. The shared JSON parser now strips the BOM;
all actual FarmIt client entity definitions passed parsing afterward.

Validation: all 2,195 recorded definitions passed capture conversion and saving.
The pack catalog produced 6,000 preview objects with texture bindings (the
existing catalog limit). The reported gate uses
`textures/goe/farm/blocks/fences/wood_fence.png`; its actual geometry and texture
were visually checked in the application's AssetViewport with headless Edge
(`tmp/farm-gate-textured.png`). Unit tests cover the producer/consumer contract,
the bundled gate association, BOM parsing, and disconnect reason preservation.
All 29 frontend tests and the production build passed.

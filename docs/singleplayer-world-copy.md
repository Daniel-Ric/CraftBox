# Copy a singleplayer world

In **Worlds**, use **Copy singleplayer world** (the folder/download icon in the header).
Choose a discovered Minecraft world, or select its world folder, `minecraftWorlds`, or
`com.mojang` manually. Close the selected world in Minecraft before exporting.

The preview checks exact pack UUIDs and versions, including transitive add-on
dependencies. Embedded packs, installed resource/behavior packs, development packs
and available template/premium pack folders are searched. **Find packs** adds a
manually chosen dependency folder. Global texture packs are optional.

**Export .mcworld** copies the original database, player inventories, metadata and
available pack files into a Bedrock import archive. Pack activation lists retain
their order and settings. Import the file on the receiving device by opening it
with Minecraft. The original world is not modified. Export progress and cancellation
use CraftBox's task system; incomplete exports do not replace an existing output.

Marketplace/template settings and encryption are preserved. Protected content is
not unlocked and may still require the receiving account to own it. Missing,
ambiguous or protected dependencies block the complete export; **Backup only**
explicitly permits a copy of the available files with a dependency report.

Ingame additions:

- `/cb position on`, `off` or `toggle` controls a live position dot on CraftBox's
  ingame map. It starts disabled and does not alter exported map images.
- `/cb connection` shows known connection state, source, available endpoints,
  transport, Minecraft protocol/version and measured latency without credentials.

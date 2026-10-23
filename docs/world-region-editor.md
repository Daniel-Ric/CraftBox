# Manual world regions

During a world session, the map window can select a rectangular set of complete
16 by 16 block columns. A named region has an independent ID and tab, receives
subsequent tiles from the same source visit, and is finalized on session end,
transfer, dimension change or /save-world. Coordinates remain unchanged.

The editor offers separation from the source (default) or an additional copy.
Regions of one source must not overlap. A finished visit cannot acquire new
live regions. Empty regions retain a tab but produce no empty world export.

Exports derive from the finalized capture database, preserving block inventories,
biomes, resource/behavior packs, actor inventories and world rules. Chunk data,
actor indexes and actors are filtered together. Each region receives a separate
mcworld and, when image generation is enabled, a separate PNG. Capture options
and encrypted-pack restrictions continue to apply.

If the last player position lies outside an export, the player and respawn point
are moved into retained terrain. The search stays inside the dimension, requires
floor and two air blocks, and works below Nether ceilings. If no standing space
exists, saving reports an error and retains the captured folders for recovery;
choose a larger region. Local player inventory is preserved in each export.

The local authenticated bridge adds POST /world-region with JSON
{worldId, name, bounds: {minX, minZ, maxX, maxZ}, move}. Bounds are inclusive
chunk coordinates. Success returns the new WorldRegion DTO. Invalid input,
inactive visits and overlaps produce a readable error. This bridge exposes no
login, capture-stop or update operation to the map process.

The map starts with all captured terrain in view. The overview button fits all
terrain again, while player follow is an explicit toggle. Holding the middle
mouse button pans even during rectangle or spawn selection. Spawn selection is
per tab: click captured terrain and adjust X/Y/Z, then save. Double-click a tab
to edit its world name. Both operations also work on finished exports and rebuild
the mcworld with its existing packs/inventories intact. File paths remain stable.
Extracting a region containing the source's manual spawn clears that spawn;
the retained source gets an automatic start point when finalized.

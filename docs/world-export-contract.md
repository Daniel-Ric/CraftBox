# Local world export contract

The desktop bridge exposes `PreviewWorldExport({worldId, options})` and
`ExportWorldCopy({worldId, options})` for completed worlds only. Neither accepts a
filesystem path. Exports create a new sibling world/archive and preserve the source.
`CancelWorldExport()` cancels the current job; `GetWorldExportProgress()` reports
copy, database, metadata, compaction and archive stages.

`GetWorldExportCatalog(worldId)` enumerates dimensions/chunks/biomes without
validating player or spawn positions, keeping repair controls accessible.
`playerDimension?: int32` chooses the retained source dimension for an explicit
replacement spawn; it requires `settings.spawn`.

Options extend the world settings contract with:

- `dimensions`: optional array of `{source: int32, target: int32 | null, bounds?:
  {minX,minZ,maxX,maxZ}}`. Missing rules retain a dimension. `target:null` excludes
  it. Bounds are inclusive chunk coordinates in the source dimension.
- `biomes`: optional array of `{source: uint32, target: uint32}`. Target IDs must
  exist in the bundled vanilla biome registry. Replacements apply once, simultaneously.
- `compact`: optional boolean; compaction runs on the new destination database.

Preview enumerates actual source dimensions, included/excluded chunk coordinates,
biome choices, record counts and blocking errors. Duplicate dimension/biome rules,
colliding target chunk coordinates, an empty selection, invalid bounds and an
excluded player/spawn location are blocking errors. Custom dimension IDs are
supported. Relocation uses an explicit spawn override in a retained dimension;
custom targets must already occur in the source; this API does not create new
dimension definitions or behavior packs.
dimension remapping preserves coordinates and must fit the target height range.
Terrain, block entities, item NBT, light/height records, resource packs and unknown
global records are copied as data. Dropped actors and dangling retained-actor links
are removed. Any invalid transformed record aborts the destination before publication.
Cancellation and failures remove the incomplete copy. Source data is opened read-only.

Explicit game-mode overrides offer Survival, Creative and Adventure. Existing
source spectator values retain Bedrock ID 6 when untouched. New spectator world
initialization is excluded, following Chunker's documented post-1.19.50
initialization limitation in `BedrockLevelWriter.writeLevelSettings`.

## Imported render dimensions

`OpenRenderWorld` automatically selects Overworld when present, otherwise the
first recorded dimension. Its `RenderWorld` result includes `dimension` and
`dimensions`. `SwitchRenderDimension(currentDocumentID, sourceDimensionID)`
reloads the original archive/folder and returns a new document ID. Failed switches
retain the old document; switching is unavailable for standalone saved scene
projects. Unregistered source IDs use local storage identities, never global
dimension registration. Their height must be unambiguously inferable from complete
subchunk/biome data, otherwise an explicit import error is returned.

# Chunker-inspired world fidelity and export work

Scope authorized by the user: ideas **1, 2, 5, 7, 8, 9, 10, 11, 12, 13, 14, 16** from the conversation. The repeated full list is the reference for their meanings. Ideas 3, 4, 6 and 15 are not separately requested. Preserve existing worktree changes.

## Requirements and evidence ledger

| Idea | Outcome | Authoritative implementation / evidence |
|---|---|---|
| 1 | Preserve vanilla identifiers/states, detect missing mappings and avoid lossy native fallback | `utils/merge/block_registry.go` and its full native palette, extension, type and wire-hash tests; independent pinned asset audit in `palette_audit_test.go` covers 1,480 identifiers. Piston-arm aliases corrected. Later surface audit corrected 160 horizontal pillar states, all furnace directions and barrel opening/facing states. Asset exceptions and rendering limits are stated below. |
| 2 | Compute neighbor connections before shared preview/model export | `connection_rules.go`, `stairs_shape.go`, `vanilla_forms.go`, `mesh.go`; tests cover fence families, gate axes, pane/wall support, wall heights/posts, resolved stair-corner support, negative chunk boundaries, authoritative recorded connections and explicit export crops. Cached source states remain unchanged. |
| 5 | Robust import of supported historical Bedrock formats | Native `chunk` tests cover subchunk versions 1/8/9, encoded Y, legacy upgrades and malformed palette bounds/types/indices. Native `mcdb`/`leveldat` tests cover metadata versions, unknown compounds, read-only behavior, Data2D fallback and future-version errors. `render_dimensions_test.go` covers Nether-only/custom imports and dimension switching. |
| 7 | Editable world name/spawn/mode/difficulty/time/weather/rules | `internal/worldexport/settings.go`, `metadata.go`, capture finalization and WorldExportSettingsDialog. Metadata tests retain unknown fields and inventory; world/capture tests retain server originals; explicit validation rejects bad settings. Native spectator source values are preserved, with UI override limits documented below. HTTP contract and local tests are in `docs/map-window-api.yaml` and `utils/nethernet/p2p/world_metadata_test.go`. |
| 8 | Original, creative visit and screenshot profiles | `profiles.go`, metadata baseline/reversal tests and settings UI test. Explicit overrides take precedence; returning to original restores gameplay fields without overwriting subsequent inventory/unknown metadata. |
| 9 | Per-dimension chunk pruning with excluded-chunk preview | `transform.go`, WorldCopyDialog and its tests. Bounds use inclusive source chunk coordinates; preview reports kept/excluded full counts and explicitly indicates the 50,000-chunk drawing cap. Cancellation/failure removes incomplete destinations. |
| 10 | Keep/drop/remap vanilla and recorded custom dimensions | Raw chunk/digp keys and actor dimension IDs remap; tests reject collisions, undefined custom targets, excluded player/world/bed spawns and target height violations. Explicit relocation repairs invalid selection. Custom IDs import locally without global registry mutation; missing custom definitions are not invented. |
| 11 | Biome replacement rules in exported copies | Simultaneous 3D/legacy-2D palette replacement tests preserve indices and all 512 height bytes and reject invalid target IDs/truncation. WorldCopyDialog exposes source IDs and native target choices. The 1,755-chunk Hive fixture exports with a biome rule and reimports. |
| 12 | Preserve inventory, block entity and frame data | Raw copy regression preserves item/entity/global NBT; `utils/item_nbt_test.go` checks slots, unknown tags and source-session identifiers. Open-container flush, native block-NBT initialization and item-frame capture/model tests cover live capture. Invalid representations fail explicitly instead of silently dropping items. |
| 13 | Compare and preserve light/height processing | Chunker ColumnReader/Writer and ColumnUtil were compared with mcdb. Copy tests preserve arbitrary light records and raw height bytes during transformations. Existing propagation/persistence tests remain green. Missing server light and unchanged new-capture height cache behavior are explicit limits below. |
| 14 | Optional LevelDB compaction with progress/cancel | Destination-only CompactRange in `transform.go`, desktop progress/cancel bridge and WorldCopyDialog. Tests cover compacted copies, unchanged source and cancellation cleanup. Real Hive fixture copied/compacted/reimported 1,755 chunks. |
| 16 | Mapping/format/state regression coverage | Native palette roundtrip, independent asset audit, legacy/modern format guards, neighbor geometry, inventory/metadata fidelity, biome and dimension transformations, UI regressions and actual WebGL/GLB fixtures. Final production-package test command and known scratch-test exclusions are recorded below. |

The selected implementation is reviewable in the worktree. Installation and old
EXE cleanup were subsequently verified as completed by read-only file inspection.
No claim is made that every vanilla block already has a bespoke visual model.

## Architecture inventory

- Scene models: `internal/scene/mesh.go`, `vanilla_forms.go`, `connected_native_models.go` and model family helpers.
- Native palettes/upgrades: `third_party/dragonfly/server/world/block_registry.go`, `block_state.go`, `world/chunk/encoding.go`, `decode.go`.
- Imported worlds: `ui/desktop/render_editor.go:importRenderWorld` -> read-only mcdb -> scene store.
- Captured worlds: `handlers/worlds/worldstate/world.go` -> mcdb; main packaging in `handlers/worlds/world.go`.
- Region exports: `handlers/worlds/region_exports.go`, `region_database.go`; requests in `ui/messages/world_regions.go`.
- UI/API: `ui/desktop/world_regions.go`, `frontend/src/bridge.ts`, atlas/asset library/editor dialogs.

## References

- [Chunker WorldConverter](https://github.com/HiveGamesOSS/Chunker/blob/main/cli/src/main/java/com/hivemc/chunker/conversion/WorldConverter.java): separate settings, dimension mappings, biome mappings, pruning and processing controls.
- [Chunker mappings](https://github.com/HiveGamesOSS/Chunker/tree/main/cli/src/main/java/com/hivemc/chunker/mapping).
- [Advanced options](https://learn.microsoft.com/en-us/minecraft/creator/documents/chunkeradvancedmode).
- [World settings](https://learn.microsoft.com/en-us/minecraft/creator/documents/chunkerworldsettings).

## Verification environment

Go executable: `C:/Users/danie/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.windows-amd64/bin/go.exe`; GOPATH `C:/Users/danie/go`; GOCACHE `D:/Code Projekte/GoLand/.cache/go-build`. Frontend tests use Node `C:/Program Files/nodejs/node.exe` with Vitest, maxWorkers=2. Workspace writes are limited to GoLand; installation outside it uses the existing install script.

## Additional user correction: candles

User screenshot showed transparent candle sprites wrapped onto a full cube. Added
small separate wax cuboids with cropped UVs, wicks, 1–4 layouts, lit variants and
the pinned vanilla flame sprite; candle cakes combine raised candle and cake.
Tests cover all 17 colors × 4 counts × 2 light states, opaque wax UV samples and
cake geometry. Actual WebGL and GLB load screenshots match:
`tools/light-audit/candles-closeup.png`, `candles-overview.png`, `candles-exported.png`.
Independent review found no concrete candle bug. Included in the verified installed EXE.

Final workspace build: `builds/Craft-Box.exe`, version
`1.63.1-world-fidelity-dev`, 85,547,520 bytes; SHA256
`5C561450F6570D83A6D1DFE043885F2D7D6F3AD23803539702CB99826226EBF6`.
Windows icon group (9 images) and application manifest verification passed.

## UI and export verification

`tools/verify-world-export.mjs` passed in actual headless Edge at 1280×820 and
1024×640; dialogs remain within the viewport, scroll correctly, and SVG preview
shows exclusions. Python/Playwright helper was unavailable, so the existing
Edge DevTools approach was reused. Reviewer found and tests fixed initial catalog
validation disabling repairs, flying player target Y, and independent bed spawn
remapping. Contract: `docs/world-export-contract.md`.

## Final verification and limits (2026-11-01)

The earlier table records implementation checkpoints; its remaining audit notes
are superseded by the following results:

- Full native state palette and pinned asset audit pass. The 172 explicitly
  unavailable assets remain a documented limitation, not silently substituted.
- Recorded directional states were inspected in the real Hive capture. Preview
  and GLB preserve them normally; an explicit crop clips fence/pane/wall arms and
  recalculates stair corners/end posts without mutating cached source models.
- Imports select a stored dimension, including Nether-only worlds, and both
  editors can switch dimensions. Custom dimensions require complete contiguous
  subchunk/biome records to infer their height range; incomplete data fails.
- Metadata, player/spawn/bed-spawn remapping and profile reversal pass regression
  tests. New game-mode overrides support Survival/Creative/Adventure; an existing
  source Spectator value is preserved. New custom target dimension definitions
  are not fabricated: a custom target must already exist in the source.
- A copy of the user's Hive recording exported and reimported all 1,755 chunks
  with a biome rule and LevelDB compaction. Source metadata remained unchanged.
- Raw copies retain inventory, block entities, item frames, unknown NBT and all
  512 height bytes. Captured items preserve source slots and NBT; open containers
  flush before saving, and unrepresentable item values fail explicitly.
- Chunker's height/light pipeline was compared. Existing recorded light records
  are retained; unavailable server-side light is not reconstructed. New-capture
  height cache generation remains unchanged rather than using a guessed light
  heuristic. No Minecraft-client reopening test was performed.
- Final full frontend run: 101 files / 339 tests passed, including the editor
  dimension selector regression test. TypeScript and production frontend build
  passed. All production Go packages and native chunk/mcdb tests passed; changed
  scene/export/desktop packages were rerun after the last fixes.
- Unfiltered `go test ./...` includes preexisting scratch failures (old NUL source
  in resourcepack-d and mixed packages in tmp). The production package list and
  GUI entry point exclude these scratch files.
- Candle geometry has 136 color/count/light variants covered, plus candle cakes.
  A persisted byte/bool state issue was also fixed so lit candles keep emission.
  Actual WebGL and exported GLB previews were rendered and inspected.
- A subsequent completion audit reproduced ignored texture orientation in native
  cube surfaces. Added complete surface/UV rotations for 160 horizontal native
  pillar states, all 24 furnace family/direction combinations, and all 12 barrel
  facing/open combinations. Regression tests first failed and now pass. Actual
  mesh cap colors were verified on X/Y/Z; WebGL and loaded GLB galleries match
  (`tools/light-audit/state-surfaces-overview.png` and `state-surfaces-exported.png`).
  Independent review found no concrete regression. Full production Go packages
  and affected scene/worldstate/desktop packages passed after the change.

## Resumed completion audit

The earlier installation blocker was resolved externally. Read-only inspection
now confirms that both `builds/Craft-Box.exe` and
`D:\bedrock-tools\Craft-Box.exe` have SHA256
`5C561450F6570D83A6D1DFE043885F2D7D6F3AD23803539702CB99826226EBF6`.
No obsolete matching CraftBox/BedrockTool executables remain in either named
directory (previously 46 files / 3,103,607,296 bytes). No rejected operation was
retried through another tool. The installer syntax and embedded build hash match.

Added `utils/merge/palette_disk_test.go` for the final mapping/format gate:

- All 23,960 native states pass real disk chunk encoding and decoding through an
  independent merged reader registry at negative chunk Y, including light flags.
- Exactly two legacy red mushroom states (bits 10/15) are expected to normalize
  to `mushroom_stem`, matching explicit worldupgrader schemas 0311/0321. No other
  identifier/state changes are ignored by the test.
- A fresh reader retains custom states, vanilla extension properties and explicit
  fence connections, including the native light/NBT semantics of extensions.
- The embedded native palette version matches the actual disk-write version.

All merge tests pass. Independent review found no concrete test coverage issue.
Only test/documentation files changed in this resumed audit; no production Go,
bundled asset or frontend dist source is newer than the installed build.
The known format/asset/rendering limits above remain explicit.

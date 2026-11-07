# Render-editor selections

In the 2D map, draw a selection and open **Selected area**:

- **Open selection in 3D** loads geometry for that rectangle only. The scene
  request is filtered before meshing, with a separate mesh-cache scope. Full
  height is retained. **Show full world** returns to the currently loaded source.
- **Create separate map…** asks for a name, then creates a new Bedrock folder and
  `.mcworld` archive. It keeps only selected chunks in the current dimension,
  includes the source resources, moves the spawn into retained terrain and adds
  the copy to the maps list. The original imported document stays open and
  unchanged. The new map opens in 3D.

Copies normally go to `worlds/regions`. If that would place output inside the
source world, a sibling `CraftBox regions` directory is used. The confirmation
opens the actual output location. Cancellation removes incomplete output. Copies
of captures require a finished source; live capture regions remain available in
the Capture window. Geometry-only `.craftbox` projects support selected 3D and
model/project export, but creating a Bedrock world requires the original
`.mcworld` or world folder.

Model export includes the active bounds. Saving a project physically retains
only selected geometry and records `document.selection`; reopening that project
cannot restore chunks excluded from the saved file. Open the original source
for the complete world. The 3D memory budget remains enforced; a large selection
may still require a smaller rectangle. An initially hidden renderer defers
geometry loading until activated.

## Local contracts

`ScenePageRequest.bounds?: {minX,minZ,maxX,maxZ}` uses inclusive chunk coordinates.
Bounds must be ordered and lie within [-1875000,1874999]. Requests must reset the
cursor/sequence when switching bounds; the frontend replaces its renderer for
that transition. Bounds are intersected with any existing capture-region filter.
Packed project pages use the same bounds and report the selected total.

`ExportModel` accepts optional `bounds` with the same shape. Legacy callers
without bounds retain their full-source behavior.

Wails `CreateRenderRegion({worldId,name,bounds})` returns `{world,path}`, where
`world` is a RenderWorld and `path` is the saved `.mcworld`. The host chooses the
destination; caller paths are not accepted. `GetWorldExportProgress` and
`CancelWorldExport` cover copying, archiving and loading. Creation shares the
world-export lock and holds the imported source lifetime through completion.

Verification before requested test removal covers selected paging, bounded GLB,
project round trip, independent copy/source preservation, dropping other
dimensions, cancellation, and both frontend selection actions.

## Build and source cleanup (2026-11-02)

Version: 1.63.1-selection-maps-wails.20261102-dev.

- Before requested test removal: 376 frontend cases passed, zero failures.
- Runtime Go packages: 669 distinct passing case/subcase records across the broad
  run and serial network recheck. Three UDP binding failures in the parallel run
  passed in the serial recheck. Optional data-dependent cases were skipped.
- Additional passing checks: root build-script tests, legacy map/transport tests,
  full embedded-asset restoration, and both portable-render asset checks.
- Removed 156 Go test files, 110 frontend test files, and 2 test configuration /
  setup files, after verification. The removal manifest with SHA-256 digests is
  stored at tmp/verified-test-removal-manifest.json.
- Removed the frontend test script and six direct test-only dev dependencies;
  the existing lockfile was pruned without changing retained package versions.
- Normal Go, TypeScript/JavaScript, CSS, HTML and TypeScript-config comments were
  removed. Go lexical tokens were compared before/after comment removal.
  Compiler/cgo directives and required legal notices were preserved. Encrypted
  optional legacy sources and third-party dependencies were left intact.
- Visible UI text was reviewed and rewritten in plain English. Technical labels
  needed for settings and file formats remain. The copy inventory is saved under tmp.
- TypeScript, Vite production build and the Windows GUI Go build passed after
  cleanup. The executable retains the embedded vanilla assets and GUI subsystem
  2, so it has no companion console window.
- EXE: builds/resourcepack-fix/Craft-Box.exe (250842624 bytes).
- SHA-256: 6B1609BB562AAB7192376BA14AC29F843B02342125202DB54B2636A40334B2D3.

A supplementary attempt to delete the unused frontend export test-data directory
was rejected by automatic approval review because execution approval was disabled.
The test-data directory remains in source. Native visual review was not available.
The Go build emitted a non-fatal module stat-cache write warning outside the
workspace and exited successfully.
## Windows volume fix (2026-11-02)

The outer containment guard in worldexport.Copy used filepath.Rel on absolute
paths from different Windows volumes. A temporary import on C: and an output
on D: therefore failed before any data was copied. The guard now calls Rel only
when volume names match, case-insensitively. Different drives/shares are separate
locations; equal/nested destinations on the same volume remain rejected.

The exact C:-to-D: failure was reproduced with a temporary Windows regression
check. Both that check and same-folder/nested-folder protection passed after the
fix. The passing temporary test file was removed as requested. This verification
exercised the guard, not a full copy of the user's world across drives.

Build: 1.63.1-cross-volume-maps-wails.20261102-dev, 250843136 bytes,
GUI subsystem 2. Go production build exited successfully. Embedded frontend
and vanilla assets are retained. SHA-256: 2574AC7A6725C7A2DD2D6B3C78024697145F80C30E248F5D8A55DA9FE32F04CD.
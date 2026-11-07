# Portable assets and SoulSteel rendering

The executable previously depended on a separate `assets/vanilla-3d` directory.
Copying only the EXE or deleting that directory left its catalog without vanilla
textures. Server resource packs are overlays and do not replace that base.

A second independent problem affected large captured packs: the fixed 2048px
atlas could hold only 49 padded 256px model textures before returning its missing
tile for otherwise valid images. The atlas is now 4096px (64 MiB when allocated),
keeping existing geometry UVs stable. A regression loads 100 such materials.

## Portable packaging

`go run ./tools/bundle-assets` deterministically packages the complete locally
prepared vanilla tree into an ignored ZIP embedded by the `assets` package.
The standard GUI build runs this step and fails if the prepared source is absent.
Mojang runtime data and the generated ZIP are not committed to Git.

The GUI restores the complete tree beside the EXE in the background after opening
its window. Missing files are repaired on subsequent launches; a changed bundle
also updates same-size files. Atomic file replacement and a process lock cover
the entire extraction/marker transaction. `os.Root` confines nested paths.

Rendering reads the embedded base immediately and overlays local/captured packs.
Even a partially deleted, unwritable local tree cannot hide the embedded base.
The archive is opened directly from immutable embedded storage without retaining
an additional archive-sized byte copy in every GUI/map-window process.

## Onboarding

`GetBootstrap.installationId` is an opaque hash of the normalized executable
directory. The frontend scopes completion to that identifier. A new folder shows
the account guide again when signed out; updating/renaming the EXE in the same
folder preserves completion. Unscoped historical flags no longer suppress the
guide in a scoped installation. Signed-in users and map windows skip the guide.
The titlebar close button is hidden; explicit skip/link actions remain available.
The three onboarding component tests and frontend build were rerun after this
final presentation change.

## Validation

- Portable stone rendering with no external assets: red before fix, green after.
- Partial local tree (provenance present, images absent): red before fix, green.
- Concurrent different bundles: red without process lock, green with lock.
- All 18,234 embedded files restored and byte-verified, then a deleted PNG repaired.
- A local snapshot of the active SoulSteel DB and its resource packs was audited
  with 1,094 block definitions extracted from StartGame. The source LOCK file
  was not copied; standard Overworld metadata was supplied only for the audit.
  With embedded vanilla and 4K atlas: **3,593 chunks, 623 materials, zero unresolved
  terrain textures, zero approximate blocks, zero unresolved decorative visuals**.
  This is a saved-data audit, not a new live-server session.
- All affected Go packages, build-helper tests and opt-in portable tests passed.
- All 359 frontend tests passed with `--testTimeout=15000`; TypeScript/Vite passed.
- Independent review found the partial-tree and concurrent-update cases; both
  were corrected and independently re-reviewed without remaining must-fix issues.

Artifact: `builds/resourcepack-fix/Craft-Box.exe` (250,198,528 bytes),
`1.63.1-portable-assets-onboarding-wails.20261102-dev`, SHA256
`80DF57F0D45D4AB84CE97F2B0028040E160EE004A92C569347DBEDF97D239D78`.
Built with `gui,production` and Windows GUI subsystem; no console window.

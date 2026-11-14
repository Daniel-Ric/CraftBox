# Contributing to CraftBox

Contributions can be code, documentation, reproducible bug reports or feedback on an existing workflow. CraftBox is maintained by **SpindexGFX** and builds on [bedrocktool](https://github.com/bedrock-tool/bedrocktool).

Read the [README](Readme.md) for the application overview and the [Code of Conduct](CODE_OF_CONDUCT.md) before participating. Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## Before starting

Search existing issues and pull requests for related work. For a large feature, a new dependency or a change to capture behavior, exports or saved project formats, open an issue first to discuss the approach.

Keep each pull request focused. A bug fix should not also replace unrelated UI, reformat whole directories or update unrelated dependencies.

## Reporting a bug

Use [GitHub Issues](https://github.com/Daniel-Ric/CraftBox/issues) and include:

- The CraftBox version or commit, operating system and Minecraft version.
- The affected workspace or CLI command, including relevant options.
- Steps to reproduce the problem, what happened and what you expected.
- A relevant log excerpt or screenshot. For visual problems, include the selected style, resource packs and preview/export settings when relevant.
- Whether it happens with a local world, a replay or a live connection. If server-specific, include the source type and public server name where appropriate.

Small examples are easier to investigate than a complete capture. Do not attach Microsoft/Xbox tokens, resource-pack keys, account files or private recordings. Check [SECURITY.md](SECURITY.md) if the report involves exposure of sensitive data.

## Development setup

The backend uses Go and Wails. The desktop frontend uses React, TypeScript and Three.js.

The current build workflow uses Go **1.26.10**, Node.js **22** and Python **3.13**. Windows desktop development needs WebView2 Runtime. Linux desktop development needs a compiler, `pkg-config`, GTK 3 and WebKitGTK 4.1 development packages.

```sh
git clone https://github.com/Daniel-Ric/CraftBox.git
cd CraftBox
go mod download
```

Prepare the rendering assets before building the desktop:

```sh
python tools/prepare-ci-assets.py
```

The preparation script downloads pinned assets and checks their hashes. Generated catalogs and bundles are not source files to commit. Local asset preparation is described in the [3D viewer guide](docs/3d-viewer.md).

Build from the repository root:

```sh
go run ./build.go windows gui,cli
```

Use `linux` instead of `windows` for the Linux build. The build script regenerates `builds/`; move any files you need to keep out of that directory first.

For frontend development:

```sh
cd ui/desktop/frontend
npm ci
npm run dev
```

The browser preview does not provide native capture, authentication, file dialogs or filesystem access on its own. See [Desktop UI development](docs/wails-ui.md) for the backend bridge and platform-specific build instructions.

Optional extension sources are encrypted with git-crypt. Public contributions should work without access to those sources. Do not commit decrypted copies or change encrypted files as part of an unrelated contribution.

## Where changes belong

| Path | Purpose |
| --- | --- |
| `cmd/CraftBox/`, `subcommands/` | Application entry point and CLI commands. |
| `handlers/worlds/` | World capture, entities, recorded interfaces and world state. |
| `utils/proxy/` | Connections, packet forwarding, recordings and resource-pack transfer. |
| `internal/` | Scene generation, structures, asset libraries, inspection and export services. |
| `ui/desktop/` | Native desktop behavior and Go bridge methods. |
| `ui/desktop/frontend/src/` | React workspaces, editors and Three.js rendering. |
| `scripting/` | Capture-hook examples and public type definitions. |
| `docs/` | Workflow guides, contracts and implementation notes. |
| `third_party/` | Tracked dependency snapshots with local compatibility changes. |

The networking snapshots are tracked source, not Git submodules. Follow the [snapshot maintenance notes](docs/nethernet-migration.md) when changing them, and preserve upstream notices.

## Code and behavior

- Follow the surrounding Go and TypeScript style. Run `gofmt` on changed Go files and keep formatting changes limited to the work at hand.
- Keep the interface and project documentation in English. Do not add translation files or a new localization system.
- Use existing UI components, theme tokens, dialogs and task controls where they fit.
- Keep command options and defaults consistent between the CLI, desktop bridge and interface.
- Preserve saved worlds, project data, block states and supported source-model information. Document intentional format or behavior changes.
- Handle errors and cancellation explicitly. Do not report a failed or incomplete export as a successful save.
- Keep authentication, untrusted input parsing, archive extraction, packet intervention and native command execution changes narrow and reviewable.

## Checking a change

Run the checks relevant to the affected code and include the results in the pull request. For frontend changes, run these from `ui/desktop/frontend`:

```sh
npm test
npm run build
```

From the repository root, check the public application packages:

```sh
go test ./cmd/CraftBox ./handlers/... ./internal/... ./subcommands ./ui/desktop ./ui/messages ./utils/... ./locale
```

Also check the native desktop path for desktop changes:

```sh
go test -tags gui ./ui/desktop
```

On Linux, use `-tags gui,webkit2_41` with the native dependencies installed. An unrestricted `go test ./...` can discover encrypted optional sources and local experiments under `tmp/`; select the public packages explicitly when those sources are locked.

For behavior changes, use a meaningful regression case where practical. Rendering changes need a visual check in addition to compilation. Capture changes should distinguish local fixtures or replay checks from an actual live-server test. State what was tested, what was not and why.

Use small, synthetic or authorized fixtures. Do not commit generated builds, account data, captured private traffic or third-party packs without permission. Temporary diagnostic tools and output that are not part of the contribution should be removed before submitting.

## Commits and pull requests

Use short, descriptive commit messages with the prefixes already used in the repository: `fix:`, `feat:`, `docs:`, `perf:`, `refactor:`, `test:`, `build:`, `ci:` or `chore:`. Keep separate fixes and features in separate commits where they can be reviewed independently.

Examples:

```text
fix: preserve sign text when exporting world geometry
feat: add selection export to the world atlas
docs: explain editable Blockbench model exports
```

A pull request should explain the problem, resulting behavior and checks performed. Link related issues, note compatibility or data-format changes, and include screenshots for visible UI changes. List known gaps rather than implying that an untested platform or server was verified.

Update the relevant guide in `docs/` when a workflow changes. Update the README when the change affects setup or a main feature, and scripting definitions when public capture hooks change.

## Attribution and license

Contributions are made under the repository's [GPL-3.0 license](LICENSE). Preserve the original bedrocktool attribution and the notices attached to bundled dependencies. Identify adapted upstream work in the pull request so its source and authors remain clear.

Captured server assets, skins, worlds and packs retain their original ownership. A contribution to CraftBox does not grant permission to redistribute someone else's content.

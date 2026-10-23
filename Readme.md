# CraftBox - Minecraft Bedrock Edition

> Desktop studio and proxy toolkit for capturing, exploring and rendering Minecraft Bedrock worlds — with live maps, textured 3D export, cinematic camera sequences and portable projects. Built with Go, Wails, React and Three.js.

> [!IMPORTANT]
> **Project Disclaimer**
>
> CraftBox is an independent project. It has no official relationship with Mojang AB or Microsoft Corporation. Minecraft and other third-party names and assets belong to their respective owners.

---

## Credits & Attribution

This project is based on [bedrocktool](https://github.com/bedrock-tool/bedrocktool), originally developed by its respective contributors.

This is an independently maintained and modified version, featuring additional functionality, improvements, and ongoing development.

The original project is licensed under the **GNU General Public License v3.0 (GPL-3.0)**. This derivative project remains licensed under GPL-3.0.

See the [LICENSE](LICENSE) file for details.

This project is not officially affiliated with or endorsed by the original developers.

---

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Quickstart](#quickstart)
  - [Desktop](#desktop)
  - [Command line](#command-line)
- [Configuration (Flags & Environment)](#configuration-flags--environment)
  - [Proxy & Connections](#proxy--connections)
  - [World Capture](#world-capture)
  - [Environment](#environment)
- [Data Files & Local Storage](#data-files--local-storage)
- [Runtime & Architecture](#runtime--architecture)
- [Command-Line Usage](#command-line-usage)
  - [Commands Summary](#commands-summary)
  - [Usage Examples](#usage-examples)
- [Logging](#logging)
- [Caching & Replay](#caching--replay)
- [Directory Layout](#directory-layout)
- [Development & Utilities](#development--utilities)
- [Testing](#testing)
- [Performance Tuning](#performance-tuning)
- [Build from Source](#build-from-source)
  - [Requirements](#requirements)
  - [Windows builds](#windows-builds)
- [Compatibility & Limitations](#compatibility--limitations)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

CraftBox connects a Minecraft Bedrock client to an upstream server through a local proxy and records the data delivered during play. Captured terrain can be saved as a Bedrock world, inspected on a live map, or used as the starting point for a render or cinematic project.

The desktop workspace brings together source selection, account management, capture progress, world maps and creative editors. A shared Go backend powers the interface and command-line workflows. The desktop app, Go module and CLI use the CraftBox name.

---

## Key Features

- 🌍 World capture with chunks, entities, inventories and `.mcworld` export.
- 📦 Server resource-pack downloads and player-skin collection.
- 🔐 Account selection and device-code sign-in for authenticated workflows.
- 🧭 Sources for server addresses, Realms, featured servers and friends' worlds.
- 🗺️ Live maps with player follow, separate world visits and named regions.
- 🧊 Interactive 3D terrain and textured GLB model export.
- 🎨 Render editor with lighting, HDR skies, color grading and transparent PNGs.
- 🎬 Cinematic editing with keyframes, camera paths, shots and export controls.
- 💾 Portable `.craftbox` projects containing terrain and editor settings.
- 📡 Packet capture and replay for repeatable session inspection.
- 🧰 CLI commands for capture, world merging and rendering, plus JavaScript capture hooks.

---

## Quickstart

### Desktop

1. Download the appropriate build from [Releases](https://github.com/Daniel-Ric/CraftBox/releases). Check its release notes for the included features.
2. Launch the desktop application. On Windows, the desktop UI requires Microsoft Edge WebView2 Runtime.
3. Choose a source and workflow, such as world capture, resource packs or skins. Complete account sign-in when requested.
4. Start the session and connect Minecraft to the local proxy using the connection details shown by the app.
5. Explore the server to receive terrain. Use the live map to inspect progress and the result-folder controls to find saved files.

For creative work, open a captured map or an existing Bedrock world in the Render editor. Save a `.craftbox` project to resume work with its scene and editor settings later.

### Command line

These examples use the Windows CLI executable from the application directory:

```powershell
# List commands available in your build
.\CraftBox.exe help

# Capture a world through the local proxy
.\CraftBox.exe worlds -address=server.example:19132

# Download resource packs offered by a server
.\CraftBox.exe packs -address=server.example:19132

# List Realms accessible to the selected account
.\CraftBox.exe list-realms
```

Replace the example address with a compatible server. Minecraft must connect through the proxy for world capture to receive gameplay data.

---

## Configuration (Flags & Environment)

### Proxy & Connections

Capture settings are command options, also exposed by the desktop interface. Common proxy options include:

| Option | Default | Purpose |
| --- | --- | --- |
| `-address` | Selected or entered source | Upstream connection target. |
| `-listen` | `0.0.0.0:19132` | Local proxy listening address. |
| `-capture` | `true` | Record the session as a packet capture. |
| `-client-cache` | `true` | Enable the Minecraft client cache. |
| `-debug` | `false` | Enable additional diagnostic logging. |
| `-extra-debug` | `false` | Enable extra packet logging. |

### World Capture

| Option | Default | Description |
| --- | --- | --- |
| `-image` | `false` | Save a map image with the captured world. |
| `-save-entities` | `true` | Save captured entities. |
| `-save-custom-entities` | `false` | Record custom server entities and their models. |
| `-save-players` | `false` | Save other players as entities. |
| `-save-inventories` | `true` | Preserve captured inventories. |
| `-chunk-radius` | `0` | Requested maximum chunk radius; zero leaves the override unset. |
| `-script` | Unset | Path to a JavaScript capture script. |

Options belong to the commands that expose them; they are not a universal set of flags for every command.

### Environment

| Variable | Default | Description |
| --- | --- | --- |
| `BEDROCK_ENV` | `prod` | Authentication environment selected at startup. |
| `TOKEN_NAME` | Unset | Named account to load at startup. |

The desktop interface manages account selection directly. There is no required `.env` setup for the normal desktop workflow.

---

## Data Files & Local Storage

| Output | Description |
| --- | --- |
| `.mcworld` | Bedrock world archive for captured terrain and exported world data. |
| `.png` | Optional capture map or an image produced by the rendering tools. |
| `.glb` | Static 3D model with embedded textures. |
| `.craftbox` | Portable studio project containing terrain geometry, texture data and editor settings. |
| Packet captures | Recorded session data for replay and diagnostics. |

World visits are stored separately, including repeated visits to worlds with the same name. Desktop exports use names such as `worlds/<server>/<name>--<UUID>.mcworld`, with a matching PNG when map-image saving is enabled. Visits without received chunks do not produce a world archive.

A `.craftbox` file preserves a creative workspace; use `.mcworld` for a world that Minecraft can import. Desktop result-folder buttons open the relevant output location.

---

## Runtime & Architecture

```text
Minecraft client <----> Local proxy <----> Bedrock server
                            |
                            v
                    Shared Go handlers
                      /           \
                     v             v
             Captured worlds   Packs / skins / packets
                     |
                     v
              Live map & 3D scene
                     |
                     v
              CraftBox desktop
              /       |       \
             v        v        v
          Render   Cinematic  Studio project
          PNG/GLB  Sequence   .craftbox
```

**Highlights**

- **Go backend:** shared commands, authentication, proxy sessions, packet handling and world export.
- **Transport:** official go-nethernet with local Xbox signaling adapters.
- **Minecraft libraries:** tracked Gophertunnel and Dragonfly source snapshots under `third_party/`, including compatibility changes and upstream licenses.
- **Desktop shell:** Wails v2 with a React/TypeScript frontend and bundled assets.
- **Creative workspace:** Three.js rendering, a live terrain viewer, render controls and cinematic editing.

No Git submodules are required. Snapshot provenance is recorded in [third_party/upstream.json](third_party/upstream.json); maintenance details are in [NetherNet migration](docs/nethernet-migration.md).

---

## Command-Line Usage

```text
CraftBox <command> [options]
```

### Commands Summary

| Command | Purpose |
| --- | --- |
| `worlds` | Capture a world from a server. |
| `packs` | Download server resource packs. |
| `skins` | Download player skins. |
| `skins-proxy` | Download player skins while using the proxy. |
| `capture` | Record a packet capture. |
| `list-realms` | List accessible Realms. |
| `merge` | Combine saved worlds. |
| `render` | Render a saved world to PNG. |
| `render3d` | Export a saved world as a textured GLB model. |

Command availability follows the installed build. Run `help` for its registered command list. Desktop command forms read their options and defaults from the Go handlers.

### Usage Examples

Examples for a saved world folder:

```powershell
.\CraftBox.exe render -world="C:/Worlds/MyWorld" -out="C:/Exports/world.png"
.\CraftBox.exe render3d -world="C:/Worlds/MyWorld" -assets="assets/vanilla-3d" -out="C:/Exports/world.glb"
```

The 3D exporter needs a prepared asset catalog. See [Live 3D and GLB export](docs/3d-viewer.md) for asset preparation and the viewer workflow.

---

## Logging

Capture progress, connection state and errors appear in the desktop log view and CLI output. Use `-debug` on proxy commands for additional diagnostics; `-extra-debug` enables extra packet logging.

For a useful issue report, include the CraftBox build version, selected workflow, connection type, reproduction steps and the relevant log excerpt. Remove credentials before sharing logs or captures.

---

## Caching & Replay

Proxy workflows enable the Minecraft client cache by default. Resource-pack handling also reuses cached data, while packet captures allow supported workflows to replay previously recorded sessions.

A cache or capture contains only data received by the application. It cannot supply terrain, textures or server logic that was never delivered. For transport and replay coverage, see [NetherNet migration](docs/nethernet-migration.md).

---

## Directory Layout

```text
.
├── cmd/CraftBox/          Application entry point and UI selection
├── handlers/                World capture and packet handlers
├── subcommands/             CLI commands and shared option schemas
├── ui/
│   ├── cli/                 Terminal interface
│   ├── desktop/             Wails shell and desktop backend
│   │   └── frontend/        React, TypeScript and Three.js workspace
│   └── gui/                 Existing Gio interface
├── internal/                Internal application packages
├── utils/                   Authentication, proxy, packs and export helpers
├── third_party/             Tracked Minecraft library snapshots and licenses
├── tools/                   Asset preparation and verification utilities
├── assets/                  Application assets and icon provenance
├── docs/                    Workflow and development documentation
├── build.go                 Platform build orchestration
└── LICENSE                  GNU GPL v3.0
```

---

## Development & Utilities

The desktop command forms derive options from the Go command handlers. Change a command's schema in `subcommands/` when adding backend options; the frontend bridge and workspace live in `ui/desktop/`.

```powershell
# Install frontend dependencies and start the UI preview
cd ui/desktop/frontend
npm ci
npm run dev
```

The preview supports interface development. Authentication, capture and native file dialogs require the compiled application. Use [Desktop UI development](docs/wails-ui.md) for backend integration and [3D asset preparation](docs/3d-viewer.md) for the model catalog.

Official Gophertunnel and Dragonfly snapshots contain local compatibility changes. Follow the [snapshot maintenance workflow](docs/nethernet-migration.md) when updating them and preserve their license notices.

---

## Testing

From the repository root:

```powershell
cd ui/desktop/frontend
npm ci
npm test
npm run build
cd ../../..
go test ./ui/desktop ./utils/auth ./utils/proxy ./handlers/worlds
```

`npm run dev` provides a frontend preview. Native sign-in, capture, file dialogs and other desktop operations require the compiled application and its Go bridge.

Select Go test packages explicitly when the private extension remains locked: `go test ./...` also discovers its encrypted source directory. Frontend tests and builds validate the interface; testing a real server connection additionally requires Minecraft and a compatible endpoint.

---

## Performance Tuning

- Select a named region when only part of a captured world is needed for a model or composition.
- Use the Render editor's faster quality modes while composing, then increase quality for final output.
- Choose export dimensions deliberately; larger images increase rendering and memory costs.
- Enable custom-entity and player recording only when the capture workflow needs them.
- Keep extra packet logging for diagnostics, since it increases the amount of output produced during play.

The creative editors expose quality, shadow and output settings independently. See [Render and cinematic editors](docs/render-editor.md) for their behavior.

---

## Build from Source

### Requirements

- Go **1.26**, as declared in `go.mod` and `go.work`.
- Git.
- Node.js **22** and npm for desktop frontend builds.
- Microsoft Edge WebView2 Runtime on Windows; GTK 3 and WebKitGTK 4.1 development packages for Linux desktop builds.

### Windows builds

```powershell
git clone https://github.com/Daniel-Ric/CraftBox.git
cd CraftBox
go run ./build.go windows gui,cli
```

The script installs frontend dependencies, builds desktop assets and writes application builds to `builds/`. **It regenerates that directory**, so move files you want to keep elsewhere before running it.

For a CLI-only build:

```powershell
go run ./build.go windows cli
```

See [Desktop UI development](docs/wails-ui.md) for direct Go build commands and platform details. Optional private extension sources are encrypted with git-crypt; public development does not require unlocking them.

---

## Compatibility & Limitations

- This development branch uses **NetherNet**. Both the Minecraft client and upstream server must support it; the former RakNet transport is unavailable.
- The documented transport target is Minecraft protocol **2193**. Consult the selected release and [connection documentation](docs/nethernet-migration.md) for compatibility details.
- Capture records data delivered to the client. Unvisited terrain and server-side scripts or behavior that the server never sends cannot be reconstructed.
- Resource and custom-block fidelity depends on assets supplied during capture. See [World pack export](docs/world-pack-export.md) for current export behavior and asset requirements.
- Studio and rendering features described here reflect this source tree. Older release binaries may provide a smaller feature set.

---

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Minecraft cannot join the proxy | Confirm the displayed address and port, firewall rules, and NetherNet support on both ends. |
| LAN discovery is unavailable | Check whether another application occupies UDP 7551; try the direct connection details. |
| No world file appears | Join through the proxy and move through the world so the session receives chunks. |
| A saved world lacks expected assets | Review pack errors and the [export notes](docs/world-pack-export.md); missing server files require a new capture. |
| Frontend preview cannot start a capture | Use the compiled app for operations that need the native bridge. |
| A broad Go test run fails on private source | Run the relevant public packages explicitly as shown above. |

For crashes or reproducible problems, [open an issue](https://github.com/Daniel-Ric/CraftBox/issues) with the application version, operating system, steps to reproduce, expected result and relevant log excerpt. Add a screenshot for visual issues; remove account tokens and other credentials from diagnostics.

---

## FAQ

### Is CraftBox the original bedrocktool project?

CraftBox is an independently maintained derivative. The original project and license are acknowledged in [Credits & Attribution](#credits--attribution). The original project is credited above.

### Does capturing a server download its entire world?

Capture saves the data sent to the client during the session. Explore the areas you want to retain; unvisited chunks and private server logic are unavailable.

### Which format should I save?

Use `.mcworld` to import a captured world into Minecraft, `.glb` for a static textured model, and `.craftbox` to continue a studio project with its scene and editor settings.

### Can I use CraftBox entirely in a browser?

The frontend can be previewed in a browser for development. Capture, authentication and native file operations use the desktop backend.

### Which executable should I use?

Use the CraftBox desktop build for the studio interface and the CraftBox CLI build for terminal workflows.

---

## Documentation

| Guide | Topics |
| --- | --- |
| [Desktop UI development](docs/wails-ui.md) | Native UI, build setup, command integration and world visits. |
| [NetherNet migration](docs/nethernet-migration.md) | Connections, snapshots and transport compatibility. |
| [World pack export](docs/world-pack-export.md) | Resource packs, custom blocks and captured entities. |
| [World region editor](docs/world-region-editor.md) | Region selection in the map window. |
| [Live 3D and GLB export](docs/3d-viewer.md) | Viewer controls, asset preparation and model export. |
| [Render and cinematic editors](docs/render-editor.md) | Lighting, skies, image output and cinematic tools. |
| [Portable studio projects](docs/studio-projects.md) | `.craftbox` files, saved state and map presentation. |

---

## Contributing

Use [GitHub Issues](https://github.com/Daniel-Ric/CraftBox/issues) to report bugs or discuss improvements. For a pull request, describe the affected workflow, resulting behavior and checks you ran. Include screenshots for interface changes and preserve upstream attribution and licenses when modifying bundled dependencies.

---

## License

Distributed under the **GNU General Public License v3.0 (GPL-3.0)**. See [LICENSE](LICENSE) and the [Credits & Attribution](#credits--attribution) above. Bundled third-party components retain their respective license notices.

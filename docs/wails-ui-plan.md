# Wails UI redesign

## Product and visual direction

CraftBox is a desktop workspace for downloading Minecraft Bedrock worlds,
resource packs and player skins. The interface prioritises choosing a source,
understanding what will be saved, and watching an actual capture. No decorative
dashboard metrics or fictitious servers/downloads are shown.

The distinguishing element is the live world map: a precise, pixel-sharp canvas
with player position, zoom/pan controls and a quiet coordinate grid. Everything
else supports this work area.

The visible product name is **CraftBox**. Palette follows Checkmk's official
modern-dark and facelift themes: `#1c2228` (dark canvas), `#20272e` (panels),
`#11181d` (sidebar), `#15d1a0` (corporate green), `#fafaff` (light canvas),
`#f5f5fb` (light panels). Light-mode text uses conference grey `#1e262e` and
the darker corporate green `#0c7358` for readable accents. Menu rows span the
whole sidebar, with a green left border on the active item. Corners use 2px;
status dots remain circular. Source colour values:
https://github.com/Checkmk/checkmk/blob/master/packages/cmk-frontend/src/themes/facelift/colors.scss
Use locally bundled Manrope for interface text and system monospace only for
addresses/log lines. Keep sentence-case labels and visible keyboard focus.

## Layout

Desktop window: initially 1240 x 820, resizable. Navigation is 208 px wide;
content fills the remaining space. At narrower widths, collapse navigation to
icons and move the summary underneath the form. Content is left-aligned.

```
+------------------+----------------------------------------------------+
| CraftBox      | Worlds                         Session status      |
|                  +--------------------------------+-------------------+
| Worlds           | Source                         | Download summary  |
| Resource packs   | Address / Realms / Events /    | Selected source   |
| Skins            | Featured / Friends' worlds     | Save location     |
|                  |                                | Key options       |
|                  | Save options                   |                   |
|                  | Advanced options (expand)      | Start download    |
|                  +--------------------------------+-------------------+
| Account          | Live session: map / pack progress / player skins   |
| Settings         | Results: actual saved files and open-folder action |
+------------------+----------------------------------------------------+
| Logs drawer: searchable, severity filter, copy / clear                 |
+-----------------------------------------------------------------------+
```

The same main workspace supports all three existing GUI commands. Changing
sections must not discard configuration or hide a running session. Sessions
have distinct preparing/listening/connecting/downloading/saving/finished/error
states. Starting is validated; stopping cancels the Go context and waits for
existing save/cleanup behaviour. Connection instructions include a button to
open Minecraft and the actual local listener address.

## Information placement

- Source selector: one address field with a chooser dialog for Realms, events,
  featured servers and Xbox-hosted worlds. Loading, empty, auth-required and
  failure states are explicit and retryable. Preserve replay address support.
- Save options: capture/cache, entities/players/inventories, void generation and
  image output use clear labels with explanations. All existing command options
  remain available; technical flags are grouped under advanced options.
- Advanced options: listener address, excluded mobs, chunk radius, scripts,
  block updates, entity culling and debug flags. File inputs use native pickers.
- Session area: live map with zoom, pan, follow-player and reset-view controls;
  pack progress and icons; skin names and image previews; saved world names,
  chunk/entity counts and paths. These use real backend events.
- Account menu: Microsoft device-code login, copy code/open browser, cancel,
  logout and named account-instance switching. No token is sent to React.
- Settings: dark/light/system theme, version/protocol information and updater
  with explicit check/download/install states and progress.
- Logs: persistent collapsible drawer, readable severity/time/source, search,
  clipboard and bounded in-memory history. Error dialogs offer useful actions.

## Components and backend contract

Use Wails v2 stable, React/TypeScript/Vite and real shadcn/ui components with
Tailwind. Buttons, tabs, dialogs, switches, inputs, badges and progress share
one token system; icons use Lucide. UI assets and fonts are embedded for offline
startup. Android retains its existing native UI because Wails v2 targets desktop.

The Go bridge reuses the existing command registry, settings parser, auth,
source services, updater and UI event types. It exposes typed bootstrap state,
command schemas, start/stop, source listing, account actions, native file pickers,
safe URL opening, result-folder actions, and updater actions. State is buffered
before React subscribes. Events include connection phases, world-save progress,
map tiles, player position, packs, skins, device codes, logs and errors. Do not
reimplement NetherNet, capture or world-export internals in JavaScript.

## Delivery and verification

1. Inventory existing functions and finalise state/bridge contract.
2. Test settings translation, cancellation and event conversion before migration.
3. Implement bridge and desktop lifecycle without altering CLI behaviour.
4. Build the React application, complete every screen and integrate real events.
5. Test frontend interactions, dark/light/narrow layouts and native Windows launch.
6. Verify CLI, desktop builds and the existing NetherNet/capture/export tests.
7. Review feature parity; supply a runnable Windows EXE and report limitations.

Design review: avoid an interchangeable grid of cards. The source form and
session canvas form one continuous workspace, with the summary as a subordinate
rail. Restrict accent colour to actions and current state. Use motion only for
drawers, dialogs and progress changes; respect reduced-motion settings.

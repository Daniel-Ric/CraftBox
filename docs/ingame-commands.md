# Ingame capture commands

These commands are registered during a world capture. CraftBox consumes them locally rather than forwarding them to the game server. Other server commands, including `/help`, continue to reach the server. Use `/cb` or `/cb help` for CraftBox's own paginated help.

`/cb menu` opens an icon-free native Bedrock GUI. Category buttons use pink, blue, purple and yellow accents with dark secondary text for light button backgrounds. Subpages provide toggles, a stepped zoom slider, dropdowns, named text inputs, coordinate fields, paginated exclusions/help, and confirmation dialogs for saving or clearing exclusions. Closing a form cancels it. Menu submissions are bound to the active capture; a dimension/capture switch invalidates older forms. Server forms use separate client-visible IDs and their original IDs are restored upstream.

The live position dot now uses `/cb position [on|off|toggle]`. `/cb map` only controls map visibility; the former `map marker` / `map position` overloads are removed. Named markers still use `/cb marker <icon> <name>`.

The menu dashboard shows the current capture, recorded chunk count, player position and map state. Quick Actions provides common recording/map controls. Points have a read-only browser with case-insensitive name search, type filtering, newest-first order and detail pages. Saved blueprints have a searchable, alphabetical library with dimensions and a desktop-open action; invalid projects are skipped and counted. The server entity catalog supports search and exact exclusion management. Blueprint and region pages show live selection dimensions, and created regions have detail pages. Browsers use six entries per page. Refresh actions update status in place, input retries retain entered text, and successful actions return to their parent section. Selection resets and save cancellation require confirmation. The GUI remains icon-free.

| Command | Action |
| --- | --- |
| `/cb menu` | Open the native Bedrock menu with Capture, Map, Points/Notes, Blueprints, Regions, Entity Exclusions, Connection and Help pages. |
| `/cb help [1|2|3]` or `/cbhelp [1|2|3]` | Show help and usage examples. |
| `/setname <world name>` or `/cb name <world name>` | Rename the active capture. Spaces and UTF-8 names are supported; quotes are unnecessary. |
| `/void [on|off|toggle]` or `/cb void [on|off|toggle]` | With no argument, toggle the void generator. Explicit states are supported. The setting also applies to the next world created in this connection. |
| `/save-world` or `/cb save` | Save the captured chunks and begin a fresh capture without disconnecting. Empty captures are rejected. |
| `/cb status` | Show the world name, recorded chunk count, void setting and local map state. |
| `/cb connection` | Show session state, source, available endpoints, transport, Minecraft version/protocol and measured RTT. No credentials or URL query strings are displayed. |
| `/cb position [on|off|toggle]` | Show/hide a live current-position dot on the ingame CraftBox map. Defaults to off. Saved map images stay unchanged. |
| `/cb version` | Show the running CraftBox version. |
| `/cb map [on|off|toggle]` | Show or hide the client-side map. With no argument, toggle it. Hiding it restores the latest server-side offhand item. |
| `/cb zoom [1|2|4|8|16]` | Show or set map zoom. Larger values give a closer view. Arm swings still cycle zoom. |
| `/exclude-mob <id/pattern> ...` or `/cb exclude ...` | Exclude entity types from the saved world. `zombie` expands to `minecraft:zombie`. Glob patterns such as `minecraft:*` are supported. |
| `/include-mob <pattern|all> ...` or `/cb include ...` | Remove exact configured exclusion patterns, or clear all exclusions. |
| `/cb exclusions [page]` | List configured entity exclusions, eight per page. |
| `/cb marker <icon> <name>` | Place a named 2D marker at the exact player X/Z position; names may contain spaces. |
| `/cb mark <name>` | Save a position, timestamp and elapsed session time with the default pin. |
| `/cb note <text>` | Save a positional note and timestamp. |
| `/cb region pos1` / `pos2` / `clear` | Select/reset the current world's chunk-aligned region corners. |
| `/cb region create <name>` | Create an additional named region copy from the selected corners. |
| `/cb camera add <name>` | Save the player's position, pitch and yaw for Cinematic keyframe import. |
| `/cb capture status` | Show terrain/entity/inventory/image configuration, packet archive configuration and attached modules. |

Marker icons use hard-enum autocomplete: `pin`, `home`, `sword`, `star`, `flag`, `shop`, `camera`, `portal`, `warning`, `chest`. Example: `/cb marker home Meine Basis`. Labels accept 1–100 UTF-8 code points, reject blank/formatting/control input, and preserve spaces. All three coordinates retain the received float32 precision; the atlas uses original X/Z directly, without rounding to blocks or chunk centers. Region corners deliberately use floor(X/16), including negative positions. A corner selection is scoped to its world visit and cannot be reused after transfer, dimension change or save/reset.

Points persist in the owned world's `craftbox-capture.json`, are retained through first database creation/rename, and travel inside `.mcworld`. Reopening a captured archive loads them into Render/Cinematic. Region copies retain only points within their chunk bounds. UI marker edits/deletions remain project/map presentation changes; recorded evidence remains unchanged, and imported-ID tracking prevents deleted markers from reappearing in that presentation. A capture holds at most 512 points. An invalid existing notes file is preserved and blocks further writes rather than being overwritten.

When the Packet Inspector is recording, the command moment bookmarks the preceding recorded packet through the recorder's ordered queue, without creating a synthetic packet. Cinematic offers saved camera import and explicit placement of a captured note/marker at the current playhead; its original timestamp, session offset and coordinates remain in the marker comment. Importing a camera converts the recorded world position and view direction to the selected scene's coordinate frame.

Names must contain 1–80 Unicode code points. Blank names, control characters, Minecraft formatting markers, filesystem-reserved characters and reserved Windows names are rejected before the rename. Existing names remain unchanged after a failed command. Commands validate argument counts, help pages, toggle states, zoom levels and exclusion pattern syntax before applying changes. Invalid exclusion batches do not partially apply. Exclusions use a synchronized copy for saving, so later changes cannot mutate a save's array.

Chat messages use a reset-delimited `§8[§aCraftBox§8]` prefix, Minecraft color formatting, UTF-8 text, and usage hints on errors. The existing English/German rename, void and startup messages were updated as well.

## Autocomplete and entity validation

CraftBox sends structured command overloads through `AvailableCommands`: `/cb` subcommands, help pages, `on/off/toggle`, supported zoom levels, required world-name placeholders, and up to 32 entity-pattern arguments. Fixed values use hard enums; entity IDs and current exclusion rules use soft enums so intentional wildcard patterns remain enterable. Existing server commands and enum offsets are preserved. Each session uses distinct enum names.

Entity suggestions are learned from the server's actor catalog and clientbound actor/player packets, including custom actors when custom-entity capture is disabled. Exact exclusion IDs must be known to this connection; an unknown ID rejects the entire batch. Before any entity catalog arrives, the command asks the user to wait rather than accepting an invented ID. Wildcards remain deliberate pattern rules. Include suggestions reflect configured exclusions and `all`; `all` must be used alone.

Soft-enum updates are activated only after the definitions containing CraftBox's enum names were successfully written to the client. Updates then publish fresh snapshots under a dedicated send-order mutex, preventing a newly sent definition from leaving stale suggestions after concurrent exclusion changes. Actor IDs supplied by the client are not learned.

Three native regressions validate actual `AvailableCommands` wire encode/decode with preexisting server tables, enum indices and contextual/repeated argument definitions, live entity/include updates, unknown-ID and mixed-`all` batch rejection, custom actor observation, clientbound trust, and definition/update ordering including refreshes. The original invalid-ID acceptance and premature-update failures were reproduced. Evidence remains under `tmp/autocomplete-*`; successful temporary test sources are removed before release. A scoped review's ordering finding was corrected. No live-client UI comparison was performed.

## Default capture option values

Packet recording, client cache, void generation, map PNG export and inventory capture default to on. Entity capture and custom entity capture default to off. The native argument metadata supplies these values for new configurations and omitted settings; saved projects and explicitly supplied choices are not forcibly overwritten.

## Water map correction

Water depth shading previously subtracted directly from unsigned 8-bit RGB channels. Deep-water values could wrap around to red or yellow. Shading now uses bounded integer arithmetic with a capped depth shade, and each water sample uses its own overlay value instead of mutating a shared color.

## Verification

Seven native tests cover blank/invalid/UTF-8 names, void toggling and explicit values, help/status/version, map/zoom validation, offhand restoration through the real packet handler, empty-save rejection, exclusion batch atomicity and save snapshot independence, safe local command parsing and forwarding, all seven defaults, and water colors at depths 1, 4, 8, 16, 32 and 64. The original blank-name, deep-water, stale-offhand and shared-array failures were reproduced before correction. Tests use temporary directories and a recording client connection; they do not connect to a live Minecraft server. Logs remain under `tmp/ingame-*`, `tmp/water-*`, and `tmp/capture-defaults-*`. Successful temporary test sources are removed before building the EXE, with hashes retained. A scoped code review was completed and its exclusion-snapshot finding corrected.

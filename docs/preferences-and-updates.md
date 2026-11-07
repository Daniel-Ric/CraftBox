# Preferences and GitHub updates

Preferences follow a searchable category layout inspired by Adobe's categorized preferences and JetBrains' searchable settings. Values are validated before persistence in `craftbox-workspace-preferences`; older three-field preferences are migrated automatically. The six categories contain 28 controls, plus the existing theme chooser. Each category can restore its defaults.

## Scope and consumers

- Appearance: density, interface scale, reduced motion, tooltips and tooltip delay apply to the workspace.
- Workspace: welcome dialog, initial/remembered workspace, recent-project retention and title-bar dialog movement.
- 3D editor and export: preview quality, image dimensions, aspect ratio and transparency initialize new projects. Saved documents retain their own values. Geometry memory applies to subsequently loaded browser scenes; backend import limits still apply.
- Navigation: speed, field of view, sensitivity, smoothing and level movement initialize editor camera controls.
- Console: font size, wrapping, timestamps, line numbers and following initialize new console tabs. Existing tabs retain their configuration.
- Updates: automatic checks, stable/preview channel and 6-hour/daily/weekly interval.

Changing recent retention takes effect when the recent list is next normalized. Pinned projects sort before other entries and count toward the limit. Camera defaults apply when an editor is next opened. Preference save failures are shown in the dialog rather than claiming success.

## Experimental destination hostname

Preferences > Connection contains five saved settings: Destination hostname (off), automatic Minecraft opening (on), automatic 2D capture-map opening (on), LAN visibility (on), and the local port (19132). Launch choices are snapshotted when a capture starts; changing them applies to the next capture. Ports must be whole numbers from 1 to 65535. The preferred port replaces the default listen address; an explicitly customized advanced listen address takes priority. Destination hostname disables LAN advertising while retaining the user's saved LAN preference for when hostname mode is switched off.

Destination hostname is passed to new capture commands as `hostname-alias`; LAN visibility uses `lan-discovery`. The normal local address remains 127.0.0.1 when hostname mode is off. CLI defaults also retain LAN advertising.

For direct server sources on Windows with a local IPv4/dual-stack listener and a local Minecraft client, CraftBox temporarily maps the **actual destination hostname**, without an application suffix, to 127.0.0.1. This is a systemwide Windows hosts redirect while the capture is active. A separate helper requests UAC permission; the main application stays unelevated. Cancelling UAC produces an actionable error. Bare IP sources, Realms and online discovery sources remain unsupported by this experimental mode. Captures without a local Minecraft client and offline replays use their normal routes.

Before redirecting, the helper removes stale tagged entries and resolves the destination's original non-loopback IPs. CraftBox pins outgoing NetherNet signaling, server pings, authentication HTTP and URL resource-pack downloads to that snapshot for this hostname. The original URL Host and HTTPS certificate/SNI checks are retained. Other destination hostnames use normal resolution. This prevents an outgoing connection from circling back into the local proxy.

The signaling endpoint is `http://<destination-hostname>:<local-port>`, but the Minecraft Connect deep link passes **only the hostname/IP in `serverUrl` and the numeric port in `serverPort`**, without HTTP/HTTPS prefixes or duplicated ports. A ready-listener flag prevents launching during setup, offline replay, non-proxy captures or shutdown. A capture ID limits automatic opening to one attempt in the main window per capture and avoids duplicate launches during internal server transfers. Manual Open Minecraft uses the same validated link builder. LAN discovery is disabled for hostname mode so its physical-IP advertisement cannot bypass the hostname link. Capture controls expose the signaling endpoint with Open Minecraft and Copy actions. A flyout reports the actual Login ServerAddress. Saved localhost entries retain their original address.

The initial hostname is retained across server transfers. The helper removes the override on stop or parent control-socket EOF, including a main-process crash. A systemwide Windows ownership handle prevents simultaneous helpers from replacing each other's mappings. Only tagged entries are managed; unrelated hosts entries and concurrent changes are retained. Windows ReplaceFileW preserves ACLs and uses a recovery backup. If both processes are interrupted, stale tagged entries are removed before resolving the next enabled session. The helper command is handled before account/UI/log initialization, uses a nonce-authenticated loopback control connection and accepts no arbitrary hosts-file path or shell commands.

The transport prototype confirms the real hostname in the Go Minecraft client's login and delivery through a local NetherNet relay. Additional tests cover permission cancellation, hosts-byte restoration, TLS SNI/Host, loop prevention, helper EOF/stop cleanup, same-host pack downloads, validated launch parameters and listener readiness. The user subsequently confirmed the destination hostname in the actual Flarial HUD when manually joining the hostname entry. The new bare-hostname Connect-link variant still needs live client confirmation; local tests validate its contents and lifecycle guards. Signaling remains HTTP; CraftBox does not install a trusted CA or disable TLS checks. Test artifacts are under `tmp/hostname-prototype`; real Windows hosts content was not changed during automated verification.

Win32 references: [global object namespaces](https://learn.microsoft.com/en-us/windows/win32/termserv/kernel-object-namespaces), [ReplaceFileW and preserved file metadata](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-replacefilew).

## GitHub integration contract

Repository: `https://github.com/Daniel-Ric/CraftBox`.

`CheckUpdateDetails(channel)` accepts `stable` or `preview` and returns currentVersion, channel, checkedAt, available and optional release metadata. Stable uses `/repos/Daniel-Ric/CraftBox/releases/latest`; preview uses `/repos/Daniel-Ric/CraftBox/releases?per_page=100` and selects the highest valid published version. Drafts are excluded. The provider contract is [GitHub's Releases REST API](https://docs.github.com/en/rest/releases/releases).

Requests use GitHub's JSON Accept header, API version and a CraftBox User-Agent, with a 15-second context deadline and bounded response size. Manual checks always refresh. Automatic checks respect the persisted interval; failures retry no sooner than 15 minutes later. Separate map windows do not make automatic requests. Switching channels invalidates old responses in both frontend and backend.

Metadata includes release title, version, notes, publication date, preview status, GitHub URL, asset name/size and installation availability. Notes render as plain text, without executing release HTML. Missing releases, missing compatible assets, missing checksums, network failures and GitHub rate limits have distinct states.

Installer assets must match `CraftBox[-gui]-<os>-<arch>-*`, have a valid 32-byte SHA-256 digest and use an HTTPS download URL within this repository. The downloader verifies bytes before applying an update; older releases are never installed. A local development build is not replaced by its same-version base release. Installation requires the separately selected verified release, an explicit user action, and an ended capture session. Installation then restarts CraftBox.

## Verification

Updater tests exercise version ordering, refreshed metadata, preview selection, rate limits and the actual HTTP contract against a local server. UI tests cover validation/migration, search, category reset, persistence, new-project defaults, consumer initialization, automatic/manual checks and channel switches during a pending request. Browser verification: `node tools/verify-preferences.mjs` with Vite running on 127.0.0.1:5204.

Live repository access may be unavailable in restricted development environments; local contract tests do not prove that a particular GitHub release currently has compatible published assets.

## Action flyouts

`StudioFlyouts` is mounted once in the desktop root, using the existing Sonner dependency. Cards inherit the current theme's popover, foreground, border and status tokens. Success, information, warning and error messages have distinct symbols and restrained color accents. Success/information messages disappear after 5.5 seconds, actionable cards after 9 seconds, warnings after 10 seconds; errors remain until dismissed. Hover/focus pauses Sonner's timer. Reduced-motion preferences disable transitions.

Exports and saves provide the destination and an Open folder action. Render images, panoramas, models, maps, camera sequences, scene styles, skins, console/terminal exports and clipboard actions use flyouts instead of transient inline banners. Successful world captures and resource-pack transfers notify on state transitions without replaying historical downloads. New GitHub releases provide a View release action. Known failure paths use explicit errors; recurring recovery-save errors use stable IDs to avoid flooding the stack. Field validation and ongoing progress remain next to their controls.

Browser verification: `node tools/verify-flyouts.mjs` exercises folder actions, modal coexistence, distinct tones and dark/light theme colors. `node tools/verify-preferences-zoom.mjs` checks the preferences at 125% interface scaling in a 760×600 viewport.

## Notification history

A bell at the right of the Console header opens the notification history, including while the console is collapsed and across workspace views. Every `notify` call records its value and action in a shared, in-memory store. The most recent 100 entries are retained for the current window/session; details and callback functions are not persisted to disk. Identical notifications with the same stable ID are coalesced with a repeat count; changes of content or severity remain separate entries.

The panel provides unread counts, search, severity/unread filters, full expandable details, Copy, Show again, original actions, per-entry removal, mark-all-read and clear-history controls. Replaying a notification does not append another history entry. Failed historical actions produce a new error notification. Viewing details or invoking an action marks that entry read; merely opening the panel does not consume unread entries. The popover supports Escape/outside dismissal and keyboard navigation. Its placement follows the bell and accounts for workspace zoom and small windows.

Model verification covers immutable snapshots, subscriptions, retention, deduplication and read/delete/clear behavior. Browser verification confirms expired flyouts remain accessible, actions still run, filters/unread work, replay does not duplicate, and the popup fits dark/light themes and 760×600 at 125% zoom. Reports and screenshots are in `tmp/hostname-prototype/notification-*` and `notifications-*`.
## Export progress

The console/status strip also opens the export task list. Each export uses a compact centered dialog that can be sent to the background and reopened. Known work counts display measured progress; other operations use an indeterminate indicator. Phase, elapsed time, supported cancellation and recent success/error/cancellation states remain available. Model/world progress and cancellation use matching native operation IDs across windows. See `editor-working-preview.md` for quality and task semantics.

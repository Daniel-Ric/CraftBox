# Unified workspaces

World overview is the visual reference. Other workspaces opt into `.workbench-view` on the desktop root. The shared stylesheet uses existing theme variables, so graphite, dark, gray and light retain the same palette as Worlds. It does not introduce another theme.

The frame uses a 39px module heading, 42px tool rows, 28px fields, 12px interface text, 11px secondary text, 14px content insets and 8px control gaps. Library and properties rails use consistent widths (250px / 310px on wide windows), with narrower rails or inspectable overlays at smaller sizes. Selection, focus, separators and tabs reuse existing primary, selected, ring and border tokens.

| Workspace | Interaction and hierarchy |
| --- | --- |
| Resource packs | Library and search on the left, content tools and preview in the middle, metadata and preview/export settings on the right. The properties panel can be opened on narrow windows. |
| Player skins | Same library frame, stable head thumbnails, model preview and pose/cosmetics controls. The character rig retains its specialized large editor. |
| Packet Inspector | Recordings/types on the left, packet table and analysis tabs in the center, packet details on the right. Dense packet data and semantic direction colors remain intact. |
| Marketplace | Discovery sources on the left, browsing/comparison in the center, current offer in the inspector. Search, sort and content type remain visible; additional labeled filters expand in place with a visible active-filter count and a clear action. |
| Render editor | Compact module and source bars, maximum canvas area, scene properties on the right. Existing scene tools, exports and map/render modes are retained. |
| Cinematic editor | Same module/source bars and properties styling, with camera/sequence/lighting modes and the existing timeline and playback controls. |

Marketplace cards put the item title first, then creator, price and available rating. Offer art keeps its aspect ratio without cropping. Favorite and comparison actions are separate from the card's selection button, with explicit pressed states. The list alternative offers a denser layout. Unknown prices remain unknown, never free. Discovery servers and experiences use images marked `Banner` by either Type or Tag, falling back to their existing preview if unavailable. Creator collection imagery remains beside creator names.

Dialogs use the existing CraftBox frame: 44px titlebar, 20px horizontal body insets, internal content scrolling and a separate action footer. Message dialogs retain their warning/error/confirmation icon layout. Export dialogs now follow the current theme, including light mode. Packet settings/details/export/send have a dedicated body rather than controls against the frame edge. The rig editor retains its three-part working area with the common header and spacing. Specialized source/preferences/help grids remain intact.

Verification artifacts are under `tmp/unified-*`: before/after screenshots, measured heading geometry, narrow-window captures and interaction checks. Tests use temporary local UI fixtures rather than changing user data or contacting Marketplace. These checks verify layout and interaction; they do not revalidate the already working PlayFab integration or 3D geometry.

## Follow-up refinements

All library empty states, including Worlds in both list and grid layouts, center in the available content area. Working surfaces share the editors' `--studio-canvas` theme token. An unavailable or audio-only pack preview uses a solid surface; checkerboards remain for actual transparent image previews. Packs without previewable objects omit irrelevant object filters and model export controls, retain direct access to Files, and collapse metadata editing. A search yielding zero matches in a nonempty catalog keeps its filters available.

Discovery source rows place the natural-ratio Banner above the name, with `Creator - Featured server/Experience` below. The Properties rail for Worlds draws existing map tiles as a terrain preview and offers Delete capture using the existing confirmation, cancellation, busy state, and backend deletion checks. No arbitrary filesystem image endpoint was added.

Follow-up verification: 33 empty-state/theme/layout checks, 14 resource-pack and source-banner interaction checks, and 8 terrain-preview/main-window delete checks in Chromium. Evidence is recorded in `tmp/empty-centered-verification.json`, `tmp/content-layout-verification.json`, and `tmp/world-properties-verification.json`.

## Collection layouts

Worlds, resource packs, skins and Marketplace offers share `collection-layouts.css`. Scrollable grids use intrinsic row heights instead of compressing cards into the viewport. Names wrap; metadata and favorite/comparison controls stay inside their card. Asset browsers offer two readable columns on wide windows and adapt to their available width. Skin heads use square media; packs/worlds use a wider preview and Marketplace offers retain 16:9 art. List mode uses compact horizontal rows and preserves selection when changing layout. Old asset grid sizing rules were removed to prevent lazy-loaded styles from distorting media.

Marketplace source context and Join in Minecraft now sit in the section strip rather than a second large title block. Packet search sits at the top of the recordings rail and retains the original query/debounce handlers. Collection checks cover 25 populated items in each view, 800/1100/1500px widths, media proportions, wrapped titles, selection, list/grid changes, the relocated controls and saving a Slim arm override. See `tmp/collection-layout-verification.json` (48 checks).

## Pack object browser

Pack preview selection now uses a shared, searchable dialog instead of a native select with long technical identifiers. Display labels separate readable object names from namespace/group context; exact IDs, geometry and source files remain available under the preview's Identifiers and source disclosure. Labels do not change lookup IDs or stored data. Search matches multiple words across the readable name, original identifier, geometry and file paths, ignoring case and separator differences. The default scope prefers 3D models when the catalog contains them, with individual type filters and counts available in the browser.

Previous/next buttons and focused-toolbar left/right keys switch previews directly. The dialog supports arrow keys, Home/End, Page Up/Down and Enter, keeps selection visible, and restores focus through the shared dialog trigger. A virtual list renders only visible rows plus overscan; the previous 250-object display cap is removed. Changing search/type does not replace the current preview until an object is chosen. A manual file/model preview is retained during catalog loading and search. Texture-variant keys and the original asynchronous request invalidation remain intact. Checks and Pack order were removed from the asset navigation as requested.

Verification: 16 Chromium checks against the actual component with 603 catalog entries cover readable labels, multiword search, direct navigation, keyboard access beyond entry 250, bounded rendering, no-results behavior, type filtering, pack switching, small-window fit and manual-preview retention. The manual-mount regression was reproduced before correction. TypeScript and the production frontend build passed. Evidence: tmp/pack-object-browser-checks.json and tmp/pack-object-browser.png.

### Packet Inspector browsing and wire data (v9)

The inspector retains Bedrock packet bodies on both proxy legs, including packets that cannot be decoded. New observations also retain their source/destination address, client/server leg and item shield context. Received/sent observations are separate records, not duplicate capture bugs. The packet list shows a short redacted content summary; addresses and the original header are available in details. Existing archives remain readable, and missing historical address data is labelled explicitly.

The upper-left search opens a shared studio dialog with recording-wide results, packet-type search, direction/stage/bookmark filters, keyboard selection and paging. Search supports wildcard names (`type:Move*`), OR (`type:Text|type:Modal*`), exclusions, `bytes:>1024`, `field:FormID=7`, `field:Position[1]>=64`, `has:FormData`, `is:partial` and raw hex sequences. Numeric comparisons preserve full integer identifiers. Content and raw-byte searches obey the same redaction rules as details. Queries run one at a time per view, discard obsolete results, and do not expose old rows after a failed query. Statistics and recording/packet Compare controls have been removed.

Decoded form byte fields are shown as JSON rather than base64. Protocol Optional values and input bitsets have readable representations; compatible original wire data can recover these values from older archives. The editable payload and original wire bytes remain separate from this presentation. Derived views are not duplicated on disk. Current/before payloads are redacted together, including secrets inside encoded form JSON. Detail decoding uses an independent reader snapshot so it does not hold the recorder's writer lock.

The field explorer searches all leaves and pages its results, the JSON tree expands further batches, and the hex viewer includes byte-sequence search and navigation through the complete retained body. Limits in evidence/analysis views do not remove packets from an archive. Recording still has visible queue/disk/record-size limits; this is a Bedrock packet inspector, not a transport-frame or HTTP recorder.

### Library menus and wireframe preview (v9)

File > Import & browse opens pack/skin imports, installed Minecraft content and packet recording import. The Library menu links to asset files, packet recording settings/search/rules, marketplace filters/collections/favorites and account settings. View offers list/tile layouts for applicable libraries. Workspace actions run after the project-change guard has accepted navigation and can wait for a lazily loaded target view.

Pack and skin previews (including the skin rig editor) share wireframe controls. The wire-only option hides model surfaces during rendering, while preserving original materials for export and restoring them after each draw. Presets cover overlay, wire-only and topology inspection. Controls include edge angle, pixel line width, opacity, visible/hidden colors, hidden-edge dashes and density, and vertex markers/color/size. Mesh/edge/vertex counts and preview-budget omissions are shown. Animated positions update existing line, dash-distance and marker buffers; helper geometry remains preview-only and is disposed with its overlay.

### Packet filters and form preview (v10)

Type counts in the packet sidebar explicitly cover both directions and all stages. A zero-result state explains a selected type with no observations in the chosen direction and offers Show all <type>. This clears other filters while retaining that type; Reset filters clears direction, stage, search, bookmarks and playback bounds as well. A read-only index audit of the user's active capture confirmed that its LevelEvent observations were clientbound, so a serverbound filter correctly excluded them.

Packet details offer a Form tab and a Preview form shortcut for valid ModalFormRequest/ServerSettingsResponse payloads. The preview supports legacy button menus, modern elements with headers/labels/dividers, two-choice modal dialogs, and custom forms with inputs, toggles, dropdowns, numeric sliders and step sliders. Minecraft section-code colors, bold/italic/reset and real line breaks are rendered without HTML injection. Local interaction exposes a response preview without sending any packet. The enlarged view uses the shared studio dialog; JSON, fields and original packet bytes remain available.

Server font glyphs can be inspected but are hidden by default without their resource font. Referenced pack images have labelled placeholders; image URLs are not fetched automatically. Preview text has shared character/run budgets and dropdowns render at most 200 matches plus their selected value. Large dropdowns are searchable with original response indices preserved. Full source data remains in JSON.

Schema references: the local Dragonfly form/menu/modal/element JSON serializers and Microsoft's official [server-ui module](https://github.com/MicrosoftDocs/minecraft-creator/blob/main/creator/ScriptAPI/minecraft/server-ui/minecraft-server-ui.md), [ActionFormData](https://learn.microsoft.com/en-us/minecraft/creator/scriptapi/minecraft/server-ui/actionformdata?view=minecraft-bedrock-stable), and [ModalFormData](https://github.com/MicrosoftDocs/minecraft-creator/blob/main/creator/ScriptAPI/minecraft/server-ui/ModalFormData.md) documentation.

### Welcome navigation (v11)

Welcome uses one left navigation rail instead of stacked horizontal project tabs and workspace shortcuts. Recent is the home view, Create opens still-render/camera/capture setup, and Explore opens packs, skins, Packet Inspector and Marketplace. Existing open-file/folder/project actions sit below the rail navigation. The content and properties occupy the right side, with independently scrollable navigation/content on smaller windows. The recent-project pin filter is a labelled compact icon toggle to keep the desktop search toolbar on one row.

The standalone New Project dialog retains its three document-type tabs. Welcome starts on Recent even without history. Arrow Up/Down and Home/End move focus through the rail; Enter activates a choice. Design reference: Adobe's [Photoshop Home screen](https://helpx.adobe.com/photoshop/desktop/get-started/learn-the-basics/homescreen-overview.html).

### Lossless GLB storage optimization (v12)

World exports and pack/skin model exports now share the native GLB optimization pass. It removes unreferenced accessors, images, textures, samplers, materials, meshes, skins, cameras and buffer views, repacks live binary ranges, and shares byte-identical geometry/image storage and matching accessor/texture/sampler descriptors. Mesh and material identities remain separate: merging material entries can change renderer sorting of coplanar faces even for opaque materials. Names, metadata, node transforms, attributes, UVs, indices, material settings, skin/morph/animation references and original numeric JSON values are preserved.

PNG compression recompresses the existing filtered IDAT scanlines with the strongest standard zlib compression. It does not decode/re-encode colors, resize images, quantize samples, remove metadata, change palettes or discard alpha. All other chunks and native image-view padding survive. Invalid/unsupported images and recompressions without a size benefit retain their original bytes. Unknown glTF extensions preserve their original resource/binary layout rather than risking invalid reference rewrites. This adds no decoder extension or dependency to exported GLBs.

Validation: seven native tests, including seven existing GLB content comparisons, source immutability, repeat optimization, unknown extensions, sparse/morph/skin/animation references, material differences, texture omission, cancellation, and 8/16-bit/palette PNG samples/metadata. Four actual GLTFLoader/WebGL before/after renders had zero differing color channels. The existing Hive model decreased from 470,444,448 to 423,383,364 bytes (10.0%); six smaller fixtures saved 2.4–16.4%. Evidence is in tmp/glb-lossless and tmp/glb-lossless-verification.log. The successful temporary test/harness sources were removed before the release build, with their hashes retained.

## Document workspaces — v13

Render, Cinematic, resource packs, skins and Packet Inspector each have independent document tabs. Preview/keep, pin/reorder, context menus, keyboard controls, per-area task locks and shared quit saving follow docs/document-tabs.md. Workspace projects reference local originals and preserve presentation state, texture drafts and audio arrangements. Startup always begins with an empty session.


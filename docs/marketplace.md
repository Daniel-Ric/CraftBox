# Marketplace

CraftBox's Marketplace is a native English-only workspace for the same `3PP_V2.0` Discovery list used by Featured Servers and Experiences. There is no unrestricted Marketplace search and no bundled Node service.

Production (`20CA2`) is selected by default. Staging (`EE7B`) is selectable in the workspace header. Both use the existing Discovery list as the allowlist; the catalog requests use the selected PlayFab title. Credentials, scoped item identities and favorite identifiers are separated by environment. The viewer does not change CraftBox's capture connection or start an additional session.

Every offer must have the exact, case-sensitive PlayFab tag `3PServerContent`. Search requests always include that tag, and responses are checked again before offers are retained or displayed. Other creator Marketplace items are excluded. Store and collection associations come from published Discovery references. When a source has no usable published store/collection membership, offers use its exact creator entity together with the mandatory tag, and their provenance says `creator`. A shared creator name alone never establishes an association.

Discovery creator resolution first uses an exact, unambiguous creator mapping published by the Minecraft store's `/api/v1.0/session/config` registry. The registry's qualified entity ID takes precedence over legacy original IDs and the Discovery listing publisher. That mapping is shown in Discovery technical details. If registry data is unavailable, explicit `DisplayProperties.originalCreatorId` takes precedence over the listing publisher; bare original IDs use `master_player_account`, and qualified IDs retain their entity type. Search result adapters accept both direct catalog items and wrapped `Item` hits. Diagnostics record public creator IDs, raw search counts and tagged offer counts without credentials.

Creator images come from API collections with `ContentType` `OfferCollectionQueries_V3.0` and tag `collection.creator`. Thumbnail images are associated with existing Discovery sources through exact `DisplayProperties.creatorIds` entries. The viewer loads them separately in the background for the selected environment, uses the most recently modified matching collection, and retains the collection ID as image provenance. These collections are image metadata, never offers. Images appear in the source list, source overview and creator labels. Unavailable thumbnails fall back to Discovery imagery; switching environment, account or going offline clears image mappings and ignores stale responses. Creator mappings and thumbnails are not persisted offline.

The v3 update was checked with four native regression tests (including local HTTP requests), five Chromium checks for images/environment isolation/fallback, frontend production compilation and scoped Go vet. An authenticated live Hive catalog check remains unverified.

The workspace uses a Discovery source list, a flexible offer grid/list and an item inspector. Main sections are Offers, Collections, More server offers and Favorites. Source banners retain their natural aspect ratio. Sources are sorted by descending current player count after background status checks; unknown counts sort last. Filters open in the shared dialog; search stays at the upper left. Catalog diagnostics remain available in logs rather than occupying a notice panel.

Catalog data, images and history are not saved for offline browsing. Disconnecting clears the displayed catalog; failed requests do not restore cached pages. Ordinary WebView/network caching remains browser-managed, not a CraftBox offline library. Favorites save only environment/source/item identifiers and the relation needed to verify them again online. Unavailable favorites can be removed without retrieving their offers.

## Native bridge contract

- `ListMarketplaceSources(environment)` refreshes Discovery and returns its revision, source overviews and check time. Older concurrent refreshes cannot replace newer ones.
- `SearchMarketplace(query)` accepts `environment` (`production` or `staging`), a discovered source ID or `all`, a relation (`offers`, `collections`, `creator`), text, cursor and filters. The mandatory tag cannot be overridden. Cursors bind to the query and Discovery revision.
- `GetMarketplaceItem(source, item, environment)` refreshes an already scoped item's details. Current store membership/prices are rechecked in store context, including favorites opened after restarting the app.
- `ResolveMarketplaceReference(reference)` validates a saved identifier against the current Discovery source and store/creator relation before opening it.
- `GetMarketplaceFavorites(environment)`, `SetMarketplaceFavorite(source, item, enabled, environment)` and `GetMarketplaceFavoriteItems(environment)` manage identifier-only favorites and their online resolution.
- `GetMarketplacePlayerCounts(environment)` loads Experience counts and read-only featured NetherNet HTTP status separately from Discovery and catalog loading. Unknown counts are omitted, never invented as zero.
- `GetMarketplaceAssetMatches(asset)` returns known, exact server-offer associations with explicit environments, enabling reverse links from Packs and Skins.
- `OpenMarketplaceItem(source, item, environment)` opens the documented Minecraft offer URI. `JoinMarketplaceSource(source, environment)` opens `minecraft://connect` or `minecraft://joinExperience`. Neither prepares capture nor purchases anything.
- `ExportMarketplaceItems(selection, format)` refreshes selected offers online and writes JSON or CSV through a native dialog. Mixed-environment exports and non-server offers are rejected. CSV values are protected against spreadsheet formula interpretation.

## Data and limits

Discovery uses the existing Minecraft Services `POST /api/v2.0/discovery/blob/client` integration, shared with the source picker. Catalog requests use PlayFab `Catalog/SearchItems`, `Catalog/GetItem` and `Catalog/GetItems`, authenticated by `X-EntityToken` from the signed-in Xbox account. Search fixes `Language=en-US`, uses `Count <= 50` and continuation tokens, and restricts each call to a published store or exact creator entity. At most four sources are queried concurrently per search and eight catalog HTTP requests run concurrently across the module. Calls have deadlines and bounded response sizes; only limited 429/503 retries are made.

Prices preserve alternative and composite currency amounts, units and durations. Cards prefer readable payment options, show Minecoins for the documented currency identifier and use Virtual currency for unknown identifiers; original IDs remain in Technical details and exports. Cards also show the item description and rating. Unknown prices are never treated as free.

Creator filtering can only select creators already bound to Discovery entries. Search, type and tag filters are passed upstream where supported. Price, currency and compatibility filters also apply to returned pages; an empty page can therefore still have a continuation. Result counts are loaded counts, not invented catalog totals. Name and price ordering compare loaded results; price ordering requires a selected currency. Collections expose published grouping metadata, not unrelated offers. Individual collection resolution is bounded to 50 referenced items.

3D preview appears only when an item publishes an HTTPS texture and inline Bedrock geometry containing renderable cubes. The viewer uses the existing CraftBox renderer. Captured-content lookup and comparison have been removed. Reverse navigation from existing library assets to known Marketplace associations remains available.

Technical details contain formatted JSON, IDs, tags and association provenance. Known key/token fields and signed/binary content URLs are omitted from that metadata. This redaction applies to exports as well.

## References and verification

Reference behavior: [PlayFab SearchItems](https://learn.microsoft.com/en-us/rest/api/playfab/economy/catalog/search-items?view=playfab-rest), [GetItem](https://learn.microsoft.com/en-us/rest/api/playfab/economy/catalog/get-item?view=playfab-rest), [Minecraft deep links](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/deep-links?view=minecraft-bedrock-stable), and [Daniel-Ric/PlayFab-Catalog-Service-Bedrock](https://github.com/Daniel-Ric/PlayFab-Catalog-Service-Bedrock), particularly `marketplaceSearchService.js`, `featuredServersService.js` and `utils/playfab.js`. This is an independent native implementation following those APIs.

Verification exercises real HTTP requests to a local catalog fixture through the native marketplace service. It checks authentication headers, scope, exact tag enforcement, pagination, environment titles, malformed responses, prices, redaction, cold favorites, CSV safety and lifecycle behavior. Chromium checks the actual React workspace through a temporary local native-contract adapter: environment switching, delayed replies, source navigation, filters, pagination, details, gallery, favorites, comparison, exports, capture links, missing-model fallback, offline states, sign-in and responsive layout. The adapter and synthetic data are verification-only and are removed before the executable build. The Postman CLI is unavailable here, so no Postman run is claimed.

Live authenticated catalog access against Production or Staging has not been exercised in this environment. That remains a separate verification step. No claim is made that a Discovery source necessarily publishes offers in both environments.

Verified Windows GUI artifact: builds/marketplace/Craft-Box.exe, version 1.65.0-marketplace-v1-wails.20261103-dev, 253,323,264 bytes, SHA-256 DD1C946803B57336277DA64E6BAB85789F5A2CE4DCCC8EF4C1B3AE51B0F22EA8. Fourteen module tests, 27 Chromium checks, native protocol regressions, scoped Go vet and frontend production compilation passed. Two scoped reviews found no remaining important blocker. Evidence: tmp/marketplace-final-verification.log, tmp/marketplace-browser-verification.json, tmp/marketplace-build-verified.json and tmp/marketplace-verified-removals.json. Successful temporary test sources were removed before building.


## Marketplace v6 updates

Published Discovery page references may be FriendlyIds rather than catalog item IDs. Page resolution first tries the exact ID, then validates the FriendlyId returned by GetItem on a 400/404. Collection browsing also queries published OfferCollectionQueries_V3.0 metadata and associates it only through exact Discovery page/alternate IDs or creator entity IDs, including DisplayProperties.creatorIds. Metadata collections are not offers; all offer paths still require 3PServerContent. Collection cursors retain query/environment/revision binding. Compatibility is evaluated before filtering collection search results.

Verification: native service tests exercise actual HTTP requests to local PlayFab fixtures for ID/FriendlyId resolution, exact creator scope and compatibility filtering. Twenty-five Chromium checks cover source sorting, notices, simplified tabs, prices, filter dialog, preview gating, collection metadata, centered loading, welcome shortcuts, account capability content and slider keyboard/preset behavior. Live authenticated Production/Staging collection access was not exercised. Postman CLI and Orbit network access were unavailable; local HTTP fixtures were used instead of claiming a Postman or live run.

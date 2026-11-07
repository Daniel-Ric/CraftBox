# 2D atlas and world composition

The recording window uses a paged 2D atlas. The render editor offers Atlas and
Rendered views; the existing rendered map and its styling controls remain usable.

Atlas reads captured raw chunks from stable snapshots without constructing 3D
meshes. Tiles contain actual block heights, biome IDs/names and block names.
Texture sampling supports up to 16 pixels per block in visible areas. Overview
groups, offscreen culling and bounded detail caches keep wide maps responsive.
Water composites over terrain with depth-dependent opacity. Custom top surfaces
composite by height so transparent pixels reveal lower geometry.

Available tools include pointer-anchored zoom, hover coordinates, chunk/block
inspection, coordinate copying, coordinate jumps, adaptive grids, relief,
elevation colors, contours, biome overlays, capture coverage and update flashes.
Selections can be resized or entered numerically and report actual captured and
missing chunks. Markers, distance paths and closed polygon area measurements
can be added. Position picking opens the corresponding location in 3D.

PNG export supports the whole map or a selection, resource-pack texture detail,
optional annotations and a transparent background. It processes eight tiles at
a time. The dialog displays the actual output dimensions and pixel density;
large areas fit a 64 MP / 16384-pixel-side budget. Editor atlas preferences and
markers are saved in studio documents; live atlas preferences are stored per tab.
Geometry-only saved studio projects retain the Rendered map option.

## Merge worlds

`Merge worlds…` in the recording window combines up to 16 placements from tabs
or saved regions. Each placement has a source area and block X/Y/Z offsets.
Drag placements in the preview, optionally snap to chunks, or arrange them east
or south with an exact gap. Y offsets must keep non-air blocks within the
destination Overworld range (-64 through 319); out-of-range blocks are rejected
rather than clipped.

Preparation reads source snapshots into a separate staged world. Air positions
are skipped and never erase earlier terrain. Conflicts count real occupied block
positions, including secondary block layers, rather than rectangle intersections.
Non-air conflicts require a second explicit confirmation. Later placements win
at confirmed conflicts. The confirmed staged snapshot is saved; live source
changes cannot silently add new collisions after review.

Original tabs and archives remain intact. The result is a new finished tab and a
`.mcworld` under `worlds/Compositions`. Block states are translated from actual
palette runtime IDs, including hashed custom states. Block entities, captured
actor NBT, shifted positions and remapped actor IDs are retained. Sources with
incompatible versions/content for the same pack identity are rejected to avoid
silently replacing their assets. Pack folders are stored by identity/version.
Preparation and saving expose progress stages and cancellation.

## Verification

Tests cover negative heights, slices, pagination, alpha compositing, pointer
zoom, coverage gaps, export dimensions, overview image decoding, packing gaps,
overlap confirmation, custom runtime IDs, negative chunk offsets, block-entity
translation, actor relocation and pack conflicts. A real Hive-city fixture
provided 440 atlas chunks and 49 full-detail tiles for browser checks. Native
composition produced an 18-chunk world from 89,050 non-air block placements;
overlapping the same selected area detected 44,525 conflicts. Originals remained
available. Browser checks exercised atlas tools and the composition UI with real
map tiles; native save/import verification is tested separately.

Live atlas performance: detail pages retain their images and request incremental
changes instead of restarting the viewport on every overview update. Unchanged
polls preserve React state identity; recent-change animation runs at 10 Hz.
Shading prepares height/color coefficients once per block rather than per texel.
The 49-tile, 256×256-pixel Hive fixture dropped from a median 1,422.5 ms to 73 ms
for cold styled-image preparation in headless Edge on this machine (three runs).
This is a preparation benchmark, not an end-to-end frame-rate measurement.
Native rendered tiles use a versioned 32 MiB LRU cache, including resolution and
slice height in its key; pages yield after roughly 40 ms of tile work. Retained
four-tile native requests measured approximately 0.154 ms after warmup.

Large captures additionally use spatial groups and cached group signatures,
adaptive overview resolution, local image invalidation and an 8 ms preparation
budget per interactive frame. PNG export retains an unlimited preparation budget.
Base pages publish in batches, decoded-image notifications coalesce per frame,
and per-tile height ranges are retained. Native paging reuses sorted chunk
coordinates and seeks to the cursor instead of sorting the remaining capture
for every page. In an expanded 25,600-tile fixture, warm redraws dropped from
approximately 21 seconds to 0.5 ms. Progressive cold preparation completed in
470 frames: p95 draw time 9.6 ms, maximum 31.4 ms in headless Edge on this machine.
These timings measure canvas drawing, not end-to-end capture throughput.

The merge dialog lists captured worlds with direct Add actions and supports
Duplicate on every placement. Appending a source preserves existing offsets;
merging waits until source bounds and automatic placement are ready. Regression
tests cover fresh-process vanilla registry initialization and equivalent live
boolean/NBT byte states, which previously could panic and surface as HTTP EOF.
Typed state identities retain byte/int distinctions; incompatible registry schemas
return a named merge error. Two full archived Hive sources were prepared together
successfully (2,315 destination chunks), in addition to the smaller save tests.

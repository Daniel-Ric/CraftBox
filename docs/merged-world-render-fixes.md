# Merged-world model corrections

Verified against the 2,915-chunk merged world on 2026-10-30. The read-only audit
identified 6,317 blocks using approximate models and 13 unresolved item-frame
contents. The world data and resource packs were preserved.

The item frames contained beef, bow, chicken, cod, cooked cod, crossbow,
golden apple, mutton, oak boat and porkchop. Their canonical item names did not
match the Bedrock item-texture keys. Alias resolution now respects explicit pack
entries first, then resolves the vanilla aliases and boat variant index.

Connected fences, panes, iron bars and walls now retain captured connection
states. Walls retain short/tall arms and the post flag. Pane meshes use exposed
union surfaces with separate broad/edge sprites and correct cutout transparency.
Beds, chests, banners, torches and wall coral fans have dedicated geometry.
Bed colors and banner colors/patterns use block-entity data. Paired chests use a
single wide skin and lock, including pairs across chunk boundaries; a cropped
chest without its partner remains a single chest.

Verification:

- All 2,915 chunks audited: **0 approximate terrain models, 0 unresolved terrain
  textures, 0 unresolved block-entity/item-frame visuals, no missing paths**.
- Scene, desktop and world-state package tests pass; GUI-tagged desktop and scene
  tests pass. Frontend: 86 files / 291 tests pass; TypeScript and Vite build pass.
- A complete Store → Snapshot → GLB fixture with colored beds, double chests,
  straight/corner panes, iron bars, torches and banners rendered successfully in
  Edge from two angles. Tests cover overlap-free pane surfaces, transparency,
  mattress UV regions and paired chests in all four directions across chunks.

The warning counters remain active for unsupported models or missing assets in
other worlds. These results describe this audited world and the tested fixtures;
they do not suppress warnings globally.

Installed build: `1.0.0-merged-models.20261030-dev`,
`D:/bedrock-tools/Craft-Box.exe` (84,831,744 bytes).
SHA-256: `3AF988B72DCAA89AEFF05ADEE1DB09C14E4FFB60C2ABDB08553BDAA0DED7EF84`.
The installed bytes match the tested build. Native production window startup
was verified using an isolated test process. The previous EXE is preserved as
`Craft-Box-backup-20261030-174701.exe`.

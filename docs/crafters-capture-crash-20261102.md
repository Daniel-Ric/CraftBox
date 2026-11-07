# CraftersMC capture crash — 2026-11-02

The user reported that CraftBox exited while recording CraftersMC. The installed
version was `1.63.1-pack-review-shapes-wails.20261102-dev`. Its log ends just after
joining and writing seven packs at 07:21:25. No fatal stack or matching Windows
Application event was available, so the original process's final stack is unknown.

The latest capture was copied to `tmp/crafters-crash/input.pcap2`, SHA256
`983A9E2240FA38A215C8CF9BEE43BAFCE3B856AC296B269A3F11670BA8BE4890`.
The original log, replay harness, extracted slab entry and before/after replay
logs are in that directory. Originals under D:/bedrock-tools were not modified.

Replaying this exact capture through the proxy/world handler reproduced:

```
cannot register the same state twice
Name: trpixel:end_stone_slab
Properties: minecraft:vertical_half:top
```

The StartGame block entry declares `minecraft:vertical_half` twice: as an
explicit enum `[bottom, top]` and through the placement-position trait. The
Cartesian state generator treated them as separate axes, then assigned both to
the same map key, producing duplicate registry entries and a fatal panic.

`AddCustomBlocks` now adds each property axis once. Explicit server enums take
precedence over trait defaults. The same deduplication handles connection traits.
A regression test first reproduced the exact panic, then passed with two distinct
slab states. Independent review found no must-fix issue in the fix.

The identical capture subsequently replayed to end with a nil result and saved
44 chunks with all seven resource packs. This verifies the reproduced panic fix;
it does not substitute for retesting the live CraftersMC server. Other nonfatal
component/export warnings in the replay were not treated as crash causes.

The pending UI request is also included: creating a new Capture opens its selected
workspace and the source picker directly. All three capture types are covered by
integration tests. Recent-project resume keeps its previous behavior. App and
StartScreen tests passed (28 tests), the frontend production build passed, and
six affected Go packages passed (worlds, worldstate, behaviourpack, proxy,
resourcepacks and scene).

Artifact: `builds/resourcepack-fix/Craft-Box.exe`, version
`1.63.1-capture-crashfix-wails.20261102-dev`, SHA256
`33B6C0EDD0191BD6D6ABB52E12DD5D61994E27692F09D680780EF25973979D6C`.
Built with `gui,production` and `-H=windowsgui`; PE GUI subsystem verified as 2.
Installer source hash refreshed; the installed executable remains unchanged.

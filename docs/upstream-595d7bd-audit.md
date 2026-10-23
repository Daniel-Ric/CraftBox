# CraftBoxs compatibility audit — 2026-10-20

Compared upstream [595d7bd33c41d08bbe8b0d76124560797739cba9](https://github.com/Daniel-Ric/CraftBox#credits--attribution) with CraftBox's vendored dependencies. The parent patch was retrieved from GitHub's commit API and the dependency ranges reviewed against their target revisions.

| Upstream change | CraftBox result |
| --- | --- |
| Dragonfly `4333b778` → `01997e16` | Relevant registry-clone hashing, 1.26.50 palette/stair states, derived-state upgrades and pending-upgrade persistence already present. Preserve local registry/capture changes. |
| Gophertunnel `b614cb65` → `4c798322` | Login timeout/order, empty resource packs, decompression bounds/pooling, protocol/NBT validation, public-key verification, debug packet encoding and Realms version negotiation already present. Preserve local hooks. |
| RakNet `96333dfa` → `33c6c4d` | Not used: no module dependency; only NetherNet is registered in the local transport implementation. Do not introduce a new transport for unreachable changes. |
| NetherNet checksums | Local selected version is v1.0.24, newer than the commit's v1.0.20 prerelease. No downgrade. |
| Dimension `MinimumY` / `HeightRange` fields | Already supported. CraftBox correctly calculates inclusive maximum as `MinimumY + HeightRange - 1`, rather than treating height as an absolute Y coordinate. |

No missing relevant production dependency backport was identified. This audit does not claim byte-for-byte equivalence to the upstream libraries. A regression test now protects negative, positive and zero dimension minima and fallback to the vanilla dimension range.

Validation: world capture/state, proxy/resource-pack and registry merge suites; vendored Gophertunnel Minecraft/resource suites, Dragonfly chunk/MCDB suites. All passed locally. These are fixture and loopback checks, not a live remote-server compatibility claim.

Delivered together with the verified shared lighting revision as `1.59.0-studio-lighting-upstream.20261020-dev` to `D:\CraftBoxs\craft-box.exe`. SHA256: `30B705D32CA53D9CB6E8C37C244CF2EF7DF5470EAD358DF8744D594C88B0317C`. Previous EXE: `craft-box-before-lighting-20261020-124506.exe`. Embedded icon group, nine icon images and application manifest verified before replacement.

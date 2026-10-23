# World pack export

Every activated pack must have a matching manifest and version inside the
mcworld. Resource packs are embedded under UUID-qualified folder names to avoid
name collisions. Copy/decode failures fail saving rather than producing a world
with missing dependencies. Generated behavior packs include entity-only content.

Encrypted server packs are decoded for rendering/export using the content key
already supplied in the resource-pack handshake. The original network archive
and key are untouched. The export copy decodes contents.json (256-byte header,
AES-256-CFB8 payload) and the per-file keys it contains. Exported contents entries
contain no encryption keys. Keys are not logged. Invalid/missing keys report an
export failure; they cannot be invented or recovered from a world without packs.

Previously exported worlds with missing pack folders need a new capture with
this version. Adding activation JSON alone cannot restore absent textures or
geometries. Server-side behavior/scripts not supplied to the client remain
unavailable; generated definitions reproduce the captured client-side data.

Format reference: https://github.com/iteplenky/bedrock-pack-tools/blob/main/decrypt.go
Original export comparison: https://github.com/Daniel-Ric/CraftBox#credits--attribution

Generated behavior packs are snapshots with a world-visit-specific UUID and
module identity. They do not modify the session pack or retain player models
from earlier visits. Other players are excluded by default; the optional player
model export is explicitly labelled under advanced options. Custom NPCs remain.

Block export targets format 1.26.30 consistently. Legacy block_property queries
become block_state queries, legacy light/unit-cube components are converted,
geometry visibility/culling/UV options are preserved, and omitted transformation
scale defaults to one. Modern ambient occlusion remains numeric. Components are
copied before conversion so rendering/capture input is not mutated.

Custom actor physics follows captured gravity/collision flags. Gravity-free
display actors receive full knockback resistance and cannot be pushed, so melee
hits do not launch them permanently. Decimal attribute values retain precision.

Entity catalog and SyncActorProperty packets are handled during login and normal
play. Definitions also include custom client entities from all exported resource
packs, even when that type did not spawn in the capture. This creates definitions,
not additional actors. Observed properties/physics remain authoritative; vanilla
entity behavior is preserved. Enum property indices are translated to their
declared string values before saving, as required by Minecraft's persistence
format. Actor-data changes update the generated definition after metadata merge.

The SG-Lobby fixture contained 712 client definitions (710 custom, two vanilla)
and 838 models but only 40 generated behavior definitions before this correction.
Some references target built-in vanilla assets and are not separate server files.
Files the server never supplies cannot be recovered by copying the original repo.

Entity-property reference:
https://learn.microsoft.com/en-us/minecraft/creator/documents/introductiontoentityproperties

Schema references:
- https://learn.microsoft.com/en-us/minecraft/creator/reference/content/blockreference/examples/blockcomponents/minecraftblock_geometry
- https://learn.microsoft.com/en-us/minecraft/creator/reference/content/blockreference/examples/blockcomponents/minecraftblock_material_instances
- https://learn.microsoft.com/en-us/minecraft/creator/reference/content/entityreference/examples/entitycomponents/minecraftcomponent_knockback_resistance

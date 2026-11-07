package assetlibrary

// A few shipped resource packs contain geometry and matching textures without
// the client entity that normally links them. These verified exceptions are
// scoped to the pack UUID, exact geometry identifier and original model path.
// Never guess another texture for a variant whose source image is missing.
func (x objectIndex) verifiedGeometryTexture(id, file string) (string, bool) {
	header := objectMap(x.docs["manifest.json"]["header"])
	if objectString(header["uuid"]) != "3cceca35-04df-4d2c-aaaf-e682d4b584bc" {
		return "", false
	}
	switch {
	case id == "geometry.cubecraft.lobby.hotairballoon.fire" && file == "models/entity/transport/fire.hotairballoon.geo.json":
		return "textures/transport/balloon_fire", true
	case file == "models/entity/marketplace_stand/texture_pack.marketplace.geo.json":
		switch id {
		case "geometry.cubecraft.marketplace.texture_pack.pvp":
			return "textures/marketplace_stand/pvp_texture_pack", true
		case "geometry.cubecraft.marketplace.texture_pack.melior":
			// This unused geometry is present, but its image is absent from
			// both the pack's contents list and the supplied texture files.
			return "", true
		}
	}
	return "", false
}

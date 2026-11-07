package behaviourpack

import (
	"encoding/json"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func BlockRenderJSON(block protocol.BlockEntry) ([]byte, error) {
	definition, version := parseBlock(block)
	return json.Marshal(BlockBehaviour{FormatVersion: version, MinecraftBlock: definition})
}

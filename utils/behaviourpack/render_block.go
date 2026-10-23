package behaviourpack

import (
	"encoding/json"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func BlockRenderJSON(block protocol.BlockEntry) ([]byte, error) {
	definition, _ := parseBlock(block)
	return json.Marshal(definition)
}

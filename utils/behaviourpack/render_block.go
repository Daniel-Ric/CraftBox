package behaviourpack

import (
	"encoding/json"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func BlockRenderJSON(block protocol.BlockEntry) (data []byte, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("block %s: %v", block.Name, value)
		}
	}()

	definition, version := parseBlock(block)
	return json.Marshal(BlockBehaviour{FormatVersion: version, MinecraftBlock: definition})
}

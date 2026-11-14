package behaviourpack

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sirupsen/logrus"
)

type BlockBehaviour struct {
	FormatVersion  string         `json:"format_version"`
	MinecraftBlock MinecraftBlock `json:"minecraft:block"`
}

func (bp *Pack) AddBlock(block protocol.BlockEntry) {
	defer func() {
		if value := recover(); value != nil {
			logrus.Warnf("Skipping block %s: %v", block.Name, value)
		}
	}()

	ns, _ := splitNamespace(block.Name)
	if ns == "minecraft" {
		return
	}

	minecraftBlock, version := parseBlock(block)

	bp.blocks[block.Name] = &BlockBehaviour{
		FormatVersion:  version,
		MinecraftBlock: minecraftBlock,
	}
}

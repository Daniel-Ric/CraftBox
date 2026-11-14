package worldstate

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
)

type Generator string

const (
	GeneratorVoid Generator = "void"

	GeneratorFlat Generator = "flat"

	GeneratorNormal Generator = "normal"
)

var Generators = []Generator{GeneratorVoid, GeneratorFlat, GeneratorNormal}

func ParseGenerator(name string) (Generator, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return GeneratorVoid, nil
	}
	for _, g := range Generators {
		if string(g) == name {
			return g, nil
		}
	}
	return "", fmt.Errorf("unknown generator %q (use void, flat or normal)", name)
}

const voidLayers = `{"biome_id":1,"block_layers":[{"block_data":0,"block_id":0,"count":1},{"block_data":0,"block_id":0,"count":2},{"block_data":0,"block_id":0,"count":1}],"encoding_version":3,"structure_options":null}`

func applyGenerator(ld *leveldat.Data, g Generator) {
	switch g {
	case GeneratorFlat:

		ld.Generator = 2
		ld.FlatWorldLayers = ""
	case GeneratorNormal:
		ld.Generator = 1
		ld.FlatWorldLayers = ""
	default:
		ld.Generator = 2
		ld.FlatWorldLayers = voidLayers
	}
}

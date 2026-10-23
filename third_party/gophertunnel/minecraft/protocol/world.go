package protocol

import "github.com/google/uuid"

const (
	GeneratorLegacy    = 0
	GeneratorOverworld = 1
	GeneratorFlat      = 2
	GeneratorNether    = 3
	GeneratorEnd       = 4
	GeneratorVoid      = 5
)

type DimensionDefinition struct {
	Name string

	MinimumY int32

	HeightRange int32

	Generator int32

	DimensionType int32

	PackID uuid.UUID

	DefaultBiome string
}

func (x *DimensionDefinition) Marshal(r IO) {
	r.String(&x.Name)
	r.Varint32(&x.MinimumY)
	r.Varint32(&x.HeightRange)
	r.Varint32(&x.Generator)
	r.Varint32(&x.DimensionType)
	r.UUID(&x.PackID)
	r.String(&x.DefaultBiome)
}

type GenerationFeature struct {
	Name string

	JSON []byte
}

func (x *GenerationFeature) Marshal(r IO) {
	r.String(&x.Name)
	r.ByteSlice(&x.JSON)
}

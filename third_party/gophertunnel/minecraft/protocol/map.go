package protocol

import (
	"image/color"
)

const (
	MapDecorationTypeMarkerWhite = iota
	MapDecorationTypeMarkerGreen
	MapDecorationTypeMarkerRed
	MapDecorationTypeMarkerBlue
	MapDecorationTypeCrossWhite
	MapDecorationTypeTriangleRed
	MapDecorationTypeSquareWhite
	MapDecorationTypeMarkerSign
	MapDecorationTypeMarkerPink
	MapDecorationTypeMarkerOrange
	MapDecorationTypeMarkerYellow
	MapDecorationTypeMarkerTeal
	MapDecorationTypeTriangleGreen
	MapDecorationTypeSmallSquareWhite
	MapDecorationTypeMansion
	MapDecorationTypeMonument
	MapDecorationTypeNoDraw
	MapDecorationTypeVillageDesert
	MapDecorationTypeVillagePlains
	MapDecorationTypeVillageSavanna
	MapDecorationTypeVillageSnowy
	MapDecorationTypeVillageTaiga
	MapDecorationTypeJungleTemple
	MapDecorationTypeWitchHut
	MapDecorationTypeTrialChambers
	MapDecorationTypeAbandonedCamp
	MapDecorationTypeBuriedAncientCity
	MapDecorationTypeBuriedMineshaft
	MapDecorationTypeDesertPyramid
	MapDecorationTypeWarmOceanRuins
)

const (
	MapObjectTypeEntity = iota
	MapObjectTypeBlock
)

type MapTrackedObject struct {
	Type int32

	EntityUniqueID Optional[int64]

	BlockPosition Optional[BlockPos]
}

func (x *MapTrackedObject) Marshal(r IO) {
	r.Int32(&x.Type)
	OptionalFunc(r, &x.EntityUniqueID, r.ActorUniqueID)
	OptionalFunc(r, &x.BlockPosition, r.BlockPos)
	if x.Type != MapObjectTypeEntity && x.Type != MapObjectTypeBlock {
		r.UnknownEnumOption(x.Type, "map tracked object type")
	}
}

type MapDecoration struct {
	Type byte

	Rotation byte

	X byte

	Y byte

	Label string

	Colour color.RGBA
}

func (x *MapDecoration) Marshal(r IO) {
	r.Uint8(&x.Type)
	r.Uint8(&x.Rotation)
	r.Uint8(&x.X)
	r.Uint8(&x.Y)
	r.String(&x.Label)
	r.BEARGB(&x.Colour)
}

type PixelRequest struct {
	Colour color.RGBA
	Index  uint16
}

func (x *PixelRequest) Marshal(r IO) {
	r.RGBA(&x.Colour)
	r.Uint16(&x.Index)
}

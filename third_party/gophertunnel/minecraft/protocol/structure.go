package protocol

import (
	"github.com/go-gl/mathgl/mgl32"
)

const (
	StructureMirrorNone = iota
	StructureMirrorXAxis
	StructureMirrorZAxis
	StructureMirrorBothAxes
)

const (
	StructureRotationNone = iota
	StructureRotationRotate90
	StructureRotationRotate180
	StructureRotationRotate270
)

const (
	AnimationModeNone = iota
	AnimationModeLayers
	AnimationModeBlocks
)

type StructureSettings struct {
	PaletteName string

	IgnoreEntities bool

	IgnoreBlocks bool

	AllowNonTickingChunks bool

	Size BlockPos

	Offset BlockPos

	LastEditingPlayerUniqueID int64

	Rotation byte

	Mirror byte

	AnimationMode byte

	AnimationDuration float32

	Integrity float32

	Seed uint32

	Pivot mgl32.Vec3
}

func (x *StructureSettings) Marshal(r IO) {
	r.String(&x.PaletteName)
	r.Bool(&x.IgnoreEntities)
	r.Bool(&x.IgnoreBlocks)
	r.Bool(&x.AllowNonTickingChunks)
	r.BlockPos(&x.Size)
	r.BlockPos(&x.Offset)
	r.ActorUniqueID(&x.LastEditingPlayerUniqueID)
	r.Uint8(&x.Rotation)
	r.Uint8(&x.Mirror)
	r.Uint8(&x.AnimationMode)
	r.Float32(&x.AnimationDuration)
	r.Float32(&x.Integrity)
	r.Uint32(&x.Seed)
	r.Vec3(&x.Pivot)
}

package protocol

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
)

const (
	WaypointActionNone = iota
	WaypointActionAdd
	WaypointActionRemove
	WaypointActionUpdate
)

const (
	WaypointUpdateFlagVisible = 1 << iota
	WaypointUpdateFlagPosition
	WaypointUpdateFlagTextureID
	WaypointUpdateFlagColour
	WaypointUpdateFlagClientPositionAuthority
	WaypointUpdateFlagActorUniqueID
)

const (
	WaypointTextureSquare = iota + 2
	WaypointTextureCircle
	WaypointTextureSmallSquare
	WaypointTextureSmallStar
)

type LocatorBarWaypoint struct {
	GroupHandle uuid.UUID

	Waypoint Waypoint

	Action uint8
}

func (x *LocatorBarWaypoint) Marshal(r IO) {
	r.UUID(&x.GroupHandle)
	Single(r, &x.Waypoint)
	r.Uint8(&x.Action)
}

type WaypointWorldPosition struct {
	Position mgl32.Vec3

	DimensionID int32
}

func (x *WaypointWorldPosition) Marshal(r IO) {
	r.Vec3(&x.Position)
	r.Varint32(&x.DimensionID)
}

type Waypoint struct {
	UpdateFlag uint32

	Visible Optional[bool]

	WorldPosition Optional[WaypointWorldPosition]

	TexturePath Optional[string]

	IconSize Optional[mgl32.Vec2]

	Colour Optional[int32]

	ClientPositionAuthority Optional[bool]

	ActorUniqueID Optional[int64]
}

func (x *Waypoint) Marshal(r IO) {
	r.Uint32(&x.UpdateFlag)
	OptionalFunc(r, &x.Visible, r.Bool)
	OptionalMarshaler(r, &x.WorldPosition)
	OptionalFunc(r, &x.TexturePath, r.String)
	OptionalFunc(r, &x.IconSize, r.Vec2)
	OptionalFunc(r, &x.Colour, r.Int32)
	OptionalFunc(r, &x.ClientPositionAuthority, r.Bool)
	OptionalFunc(r, &x.ActorUniqueID, r.ActorUniqueID)
}

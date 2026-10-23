package protocol

import "github.com/google/uuid"

type GatheringJoinInfo struct {
	ExperienceID uuid.UUID

	ExperienceName string

	ExperienceWorldID Optional[uuid.UUID]

	ExperienceWorldName Optional[string]

	CreatorID string

	TargetID Optional[uuid.UUID]

	ScenarioID Optional[string]

	ServerID Optional[string]
}

func (x *GatheringJoinInfo) Marshal(r IO) {
	r.UUID(&x.ExperienceID)
	r.String(&x.ExperienceName)
	OptionalFunc(r, &x.ExperienceWorldID, r.UUID)
	OptionalFunc(r, &x.ExperienceWorldName, r.String)
	r.String(&x.CreatorID)
	OptionalFunc(r, &x.TargetID, r.UUID)
	OptionalFunc(r, &x.ScenarioID, r.String)
	OptionalFunc(r, &x.ServerID, r.String)
}

type StoreEntryPointInfo struct {
	StoreID string

	StoreName string
}

func (x *StoreEntryPointInfo) Marshal(r IO) {
	r.String(&x.StoreID)
	r.String(&x.StoreName)
}

type PresenceInfo struct {
	RichPresenceID Optional[string]
}

func (x *PresenceInfo) Marshal(r IO) {
	OptionalFunc(r, &x.RichPresenceID, r.String)
}

type ServerJoinInformation struct {
	GatheringJoinInfo Optional[GatheringJoinInfo]

	StoreEntryPointInfo Optional[StoreEntryPointInfo]

	PresenceInfo Optional[PresenceInfo]
}

func (x *ServerJoinInformation) Marshal(r IO) {
	OptionalMarshaler(r, &x.GatheringJoinInfo)
	OptionalMarshaler(r, &x.StoreEntryPointInfo)
	OptionalMarshaler(r, &x.PresenceInfo)
}

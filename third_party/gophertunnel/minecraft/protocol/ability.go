package protocol

const (
	AbilityBuild = 1 << iota
	AbilityMine
	AbilityDoorsAndSwitches
	AbilityOpenContainers
	AbilityAttackPlayers
	AbilityAttackMobs
	AbilityOperatorCommands
	AbilityTeleport
	AbilityInvulnerable
	AbilityFlying
	AbilityMayFly
	AbilityInstantBuild
	AbilityLightning
	AbilityFlySpeed
	AbilityWalkSpeed
	AbilityMuted
	AbilityWorldBuilder
	AbilityNoClip
	AbilityPrivilegedBuilder
	AbilityVerticalFlySpeed
	AbilityCount
)

const (
	AbilityLayerTypeCustomCache = iota
	AbilityLayerTypeBase
	AbilityLayerTypeSpectator
	AbilityLayerTypeCommands
	AbilityLayerTypeEditor
	AbilityLayerTypeLoadingScreen
)

const (
	AbilityBaseFlySpeed         = 0.05
	AbilityBaseWalkSpeed        = 0.1
	AbilityBaseVerticalFlySpeed = 1.0
)

type AbilityData struct {
	EntityUniqueID int64

	PlayerPermissions byte

	CommandPermissions byte

	Layers []AbilityLayer
}

func (x *AbilityData) Marshal(r IO) {
	r.ActorUniqueIDInt64(&x.EntityUniqueID)
	r.Uint8(&x.PlayerPermissions)
	r.Uint8(&x.CommandPermissions)
	Slice(r, &x.Layers)
}

type AbilityLayer struct {
	Type uint16

	Abilities uint32

	Values uint32

	FlySpeed float32

	VerticalFlySpeed float32

	WalkSpeed float32
}

func (x *AbilityLayer) Marshal(r IO) {
	r.Uint16(&x.Type)
	r.Uint32(&x.Abilities)
	r.Uint32(&x.Values)
	r.Float32(&x.FlySpeed)
	r.Float32(&x.VerticalFlySpeed)
	r.Float32(&x.WalkSpeed)
}

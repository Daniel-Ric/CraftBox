package protocol

const (
	ScoreboardIdentityRemove = iota
	ScoreboardIdentityPlayer
	ScoreboardIdentityEntity
	ScoreboardIdentityFakePlayer
)

// line on the scoreboard each.
type ScoreboardEntry struct {
	EntryID int64

	ObjectiveName string

	Score int32

	IdentityType byte

	EntityUniqueID int64

	DisplayName string
}

func (x *ScoreboardEntry) Marshal(r IO) {
	variant := uint32(x.IdentityType)
	r.Varuint32(&variant)
	x.IdentityType = byte(variant)

	typeNames := [...]string{"remove", "changeplayer", "changeentity", "changefakeplayer"}
	if variant >= uint32(len(typeNames)) {
		r.UnknownEnumOption(variant, "scoreboard entry variant")
		return
	}
	typeName := typeNames[variant]
	r.String(&typeName)
	r.Varint64(&x.EntryID)
	switch x.IdentityType {
	case ScoreboardIdentityRemove:
		objective := Optional[string]{}
		if x.ObjectiveName != "" {
			objective = Option(x.ObjectiveName)
		}
		OptionalFunc(r, &objective, r.String)
		x.ObjectiveName, _ = objective.Value()
	case ScoreboardIdentityEntity, ScoreboardIdentityPlayer:
		r.String(&x.ObjectiveName)
		r.Int32(&x.Score)
		r.ActorUniqueID(&x.EntityUniqueID)
	case ScoreboardIdentityFakePlayer:
		r.String(&x.ObjectiveName)
		r.Int32(&x.Score)
		r.String(&x.DisplayName)
	}
}

type ScoreboardIdentityEntry struct {
	EntryID int64

	EntityUniqueID Optional[int64]
}

func (x *ScoreboardIdentityEntry) Marshal(r IO) {
	r.Varint64(&x.EntryID)
	OptionalFunc(r, &x.EntityUniqueID, r.ActorUniqueID)
}

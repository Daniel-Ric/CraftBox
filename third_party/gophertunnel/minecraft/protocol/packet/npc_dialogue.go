package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

const (
	NPCDialogueActionOpen int32 = iota
	NPCDialogueActionClose
)

type NPCDialogue struct {
	EntityUniqueID uint64

	ActionType int32

	Dialogue string

	SceneName string

	NPCName string

	ActionJSON string
}

func (*NPCDialogue) ID() uint32 {
	return IDNPCDialogue
}

func (pk *NPCDialogue) Marshal(io protocol.IO) {
	io.ActorUniqueIDUint64(&pk.EntityUniqueID)
	io.Varint32(&pk.ActionType)
	io.String(&pk.Dialogue)
	io.String(&pk.SceneName)
	io.String(&pk.NPCName)
	io.String(&pk.ActionJSON)
}

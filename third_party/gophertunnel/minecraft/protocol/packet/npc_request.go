package packet

import "github.com/sandertv/gophertunnel/minecraft/protocol"

const (
	NPCRequestActionSetActions = iota
	NPCRequestActionExecuteAction
	NPCRequestActionExecuteClosingCommands
	NPCRequestActionSetName
	NPCRequestActionSetSkin
	NPCRequestActionSetInteractText
	NPCRequestActionExecuteOpeningCommands
)

type NPCRequest struct {
	EntityRuntimeID uint64

	RequestType byte

	CommandString string

	ActionType byte

	SceneName string
}

func (*NPCRequest) ID() uint32 {
	return IDNPCRequest
}

func (pk *NPCRequest) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Uint8(&pk.RequestType)
	io.String(&pk.CommandString)
	io.Uint8(&pk.ActionType)
	io.String(&pk.SceneName)
}

package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	AgentActionTypeAttack = iota + 1
	AgentActionTypeCollect
	AgentActionTypeDestroy
	AgentActionTypeDetectRedstone
	AgentActionTypeDetectObstacle
	AgentActionTypeDrop
	AgentActionTypeDropAll
	AgentActionTypeInspect
	AgentActionTypeInspectData
	AgentActionTypeInspectItemCount
	AgentActionTypeInspectItemDetail
	AgentActionTypeInspectItemSpace
	AgentActionTypeInteract
	AgentActionTypeMove
	AgentActionTypePlaceBlock
	AgentActionTypeTill
	AgentActionTypeTransferItemTo
	AgentActionTypeTurn
)

type AgentAction struct {
	Identifier string

	Action int32

	Response []byte
}

func (*AgentAction) ID() uint32 {
	return IDAgentAction
}

func (pk *AgentAction) Marshal(io protocol.IO) {
	io.String(&pk.Identifier)
	io.Int32(&pk.Action)
	io.ByteSlice(&pk.Response)
}

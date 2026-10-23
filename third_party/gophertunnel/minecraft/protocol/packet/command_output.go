package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CommandOutputTypeNone = iota
	CommandOutputTypeLastOutput
	CommandOutputTypeSilent
	CommandOutputTypeAllOutput
	CommandOutputTypeDataSet
)

type CommandOutput struct {
	CommandOrigin protocol.CommandOrigin

	OutputType byte

	SuccessCount uint32

	OutputMessages []protocol.CommandOutputMessage

	DataSet protocol.Optional[string]
}

func (*CommandOutput) ID() uint32 {
	return IDCommandOutput
}

func (pk *CommandOutput) Marshal(io protocol.IO) {
	protocol.CommandOriginData(io, &pk.CommandOrigin)
	outputTypeStr := commandOutputTypeToString(pk.OutputType)
	io.String(&outputTypeStr)
	commandOutputTypeFromString(io, &pk.OutputType, outputTypeStr)
	io.Uint32(&pk.SuccessCount)
	protocol.Slice(io, &pk.OutputMessages)
	protocol.OptionalFunc(io, &pk.DataSet, io.String)
}

func commandOutputTypeToString(x byte) string {
	switch x {
	case CommandOutputTypeNone:
		return "none"
	case CommandOutputTypeLastOutput:
		return "lastoutput"
	case CommandOutputTypeSilent:
		return "silent"
	case CommandOutputTypeAllOutput:
		return "alloutput"
	case CommandOutputTypeDataSet:
		return "dataset"
	default:
		return "unknown"
	}
}

func commandOutputTypeFromString(io protocol.IO, x *byte, s string) {
	switch s {
	case "none":
		*x = CommandOutputTypeNone
	case "lastoutput":
		*x = CommandOutputTypeLastOutput
	case "silent":
		*x = CommandOutputTypeSilent
	case "alloutput":
		*x = CommandOutputTypeAllOutput
	case "dataset":
		*x = CommandOutputTypeDataSet
	default:
		io.InvalidValue(s, "outputType", "unknown type")
	}
}

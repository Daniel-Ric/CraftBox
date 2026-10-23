package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type ClientBoundAttributeLayerSync struct {
	PayloadType uint32

	Layers []protocol.AttributeLayerData

	LayerName string

	DimensionID int32

	Settings protocol.AttributeLayerSettings

	EnvironmentAttributes []protocol.EnvironmentAttributeData

	RemoveAttributeNames []string
}

func (*ClientBoundAttributeLayerSync) ID() uint32 {
	return IDClientBoundAttributeLayerSync
}

func (pk *ClientBoundAttributeLayerSync) Marshal(io protocol.IO) {
	io.Varuint32(&pk.PayloadType)
	switch pk.PayloadType {
	case protocol.AttributeLayerPayloadTypeUpdateLayers:
		protocol.Slice(io, &pk.Layers)
	case protocol.AttributeLayerPayloadTypeUpdateSettings:
		io.String(&pk.LayerName)
		io.Varint32(&pk.DimensionID)
		protocol.Single(io, &pk.Settings)
	case protocol.AttributeLayerPayloadTypeUpdateEnvironment:
		io.String(&pk.LayerName)
		io.Varint32(&pk.DimensionID)
		protocol.Slice(io, &pk.EnvironmentAttributes)
	case protocol.AttributeLayerPayloadTypeRemoveEnvironment:
		io.String(&pk.LayerName)
		io.Varint32(&pk.DimensionID)
		protocol.FuncSlice(io, &pk.RemoveAttributeNames, io.String)
	default:
		io.UnknownEnumOption(pk.PayloadType, "attribute layer payload type")
	}
}

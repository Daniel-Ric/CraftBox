package scene

import (
	"github.com/Daniel-Ric/CraftBox/utils/nbtconv"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func ActorFromMetadata(id int64, typ string, pos [3]float32, yaw float32, metadata protocol.EntityMetadata, properties map[string]any) Actor {
	a := Actor{ID: id, Type: typ, Position: pos, Yaw: yaw, Scale: 1, Height: 1.8, Properties: properties}
	if v, ok := metadata[protocol.EntityDataKeyScale]; ok {
		a.Scale = float32(numeric(v))
	}
	if v, ok := metadata[protocol.EntityDataKeyHeight]; ok {
		a.Height = float32(numeric(v))
	}
	variant, _ := nbtconv.Integer(metadata[protocol.EntityDataKeyVariant])
	a.Variant = int(variant)
	variant, _ = nbtconv.Integer(metadata[protocol.EntityDataKeyMarkVariant])
	a.MarkVariant = int(variant)
	a.OwnerID, _ = nbtconv.Integer(metadata[protocol.EntityDataKeyOwner])
	a.Name, _ = metadata[protocol.EntityDataKeyName].(string)
	if raw, ok := metadata[protocol.EntityDataKeyNameRawText].(string); ok && raw != "" {
		a.Name = raw
	}
	flags, _ := nbtconv.Integer(metadata[protocol.EntityDataKeyFlags])
	a.Invisible = flags&(1<<protocol.EntityDataFlagInvisible) != 0
	a.NameVisible = flags&(1<<protocol.EntityDataFlagAlwaysShowName|1<<protocol.EntityDataFlagShowName) != 0 || nbtconv.Flag(metadata[protocol.EntityDataKeyAlwaysShowNameTag])
	return a
}

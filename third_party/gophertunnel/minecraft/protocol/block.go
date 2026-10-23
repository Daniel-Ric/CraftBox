package protocol

import "github.com/sandertv/gophertunnel/minecraft/nbt"

type BlockEntry struct {
	Name string

	Properties map[string]any
}

func (x *BlockEntry) Marshal(r IO) {
	r.String(&x.Name)
	r.NBT(&x.Properties, nbt.NetworkLittleEndian)
}

type BlockChangeEntry struct {
	BlockPos

	BlockRuntimeID uint32

	Flags uint32

	SyncedUpdateEntityUniqueID uint64

	SyncedUpdateType uint32
}

func (x *BlockChangeEntry) Marshal(r IO) {
	r.BlockPos(&x.BlockPos)
	r.Varuint32(&x.BlockRuntimeID)
	r.Varuint32(&x.Flags)
	r.ActorUniqueIDVaruint64(&x.SyncedUpdateEntityUniqueID)
	r.Varuint32(&x.SyncedUpdateType)
}

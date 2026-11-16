package worlds

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (w *worldsHandler) handleLevelChunk(pk *packet.LevelChunk, timeReceived time.Time) (err error) {
	if len(pk.RawPayload) == 0 {
		w.log.Info(locale.Loc("empty_chunk", nil))
		return
	}

	if pk.CacheEnabled {
		return errors.New("cache is supposed to be handled in proxy")
	}

	w.worldStateMu.Lock()
	defer w.worldStateMu.Unlock()
	if w.serverState.blocks == nil || w.worldState == nil {
		return errors.New("cannot capture chunk: world StartGame initialisation did not complete")
	}

	levelChunk, blockNBTs, err := chunk.DecodeCapturedNetwork(
		w.serverState.blocks,
		pk.RawPayload, int(pk.SubChunkCount),
		w.worldState.Range(),
		w.serverState.useHashedRids,
	)
	if err != nil {
		return err
	}

	ch := &worldstate.Chunk{
		Chunk:         levelChunk,
		BlockEntities: make(map[cube.Pos]map[string]any),
	}

	invalidBlockEntities := 0
	for _, blockNBT := range blockNBTs {
		pos, ok := blockEntityPosition(blockNBT)
		if !ok {
			invalidBlockEntities++
			continue
		}
		ch.BlockEntities[pos] = blockNBT
	}
	if invalidBlockEntities > 0 {
		w.log.Warnf("Skipped %d block entity NBT records without valid int32 coordinates in level chunk %v", invalidBlockEntities, pk.Position)
	}

	pos := world.ChunkPos(pk.Position)
	if w.scripting != nil {
		if !w.scripting.OnChunkAdd(pos, timeReceived) {
			w.worldState.IgnoredChunks[pos] = true
			return
		}
	}
	w.worldState.IgnoredChunks[pos] = false

	// A connected Minecraft client schedules streamed subchunks itself. Capture
	// its responses without injecting a second request stream during loading.
	// Clientless captures still need to request their own terrain.
	if subChunkLimit, ok := pk.SubChunkLimit.Value(); ok && w.session.Client == nil {
		max := w.worldState.Dimension().Range().Height() / 16
		if subChunkLimit != -1 {
			max = int(subChunkLimit)
		}

		var offsetTable []protocol.SubChunkOffset
		r := w.worldState.Dimension().Range()
		for y := int8(r.Min() / 16); y < int8(r.Max()/16)+1; y++ {
			offsetTable = append(offsetTable, protocol.SubChunkOffset{0, y, 0})
		}

		dimId, _ := world.DimensionID(w.worldState.Dimension())
		_ = w.session.Server.WritePacket(&packet.SubChunkRequest{
			Dimension: int32(dimId),
			Position: protocol.SubChunkPos{
				pk.Position.X(), 0, pk.Position.Z(),
			},
			Offsets: offsetTable[:min(max+1, len(offsetTable))],
		})
	}

	err = w.worldState.StoreChunk(pos, ch)
	if err != nil {
		w.log.Error(err)
	}

	if w.scripting != nil {
		w.scripting.OnChunkData(pos)
	}

	total, active := w.worldState.EntityCounts(w.serverState.getEntityRenderDistance())
	w.session.SendPopup(locale.Locm("popup_chunk_count", locale.Strmap{
		"Chunks":   len(w.worldState.StoredChunks),
		"Entities": fmt.Sprintf("%d (%d)", total, active),
		"Name":     w.worldState.Name,
	}, len(w.worldState.StoredChunks)))

	return nil
}

func (w *worldsHandler) processSubChunk(pk *packet.SubChunk) error {
	w.worldStateMu.Lock()
	defer w.worldStateMu.Unlock()

	var chunks = make(map[world.ChunkPos]*worldstate.Chunk)
	invalidBlockEntities := 0
	for _, ent := range pk.SubChunkEntries {
		if ent.Result != protocol.SubChunkResultSuccess {
			continue
		}
		var (
			absX = pk.Position[0] + int32(ent.Offset[0])
			absZ = pk.Position[2] + int32(ent.Offset[2])
			pos  = world.ChunkPos{absX, absZ}
		)

		if w.worldState.IgnoredChunks[pos] {
			continue
		}

		if _, ok := chunks[pos]; ok {
			continue
		}
		ch, ok, err := w.worldState.LoadChunk(pos)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("bug check: subchunk received before chunk")
		}
		chunks[pos] = ch
	}

	for _, ent := range pk.SubChunkEntries {
		var (
			absX = pk.Position[0] + int32(ent.Offset[0])
			absY = pk.Position[1] + int32(ent.Offset[1])
			absZ = pk.Position[2] + int32(ent.Offset[2])
			pos  = world.ChunkPos{absX, absZ}
		)

		ch, ok := chunks[pos]
		if !ok {
			continue
		}

		switch ent.Result {
		case protocol.SubChunkResultSuccessAllAir:
		case protocol.SubChunkResultSuccess:
			payload, ok := ent.RawPayload.Value()
			if !ok {
				continue
			}
			buf := bytes.NewBuffer(payload)
			index := uint8(absY)
			sub, err := chunk.DecodeCapturedSubChunk(
				buf,
				w.serverState.blocks,
				w.worldState.Dimension().Range(),
				&index,
				chunk.NetworkEncoding,
				w.serverState.useHashedRids,
			)
			if err != nil {
				return err
			}
			ch.Chunk.Sub()[index] = sub

			if buf.Len() > 0 {
				dec := nbt.NewDecoderWithEncoding(buf, nbt.NetworkLittleEndian)
				for buf.Len() > 0 {
					blockNBT := make(map[string]any, 0)
					if err := dec.Decode(&blockNBT); err != nil {
						return err
					}
					blockPos, ok := blockEntityPosition(blockNBT)
					if !ok {
						invalidBlockEntities++
						continue
					}
					ch.BlockEntities[blockPos] = blockNBT
				}
			}
		}
	}

	if invalidBlockEntities > 0 {
		w.log.Warnf("Skipped %d block entity NBT records without valid int32 coordinates in subchunk packet %v", invalidBlockEntities, pk.Position)
	}
	for pos, ch := range chunks {
		w.worldState.StoreChunk(pos, ch)
	}
	if w.scripting != nil {
		for pos := range chunks {
			w.scripting.OnChunkData(pos)
		}
	}

	w.mapUI.SchedRedraw()
	return nil
}

// Block entity coordinates are NBT Int tags. A decoded compound alone is not
// sufficient: servers may also append metadata compounds without coordinates.
// Do not invent a position for these records or discard the other chunk data.
func blockEntityPosition(data map[string]any) (cube.Pos, bool) {
	x, validX := data["x"].(int32)
	y, validY := data["y"].(int32)
	z, validZ := data["z"].(int32)
	if !validX || !validY || !validZ {
		return cube.Pos{}, false
	}
	return cube.Pos{int(x), int(y), int(z)}, true
}

package worldstate

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"slices"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const seatLift = 1.22

func (w *World) entityLinksLocked(id entity.UniqueID) []int64 {
	var links []int64
	for rider := range w.memState.entityLinks[id] {
		ent := w.memState.entities[w.memState.uniqueIDsToRuntimeIDs[rider]]
		if ent != nil && ent.UniqueID == rider && !w.skipEntity(ent.EntityType) {
			links = append(links, rider)
		}
	}
	slices.Sort(links)
	return links
}

func (w *World) mountSeatsLocked() map[string][]mgl32.Vec3 {
	out := make(map[string][]mgl32.Vec3)
	for ridden, riders := range w.memState.entityLinks {
		if len(riders) == 0 {
			continue
		}
		mount := w.memState.entities[w.memState.uniqueIDsToRuntimeIDs[ridden]]
		if mount == nil || mount.UniqueID != ridden || strings.HasPrefix(mount.EntityType, "minecraft:") {
			continue
		}
		scale := float32(1)
		if s, ok := mount.Metadata[protocol.EntityDataKeyScale].(float32); ok && s > 0 {
			scale = s
		}
		ids := w.entityLinksLocked(ridden)
		var seats []mgl32.Vec3
		for _, id := range ids {
			rider := w.memState.entities[w.memState.uniqueIDsToRuntimeIDs[id]]
			if rider == nil || rider.UniqueID != id {
				continue
			}
			off, _ := rider.Metadata[protocol.EntityDataKeySeatOffset].(mgl32.Vec3)

			seats = append(seats, mgl32.Vec3{off.X() / scale, (off.Y() - seatLift) / scale, off.Z() / scale})
		}
		if len(seats) > len(out[mount.EntityType]) {
			out[mount.EntityType] = seats
		}
	}
	return out
}

func rideable(seats []mgl32.Vec3) map[string]any {
	list := make([]any, len(seats))
	for i, s := range seats {
		list[i] = map[string]any{"position": []float32{s.X(), s.Y(), s.Z()}}
	}
	return map[string]any{
		"seat_count":              len(seats),
		"seats":                   list,
		"pull_in_entities":        false,
		"crouching_skip_interact": true,
	}
}

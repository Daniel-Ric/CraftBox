package proxy

import (
	"math"
	"sync"
	"time"

	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// The upstream connection finishes its own spawn before the real Minecraft
// client. A server may already teleport that player while Minecraft is loading.
// Keep the wire observation used for StartGame separate from the ordered relay
// queue: observing a newer packet must not replay it ahead of older relay data.
type spawnPlacement struct {
	mu        sync.Mutex
	runtimeID uint64
	known     bool
	initial   *packet.MovePlayer
	pending   *packet.MovePlayer
	readyAt   time.Time
	corrected bool
}

func (s *Session) configureSpawn(runtimeID uint64) {
	s.spawnPlacement.mu.Lock()
	defer s.spawnPlacement.mu.Unlock()
	s.spawnPlacement.runtimeID = runtimeID
	s.spawnPlacement.known = true
}

func (s *Session) observeSpawnPlacement(pk packet.Packet) {
	if s.Client == nil {
		return
	}
	p := &s.spawnPlacement
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.readyAt.IsZero() {
		return
	}
	switch pk := pk.(type) {
	case *packet.StartGame:
		p.runtimeID, p.known = pk.EntityRuntimeID, true
	case *packet.MovePlayer:
		if !p.known || pk.EntityRuntimeID != p.runtimeID || pk.Mode == packet.MoveModeRotation {
			return
		}
		for _, value := range pk.Position {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return
			}
		}
		copy := *pk
		p.initial = &copy
	}
}

func (s *Session) spawnGameData(data minecraft.GameData) minecraft.GameData {
	p := &s.spawnPlacement
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.initial != nil && p.initial.EntityRuntimeID == data.EntityRuntimeID {
		p.corrected = data.PlayerPosition != p.initial.Position
		data.PlayerPosition = p.initial.Position
		data.Pitch, data.Yaw = p.initial.Pitch, p.initial.Yaw
	}
	return data
}

func (s *Session) holdSpawnMovement(pk packet.Packet) bool {
	move, ok := pk.(*packet.MovePlayer)
	if !ok || s.Client == nil {
		return false
	}
	p := &s.spawnPlacement
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.known || move.EntityRuntimeID != p.runtimeID || !p.readyAt.IsZero() {
		return false
	}
	copy := *move
	if copy.Mode == packet.MoveModeRotation && p.pending != nil {
		p.pending.Pitch, p.pending.Yaw, p.pending.HeadYaw = copy.Pitch, copy.Yaw, copy.HeadYaw
	} else {
		p.pending = &copy
	}
	return true
}

func (s *Session) releaseSpawnPlacement() error {
	if s.Client == nil {
		return nil
	}
	p := &s.spawnPlacement
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.readyAt.IsZero() {
		return nil
	}
	held := p.pending != nil
	if p.pending != nil {
		if err := s.Client.WritePacket(p.pending); err != nil {
			return err
		}
		if err := s.Client.Flush(); err != nil {
			return err
		}
		p.pending = nil
	}
	p.readyAt = time.Now()
	diagnostics.Record("info", "connection", "connection.spawn_ready", "Minecraft client spawn completed", map[string]any{"startPositionCorrected": p.corrected, "heldPlayerMovement": held})
	return nil
}

func (s *Session) uninitialisedMovement(pk packet.Packet, at time.Time) bool {
	if s.Client == nil {
		return false
	}
	p := &s.spawnPlacement
	p.mu.Lock()
	defer p.mu.Unlock()
	switch pk := pk.(type) {
	case *packet.PlayerAuthInput:
	case *packet.MovePlayer:
		if !p.known || pk.EntityRuntimeID != p.runtimeID {
			return false
		}
	default:
		return false
	}
	// Also reject pre-spawn packets still buffered when spawn completes.
	return p.readyAt.IsZero() || at.Before(p.readyAt)
}

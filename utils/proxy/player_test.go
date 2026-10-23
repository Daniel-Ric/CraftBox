package proxy

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"testing"
)

func TestStartGameInitialisesPlayerPosition(t *testing.T) {
	p := Player{}
	position := mgl32.Vec3{2048, 75, -512}
	p.handlePackets(&packet.StartGame{EntityRuntimeID: 42, PlayerPosition: position})
	if p.Position != position || p.RuntimeID != 42 {
		t.Fatalf("spawn position not retained: %+v", p)
	}
}

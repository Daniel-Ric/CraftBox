package worlds

import (
	"strings"
	"testing"
	"time"

	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

func TestFailedStartGameDoesNotPublishReadyState(t *testing.T) {
	w := &worldsHandler{log: logrus.NewEntry(logrus.New()), serverState: serverState{behaviorPack: behaviourpack.New("test")}}
	_, err := w.packetHandlerPreLogin(&packet.StartGame{Blocks: []protocol.BlockEntry{{Name: "test:block", Properties: map[string]any{
		"traits": []any{map[string]any{"name": "minecraft:future_trait", "enabled_states": int32(1)}},
	}}}}, false, time.Now())
	if err == nil {
		t.Fatal("expected unsupported trait error")
	}
	if w.serverState.haveStartGame {
		t.Fatal("failed StartGame was marked ready")
	}
}

func TestChunkBeforeStartGameReturnsError(t *testing.T) {
	w := &worldsHandler{}
	err := w.handleLevelChunk(&packet.LevelChunk{RawPayload: []byte{0}}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "StartGame") {
		t.Fatalf("got %v", err)
	}
}

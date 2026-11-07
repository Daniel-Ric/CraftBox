package leveldat

import (
	"github.com/df-mc/dragonfly/server/world"
	"testing"
)

func TestSpectatorUsesBedrockDiskID(t *testing.T) {
	var data Data
	data.FillDefault()
	data.GameType = 6
	settings := data.Settings()
	if settings.DefaultGameMode != world.GameModeSpectator {
		t.Fatal("Bedrock spectator read as a different game mode")
	}
	data.PutSettings(settings)
	if data.GameType != 6 {
		t.Fatal("native spectator ID leaked into Bedrock level.dat", data.GameType)
	}
}

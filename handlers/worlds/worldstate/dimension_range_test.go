package worldstate

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDimensionDefinitionRange(t *testing.T) {
	for _, tt := range []struct {
		name            string
		minimum, height int32
		want            cube.Range
	}{
		{"negative minimum", -64, 384, cube.Range{-64, 319}},
		{"positive minimum", 80, 128, cube.Range{80, 207}},
		{"zero minimum", 0, 256, cube.Range{0, 255}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := &World{dimensionDefinitions: map[int]protocol.DimensionDefinition{
				0: {MinimumY: tt.minimum, HeightRange: tt.height},
			}}
			w.SetDimension(world.Overworld)
			if got := w.Range(); got != tt.want {
				t.Fatalf("dimension range = %v, want %v", got, tt.want)
			}
			w.SetDimension(world.Nether)
			if got := w.Range(); got != world.Nether.Range() {
				t.Fatalf("dimension without definition retained override: %v", got)
			}
		})
	}
}

package worlds

import (
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"math"
	"slices"
	"testing"
)

func TestStructureCornersUseFeetCoordinates(t *testing.T) {
	for _, c := range []struct {
		position mgl32.Vec3
		want     structure.Vec3
	}{
		{mgl32.Vec3{66.7, 47.62, -36.2}, structure.Vec3{66, 46, -37}},
		{mgl32.Vec3{-0.1, -3.38, 0.9}, structure.Vec3{-1, -5, 0}},
	} {
		got, err := structureFeetPosition(c.position)
		if err != nil || got != c.want {
			t.Fatalf("%v: got %v, %v; want %v", c.position, got, err, c.want)
		}
	}
	if _, err := structureFeetPosition(mgl32.Vec3{0, float32(math.NaN()), 0}); err == nil {
		t.Fatal("accepted invalid position")
	}
}

func TestStructureCommandsHaveLocalTypedCompletions(t *testing.T) {
	w := &worldsHandler{}
	p := &packet.AvailableCommands{}
	local := w.completeCommands(p, []protocol.Command{{Name: "cb"}, {Name: "blueprint"}})
	for _, word := range []string{"structure", "pos1", "pos2", "save", "status", "cancel", "open", "list"} {
		if !slices.Contains(p.EnumValues, word) {
			t.Fatal("missing completion", word)
		}
	}
	if len(local[1].Overloads) < 4 {
		t.Fatal("blueprint alias lacks typed overloads")
	}
	if len(local[1].Overloads[2].Parameters) != 4 {
		t.Fatal("corner coordinate completion missing")
	}
}

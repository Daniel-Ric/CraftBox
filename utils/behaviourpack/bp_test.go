package behaviourpack

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"testing"
)

func TestEntityOnlyPackHasContent(t *testing.T) {
	bp := New("test")
	bp.AddEntity("test:mob", nil, protocol.NewEntityMetadata(), nil)
	if !bp.HasContent() {
		t.Fatal("entity-only behavior pack must be exported")
	}
}

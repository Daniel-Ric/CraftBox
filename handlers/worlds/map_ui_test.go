package worlds

import (
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/go-gl/mathgl/mgl32"
	"testing"
)

func TestRedrawPublishesPlayerPositionForDesktopCentering(t *testing.T) {
	position := mgl32.Vec3{2048, 75, -512}
	var got *messages.EventPlayerPosition
	messages.SetEventHandler(func(event any) error {
		if p, ok := event.(*messages.EventPlayerPosition); ok {
			got = p
		}
		return nil
	})
	t.Cleanup(func() { messages.SetEventHandler(func(any) error { return nil }) })
	m := NewMapUI(&worldsHandler{session: &proxy.Session{Player: proxy.Player{Position: position}}})
	close(m.haveColors)
	m.redraw()
	if got == nil || got.Position != position {
		t.Fatalf("player position not published: %+v", got)
	}
}

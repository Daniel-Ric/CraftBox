package worlds

import (
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/go-gl/mathgl/mgl32"
	"testing"
)

func TestMapMarkerCommandsAndExportIsolation(t *testing.T) {
	m := NewMapUI(nil)
	close(m.haveColors)
	w := &worldsHandler{mapUI: m, session: &proxy.Session{}}
	if m.MarkerEnabled() {
		t.Fatal("marker must be opt-in")
	}
	if !w.commandPosition([]string{"on"}) || !m.MarkerEnabled() {
		t.Fatal("enable failed")
	}
	m.redrawAt(mgl32.Vec3{-0.5, 70, 0.5})
	if m.mapImage.RGBAAt(64, 64).A == 0 {
		t.Fatal("no visible marker")
	}
	for _, b := range m.ToImage().Pix {
		if b != 0 {
			t.Fatal("marker leaked into saved map")
		}
	}
	if w.commandPosition([]string{"invalid"}) || !m.MarkerEnabled() {
		t.Fatal("invalid command mutated state")
	}
	if !w.commandPosition([]string{"off"}) {
		t.Fatal("disable failed")
	}
	m.redrawAt(mgl32.Vec3{})
	if m.mapImage.RGBAAt(64, 64).A != 0 {
		t.Fatal("marker persisted")
	}
}

func TestConnectionCommandValidation(t *testing.T) {
	w := &worldsHandler{session: &proxy.Session{}}
	if !w.commandRoot([]string{"connection"}) {
		t.Fatal("connection command rejected")
	}
	if w.commandRoot([]string{"connection", "extra"}) {
		t.Fatal("unexpected arguments accepted")
	}
}

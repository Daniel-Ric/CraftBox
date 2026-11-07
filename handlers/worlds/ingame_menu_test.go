package worlds

import (
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type menuClient struct {
	minecraft.IConn
	writes []packet.Packet
}

func (c *menuClient) WritePacket(pk packet.Packet) error {
	c.writes = append(c.writes, pk)
	return nil
}

func (c *menuClient) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 19132}
}

func TestMenuCommandOpensBedrockForm(t *testing.T) {
	c := &menuClient{}
	w := &worldsHandler{session: &proxy.Session{Client: c}}
	if !w.commandRoot([]string{"menu"}) {
		t.Fatal("/cb menu was rejected")
	}
	if len(c.writes) != 1 {
		t.Fatalf("expected one form, got %d packets", len(c.writes))
	}
	form, ok := c.writes[0].(*packet.ModalFormRequest)
	if !ok {
		t.Fatalf("expected Bedrock form, got %T", c.writes[0])
	}
	var data struct {
		Type    string `json:"type"`
		Buttons []struct {
			Text string `json:"text"`
		} `json:"buttons"`
	}
	if err := json.Unmarshal(form.FormData, &data); err != nil || data.Type != "form" || len(data.Buttons) < 5 {
		t.Fatalf("invalid menu: %s (%v)", form.FormData, err)
	}
	if w.commandRoot([]string{"menu", "extra"}) {
		t.Fatal("menu accepted unexpected arguments")
	}
}

func TestPositionCommandHasSeparateMapToggle(t *testing.T) {
	m := NewMapUI(nil)
	w := &worldsHandler{session: &proxy.Session{}, mapUI: m}
	if !w.commandRoot([]string{"position", "on"}) || !m.MarkerEnabled() {
		t.Fatal("/cb position on did not enable the position marker")
	}
	if w.commandRoot([]string{"map", "marker", "off"}) || !m.MarkerEnabled() {
		t.Fatal("the old map overload still controls the marker")
	}
	if !w.commandRoot([]string{"position", "off"}) || m.MarkerEnabled() {
		t.Fatal("/cb position off did not disable the marker")
	}
}

func TestMapFormValidatesBeforeChangingAnySetting(t *testing.T) {
	c := &menuClient{}
	w := &worldsHandler{session: &proxy.Session{Client: c}}
	w.mapUI = NewMapUI(w)
	for _, data := range []string{
		`[null,false,true,99]`, `[null,false,true,-1]`,
		`[null,false,true,null]`, `[null,false,"true",2]`,
		`[null,false,true,1.5]`, `[null,false,true]`, `null`,
	} {
		if w.applyMapMenu(json.RawMessage(data)) {
			t.Fatalf("accepted invalid response %s", data)
		}
		if !w.mapUI.Enabled() || w.mapUI.MarkerEnabled() || w.mapUI.Zoom() != 16 {
			t.Fatalf("invalid response partially applied: %s", data)
		}
	}
	if !w.applyMapMenu(json.RawMessage(`[null,false,true,2]`)) || w.mapUI.Enabled() || !w.mapUI.MarkerEnabled() || w.mapUI.Zoom() != 4 {
		t.Fatal("valid map settings not applied")
	}
	if len(c.writes) != 1 {
		t.Fatal("map visibility change did not update Minecraft offhand")
	}
}

func TestMenuPagesUseBedrockControlsWithoutImages(t *testing.T) {
	c := &menuClient{}
	w := &worldsHandler{session: &proxy.Session{Client: c}, worldState: &worldstate.World{ID: "menu-test", Name: "Test capture"}}
	w.mapUI = NewMapUI(w)
	w.settings.ExcludedMobs = []string{"minecraft:zombie"}
	for _, show := range []func(){
		func() { w.commandMenu(nil) }, w.captureMenu, w.mapMenu, w.mapSettingsMenu,
		w.worldOptionsMenu, w.markerMenu, w.pointsMenu, w.blueprintMenu, w.cornerMenu, w.regionMenu,
		func() { w.exclusionsMenu(0) },
		w.quickMenu, w.playerMenu, w.connectionMenu,
		func() { w.pointBrowserMenu("", "", 0) },
		func() { w.pointFilterMenu("", "") },
		func() { w.entityBrowserMenu("", 0) },
		func() { w.regionBrowserMenu(0) },
		func() { w.searchMenu("Search", "", func(string) {}) },
	} {
		show()
		form, ok := c.writes[len(c.writes)-1].(*packet.ModalFormRequest)
		if !ok {
			t.Fatalf("page sent %T instead of a form", c.writes[len(c.writes)-1])
		}
		var page map[string]any
		if err := json.Unmarshal(form.FormData, &page); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(page["title"].(string), "§8") {
			t.Fatal("title does not use dark text")
		}
		if buttons, ok := page["buttons"].([]any); ok {
			for _, b := range buttons {
				button := b.(map[string]any)
				if len(button) != 1 || !strings.HasPrefix(button["text"].(string), "§") {
					t.Fatalf("uncolored button or unwanted image: %v", button)
				}
			}
		}
		if content, ok := page["content"].([]any); ok {
			for _, item := range content {
				control := item.(map[string]any)
				if !strings.HasPrefix(control["text"].(string), "§8") {
					t.Fatalf("control text is not dark: %v", control)
				}
				if _, ok := control["image"]; ok {
					t.Fatal("image in form")
				}
			}
		}
	}
}

func TestMenuAndPositionHaveUnambiguousCompletions(t *testing.T) {
	w := &worldsHandler{}
	definitions := &packet.AvailableCommands{}
	commands := w.completeCommands(definitions, []protocol.Command{{Name: "cb"}})
	counts := map[string]int{}
	for _, overload := range commands[0].Overloads {
		if len(overload.Parameters) == 0 {
			continue
		}
		index := overload.Parameters[0].Type & 0xffff
		for _, value := range definitions.Enums[index].ValueIndices {
			counts[definitions.EnumValues[value]]++
		}
	}
	for _, name := range []string{"menu", "map", "position"} {
		if counts[name] != 1 {
			t.Fatalf("%s has %d overloads", name, counts[name])
		}
	}
}

func TestStaleWorldMenuDoesNotModifyTheNewCapture(t *testing.T) {
	w := &worldsHandler{session: &proxy.Session{}, worldState: &worldstate.World{ID: "new-capture", Name: "Keep this name"}}
	called := false
	w.submitMenuForm("old-capture", func(json.RawMessage) { called = true; w.worldState.Name = "Wrong capture" }, json.RawMessage("true"))
	if called || w.worldState.Name != "Keep this name" {
		t.Fatal("stale form modified the new capture")
	}
	w.submitMenuForm("new-capture", func(json.RawMessage) { called = true }, json.RawMessage("true"))
	if !called {
		t.Fatal("current form was rejected")
	}
}

func TestMenuDashboardShowsCurrentCapture(t *testing.T) {
	c := &menuClient{}
	w := &worldsHandler{session: &proxy.Session{Client: c}, worldState: &worldstate.World{ID: "dashboard", Name: "My capture"}}
	w.commandMenu(nil)
	var page struct {
		Content string `json:"content"`
		Buttons []struct {
			Text string `json:"text"`
		} `json:"buttons"`
	}
	json.Unmarshal(c.writes[0].(*packet.ModalFormRequest).FormData, &page)
	if !strings.Contains(page.Content, "My capture") {
		t.Fatal("dashboard omits the active capture")
	}
	found := false
	for _, button := range page.Buttons {
		found = found || strings.Contains(button.Text, "Quick actions")
	}
	if !found {
		t.Fatal("dashboard has no quick actions")
	}
}

func TestBlueprintMenuShowsSelectionDimensions(t *testing.T) {
	c := &menuClient{}
	w := &worldsHandler{session: &proxy.Session{Client: c}, worldState: &worldstate.World{ID: "selection", Name: "Capture"}}
	a, b := structure.Vec3{1, 2, 3}, structure.Vec3{4, 6, 8}
	w.commandStructureSelection = structureCommandSelection{worldID: "selection", first: &a, second: &b}
	w.blueprintMenu()
	var page struct {
		Content string `json:"content"`
	}
	json.Unmarshal(c.writes[0].(*packet.ModalFormRequest).FormData, &page)
	if !strings.Contains(page.Content, "4 x 5 x 6") {
		t.Fatalf("selection dimensions missing: %s", page.Content)
	}
}

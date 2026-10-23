package desktop

import (
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorldMetadataAPIValidatesAndKeepsEditsSeparate(t *testing.T) {
	messages.RegisterMetadataEditor("source", func(request messages.EditWorld) error { messages.StoreWorldMetadata(request); return nil })
	defer messages.UnregisterWorldEditor("source")
	a := NewApp()
	a.state.Worlds = []SavedWorld{{ID: "source", Name: "Lobby", State: "Capturing"}, {ID: "region", ParentID: "source", Name: "Island", State: "Capturing"}}
	a.worldTiles["source"] = map[[2]int32]Tile{{0, 0}: {X: 0, Z: 0}}
	a.worldTiles["region"] = map[[2]int32]Tile{{4, 4}: {X: 4, Z: 4}}
	handler := a.mapWindowHandler("secret")
	for _, tc := range []struct {
		body, token, origin string
		status              int
	}{{`{"worldId":"region","spawn":[65,80,65]}`, "secret", "", 204}, {`{"worldId":"source","spawn":[65,80,65]}`, "secret", "", 409}, {`{"worldId":"source","spawn":[1]}`, "secret", "", 400}, {`{"worldId":"source","name":"New lobby"}`, "secret", "", 204}, {`{"worldId":"source","name":"bad"}`, "wrong", "", 403}, {`{"worldId":"source","name":"bad"}`, "secret", "browser", 403}} {
		req := httptest.NewRequest(http.MethodPost, "/world-metadata", strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.token)
		req.Header.Set("Origin", tc.origin)
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, req)
		if out.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.body, out.Code, out.Body.String())
		}
	}
	if messages.WorldMetadata("source").Spawn != nil || messages.WorldMetadata("region").Spawn == nil {
		t.Fatal("spawn crossed worlds")
	}
	if a.state.Worlds[0].Name != "New lobby" {
		t.Fatal("name not updated")
	}
}
func TestFinishedNetherRejectsSpawnOutsideItsHeight(t *testing.T) {
	a := NewApp()
	a.state.Worlds = []SavedWorld{{ID: "nether", State: "finished", HeightRange: [2]int{0, 127}}}
	a.worldTiles["nether"] = map[[2]int32]Tile{{0, 0}: {X: 0, Z: 0}}
	for _, y := range []int32{-1, 127, 128, 200} {
		spawn := [3]int32{1, y, 1}
		if err := a.EditWorld(messages.EditWorld{WorldID: "nether", Spawn: &spawn}); err == nil || !strings.Contains(err.Error(), "dimension height") {
			t.Fatalf("Y%d must fail height validation: %v", y, err)
		}
	}
}

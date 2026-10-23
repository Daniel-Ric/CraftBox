package desktop

import (
	"bytes"
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"image"
	"net/http/httptest"
	"testing"
)

func TestRegionProjectionMovesExistingAndFutureTilesWithoutMixingVisits(t *testing.T) {
	for _, move := range []bool{true, false} {
		a := NewApp()
		tile := messages.MapTile{Pos: protocol.ChunkPos{-1, 2}, Img: *image.NewRGBA(image.Rect(0, 0, 16, 16))}
		a.HandleEvent(&messages.EventMapTiles{WorldID: "source", Tiles: []messages.MapTile{tile}})
		region := messages.WorldRegion{ID: "region", ParentID: "source", Name: "Island", Move: move, Bounds: messages.ChunkBounds{MinX: -1, MaxX: -1, MinZ: 2, MaxZ: 2}}
		a.HandleEvent(&messages.EventWorldRegionCreated{Region: region})
		if len(a.worldTiles["region"]) != 1 {
			t.Fatal("existing tile not split")
		}
		wantSource := 1
		if move {
			wantSource = 0
		}
		if len(a.worldTiles["source"]) != wantSource {
			t.Fatal("source move/copy incorrect")
		}
		a.HandleEvent(&messages.EventMapTiles{WorldID: "source", Tiles: []messages.MapTile{tile}})
		if len(a.worldTiles["source"]) != wantSource {
			t.Fatal("future tile leaked back")
		}
		a.HandleEvent(&messages.EventMapTiles{WorldID: "another-visit", Tiles: []messages.MapTile{tile}})
		if len(a.worldTiles["another-visit"]) != 1 || len(a.worldTiles["region"]) != 1 {
			t.Fatal("another visit mixed")
		}
		if !a.mapWindowSnapshot(1).Reset {
			t.Fatal("split did not invalidate retained source tiles")
		}
	}
}

func TestWorldRegionEndpointRequiresCapabilityAndRejectsInvalidRequests(t *testing.T) {
	a := NewApp()
	calls := 0
	messages.RegisterWorldEditor("source", func(request messages.CreateWorldRegion) (messages.WorldRegion, error) {
		calls++
		return messages.WorldRegion{ID: "new-world", ParentID: request.WorldID, Name: request.Name, Bounds: request.Bounds, Move: request.Move}, nil
	})
	defer messages.UnregisterWorldEditor("source")
	handler := a.mapWindowHandler("secret")
	for _, tc := range []struct {
		token, body, origin string
		want                int
	}{
		{"wrong", `{"worldId":"source","name":"Island","bounds":{"minX":0,"maxX":1,"minZ":0,"maxZ":1},"move":true}`, "", 403},
		{"secret", `{"worldId":"source","name":"Island","bounds":{"minX":0,"maxX":1,"minZ":0,"maxZ":1},"move":true}`, "https://browser.invalid", 403},
		{"secret", `{"worldId":"source","name":""}`, "", 400},
		{"secret", `{"worldId":"source","name":"Island","unknown":true}`, "", 400},
		{"secret", `{"worldId":"finished","name":"Island","bounds":{"minX":0,"maxX":1,"minZ":0,"maxZ":1},"move":true}`, "", 409},
		{"secret", `{"worldId":"source","name":"Island","bounds":{"minX":0,"maxX":1,"minZ":0,"maxZ":1},"move":true}`, "", 201},
	} {
		req := httptest.NewRequest("POST", "http://127.0.0.1/world-region", bytes.NewBufferString(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.token)
		req.Header.Set("Origin", tc.origin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != tc.want {
			t.Fatalf("%s => %d: %s", tc.body, response.Code, response.Body.String())
		}
		if response.Code == 201 {
			var region messages.WorldRegion
			if err := json.Unmarshal(response.Body.Bytes(), &region); err != nil || region.ID != "new-world" {
				t.Fatal("missing created world")
			}
		}
	}
	if calls != 1 {
		t.Fatalf("invalid requests changed the world: %d calls", calls)
	}
}

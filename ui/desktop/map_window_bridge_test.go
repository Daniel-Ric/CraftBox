package desktop

import (
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"image"
	"net/http/httptest"
	"testing"
)

func TestMapWindowBridgeIsReadOnlyAndDoesNotExposeAuthCodes(t *testing.T) {
	a := NewApp()
	a.state.AuthCode = "private-code"
	a.state.AuthURI = "private-uri"
	a.state.Running = true
	handler := a.mapWindowHandler("secret")
	for _, tc := range []struct {
		method, token, origin string
		status                int
	}{{"GET", "", "", 403}, {"GET", "wrong", "", 403}, {"POST", "secret", "", 405}, {"GET", "secret", "https://example.com", 403}, {"GET", "secret", "", 200}} {
		req := httptest.NewRequest(tc.method, "http://127.0.0.1/map-window", nil)
		req.Header.Set("Authorization", "Bearer "+tc.token)
		req.Header.Set("Origin", tc.origin)
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, req)
		if out.Code != tc.status {
			t.Fatalf("%+v returned %d", tc, out.Code)
		}
		if out.Code == 200 {
			var result MapWindowSnapshot
			if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.State.AuthCode != "" || result.State.AuthURI != "" {
				t.Fatal("auth code exposed")
			}
		}
	}
	if !a.state.Running {
		t.Fatal("map requests stopped capture")
	}
}

func TestMapWindowBridgeSendsTileDeltasAndSessionReset(t *testing.T) {
	a := NewApp()
	tile := messages.MapTile{Pos: protocol.ChunkPos{}, Img: *image.NewRGBA(image.Rect(0, 0, 16, 16))}
	a.HandleEvent(&messages.EventMapTiles{WorldID: "one", Tiles: []messages.MapTile{tile}})
	first := a.mapWindowSnapshot(0)
	if !first.Reset || len(first.Tiles) != 1 {
		t.Fatal("missing initial tiles")
	}
	if next := a.mapWindowSnapshot(first.Sequence); next.Reset || len(next.Tiles) != 0 {
		t.Fatal("unchanged tiles resent")
	}
	a.HandleEvent(&messages.EventMapTiles{WorldID: "two", Tiles: []messages.MapTile{tile}})
	next := a.mapWindowSnapshot(first.Sequence)
	if len(next.Tiles) != 1 || next.Tiles[0].WorldID != "two" {
		t.Fatal("world delta lost")
	}
	a.HandleEvent(&messages.EventResetMap{})
	if reset := a.mapWindowSnapshot(next.Sequence); !reset.Reset || len(reset.Tiles) != 0 {
		t.Fatal("new session retained old map")
	}
}

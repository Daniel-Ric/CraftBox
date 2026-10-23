package desktop

import (
	"context"
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenePageBridgeValidationAndOwnership(t *testing.T) {
	a := sceneApp(t)
	for _, tc := range []struct {
		token, body string
		code        int
	}{
		{"", `{"worldId":"source","page":{}}`, 403},
		{"secret", `{"worldId":"source"}`, 400},
		{"secret", `{"worldId":"source","page":{"path":"x"}}`, 400},
		{"secret", `{"worldId":"source","page":{}} {}`, 400},
		{"secret", `{"worldId":"absent","page":{}}`, 409},
		{"secret", `{"worldId":"source","page":{"limit":1}}`, 200},
	} {
		r := httptest.NewRequest("POST", "/scene-page", strings.NewReader(tc.body))
		r.Header.Set("Authorization", "Bearer "+tc.token)
		w := httptest.NewRecorder()
		a.mapWindowHandler("secret").ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatalf("%s: %d %s", tc.body, w.Code, w.Body.String())
		}
	}
	before, err := a.GetScenePage("source", scene.PageRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	a.addWorldRegion(messages.WorldRegion{ID: "region", ParentID: "source", Move: true, Bounds: messages.ChunkBounds{MinX: 1, MaxX: 1, MinZ: 0, MaxZ: 0}})
	next, err := a.GetScenePage("source", scene.PageRequest{Target: before.Sequence, Cursor: before.Next, Scope: before.Scope})
	if err != nil || !next.Reset || len(next.Chunks) != 1 || next.Chunks[0].X != 0 {
		t.Fatalf("ownership changed: %+v %v", next, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = a.getScenePage(ctx, "source", scene.PageRequest{}); err != context.Canceled {
		t.Fatalf("cancel: %v", err)
	}
}

func TestSceneExportWritesGLBWithoutChangingWorldArchive(t *testing.T) {
	a := sceneApp(t)
	archive := filepath.Join(t.TempDir(), "capture.mcworld")
	if err := os.WriteFile(archive, []byte("original world archive"), 0644); err != nil {
		t.Fatal(err)
	}
	a.state.Worlds = []SavedWorld{{ID: "source", Name: "Capture", Path: archive}}
	for i := 0; i < 2; i++ {
		path, err := a.ExportScene("source")
		if err != nil {
			t.Fatal(err)
		}
		if path != strings.TrimSuffix(archive, ".mcworld")+".glb" {
			t.Fatal("wrong output destination")
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) < 12 || string(data[:4]) != "glTF" {
			t.Fatal("model not saved", err)
		}
	}
	data, err := os.ReadFile(archive)
	if err != nil || string(data) != "original world archive" {
		t.Fatal("world archive changed")
	}
}

func TestSceneBridgeRejectsCallerChosenPathsAndMalformedSequences(t *testing.T) {
	a := sceneApp(t)
	for _, tc := range []struct{ method, url, body string }{{"POST", "/scene-export", `{"worldId":"source","path":"arbitrary.glb"}`}, {"GET", "/scene?worldId=source&after=-1", ""}} {
		req := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer secret")
		res := httptest.NewRecorder()
		a.mapWindowHandler("secret").ServeHTTP(res, req)
		if res.Code != 400 {
			t.Fatalf("expected malformed request rejection: %d", res.Code)
		}
	}
}

func sceneApp(t *testing.T) *App {
	t.Helper()
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	store := scene.NewStore(reg, scene.NewCatalog())
	for x := int32(0); x < 2; x++ {
		c := chunk.New(reg, cube.Range{0, 15})
		c.SetBlock(0, 1, 0, 0, stone)
		store.Put(world.ChunkPos{x, 0}, c)
	}
	a := NewApp()
	t.Cleanup(a.closeScenes)
	a.HandleEvent(&messages.EventSceneAvailable{WorldID: "source", Store: store})
	return a
}
func TestSceneBridgeAuthAndWorldIsolation(t *testing.T) {
	a := sceneApp(t)
	for _, tc := range []struct {
		token, id string
		code      int
	}{{"", "source", 403}, {"secret", "absent", 409}, {"secret", "source", 200}} {
		r := httptest.NewRequest(http.MethodGet, "/scene?worldId="+tc.id, nil)
		r.Header.Set("Authorization", "Bearer "+tc.token)
		w := httptest.NewRecorder()
		a.mapWindowHandler("secret").ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatalf("%s: %d %s", tc.id, w.Code, w.Body.String())
		}
		if tc.code == 200 {
			var got scene.Snapshot
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got.Chunks) != 2 {
				t.Fatalf("scene: %v", err)
			}
		}
	}
}
func TestSceneMovedRegionResetsAndSeparatesSource(t *testing.T) {
	a := sceneApp(t)
	before, _ := a.GetScene("source", 0)
	a.addWorldRegion(messages.WorldRegion{ID: "region", ParentID: "source", Name: "Selection", Move: true, Bounds: messages.ChunkBounds{MinX: 1, MaxX: 1, MinZ: 0, MaxZ: 0}})
	parent, err := a.GetScene("source", before.Sequence)
	if err != nil {
		t.Fatal(err)
	}
	region, err := a.GetScene("region", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !parent.Reset || len(parent.Chunks) != 1 || parent.Chunks[0].X != 0 || len(region.Chunks) != 1 || region.Chunks[0].X != 1 {
		t.Fatal("region data leaked or reset was lost")
	}
}

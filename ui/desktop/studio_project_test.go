package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStudioPendingOpenPreservesActiveUntilSceneRequested(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.craftbox")
	page := scene.Page{Sequence: 1, AtlasSize: 2048, Total: 1, Complete: true, Chunks: []scene.PackedChunk{{X: 7}}}
	if err := writeStudioArchive(path, json.RawMessage(`{}`), func(scene.PageRequest) (scene.Page, error) { return page, nil }); err != nil {
		t.Fatal(err)
	}
	archive, err := readStudioArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	old := &renderDocument{info: RenderWorld{ID: "old"}}
	pending := &renderDocument{info: RenderWorld{ID: "pending"}, project: archive}
	a := &App{ctx: context.Background(), renderDocument: old, studioProjectPath: "old.craftbox", pendingStudioDocument: pending, pendingStudioPath: path}

	a.CloseRenderWorld("pending")
	if a.renderDocument != old || a.studioProjectPath != "old.craftbox" || a.pendingStudioDocument != nil {
		t.Fatal("cancelled pending open changed the active project or leaked pending source")
	}
	archive, err = readStudioArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	pending.project = archive
	a.pendingStudioDocument = pending
	a.pendingStudioPath = path
	defer a.CloseRenderWorld("")
	restored, err := a.GetRenderScenePage("pending", scene.PageRequest{})
	if err != nil || len(restored.Chunks) != 1 || a.renderDocument != pending || a.pendingStudioDocument != nil || a.studioProjectPath != path {
		t.Fatalf("valid project was not adopted on first scene request: %v", err)
	}
}

func TestStudioCloseGateDoesNotLoseDirtyStateWhenAnotherCloseDialogCancels(t *testing.T) {
	events := 0
	a := &App{studioProjectDirty: true, emit: func(name string, _ any) {
		if name == "studio:close-requested" {
			events++
		}
	}}
	if !a.shouldPreventStudioClose() || events != 1 {
		t.Fatal("dirty native close bypassed the frontend dialog")
	}
	a.studioCloseApproved = true
	if a.shouldPreventStudioClose() {
		t.Fatal("approved close blocked")
	}
	if !a.shouldPreventStudioClose() || events != 2 {
		t.Fatal("cancelled close lost dirty state")
	}
}

func TestStudioProjectRealSceneRoundtrip(t *testing.T) {
	root := os.Getenv("CRAFTBOX_STUDIO_FIXTURE_DIR")
	if root == "" {
		t.Skip("real terrain fixture not requested")
	}
	raw, err := os.ReadFile(filepath.Join(root, "render-editor", "terrain-pages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pages []scene.Page
	if err = json.Unmarshal(raw, &pages); err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(root, "studio-workflow", "saved-document.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "studio-workflow", "roundtrip.craftbox")
	index := 0
	if err = writeStudioArchive(path, document, func(scene.PageRequest) (scene.Page, error) { page := pages[index]; index++; return page, nil }); err != nil {
		t.Fatal(err)
	}
	archive, err := readStudioArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	request := scene.PageRequest{}
	loaded := 0
	for i := range pages {
		page, e := archive.Page(request)
		if e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(page.Chunks, pages[i].Chunks) || !reflect.DeepEqual(page.Patches, pages[i].Patches) {
			t.Fatal("terrain, emitters or atlas patches changed")
		}
		loaded += len(page.Chunks)
		request.Cursor = page.Next
	}
	if loaded != 48 {
		t.Fatalf("got %d chunks", loaded)
	}
	archive.Close()

	index = 0
	if err = writeStudioArchive(path, document, func(scene.PageRequest) (scene.Page, error) { page := pages[index]; index++; return page, nil }); err != nil {
		t.Fatal(err)
	}
}

func TestStudioProjectRoundtripAndFailedSavePreservesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.craftbox")
	document := json.RawMessage(`{"name":"Test","settings":{"exposure":1.2},"sequence":{"keys":[]}}`)
	page := scene.Page{Sequence: 3, AtlasSize: 2048, Total: 1, Complete: true, Chunks: []scene.PackedChunk{{X: 7, Z: -9}}}
	if err := writeStudioArchive(path, document, func(scene.PageRequest) (scene.Page, error) { return page, nil }); err != nil {
		t.Fatal(err)
	}
	archive, err := readStudioArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if string(archive.Document) != string(document) {
		t.Fatal("document changed")
	}
	restored, err := archive.Page(scene.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Chunks) != 1 || restored.Chunks[0].Z != -9 || !restored.Complete {
		t.Fatal("map changed")
	}
	before, _ := os.ReadFile(path)
	err = writeStudioArchive(path, document, func(scene.PageRequest) (scene.Page, error) { return scene.Page{}, errors.New("capture failed") })
	if err == nil {
		t.Fatal("expected failed save")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed save destroyed previous project")
	}
}

func TestStudioProjectRejectsIncompleteAndChangingScene(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.craftbox")
	calls := 0
	err := writeStudioArchive(path, json.RawMessage(`{}`), func(scene.PageRequest) (scene.Page, error) {
		calls++
		return scene.Page{Sequence: uint64(calls), AtlasSize: 2048, Total: 2, Next: &[2]int32{int32(calls), 0}, Complete: calls == 2}, nil
	})
	if err == nil {
		t.Fatal("changing map must not be saved as a consistent snapshot")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("failed project became visible")
	}
}
func TestStudioProjectLargeCaptureCanReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.craftbox")
	loaded := 0
	err := writeStudioArchive(path, json.RawMessage(`{}`), func(scene.PageRequest) (scene.Page, error) {
		chunks := make([]scene.PackedChunk, 8)
		for i := range chunks {
			chunks[i].X = int32(loaded + i)
		}
		loaded += 8
		return scene.Page{Sequence: 1, AtlasSize: 2048, Total: 5000, Chunks: chunks, Complete: loaded == 5000, Next: &[2]int32{int32(loaded), 0}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := readStudioArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if archive.Total != 5000 {
		t.Fatal("large capture lost chunks")
	}
}

package worldstate

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"testing"
)

func testPack(t *testing.T) *resource.Pack {
	t.Helper()
	id := uuid.NewString()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	manifest, _ := json.Marshal(map[string]any{"format_version": 2, "header": map[string]any{"name": "RP", "description": "test", "uuid": id, "version": []int{1, 0, 0}}, "modules": []map[string]any{{"type": "resources", "uuid": uuid.NewString(), "version": []int{1, 0, 0}}}})
	for name, data := range map[string][]byte{"manifest.json": manifest, "textures/test.png": []byte("pixels")} {
		f, _ := z.Create(name)
		f.Write(data)
	}
	z.Close()
	p, err := resource.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestPackActivationOnlyReferencesEmbeddedManifests(t *testing.T) {
	messages.SetEventHandler(func(any) error { return nil })
	w := &World{Folder: t.TempDir(), log: logrus.New().WithField("test", true), ResourcePacks: []*resource.Pack{testPack(t), testPack(t)}}
	if err := w.addResourcePacks(); err != nil {
		t.Fatal(err)
	}
	w.resourcePacksDone = make(chan error)
	close(w.resourcePacksDone)
	if err := w.finalizePacks(func(utils.WriterFS) ([]addedPack, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(w.Folder, "world_resource_packs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var deps []resourcePackDependency
	if err = json.Unmarshal(data, &deps); err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatal(deps)
	}
	for _, dep := range deps {
		filename := filepath.Join(w.Folder, "resource_packs", "RP--"+dep.UUID, "manifest.json")
		raw, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		var manifest resource.Manifest
		if err = json.Unmarshal(raw, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Header.UUID.String() != dep.UUID || manifest.Header.Version != dep.Version {
			t.Fatal("activation mismatch")
		}
	}
}
func TestPackCopyFailureCannotReportSuccess(t *testing.T) {
	messages.SetEventHandler(func(any) error { return nil })
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "resource_packs"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &World{Folder: folder, log: logrus.New().WithField("test", true), ResourcePacks: []*resource.Pack{testPack(t)}}
	if err := w.addResourcePacks(); err == nil {
		t.Fatal("copy failure ignored")
	}
	if len(w.exportedResourcePacks) != 0 {
		t.Fatal("missing pack activated")
	}
}

package localworlds

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

const rpID = "11111111-1111-4111-8111-111111111111"
const bpID = "22222222-2222-4222-8222-222222222222"
const depID = "33333333-3333-4333-8333-333333333333"

func TestCommentedPackManifest(t *testing.T) {
	storage, w := fixture(t)
	refs(t, w.Folder, "resource", rpID)
	folder := filepath.Join(storage, "resource_packs", "commented")
	pack(t, folder, rpID, "resources", nil)
	path := filepath.Join(folder, "manifest.json")
	raw, _ := os.ReadFile(path)
	write(t, path, append([]byte("// Minecraft pack manifest\n"), raw...))
	plan, err := Inspect(context.Background(), w, Options{})
	if err != nil || !plan.Portable {
		t.Fatalf("commented pack was not resolved: %+v %v", plan.Issues, err)
	}
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T) (string, World) {
	t.Helper()
	storage := filepath.Join(t.TempDir(), "private-profile", "com.mojang")
	folder := filepath.Join(storage, "minecraftWorlds", "world")
	data, err := nbt.MarshalEncoding(map[string]any{"LevelName": "My world 世界", "isFromWorldTemplate": uint8(1)}, nbt.LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header, 10)
	binary.LittleEndian.PutUint32(header[4:], uint32(len(data)))
	write(t, filepath.Join(folder, "level.dat"), append(header, data...))
	write(t, filepath.Join(folder, "levelname.txt"), []byte("My world 世界"))
	db, err := leveldb.OpenFile(filepath.Join(folder, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Put([]byte("test-key"), []byte("original-world-data"), nil); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Describe(folder, storage, "Minecraft"); err != nil {
		t.Fatal(err)
	}
	worlds, err := Discover(context.Background(), []Root{{Storage: storage, Name: "Minecraft"}})
	if err != nil || len(worlds) != 1 {
		t.Fatalf("worlds=%v err=%v", worlds, err)
	}
	return storage, worlds[0]
}
func refs(t *testing.T, folder, kind, id string) {
	t.Helper()
	data, _ := json.Marshal([]any{map[string]any{"pack_id": id, "version": []int{1, 0, 0}}})
	write(t, filepath.Join(folder, "world_"+kind+"_packs.json"), data)
}
func pack(t *testing.T, root, id, kind string, deps []any) {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"format_version": 2, "header": map[string]any{"uuid": id, "version": []int{1, 0, 0}, "name": "Pack " + id}, "modules": []any{map[string]any{"type": kind, "uuid": id, "version": []int{1, 0, 0}}}, "dependencies": deps})
	write(t, filepath.Join(root, "manifest.json"), data)
}
func archive(t *testing.T, path string) map[string][]byte {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	out := map[string][]byte{}
	for _, f := range r.File {
		if _, ok := out[f.Name]; ok {
			t.Fatalf("duplicate archive entry %s", f.Name)
		}
		rc, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(rc)
		rc.Close()
		if e != nil {
			t.Fatal(e)
		}
		out[f.Name] = b
	}
	return out
}

func TestExportIncludesWorldPacksAndTransitiveDependencies(t *testing.T) {
	storage, w := fixture(t)
	refs(t, w.Folder, "behavior", bpID)
	refs(t, w.Folder, "resource", rpID)
	pack(t, filepath.Join(storage, "behavior_packs", "addon"), bpID, "script", []any{map[string]any{"uuid": depID, "version": []int{1, 0, 0}}, map[string]any{"module_name": "@minecraft/server", "version": "2.0.0"}})
	write(t, filepath.Join(storage, "behavior_packs", "addon", "scripts", "main.js"), []byte("world logic"))
	pack(t, filepath.Join(storage, "development_behavior_packs", "dependency"), depID, "data", nil)
	pack(t, filepath.Join(w.Folder, "resource_packs", "embedded"), rpID, "resources", nil)
	write(t, filepath.Join(w.Folder, "resource_packs", "embedded", "textures", "block.png"), []byte("texture"))
	write(t, filepath.Join(storage, "token.json"), []byte("PRIVATE-ACCOUNT-TOKEN"))
	plan, err := Inspect(context.Background(), w, Options{})
	if err != nil || !plan.Portable || len(plan.Packs) != 3 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	output := filepath.Join(t.TempDir(), "share.mcworld")
	result, err := Export(context.Background(), w, Options{}, output, plan.Fingerprint, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Portable {
		t.Fatal("share marked incomplete")
	}
	files := archive(t, output)
	restored := filepath.Join(t.TempDir(), "restored")
	for name, data := range files {
		if strings.HasPrefix(name, "db/") {
			write(t, filepath.Join(restored, filepath.FromSlash(name)), data)
		}
	}
	db, err := leveldb.OpenFile(filepath.Join(restored, "db"), nil)
	if err != nil {
		t.Fatalf("exported database cannot be reopened: %v", err)
	}
	value, err := db.Get([]byte("test-key"), nil)
	db.Close()
	if err != nil || string(value) != "original-world-data" {
		t.Fatalf("world data lost: %q %v", value, err)
	}
	for _, name := range []string{"level.dat", "world_behavior_packs.json", "world_resource_packs.json", "resource_packs/embedded/textures/block.png", "behavior_packs/" + bpID + "_1.0.0/scripts/main.js", "behavior_packs/" + depID + "_1.0.0/manifest.json", "craftbox-export-report.json"} {
		if _, ok := files[name]; !ok {
			t.Errorf("missing %s", name)
		}
	}
	original, _ := os.ReadFile(filepath.Join(w.Folder, "level.dat"))
	if !bytes.Equal(original, files["level.dat"]) {
		t.Fatal("world metadata changed")
	}
	for name, data := range files {
		if strings.Contains(name, "token.json") || bytes.Contains(data, []byte("PRIVATE-ACCOUNT-TOKEN")) {
			t.Fatal("account storage leaked")
		}
	}
	if bytes.Contains(files["craftbox-export-report.json"], []byte(storage)) {
		t.Fatal("local profile path leaked into share report")
	}
}
func TestMissingOrWrongVersionPacksBlockShareButAllowExplicitBackup(t *testing.T) {
	storage, w := fixture(t)
	refs(t, w.Folder, "resource", rpID)
	pack(t, filepath.Join(storage, "resource_packs", "wrong"), rpID, "resources", nil)
	manifest := filepath.Join(storage, "resource_packs", "wrong", "manifest.json")
	raw, _ := os.ReadFile(manifest)
	write(t, manifest, bytes.ReplaceAll(raw, []byte("[1,0,0]"), []byte("[2,0,0]")))
	plan, err := Inspect(context.Background(), w, Options{})
	if err != nil || plan.Portable {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	target := filepath.Join(t.TempDir(), "share.mcworld")
	if _, err = Export(context.Background(), w, Options{}, target, "", nil); err == nil {
		t.Fatal("incomplete world exported as portable")
	}
	if _, err = os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("failed share left an archive")
	}
	result, err := Export(context.Background(), w, Options{BackupOnly: true}, target, "", nil)
	if err != nil || result.Portable {
		t.Fatalf("backup=%+v err=%v", result, err)
	}
}
func TestEncryptedMarketplacePackIsPreservedWithoutClaimingItIsPortable(t *testing.T) {
	storage, w := fixture(t)
	refs(t, w.Folder, "resource", rpID)
	root := filepath.Join(storage, "premium_cache", "resource_packs", "premium")
	pack(t, root, rpID, "resources", nil)
	protected := []byte{0, 252, 13, 17, 88, 94}
	write(t, filepath.Join(root, "contents.json"), protected)
	p, err := Inspect(context.Background(), w, Options{})
	if err != nil || p.Portable {
		t.Fatalf("protected plan=%+v err=%v", p, err)
	}
	output := filepath.Join(t.TempDir(), "backup.mcworld")
	if _, err = Export(context.Background(), w, Options{BackupOnly: true}, output, "", nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(archive(t, output)["resource_packs/"+rpID+"_1.0.0/contents.json"], protected) {
		t.Fatal("protection data was changed")
	}
}
func TestCannotCopyAnOpenWorldDatabase(t *testing.T) {
	_, w := fixture(t)
	db, err := leveldb.OpenFile(filepath.Join(w.Folder, "db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = Export(context.Background(), w, Options{}, filepath.Join(t.TempDir(), "busy.mcworld"), "", nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "close") {
		t.Fatalf("expected actionable busy error, got %v", err)
	}
}
func TestCancellationPreservesExistingDestination(t *testing.T) {
	_, w := fixture(t)
	target := filepath.Join(t.TempDir(), "share.mcworld")
	write(t, target, []byte("previous export"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := Export(ctx, w, Options{}, target, "", func(Progress) { cancel() })
	if err == nil {
		t.Fatal("cancel ignored")
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != "previous export" {
		t.Fatal("cancel damaged existing output")
	}
}
func TestChangedWorldRequiresNewInspectionAndOutputCannotBeInsideWorld(t *testing.T) {
	_, w := fixture(t)
	p, err := Inspect(context.Background(), w, Options{})
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(w.Folder, "new.txt"), []byte("changed"))
	if _, err = Export(context.Background(), w, Options{}, filepath.Join(t.TempDir(), "changed.mcworld"), p.Fingerprint, nil); err == nil {
		t.Fatal("stale plan accepted")
	}
	if _, err = Export(context.Background(), w, Options{}, filepath.Join(w.Folder, "share.mcworld"), "", nil); err == nil {
		t.Fatal("archive inside source accepted")
	}
}
func TestGlobalResourcesAreOptionalAndTheirActivationIsIncluded(t *testing.T) {
	storage, w := fixture(t)
	data, _ := json.Marshal([]any{map[string]any{"pack_id": rpID, "version": []int{1, 0, 0}}})
	write(t, filepath.Join(storage, "global_resource_packs.json"), data)
	pack(t, filepath.Join(storage, "resource_packs", "global"), rpID, "resources", nil)
	p, err := Inspect(context.Background(), w, Options{})
	if err != nil || len(p.Packs) != 0 {
		t.Fatal("global resources included implicitly")
	}
	output := filepath.Join(t.TempDir(), "global.mcworld")
	if _, err = Export(context.Background(), w, Options{IncludeGlobal: true}, output, "", nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(archive(t, output)["world_resource_packs.json"], []byte(rpID)) {
		t.Fatal("global texture activation missing")
	}
}

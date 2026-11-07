package structure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func NewCapturedSession(ctx context.Context, root string, store *scene.Store, name string, b Bounds, packs []*resource.Pack, entries []protocol.BlockEntry) (*Session, error) {
	if store == nil {
		return nil, fmt.Errorf("The capture scene is not ready")
	}
	d, err := FromScene(ctx, store, name, b.Normalized())
	if err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp("", "craftbox-live-structure-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)
	for i, p := range packs {
		source, e := utils.ReadablePack(p)
		if e != nil {
			return nil, e
		}
		folder := filepath.Join(temporary, fmt.Sprint(i))
		if e = copyStructurePack(ctx, source, folder); e != nil {
			return nil, e
		}
		raw, e := os.ReadFile(filepath.Join(folder, "manifest.json"))
		if e != nil {
			return nil, e
		}
		var m struct {
			Header  struct{ UUID, Name string }
			Modules []struct{ Type string }
		}
		if e = utils.ParseJson(raw, &m); e != nil {
			return nil, e
		}
		kind := "resource_packs"
		for _, module := range m.Modules {
			if module.Type == "data" {
				kind = "behavior_packs"
			}
		}
		d.Packs = append(d.Packs, Dependency{ID: m.Header.UUID, UUID: m.Header.UUID, Name: m.Header.Name, Kind: kind, Source: folder})
	}
	if len(entries) > 0 {
		folder := filepath.Join(temporary, "captured-block-definitions")
		if err = os.MkdirAll(filepath.Join(folder, "blocks"), 0700); err != nil {
			return nil, err
		}
		id := uuid.NewString()
		manifest := map[string]any{"format_version": 2, "header": map[string]any{"name": "Captured block definitions", "description": "CraftBox captured custom block definitions", "uuid": id, "version": []int{1, 0, 0}, "min_engine_version": []int{1, 21, 0}}, "modules": []map[string]any{{"type": "data", "uuid": uuid.NewString(), "version": []int{1, 0, 0}}}}
		raw, _ := json.Marshal(manifest)
		if err = os.WriteFile(filepath.Join(folder, "manifest.json"), raw, 0600); err != nil {
			return nil, err
		}
		for _, entry := range entries {
			raw, e := behaviourpack.BlockRenderJSON(entry)
			if e != nil {
				return nil, e
			}
			hash := sha256.Sum256([]byte(entry.Name))
			if e = os.WriteFile(filepath.Join(folder, "blocks", hex.EncodeToString(hash[:8])+".json"), raw, 0600); e != nil {
				return nil, e
			}
		}
		d.Packs = append(d.Packs, Dependency{ID: id, UUID: id, Name: "Captured block definitions", Kind: "behavior_packs", Source: folder})
	}
	d.MarkMissingDependencies()
	return NewSession(ctx, root, d, nil)
}
func copyStructurePack(ctx context.Context, source fs.FS, target string) error {
	return fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if path != "." && (!fs.ValidPath(path) || strings.ContainsAny(path, "\\:")) {
			return fmt.Errorf("Invalid dependency path")
		}
		to := filepath.Join(target, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(to, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("Unsupported dependency entry")
		}
		in, err := source.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
}

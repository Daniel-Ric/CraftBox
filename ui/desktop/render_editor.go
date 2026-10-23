package desktop

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RenderWorld struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Chunks int    `json:"chunks"`
}
type renderDocument struct {
	info      RenderWorld
	store     *scene.Store
	temporary string
	project   *studioArchive
}

type renderContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r renderContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func extractRenderArchive(ctx context.Context, path string) (root, temporary string, err error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return "", "", err
	}
	defer z.Close()
	temporary, err = os.MkdirTemp("", "craftbox-render-")
	if err != nil {
		return "", "", err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(temporary)
		}
	}()
	if len(z.File) > 100000 {
		return "", temporary, fmt.Errorf("Archive contains too many files")
	}
	var total uint64
	for _, entry := range z.File {
		if err = ctx.Err(); err != nil {
			return "", temporary, err
		}
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		clean := filepath.Clean(filepath.FromSlash(name))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || strings.Contains(clean, ":") || entry.Mode()&os.ModeSymlink != 0 {
			return "", temporary, fmt.Errorf("Invalid archive path")
		}
		if entry.UncompressedSize64 > 8<<30-total {
			return "", temporary, fmt.Errorf("World archive exceeds 8 GiB")
		}
		total += entry.UncompressedSize64
		target := filepath.Join(temporary, clean)
		if entry.FileInfo().IsDir() {
			err = os.MkdirAll(target, 0700)
		} else {
			err = os.MkdirAll(filepath.Dir(target), 0700)
			if err == nil {
				var input io.ReadCloser
				input, err = entry.Open()
				if err == nil {
					var output *os.File
					output, err = os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
					if err == nil {
						_, err = io.Copy(output, renderContextReader{ctx: ctx, reader: io.LimitReader(input, int64(entry.UncompressedSize64)+1)})
						closeErr := output.Close()
						if err == nil {
							err = closeErr
						}
					}
					input.Close()
				}
			}
		}
		if err != nil {
			return "", temporary, err
		}
	}
	root = temporary
	if _, e := os.Stat(filepath.Join(root, "level.dat")); e != nil {
		entries, _ := os.ReadDir(root)
		if len(entries) == 1 && entries[0].IsDir() {
			root = filepath.Join(root, entries[0].Name())
		}
	}
	return root, temporary, nil
}

func renderAssetCatalog() *scene.Catalog {
	paths := []string{os.Getenv("CRAFT_BOX_3D_ASSETS"), utils.PathData("scene-assets"), filepath.Join("assets", "vanilla-3d")}
	if exe, e := os.Executable(); e == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "assets", "vanilla-3d"))
	}
	for _, path := range paths {
		if path != "" {
			if catalog, e := scene.LoadCatalog(path); e == nil {
				return catalog
			}
		}
	}
	return scene.NewCatalog()
}

func renderPackPaths(root, folder, stack string) ([]string, error) {
	entries, e := os.ReadDir(filepath.Join(root, folder))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	byID := map[string]string{}
	byVersion := map[string]string{}
	all := []string{}
	for _, entry := range entries {
		path := filepath.Join(root, folder, entry.Name())
		all = append(all, path)
		var manifest struct {
			Header struct {
				UUID    string `json:"uuid"`
				Version [3]int `json:"version"`
			} `json:"header"`
		}
		if raw, e := os.ReadFile(filepath.Join(path, "manifest.json")); e == nil && json.Unmarshal(raw, &manifest) == nil {
			byID[manifest.Header.UUID] = path
			byVersion[manifest.Header.UUID+fmt.Sprint(manifest.Header.Version)] = path
		} else if !entry.IsDir() {
			if pack, e := resource.ReadPath(path); e == nil {
				m := pack.Manifest()
				byID[pack.UUID().String()] = path
				byVersion[pack.UUID().String()+fmt.Sprint(m.Header.Version)] = path
			}
		}
	}
	sort.Strings(all)
	var order []struct {
		ID      string  `json:"pack_id"`
		Version *[3]int `json:"version"`
	}
	raw, e := os.ReadFile(filepath.Join(root, stack))
	if e != nil {
		if os.IsNotExist(e) {
			return all, nil
		}
		return nil, e
	}
	if e = json.Unmarshal(raw, &order); e != nil {
		return nil, e
	}
	selected := map[string]bool{}
	ordered := []string{}
	for _, item := range order {
		path := byID[item.ID]
		if item.Version != nil {
			path = byVersion[item.ID+fmt.Sprint(*item.Version)]
		}
		if path == "" {
			return nil, fmt.Errorf("Active pack %s is missing at the selected version", item.ID)
		}
		if !selected[path] {
			selected[path] = true
			ordered = append(ordered, path)
		}
	}
	return ordered, nil
}

func (a *App) importRenderWorld(ctx context.Context, path string) (RenderWorld, error) {
	a.mu.Lock()
	if a.renderCancel != nil {
		a.renderCancel()
	}
	ctx, cancel := context.WithCancel(ctx)
	a.renderCancel = cancel
	a.renderGeneration++
	generation := a.renderGeneration
	a.mu.Unlock()
	defer cancel()
	root, temporary := path, ""
	if strings.EqualFold(filepath.Ext(path), ".mcworld") || strings.EqualFold(filepath.Ext(path), ".zip") {
		var err error
		root, temporary, err = extractRenderArchive(ctx, path)
		if err != nil {
			return RenderWorld{}, err
		}
	}
	var store *scene.Store
	adopted := false
	defer func() {
		if !adopted {
			if store != nil {
				store.Close()
			}
			if temporary != "" {
				os.RemoveAll(temporary)
			}
		}
	}()
	if _, e := os.Stat(filepath.Join(root, "level.dat")); e != nil {
		return RenderWorld{}, fmt.Errorf("Not a Bedrock world: level.dat is missing")
	}
	catalog := renderAssetCatalog()
	paths, e := renderPackPaths(root, "resource_packs", "world_resource_packs.json")
	if e != nil {
		return RenderWorld{}, e
	}
	for _, path := range paths {
		if e = ctx.Err(); e != nil {
			return RenderWorld{}, e
		}
		pack, e := resource.ReadPath(path)
		if e != nil {
			return RenderWorld{}, e
		}
		readable, e := utils.ReadablePack(pack)
		if e != nil {
			return RenderWorld{}, e
		}
		if e = catalog.AddPack(readable); e != nil {
			return RenderWorld{}, e
		}
	}
	paths, e = renderPackPaths(root, "behavior_packs", "world_behavior_packs.json")
	if e != nil {
		return RenderWorld{}, e
	}
	for _, path := range paths {
		if e = ctx.Err(); e != nil {
			return RenderWorld{}, e
		}
		var source fs.FS = os.DirFS(path)
		if info, statErr := os.Stat(path); statErr != nil {
			return RenderWorld{}, statErr
		} else if !info.IsDir() {
			pack, readErr := resource.ReadPath(path)
			if readErr != nil {
				return RenderWorld{}, readErr
			}
			source, e = utils.ReadablePack(pack)
			if e != nil {
				return RenderWorld{}, e
			}
		}
		if e = catalog.AddBlockDefinitions(source); e != nil {
			return RenderWorld{}, e
		}
	}
	registry := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: map[uint32]merge.Block{}}
	db, e := (mcdb.Config{Blocks: registry, LDBOptions: &opt.Options{ReadOnly: true}}).Open(root)
	if e != nil {
		return RenderWorld{}, fmt.Errorf("Cannot open world. Stop any active capture of this world: %w", e)
	}
	defer db.Close()
	store = scene.NewStore(registry, catalog)
	iterator := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: world.Overworld})
	defer iterator.Release()
	count := 0
	for iterator.Next() {
		if e = ctx.Err(); e != nil {
			return RenderWorld{}, e
		}
		count++
		if count > 4096 {
			return RenderWorld{}, fmt.Errorf("The world contains more than 4096 chunks. Open a smaller exported region")
		}
		column := iterator.Column()
		entities := map[cube.Pos]map[string]any{}
		for _, v := range column.BlockEntities {
			entities[v.Pos] = v.Data
		}
		store.Put(iterator.Position(), column.Chunk, entities)
	}
	if e = iterator.Error(); e != nil {
		return RenderWorld{}, e
	}
	if e = ctx.Err(); e != nil {
		return RenderWorld{}, e
	}
	if count == 0 {
		return RenderWorld{}, fmt.Errorf("No captured chunks in the Overworld")
	}
	name := strings.TrimSpace(filepath.Base(path))
	if text, e := os.ReadFile(filepath.Join(root, "levelname.txt")); e == nil && strings.TrimSpace(string(text)) != "" {
		name = strings.TrimSpace(string(text))
	}
	info := RenderWorld{ID: "render-" + uuid.NewString(), Name: name, Chunks: count}
	a.mu.Lock()
	if a.renderGeneration != generation {
		a.mu.Unlock()
		return RenderWorld{}, context.Canceled
	}
	old := a.renderDocument
	a.renderDocument = &renderDocument{info: info, store: store, temporary: temporary}
	a.renderCancel = nil
	adopted = true
	a.mu.Unlock()
	closeRenderDocument(old)
	return info, nil
}

func closeRenderDocument(doc *renderDocument) {
	if doc != nil {
		if doc.store != nil {
			doc.store.Close()
		}
		if doc.project != nil {
			doc.project.Close()
		}
		if doc.temporary != "" {
			os.RemoveAll(doc.temporary)
		}
	}
}
func (a *App) CancelRenderImport() {
	a.mu.Lock()
	a.renderGeneration++
	if a.renderCancel != nil {
		a.renderCancel()
		a.renderCancel = nil
	}
	a.mu.Unlock()
}
func (a *App) CloseRenderWorld(id string) {
	a.mu.Lock()
	pending := a.pendingStudioDocument
	if pending != nil && (id == "" || pending.info.ID == id) {
		a.pendingStudioDocument = nil
		a.pendingStudioPath = ""
	} else {
		pending = nil
	}
	doc := a.renderDocument
	if doc != nil && (id == "" || doc.info.ID == id) {
		a.renderDocument = nil
	} else {
		doc = nil
	}
	a.mu.Unlock()
	closeRenderDocument(doc)
	closeRenderDocument(pending)
}
func (a *App) GetRenderScenePage(id string, request scene.PageRequest) (scene.Page, error) {
	request.TerrainOnly = true
	a.mu.Lock()
	var old *renderDocument
	if pending := a.pendingStudioDocument; pending != nil && pending.info.ID == id {
		old = a.renderDocument
		a.renderDocument = pending
		a.pendingStudioDocument = nil
		a.studioProjectPath = a.pendingStudioPath
		a.pendingStudioPath = ""
	}
	doc := a.renderDocument
	ctx := a.ctx
	a.mu.Unlock()
	closeRenderDocument(old)
	if doc != nil && doc.info.ID == id {
		if doc.project != nil {
			return doc.project.Page(request)
		}
		return doc.store.Page(ctx, request, nil)
	}
	return a.getScenePage(ctx, id, request)
}

func decodeRenderPNG(data string) ([]byte, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(data, prefix) || len(data) > 96<<20 {
		return nil, fmt.Errorf("Invalid PNG or image too large")
	}
	raw, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, prefix))
	if e != nil {
		return nil, e
	}
	config, e := png.DecodeConfig(bytes.NewReader(raw))
	if e != nil || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 {
		return nil, fmt.Errorf("PNG resolution must be between 1 and 4096 pixels per side")
	}
	if _, e = png.Decode(bytes.NewReader(raw)); e != nil {
		return nil, e
	}
	return raw, nil
}

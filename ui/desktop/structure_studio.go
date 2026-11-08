package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type structureScene struct {
	Store    *scene.Store
	Revision uint64
}
type StructureSceneRequest struct {
	scene.PageRequest
	Generation string `json:"generation"`
	Viewer     string `json:"viewer"`
	MinY       *int32 `json:"minY,omitempty"`
	MaxY       *int32 `json:"maxY,omitempty"`
}
type structurePreview struct {
	Generation string
	Context    context.Context
	Cancel     context.CancelFunc
}

func (a *App) structurePreviewContext(id, viewer, generation string) context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.structurePreviews == nil {
		a.structurePreviews = map[string]*structurePreview{}
	}
	key := id + "|" + viewer
	p := a.structurePreviews[key]
	if p == nil || p.Generation != generation || p.Context.Err() != nil {
		if p != nil {
			p.Cancel()
		}
		ctx, cancel := context.WithCancel(a.ctx)
		p = &structurePreview{generation, ctx, cancel}
		a.structurePreviews[key] = p
	}
	return p.Context
}
func (a *App) CancelStructurePreview(id, viewer string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for key, p := range a.structurePreviews {
		if key == id+"|"+viewer || id == "" && strings.HasSuffix(key, "|"+viewer) {
			p.Cancel()
		}
	}
}
func (a *App) cancelStructurePreviews(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for key, p := range a.structurePreviews {
		if strings.HasPrefix(key, id+"|") {
			p.Cancel()
			delete(a.structurePreviews, key)
		}
	}
}

func structureRoot() string { return utils.PathData("structures", "sessions") }
func (a *App) structureOperation(id, stage string) (context.Context, *structure.Session, func(), error) {
	if id == "" {
		id = uuid.NewString()
	}
	if _, e := uuid.Parse(id); e != nil {
		return nil, nil, nil, fmt.Errorf("Invalid operation identifier")
	}
	s := &structure.Session{ID: id}
	ctx, done, e := s.Begin(a.ctx, stage)
	if e != nil {
		return nil, nil, nil, e
	}
	a.mu.Lock()
	if a.structureOperations == nil {
		a.structureOperations = map[string]*structure.Session{}
	}
	if a.structureOperations[id] != nil {
		a.mu.Unlock()
		done()
		return nil, nil, nil, fmt.Errorf("Operation is already running")
	}
	a.structureOperations[id] = s
	a.mu.Unlock()
	return ctx, s, func() { done(); a.mu.Lock(); delete(a.structureOperations, id); a.mu.Unlock() }, nil
}
func (a *App) structureSession(id string) (*structure.Session, error) {
	a.mu.Lock()
	s := a.structureDocuments[id]
	a.mu.Unlock()
	if s == nil {
		return nil, fmt.Errorf("Structure document is no longer open")
	}
	return s, nil
}
func (a *App) keepStructure(ctx context.Context, d *structure.Document, view map[string]any) (structure.Summary, error) {
	d.MarkMissingDependencies()
	s, err := structure.NewSession(ctx, structureRoot(), d, view)
	if err != nil {
		return structure.Summary{}, err
	}
	a.mu.Lock()
	if a.structureDocuments == nil {
		a.structureDocuments = map[string]*structure.Session{}
	}
	a.structureDocuments[s.ID] = s
	a.mu.Unlock()
	return s.Summary(), nil
}
func (a *App) NewStructure(name string, size structure.Vec3) (structure.Summary, error) {
	d, e := structure.New(name, size)
	if e != nil {
		return structure.Summary{}, e
	}
	return a.keepStructure(a.ctx, d, nil)
}
func (a *App) GetStructure(id string) (structure.Summary, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.Summary(), nil
}
func (a *App) EditStructure(id string, edit structure.Edit) (structure.Summary, error) {
	a.cancelStructurePreviews(id)
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	ctx, done, e := s.Begin(a.ctx, "Editing structure")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	previous := s.History
	s.History = previous.Fork()
	if e = s.History.Apply(ctx, edit); e != nil {
		s.History = previous
		return structure.Summary{}, e
	}
	if e = s.Autosave(ctx); e != nil {
		s.History = previous
		return structure.Summary{}, fmt.Errorf("Edit was not applied because recovery could not be saved: %w", e)
	}
	s.Revision++
	return s.Summary(), nil
}
func (a *App) StructureHistory(id string, redo bool) (structure.Summary, error) {
	a.cancelStructurePreviews(id)
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	ctx, done, e := s.Begin(a.ctx, "Restoring edit")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	previous := s.History
	s.History = previous.Fork()
	if redo {
		s.History.Redo()
	} else {
		s.History.Undo()
	}
	e = s.Autosave(ctx)
	if e != nil {
		s.History = previous
		return structure.Summary{}, e
	}
	s.Revision++
	return s.Summary(), e
}
func (a *App) GetStructureJob(id string) (structure.Job, error) {
	a.mu.Lock()
	operation := a.structureOperations[id]
	a.mu.Unlock()
	if operation != nil {
		return operation.Job(), nil
	}
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Job{}, nil
	}
	return s.Job(), nil
}
func (a *App) CancelStructureJob(id string) {
	a.mu.Lock()
	operation := a.structureOperations[id]
	a.mu.Unlock()
	if operation != nil {
		operation.Cancel()
		return
	}
	if s, e := a.structureSession(id); e == nil {
		s.Cancel()
	}
}
func (a *App) SetStructureView(id string, view map[string]any) (structure.Summary, error) {
	raw, e := json.Marshal(view)
	if e != nil || len(raw) > 65536 {
		return structure.Summary{}, fmt.Errorf("Invalid structure view")
	}
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	previous := s.View
	s.View = view
	if e = s.Autosave(a.ctx); e != nil {
		s.View = previous
		return structure.Summary{}, e
	}
	return s.Summary(), nil
}
func (a *App) CloseStructure(id string) error {
	a.cancelStructurePreviews(id)
	s, e := a.structureSession(id)
	if e != nil {
		return nil
	}
	s.Cancel()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	a.mu.Lock()
	delete(a.structureDocuments, id)
	stores := []*scene.Store{}
	for key, cached := range a.structureScenes {
		if key == id || strings.HasPrefix(key, id+"|") {
			stores = append(stores, cached.Store)
			delete(a.structureScenes, key)
		}
	}
	a.mu.Unlock()
	for _, store := range stores {
		store.Close()
	}
	return nil
}

type StructureRecovery struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Modified int64  `json:"modified"`
	Open     bool   `json:"open"`
}

func (a *App) DeleteStructures(ids []string) error {
	// Validate the entire selection before removing any library copies.
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed.String() != id {
			return fmt.Errorf("Invalid structure identifier")
		}
	}
	for _, id := range ids {
		if err := a.CloseStructure(id); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(structureRoot(), id)); err != nil {
			return fmt.Errorf("Could not delete blueprint: %w", err)
		}
	}
	return nil
}

func (a *App) ListStructureRecovery() ([]StructureRecovery, error) {
	items, e := os.ReadDir(structureRoot())
	if os.IsNotExist(e) {
		return []StructureRecovery{}, nil
	}
	if e != nil {
		return nil, e
	}
	out := []StructureRecovery{}
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		p := filepath.Join(structureRoot(), item.Name(), "recovery.craftbox")
		info, e := os.Stat(p)
		if e != nil {
			continue
		}
		a.mu.Lock()
		s := a.structureDocuments[item.Name()]
		a.mu.Unlock()
		name := "Recovered blueprint"
		if s != nil {
			s.Mu.Lock()
			name = s.History.Doc.Name
			s.Mu.Unlock()
		} else {
			if d, e := structure.ReadProjectBrief(p); e == nil {
				name = d.Name
			} else {
				continue
			}
		}
		out = append(out, StructureRecovery{item.Name(), name, info.ModTime().UnixMilli(), s != nil})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified > out[j].Modified })
	return out, nil
}
func (a *App) GetStructureBlockCatalog(query string) []structure.State {
	query = strings.ToLower(strings.TrimSpace(query))
	if len(query) > 128 {
		return []structure.State{}
	}
	reg := structure.Registry()
	seen := map[string]bool{}
	out := []structure.State{}
	for _, b := range world.DefaultBlockRegistry.Blocks() {
		name, _ := b.EncodeBlock()
		if seen[name] || !strings.Contains(name, query) {
			continue
		}
		seen[name] = true
		rid, ok := reg.StateToRuntimeID(name, nil)
		if !ok {
			continue
		}
		name, props, ok := reg.RuntimeIDToState(rid)
		if !ok {
			continue
		}
		s, e := structure.NormalizeState(structure.State{Name: name, States: props, Version: chunk.CurrentBlockVersion})
		if e == nil {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if len(out) > 300 {
		out = out[:300]
	}
	return out
}
func (a *App) RecoverStructure(id string) (structure.Summary, error) {
	if s, e := a.structureSession(id); e == nil {
		s.Mu.Lock()
		defer s.Mu.Unlock()
		return s.Summary(), nil
	}
	s, e := structure.RecoverSession(a.ctx, structureRoot(), id)
	if e != nil {
		return structure.Summary{}, e
	}
	a.mu.Lock()
	if a.structureDocuments == nil {
		a.structureDocuments = map[string]*structure.Session{}
	}
	a.structureDocuments[id] = s
	a.mu.Unlock()
	return s.Summary(), nil
}
func (a *App) OpenStructurePath(path string) (structure.Summary, error) {
	return a.openStructurePath(a.ctx, path)
}
func (a *App) OpenStructureFile(path, jobID string) (structure.Summary, error) {
	ctx, _, done, e := a.structureOperation(jobID, "Opening blueprint and dependencies")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	return a.openStructurePath(ctx, path)
}
func (a *App) openStructurePath(ctx context.Context, path string) (structure.Summary, error) {
	if strings.EqualFold(filepath.Ext(path), ".craftbox") {
		absolute, _ := filepath.Abs(path)
		a.mu.Lock()
		sessions := make([]*structure.Session, 0, len(a.structureDocuments))
		for _, session := range a.structureDocuments {
			sessions = append(sessions, session)
		}
		a.mu.Unlock()
		for _, session := range sessions {
			session.Mu.Lock()
			other, _ := filepath.Abs(session.Path)
			if session.Path != "" && strings.EqualFold(absolute, other) {
				result := session.Summary()
				session.Mu.Unlock()
				return result, nil
			}
			session.Mu.Unlock()
		}
	}
	f, e := os.Open(path)
	if e != nil {
		return structure.Summary{}, e
	}
	defer f.Close()
	var d *structure.Document
	var view map[string]any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mcstructure":
		d, e = structure.ReadMCStructure(ctx, f, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	case ".craftbox":
		info, err := f.Stat()
		if err != nil {
			return structure.Summary{}, err
		}
		temp, err := os.MkdirTemp("", "craftbox-structure-open-")
		if err != nil {
			return structure.Summary{}, err
		}
		defer os.RemoveAll(temp)
		d, view, e = structure.ReadProject(ctx, f, info.Size(), temp)
	default:
		return structure.Summary{}, fmt.Errorf("Choose a Bedrock .mcstructure or Structure Studio .craftbox project")
	}
	if e != nil {
		return structure.Summary{}, e
	}
	result, e := a.keepStructure(ctx, d, view)
	if e == nil && strings.EqualFold(filepath.Ext(path), ".craftbox") {
		s, _ := a.structureSession(result.ID)
		s.Path = path
		s.MarkSaved()
		e = s.Autosave(a.ctx)
		result = s.Summary()
	}
	return result, e
}

type StructureImport struct {
	JobID     string           `json:"jobId"`
	Name      string           `json:"name"`
	WorldID   string           `json:"worldId"`
	Path      string           `json:"path"`
	Bounds    structure.Bounds `json:"bounds"`
	Dimension int32            `json:"dimension"`
}

func (a *App) ImportStructureWorld(r StructureImport) (structure.Summary, error) {
	ctx, operation, done, e := a.structureOperation(r.JobID, "Extracting Bedrock region")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	b := r.Bounds.Normalized()
	if _, e := structure.NewImporter(r.Name, b); e != nil {
		return structure.Summary{}, e
	}
	a.mu.Lock()
	store := a.scenes[r.WorldID]
	assets := a.sceneSources[r.WorldID]
	root := r.Path
	for _, w := range a.state.Worlds {
		if r.WorldID != "" && w.ID == r.WorldID {
			root = w.CaptureFolder
			if root == "" {
				root = w.Path
			}
			break
		}
	}
	a.mu.Unlock()
	var d *structure.Document
	var err error
	temporary := ""
	if store != nil {
		if assets != nil {
			s, e := structure.NewCapturedSession(ctx, structureRoot(), store, r.Name, b, assets.ResourcePacks, assets.BlockEntries)
			if e != nil {
				return structure.Summary{}, e
			}
			a.mu.Lock()
			if a.structureDocuments == nil {
				a.structureDocuments = map[string]*structure.Session{}
			}
			a.structureDocuments[s.ID] = s
			a.mu.Unlock()
			return s.Summary(), nil
		}
		d, err = structure.FromScene(ctx, store, r.Name, b)
	} else {
		if root == "" {
			return structure.Summary{}, fmt.Errorf("Select a source world")
		}
		if !strings.EqualFold(filepath.Ext(root), ".mcworld") {
			if info, e := os.Stat(root); e != nil || !info.IsDir() {
				return structure.Summary{}, fmt.Errorf("Choose a Bedrock folder or .mcworld")
			}
		} else {
			root, temporary, err = extractRenderArchive(ctx, root)
			if err != nil {
				return structure.Summary{}, err
			}
			defer os.RemoveAll(temporary)
		}
		total := int64((b.Max[0]>>4)-(b.Min[0]>>4)+1) * int64((b.Max[2]>>4)-(b.Min[2]>>4)+1)
		d, err = structure.FromWorld(ctx, root, r.Name, b, r.Dimension, func(n int) { operation.Progress("Reading Bedrock chunks", int64(n), total) })
	}
	if err != nil {
		return structure.Summary{}, err
	}
	if root != "" {
		d.Packs, err = structureDependencies(root)
		if err != nil {
			return structure.Summary{}, err
		}
	}
	operation.Progress("Saving independent blueprint", 0, 0)
	return a.keepStructure(ctx, d, nil)
}
func structureDependencies(root string) ([]structure.Dependency, error) {
	out := []structure.Dependency{}
	for _, kind := range []string{"resource_packs", "behavior_packs"} {
		paths, err := renderPackPaths(root, kind, "world_"+kind+".json")
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			raw, e := os.ReadFile(filepath.Join(p, "manifest.json"))
			if e != nil {
				return nil, fmt.Errorf("Cannot read dependency %s: %w", p, e)
			}
			var m struct{ Header struct{ UUID, Name string } }
			if e = utils.ParseJson(raw, &m); e != nil {
				return nil, e
			}
			out = append(out, structure.Dependency{ID: m.Header.UUID, UUID: m.Header.UUID, Name: m.Header.Name, Kind: kind, Source: p})
		}
	}
	return out, nil
}
func (a *App) GetStructureMaterials(id string, boxes []structure.Bounds) (structure.Materials, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Materials{}, e
	}
	s.Mu.Lock()
	d := s.History.Doc
	s.Mu.Unlock()
	return structure.CountMaterials(a.ctx, d, boxes)
}
func (a *App) ExtractStructure(id, name string, boxes []structure.Bounds) (structure.Summary, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	ctx, done, e := s.Begin(a.ctx, "Extracting blueprint selection")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	s.Mu.Lock()
	d := s.History.Doc
	s.Mu.Unlock()
	out, e := structure.Extract(ctx, d, boxes, name)
	if e != nil {
		return structure.Summary{}, e
	}
	return a.keepStructure(ctx, out, nil)
}

type StructureLayer struct {
	Y     int32            `json:"y"`
	X     int32            `json:"x"`
	Z     int32            `json:"z"`
	Width int32            `json:"width"`
	Depth int32            `json:"depth"`
	Cells []structure.Cell `json:"cells"`
}

func (a *App) GetStructureLayer(id string, y, x, z, width, depth int32) (StructureLayer, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return StructureLayer{}, e
	}
	s.Mu.Lock()
	d := s.History.Doc
	s.Mu.Unlock()
	if y < 0 || y >= d.Size[1] || width < 1 || depth < 1 || width > 512 || depth > 512 || x < 0 || z < 0 || x+width > d.Size[0] || z+depth > d.Size[2] {
		return StructureLayer{}, fmt.Errorf("Invalid layer window (maximum 512 x 512)")
	}
	out := StructureLayer{Y: y, X: x, Z: z, Width: width, Depth: depth, Cells: make([]structure.Cell, 0, int(width*depth))}
	for az := z; az < z+depth; az++ {
		for ax := x; ax < x+width; ax++ {
			out.Cells = append(out.Cells, d.Cell(structure.Vec3{ax, y, az}))
		}
	}
	return out, nil
}
func (a *App) GetStructureBlock(id string, p structure.Vec3) (structure.State, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.State{}, e
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	c := s.History.Doc.Cell(p)
	if !c.Known || c.Layers[0] < 0 {
		return structure.State{}, fmt.Errorf("This position is unknown or structure void")
	}
	return s.History.Doc.Palette[c.Layers[0]], nil
}
func (a *App) GetStructureScenePage(id string, r StructureSceneRequest) (scene.Page, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return scene.Page{}, e
	}
	ctx := a.structurePreviewContext(id, r.Viewer, r.Generation)
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if e = ctx.Err(); e != nil {
		return scene.Page{}, e
	}
	key := id
	if r.MinY != nil && r.MaxY != nil {
		if *r.MinY < 0 || *r.MaxY < *r.MinY || *r.MaxY >= s.History.Doc.Size[1] {
			return scene.Page{}, fmt.Errorf("Invalid preview layer")
		}
		key = fmt.Sprintf("%s|%d:%d", id, *r.MinY, *r.MaxY)
	}
	a.mu.Lock()
	cached := a.structureScenes[key]
	a.mu.Unlock()
	if cached == nil || cached.Revision != s.Revision {
		cat := renderAssetCatalog()
		for _, p := range s.History.Doc.Packs {
			if p.Missing || p.Source == "" {
				continue
			}
			if p.Kind == "behavior_packs" {
				if e = cat.AddBlockDefinitions(os.DirFS(p.Source)); e != nil {
					return scene.Page{}, e
				}
			} else {
				pack, err := resource.ReadPath(p.Source)
				if err != nil {
					return scene.Page{}, err
				}
				readable, err := utils.ReadablePack(pack)
				if err != nil {
					return scene.Page{}, err
				}
				if err = cat.AddPack(readable); err != nil {
					return scene.Page{}, err
				}
			}
		}
		reg := structure.Registry()
		store := scene.NewStore(reg, cat)
		renderDoc := s.History.Doc
		offset := structure.Vec3{}
		if r.MinY != nil && r.MaxY != nil {
			b := renderDoc.Bounds()
			b.Min[1] = *r.MinY
			b.Max[1] = *r.MaxY
			renderDoc, e = structure.Extract(ctx, renderDoc, []structure.Bounds{b}, renderDoc.Name)
			if e != nil {
				return scene.Page{}, e
			}
			offset[1] = *r.MinY
		}
		e = renderDoc.Columns(ctx, reg, offset, func(p world.ChunkPos, c *chunk.Column) error {
			be := map[cube.Pos]map[string]any{}
			for _, v := range c.BlockEntities {
				be[v.Pos] = v.Data
			}
			store.Put(p, c.Chunk, be)
			for _, v := range c.Entities {
				if err := store.PutEntityNBT(v.ID, v.Data); err != nil {
					return err
				}
			}
			return nil
		})
		if e != nil {
			store.Close()
			return scene.Page{}, e
		}
		if cached != nil {
			cached.Store.Close()
		}
		cached = &structureScene{store, s.Revision}
		a.mu.Lock()
		if a.structureScenes == nil {
			a.structureScenes = map[string]*structureScene{}
		}
		a.structureScenes[key] = cached
		old := []*scene.Store{}
		count := 0
		for k, v := range a.structureScenes {
			if strings.HasPrefix(k, id+"|") {
				count++
				if count > 3 && k != key {
					delete(a.structureScenes, k)
					old = append(old, v.Store)
				}
			}
		}
		a.mu.Unlock()
		for _, store := range old {
			store.Close()
		}
	}
	return cached.Store.Page(ctx, r.PageRequest, nil)
}
func (a *App) CompareStructures(sourceID, targetID string, offset structure.Vec3) (structure.Comparison, error) {
	s, e := a.structureSession(sourceID)
	if e != nil {
		return structure.Comparison{}, e
	}
	t, e := a.structureSession(targetID)
	if e != nil {
		return structure.Comparison{}, e
	}
	s.Mu.Lock()
	source := s.History.Doc
	s.Mu.Unlock()
	t.Mu.Lock()
	target := t.History.Doc
	t.Mu.Unlock()
	return structure.Compare(a.ctx, source, target, offset, nil)
}
func (a *App) PlaceStructureDocument(sourceID, targetID string, anchor structure.Vec3, mode string, entities bool) (structure.Summary, error) {
	s, e := a.structureSession(sourceID)
	if e != nil {
		return structure.Summary{}, e
	}
	t, e := a.structureSession(targetID)
	if e != nil {
		return structure.Summary{}, e
	}
	s.Mu.Lock()
	source := s.History.Doc
	s.Mu.Unlock()
	ctx, done, e := t.Begin(a.ctx, "Placing blueprint")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	t.Mu.Lock()
	defer t.Mu.Unlock()
	next, _, e := structure.Place(ctx, source, t.History.Doc, anchor, mode, entities)
	if e != nil {
		return structure.Summary{}, e
	}
	previous := t.History
	t.History = previous.Fork()
	t.History.Commit(next)
	e = t.AdoptDependencies(ctx)
	if e == nil {
		e = t.Autosave(ctx)
	}
	if e != nil {
		t.History = previous
		return structure.Summary{}, e
	}
	t.Revision++
	return t.Summary(), e
}
func (a *App) AttachStructurePack(id, path, replaceID string) (structure.Summary, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	ctx, done, e := s.Begin(a.ctx, "Attaching resource dependency")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	root := path
	if info, err := os.Stat(path); err != nil {
		return structure.Summary{}, err
	} else if !info.IsDir() {
		temp := ""
		root, temp, e = extractRenderArchive(ctx, path)
		if e != nil {
			return structure.Summary{}, e
		}
		defer os.RemoveAll(temp)
	}
	raw, e := os.ReadFile(filepath.Join(root, "manifest.json"))
	if e != nil {
		return structure.Summary{}, e
	}
	var manifest struct {
		Header  struct{ UUID, Name string }
		Modules []struct{ Type string }
	}
	if e = utils.ParseJson(raw, &manifest); e != nil {
		return structure.Summary{}, e
	}
	if _, e = uuid.Parse(manifest.Header.UUID); e != nil {
		return structure.Summary{}, fmt.Errorf("Dependency manifest requires a valid UUID")
	}
	kind := "resource_packs"
	for _, module := range manifest.Modules {
		if module.Type == "data" {
			kind = "behavior_packs"
		}
	}
	d := s.History.Doc.Clone()
	deps := []structure.Dependency{}
	for _, p := range d.Packs {
		if p.ID != replaceID && p.UUID != manifest.Header.UUID {
			deps = append(deps, p)
		}
	}
	d.Packs = append(deps, structure.Dependency{ID: manifest.Header.UUID, UUID: manifest.Header.UUID, Name: manifest.Header.Name, Kind: kind, Source: root})
	previous := s.History
	s.History = previous.Fork()
	s.History.Commit(d)
	if e = s.AdoptDependencies(ctx); e == nil {
		e = s.Autosave(ctx)
	}
	if e != nil {
		s.History = previous
		return structure.Summary{}, e
	}
	s.Revision++
	return s.Summary(), nil
}
func (a *App) WriteStructureFile(id, path string, project bool, o structure.ExportOptions) (structure.Summary, error) {
	s, e := a.structureSession(id)
	if e != nil {
		return structure.Summary{}, e
	}
	ctx, done, e := s.Begin(a.ctx, "Saving structure")
	if e != nil {
		return structure.Summary{}, e
	}
	defer done()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if project && !strings.EqualFold(filepath.Ext(path), ".craftbox") {
		return structure.Summary{}, fmt.Errorf("Project filename must end in .craftbox")
	}
	if !project && !strings.EqualFold(filepath.Ext(path), ".mcstructure") {
		return structure.Summary{}, fmt.Errorf("Structure filename must end in .mcstructure")
	}
	e = structure.WriteAtomic(path, func(w io.Writer) error {
		if project {
			return structure.WriteProject(ctx, w, s.History.Doc, s.View, o.Packs)
		}
		return structure.WriteMCStructure(ctx, w, s.History.Doc, o)
	})
	if e != nil {
		return structure.Summary{}, e
	}
	if project {
		s.Path = path
		s.MarkSaved()
		e = s.Autosave(ctx)
	}
	return s.Summary(), e
}

type StructurePlacement struct {
	ID          string         `json:"id"`
	Source      string         `json:"source"`
	Destination string         `json:"destination"`
	Anchor      structure.Vec3 `json:"anchor"`
	Mode        string         `json:"mode"`
	Dimension   int32          `json:"dimension"`
	Entities    bool           `json:"entities"`
	Packs       bool           `json:"packs"`
	Inventories bool           `json:"inventories"`
	BlockData   bool           `json:"blockData"`
}

func (a *App) ExportStructureWorld(r StructurePlacement) (string, error) {
	s, e := a.structureSession(r.ID)
	if e != nil {
		return "", e
	}
	ctx, done, e := s.Begin(a.ctx, "Preparing new world copy")
	if e != nil {
		return "", e
	}
	defer done()
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if !strings.EqualFold(filepath.Ext(r.Destination), ".mcworld") {
		return "", fmt.Errorf("Choose a new .mcworld output file")
	}
	if r.Source != "" {
		sourceAbs, _ := filepath.Abs(r.Source)
		destAbs, _ := filepath.Abs(r.Destination)
		if strings.EqualFold(sourceAbs, destAbs) {
			return "", fmt.Errorf("The original world cannot be overwritten")
		}
	}
	temporaryWorld, e := os.MkdirTemp("", "craftbox-structure-world-")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(temporaryWorld)
	folder := filepath.Join(temporaryWorld, "world")
	source := r.Source
	if source != "" {
		if strings.EqualFold(filepath.Ext(source), ".mcworld") {
			root, temp, err := extractRenderArchive(ctx, source)
			if err != nil {
				return "", err
			}
			defer os.RemoveAll(temp)
			source = root
		}
		if e = worldexport.Copy(ctx, source, folder, worldexport.Options{Profile: "original"}, nil); e != nil {
			return "", e
		}
	}
	reg := structure.Registry()
	db, e := (mcdb.Config{Blocks: reg}).Open(folder)
	if e != nil {
		return "", e
	}
	dim, ok := world.DimensionByID(int(r.Dimension))
	if !ok {
		db.Close()
		return "", fmt.Errorf("Unsupported dimension")
	}
	if source == "" {
		db.LevelDat().Generator = 2
		db.LevelDat().FlatWorldLayers = `{"biome_id":1,"block_layers":[{"block_data":0,"block_id":0,"count":1}],"encoding_version":3,"structure_options":null}`
		set := db.Settings()
		set.Name = s.History.Doc.Name
		set.Spawn = cube.Pos{int(r.Anchor[0]), int(r.Anchor[1]) + int(s.History.Doc.Size[1]) + 2, int(r.Anchor[2])}
		db.SaveSettings(set)
	}
	doc := s.History.Doc.ExportCopy(structure.ExportOptions{Entities: r.Entities, Inventories: r.Inventories, BlockData: r.BlockData})
	e = structure.PlaceInDatabase(ctx, doc, db, reg, r.Anchor, r.Mode, dim, r.Entities, func(n, total int) { s.Progress("Placing blocks", int64(n), int64(total)) })
	closeErr := db.Close()
	if e != nil {
		return "", e
	}
	if closeErr != nil {
		return "", closeErr
	}
	if r.Packs {
		if e = installStructurePacks(ctx, s.History.Doc, folder); e != nil {
			return "", e
		}
	}
	s.Progress("Writing .mcworld", 0, 0)
	if e = compositionArchive(ctx, folder, r.Destination); e != nil {
		return "", e
	}
	return r.Destination, nil
}
func installStructurePacks(ctx context.Context, d *structure.Document, target string) error {
	stage, e := os.MkdirTemp("", "craftbox-structure-packs-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	for _, p := range d.Packs {
		if p.Kind != "resource_packs" && p.Kind != "behavior_packs" {
			return fmt.Errorf("Unknown dependency type %s", p.Kind)
		}
		if p.Missing || p.Source == "" {
			return fmt.Errorf("Missing dependency %s", p.Name)
		}
		to := filepath.Join(stage, p.Kind, utils.MakeValidFilename(p.ID))
		if e = structure.CopyDirectory(ctx, p.Source, to); e != nil {
			return e
		}
	}
	for _, kind := range []string{"resource_packs", "behavior_packs"} {
		if e = copyIdentityPacks(ctx, stage, target, kind); e != nil {
			return e
		}
		stack := []map[string]any{}
		stackPath := filepath.Join(target, "world_"+kind+".json")
		if raw, err := os.ReadFile(stackPath); err == nil {
			if err = json.Unmarshal(raw, &stack); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		active := map[string]bool{}
		for _, entry := range stack {
			if id, ok := entry["pack_id"].(string); ok {
				active[strings.ToLower(id)] = true
			}
		}
		for _, p := range d.Packs {
			if p.Kind != kind || active[strings.ToLower(p.UUID)] {
				continue
			}
			raw, e := os.ReadFile(filepath.Join(p.Source, "manifest.json"))
			if e != nil {
				return e
			}
			var m compositionPackManifest
			if e = utils.ParseJson(raw, &m); e != nil {
				return e
			}
			stack = append(stack, map[string]any{"pack_id": m.Header.UUID, "version": m.Header.Version})
			active[strings.ToLower(m.Header.UUID)] = true
		}
		raw, _ := json.Marshal(stack)
		if e = os.WriteFile(filepath.Join(target, "world_"+kind+".json"), raw, 0600); e != nil {
			return e
		}
	}
	return nil
}

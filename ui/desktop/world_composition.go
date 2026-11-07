package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/google/uuid"
)

type WorldPlacement struct {
	WorldID string                `json:"worldId"`
	Offset  [3]int32              `json:"offset"`
	Bounds  *messages.ChunkBounds `json:"bounds,omitempty"`
}
type WorldCompositionRequest struct {
	Name  string           `json:"name"`
	Parts []WorldPlacement `json:"parts"`
}
type WorldCompositionPreview struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	scene.CompositionResult
}
type compositionStage struct {
	preview WorldCompositionPreview
	folder  string
	sources []string
}
type CompositionProgress struct {
	Stage   string `json:"stage"`
	Running bool   `json:"running"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
}

func (a *App) GetCompositionProgress() CompositionProgress {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.compositionProgress
}
func (a *App) CancelComposition() {
	a.mu.Lock()
	cancel := a.compositionCancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
func (a *App) DiscardComposition(id string) {
	a.mu.Lock()
	stage := a.compositionStage
	if stage != nil && stage.preview.ID == id && !a.compositionProgress.Running {
		a.compositionStage = nil
	} else {
		stage = nil
	}
	a.mu.Unlock()
	if stage != nil {
		os.RemoveAll(stage.folder)
	}
}

func (a *App) PrepareWorldComposition(request WorldCompositionRequest) (WorldCompositionPreview, error) {
	var empty WorldCompositionPreview
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len(request.Name) > 100 || len(request.Parts) < 1 || len(request.Parts) > 16 {
		return empty, fmt.Errorf("Enter a name and choose 1–16 placements")
	}
	a.mu.Lock()
	if a.compositionProgress.Running {
		a.mu.Unlock()
		return empty, fmt.Errorf("A composition is already in progress")
	}
	old := a.compositionStage
	a.compositionStage = nil
	ctx, cancel := context.WithCancel(a.ctx)
	a.compositionCancel = cancel
	a.compositionProgress = CompositionProgress{Running: true, Stage: "Reading source blocks"}
	a.mu.Unlock()
	if old != nil {
		os.RemoveAll(old.folder)
	}
	defer func() {
		cancel()
		a.mu.Lock()
		a.compositionCancel = nil
		a.compositionProgress.Running = false
		a.mu.Unlock()
	}()
	parts := []scene.CompositionPart{}
	folders := []string{}
	for _, p := range request.Parts {
		for _, v := range p.Offset {
			if v < -30_000_000 || v > 30_000_000 {
				return empty, fmt.Errorf("Offset exceeds the world border")
			}
		}
		store, parent, _, keep, err := a.sceneSelection(p.WorldID)
		if err != nil {
			return empty, err
		}
		baseKeep := keep
		if p.Bounds != nil {
			bounds := *p.Bounds
			if bounds.MinX > bounds.MaxX || bounds.MinZ > bounds.MaxZ {
				return empty, fmt.Errorf("Invalid source bounds")
			}
			keep = func(x, z int32) bool { return bounds.Contains(x, z) && (baseKeep == nil || baseKeep(x, z)) }
		}
		parts = append(parts, scene.CompositionPart{Store: store, Keep: keep, Offset: p.Offset})
		a.mu.Lock()
		for _, w := range a.state.Worlds {
			if w.ID == parent {
				folder := w.CaptureFolder
				if folder == "" && w.Path != "" {
					folder = strings.TrimSuffix(w.Path, ".mcworld")
				}
				if folder != "" {
					folders = append(folders, folder)
				}
				break
			}
		}
		a.mu.Unlock()
	}
	folder, err := os.MkdirTemp("", "craftbox-compose-")
	if err != nil {
		return empty, err
	}
	keepStage := false
	defer func() {
		if !keepStage {
			os.RemoveAll(folder)
		}
	}()
	result, err := scene.Compose(ctx, parts, folder, request.Name, func(done, total int) {
		a.mu.Lock()
		a.compositionProgress.Done = done
		a.compositionProgress.Total = total
		a.mu.Unlock()
	})
	if err != nil {
		return empty, err
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	a.setCompositionPhase("Copying resource packs")
	for _, source := range folders {
		if err = copyCompositionPacks(ctx, source, folder); err != nil {
			return empty, err
		}
	}
	preview := WorldCompositionPreview{ID: uuid.NewString(), Name: request.Name, CompositionResult: result}
	a.mu.Lock()
	a.compositionStage = &compositionStage{preview: preview, folder: folder, sources: folders}
	a.mu.Unlock()
	keepStage = true
	return preview, nil
}

func copyCompositionPacks(ctx context.Context, source, target string) error {
	for _, kind := range []string{"resource_packs", "behavior_packs"} {
		if err := copyIdentityPacks(ctx, source, target, kind); err != nil {
			return err
		}
	}
	for _, name := range []string{"world_resource_packs.json", "world_behavior_packs.json"} {
		raw, err := os.ReadFile(filepath.Join(source, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var incoming, existing []map[string]any
		if err = json.Unmarshal(raw, &incoming); err != nil {
			return err
		}
		if old, err := os.ReadFile(filepath.Join(target, name)); err == nil {
			if err = json.Unmarshal(old, &existing); err != nil {
				return err
			}
		}
		seen := map[string]bool{}
		combined := []map[string]any{}
		for _, p := range append(existing, incoming...) {
			b, err := json.Marshal(p)
			if err != nil {
				return err
			}
			if !seen[string(b)] {
				combined = append(combined, p)
				seen[string(b)] = true
			}
		}
		b, _ := json.Marshal(combined)
		if err = os.WriteFile(filepath.Join(target, name), b, 0600); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) SaveWorldComposition(id string, allowConflicts bool) (SavedWorld, error) {
	var empty SavedWorld
	a.mu.Lock()
	stage := a.compositionStage
	if stage == nil || stage.preview.ID != id || a.compositionProgress.Running {
		a.mu.Unlock()
		return empty, fmt.Errorf("Preview expired; check placements again")
	}
	if stage.preview.Conflicts > 0 && !allowConflicts {
		a.mu.Unlock()
		return empty, fmt.Errorf("Confirm non-air block overlaps before saving")
	}
	a.compositionProgress.Running = true
	a.compositionProgress.Stage = "Copying merged world"
	ctx, cancel := context.WithCancel(a.ctx)
	a.compositionCancel = cancel
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.compositionCancel = nil
		a.compositionProgress.Running = false
		a.mu.Unlock()
	}()
	root := utils.PathData("worlds", "Compositions")
	if err := os.MkdirAll(root, 0700); err != nil {
		return empty, err
	}
	target := filepath.Join(root, utils.MakeValidFilename(stage.preview.Name)+"--"+id)
	if err := os.Mkdir(target, 0700); err != nil {
		return empty, err
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(target)
			os.Remove(target + ".mcworld")
		}
	}()

	err := filepath.WalkDir(stage.folder, func(path string, entry fs.DirEntry, err error) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(stage.folder, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("Unsupported composition file")
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, renderContextReader{ctx: ctx, reader: in})
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if err != nil {
		return empty, err
	}
	a.setCompositionPhase("Writing .mcworld")
	if err = compositionArchive(ctx, target, target+".mcworld"); err != nil {
		return empty, err
	}
	imported := NewApp()
	a.setCompositionPhase("Loading merged world")
	doc, err := imported.importRenderWorld(ctx, target)
	if err != nil {
		return empty, err
	}
	store := imported.renderDocument.store
	imported.renderDocument = nil
	world := SavedWorld{ID: id, Name: stage.preview.Name, CaptureFolder: target, Path: target + ".mcworld", State: "finished", Chunks: stage.preview.Chunks, Entities: stage.preview.Entities, HeightRange: [2]int{-64, 319}, Spawn: &stage.preview.Spawn}
	tileMap := map[[2]int32]Tile{}
	var cursor *[2]int32
	var sequence uint64
	a.setCompositionPhase("Preparing map")
	for {
		page, err := store.MapPage(ctx, scene.MapRequest{Cursor: cursor, Target: sequence, Limit: 32}, nil)
		if err != nil {
			store.Close()
			return empty, err
		}
		sequence = page.Sequence
		a.mu.Lock()
		a.compositionProgress.Done += len(page.Tiles)
		a.mu.Unlock()
		for _, t := range page.Tiles {
			tileMap[[2]int32{t.X, t.Z}] = Tile{WorldID: id, X: t.X, Z: t.Z, Image: t.Image}
		}
		cursor = page.Next
		if page.Complete {
			break
		}
	}
	a.mu.Lock()
	if a.scenes == nil {
		a.scenes = map[string]*scene.Store{}
	}
	if err = ctx.Err(); err != nil {
		a.mu.Unlock()
		store.Close()
		return empty, err
	}
	a.scenes[id] = store
	a.worldTiles[id] = tileMap
	a.state.Worlds = append(a.state.Worlds, world)
	a.tileSequence++
	a.tileResetSequence = a.tileSequence
	a.compositionStage = nil
	emit := a.emit
	a.mu.Unlock()
	messages.RecordCaptureFolder(id, target)
	committed = true
	os.RemoveAll(stage.folder)
	if emit != nil {
		emit("desktop:reset-map", nil)
		emit("desktop:tiles", a.GetMapTiles())
	}
	a.publish()
	_ = doc
	return world, nil
}

func (a *App) setCompositionPhase(stage string) {
	a.mu.Lock()
	a.compositionProgress.Stage = stage
	a.compositionProgress.Done = 0
	a.mu.Unlock()
}

func (a *App) serveComposition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var p struct {
		Request        WorldCompositionRequest `json:"request"`
		ID             string                  `json:"id"`
		AllowConflicts bool                    `json:"allowConflicts"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		http.Error(w, "Invalid composition", 400)
		return
	}
	var result any
	var err error
	switch r.URL.Path {
	case "/composition-prepare":
		result, err = a.PrepareWorldComposition(p.Request)
	case "/composition-save":
		result, err = a.SaveWorldComposition(p.ID, p.AllowConflicts)
	case "/composition-cancel":
		a.CancelComposition()
		result = a.GetCompositionProgress()
	case "/composition-discard":
		a.DiscardComposition(p.ID)
		result = true
	case "/composition-progress":
		result = a.GetCompositionProgress()
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(result)
}

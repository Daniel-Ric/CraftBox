package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/google/uuid"
)

type RenderRegionRequest struct {
	WorldID string       `json:"worldId"`
	Name    string       `json:"name"`
	Bounds  scene.Bounds `json:"bounds"`
}
type RenderRegionResult struct {
	World RenderWorld `json:"world"`
	Path  string      `json:"path"`
}

func (a *App) CreateRenderRegion(request RenderRegionRequest) (result RenderRegionResult, err error) {
	if err = request.Bounds.Validate(); err != nil {
		return result, err
	}
	request.Name = strings.TrimSpace(request.Name)
	options := worldexport.Options{Settings: worldexport.Settings{Name: &request.Name}}
	if err = options.Validate(); err != nil {
		return result, err
	}
	a.mu.Lock()
	if a.worldExportCancel != nil {
		a.mu.Unlock()
		return result, fmt.Errorf("Another world export is running")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	a.worldExportCancel = cancel
	a.worldExportSource = request.WorldID
	a.worldExportStatus = WorldExportStatus{Running: true}
	doc := a.renderDocumentLocked(request.WorldID)
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.worldExportCancel = nil
		a.worldExportSource = ""
		a.worldExportStatus.Running = false
		a.mu.Unlock()
	}()
	if err = ctx.Err(); err != nil {
		return result, err
	}
	a.editMu.Lock()
	defer a.editMu.Unlock()
	var source string
	var dimension int32
	var sourceStore *scene.Store
	if doc != nil && doc.info.ID == request.WorldID {
		doc.exportMu.Lock()
		defer doc.exportMu.Unlock()
		source, dimension = doc.sourceRoot, doc.info.Dimension
		sourceStore = doc.store
		if source == "" {
			return result, fmt.Errorf("This project contains geometry only. Open the original Bedrock world to create a .mcworld region")
		}
	} else {
		_, source, err = a.finishedExportWorld(request.WorldID)
		if err != nil {
			return result, err
		}
		sourceStore, _, _, _, _ = a.sceneSelection(request.WorldID)
	}
	plan, err := worldexport.Describe(ctx, source)
	if err != nil {
		return result, err
	}
	if doc == nil || doc.info.ID != request.WorldID {
		dimension = plan.Dimensions[0].ID
		for _, d := range plan.Dimensions {
			if d.ID == 0 {
				dimension = 0
				break
			}
		}
	}
	bounds := worldexport.Bounds{MinX: request.Bounds.MinX, MaxX: request.Bounds.MaxX, MinZ: request.Bounds.MinZ, MaxZ: request.Bounds.MaxZ}
	for _, d := range plan.Dimensions {
		rule := worldexport.DimensionRule{Source: d.ID}
		if d.ID == dimension {
			target := dimension
			rule.Target = &target
			rule.Bounds = &bounds
		}
		options.Dimensions = append(options.Dimensions, rule)
	}
	var spawn *[3]int32
	for _, p := range plan.Chunks {
		if p.Dimension == dimension && request.Bounds.Contains(p.X, p.Z) {
			position := [3]int32{p.X*16 + 8, 64, p.Z*16 + 8}
			spawn = &position
			break
		}
	}
	if sourceStore != nil {
		page, e := sourceStore.MapPage(ctx, scene.MapRequest{Bounds: &[4]int32{bounds.MinX, bounds.MinZ, bounds.MaxX, bounds.MaxZ}, Limit: 1, OmitCoverage: true}, request.Bounds.Contains)
		if e != nil {
			return result, e
		}
		if len(page.Tiles) > 0 {
			tile := page.Tiles[0]
			position := [3]int32{tile.X*16 + 8, 64, tile.Z*16 + 8}
			for i, height := range tile.Heights {
				if height != -32768 && (dimension != 1 || height < 125) && (dimension != 2 || height < 253) && (dimension != 0 || height < 317) {
					position = [3]int32{tile.X*16 + int32(i%16), int32(height) + 2, tile.Z*16 + int32(i/16)}
					break
				}
			}
			spawn = &position
		}
	}
	if spawn == nil {
		return result, fmt.Errorf("The selection contains no chunks in this dimension")
	}
	options.Settings.Spawn = spawn
	options.PlayerDimension = &dimension
	plan, err = worldexport.Preview(ctx, source, options)
	if err != nil {
		return result, err
	}
	if plan.KeptChunks == 0 {
		return result, fmt.Errorf("The selection contains no chunks in this dimension")
	}
	parent, err := filepath.Abs(utils.PathData("worlds", "regions"))
	if err != nil {
		return result, err
	}
	sourceAbsolute, err := filepath.Abs(source)
	if err != nil {
		return result, err
	}
	inside := func(base, path string) bool {
		relative, e := filepath.Rel(base, path)
		return e == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
	}
	if inside(sourceAbsolute, parent) {
		parent = filepath.Join(filepath.Dir(sourceAbsolute), "CraftBox regions")
	}
	if inside(sourceAbsolute, parent) {
		return result, fmt.Errorf("Choose a world folder below the drive root before creating a region")
	}
	if err = os.MkdirAll(parent, 0755); err != nil {
		return result, err
	}
	destination := filepath.Join(parent, utils.MakeValidFilename(request.Name)+"--"+uuid.NewString())
	if filepath.Dir(destination) != parent {
		return result, fmt.Errorf("Invalid region destination")
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(destination)
			os.Remove(destination + ".mcworld")
		}
	}()
	progress := func(p worldexport.Progress) {
		a.mu.Lock()
		a.worldExportStatus = WorldExportStatus{Progress: p, Running: true}
		a.mu.Unlock()
	}
	if err = worldexport.Copy(ctx, source, destination, options, progress); err != nil {
		return result, err
	}
	progress(worldexport.Progress{Stage: "archive", Total: 1})
	if err = compositionArchive(ctx, destination, destination+".mcworld"); err != nil {
		return result, err
	}
	progress(worldexport.Progress{Stage: "load", Total: plan.KeptChunks})
	scratch := NewApp()
	info, err := scratch.importRenderWorldDimension(ctx, destination, &dimension)
	if err != nil {
		return result, err
	}
	derived := scratch.renderDocument
	if err = ctx.Err(); err != nil {
		closeRenderDocument(derived)
		return result, err
	}
	a.mu.Lock()
	if a.scenes == nil {
		a.scenes = make(map[string]*scene.Store)
	}
	a.scenes[info.ID] = derived.store
	a.state.Worlds = append(a.state.Worlds, SavedWorld{ID: info.ID, Name: request.Name, Chunks: info.Chunks, State: "finished", CaptureFolder: destination, Path: destination + ".mcworld"})
	a.mu.Unlock()
	committed = true
	progress(worldexport.Progress{Stage: "complete", Completed: info.Chunks, Total: info.Chunks})
	a.publish()
	return RenderRegionResult{World: info, Path: destination + ".mcworld"}, nil
}

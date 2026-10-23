package worldstate

import (
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/entity"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/behaviourpack"
	"os"
	"path/filepath"
)

func (w *World) initScene() {
	if w.BlockRegistry == nil {
		return
	}
	paths := []string{os.Getenv("CRAFT_BOX_3D_ASSETS"), utils.PathData("scene-assets"), filepath.Join("assets", "vanilla-3d")}
	if exe, e := os.Executable(); e == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "assets", "vanilla-3d"))
	}
	catalog := scene.NewCatalog()
	for _, p := range paths {
		if p == "" {
			continue
		}
		if c, e := scene.LoadCatalog(p); e == nil {
			catalog = c
			break
		}
	}
	for _, original := range w.ResourcePacks {
		if pack, e := utils.ReadablePack(original); e == nil {
			if e = catalog.AddPack(pack); e != nil {
				w.log.WithError(e).Warn("3D resource pack could not be read")
			}
		}
	}
	w.scene = scene.NewStore(w.BlockRegistry, catalog)
	for _, entry := range w.BlockEntries {
		data, err := behaviourpack.BlockRenderJSON(entry)
		if err == nil {
			err = catalog.AddBlockDefinition(data)
		}
		if err != nil {
			w.log.WithError(err).Warn("3D custom block definition could not be read")
		}
	}
	event := &messages.EventSceneAvailable{WorldID: w.ID, Store: w.scene}
	messages.SendEvent(event)
	w.sceneRetained = event.Retained
	if w.LocalPlayerUniqueID != 0 {
		w.scene.PutActor(scene.Actor{ID: w.LocalPlayerUniqueID, Type: "minecraft:player", Scale: 1})
	}
	for _, ent := range w.memState.entities {
		w.updateSceneActor(ent)
	}
	for id := range w.scenePlayerIDs {
		w.scene.PutActor(scene.Actor{ID: id, Type: "minecraft:player", Scale: 1})
	}
	for root, links := range w.memState.entityLinks {
		for rider := range links {
			w.scene.LinkActor(root, rider, true)
		}
	}
}

func (w *World) RegisterScenePlayerID(uniqueID int64) {
	if uniqueID == 0 {
		return
	}
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if w.scenePlayerIDs == nil {
		w.scenePlayerIDs = map[int64]bool{}
	}
	w.scenePlayerIDs[uniqueID] = true
	if w.scene != nil {
		w.scene.PutActor(scene.Actor{ID: uniqueID, Type: "minecraft:player", Scale: 1})
	}
}

func (w *World) updateSceneActor(ent *entity.Entity) {
	if w.scene == nil {
		return
	}
	if w.SkipCustomEntities && entity.IsCustomType(ent.EntityType) {
		return
	}
	a := scene.ActorFromMetadata(ent.UniqueID, ent.EntityType, [3]float32(ent.Position), ent.Yaw, ent.Metadata, ent.Properties)
	w.scene.PutActor(a)
}

func (w *World) ReleaseScene() {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	if w.scene != nil && !w.sceneRetained {
		w.scene.Close()
		w.scene = nil
	}
}

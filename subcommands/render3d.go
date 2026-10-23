package subcommands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/commands"
	"github.com/Daniel-Ric/CraftBox/utils/merge"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"os"
	"path/filepath"
)

type Render3DSettings struct {
	WorldPath string `opt:"World Path" flag:"world"`
	Assets    string `opt:"Imported official 3D assets" flag:"assets" default:"assets/vanilla-3d"`
	Out       string `opt:"Output GLB filename" flag:"out" default:"world.glb"`
	Dimension int    `opt:"Dimension ID (0 overworld, 1 nether, 2 end)" flag:"dimension" default:"0"`
	SceneJSON string `opt:"Optional viewer snapshot JSON" flag:"scene-json"`
}
type Render3DCMD struct{}

func (Render3DCMD) Name() string        { return "render3d" }
func (Render3DCMD) Description() string { return "export a captured world as a textured GLB model" }
func (Render3DCMD) Settings() any       { return new(Render3DSettings) }
func (Render3DCMD) Run(ctx context.Context, value any) error {
	settings := value.(*Render3DSettings)
	if settings.WorldPath == "" || settings.Out == "" {
		return fmt.Errorf("--world and --out are required")
	}
	dimension, ok := world.DimensionByID(settings.Dimension)
	if !ok {
		return fmt.Errorf("unknown dimension %d", settings.Dimension)
	}
	catalog, err := scene.LoadCatalog(settings.Assets)
	if err != nil {
		return err
	}

	registry := &merge.BlockRegistry{BlockRegistry: world.DefaultBlockRegistry, Rids: make(map[uint32]merge.Block)}
	packs, _ := os.ReadDir(filepath.Join(settings.WorldPath, "resource_packs"))
	for _, p := range packs {
		pack, e := resource.ReadPath(filepath.Join(settings.WorldPath, "resource_packs", p.Name()))
		if e != nil {
			return e
		}
		readable, e := utils.ReadablePack(pack)
		if e != nil {
			return e
		}
		if e = catalog.AddPack(readable); e != nil {
			return e
		}
	}
	behaviors, _ := os.ReadDir(filepath.Join(settings.WorldPath, "behavior_packs"))
	for _, p := range behaviors {
		if !p.IsDir() {
			continue
		}
		if err = catalog.AddBlockDefinitions(os.DirFS(filepath.Join(settings.WorldPath, "behavior_packs", p.Name()))); err != nil {
			return err
		}
	}
	db, err := (mcdb.Config{Blocks: registry, LDBOptions: &opt.Options{ReadOnly: true}}).Open(settings.WorldPath)
	if err != nil {
		return err
	}
	defer db.Close()
	store := scene.NewStore(registry, catalog)
	defer store.Close()
	iterator := db.NewColumnIterator(&mcdb.IteratorRange{Dimension: dimension})
	defer iterator.Release()
	count := 0
	for iterator.Next() {
		if err = ctx.Err(); err != nil {
			return err
		}
		count++
		if count > 4096 {
			return fmt.Errorf("world exceeds 4096 chunks; export a smaller region")
		}
		column := iterator.Column()
		entities := map[cube.Pos]map[string]any{}
		for _, e := range column.BlockEntities {
			entities[e.Pos] = e.Data
		}
		store.Put(iterator.Position(), column.Chunk, entities)
		for _, e := range column.Entities {
			store.PutActor(scene.ActorFromNBT(e.ID, e.Data))
		}
	}
	if err = iterator.Error(); err != nil {
		return err
	}
	snapshot, err := store.Snapshot(0, nil)
	if err != nil {
		return err
	}
	data, err := scene.EncodeGLB(snapshot)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(settings.Out), 0755); err != nil {
		return err
	}
	if err = os.WriteFile(settings.Out, data, 0644); err != nil {
		return err
	}
	if settings.SceneJSON != "" {
		js, e := json.Marshal(snapshot)
		if e != nil {
			return e
		}
		if e = os.WriteFile(settings.SceneJSON, js, 0644); e != nil {
			return e
		}
	}
	fmt.Printf("Exported %d chunks to %s\n", count, settings.Out)
	return nil
}
func init() { commands.RegisterCommand(Render3DCMD{}) }

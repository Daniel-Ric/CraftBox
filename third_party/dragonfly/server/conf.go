package server

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/internal/packbuilder"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/playerdb"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/generator"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type Config struct {
	Log *slog.Logger

	Listeners []func(conf Config) (Listener, error)

	Name string

	Resources []*resource.Pack

	ResourcesRequired bool

	DisableResourceBuilding bool

	Allower Allower

	AuthDisabled bool

	MuteEmoteChat bool

	MaxPlayers int

	MaxChunkRadius int

	JoinMessage, QuitMessage, ShutdownMessage chat.Translation

	StatusProvider minecraft.ServerStatusProvider

	Compression packet.Compression

	PlayerProvider player.Provider

	WorldProvider world.Provider

	ReadOnlyWorld bool

	Generator func(dim world.Dimension) world.Generator

	RandomTickSpeed int

	SaveInterval time.Duration

	ChunkUnloadInterval time.Duration

	ChunkLoadWorkers int

	Entities world.EntityRegistry

	Blocks world.BlockRegistry
}

func (conf Config) New() *Server {
	if conf.Log == nil {
		conf.Log = slog.Default()
	}
	if len(conf.Listeners) == 0 {
		conf.Log.Warn("config: no listeners set, no connections will be accepted")
	}
	if conf.Name == "" {
		conf.Name = "Dragonfly Server"
	}
	if conf.StatusProvider == nil {
		conf.StatusProvider = statusProvider{name: conf.Name}
	}
	if conf.PlayerProvider == nil {
		conf.PlayerProvider = player.NopProvider{}
	}
	if conf.Allower == nil {
		conf.Allower = allower{}
	}
	if conf.WorldProvider == nil {
		conf.WorldProvider = world.NopProvider{}
	}
	if conf.Generator == nil {
		conf.Generator = loadGenerator
	}
	if conf.MaxChunkRadius == 0 {
		conf.MaxChunkRadius = 12
	}
	if conf.ShutdownMessage.Zero() {
		conf.ShutdownMessage = chat.MessageServerDisconnect
	}
	if len(conf.Entities.Types()) == 0 {
		conf.Entities = entity.DefaultRegistry
	}
	if conf.Blocks == nil {
		conf.Blocks = world.DefaultBlockRegistry
	}

	conf.Blocks.Finalize()
	world.DefaultBlockRegistry.Finalize()

	if !conf.DisableResourceBuilding {
		if pack, ok := packbuilder.BuildResourcePack(conf.Blocks); ok {
			conf.Resources = append(conf.Resources, pack)
		}
	}

	conf.Resources = slices.Clone(conf.Resources)

	srv := &Server{
		conf:     conf,
		incoming: make(chan incoming),
		p:        make(map[uuid.UUID]*onlinePlayer),
		world:    &world.World{}, nether: &world.World{}, end: &world.World{},
	}
	for _, lf := range conf.Listeners {
		l, err := lf(conf)
		if err != nil {
			conf.Log.Error("create listener: " + err.Error())
			continue
		}
		srv.listeners = append(srv.listeners, l)
	}
	if len(conf.Listeners) > 0 && len(srv.listeners) == 0 {
		conf.Log.Error("no listeners could be created; the server is unreachable")
	}

	creative_registerCreativeItems()
	recipe_registerVanilla()

	srv.world = srv.createWorld(world.Overworld, &srv.nether, &srv.end)
	srv.nether = srv.createWorld(world.Nether, &srv.world, &srv.end)
	srv.end = srv.createWorld(world.End, &srv.nether, &srv.world)

	return srv
}

type UserConfig struct {
	Network struct {
		Address string

		Transport []string

		NetherNet struct {
			Address string

			KeyFile string

			Domain string

			UDPPorts string
		}
	}
	Server struct {
		Name string

		AuthEnabled bool

		DisableJoinQuitMessages bool

		MuteEmoteChat bool
	}
	World struct {
		SaveData bool

		Folder string
	}
	Players struct {
		MaxCount int

		MaximumChunkRadius int

		SaveData bool

		Folder string
	}
	Resources struct {
		AutoBuildPack bool

		Folder string

		Required bool
	}
}

func (uc UserConfig) Config(log *slog.Logger) (Config, error) {
	var err error
	conf := Config{
		Log:                     log,
		Name:                    uc.Server.Name,
		ResourcesRequired:       uc.Resources.Required,
		AuthDisabled:            !uc.Server.AuthEnabled,
		MuteEmoteChat:           uc.Server.MuteEmoteChat,
		MaxPlayers:              uc.Players.MaxCount,
		MaxChunkRadius:          uc.Players.MaximumChunkRadius,
		DisableResourceBuilding: !uc.Resources.AutoBuildPack,
	}
	listeners, err := uc.transportListeners()
	if err != nil {
		return conf, err
	}
	if !uc.Server.DisableJoinQuitMessages {
		conf.JoinMessage, conf.QuitMessage = chat.MessageJoin, chat.MessageQuit
	}
	if uc.World.SaveData {
		conf.WorldProvider, err = mcdb.Config{Log: log}.Open(uc.World.Folder)
		if err != nil {
			return conf, fmt.Errorf("create world provider: %w", err)
		}
	}
	conf.Resources, err = loadResources(uc.Resources.Folder)
	if err != nil {
		return conf, fmt.Errorf("load resources: %w", err)
	}
	if uc.Players.SaveData {
		conf.PlayerProvider, err = playerdb.NewProvider(uc.Players.Folder)
		if err != nil {
			return conf, fmt.Errorf("create player provider: %w", err)
		}
	}
	conf.Listeners = append(conf.Listeners, listeners...)
	return conf, nil
}

func (uc UserConfig) transportListeners() ([]func(Config) (Listener, error), error) {
	names := uc.Network.Transport
	if len(names) == 0 {
		names = []string{"nethernet"}
	}
	var listeners []func(Config) (Listener, error)
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if seen[name] {
			continue
		}
		seen[name] = true
		switch name {
		case "nethernet":
			listeners = append(listeners, uc.netherNetListenerFunc)
		default:
			return nil, fmt.Errorf("unknown network transport %q", name)
		}
	}
	return listeners, nil
}

func loadResources(dir string) ([]*resource.Pack, error) {
	_ = os.MkdirAll(dir, 0777)

	resources, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	packs := make([]*resource.Pack, len(resources))
	for i, entry := range resources {
		packs[i], err = resource.ReadPath(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("compile resource (%v): %w", entry.Name(), err)
		}
	}
	return packs, nil
}

func loadGenerator(dim world.Dimension) world.Generator {
	switch dim {
	case world.Overworld:
		return generator.NewFlat(biome.Plains{}, []world.Block{block.Grass{}, block.Dirt{}, block.Dirt{}, block.Bedrock{}})
	case world.Nether:
		return generator.NewFlat(biome.NetherWastes{}, []world.Block{block.Netherrack{}, block.Netherrack{}, block.Netherrack{}, block.Bedrock{}})
	case world.End:
		return generator.NewFlat(biome.End{}, []world.Block{block.EndStone{}, block.EndStone{}, block.EndStone{}, block.Bedrock{}})
	}
	panic("should never happen")
}

func DefaultConfig() UserConfig {
	c := UserConfig{}
	c.Network.Address = ":19132"
	c.Network.Transport = []string{"nethernet"}
	c.Network.NetherNet.KeyFile = "keys/server_identity_key.pem"
	c.Network.NetherNet.Domain = "self"
	c.Network.NetherNet.UDPPorts = "19133"
	c.Server.Name = "Dragonfly Server"
	c.Server.AuthEnabled = true
	c.World.SaveData = true
	c.World.Folder = "world"
	c.Players.MaximumChunkRadius = 32
	c.Players.SaveData = true
	c.Players.Folder = "players"
	c.Resources.AutoBuildPack = true
	c.Resources.Folder = "resources"
	c.Resources.Required = false
	return c
}

//go:linkname creative_registerCreativeItems github.com/df-mc/dragonfly/server/item/creative.registerCreativeItems
func creative_registerCreativeItems()

//go:linkname recipe_registerVanilla github.com/df-mc/dragonfly/server/item/recipe.registerVanilla
func recipe_registerVanilla()

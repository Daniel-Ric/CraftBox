package worlds

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sirupsen/logrus"
)

type dimensionMerge struct {
	mu sync.Mutex

	mainFolder    string
	mainID        string
	mainDimension string
	mainTitle     string
	mainSpawn     mgl32.Vec3
	dims          []worldstate.MergedDimension
	taken         map[string]bool
}

func (w *worldsHandler) mergeSavedWorld(m *dimensionMerge, worldState *worldstate.World) (merged bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mainID != "" && messages.WorldDeleted(m.mainID) {
		m.mainFolder = ""
		m.dims = nil
		m.taken = make(map[string]bool)
	}
	if m.mainFolder == "" {
		return w.mergeSavedWorldLocked(m, worldState)
	}
	err = messages.WithCaptureWrite(m.mainID, func() error {
		merged, err = w.mergeSavedWorldLocked(m, worldState)
		return err
	})
	return merged, err
}

func (w *worldsHandler) mergeSavedWorldLocked(m *dimensionMerge, worldState *worldstate.World) (bool, error) {

	if m.mainFolder == "" {
		m.mainFolder = worldState.Folder
		m.mainID = worldState.ID
		id, _ := world.DimensionID(worldState.Dimension())
		switch id {
		case 1:
			m.mainDimension = "minecraft:nether"
		case 2:
			m.mainDimension = "minecraft:the_end"
		default:
			m.mainDimension = "minecraft:overworld"
		}
		m.mainTitle = worldState.Name
		m.mainSpawn = worldState.Spawn()
		return false, nil
	}

	dim := worldstate.MergedDimension{
		ID:         worldstate.FirstCustomDimensionID + len(m.dims),
		Identifier: worldstate.DimensionIdentifier(worldState.Name, m.taken),
		Title:      worldState.Name,
		Spawn:      worldState.Spawn(),
		Range:      worldstate.TargetRange(worldState.Range()),
	}
	srcDim, _ := world.DimensionID(worldState.Dimension())

	w.log.Info(locale.Loc("worlds_merging_dimension", locale.Strmap{
		"Name": worldState.Name, "Main": m.mainTitle, "Dimension": dim.Identifier,
	}))
	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   worldState.ID,
		WorldName: worldState.Name,
		State:     "Merging into " + m.mainTitle,
	})

	stats, err := worldstate.MergeWorld(m.mainFolder, worldState.Folder, srcDim, worldState.Range(), dim)
	if err != nil {
		return false, err
	}
	if err := worldstate.MergePacks(m.mainFolder, worldState.Folder); err != nil {
		return false, err
	}
	if err := scopeMergedUI(m.mainFolder, worldState.Folder, dim.Identifier); err != nil {
		return false, err
	}
	m.taken[dim.Identifier] = true
	m.dims = append(m.dims, dim)

	err = worldstate.WriteDimensionPack(m.mainFolder, utils.MakeValidFilename(w.serverState.serverName+" "+m.mainTitle), m.mainTitle, m.mainSpawn, m.dims, m.mainDimension)
	if err != nil {
		return true, err
	}

	filename := m.mainFolder + ".mcworld"
	if err := zipWorld(m.mainFolder, filename); err != nil {
		return true, err
	}

	w.log.WithFields(logrus.Fields{
		"Chunks":   stats.Chunks,
		"Entities": stats.Entities,
		"Maps":     stats.Maps,
	}).Info(locale.Loc("worlds_merged_dimension", locale.Strmap{
		"Name":      worldState.Name,
		"Dimension": dim.Identifier,
		"File":      filename,
		"Command":   fmt.Sprintf("/%s %s", worldstate.DimensionCommand, dim.ShortName()),
	}))
	if stats.RenamedIDs > 0 {
		w.log.Debugf("%d entities got new unique ids", stats.RenamedIDs)
	}
	if len(m.dims) == 1 {
		w.log.Info(locale.Loc("worlds_merge_requirements", locale.Strmap{"Command": "/" + worldstate.DimensionCommand}))
	}

	total, _ := worldState.EntityCounts(w.serverState.getEntityRenderDistance())
	messages.SendEvent(&messages.EventFinishedSavingWorld{
		WorldID:   m.mainID,
		WorldName: m.mainTitle,
		Filepath:  filename,
		Chunks:    len(worldState.StoredChunks),
		Entities:  total,
	})
	return true, nil
}

func zipWorld(folder, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	utils.ZipCompressPool(z)
	if err := z.AddFS(os.DirFS(folder)); err != nil {
		z.Close()
		return err
	}
	return z.Close()
}

func scopeMergedUI(destination, source, dimension string) error {
	scripts, err := filepath.Glob(filepath.Join(source, "behavior_packs", "*", "scripts", "main.js"))
	if err != nil {
		return err
	}
	encoded, _ := json.Marshal(dimension)
	for _, file := range scripts {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		text := string(data)
		const marker = "const CAPTURE_DIMENSION = "
		start := strings.Index(text, marker)
		if start < 0 {
			continue
		}
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			return fmt.Errorf("invalid recorded UI script: %s", file)
		}
		text = text[:start] + marker + string(encoded) + ";" + text[start+end:]
		relative, err := filepath.Rel(source, file)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destination, relative), []byte(text), 0644); err != nil {
			return err
		}
	}
	return nil
}

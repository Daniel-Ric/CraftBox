package worldstate

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sandertv/gophertunnel/minecraft/text"
	"github.com/sirupsen/logrus"
)

func addPacksJSON(fs utils.WriterFS, name string, deps []resourcePackDependency) error {
	f, err := fs.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(deps); err != nil {
		return err
	}
	return nil
}

func (w *World) addResourcePacks() error {
	w.exportedResourcePacks = nil
	for _, original := range w.ResourcePacks {
		pack, err := utils.ReadablePack(original)
		if err != nil {
			return fmt.Errorf("Cannot export pack %s: %w", original.Name(), err)
		}
		log := w.log.WithField("pack", pack.Name())
		logrus.Info(locale.Loc("adding_pack", locale.Strmap{"Name": text.Clean(pack.Name())}))

		messages.SendEvent(&messages.EventProcessingWorldUpdate{
			WorldID:   w.ID,
			WorldName: w.Name,
			State:     "Adding Resourcepack " + text.Clean(pack.Name()),
		})

		packName := utils.FormatPackName(pack.Name()) + "--" + pack.UUID().String()

		err = utils.CopyFS(pack, utils.SubFS(utils.OSWriter{Base: w.Folder}, path.Join("resource_packs", packName)))
		if err != nil {
			log.Error(err)
			return err
		}
		w.exportedResourcePacks = append(w.exportedResourcePacks, pack)
	}

	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   w.ID,
		WorldName: w.Name,
		State:     "",
	})

	return nil
}

type addedPack struct {
	BehaviorPack bool
	Header       *resource.Header
}

func exportPackDependencies(pack *resource.Pack, target utils.WriterFS, available map[string]bool) error {
	data, err := fs.ReadFile(pack, "manifest.json")
	if err != nil {
		return err
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if _, ok := manifest["dependencies"]; !ok {
		return nil
	}
	var dependencies []json.RawMessage
	if err := json.Unmarshal(manifest["dependencies"], &dependencies); err != nil {
		return err
	}
	retained := make([]json.RawMessage, 0, len(dependencies))
	for _, raw := range dependencies {
		var dependency struct {
			UUID string `json:"uuid"`
		}
		if err := json.Unmarshal(raw, &dependency); err != nil {
			return err
		}
		if dependency.UUID == "" || available[strings.ToLower(dependency.UUID)] {
			retained = append(retained, raw)
		}
	}
	if len(retained) == len(dependencies) {
		return nil
	}
	if len(retained) == 0 {
		delete(manifest, "dependencies")
	} else {
		manifest["dependencies"], err = json.Marshal(retained)
		if err != nil {
			return err
		}
	}
	f, err := target.Create("manifest.json")
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(manifest); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (w *World) finalizePacks(addAdditionalPacks func(fs utils.WriterFS) ([]addedPack, error)) error {
	err := <-w.resourcePacksDone
	if err != nil {
		return err
	}

	messages.SendEvent(&messages.EventProcessingWorldUpdate{
		WorldID:   w.ID,
		WorldName: w.Name,
		State:     "Adding Behaviorpack",
	})

	fs := utils.OSWriter{Base: w.Folder}
	additionalPacks, err := addAdditionalPacks(fs)
	if err != nil {
		return err
	}

	var resourcePackDependencies []resourcePackDependency
	for _, pack := range w.exportedResourcePacks {
		resourcePackDependencies = append(resourcePackDependencies, resourcePackDependency{
			UUID:    pack.Manifest().Header.UUID.String(),
			Version: pack.Manifest().Header.Version,
		})
	}

	var behaviorPackDependencies []resourcePackDependency
	for _, p := range additionalPacks {
		dep := resourcePackDependency{
			UUID:    p.Header.UUID.String(),
			Version: p.Header.Version,
		}
		if p.BehaviorPack {
			behaviorPackDependencies = append(behaviorPackDependencies, dep)
		} else {
			resourcePackDependencies = append(resourcePackDependencies, dep)
		}
	}

	available := make(map[string]bool, len(resourcePackDependencies)+len(behaviorPackDependencies))
	for _, dependencies := range [][]resourcePackDependency{resourcePackDependencies, behaviorPackDependencies} {
		for _, dependency := range dependencies {
			available[strings.ToLower(dependency.UUID)] = true
		}
	}
	for _, pack := range w.exportedResourcePacks {
		packName := utils.FormatPackName(pack.Name()) + "--" + pack.UUID().String()
		if err := exportPackDependencies(pack, utils.SubFS(fs, path.Join("resource_packs", packName)), available); err != nil {
			return fmt.Errorf("finalize dependencies for pack %s: %w", pack.Name(), err)
		}
	}

	if len(behaviorPackDependencies) > 0 {
		err := addPacksJSON(fs, "world_behavior_packs.json", behaviorPackDependencies)
		if err != nil {
			return err
		}
	}

	if len(resourcePackDependencies) > 0 {
		err := addPacksJSON(fs, "world_resource_packs.json", resourcePackDependencies)
		if err != nil {
			return err
		}
	}

	return nil
}

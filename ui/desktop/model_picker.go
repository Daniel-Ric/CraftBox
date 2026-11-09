package desktop

import (
	"context"
	"errors"
	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/modelpicker"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"strings"
)

type ModelPickResult struct {
	Kind       string         `json:"kind"`
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Object     string         `json:"object"`
	Geometry   string         `json:"geometry"`
	RuntimeID  uint64         `json:"runtimeId"`
	TargetType string         `json:"targetType"`
	States     map[string]any `json:"states,omitempty"`
}

var watchModelPickerKey = func(context.Context, func()) {}

func (a *App) PickMinecraftModel() (ModelPickResult, error) {
	a.packetMu.Lock()
	read, session := a.modelPickRead, a.modelPickSession
	if a.modelPickBusy {
		a.packetMu.Unlock()
		return ModelPickResult{}, errors.New("A model is already being resolved")
	}
	if read == nil {
		a.packetMu.Unlock()
		return ModelPickResult{}, errors.New("Join a server with Minecraft before picking a model")
	}
	a.modelPickBusy = true
	a.packetMu.Unlock()
	defer func() { a.packetMu.Lock(); a.modelPickBusy = false; a.packetMu.Unlock() }()
	result, err := read()
	if err != nil {
		return result, err
	}
	a.packetMu.Lock()
	sameSession := session == a.modelPickSession
	a.packetMu.Unlock()
	if !sameSession {
		return ModelPickResult{}, errors.New("The Minecraft session changed while resolving the model")
	}
	a.mu.Lock()
	emit := a.emit
	a.mu.Unlock()
	if emit != nil {
		emit("desktop:model-pick", result)
	}
	if a.ctx != nil && proxy.IngameInterfaceSettings().ModelPickFocus {
		wruntime.WindowShow(a.ctx)
		wruntime.WindowUnminimise(a.ctx)
	}
	return result, nil
}
func (a *App) startModelPicker(s *proxy.Session, tracker *modelpicker.Tracker, session string) {
	a.packetMu.Lock()
	a.modelPickSession = session
	a.clearHotbarFeedback = s.ClearHotbarFeedback
	a.modelPickRead = func() (ModelPickResult, error) {
		if s.SessionContext().Err() != nil {
			return ModelPickResult{}, errors.New("The Minecraft session has ended")
		}
		player := s.PlayerSnapshot()
		actor, actorErr := tracker.Pick(player.Position, player.Pitch, player.Yaw)
		block, blockErr := s.PickBlock()
		if blockErr == nil && (actorErr != nil || block.Distance < actor.Distance) {
			return a.resolvePickedBlock(block, session)
		}
		if actorErr != nil {
			if blockErr != nil {
				return ModelPickResult{}, blockErr
			}
			return ModelPickResult{}, actorErr
		}
		return a.resolvePickedActor(actor, session)
	}
	a.packetMu.Unlock()
	a.warmModelPicker()
	watchModelPickerKey(s.SessionContext(), func() {
		if _, err := a.PickMinecraftModel(); err != nil {
			_ = s.SendModelPickFeedback("§eCraftBox: " + err.Error())
		}
	})
}
func definitionPriority(file, subpack string) int {
	if strings.HasPrefix(file, "subpacks/") {
		if subpack != "" && strings.HasPrefix(file, "subpacks/"+subpack+"/") {
			return 10
		}
		return -1
	}
	return 0
}
func (a *App) modelCandidates(identifier, kind, session string) (string, []assetlibrary.Object, error) {
	stack, err := a.commandExplorer().ResourcePacks(session)
	if err != nil {
		return "", nil, err
	}
	previews, err := a.modelSources()
	if err != nil {
		return "", nil, err
	}
	for i := len(stack) - 1; i >= 0; i-- {
		for _, preview := range previews {
			if preview.Source == nil || !strings.EqualFold(preview.Item.UUID, stack[i].UUID) || preview.Item.Version != stack[i].Version {
				continue
			}
			objects, err := a.modelObjects(preview.Item.ID)
			if err != nil {
				return "", nil, err
			}
			matches := []assetlibrary.Object{}
			priority := -1
			for _, object := range objects {
				name := object.Name
				if kind == "block" && strings.HasPrefix(identifier, "minecraft:") && !strings.Contains(strings.SplitN(name, " / ", 2)[0], ":") {
					name = "minecraft:" + name
				}
				if object.Kind != kind || (name != identifier && !strings.HasPrefix(name, identifier+" / ")) {
					continue
				}
				rank := definitionPriority(object.Definition, stack[i].SubPackName)
				if rank < 0 || rank < priority {
					continue
				}
				if rank > priority {
					matches = nil
					priority = rank
				}
				matches = append(matches, object)
			}
			if len(matches) > 0 {
				return preview.Item.ID, matches, nil
			}
		}
	}
	return "", nil, errors.New("No readable captured " + kind + " model for " + identifier + " exists in the active server packs")
}
func (a *App) resolvePickedActor(actor modelpicker.Actor, session string) (ModelPickResult, error) {
	pack, candidates, err := a.modelCandidates(actor.Identifier, "entity", session)
	if err != nil {
		return ModelPickResult{}, err
	}
	var chosen *assetlibrary.Object
	best := -1
	values := map[string]float64{"query.variant": float64(actor.Variant), "query.mark_variant": float64(actor.MarkVariant), "query.skin_id": float64(actor.SkinID)}
	for _, object := range candidates {
		score := 0
		for query, want := range object.PreviewQueries {
			if value, known := values[query]; known {
				if value == want {
					score++
				} else {
					score = -100
					break
				}
			}
		}
		if score > best {
			copy := object
			chosen = &copy
			best = score
		}
	}
	if chosen == nil {
		return ModelPickResult{}, errors.New("The entity's live model variant is not available in the captured definitions")
	}
	return ModelPickResult{Kind: "packs", ID: pack, Name: actor.Identifier, Object: chosen.ID, Geometry: chosen.GeometryID, RuntimeID: actor.RuntimeID, TargetType: "entity"}, nil
}
func (a *App) resolvePickedBlock(block modelpicker.BlockTarget, session string) (ModelPickResult, error) {
	pack, candidates, err := a.modelCandidates(block.Identifier, "block", session)
	if err != nil {
		return ModelPickResult{}, err
	}
	var chosen *assetlibrary.Object
	for _, object := range candidates {
		if object.Condition != "" {
			matches, supported := scene.MatchBlockCondition(object.Condition, block.States)
			if !supported || !matches {
				continue
			}
		}
		copy := object
		if chosen == nil || object.Permutation >= chosen.Permutation {
			chosen = &copy
		}
	}
	if chosen == nil {
		return ModelPickResult{}, errors.New("The block's live model state is not available in the captured definitions")
	}
	return ModelPickResult{Kind: "packs", ID: pack, Name: block.Identifier, Object: chosen.ID, Geometry: chosen.GeometryID, RuntimeID: uint64(block.RuntimeID), TargetType: "block", States: block.States}, nil
}

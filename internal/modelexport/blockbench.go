package modelexport

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateBlockbenchProject(raw []byte) error {
	if len(raw) > 72<<20 {
		return fmt.Errorf("Blockbench project exceeds 72 MiB")
	}
	var project struct {
		Meta struct {
			Version string `json:"format_version"`
			Format  string `json:"model_format"`
		} `json:"meta"`
		Groups []struct {
			UUID string `json:"uuid"`
		} `json:"groups"`
		Elements []struct {
			UUID string `json:"uuid"`
		} `json:"elements"`
		Outliner []json.RawMessage `json:"outliner"`
		Textures []struct {
			Source       string `json:"source"`
			Path         string `json:"path"`
			RelativePath string `json:"relative_path"`
		} `json:"textures"`
		Animations []struct {
			UUID      string  `json:"uuid"`
			Length    float64 `json:"length"`
			Animators map[string]struct {
				Keyframes []struct {
					UUID    string  `json:"uuid"`
					Time    float64 `json:"time"`
					Channel string  `json:"channel"`
				} `json:"keyframes"`
			} `json:"animators"`
		} `json:"animations"`
		Controllers []struct {
			UUID    string `json:"uuid"`
			Initial string `json:"initial_state"`
			States  []struct {
				UUID       string `json:"uuid"`
				Animations []struct {
					UUID      string `json:"uuid"`
					Animation string `json:"animation"`
				} `json:"animations"`
				Transitions []struct {
					UUID   string `json:"uuid"`
					Target string `json:"target"`
				} `json:"transitions"`
			} `json:"states"`
		} `json:"animation_controllers"`
	}
	if err := json.Unmarshal(raw, &project); err != nil {
		return fmt.Errorf("Invalid Blockbench project: %w", err)
	}
	if project.Meta.Version != "5.0" || (project.Meta.Format != "bedrock" && project.Meta.Format != "bedrock_block" && project.Meta.Format != "free") {
		return fmt.Errorf("Unsupported Blockbench project format")
	}
	if len(project.Groups) > 10100 || len(project.Elements) > 50000 || len(project.Textures) > 256 {
		return fmt.Errorf("Blockbench project exceeds editable model limits")
	}
	ids := map[string]bool{}
	for _, group := range project.Groups {
		if group.UUID == "" || ids[group.UUID] {
			return fmt.Errorf("Duplicate or missing model node identifier")
		}
		ids[group.UUID] = true
	}
	for _, element := range project.Elements {
		if element.UUID == "" || ids[element.UUID] {
			return fmt.Errorf("Duplicate or missing model node identifier")
		}
		ids[element.UUID] = true
	}
	seen := map[string]bool{}
	var visit func([]json.RawMessage, int) error
	visit = func(nodes []json.RawMessage, depth int) error {
		if depth > 512 {
			return fmt.Errorf("Model hierarchy exceeds 512 levels")
		}
		for _, raw := range nodes {
			var ref string
			var children []json.RawMessage
			if json.Unmarshal(raw, &ref) != nil {
				var group struct {
					UUID     string            `json:"uuid"`
					Children []json.RawMessage `json:"children"`
				}
				if err := json.Unmarshal(raw, &group); err != nil {
					return err
				}
				ref, children = group.UUID, group.Children
			}
			if !ids[ref] || seen[ref] {
				return fmt.Errorf("Invalid or repeated hierarchy reference")
			}
			seen[ref] = true
			if err := visit(children, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(project.Outliner, 0); err != nil {
		return err
	}
	if len(seen) != len(ids) {
		return fmt.Errorf("Model contains nodes missing from the hierarchy")
	}
	for _, texture := range project.Textures {
		if !strings.HasPrefix(texture.Source, "data:image/png;base64,") || texture.Path != "" || texture.RelativePath != "" {
			return fmt.Errorf("Blockbench textures must be embedded PNG images")
		}
	}
	if len(project.Animations) > 4096 || len(project.Controllers) > 1024 {
		return fmt.Errorf("Blockbench project exceeds editable animation graph limits")
	}
	reserve := func(id string) error {
		if id == "" || ids[id] {
			return fmt.Errorf("Duplicate or missing animation graph identifier")
		}
		ids[id] = true
		return nil
	}
	animations := map[string]bool{}
	keyframes, states := 0, 0
	for _, animation := range project.Animations {
		if err := reserve(animation.UUID); err != nil {
			return err
		}
		animations[animation.UUID] = true
		if animation.Length < 0 {
			return fmt.Errorf("Invalid animation length")
		}
		for _, animator := range animation.Animators {
			keyframes += len(animator.Keyframes)
			for _, keyframe := range animator.Keyframes {
				if err := reserve(keyframe.UUID); err != nil {
					return err
				}
				if keyframe.Time < 0 {
					return fmt.Errorf("Invalid animation keyframe time")
				}
				switch keyframe.Channel {
				case "position", "rotation", "scale", "sound", "particle", "timeline":
				default:
					return fmt.Errorf("Invalid animation channel")
				}
			}
		}
	}
	for _, controller := range project.Controllers {
		if err := reserve(controller.UUID); err != nil {
			return err
		}
		stateIDs := map[string]bool{}
		states += len(controller.States)
		for _, state := range controller.States {
			if err := reserve(state.UUID); err != nil {
				return err
			}
			stateIDs[state.UUID] = true
		}
		if controller.Initial != "" && !stateIDs[controller.Initial] {
			return fmt.Errorf("Invalid animation controller initial state")
		}
		for _, state := range controller.States {
			for _, link := range state.Animations {
				if err := reserve(link.UUID); err != nil {
					return err
				}
				if link.Animation != "" && !animations[link.Animation] {
					return fmt.Errorf("Invalid controller animation reference")
				}
			}
			for _, transition := range state.Transitions {
				if err := reserve(transition.UUID); err != nil {
					return err
				}
				if transition.Target != "" && !stateIDs[transition.Target] {
					return fmt.Errorf("Invalid controller transition reference")
				}
			}
		}
	}
	if keyframes > 500000 || states > 16384 {
		return fmt.Errorf("Blockbench project exceeds editable animation graph limits")
	}
	return nil
}

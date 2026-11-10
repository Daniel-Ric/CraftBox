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
	return nil
}

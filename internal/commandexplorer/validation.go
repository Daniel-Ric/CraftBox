package commandexplorer

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func validateEvent(e *Event) error {
	if len(e.Data) > 16<<20 || !json.Valid(e.Data) || len(e.Summary) > 1<<20 {
		return errors.New("Invalid server archive event")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(e.Data, &object) != nil || object == nil {
		return errors.New("Server archive event must contain an object")
	}
	if e.Direction != "clientbound" && e.Direction != "serverbound" {
		return errors.New("Invalid event direction")
	}
	switch e.Kind {
	case "commands":
		var snapshot commandSnapshot
		if json.Unmarshal(e.Data, &snapshot) != nil || snapshot.Commands == nil || len(snapshot.Commands) > 20000 {
			return errors.New("Invalid command definitions")
		}
		seen := map[string]bool{}
		for i := range snapshot.Commands {
			c := &snapshot.Commands[i]
			if c.Name == "" || len(c.Name) > 256 || strings.ContainsAny(c.Name, "\r\n\x00") || seen[c.Name] || len(c.Overloads) > 128 {
				return errors.New("Invalid command definition")
			}
			seen[c.Name] = true
			if c.Aliases == nil {
				c.Aliases = []string{}
			}
			if c.Chained == nil {
				c.Chained = []string{}
			}
			if c.Overloads == nil {
				c.Overloads = [][]Parameter{}
			}
			for j := range c.Overloads {
				if c.Overloads[j] == nil {
					c.Overloads[j] = []Parameter{}
				}
				if len(c.Overloads[j]) > 256 {
					return errors.New("Command parameter limit exceeded")
				}
				for k := range c.Overloads[j] {
					p := &c.Overloads[j][k]
					if len(p.Values) > 100000 {
						return errors.New("Command enum limit exceeded")
					}
					if p.Values == nil {
						p.Values = []string{}
					}
				}
			}
		}
		raw, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		e.Data = raw
	case "enum":
		var p packet.UpdateSoftEnum
		if json.Unmarshal(e.Data, &p) != nil || p.EnumType == "" || p.ActionType > packet.SoftEnumActionSet || len(p.Options) > 100000 {
			return errors.New("Invalid enum update")
		}
	case "packs":
		var p packet.ResourcePackStack
		if json.Unmarshal(e.Data, &p) != nil || len(p.TexturePacks) > 256 {
			return errors.New("Invalid resource pack stack")
		}
	case "command":
		var p packet.CommandRequest
		if json.Unmarshal(e.Data, &p) != nil || len(p.CommandLine) > 32768 {
			return errors.New("Invalid command observation")
		}
	case "output":
		var p packet.CommandOutput
		if json.Unmarshal(e.Data, &p) != nil {
			return errors.New("Invalid command output")
		}
	default:
		return errors.New("Unsupported server archive event kind")
	}
	return nil
}

func dialogEvent(kind string) bool {
	switch kind {
	case "form", "response", "close", "npc", "npc-close", "npc-response":
		return true
	}
	return false
}

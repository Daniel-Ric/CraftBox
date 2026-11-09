package commandexplorer

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Session struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capture  string `json:"capture"`
	Protocol int    `json:"protocol"`
	Started  int64  `json:"started"`
	Ended    int64  `json:"ended"`
	Updated  int64  `json:"updated"`
	Revision uint64 `json:"revision"`
	Commands int    `json:"commands"`
	Events   int    `json:"events"`
	Bytes    int64  `json:"bytes"`
	Error    string `json:"error,omitempty"`
}
type Parameter struct {
	Name     string   `json:"name"`
	Type     uint32   `json:"type"`
	Kind     string   `json:"kind"`
	Optional bool     `json:"optional"`
	Options  byte     `json:"options"`
	Enum     string   `json:"enum,omitempty"`
	Dynamic  bool     `json:"dynamic"`
	Values   []string `json:"values"`
}
type Command struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Permission  byte          `json:"permission"`
	Flags       uint16        `json:"flags"`
	Aliases     []string      `json:"aliases"`
	Overloads   [][]Parameter `json:"overloads"`
	Chained     []string      `json:"chained"`
	Time        int64         `json:"time"`
	Event       string        `json:"event"`
}
type Event struct {
	ID        string          `json:"id"`
	Time      int64           `json:"time"`
	Kind      string          `json:"kind"`
	Name      string          `json:"name"`
	Direction string          `json:"direction"`
	PacketID  uint32          `json:"packetId"`
	Packet    string          `json:"packet,omitempty"`
	Summary   string          `json:"summary"`
	Data      json.RawMessage `json:"data"`
}
type Document struct {
	Format   string    `json:"format"`
	Version  int       `json:"version"`
	Session  Session   `json:"session"`
	Commands []Command `json:"commands"`
	Events   []Event   `json:"events"`
	Warnings []string  `json:"warnings"`
}
type commandSnapshot struct {
	Commands []Command `json:"commands"`
}

func parameterKind(t uint32) string {
	switch t & 0xfffff {
	case protocol.CommandArgTypeInt:
		return "integer"
	case protocol.CommandArgTypeFloat, protocol.CommandArgTypeRValue:
		return "number"
	case protocol.CommandArgTypeTarget, protocol.CommandArgTypeStandaloneTarget:
		return "target"
	case protocol.CommandArgTypePosition, protocol.CommandArgTypeBlockPosition:
		return "position"
	case protocol.CommandArgTypeJSON, protocol.CommandArgTypeJSONValue, protocol.CommandArgTypeJSONObjectFields:
		return "json"
	case protocol.CommandArgTypeRawText, protocol.CommandArgTypeMessage:
		return "text"
	case protocol.CommandArgTypeCommand, protocol.CommandArgTypeSlashCommand:
		return "command"
	default:
		return "string"
	}
}
func Commands(pk *packet.AvailableCommands, at int64, event string) []Command {
	resolve := func(index uint32) (string, []string) {
		if int(index) >= len(pk.Enums) {
			return "", []string{}
		}
		e := pk.Enums[index]
		values := []string{}
		for _, i := range e.ValueIndices {
			if int(i) < len(pk.EnumValues) {
				values = append(values, pk.EnumValues[i])
			}
		}
		return e.Type, values
	}
	out := []Command{}
	for _, c := range pk.Commands {
		_, aliases := resolve(c.AliasesOffset)
		v := Command{Name: c.Name, Description: c.Description, Permission: c.PermissionLevel, Flags: c.Flags, Aliases: aliases, Overloads: [][]Parameter{}, Chained: []string{}, Time: at, Event: event}
		for _, index := range c.ChainedSubcommandOffsets {
			if int(index) < len(pk.ChainedSubcommands) {
				v.Chained = append(v.Chained, pk.ChainedSubcommands[index].Name)
			}
		}
		for _, overload := range c.Overloads {
			params := []Parameter{}
			for _, p := range overload.Parameters {
				x := Parameter{Name: p.Name, Type: p.Type, Kind: parameterKind(p.Type), Optional: p.Optional, Options: p.Options, Values: []string{}}
				index := p.Type & 0xfffff
				if p.Type&protocol.CommandArgSoftEnum != 0 {
					x.Kind = "enum"
					x.Dynamic = true
					if int(index) < len(pk.DynamicEnums) {
						x.Enum = pk.DynamicEnums[index].Type
						x.Values = append(x.Values, pk.DynamicEnums[index].Values...)
					}
				} else if p.Type&protocol.CommandArgEnum != 0 {
					x.Kind = "enum"
					x.Enum, x.Values = resolve(index)
				} else if p.Type&protocol.CommandArgSuffixed != 0 && int(index) < len(pk.Suffixes) {
					x.Kind = "suffix"
					x.Values = []string{pk.Suffixes[index]}
				}
				params = append(params, x)
			}
			v.Overloads = append(v.Overloads, params)
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func normalize(pk packet.Packet, toServer bool) (string, any, string, uint32) {
	switch p := pk.(type) {
	case *packet.AvailableCommands:
		if !toServer {
			return "commands", commandSnapshot{Commands: Commands(p, 0, "")}, fmt.Sprintf("%d command definitions", len(p.Commands)), 0
		}
	case *packet.UpdateSoftEnum:
		if !toServer {
			return "enum", p, p.EnumType, 0
		}
	case *packet.CommandRequest:
		if toServer {
			return "command", p, p.CommandLine, 0
		}
	case *packet.CommandOutput:
		if !toServer {
			return "output", p, fmt.Sprintf("Command output: %d successful", p.SuccessCount), 0
		}
	case *packet.ResourcePackStack:
		if !toServer {
			return "packs", p, "Resource pack stack", 0
		}
	}
	return "", nil, "", 0
}
func apply(d *Document, e Event) {
	d.Events = append(d.Events, e)
	switch e.Kind {
	case "commands":
		var data commandSnapshot
		if json.Unmarshal(e.Data, &data) == nil {
			d.Commands = data.Commands
			for i := range d.Commands {
				d.Commands[i].Time = e.Time
				d.Commands[i].Event = e.ID
			}
		}
	case "enum":
		var p packet.UpdateSoftEnum
		_ = json.Unmarshal(e.Data, &p)
		for i := range d.Commands {
			for j := range d.Commands[i].Overloads {
				for k := range d.Commands[i].Overloads[j] {
					v := &d.Commands[i].Overloads[j][k]
					if !v.Dynamic || v.Enum != p.EnumType {
						continue
					}
					switch p.ActionType {
					case packet.SoftEnumActionSet:
						v.Values = append([]string{}, p.Options...)
					case packet.SoftEnumActionAdd:
						for _, s := range p.Options {
							found := false
							for _, old := range v.Values {
								if s == old {
									found = true
									break
								}
							}
							if !found {
								v.Values = append(v.Values, s)
							}
						}
					case packet.SoftEnumActionRemove:
						next := []string{}
						for _, s := range v.Values {
							remove := false
							for _, x := range p.Options {
								if x == s {
									remove = true
									break
								}
							}
							if !remove {
								next = append(next, s)
							}
						}
						v.Values = next
					}
				}
			}
		}
	}
	d.Session.Revision++
	d.Session.Updated = e.Time
	d.Session.Events = len(d.Events)
	d.Session.Commands = len(d.Commands)
}

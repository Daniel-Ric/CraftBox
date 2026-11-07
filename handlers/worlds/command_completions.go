package worlds

import (
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"slices"
	"strings"
)

func (w *worldsHandler) entitySuggestionsLocked() []string {
	values := make([]string, 0, len(w.commandEntityTypes))
	namespaces := map[string]bool{}
	for id := range w.commandEntityTypes {
		values = append(values, id)
		namespace, _, _ := strings.Cut(id, ":")
		namespaces[namespace] = true
	}
	for namespace := range namespaces {
		values = append(values, namespace+":*")
	}
	slices.Sort(values)
	return values
}
func (w *worldsHandler) exclusionSuggestionsLocked() []string {
	values := append([]string{"all"}, w.settings.ExcludedMobs...)
	slices.Sort(values)
	return slices.Compact(values)
}
func (w *worldsHandler) rememberCommandEntities(ids []string) {
	w.captureSettingsMu.Lock()
	if w.commandEntityTypes == nil {
		w.commandEntityTypes = map[string]struct{}{}
	}
	changed := false
	for _, id := range ids {
		if len(id) > 128 || !entityPattern.MatchString(id) || strings.ContainsAny(id, "*?[]") {
			continue
		}
		if _, exists := w.commandEntityTypes[id]; !exists && len(w.commandEntityTypes) < 8192 {
			w.commandEntityTypes[id] = struct{}{}
			changed = true
		}
	}
	w.captureSettingsMu.Unlock()
	if changed {
		w.pushCommandSuggestions(true, false, false)
	}
}
func (w *worldsHandler) updateExclusionSuggestions() {
	w.pushCommandSuggestions(false, true, false)
}
func (w *worldsHandler) publishCommandSuggestions(definitions *packet.AvailableCommands) {
	w.captureSettingsMu.Lock()
	prefix := w.commandCompletionPrefix
	w.captureSettingsMu.Unlock()
	if slices.ContainsFunc(definitions.DynamicEnums, func(value protocol.DynamicEnum) bool { return value.Type == prefix+"_entities" }) && slices.ContainsFunc(definitions.DynamicEnums, func(value protocol.DynamicEnum) bool { return value.Type == prefix+"_exclusions" }) {
		w.pushCommandSuggestions(true, true, true)
	}
}
func (w *worldsHandler) pushCommandSuggestions(entities, exclusions, activate bool) {
	w.commandCompletionMu.Lock()
	defer w.commandCompletionMu.Unlock()
	w.captureSettingsMu.Lock()
	if activate {
		w.commandMetadataReady = true
	}
	ready, prefix := w.commandMetadataReady, w.commandCompletionPrefix
	var ids, rules []string
	if ready && entities {
		ids = w.entitySuggestionsLocked()
	}
	if ready && exclusions {
		rules = w.exclusionSuggestionsLocked()
	}
	w.captureSettingsMu.Unlock()
	if ready && w.session != nil {
		if entities {
			_ = w.session.ClientWritePacket(&packet.UpdateSoftEnum{EnumType: prefix + "_entities", Options: ids, ActionType: packet.SoftEnumActionSet})
		}
		if exclusions {
			_ = w.session.ClientWritePacket(&packet.UpdateSoftEnum{EnumType: prefix + "_exclusions", Options: rules, ActionType: packet.SoftEnumActionSet})
		}
	}
}

func (w *worldsHandler) completeCommands(pk *packet.AvailableCommands, local []protocol.Command) []protocol.Command {
	w.commandCompletionMu.Lock()
	defer w.commandCompletionMu.Unlock()
	w.captureSettingsMu.Lock()
	prefix := w.commandCompletionPrefix
	entities := w.entitySuggestionsLocked()
	exclusions := w.exclusionSuggestionsLocked()
	w.commandMetadataReady = false
	w.captureSettingsMu.Unlock()
	// Hard enums validate fixed arguments in Minecraft. Soft enums suggest IDs
	// while retaining user-entered wildcard patterns, which the handler validates.
	indices := map[string]uint32{}
	for i, value := range pk.EnumValues {
		indices[value] = uint32(i)
	}
	hard := func(name string, values ...string) uint32 {
		index := uint32(len(pk.Enums))
		enum := protocol.CommandEnum{Type: prefix + "_" + name}
		for _, value := range values {
			n, ok := indices[value]
			if !ok {
				n = uint32(len(pk.EnumValues))
				pk.EnumValues = append(pk.EnumValues, value)
				indices[value] = n
			}
			enum.ValueIndices = append(enum.ValueIndices, n)
		}
		pk.Enums = append(pk.Enums, enum)
		return protocol.CommandArgValid | protocol.CommandArgEnum | index
	}
	soft := func(name string, values []string) uint32 {
		index := uint32(len(pk.DynamicEnums))
		pk.DynamicEnums = append(pk.DynamicEnums, protocol.DynamicEnum{Type: prefix + "_" + name, Values: values})
		return protocol.CommandArgValid | protocol.CommandArgSoftEnum | index
	}
	states := hard("states", "on", "off", "toggle")
	zoom := hard("zoom", "1", "2", "4", "8", "16")
	pages := hard("help_pages", "1", "2", "3")
	iconNames := make([]string, 0, len(captureMarkerIcons))
	for name := range captureMarkerIcons {
		iconNames = append(iconNames, name)
	}
	slices.Sort(iconNames)
	icons := hard("marker_icons", iconNames...)
	entityType := soft("entities", entities)
	exclusionType := soft("exclusions", exclusions)
	param := func(name string, kind uint32, optional bool) protocol.CommandParameter {
		return protocol.CommandParameter{Name: name, Type: kind, Optional: optional}
	}
	raw := uint32(protocol.CommandArgValid | protocol.CommandArgTypeRawText)
	integer := uint32(protocol.CommandArgValid | protocol.CommandArgTypeInt)
	patterns := func(kind uint32) []protocol.CommandParameter {
		params := make([]protocol.CommandParameter, 32)
		for i := range params {
			params[i] = param(fmt.Sprintf("entity_%d", i+1), kind, i > 0)
		}
		return params
	}
	commands := []string{"menu", "position", "help", "name", "void", "save", "exclude", "include", "exclusions", "map", "zoom", "connection", "status", "version", "mark", "note", "marker", "region", "camera", "capture", "structure"}
	sub := map[string]uint32{}
	for _, name := range commands {
		sub[name] = hard("sub_"+name, name)
	}
	root := []protocol.CommandOverload{{}}
	addRoot := func(name string, params ...protocol.CommandParameter) {
		params = append([]protocol.CommandParameter{param("command", sub[name], false)}, params...)
		root = append(root, protocol.CommandOverload{Parameters: params})
	}
	addRoot("menu")
	addRoot("position", param("state", states, true))
	addRoot("help", param("page", pages, true))
	addRoot("name", param("world_name", raw, false))
	addRoot("void", param("state", states, true))
	addRoot("save")
	addRoot("exclude", patterns(entityType)...)
	addRoot("include", patterns(exclusionType)...)
	addRoot("exclusions", param("page", integer, true))
	addRoot("map", param("state", states, true))
	addRoot("zoom", param("level", zoom, true))
	addRoot("connection")
	addRoot("status")
	addRoot("version")
	addRoot("mark", param("name", raw, false))
	addRoot("note", param("text", raw, false))
	addRoot("marker", param("icon", icons, false), param("name", raw, false))
	addRoot("region", param("action", hard("region_corners", "pos1", "pos2", "clear"), false))
	addRoot("region", param("action", hard("region_create", "create"), false), param("name", raw, false))
	addRoot("camera", param("action", hard("camera_add", "add"), false), param("name", raw, false))
	addRoot("capture", param("action", hard("capture_status", "status"), false))
	structureSimple := hard("structure_actions", "status", "clear", "cancel", "open", "list")
	structureCorners := hard("structure_corners", "pos1", "pos2")
	structureSave := hard("structure_save", "save")
	structureOverloads := []protocol.CommandOverload{{}, {Parameters: []protocol.CommandParameter{param("action", structureSimple, false)}}, {Parameters: []protocol.CommandParameter{param("action", structureCorners, false), param("x", integer, true), param("y", integer, true), param("z", integer, true)}}, {Parameters: []protocol.CommandParameter{param("action", structureSave, false), param("name", raw, false)}}}
	for _, overload := range structureOverloads {
		addRoot("structure", overload.Parameters...)
	}
	for i := range local {
		var params []protocol.CommandParameter
		switch local[i].Name {
		case "cb":
			local[i].Overloads = root
			continue
		case "blueprint":
			local[i].Overloads = structureOverloads
			continue
		case "cbhelp":
			params = []protocol.CommandParameter{param("page", pages, true)}
		case "setname":
			params = []protocol.CommandParameter{param("world_name", raw, false)}
		case "void":
			params = []protocol.CommandParameter{param("state", states, true)}
		case "exclude-mob":
			params = patterns(entityType)
		case "include-mob":
			params = patterns(exclusionType)
		case "save-world":
			params = nil
		default:
			continue
		}
		local[i].Overloads = []protocol.CommandOverload{{Parameters: params}}
	}
	return local
}

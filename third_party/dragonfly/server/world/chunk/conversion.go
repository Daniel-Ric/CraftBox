package chunk

import (
	"bytes"
	_ "embed"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type legacyBlockEntry struct {
	Name string `nbt:"name"`
	Meta int16  `nbt:"meta"`
}

var (
	//go:embed legacy_states.nbt
	legacyMappingsData []byte

	legacyMappings = make(map[legacyBlockEntry]blockEntry)
)

func upgradeLegacyEntry(name string, meta int16) (blockEntry, bool) {
	entry, ok := legacyMappings[legacyBlockEntry{Name: name, Meta: meta}]
	if !ok {

		entry, ok = legacyMappings[legacyBlockEntry{Name: name}]
	}
	return entry, ok
}

func init() {
	var entry struct {
		Legacy  legacyBlockEntry `nbt:"legacy"`
		Updated blockEntry       `nbt:"updated"`
	}
	dec := nbt.NewDecoder(bytes.NewBuffer(legacyMappingsData))
	for {
		if err := dec.Decode(&entry); err != nil {
			break
		}
		legacyMappings[entry.Legacy] = entry.Updated
	}
}

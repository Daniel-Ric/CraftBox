package packbuilder

import (
	_ "embed"
	"os"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"golang.org/x/mod/sumdb/dirhash"
)

//go:embed pack_icon.png
var packIcon []byte

func BuildResourcePack(reg world.BlockRegistry) (*resource.Pack, bool) {
	dir, err := os.MkdirTemp("", "dragonfly_resource_pack-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	var assets int
	var lang []string

	itemCount, itemLang := buildItems(dir)
	assets += itemCount
	lang = append(lang, itemLang...)

	blockCount, blockLang := buildBlocks(reg, dir)
	assets += blockCount
	lang = append(lang, blockLang...)

	if assets > 0 {
		buildLanguageFile(dir, lang)
		if err := os.WriteFile(dir+"/pack_icon.png", packIcon, 0666); err != nil {
			panic(err)
		}
		hash, err := dirhash.HashDir(dir, "", dirhash.Hash1)
		if err != nil {
			panic(err)
		}
		var header, module [16]byte
		copy(header[:], hash)
		copy(module[:], hash[16:])
		buildManifest(dir, header, module)
		return resource.MustReadPath(dir), true
	}
	return nil, false
}

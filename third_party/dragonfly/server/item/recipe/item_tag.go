package recipe

import (
	_ "embed"
	"encoding/json"
	"slices"
)

var (
	//go:embed item_tags.json
	itemTagData []byte
	itemTags    = make(map[string][]string)
)

func init() {
	if err := json.Unmarshal(itemTagData, &itemTags); err != nil {
		panic(err)
	}
}

type ItemTag struct {
	tag   string
	count int

	items []string
}

func NewItemTag(tag string, count int) ItemTag {
	if count < 0 {
		count = 0
	}
	return ItemTag{tag: tag, count: count, items: itemTags[tag]}
}

func (i ItemTag) Count() int {
	return i.count
}

func (i ItemTag) Empty() bool {
	return i.count == 0 || i.tag == ""
}

func (i ItemTag) Tag() string {
	return i.tag
}

func (i ItemTag) Contains(name string) bool {
	return slices.Contains(i.items, name)
}

package iteminternal

import (
	"github.com/df-mc/dragonfly/server/item/category"
	"maps"
)

type ComponentBuilder struct {
	name       string
	identifier string
	category   category.Category

	properties map[string]any
	components map[string]any
}

func NewComponentBuilder(name, identifier string, category category.Category) *ComponentBuilder {
	return &ComponentBuilder{
		name:       name,
		identifier: identifier,
		category:   category,

		properties: make(map[string]any),
		components: make(map[string]any),
	}
}

func (builder *ComponentBuilder) AddProperty(name string, value any) {
	builder.properties[name] = value
}

func (builder *ComponentBuilder) AddComponent(name string, value any) {
	builder.components[name] = value
}

func (builder *ComponentBuilder) Construct() map[string]any {
	properties := maps.Clone(builder.properties)
	components := maps.Clone(builder.components)
	builder.applyDefaultProperties(properties)
	builder.applyDefaultComponents(components, properties)
	return map[string]any{"components": components}
}

func (builder *ComponentBuilder) applyDefaultProperties(x map[string]any) {
	x["minecraft:icon"] = map[string]any{
		"textures": map[string]any{
			"default": builder.identifier,
		},
	}
	x["creative_group"] = builder.category.Group()
	x["creative_category"] = int32(builder.category.Uint8())
	if _, ok := x["max_stack_size"]; !ok {
		x["max_stack_size"] = int32(64)
	}
}

func (builder *ComponentBuilder) applyDefaultComponents(x, properties map[string]any) {
	x["item_properties"] = properties
	x["minecraft:display_name"] = map[string]any{
		"value": builder.name,
	}
}

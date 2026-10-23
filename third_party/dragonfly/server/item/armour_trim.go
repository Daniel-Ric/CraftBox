package item

import "github.com/df-mc/dragonfly/server/world"

type ArmourTrim struct {
	Template SmithingTemplateType
	Material ArmourTrimMaterial
}

func (trim ArmourTrim) Zero() bool {
	return trim.Material == nil || trim.Template == TemplateNetheriteUpgrade()
}

type ArmourTrimMaterial interface {
	TrimMaterial() string

	MaterialColour() string
}

func trimMaterialFromString(name string) (ArmourTrimMaterial, bool) {
	switch name {
	case "amethyst":
		return AmethystShard{}, true
	case "copper":
		return CopperIngot{}, true
	case "diamond":
		return Diamond{}, true
	case "emerald":
		return Emerald{}, true
	case "gold":
		return GoldIngot{}, true
	case "iron":
		return IronIngot{}, true
	case "lapis":
		return LapisLazuli{}, true
	case "netherite":
		return NetheriteIngot{}, true
	case "quartz":
		return NetherQuartz{}, true
	case "resin":
		return ResinBrick{}, true
	case "redstone":
		return RedstoneWire{}, true
	}
	return nil, false
}

func ArmourTrimMaterials() []world.Item {
	return []world.Item{
		AmethystShard{},
		CopperIngot{},
		Diamond{},
		Emerald{},
		GoldIngot{},
		IronIngot{},
		LapisLazuli{},
		NetheriteIngot{},
		NetherQuartz{},
		ResinBrick{},
		RedstoneWire{},
	}
}

type Trimmable interface {
	WithTrim(trim ArmourTrim) world.Item
}

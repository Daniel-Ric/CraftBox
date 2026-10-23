package item

type SmithingTemplate struct {
	Template SmithingTemplateType
}

func (t SmithingTemplate) EncodeItem() (name string, meta int16) {
	if t.Template == TemplateNetheriteUpgrade() {
		return "minecraft:netherite_upgrade_smithing_template", 0
	}
	return "minecraft:" + t.Template.String() + "_armor_trim_smithing_template", 0
}

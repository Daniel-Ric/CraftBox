package item

type ShulkerShell struct{}

func (ShulkerShell) EncodeItem() (name string, meta int16) {
	return "minecraft:shulker_shell", 0
}

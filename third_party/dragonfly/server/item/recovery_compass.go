package item

type RecoveryCompass struct{}

func (RecoveryCompass) EncodeItem() (name string, meta int16) {
	return "minecraft:recovery_compass", 0
}

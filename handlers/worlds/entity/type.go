package entity

import "strings"

func IsCustomType(name string) bool {
	namespace, _, qualified := strings.Cut(name, ":")
	return qualified && namespace != "minecraft" && namespace != "CraftBox_player"
}

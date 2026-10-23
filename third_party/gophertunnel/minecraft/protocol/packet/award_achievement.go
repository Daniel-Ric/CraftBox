package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type AwardAchievement struct {
	AchievementID int32
}

func (*AwardAchievement) ID() uint32 {
	return IDAwardAchievement
}

func (pk *AwardAchievement) Marshal(io protocol.IO) {
	io.Int32(&pk.AchievementID)
}

package world

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/google/uuid"
)

type Sleeper interface {
	Entity

	Name() string
	UUID() uuid.UUID

	Messaget(t chat.Translation, a ...any)
	SendSleepingIndicator(sleeping, max int)

	Sleep(pos cube.Pos)
	Sleeping() (cube.Pos, bool)
	Wake()
}

const (
	TimeSleep         = 12542
	TimeWake          = 23459
	TimeSleepWithRain = 12010
	TimeWakeWithRain  = 23991
	TimeFull          = 24000
)

func (ticker) tryAdvanceDay(tx *Tx, timeCycle bool) {
	sleepers := tx.Sleepers()
	time := tx.w.Time() % TimeFull

	for s := range sleepers {
		if !tx.Thundering() {
			if !tx.Raining() && (time <= TimeSleep || time >= TimeWake) {
				return
			}
			if time <= TimeSleepWithRain || time >= TimeWakeWithRain {
				return
			}
		}

		if _, ok := s.Sleeping(); !ok {

			return
		}
	}

	for s := range sleepers {
		s.Wake()
	}

	totalTime := tx.w.Time()
	if timeCycle {
		tx.w.SetTime(totalTime + TimeFull - time)
	}
	tx.w.StopRaining()
}

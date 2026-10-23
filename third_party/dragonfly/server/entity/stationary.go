package entity

import (
	"github.com/df-mc/dragonfly/server/world"
	"math"
	"time"
)

type StationaryBehaviourConfig struct {
	ExistenceDuration time.Duration

	SpawnSounds []world.Sound

	Tick func(e *Ent, tx *world.Tx)
}

func (conf StationaryBehaviourConfig) Apply(data *world.EntityData) {
	data.Data = conf.New()
}

func (conf StationaryBehaviourConfig) New() *StationaryBehaviour {
	if conf.ExistenceDuration == 0 {
		conf.ExistenceDuration = math.MaxInt64
	}
	return &StationaryBehaviour{BaseBehaviour: NewBaseBehaviour(), conf: conf}
}

type StationaryBehaviour struct {
	BaseBehaviour

	conf  StationaryBehaviourConfig
	close bool
}

func (s *StationaryBehaviour) Tick(e *Ent, tx *world.Tx) *Movement {
	if s.close {
		_ = e.Close()
		return nil
	}

	if e.Age() == 0 {
		for _, ss := range s.conf.SpawnSounds {
			tx.PlaySound(e.Position(), ss)
		}
	}
	if s.conf.Tick != nil {
		s.conf.Tick(e, tx)
	}

	if e.Age() > s.conf.ExistenceDuration {
		s.close = true
	}

	return nil
}

func (s *StationaryBehaviour) Immobile() bool {
	return true
}

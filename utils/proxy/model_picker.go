package proxy

import (
	"errors"
	"github.com/Daniel-Ric/CraftBox/internal/modelpicker"
)

func (s *Session) PickBlock() (modelpicker.BlockTarget, error) {
	for _, handler := range s.handlers {
		if handler.PickBlock != nil {
			return handler.PickBlock(s)
		}
	}
	return modelpicker.BlockTarget{}, errors.New("Enable terrain capture to pick blocks")
}

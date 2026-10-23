package block

import "time"

type OpenAction struct{ action }

type CloseAction struct{ action }

type StartCrackAction struct {
	action
	BreakTime time.Duration
}

type ContinueCrackAction struct {
	action
	BreakTime time.Duration
}

type StopCrackAction struct{ action }

type DecoratedPotWobbleAction struct {
	action
	DecoratedPot DecoratedPot

	Success bool
}

type action struct{}

func (action) BlockAction() {}

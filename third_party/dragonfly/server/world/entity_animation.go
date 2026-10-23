package world

type EntityAnimation struct {
	name          string
	nextState     string
	controller    string
	stopCondition string
}

func NewEntityAnimation(name string) EntityAnimation {
	return EntityAnimation{name: name}
}

func (a EntityAnimation) Name() string {
	return a.name
}

func (a EntityAnimation) Controller() string {
	return a.controller
}

func (a EntityAnimation) WithController(controller string) EntityAnimation {
	a.controller = controller
	return a
}

func (a EntityAnimation) NextState() string {
	return a.nextState
}

func (a EntityAnimation) WithNextState(state string) EntityAnimation {
	a.nextState = state
	return a
}

func (a EntityAnimation) StopCondition() string {
	return a.stopCondition
}

func (a EntityAnimation) WithStopCondition(condition string) EntityAnimation {
	a.stopCondition = condition
	return a
}

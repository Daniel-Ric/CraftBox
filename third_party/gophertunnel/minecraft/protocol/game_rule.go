package protocol

type GameRule struct {
	Name string

	CanBeModifiedByPlayer bool

	Value any
}

package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	CodeBuilderOperationNone = iota
	CodeBuilderOperationGet
	CodeBuilderOperationSet
	CodeBuilderOperationReset
)

const (
	CodeBuilderCategoryNone = iota
	CodeBuilderCategoryStatus
	CodeBuilderCategoryInstantiation
)

const (
	CodeBuilderStatusNone = iota
	CodeBuilderStatusNotStarted
	CodeBuilderStatusInProgress
	CodeBuilderStatusPaused
	CodeBuilderStatusError
	CodeBuilderStatusSucceeded
)

type CodeBuilderSource struct {
	Operation byte

	Category byte

	CodeStatus byte
}

func (pk *CodeBuilderSource) ID() uint32 {
	return IDCodeBuilderSource
}

func (pk *CodeBuilderSource) Marshal(io protocol.IO) {
	io.Uint8(&pk.Operation)
	io.Uint8(&pk.Category)
	io.Uint8(&pk.CodeStatus)
}

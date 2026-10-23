package packet

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ClientInputLockCamera = 1 << (iota + 1)
	ClientInputLockMovement
	_
	ClientInputLockLateralMovement
	ClientInputLockSneak
	ClientInputLockJump
	ClientInputLockMount
	ClientInputLockDismount
	ClientInputLockMoveForward
	ClientInputLockMoveBackward
	ClientInputLockMoveLeft
	ClientInputLockMoveRight
)

type UpdateClientInputLocks struct {
	Locks uint32
}

func (pk *UpdateClientInputLocks) ID() uint32 {
	return IDUpdateClientInputLocks
}

func (pk *UpdateClientInputLocks) Marshal(io protocol.IO) {
	io.Varuint32(&pk.Locks)
}

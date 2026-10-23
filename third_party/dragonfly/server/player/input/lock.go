package input

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

type Lock struct {
	lock
}

type lock uint32

func Camera() Lock {
	return Lock{lock(packet.ClientInputLockCamera)}
}

func Movement() Lock {
	return Lock{lock(packet.ClientInputLockMovement)}
}

func LateralMovement() Lock {
	return Lock{lock(packet.ClientInputLockLateralMovement)}
}

func Sneak() Lock {
	return Lock{lock(packet.ClientInputLockSneak)}
}

func Jump() Lock {
	return Lock{lock(packet.ClientInputLockJump)}
}

func Mount() Lock {
	return Lock{lock(packet.ClientInputLockMount)}
}

func Dismount() Lock {
	return Lock{lock(packet.ClientInputLockDismount)}
}

func MoveForward() Lock {
	return Lock{lock(packet.ClientInputLockMoveForward)}
}

func MoveBackward() Lock {
	return Lock{lock(packet.ClientInputLockMoveBackward)}
}

func MoveLeft() Lock {
	return Lock{lock(packet.ClientInputLockMoveLeft)}
}

func MoveRight() Lock {
	return Lock{lock(packet.ClientInputLockMoveRight)}
}

func (l lock) Uint32() uint32 {
	return uint32(l)
}

func All() []Lock {
	return []Lock{
		Camera(), Movement(), LateralMovement(), Sneak(), Jump(), Mount(), Dismount(),
		MoveForward(), MoveBackward(), MoveLeft(), MoveRight(),
	}
}

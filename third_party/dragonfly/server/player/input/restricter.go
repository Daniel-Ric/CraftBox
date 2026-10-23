package input

type Restricter interface {
	LockInput(l Lock)

	UnlockInput(l Lock)

	ClearInputLocks()

	InputLocked(l Lock) bool
}

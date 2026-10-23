package dialogue

import "github.com/df-mc/dragonfly/server/world"

type Submittable interface {
	Submit(submitter Submitter, pressed Button, tx *world.Tx)
}

type Submitter interface {
	SendDialogue(d Dialogue, e world.Entity)
	CloseDialogue()
}

type Closer interface {
	Close(submitter Submitter, tx *world.Tx)
}

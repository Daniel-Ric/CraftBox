package form

import "github.com/df-mc/dragonfly/server/world"

type Submittable interface {
	Submit(submitter Submitter, tx *world.Tx)
}

type MenuSubmittable interface {
	Submit(submitter Submitter, pressed Button, tx *world.Tx)
}

type ModalSubmittable MenuSubmittable

type Closer interface {
	Close(submitter Submitter, tx *world.Tx)
}

type Submitter interface {
	SendForm(form Form)
	CloseForm()
}

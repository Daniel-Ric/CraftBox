package player

import (
	"errors"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"io"
)

type Provider interface {
	Save(uuid uuid.UUID, data Config, w *world.World) error

	Load(uuid uuid.UUID, world func(world.Dimension) *world.World) (Config, *world.World, error)

	io.Closer
}

var _ Provider = (*NopProvider)(nil)

type NopProvider struct{}

func (NopProvider) Save(uuid.UUID, Config, *world.World) error { return nil }
func (NopProvider) Load(uuid.UUID, func(world.Dimension) *world.World) (Config, *world.World, error) {
	return Config{}, nil, errors.New("")
}
func (NopProvider) Close() error { return nil }

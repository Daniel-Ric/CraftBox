package minecraft

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type ResourcePackCacheKey struct {
	UUID    uuid.UUID
	Version string
	Size    uint64
}

func (key ResourcePackCacheKey) Matches(pack *resource.Pack) bool {
	return pack.UUID() == key.UUID && pack.Version() == key.Version && uint64(pack.Len()) == key.Size
}

type ResourcePackCache interface {
	Load(ctx context.Context, key ResourcePackCacheKey) (*resource.Pack, error)

	Store(ctx context.Context, key ResourcePackCacheKey, pack *resource.Pack) error
}

type DirResourcePackCache struct {
	Dir string
}

func (cache DirResourcePackCache) Load(_ context.Context, key ResourcePackCacheKey) (*resource.Pack, error) {
	pack, err := resource.ReadPath(cache.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return pack, err
}

func (cache DirResourcePackCache) Store(_ context.Context, key ResourcePackCacheKey, pack *resource.Pack) error {
	if err := os.MkdirAll(cache.Dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(cache.Dir, "pack-*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if _, err := io.Copy(temp, io.NewSectionReader(pack, 0, int64(pack.Len()))); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), cache.path(key))
}

func (cache DirResourcePackCache) path(key ResourcePackCacheKey) string {
	return filepath.Join(cache.Dir, key.UUID.String()+"_"+url.PathEscape(key.Version)+".mcpack")
}

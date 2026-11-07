package assetlibrary

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

type previewFS struct{ fs.FS }

func (p previewFS) Open(name string) (fs.File, error) {
	if name != "." {
		if _, e := safeName(name); e != nil {
			return nil, e
		}
	}
	f, e := p.FS.Open(name)
	if e != nil {
		return nil, e
	}
	i, e := f.Stat()
	if e != nil {
		f.Close()
		return nil, e
	}
	if i.IsDir() {
		return f, nil
	}
	limit := int64(MaxPreviewBytes)
	ext := strings.ToLower(path.Ext(name))
	if ext == ".json" {
		limit = 4 << 20
	}
	if !i.Mode().IsRegular() || i.Size() > limit {
		f.Close()
		return nil, fmt.Errorf("%s exceeds the preview file limit", name)
	}
	{
		var w, h int
		if ext == ".tga" {
			var header [18]byte
			_, e = io.ReadFull(f, header[:])
			w = int(binary.LittleEndian.Uint16(header[12:14]))
			h = int(binary.LittleEndian.Uint16(header[14:16]))
		} else {
			var c image.Config
			c, _, e = image.DecodeConfig(f)
			w, h = c.Width, c.Height
		}
		f.Close()
		if e != nil {
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".tga" {
				return nil, e
			}
			return p.FS.Open(name)
		}
		if w < 1 || h < 1 || int64(w)*int64(h) > 16<<20 {
			return nil, errors.New("Texture exceeds the 16 megapixel preview limit")
		}
		return p.FS.Open(name)
	}
}
func (l *Library) PreviewSource(id string) (fs.FS, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	item, e := l.get(id)
	if e != nil {
		return nil, e
	}
	if item.Kind != "packs" {
		return nil, errors.New("Choose a resource pack")
	}
	if item.Encrypted {
		return nil, errors.New("Restore this pack from saved captures before applying its models and textures")
	}
	root, e := l.RootFor(id)
	if e != nil {
		return nil, e
	}
	return previewFS{os.DirFS(root)}, nil
}

package resource

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"path"
	"strings"
)

type packArchive struct {
	reader *zip.Reader
	base   string
	files  map[string]*zip.File
}

func indexPackArchive(content *bytes.Reader) (*packArchive, error) {
	z, err := zip.NewReader(content, content.Size())
	if err != nil {
		return nil, err
	}
	a := &packArchive{reader: z, files: make(map[string]*zip.File, len(z.File))}
	for _, f := range z.File {
		if strings.EqualFold(path.Base(f.Name), "manifest.json") {
			a.base = path.Dir(f.Name)
			if a.base == "." {
				a.base = ""
			}
			break
		}
	}
	base := strings.ToLower(a.base) + "/"

	for _, f := range z.File {
		name := strings.ToLower(f.Name)
		if _, exists := a.files[name]; !exists {
			a.files[name] = f
		}
		if a.base != "" && strings.HasPrefix(name, base) {
			rel := strings.TrimPrefix(name, base)
			if _, exists := a.files[rel]; !exists {
				a.files[rel] = f
			}
		}
	}
	return a, nil
}

func FromReaderAt(reader io.ReaderAt, size int64) (*Pack, error) {
	return Read(io.NewSectionReader(reader, 0, size))
}

func (p *Pack) Open(name string) (fs.File, error) {
	z := p.archive.reader
	base := p.BaseDir()
	if base != "" {
		zfs, err := fs.Sub(z, base)
		if err != nil {
			return nil, err
		}
		return zfs.Open(name)
	}
	return z.Open(name)
}

func (p *Pack) BaseDir() string {
	return p.archive.base
}

func (p *Pack) CanRead() bool { return !p.Encrypted() }

func (p *Pack) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, io.NewSectionReader(p.content, 0, p.content.Size()))
}

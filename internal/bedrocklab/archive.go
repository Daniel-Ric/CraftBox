package bedrocklab

import (
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

type ArchiveEntry struct {
	Path   string
	Offset int64
	Size   int64
}

func safePath(s string) bool {
	if s == "" || !utf8.ValidString(s) || strings.ContainsAny(s, "\\:\x00") || strings.HasPrefix(s, "/") || path.Clean(s) != s {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == "." || part == ".." || strings.TrimRight(part, " .") != part || strings.ContainsAny(part, "<>\"|?*\r\n") {
			return false
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9') {
			return false
		}
	}
	return true
}

func ReadArchive(r io.ReaderAt, size int64) ([]ArchiveEntry, error) {
	h := make([]byte, 16)
	if _, err := r.ReadAt(h, 0); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint64(h) != 0x267052a0b125277d {
		return nil, fmt.Errorf("invalid BRArchive magic")
	}
	if v := binary.LittleEndian.Uint32(h[12:]); v != 1 {
		return nil, fmt.Errorf("unsupported BRArchive version %d", v)
	}
	count := int64(binary.LittleEndian.Uint32(h[8:]))
	base := 16 + count*256
	if count > 200000 || base > size {
		return nil, fmt.Errorf("invalid archive directory size")
	}
	entries := make([]ArchiveEntry, 0, count)
	seen := map[string]bool{}
	b := make([]byte, 256)
	for i := int64(0); i < count; i++ {
		if _, err := r.ReadAt(b, 16+i*256); err != nil {
			return nil, err
		}
		n := int(b[0])
		if n == 0 || n > 247 {
			return nil, fmt.Errorf("invalid entry name length")
		}
		name := string(b[1 : 1+n])
		key := strings.ToLower(name)
		if !safePath(name) || seen[key] {
			return nil, fmt.Errorf("unsafe or duplicate archive path %q", name)
		}
		seen[key] = true
		offset, length := int64(binary.LittleEndian.Uint32(b[248:])), int64(binary.LittleEndian.Uint32(b[252:]))
		if offset > size-base || length > size-base-offset {
			return nil, fmt.Errorf("archive entry %q exceeds content bounds", name)
		}
		entries = append(entries, ArchiveEntry{name, base + offset, length})
	}
	return entries, nil
}

func archiveDestination(name string) string {
	dir, file := path.Split(name)
	parts := strings.Split(strings.TrimSuffix(dir, "/"), "/")
	for i, p := range parts {
		if strings.EqualFold(p, "__brarchive") {
			parts = append(parts[:i], parts[i+1:]...)
			break
		}
	}
	return path.Join(strings.Join(parts, "/"), strings.TrimSuffix(file, path.Ext(file)))
}

package assetlibrary

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type previewCache struct {
	mu     sync.Mutex
	bytes  int
	values map[string]ObjectPreview
}

func (c *previewCache) get(key string) (ObjectPreview, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.values[key]
	return value, ok
}
func (c *previewCache) put(key string, value ObjectPreview) {
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 8<<20 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bytes+len(raw) > 32<<20 || len(c.values) >= 64 {
		c.bytes = 0
		c.values = map[string]ObjectPreview{}
	}
	c.bytes += len(raw)
	c.values[key] = value
}

type catalogEntry struct {
	definitionStamp string
	index           objectIndex
	stamp           time.Time
	checked         time.Time
	fingerprint     string
}

func (l *Library) catalog(id string) (objectIndex, error) {
	root, err := l.RootFor(id)
	if err != nil {
		return objectIndex{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return objectIndex{}, err
	}
	l.catalogMu.Lock()
	defer l.catalogMu.Unlock()
	definitionStamp := l.blockDefinitionStamp(id)
	if entry, ok := l.catalogs[id]; ok && entry.definitionStamp == definitionStamp && entry.stamp.Equal(info.ModTime()) && time.Since(entry.checked) < 30*time.Second {
		return entry.index, nil
	}
	if encryptedPack(root) {
		return objectIndex{}, errors.New("Capture this pack again to restore readable models")
	}
	files := []File{}
	fingerprint := sha256.New()
	fmt.Fprint(fingerprint, definitionStamp)
	var total int64
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("Asset contains a symbolic link")
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if len(files) >= MaxFiles || total > MaxImportBytes {
			return errors.New("Asset exceeds preview limits")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if _, err = safeName(name); err != nil {
			return err
		}
		files = append(files, File{Path: name, Bytes: info.Size()})
		fmt.Fprintf(fingerprint, "%s\x00%d\x00%d\x00", name, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return objectIndex{}, err
	}
	signature := fmt.Sprintf("%x", fingerprint.Sum(nil))
	if entry, ok := l.catalogs[id]; ok && entry.fingerprint == signature {
		entry.checked = time.Now()
		entry.stamp = info.ModTime()
		l.catalogs[id] = entry
		return entry.index, nil
	}
	x := l.objectIndex(Detail{Item: Item{ID: id}, Files: files})
	if x.readError != nil {
		return objectIndex{}, x.readError
	}
	l.injectBlockDefinitions(id, &x)
	x.resolved = x.objects()
	x.epoch = time.Now().Format(time.RFC3339Nano)
	if l.catalogs == nil {
		l.catalogs = map[string]catalogEntry{}
	}
	delete(l.catalogs, id)
	for len(l.catalogs) > 0 {
		total := x.sourceBytes
		oldest := ""
		var checked time.Time
		for key, entry := range l.catalogs {
			total += entry.index.sourceBytes
			if oldest == "" || entry.checked.Before(checked) {
				oldest = key
				checked = entry.checked
			}
		}
		if len(l.catalogs) < 64 && total <= 32<<20 {
			break
		}
		delete(l.catalogs, oldest)
	}
	if x.sourceBytes <= 32<<20 {
		l.catalogs[id] = catalogEntry{index: x, stamp: info.ModTime(), checked: time.Now(), fingerprint: signature, definitionStamp: definitionStamp}
	}
	return x, nil
}

// Package localworlds creates complete Bedrock world copies without changing
// the installed game, the original world, or pack protection metadata.
package localworlds

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type Root struct{ Storage, Name, Folder string }
type World struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Folder         string   `json:"folder"`
	Storage        string   `json:"-"`
	Installation   string   `json:"installation"`
	Modified       int64    `json:"modified"`
	Icon           string   `json:"icon,omitempty"`
	Template       bool     `json:"template"`
	Marketplace    bool     `json:"marketplace"`
	ExtraPackRoots []string `json:"-"`
}
type Options struct {
	IncludeGlobal bool `json:"includeGlobal"`
	BackupOnly    bool `json:"backupOnly"`
}
type Issue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	PackID   string `json:"packId,omitempty"`
}
type Pack struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	RequiredBy string `json:"requiredBy"`
	Embedded   bool   `json:"embedded"`
	candidate  *candidate
}
type Plan struct {
	World       World   `json:"world"`
	Packs       []Pack  `json:"packs"`
	Issues      []Issue `json:"issues"`
	Portable    bool    `json:"portable"`
	Files       int     `json:"files"`
	Bytes       int64   `json:"bytes"`
	Fingerprint string  `json:"fingerprint"`
	entries     []entry
}
type Progress struct {
	Stage     string `json:"stage"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
	Files     int    `json:"files"`
	Running   bool   `json:"running"`
	JobID     string `json:"jobId"`
}
type Result struct {
	Path     string  `json:"path"`
	Bytes    int64   `json:"bytes"`
	Files    int     `json:"files"`
	Packs    int     `json:"packs"`
	Portable bool    `json:"portable"`
	Issues   []Issue `json:"issues"`
}

func readSmall(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("metadata file is too large")
	}
	return raw, nil
}
func metadata(folder string) (map[string]any, error) {
	raw, err := readSmall(filepath.Join(folder, "level.dat"), 16<<20)
	if err != nil {
		return nil, err
	}
	if len(raw) < 8 {
		return nil, fmt.Errorf("invalid Bedrock level.dat")
	}
	size := int(binary.LittleEndian.Uint32(raw[4:8]))
	if size <= 0 || size > len(raw)-8 {
		return nil, fmt.Errorf("invalid Bedrock metadata length")
	}
	var data map[string]any
	err = nbt.UnmarshalEncoding(raw[8:8+size], &data, nbt.LittleEndian)
	return data, err
}
func canonical(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("could not resolve %s: %w", absolute, err)
	}
	return resolved, nil
}
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func flag(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case uint8:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case string:
		return v != "" && v != "00000000-0000-0000-0000-000000000000"
	}
	return false
}

func Describe(folder, storage, name string) (World, error) {
	root, err := canonical(folder)
	if err != nil {
		return World{}, err
	}
	st, err := os.Stat(filepath.Join(root, "level.dat"))
	if err != nil {
		return World{}, fmt.Errorf("choose a Bedrock world folder containing level.dat and db")
	}
	db, err := os.Lstat(filepath.Join(root, "db"))
	if err != nil || !db.IsDir() || db.Mode()&os.ModeSymlink != 0 {
		return World{}, fmt.Errorf("the world database folder is missing or is a link")
	}
	if storage == "" {
		storage = filepath.Dir(root)
		if strings.EqualFold(filepath.Base(storage), "minecraftWorlds") {
			storage = filepath.Dir(storage)
		}
	}
	storage, err = canonical(storage)
	if err != nil {
		return World{}, err
	}
	sum := sha256.Sum256([]byte(strings.ToLower(root)))
	w := World{ID: hex.EncodeToString(sum[:16]), Name: filepath.Base(root), Folder: root, Storage: storage, Installation: name, Modified: st.ModTime().UnixMilli()}
	if raw, e := readSmall(filepath.Join(root, "levelname.txt"), 64<<10); e == nil {
		if title := strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff")); title != "" {
			w.Name = title
		}
	}
	if data, e := metadata(root); e == nil {
		if w.Name == filepath.Base(root) {
			if title, ok := data["LevelName"].(string); ok && title != "" {
				w.Name = title
			}
		}
		for key, value := range data {
			switch strings.ToLower(key) {
			case "isfromworldtemplate", "isfromlockedtemplate":
				w.Template = w.Template || flag(value)
			case "marketplaceworldtemplateid", "premiumworldtemplateid":
				w.Marketplace = w.Marketplace || flag(value)
			}
		}
	}
	if raw, e := readSmall(filepath.Join(root, "world_icon.jpeg"), 2<<20); e == nil && len(raw) > 3 && raw[0] == 255 && raw[1] == 216 {
		w.Icon = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(raw)
	}
	return w, nil
}
func Discover(ctx context.Context, roots []Root) ([]World, error) {
	worlds := []World{}
	seen := map[string]bool{}
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		base := root.Folder
		if base == "" {
			base = filepath.Join(root.Storage, "minecraftWorlds")
		}
		add := func(folder string) {
			w, err := Describe(folder, root.Storage, root.Name)
			if err == nil && !seen[w.ID] {
				seen[w.ID] = true
				worlds = append(worlds, w)
			}
		}
		if _, err := os.Stat(filepath.Join(base, "level.dat")); err == nil {
			add(base)
			continue
		}
		entries, err := os.ReadDir(base)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("Minecraft worlds could not be listed: %w", err)
		}
		for _, entry := range entries {
			if err = ctx.Err(); err != nil {
				return nil, err
			}
			if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
				add(filepath.Join(base, entry.Name()))
			}
		}
	}
	sort.Slice(worlds, func(i, j int) bool {
		if worlds[i].Modified == worlds[j].Modified {
			return worlds[i].Name < worlds[j].Name
		}
		return worlds[i].Modified > worlds[j].Modified
	})
	return worlds, nil
}

package assetlibrary

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (l *Library) SaveTexture(ctx context.Context, id, path, name, data string) ([]Item, error) {
	detail, e := l.Detail(id)
	if e != nil {
		return nil, e
	}
	path, e = safeName(path)
	if e != nil {
		return nil, e
	}
	if !strings.EqualFold(filepath.Ext(path), ".png") {
		return nil, errors.New("Edit a PNG texture")
	}
	found := false
	for _, f := range detail.Files {
		found = found || f.Path == path
	}
	if !found {
		return nil, errors.New("Texture is not part of this asset")
	}
	data = strings.TrimPrefix(data, "data:image/png;base64,")
	if len(data) > 24<<20 {
		return nil, errors.New("Edited image exceeds 16 MiB")
	}
	b, e := base64.StdEncoding.DecodeString(data)
	if e != nil {
		return nil, e
	}
	if _, e = checkedPNG(b); e != nil {
		return nil, e
	}
	dir, e := os.MkdirTemp("", "craftbox-texture-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	root, _ := l.RootFor(id)
	if e = copySource(ctx, root, dir); e != nil {
		return nil, e
	}
	if e = os.WriteFile(filepath.Join(dir, filepath.FromSlash(path)), b, 0600); e != nil {
		return nil, e
	}
	_, _, editedID, e := scan(dir)
	if e != nil {
		return nil, e
	}
	existing, e := l.List()
	if e != nil {
		return nil, e
	}
	for _, v := range existing {
		if v.ID == editedID {
			return []Item{v}, nil
		}
	}
	out, e := l.Import(ctx, detail.Item.Kind, []string{dir}, "Edited in CraftBox")
	if e != nil {
		return out, e
	}
	if len(out) > 0 {
		if out[0].ID == id {
			return out, nil
		}
		if e = l.Update(out[0].ID, name, false, detail.Item.Tags); e != nil {
			return out, e
		}
		if detail.Item.Kind == "skins" {
			if e = l.SetArmModel(out[0].ID, detail.Item.ArmModel); e != nil {
				return out, e
			}
			out[0].ArmModel = detail.Item.ArmModel
		}
		out[0].Name = name
	}
	return out, nil
}
func (l *Library) SetArmModel(id, model string) error {
	if model != "wide" && model != "slim" {
		return errors.New("Choose wide or slim arms")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	items, e := l.load()
	if e != nil {
		return e
	}
	for i := range items {
		if items[i].ID == id && items[i].Kind == "skins" {
			items[i].ArmModel = model
			items[i].ArmModelOverride = true
			return l.save(items)
		}
	}
	return errors.New("Skin not found")
}

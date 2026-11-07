package assetlibrary

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func (l *Library) CaptureSkin(ctx context.Context, name, source string, skin protocol.Skin, extra ...map[string]any) ([]Item, error) {
	w, h := int(skin.SkinImageWidth), int(skin.SkinImageHeight)
	if w < 1 || h < 1 || w > 2048 || h > 2048 || len(skin.SkinData) != w*h*4 {
		return nil, errors.New("Skin texture is invalid")
	}
	dir, e := os.MkdirTemp("", "craftbox-skin-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	writeImage := func(file string, w, h int, data []byte) error {
		if w < 1 || h < 1 || w > 2048 || h > 2048 || len(data) != w*h*4 {
			return errors.New("Invalid image dimensions")
		}
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		copy(img.Pix, data)
		var b bytes.Buffer
		if e := png.Encode(&b, img); e != nil {
			return e
		}
		return os.WriteFile(filepath.Join(dir, file), b.Bytes(), 0600)
	}
	if e = writeImage("skin.png", w, h, skin.SkinData); e != nil {
		return nil, e
	}
	if len(skin.CapeData) > 0 {
		if e = writeImage("cape.png", int(skin.CapeImageWidth), int(skin.CapeImageHeight), skin.CapeData); e != nil {
			return nil, e
		}
	}
	if len(skin.SkinGeometry) > 0 {
		if len(skin.SkinGeometry) > 4<<20 {
			return nil, errors.New("Skin geometry exceeds 4 MiB")
		}
		if e = os.WriteFile(filepath.Join(dir, "geometry.json"), skin.SkinGeometry, 0600); e != nil {
			return nil, e
		}
	}
	arm := "wide"
	if skin.ArmSize == protocol.ArmSizeSlim {
		arm = "slim"
	}
	if skinPatchSlim(map[string]any{"resourcePatch": base64.StdEncoding.EncodeToString(skin.SkinResourcePatch)}) {
		arm = "slim"
	}
	meta := map[string]any{"armModel": arm, "persona": skin.PersonaSkin, "hasAnimations": len(skin.Animations) > 0, "skinId": skin.SkinID, "player": name, "personaPieces": skin.PersonaPieces, "pieceTintColours": skin.PieceTintColours, "animationData": skin.AnimationData, "resourcePatch": skin.SkinResourcePatch, "geometryEngineVersion": skin.GeometryDataEngineVersion}
	for _, values := range extra {
		for key, value := range values {
			meta[key] = value
		}
	}
	b, e := json.Marshal(meta)
	if e != nil {
		return nil, e
	}
	if e = os.WriteFile(filepath.Join(dir, "capture.json"), b, 0600); e != nil {
		return nil, e
	}
	if len(skin.Animations) > 0 {
		for _, frame := range skin.Animations {
			if frame.AnimationType != protocol.SkinAnimationHead || frame.FrameCount < 1 || frame.FrameCount > 1024 {
				continue
			}
			width, height := int(frame.ImageWidth), int(frame.ImageHeight)
			if width < 1 || width > 2048 || height < 1 || height > 8192 || len(frame.ImageData) != width*height*4 {
				continue
			}
			frameHeight := height / int(frame.FrameCount)
			if frameHeight < 1 || frameHeight > 2048 {
				continue
			}
			if e = writeImage("face.png", width, frameHeight, frame.ImageData[:width*frameHeight*4]); e != nil {
				return nil, e
			}
			break
		}
		b, e = json.Marshal(skin.Animations)
		if e != nil {
			return nil, e
		}
		if len(b) > MaxPreviewBytes {
			return nil, errors.New("Skin animations exceed preview limit")
		}
		if e = os.WriteFile(filepath.Join(dir, "animations.json"), b, 0600); e != nil {
			return nil, e
		}
	}
	items, e := l.Import(ctx, "skins", []string{dir}, source)
	if e != nil {
		return items, e
	}
	if len(items) > 0 && strings.HasPrefix(items[0].Name, "craftbox-skin-") {
		if e = l.Update(items[0].ID, name, false, []string{}); e != nil {
			return items, e
		}
		items[0].Name = name
	}
	return items, nil
}

func (l *Library) Export(ids []string, name, format string, out io.Writer) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(ids) == 0 || len(ids) > 200 {
		return errors.New("Select between 1 and 200 assets")
	}
	items := []Item{}
	for _, id := range ids {
		v, e := l.get(id)
		if e != nil {
			return e
		}
		items = append(items, v)
	}
	if format == "png" {
		if len(items) != 1 || items[0].Kind != "skins" {
			return errors.New("Select one skin to export its PNG")
		}
		root, _ := l.RootFor(ids[0])
		file := "skin.png"
		if _, e := os.Stat(filepath.Join(root, file)); e != nil {
			b, e := readLimit(filepath.Join(root, "skins.json"), 4<<20)
			if e != nil {
				return e
			}
			var s struct {
				Skins []struct {
					Texture string `json:"texture"`
				} `json:"skins"`
			}
			if e = normalizedJSON(b, &s); e != nil || len(s.Skins) == 0 {
				return errors.New("Skin texture is missing")
			}
			file = s.Skins[0].Texture
		}
		if _, e := safeName(file); e != nil {
			return e
		}
		b, e := readLimit(filepath.Join(root, filepath.FromSlash(file)), MaxPreviewBytes)
		if e != nil {
			return e
		}
		_, e = out.Write(b)
		return e
	}
	if format != "mcpack" && format != "zip" {
		return errors.New("Choose PNG, mcpack or ZIP")
	}
	kind := items[0].Kind
	for _, v := range items {
		if v.Kind != kind {
			return errors.New("Export skins and resource packs separately")
		}
	}
	if format == "mcpack" && kind == "packs" && len(items) != 1 {
		return errors.New("Export one resource pack as mcpack, or select ZIP for several packs")
	}
	z := zip.NewWriter(out)
	write := func(name string, b []byte) error {
		f, e := z.Create(name)
		if e != nil {
			return e
		}
		_, e = f.Write(b)
		return e
	}
	if kind == "skins" && format == "mcpack" {
		key := strings.Join(ids, ",")
		packID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("craftbox:skinpack:"+key))
		moduleID := uuid.NewSHA1(packID, []byte("module"))
		serial := "craftbox_" + packID.String()
		entries := []map[string]string{}
		lang := "pack.name=" + strings.ReplaceAll(strings.ReplaceAll(name, "\n", " "), "\r", " ") + "\nskinpack." + serial + "=" + strings.ReplaceAll(strings.ReplaceAll(name, "\n", " "), "\r", " ") + "\n"
		for i, v := range items {
			texture := v.Image
			at := strings.IndexByte(texture, ',')
			if at < 0 {
				return errors.New("Skin image is missing")
			}
			b, e := base64.StdEncoding.DecodeString(texture[at+1:])
			if e != nil {
				return e
			}
			if _, e = checkedPNG(b); e != nil {
				return e
			}
			geometry := "geometry.humanoid.custom"
			if v.ArmModel == "slim" {
				geometry = "geometry.humanoid.customSlim"
			}
			skinKey := fmt.Sprintf("skin%d", i)
			file := skinKey + ".png"
			if e = write(file, b); e != nil {
				return e
			}
			entries = append(entries, map[string]string{"localization_name": skinKey, "geometry": geometry, "texture": file, "type": "free"})
			lang += "skin." + serial + "." + skinKey + "=" + strings.ReplaceAll(strings.ReplaceAll(v.Name, "\n", " "), "\r", " ") + "\n"
		}
		manifest := map[string]any{"format_version": 1, "header": map[string]any{"name": "pack.name", "uuid": packID.String(), "version": []int{1, 0, 0}}, "modules": []any{map[string]any{"type": "skin_pack", "uuid": moduleID.String(), "version": []int{1, 0, 0}}}}
		for file, value := range map[string]any{"manifest.json": manifest, "skins.json": map[string]any{"serialize_name": serial, "localization_name": serial, "skins": entries}, "texts/languages.json": []string{"en_US"}} {
			b, e := json.MarshalIndent(value, "", "  ")
			if e != nil {
				return e
			}
			if e = write(file, b); e != nil {
				return e
			}
		}
		if e := write("texts/en_US.lang", []byte(lang)); e != nil {
			return e
		}
	} else {
		for _, v := range items {
			root, _ := l.RootFor(v.ID)
			files, _, _, e := scan(root)
			if e != nil {
				return e
			}
			prefix := ""
			if len(items) > 1 {
				prefix = v.ID[:12] + "/"
			}
			if format == "zip" && v.Original != "" {
				p, e := l.originalPath(v.Original)
				if e != nil {
					return e
				}
				in, e := os.Open(p)
				if e != nil {
					return e
				}
				entry, e := z.Create(prefix + "original-pack.mcpack")
				if e != nil {
					in.Close()
					return e
				}
				_, e = io.Copy(entry, in)
				in.Close()
				if e != nil {
					return e
				}
			}
			for _, f := range files {
				in, e := os.Open(filepath.Join(root, filepath.FromSlash(f.Path)))
				if e != nil {
					return e
				}
				target, e := z.Create(prefix + f.Path)
				if e != nil {
					in.Close()
					return e
				}
				_, e = io.Copy(target, in)
				in.Close()
				if e != nil {
					return e
				}
			}
		}
	}
	return z.Close()
}

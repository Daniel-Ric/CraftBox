package assetlibrary

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type MinecraftPackage struct {
	Name            string `json:"Name"`
	Version         string `json:"Version"`
	InstallLocation string `json:"InstallLocation"`
	Family          string `json:"PackageFamilyName"`
	Application     string `json:"Application"`
}
type Installation struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Channel          string `json:"channel"`
	Layout           string `json:"layout"`
	Storage          string `json:"storage"`
	Version          string `json:"version"`
	GameVersion      string `json:"gameVersion"`
	Location         string `json:"location"`
	Application      string `json:"application"`
	StorageAvailable bool   `json:"storageAvailable"`
}

func DiscoverMinecraft(appdata, localdata string, packages []MinecraftPackage) []Installation {
	out := []Installation{}
	seen := map[string]bool{}
	add := func(root, channel, layout string) {
		i, e := os.Stat(root)
		if e != nil || !i.IsDir() {
			return
		}
		root = filepath.Clean(root)
		key := strings.ToLower(root)
		if seen[key] {
			return
		}
		seen[key] = true
		h := sha256.Sum256([]byte(key))
		name := "Minecraft for Windows"
		if channel == "preview" {
			name = "Minecraft Preview"
		}
		v := Installation{ID: hex.EncodeToString(h[:16]), Name: name, Channel: channel, Layout: layout, Storage: root, StorageAvailable: true}
		for _, p := range packages {
			preview := strings.Contains(strings.ToLower(p.Name), "preview") || strings.Contains(strings.ToLower(p.Name), "beta")
			if preview == (channel == "preview") {
				v.Version = p.Version
				v.GameVersion = GameVersion(p.Version)
				v.Location = p.InstallLocation
				if p.Family != "" && p.Application != "" {
					v.Application = p.Family + "!" + p.Application
				} else if strings.HasPrefix(p.Application, "uri:") {
					v.Application = p.Application
				}
				break
			}
		}
		if layout == "GDK" {
			v.Name += " · " + filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(root))))
		}
		out = append(out, v)
	}
	if appdata != "" {
		for _, entry := range []struct{ name, channel string }{{"Minecraft Bedrock", "stable"}, {"Minecraft Bedrock Preview", "preview"}} {
			base := filepath.Join(appdata, entry.name, "Users")
			users, _ := os.ReadDir(base)
			for _, u := range users {
				if u.IsDir() {
					add(filepath.Join(base, u.Name(), "games", "com.mojang"), entry.channel, "GDK")
				}
			}
			add(filepath.Join(appdata, entry.name, "games", "com.mojang"), entry.channel, "GDK")
		}
	}
	if localdata != "" {
		add(filepath.Join(localdata, "Packages", "Microsoft.MinecraftUWP_8wekyb3d8bbwe", "LocalState", "games", "com.mojang"), "stable", "UWP")
		add(filepath.Join(localdata, "Packages", "Microsoft.MinecraftWindowsBeta_8wekyb3d8bbwe", "LocalState", "games", "com.mojang"), "preview", "UWP")
	}
	for _, p := range packages {
		channel := "stable"
		folder := "Minecraft Bedrock"
		if strings.Contains(strings.ToLower(p.Name), "preview") || strings.Contains(strings.ToLower(p.Name), "beta") {
			channel = "preview"
			folder += " Preview"
		}
		found := false
		for _, v := range out {
			if v.Channel == channel {
				found = true
			}
		}
		if found {
			continue
		}
		storage := filepath.Join(appdata, folder, "Users", "Shared", "games", "com.mojang")
		h := sha256.Sum256([]byte(strings.ToLower(storage)))
		application := p.Application
		if p.Family != "" && !strings.HasPrefix(application, "uri:") {
			application = p.Family + "!" + application
		}
		out = append(out, Installation{ID: hex.EncodeToString(h[:16]), Name: folder, Channel: channel, Layout: "GDK", Storage: storage, Version: p.Version, GameVersion: GameVersion(p.Version), Location: p.InstallLocation, Application: application, StorageAvailable: false})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Channel != out[j].Channel {
			return out[i].Channel == "stable"
		}
		si := strings.Contains(strings.ToLower(out[i].Storage), string(filepath.Separator)+"shared"+string(filepath.Separator))
		sj := strings.Contains(strings.ToLower(out[j].Storage), string(filepath.Separator)+"shared"+string(filepath.Separator))
		if si != sj {
			return si
		}
		return out[i].Storage < out[j].Storage
	})
	return out
}

func (l *Library) ImportMinecraft(ctx context.Context, target Installation) ([]Item, error) {
	out := []Item{}
	for _, folder := range []string{"resource_packs", "development_resource_packs", "skin_packs"} {
		base := filepath.Join(target.Storage, folder)
		entries, e := os.ReadDir(base)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return out, e
		}
		for _, entry := range entries {
			if e = ctx.Err(); e != nil {
				return out, e
			}
			if !entry.IsDir() && !strings.EqualFold(filepath.Ext(entry.Name()), ".mcpack") {
				continue
			}
			p := filepath.Join(base, entry.Name())
			if entry.IsDir() {
				if _, e = os.Stat(filepath.Join(p, "manifest.json")); os.IsNotExist(e) {
					continue
				}
			}
			kind := "packs"
			if folder == "skin_packs" {
				kind = "skins"
			}
			items, e := l.Import(ctx, kind, []string{p}, target.Name)
			if e != nil {
				return out, fmt.Errorf("%s: %w", entry.Name(), e)
			}
			out = append(out, items...)
		}
	}
	return out, nil
}

func (l *Library) Install(ctx context.Context, ids []string, target Installation) (string, error) {
	if len(ids) == 0 {
		return "", errors.New("Select an asset to add to Minecraft")
	}
	all, e := l.List()
	if e != nil {
		return "", e
	}
	byID := map[string]Item{}
	for _, v := range all {
		byID[v.ID] = v
	}
	first, ok := byID[ids[0]]
	if !ok {
		return "", errors.New("Asset not found")
	}
	if first.Encrypted {
		return "", errors.New("Restore this encrypted pack from saved captures before adding it to Minecraft")
	}
	if first.Kind == "packs" && first.MinimumVersion != "" && target.GameVersion != "" && versionLess(target.GameVersion, first.MinimumVersion) {
		return "", fmt.Errorf("This pack needs Minecraft %s or later; the selected installation is %s", first.MinimumVersion, target.GameVersion)
	}
	if first.Kind == "packs" && len(ids) > 1 {
		return "", errors.New("Add resource packs to Minecraft one at a time")
	}
	for _, id := range ids {
		if v, ok := byID[id]; !ok || v.Kind != first.Kind {
			return "", errors.New("Select assets of the same type")
		}
	}
	folder := "resource_packs"
	if first.Kind == "skins" {
		folder = "skin_packs"
	}
	parent := filepath.Join(target.Storage, folder)
	if e = os.MkdirAll(parent, 0700); e != nil {
		return "", e
	}
	h := sha256.Sum256([]byte(strings.Join(ids, ",")))
	dest := filepath.Join(parent, "CraftBox-"+hex.EncodeToString(h[:12]))
	if _, e = os.Lstat(dest); !os.IsNotExist(e) {
		return "", errors.New("This selection is already installed. Existing Minecraft files were kept")
	}
	stage, e := os.MkdirTemp(parent, ".craftbox-import-")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(stage)
	if first.Kind == "packs" {
		root, _ := l.RootFor(first.ID)
		if e = copySource(ctx, root, stage); e != nil {
			return "", e
		}
	} else {
		f, e := os.CreateTemp("", "craftbox-import-*.mcpack")
		if e != nil {
			return "", e
		}
		temp := f.Name()
		defer os.Remove(temp)
		e = l.Export(ids, "CraftBox skins", "mcpack", f)
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return "", e
		}
		if e = copySource(ctx, temp, stage); e != nil {
			return "", e
		}
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	if e = os.Rename(stage, dest); e != nil {
		return "", e
	}
	return dest, nil
}

func GameVersion(packageVersion string) string {
	parts := strings.Split(packageVersion, ".")
	if len(parts) != 4 {
		return packageVersion
	}
	build, e := strconv.Atoi(parts[2])
	if e != nil {
		return ""
	}
	return parts[0] + "." + parts[1] + "." + strconv.Itoa(build/100)
}
func versionLess(a, b string) bool {
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aa) {
			av, _ = strconv.Atoi(aa[i])
		}
		if i < len(bb) {
			bv, _ = strconv.Atoi(bb[i])
		}
		if av != bv {
			return av < bv
		}
	}
	return false
}

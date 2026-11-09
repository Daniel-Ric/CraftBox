//go:build gui && !android

package desktop

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/modelexport"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/utils"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type AssetLibraryState struct {
	Items             []assetlibrary.Item `json:"items"`
	Bytes             int64               `json:"bytes"`
	Directory         string              `json:"directory"`
	EquipmentRevision string              `json:"equipmentRevision"`
	BlockRevision     string              `json:"blockRevision"`
	BlockRevisions    map[string]string   `json:"blockRevisions"`
}
type MinecraftState struct {
	Installations []assetlibrary.Installation `json:"installations"`
	Warning       string                      `json:"warning"`
}

func (a *App) GetAssetLibrary() (AssetLibraryState, error) {
	items, e := a.assets().List()
	out := AssetLibraryState{Items: items, Directory: a.assets().Root, BlockRevisions: map[string]string{}}
	out.BlockRevision = a.assets().BlockDefinitionsRevision()
	if info, err := os.Stat(filepath.Join(out.Directory, "equipment.json")); err == nil {
		out.EquipmentRevision = fmt.Sprint(info.ModTime().UnixNano())
	}
	for _, v := range items {
		out.Bytes += v.Bytes
		if v.Kind == "packs" {
			out.BlockRevisions[v.ID] = a.assets().BlockDefinitionsRevisionFor(v.ID)
		}
	}
	if entries, e := os.ReadDir(filepath.Join(a.assets().Root, "originals")); e == nil {
		for _, v := range entries {
			if info, e := v.Info(); e == nil && info.Mode().IsRegular() {
				out.Bytes += info.Size()
			}
		}
	}
	return out, e
}
func (a *App) ImportAssets(kind string, paths []string) ([]assetlibrary.Item, error) {
	if len(paths) == 0 {
		selected, e := wruntime.OpenMultipleFilesDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Import skins or resource packs", Filters: []wruntime.FileFilter{{DisplayName: "Minecraft assets", Pattern: "*.png;*.mcpack;*.zip"}}})
		if e != nil {
			return nil, e
		}
		paths = selected
	}
	return a.modelAssetsChanged(a.assets().Import(a.ctx, kind, paths, "Local file"))
}
func (a *App) ImportAssetFolder(kind string) ([]assetlibrary.Item, error) {
	p, e := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Import asset folder"})
	if e != nil || p == "" {
		return []assetlibrary.Item{}, e
	}
	return a.modelAssetsChanged(a.assets().Import(a.ctx, kind, []string{p}, "Local folder"))
}
func (a *App) GetAssetDetail(id string) (assetlibrary.Detail, error) { return a.assets().Detail(id) }
func (a *App) GetSkinAppearance(id string) (assetlibrary.SkinAppearance, error) {
	return a.assets().SkinAppearance(id)
}
func (a *App) GetAssetObjects(id string) ([]assetlibrary.Object, error) {
	return a.assets().Objects(id)
}
func (a *App) PreviewAssetObject(id, objectID, variant string) (assetlibrary.ObjectPreview, error) {
	p, e := a.assets().ObjectPreview(id, objectID, variant)
	if e != nil {
		return p, e
	}
	if p.Object.Kind == "block" && p.GeometryID == "" && (!strings.Contains(p.Object.Name, ":") || strings.HasPrefix(p.Object.Name, "minecraft:")) {
		if geometry, note, err := scene.BlockAssetGeometry(p.Object.Name, p.Object.Textures); err == nil {
			p.Geometry = geometry
			p.Warnings = append(p.Warnings, note)
		} else if strings.HasSuffix(p.Object.Definition, "blocks.json") {
			p.Geometry = nil
			p.Warnings = append(p.Warnings, err.Error())
		}
	}
	return p, nil
}

func (a *App) PreviewPickedBlock(id, objectID string, states map[string]any) (assetlibrary.ObjectPreview, error) {
	return a.assets().BlockStatePreview(id, objectID, states, scene.MatchBlockCondition)
}
func (a *App) ImportAssetBytes(kind, name, data string) ([]assetlibrary.Item, error) {
	if len(data) > 180<<20 {
		return nil, fmt.Errorf("Drop files up to 128 MiB, or use Import files for larger packs")
	}
	raw, e := base64.StdEncoding.DecodeString(data)
	if e != nil {
		return nil, e
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".png" && ext != ".zip" && ext != ".mcpack" {
		return nil, fmt.Errorf("Drop PNG, mcpack or ZIP files")
	}
	f, e := os.CreateTemp("", "craftbox-drop-*"+ext)
	if e != nil {
		return nil, e
	}
	defer os.Remove(f.Name())
	_, e = f.Write(raw)
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return nil, e
	}
	return a.modelAssetsChanged(a.assets().Import(a.ctx, kind, []string{f.Name()}, "Dropped file: "+filepath.Base(name)))
}
func (a *App) SaveAssetTexture(id, path, name, data string) ([]assetlibrary.Item, error) {
	return a.modelAssetsChanged(a.assets().SaveTexture(a.ctx, id, path, name, data))
}
func (a *App) SetAssetArmModel(id, model string) error { return a.assets().SetArmModel(id, model) }
func (a *App) ReadAssetFile(id, path string) (assetlibrary.Preview, error) {
	return a.assets().Read(id, path)
}
func (a *App) UpdateAsset(id, name string, favourite bool, tags []string) error {
	err := a.assets().Update(id, name, favourite, tags)
	if err == nil {
		a.invalidateModelPicker()
	}
	return err
}
func (a *App) DeleteAssets(ids []string) error {
	err := a.assets().Delete(ids)
	if err == nil {
		a.invalidateModelPicker()
	}
	return err
}
func (a *App) CompareAssets(left, right string) ([]assetlibrary.Change, error) {
	return a.assets().Compare(left, right)
}
func (a *App) InspectAssetStack(ids []string) (assetlibrary.Stack, error) {
	return a.assets().Stack(ids)
}
func (a *App) GetMinecraftInstallations() MinecraftState {
	all, e := assetlibrary.InstalledMinecraft()
	out := MinecraftState{Installations: all}
	if e != nil {
		out.Warning = "Minecraft storage was checked, but its installed package version could not be read: " + e.Error()
	}
	return out
}
func (a *App) ImportMinecraftAssets(id string) ([]assetlibrary.Item, error) {
	target, e := a.assetInstallation(id)
	if e != nil {
		return nil, e
	}
	return a.modelAssetsChanged(a.assets().ImportMinecraft(a.ctx, target))
}
func (a *App) InstallAssetsInMinecraft(ids []string, id string) (string, error) {
	target, e := a.assetInstallation(id)
	if e != nil {
		return "", e
	}
	return a.assets().Install(a.ctx, ids, target)
}
func (a *App) OpenMinecraftInstallation(id string) error {
	target, e := a.assetInstallation(id)
	if e != nil {
		return e
	}
	if target.Application == "" {
		return fmt.Errorf("The Minecraft application could not be identified. Open it from the Xbox app")
	}
	if target.Application == "uri:minecraft" || target.Application == "uri:minecraft-preview" {
		wruntime.BrowserOpenURL(a.ctx, strings.TrimPrefix(target.Application, "uri:")+"://")
		return nil
	}
	return openResultDirectory("shell:AppsFolder\\" + target.Application)
}
func (a *App) OpenAssetFolder(id string) error {
	if _, e := a.assets().Detail(id); e != nil {
		return e
	}
	p, e := a.assets().RootFor(id)
	if e != nil {
		return e
	}
	return a.OpenResultFolder(p)
}
func (a *App) ExportAssets(ids []string, name, format string) (string, error) {
	if format != "png" && format != "mcpack" && format != "zip" {
		return "", fmt.Errorf("Unsupported asset export format")
	}
	name = utils.MakeValidFilename(name)
	if name == "" {
		name = "CraftBox assets"
	}
	p, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export assets", DefaultFilename: name + "." + format, Filters: []wruntime.FileFilter{{DisplayName: strings.ToUpper(format), Pattern: "*." + format}}})
	if e != nil || p == "" {
		return "", e
	}
	if !strings.EqualFold(filepath.Ext(p), "."+format) {
		p += "." + format
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".craftbox-assets-*")
	if e != nil {
		return "", e
	}
	defer os.Remove(f.Name())
	e = a.assets().Export(ids, name, format, f)
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return "", e
	}
	if e = os.Rename(f.Name(), p); e != nil {
		return "", e
	}
	return p, nil
}
func (a *App) SaveAssetModel(name, data, format string) (string, error) {
	if len(data) > 96<<20 {
		return "", fmt.Errorf("Model exceeds 72 MiB")
	}
	raw, e := base64.StdEncoding.DecodeString(data)
	if e != nil {
		return "", e
	}
	if len(raw) < 12 || string(raw[:4]) != "glTF" {
		return "", fmt.Errorf("Expected a binary glTF model")
	}
	options := modelexport.Options{Format: format, Scale: 1}
	if e = options.Validate(); e != nil {
		return "", e
	}
	ext, _ := modelexport.Extension(format)
	p, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Export model", DefaultFilename: utils.MakeValidFilename(name) + ext})
	if e != nil || p == "" {
		return "", e
	}
	if !strings.HasSuffix(strings.ToLower(p), ext) {
		p += ext
	}
	source, e := os.CreateTemp("", "craftbox-asset-*.glb")
	if e != nil {
		return "", e
	}
	defer os.Remove(source.Name())
	defer source.Close()
	if _, e = source.Write(raw); e != nil {
		return "", e
	}
	source.Seek(0, 0)
	out, e := os.CreateTemp(filepath.Dir(p), ".craftbox-asset-model-*")
	if e != nil {
		return "", e
	}
	defer os.Remove(out.Name())
	e = modelexport.Convert(context.Background(), source, out, options, nil)
	ce := out.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return "", e
	}
	if e = os.Rename(out.Name(), p); e != nil {
		return "", e
	}
	return p, nil
}

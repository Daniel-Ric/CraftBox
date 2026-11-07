package assetlibrary

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed blockdata/*.json
var bundledBlockDefinitions embed.FS

const capturedBlockPrefix = "captured_blocks/"
const blockDefinitionLimit = 16 << 20

var blockIdentifier = regexp.MustCompile(`^[a-z0-9_.-]+:[a-z0-9_./-]+$`)
var visualBlockKeys = []string{"minecraft:geometry", "minecraft:material_instances", "minecraft:transformation", "minecraft:rotation", "minecraft:embedded_visual"}

type blockDefinitionSet struct {
	Version     int               `json:"version"`
	PackID      string            `json:"packId"`
	Definitions []json.RawMessage `json:"definitions"`
}

func blockDefinitionPath(name string) (string, error) {
	if len(name) > 256 || !blockIdentifier.MatchString(name) {
		return "", fmt.Errorf("Invalid block identifier")
	}
	return safeName(capturedBlockPrefix + strings.ReplaceAll(name, ":", "/") + ".json")
}
func visualBlockComponents(raw any) map[string]any {
	out := map[string]any{}
	source := objectMap(raw)
	for _, key := range visualBlockKeys {
		if value, ok := source[key]; ok {
			out[key] = value
		}
	}
	return out
}
func visualBlockDocument(raw []byte) (map[string]any, string, error) {
	if len(raw) > 256<<10 {
		return nil, "", fmt.Errorf("Block definition exceeds 256 KiB")
	}
	var doc map[string]any
	if err := normalizedJSON(raw, &doc); err != nil {
		return nil, "", err
	}
	block := objectMap(doc["minecraft:block"])
	description := objectMap(block["description"])
	name := objectString(description["identifier"])
	if _, err := blockDefinitionPath(name); err != nil {
		return nil, "", err
	}
	desc := map[string]any{"identifier": name}
	if states, ok := description["states"]; ok {
		desc["states"] = states
	}
	result := map[string]any{"description": desc, "components": visualBlockComponents(block["components"])}
	if permutations, ok := block["permutations"].([]any); ok {
		if len(permutations) > 4096 {
			return nil, "", fmt.Errorf("Too many block permutations")
		}
		var kept []any
		for _, permutation := range permutations {
			value := objectMap(permutation)
			components := visualBlockComponents(value["components"])
			if len(components) > 0 {
				kept = append(kept, map[string]any{"condition": value["condition"], "components": components})
			}
		}
		if len(kept) > 0 {
			result["permutations"] = kept
		}
	}
	version := doc["format_version"]
	if version == nil {
		version = "1.21.0"
	}
	return map[string]any{"format_version": version, "minecraft:block": result}, name, nil
}
func blockDefinitionBelongs(x objectIndex, name string, doc map[string]any) bool {
	if _, ok := x.docs["blocks.json"][name]; ok {
		return true
	}
	if _, ok := x.terrain[name]; ok {
		return true
	}
	block := objectMap(doc["minecraft:block"])
	components := objectMap(block["components"])
	if blockVisualsBelong(x, components) {
		return true
	}
	if permutations, ok := block["permutations"].([]any); ok {
		for _, p := range permutations {
			merged := map[string]any{}
			for key, value := range components {
				merged[key] = value
			}
			for key, value := range objectMap(objectMap(p)["components"]) {
				merged[key] = value
			}
			if blockVisualsBelong(x, merged) {
				return true
			}
		}
	}
	return false
}
func blockVisualsBelong(x objectIndex, components map[string]any) bool {
	merged := map[string]any{}
	for key, value := range components {
		merged[key] = value
	}
	for _, key := range []string{"geometry", "material_instances"} {
		if merged["minecraft:"+key] == nil {
			merged["minecraft:"+key] = objectMap(merged["minecraft:embedded_visual"])[key]
		}
	}
	components = merged
	geometry := objectString(components["minecraft:geometry"])
	if geometry == "" {
		geometry = objectString(objectMap(components["minecraft:geometry"])["identifier"])
	}
	if x.geometry[geometry] == nil {
		return false
	}
	materials := objectMap(components["minecraft:material_instances"])
	if nested := objectMap(materials["materials"]); len(nested) > 0 {
		materials = nested
	}
	for _, value := range materials {
		ref := objectString(objectMap(value)["texture"])
		if len(x.terrain[ref]) > 0 || x.texture(ref) != "" {
			return true
		}
	}
	return false
}
func (l *Library) blockDefinitionStamp(id string) string {
	if info, err := os.Lstat(filepath.Join(l.Root, "block-definitions", id+".json")); err == nil {
		return fmt.Sprintf("local:%d:%d", info.ModTime().UnixNano(), info.Size())
	}
	if info, err := bundledBlockDefinitions.Open("blockdata/" + id + ".json"); err == nil {
		info.Close()
		return "bundled:" + id
	}
	return ""
}
func (l *Library) blockDefinitionDocuments(id string) (map[string]map[string]any, int, error) {
	raw, err := readLimit(filepath.Join(l.Root, "block-definitions", id+".json"), blockDefinitionLimit)
	if os.IsNotExist(err) {
		raw, err = bundledBlockDefinitions.ReadFile("blockdata/" + id + ".json")
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
	}
	if err != nil {
		return nil, 0, err
	}
	if len(raw) > blockDefinitionLimit {
		return nil, 0, fmt.Errorf("Block definition cache is too large")
	}
	var set blockDefinitionSet
	if err = json.Unmarshal(raw, &set); err != nil {
		return nil, 0, err
	}
	if set.Version != 1 || set.PackID != id || len(set.Definitions) > 8192 {
		return nil, 0, fmt.Errorf("Block definition cache does not match this pack")
	}
	docs := map[string]map[string]any{}
	for _, entry := range set.Definitions {
		doc, name, err := visualBlockDocument(entry)
		if err != nil {
			return nil, 0, err
		}
		file, _ := blockDefinitionPath(name)
		if _, ok := docs[file]; ok {
			return nil, 0, fmt.Errorf("Duplicate captured block definition")
		}
		docs[file] = doc
	}
	return docs, len(raw), nil
}
func (l *Library) injectBlockDefinitions(id string, x *objectIndex) {
	docs, size, err := l.blockDefinitionDocuments(id)
	if err != nil {
		x.blockWarnings = append(x.blockWarnings, "Saved block components could not be read; capture this server again")
		return
	}
	explicit := map[string]bool{}
	for _, doc := range x.docs {
		if name := objectString(objectMap(objectMap(doc["minecraft:block"])["description"])["identifier"]); name != "" {
			explicit[name] = true
		}
	}
	for _, file := range sortedKeys(docs) {
		doc := docs[file]
		name := objectString(objectMap(objectMap(doc["minecraft:block"])["description"])["identifier"])
		if !explicit[name] && blockDefinitionBelongs(*x, name, doc) {
			if _, exists := x.docs[file]; !exists {
				x.docs[file] = doc
			}
		}
	}
	x.sourceBytes += size
}
func (l *Library) SaveBlockDefinitions(id string, definitions []json.RawMessage) error {
	if len(definitions) > 8192 {
		return fmt.Errorf("Too many captured block definitions")
	}
	x, err := l.catalog(id)
	if err != nil {
		return err
	}
	docs := map[string]json.RawMessage{}
	total := 0
	for _, raw := range definitions {
		total += len(raw)
		if total > blockDefinitionLimit {
			return fmt.Errorf("Captured block definitions exceed 16 MiB")
		}
		doc, name, err := visualBlockDocument(raw)
		if err != nil {
			return err
		}
		if blockDefinitionBelongs(x, name, doc) {
			encoded, _ := json.Marshal(doc)
			docs[name] = encoded
		}
	}
	if len(docs) == 0 {
		return nil
	}
	l.definitionMu.Lock()
	defer l.definitionMu.Unlock()
	if previous, _, readErr := l.blockDefinitionDocuments(id); readErr == nil {
		for _, doc := range previous {
			name := objectString(objectMap(objectMap(doc["minecraft:block"])["description"])["identifier"])
			if _, exists := docs[name]; !exists && blockDefinitionBelongs(x, name, doc) {
				encoded, _ := json.Marshal(doc)
				docs[name] = encoded
			}
		}
	}
	if len(docs) > 8192 {
		return fmt.Errorf("Too many captured block definitions")
	}
	set := blockDefinitionSet{Version: 1, PackID: id}
	for _, name := range sortedKeys(docs) {
		set.Definitions = append(set.Definitions, docs[name])
	}
	raw, err := json.Marshal(set)
	if err != nil {
		return err
	}
	if len(raw) > blockDefinitionLimit {
		return fmt.Errorf("Captured block definitions exceed 16 MiB")
	}
	folder := filepath.Join(l.Root, "block-definitions")
	target := filepath.Join(folder, id+".json")
	if old, e := readLimit(target, blockDefinitionLimit); e == nil && bytes.Equal(old, raw) {
		return nil
	}
	if err = os.MkdirAll(folder, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(folder, "block-components-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), target); err != nil {
		return err
	}
	l.catalogMu.Lock()
	delete(l.catalogs, id)
	l.catalogMu.Unlock()
	l.blockRevision.Add(1)
	return nil
}
func (l *Library) BlockDefinitionsRevision() string {
	var stamp int64
	if info, err := os.Stat(filepath.Join(l.Root, "block-definitions")); err == nil {
		stamp = info.ModTime().UnixNano()
	}
	return fmt.Sprintf("%d:%d", stamp, l.blockRevision.Load())
}
func (l *Library) capturedDefinitionPreview(id, name string) (Preview, error) {
	file, err := safeName(name)
	if err != nil {
		return Preview{}, err
	}
	x, err := l.catalog(id)
	if err != nil {
		return Preview{}, err
	}
	doc, ok := x.docs[file]
	if !ok {
		return Preview{}, fmt.Errorf("Captured block definition is unavailable")
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	return Preview{Path: file, Kind: "json", Data: string(raw)}, err
}
func appendCapturedDefinitionFiles(x objectIndex, d *Detail) {
	seen := map[string]bool{}
	for _, file := range d.Files {
		seen[file.Path] = true
	}
	for _, file := range sortedKeys(x.docs) {
		if !strings.HasPrefix(file, capturedBlockPrefix) || seen[file] {
			continue
		}
		raw, _ := json.Marshal(x.docs[file])
		sum := sha256.Sum256(raw)
		d.Files = append(d.Files, File{Path: file, Bytes: int64(len(raw)), Hash: hex.EncodeToString(sum[:]), Kind: "json"})
	}
}

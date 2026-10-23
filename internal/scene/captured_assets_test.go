package scene

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCapturedCustomBlockGallery(t *testing.T) {
	root := os.Getenv("CRAFT_BOX_CAPTURE_ASSETS")
	if root == "" {
		t.Skip("set CRAFT_BOX_CAPTURE_ASSETS to a captured world directory")
	}
	c, err := LoadCatalog(os.Getenv("CRAFT_BOX_3D_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"resource_packs", "behavior_packs"} {
		packs, err := os.ReadDir(filepath.Join(root, folder))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range packs {
			if !p.IsDir() {
				continue
			}
			f := os.DirFS(filepath.Join(root, folder, p.Name()))
			if folder == "resource_packs" {
				err = c.AddPack(f)
			} else {
				err = c.AddBlockDefinitions(f)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	names := []string{}
	for name := range c.definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := map[string]*Part{}
	approximate, missing, drawn := 0, 0, 0
	for i, name := range names {
		states := map[string]any{}
		for key, values := range c.definitions[name].Description.States {
			if a, ok := values.([]any); ok && len(a) > 0 {
				states[key] = a[0]
			}
		}
		model, ok := c.customModel(name, states)
		if !ok {
			t.Fatal("definition not resolved", name)
		}
		if model.approx {
			approximate++
			t.Log("approximate custom block", name)
		}
		noTexture := false
		for _, q := range model.quads {
			if q.texture.missing {
				noTexture = true
			}
			p := parts[q.mode]
			if p == nil {
				p = newPart(q.mode)
				parts[q.mode] = p
			}
			emitModelQuad(p, q, [3]float32{float32(i%12) * 3, 0, float32(i/12) * 3})
		}
		if noTexture {
			missing++
			t.Log("missing custom material", name)
		}
		if len(model.quads) > 0 {
			drawn++
		}
	}

	for i, name := range []string{"minecraft:oak_leaves", "minecraft:birch_leaves", "minecraft:azalea_leaves", "minecraft:water"} {
		mode := "mask"
		if name == "minecraft:water" {
			mode = "blend"
		}
		p := parts[mode]
		if p == nil {
			p = newPart(mode)
			parts[mode] = p
		}
		for face, d := range directions {
			var q [4][3]float32
			for v, corner := range corners[face] {
				for a := range corner {
					q[v][a] = float32(corner[a])
				}
			}
			emitQuad(p, q, [3]float32{float32(d[0]), float32(d[1]), float32(d[2])}, [3]float32{float32(i) * 3, 0, -3}, c.texture(name, face))
		}
	}
	chunk := ChunkMesh{Parts: []Part{}}
	for _, mode := range []string{"opaque", "mask", "blend"} {
		if p := parts[mode]; p != nil {
			chunk.Parts = append(chunk.Parts, *p)
		}
	}
	if len(names) == 0 || drawn == 0 {
		t.Fatal("capture provided no custom models")
	}
	t.Logf("Custom gallery: %d definitions, %d models drawn, %d approximate, %d missing materials", len(names), drawn, approximate, missing)
	t.Logf("Unresolved image references: %v", c.MissingTexturePaths())
	if output := os.Getenv("CRAFT_BOX_CUSTOM_GALLERY_OUT"); output != "" {
		page := Page{Sequence: 1, Reset: true, Complete: true, Total: 1, Chunks: []PackedChunk{packChunk(chunk)}, AtlasRevision: c.revision, AtlasSize: atlasSize, Patches: c.atlasPatches(0), Source: "Captured custom block model gallery"}
		data, err := json.Marshal([]Page{page})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(output, data, 0600); err != nil {
			t.Fatal(err)
		}
		snapshot := Snapshot{Sequence: 1, Reset: true, Chunks: []ChunkMesh{chunk}, Atlas: c.atlasURL(), Source: page.Source}
		glb, err := EncodeGLB(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(output+".glb", glb, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

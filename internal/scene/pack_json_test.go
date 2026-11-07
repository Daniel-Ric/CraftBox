package scene

import (
	"testing"
	"testing/fstest"
)

func TestPackJSONAcceptsUTF8BOM(t *testing.T) {
	for _, prefix := range []string{"", "\xef\xbb\xbf"} {
		t.Run(map[bool]string{true: "bom", false: "plain"}[prefix != ""], func(t *testing.T) {
			files := fstest.MapFS{}
			for path, text := range map[string]string{
				"blocks.json":                           `{"test:block":{"textures":"test_texture"}}`,
				"textures/terrain_texture.json":         `{"texture_data":{"test_texture":{"textures":"textures/blocks/test"}}}`,
				"models/blocks/test.json":               `{"minecraft:geometry":[{"description":{"identifier":"geometry.test","texture_width":16,"texture_height":16},"bones":[]}]}`,
				"entity/test.json":                      `{"minecraft:client_entity":{"description":{"identifier":"test:actor","geometry":{"default":"geometry.test"}}}}`,
				"render_controllers/glow/blue/1x1.json": "// pack comment\n" + `{"render_controllers":{"controller.render.test":{"geometry":"Geometry.default",},},}`,
				"animations/test.json":                  `{"animations":{"animation.test":{"loop":true}}}`,
				"animation_controllers/test.json":       `{"animation_controllers":{"controller.animation.test":{"initial_state":"default"}}}`,
			} {
				files[path] = &fstest.MapFile{Data: []byte(prefix + text)}
			}
			catalog := NewCatalog()
			if err := catalog.AddPack(files); err != nil {
				t.Fatal(err)
			}
			if catalog.blocks["test:block"]["up"] != "test_texture" {
				t.Fatal("block textures not imported")
			}
			if catalog.terrain["test_texture"] != "textures/blocks/test" {
				t.Fatal("terrain textures not imported")
			}
			if _, ok := catalog.geometries["geometry.test"]; !ok {
				t.Fatal("geometry not imported")
			}
			if catalog.actors["test:actor"].Identifier != "test:actor" {
				t.Fatal("actor not imported")
			}
			if catalog.renderControllers["controller.render.test"]["geometry"] != "Geometry.default" {
				t.Fatal("render controller not imported")
			}
			if _, ok := catalog.animations["animation.test"]; !ok {
				t.Fatal("animation not imported")
			}
			if _, ok := catalog.animationControllers["controller.animation.test"]; !ok {
				t.Fatal("animation controller not imported")
			}
			if err := catalog.AddBlockDefinition([]byte(prefix + `{"minecraft:block":{"description":{"identifier":"test:defined"},"components":{}}}`)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPackJSONStillRejectsMalformedController(t *testing.T) {
	for _, prefix := range []string{"", "\xef\xbb\xbf"} {
		catalog := NewCatalog()
		if err := catalog.AddPack(fstest.MapFS{"render_controllers/broken.json": &fstest.MapFile{Data: []byte(prefix + `{"render_controllers": ]}`)}}); err == nil {
			t.Fatal("malformed JSON accepted")
		}
	}
}

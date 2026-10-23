package worlds

import (
	"archive/zip"
	"context"
	"github.com/Daniel-Ric/CraftBox/handlers/worlds/worldstate"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sirupsen/logrus"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPlayerExportUsesVisitSnapshot(t *testing.T) {
	snapshot := proxy.Player{Position: mgl32.Vec3{100, 70, 200}, Pitch: 10, Yaw: 20}
	w := &worldsHandler{session: &proxy.Session{Player: proxy.Player{Position: mgl32.Vec3{999, 80, 999}, Pitch: 90, Yaw: 90}}}
	data := w.playerData(snapshot)
	if !reflect.DeepEqual(data["Pos"], []float32{100, 70, 200}) || !reflect.DeepEqual(data["Rotation"], []float32{10, 20}) {
		t.Fatalf("export used another visit: %+v", data)
	}
}

func TestVisitsProduceSeparateWorldArchivesAndImages(t *testing.T) {
	messages.SetEventHandler(func(any) error { return nil })
	t.Cleanup(func() { messages.SetEventHandler(func(any) error { return nil }) })
	folder := t.TempDir()
	var paths []string
	for _, c := range []color.RGBA{{R: 255, A: 255}, {B: 255, A: 255}} {
		visit, err := worldstate.New(context.Background(), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		visit.Open("Lobby", filepath.Join(folder, "Lobby--"+visit.ID))
		if err := os.MkdirAll(visit.Folder, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(visit.Folder, "levelname.txt"), []byte(visit.ID), 0600); err != nil {
			t.Fatal(err)
		}
		w := &worldsHandler{log: logrus.NewEntry(logrus.New()), session: &proxy.Session{}}
		w.mapUI = NewMapUI(w)
		close(w.mapUI.haveColors)
		tile := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				tile.SetRGBA(x, y, c)
			}
		}
		w.mapUI.renderedChunks[protocol.ChunkPos{}] = tile
		if err := w.saveMapImage(visit); err != nil {
			t.Fatal(err)
		}
		if err := w.writeWorldArchive(visit, 0, 0); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(visit.ImagePath)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := color.RGBAModel.Convert(img.At(0, 0)); got != c {
			t.Fatalf("mixed image: %v", got)
		}
		archive, err := zip.OpenReader(visit.Folder + ".mcworld")
		if err != nil {
			t.Fatal(err)
		}
		if len(archive.File) != 1 {
			t.Fatal("missing world file")
		}
		archive.Close()
		paths = append(paths, visit.Folder)
	}
	if paths[0] == paths[1] {
		t.Fatal("same-name visits reused a path")
	}
	for _, path := range paths {
		for _, suffix := range []string{".png", ".mcworld"} {
			if _, err := os.Stat(path + suffix); err != nil {
				t.Fatal(err)
			}
		}
	}
}

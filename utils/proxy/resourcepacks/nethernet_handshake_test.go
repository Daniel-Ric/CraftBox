package resourcepacks

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
)

func TestNetherNetProxyResourceHandshake(t *testing.T) {
	t.Run("without packs", func(t *testing.T) { testNetherNetProxyResourceHandshake(t, nil, &testDiskCache{dir: t.TempDir()}) })
	t.Run("multi-chunk pack", func(t *testing.T) {
		var archive bytes.Buffer
		z := zip.NewWriter(&archive)
		manifest, err := z.Create("manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.WriteString(manifest, `{"format_version":2,"header":{"name":"Proxy fixture","description":"test","uuid":"85149a83-050c-4113-8f6e-739c11a64b42","version":[1,0,0]},"modules":[{"type":"resources","uuid":"804c5aa2-37f0-4c3d-ab3e-77ceaceab17b","version":[1,0,0]}]}`)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := z.CreateHeader(&zip.FileHeader{Name: "textures/test.bin", Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = payload.Write(bytes.Repeat([]byte("nethernet pack payload\n"), 20000)); err != nil {
			t.Fatal(err)
		}
		if err = z.Close(); err != nil {
			t.Fatal(err)
		}
		pack, err := resource.Read(bytes.NewReader(archive.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		cache := &testDiskCache{dir: t.TempDir()}
		t.Run("cold cache", func(t *testing.T) { testNetherNetProxyResourceHandshake(t, pack, cache) })
		t.Run("warm cache", func(t *testing.T) { testNetherNetProxyResourceHandshake(t, pack, cache) })
		t.Run("replay cache", func(t *testing.T) {
			var capture bytes.Buffer
			z := zip.NewWriter(&capture)
			f, err := z.CreateHeader(&zip.FileHeader{Name: "packcache/" + pack.UUID().String() + "_" + pack.Version() + ".zip", Method: zip.Store})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pack.WriteTo(f); err != nil {
				t.Fatal(err)
			}
			if err = z.Close(); err != nil {
				t.Fatal(err)
			}
			replay := NewReplayCache()
			if err = replay.ReadFrom(bytes.NewReader(capture.Bytes()), int64(capture.Len())); err != nil {
				t.Fatal(err)
			}
			testNetherNetProxyResourceHandshake(t, pack, replay)
		})
	})
}

func testNetherNetProxyResourceHandshake(t *testing.T, pack *resource.Pack, cache PackCache) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var packs []*resource.Pack
	if pack != nil {
		packs = []*resource.Pack{pack}
	}
	backend, err := (minecraft.ListenConfig{AuthenticationDisabled: true, ResourcePacks: packs}).ListenNetwork(minecraft.DirectNetherNet{DisableLANDiscovery: true}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	r := NewResourcePackHandler(ctx, nil)
	r.cache = cache
	r.OnFinishedPack = func(*resource.Pack) error { return nil }
	frontendReady := make(chan struct{})
	frontend, err := (minecraft.ListenConfig{
		AuthenticationDisabled: true,
		EarlyConnHandler:       func(c *minecraft.Conn) { r.SetClient(c); c.ResourcePackHandler = r; close(frontendReady) },
	}).ListenNetwork(minecraft.DirectNetherNet{DisableLANDiscovery: true}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer frontend.Close()
	backendSpawn := make(chan error, 1)
	go func() {
		c, err := backend.AcceptMinecraft()
		if err == nil {
			defer c.Close()
			err = c.StartGameContext(ctx, minecraft.GameData{WorldName: "NetherNet proxy test", EntityUniqueID: 1, EntityRuntimeID: 1})
			if err == nil {
				<-ctx.Done()
			}
		}
		backendSpawn <- err
	}()
	upstream := make(chan *minecraft.Conn, 1)
	errors := make(chan error, 2)
	go func() {
		select {
		case <-frontendReady:
		case <-ctx.Done():
			errors <- ctx.Err()
			return
		}
		c, err := (minecraft.Dialer{EarlyConnHandler: func(c *minecraft.Conn) { r.SetServer(c); c.ResourcePackHandler = r }}).DialContext(ctx, "nethernet", backend.Addr().String())
		if err != nil {
			errors <- err
			return
		}
		upstream <- c
	}()
	frontendSpawn := make(chan error, 1)
	go func() {
		c, err := frontend.AcceptMinecraft()
		if err != nil {
			frontendSpawn <- err
			return
		}
		defer c.Close()
		select {
		case remote := <-upstream:
			defer remote.Close()
			err = c.StartGameContext(ctx, remote.GameData())
		case err = <-errors:
		case <-ctx.Done():
			err = ctx.Err()
		}
		frontendSpawn <- err
	}()
	client, err := (minecraft.Dialer{}).DialContext(ctx, "nethernet", frontend.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.GameData().WorldName != "NetherNet proxy test" {
		t.Fatalf("world = %q", client.GameData().WorldName)
	}
	if pack != nil {
		received := client.ResourcePacks()
		if len(received) != 1 || received[0].Checksum() != pack.Checksum() {
			t.Fatalf("pack changed in proxy: %#v", received)
		}
		if !cache.Has(pack.UUID(), pack.Version()) {
			t.Fatal("pack was not cached")
		}
	}
	if err := <-frontendSpawn; err != nil {
		t.Fatal(err)
	}
}

type testDiskCache struct{ dir string }

func (c *testDiskCache) name(id uuid.UUID, version string) string {
	return filepath.Join(c.dir, id.String()+"_"+version+".zip")
}
func (c *testDiskCache) Has(id uuid.UUID, version string) bool {
	_, err := os.Stat(c.name(id, version))
	return err == nil
}
func (c *testDiskCache) Get(id uuid.UUID, version string) (*resource.Pack, error) {
	return resource.ReadPath(c.name(id, version))
}
func (c *testDiskCache) Create(id uuid.UUID, version string) (*closeMoveWriter, error) {
	name := c.name(id, version)
	f, err := utils.CreateShared(name + ".tmp")
	if err != nil {
		return nil, err
	}
	return &closeMoveWriter{File: f, FinalName: name}, nil
}

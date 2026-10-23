package resourcepacks

import (
	"context"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderPacksFollowServerStackInsteadOfDownloadOrder(t *testing.T) {
	packs := []*resource.Pack{}
	for i := 1; i <= 3; i++ {
		dir := t.TempDir()
		manifest := fmt.Sprintf(`{"format_version":2,"header":{"name":"stack","description":"test","uuid":"85149a83-050c-4113-8f6e-%012d","version":[1,0,0]},"modules":[{"type":"resources","uuid":"804c5aa2-37f0-4c3d-ab3e-77ceaceab17b","version":[1,0,0]}]}`, i)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0600); err != nil {
			t.Fatal(err)
		}
		p, err := resource.ReadPath(dir)
		if err != nil {
			t.Fatal(err)
		}
		packs = append(packs, p)
	}
	stack := &packet.ResourcePackStack{TexturePacks: []protocol.StackResourcePack{{UUID: packs[1].UUID().String(), Version: packs[1].Version()}, {UUID: packs[0].UUID().String(), Version: packs[0].Version()}}}
	got := packsInStackOrder(packs, stack)
	if len(got) != 2 || got[0] != packs[1] || got[1] != packs[0] {
		t.Fatal("download order or inactive pack affects rendering", got)
	}
	got[0] = nil
	if packs[0] == nil || packs[1] == nil {
		t.Fatal("mutated downloaded pack slice")
	}
	done := make(chan struct{})
	close(done)
	handler := &ResourcePackHandler{ctx: context.Background(), Server: stackTestConn{}, receivedRemoteStack: done, resourcePacks: packs, remoteStack: stack}
	if all := handler.ResourcePacks(); len(all) != 3 || all[2] != packs[2] {
		t.Fatal("capture lost inactive advertised pack")
	}
	if active := handler.ActiveResourcePacks(); len(active) != 2 || active[0] != packs[1] || active[1] != packs[0] {
		t.Fatal("renderer lost stack order")
	}
}

type stackTestConn struct{ minecraft.IConn }

func (stackTestConn) Context() context.Context { return context.Background() }

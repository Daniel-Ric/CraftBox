package scene

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"math"
	"testing"
)

func pageFixture(t *testing.T) *Store {
	t.Helper()
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	s := newTestStore(t, reg, NewCatalog())
	for x := int32(0); x < 10; x++ {
		c := chunk.New(reg, cube.Range{0, 15})
		c.SetBlock(0, 1, 0, 0, stone)
		s.Put(world.ChunkPos{x, 0}, c)
	}
	return s
}
func TestPagesStartNearPlayerAndAcknowledgeOnlyCompleteCycle(t *testing.T) {
	s := pageFixture(t)
	request := PageRequest{Limit: 2, Focus: [2]int32{8, 0}}
	seen := map[int32]bool{}
	for pages := 0; pages < 10; pages++ {
		page, err := s.Page(context.Background(), request, nil)
		if err != nil {
			t.Fatal(err)
		}
		if pages == 0 && (!page.Reset || page.Chunks[0].X != 8 || page.Complete) {
			t.Fatal("first nearby geometry must arrive before the full world")
		}
		if pages == 0 && len(page.Patches) == 0 {
			t.Fatal("first page must supply texture patches")
		}
		request.AtlasRevision = page.AtlasRevision
		if len(page.Chunks) > 2 {
			t.Fatal("page exceeded batch size")
		}
		for _, c := range page.Chunks {
			if seen[c.X] {
				t.Fatal("duplicate chunk")
			}
			seen[c.X] = true
		}
		request.Target = page.Sequence
		request.Cursor = page.Next
		if page.Complete {
			if len(seen) != 10 {
				t.Fatalf("lost chunks: %d", len(seen))
			}
			break
		}
	}
	request.After = request.Target
	request.Target = 0
	request.Cursor = nil
	unchanged, err := s.Page(context.Background(), request, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged.Complete || len(unchanged.Chunks) != 0 || len(unchanged.Patches) != 0 {
		t.Fatal("unchanged scene resent geometry/atlas")
	}
}
func TestPageCancellationWhileRendererBusy(t *testing.T) {
	s := pageFixture(t)
	if err := s.acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Page(ctx, PageRequest{}, nil)
	s.release()
	if err != context.Canceled {
		t.Fatalf("canceled request waited for meshing: %v", err)
	}
}

func TestPageFutureTargetDiscardsStaleCursor(t *testing.T) {
	s := pageFixture(t)
	cursor := [2]int32{999, 999}
	p, err := s.Page(context.Background(), PageRequest{Target: 99999, Cursor: &cursor, Limit: 1}, nil)
	if err != nil || !p.Reset || len(p.Chunks) != 1 {
		t.Fatalf("future target skipped chunks: %+v %v", p, err)
	}
}

func TestPageResendsWritesBeforeCursorOnNextCycle(t *testing.T) {
	s := pageFixture(t)
	r := PageRequest{Limit: 1}
	p, err := s.Page(context.Background(), r, nil)
	if err != nil {
		t.Fatal(err)
	}
	target := p.Sequence
	reg := world.DefaultBlockRegistry
	empty := chunk.New(reg, cube.Range{0, 15})
	s.Put(world.ChunkPos{0, 0}, empty)
	for !p.Complete {
		r.Target = target
		r.Cursor = p.Next
		p, err = s.Page(context.Background(), r, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	delta, err := s.Page(context.Background(), PageRequest{After: target}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range delta.Chunks {
		if c.X == 0 && len(c.Parts) == 0 {
			return
		}
	}
	t.Fatal("update before cursor was lost")
}
func TestPackedMeshHasExactFloatPositionsUVsAndCompactNormals(t *testing.T) {
	s := pageFixture(t)
	full, err := s.Snapshot(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := packChunk(full.Chunks[0]).Parts[0]
	raw, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil {
		t.Fatal(err)
	}
	source := full.Chunks[0].Parts[0]
	if p.Vertices != len(source.Positions)/3 || p.Indices != len(source.Indices) {
		t.Fatal("bad packed counts")
	}
	normalOffset := p.Vertices * 12
	uvOffset := (normalOffset + p.Vertices*3 + 3) &^ 3
	indexOffset := uvOffset + p.Vertices*8
	if len(raw) != indexOffset+p.Indices*4 {
		t.Fatal("packed buffer length")
	}
	for i, v := range source.Positions {
		if math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:])) != v {
			t.Fatal("position changed")
		}
	}
	for i, v := range source.Normals {
		if float32(int8(raw[normalOffset+i]))/127 != v {
			t.Fatal("normal changed")
		}
	}
	for i, v := range source.UVs {
		if math.Float32frombits(binary.LittleEndian.Uint32(raw[uvOffset+i*4:])) != v {
			t.Fatal("UV changed")
		}
	}
	if len(raw) > meshBytes(full.Chunks[0]) {
		t.Fatal("compact transport grew geometry")
	}
}

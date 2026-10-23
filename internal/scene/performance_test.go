package scene

import (
	"context"
	"encoding/json"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"testing"
	"time"
)

func benchmarkTerrain(b *testing.B) *Store {
	b.Helper()
	reg := world.DefaultBlockRegistry
	reg.Finalize()
	stone, _ := reg.StateToRuntimeID("minecraft:stone", map[string]any{})
	s := NewStore(reg, NewCatalog())
	b.Cleanup(s.Close)
	for cx := int32(0); cx < 8; cx++ {
		for cz := int32(0); cz < 8; cz++ {
			c := chunk.New(reg, cube.Range{-64, 319})
			for x := uint8(0); x < 16; x++ {
				for z := uint8(0); z < 16; z++ {
					height := 40 + (int(x)+int(z)+int(cx)+int(cz))%12
					for y := -64; y < height; y++ {
						c.SetBlock(x, int16(y), z, 0, stone)
					}
				}
			}
			s.Put(world.ChunkPos{cx, cz}, c)
		}
	}
	return s
}

func BenchmarkSceneProgressive(b *testing.B) {
	s := benchmarkTerrain(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Invalidate()
		request := PageRequest{Focus: [2]int32{4, 4}, Limit: 8}
		start := time.Now()
		bytes, pages := 0, 0
		for {
			page, err := s.Page(context.Background(), request, nil)
			if err != nil {
				b.Fatal(err)
			}
			data, err := json.Marshal(page)
			if err != nil {
				b.Fatal(err)
			}
			pages++
			bytes += len(data)
			if pages == 1 {
				b.ReportMetric(float64(time.Since(start).Microseconds())/1000, "first-page-ms")
			}
			if page.Complete {
				break
			}
			request.Target = page.Sequence
			request.Cursor = page.Next
			request.AtlasRevision = page.AtlasRevision
		}
		b.ReportMetric(float64(bytes), "json-bytes")
		b.ReportMetric(float64(pages), "pages")
	}
}

func BenchmarkSceneCold(b *testing.B) {
	s := benchmarkTerrain(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Invalidate()
		snapshot, err := s.Snapshot(0, nil)
		if err != nil {
			b.Fatal(err)
		}
		data, err := json.Marshal(snapshot)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(data)), "json-bytes")
	}
}

package scene

import (
	"encoding/json"
	"github.com/Daniel-Ric/CraftBox/utils/proxy/pcap2"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"net"
	"os"
	"testing"
	"time"
)

func TestInspectActorPackets(t *testing.T) {
	file := os.Getenv("CRAFT_BOX_CAPTURE_PACKETS")
	if file == "" {
		t.Skip("local capture")
	}
	f, e := os.Open(file)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	r, e := pcap2.NewPcap2Reader(f)
	if e != nil {
		t.Fatal(e)
	}
	r.PacketFunc = func(_ packet.Header, _ []byte, _ net.Addr, _ net.Addr, _ time.Time) {}
	types := map[uint64]string{}
	names := map[string]int{}
	links := 0
	owners := 0
	players := 0
	count := 0
	actors := map[uint64]Actor{}
	metadata := map[uint64]protocol.EntityMetadata{}
	for {
		pk, _, _, err := r.ReadPacket(false)
		if err != nil {
			t.Logf("end: %v", err)
			break
		}
		count++
		var md protocol.EntityMetadata
		var id uint64
		switch p := pk.(type) {
		case *packet.AddActor:
			types[p.EntityRuntimeID] = p.EntityType
			id = p.EntityRuntimeID
			md = p.EntityMetadata
			links += len(p.EntityLinks)
			actors[id] = Actor{ID: p.EntityUniqueID, Type: p.EntityType, Position: [3]float32(p.Position), Yaw: p.Yaw, Scale: 1}
			if metadata[id] == nil {
				metadata[id] = protocol.EntityMetadata{}
			}
		case *packet.SetActorData:
			id = p.EntityRuntimeID
			md = p.EntityMetadata
		case *packet.AddPlayer:
			types[p.EntityRuntimeID] = "minecraft:player"
			players++
		case *packet.SetActorLink:
			links++
		}
		if md != nil {
			if metadata[id] == nil {
				metadata[id] = protocol.EntityMetadata{}
			}
			for k, v := range md {
				metadata[id][k] = v
			}
		}
		if v, ok := md[protocol.EntityDataKeyOwner].(int64); ok && v != 0 && v != -1 {
			owners++
		}
		if name, ok := md[protocol.EntityDataKeyName].(string); ok && name != "" && types[id] != "minecraft:player" {
			names[types[id]+" "+name]++
		}
	}
	raw, _ := json.MarshalIndent(names, "", "  ")
	_ = os.WriteFile("../../tmp/entity-corrections/packet-names.json", raw, 0600)
	t.Logf("packets=%d players=%d links=%d owners=%d textvariants=%d", count, players, links, owners, len(names))
	visuals := []Actor{}
	holograms := 0
	for id, a := range actors {
		if types[id] == "minecraft:player" {
			continue
		}
		a = ActorFromMetadata(a.ID, a.Type, a.Position, a.Yaw, metadata[id], nil)
		if a.Name != "" && a.NameVisible && actorAllowed(a) {
			holograms++
			if holograms < 4 {
				t.Logf("Hologram type=%s scale=%g height=%g invisible=%v", a.Type, a.Scale, a.Height, a.Invisible)
			}
		}
		visuals = append(visuals, a)
	}
	raw, _ = json.Marshal(visuals)
	_ = os.WriteFile("../../tmp/entity-corrections/packet-actors.json", raw, 0600)
	t.Logf("retained holograms=%d", holograms)
}

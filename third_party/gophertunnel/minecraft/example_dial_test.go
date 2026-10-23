package minecraft_test

import (
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func ExampleDial() {

	dialer := minecraft.Dialer{
		TokenSource: auth.TokenSource,
	}

	address := "mco.mineplex.com:19132"
	conn, err := dialer.Dial("nethernet", address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if err := conn.DoSpawn(); err != nil {
		panic(err)
	}

	for {

		pk, err := conn.ReadPacket()
		if err != nil {
			break
		}

		switch p := pk.(type) {
		case *packet.Emote:
			fmt.Printf("Emote packet received: %v\n", p.EmoteID)
		case *packet.MovePlayer:
			fmt.Printf("Player %v moved to %v\n", p.EntityRuntimeID, p.Position)
		}

		p := &packet.RequestChunkRadius{ChunkRadius: 32, MaxChunkRadius: 32}
		if err := conn.WritePacket(p); err != nil {
			break
		}
	}
}

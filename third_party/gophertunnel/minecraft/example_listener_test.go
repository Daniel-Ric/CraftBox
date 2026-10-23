package minecraft_test

import (
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func ExampleListen() {

	name := "MOTD of this server"
	cfg := minecraft.ListenConfig{
		StatusProvider: minecraft.NewStatusProvider(name, "Gophertunnel"),
	}

	address := ":19132"
	listener, err := cfg.Listen("nethernet", address)
	if err != nil {
		panic(err)
	}

	for {

		c, err := listener.Accept()
		if err != nil {
			return
		}
		conn := c.(*minecraft.Conn)

		go func() {

			defer conn.Close()

			worldData := minecraft.GameData{}
			if err := conn.StartGame(worldData); err != nil {
				return
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

				p := &packet.ChunkRadiusUpdated{ChunkRadius: 32}
				if err := conn.WritePacket(p); err != nil {
					break
				}
			}
		}()
	}
}

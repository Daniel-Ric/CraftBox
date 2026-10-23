package minecraft

import (
	"errors"
	"sync"

	"github.com/df-mc/go-nethernet/endpoint"
)

func ExampleNetherNet() {

	server, err := endpoint.ServeTLS(":19132", "/path/to/cert-file", "/path/to/key-file")
	if err != nil {
		panic(err)
	}
	defer server.Close()

	n := NetherNet{Signaling: server}

	var cfg ListenConfig
	l, err := cfg.ListenNetwork(n, server.NetworkID())
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go handleConn(conn.(*Conn), l)
	}
}

func handleConn(conn *Conn, listener *Listener) {
	client := endpoint.NewClient()
	serverConn, err := Dialer{
		ClientData: conn.ClientData(),
	}.DialContextNetwork(conn.Context(), NetherNet{Signaling: client}, "http://localhost:19133")
	if err != nil {
		panic(err)
	}
	var g sync.WaitGroup
	g.Add(2)
	go func() {
		if err := conn.StartGame(serverConn.GameData()); err != nil {
			panic(err)
		}
		g.Done()
	}()
	go func() {
		if err := serverConn.DoSpawn(); err != nil {
			panic(err)
		}
		g.Done()
	}()
	g.Wait()

	go func() {
		defer listener.Disconnect(conn, "connection lost")
		defer serverConn.Close()
		for {
			pk, err := conn.ReadPacket()
			if err != nil {
				return
			}
			if err := serverConn.WritePacket(pk); err != nil {
				var disc DisconnectError
				if ok := errors.As(err, &disc); ok {
					_ = listener.Disconnect(conn, disc.Error())
				}
				return
			}
		}
	}()
	go func() {
		defer serverConn.Close()
		defer listener.Disconnect(conn, "connection lost")
		for {
			pk, err := serverConn.ReadPacket()
			if err != nil {
				var disc DisconnectError
				if ok := errors.As(err, &disc); ok {
					_ = listener.Disconnect(conn, disc.Error())
				}
				return
			}
			if err := conn.WritePacket(pk); err != nil {
				return
			}
		}
	}()
}

package transport

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestDirectAddress(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"example.com", "http://example.com:19132"},
		{"127.0.0.1:19133", "http://127.0.0.1:19133"},
		{"[::1]:19132", "http://[::1]:19132"},
		{"https://example.com:443", "https://example.com:443"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := DirectAddress(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("DirectAddress(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
		})
	}
}

func TestDirectAddressRejectsInvalidEndpoints(t *testing.T) {
	for _, input := range []string{"", "ftp://example.com:19132", "http://example.com:19132/path", "host:invalid", "http://user:pass@host:19132"} {
		if _, err := DirectAddress(input); err == nil {
			t.Errorf("accepted invalid endpoint %q", input)
		}
	}
}

func TestDirectNetherNetPacketExchange(t *testing.T) {
	server, err := ListenDirect("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result := make(chan error, 1)
	received := make(chan struct{})
	go func() {
		conn, err := server.Accept()
		if err != nil {
			result <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		data := make([]byte, 32768)
		n, err := conn.Read(data)
		if err == nil {
			_, err = conn.Write(data[:n])
		}
		result <- err
		select {
		case <-received:
		case <-ctx.Done():
		}
	}()
	conn, err := DialDirect(ctx, server.Address())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	payload := bytes.Repeat([]byte("world-chunk"), 1000)
	if _, err := conn.Write(payload); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reply, payload) {
		t.Fatal("NetherNet changed chunk payload")
	}
	close(received)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	address := server.Address()
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := DialDirect(ctx, address); err == nil {
		t.Fatal("closed endpoint still accepts connections")
	}
}

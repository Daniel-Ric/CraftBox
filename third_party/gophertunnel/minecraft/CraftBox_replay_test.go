package minecraft

import (
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCapturedPacketDecode(t *testing.T) {
	var data bytes.Buffer
	header := packet.Header{PacketID: packet.IDText}
	if err := header.Write(&data); err != nil {
		t.Fatal(err)
	}
	text := &packet.Text{TextType: packet.TextTypeRaw, Message: "captured-nethernet-packet"}
	text.Marshal(DefaultProtocol.NewWriter(&data, 0))
	captured, err := ParseData(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	payload := bytes.Clone(captured.Payload.Bytes())
	for i := 0; i < 2; i++ {
		decoded, err := captured.Decode(DefaultProtocol.Packets(false), DefaultProtocol, nil, false, true, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(decoded) != 1 {
			t.Fatalf("decoded %d packets", len(decoded))
		}
		got, ok := decoded[0].(*packet.Text)
		if !ok || got.Message != "captured-nethernet-packet" {
			t.Fatalf("decoded %#v", decoded)
		}
		if !bytes.Equal(captured.Payload.Bytes(), payload) {
			t.Fatal("decode changed captured payload")
		}
	}
	if _, err := ParseData([]byte{0x80}); err == nil {
		t.Fatal("accepted truncated header")
	}
}

func TestDecodedPacketCanBeRetained(t *testing.T) {
	var data bytes.Buffer
	header := packet.Header{PacketID: packet.IDNetworkStackLatency}
	if err := header.Write(&data); err != nil {
		t.Fatal(err)
	}
	(&packet.NetworkStackLatency{Timestamp: 1234}).Marshal(DefaultProtocol.NewWriter(&data, 0))
	conn := &Conn{proto: DefaultProtocol, pool: DefaultProtocol.Packets(false)}
	parsed, err := parseData(data.Bytes(), conn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsed.decode(conn); err != nil {
		t.Fatal(err)
	}
	retained := parsed.ensureOwned()
	clear(data.Bytes())
	decoded, err := retained.decode(conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 {
		t.Fatalf("decoded %d packets", len(decoded))
	}
	got, ok := decoded[0].(*packet.NetworkStackLatency)
	if !ok || got.Timestamp != 1234 {
		t.Fatalf("decoded %#v", decoded)
	}
}

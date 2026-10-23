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
	if _, err := ParseData([]byte{0x80}); err == nil {
		t.Fatal("accepted truncated header")
	}
}

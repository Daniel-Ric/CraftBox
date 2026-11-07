package proxy

import (
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Independent wire fixtures: mandatory spline fields and the retail protocol's
// Start=0 video action. These catch layouts that can round-trip themselves but
// disagree with the actual wire format.
func TestProtocol2225MandatoryCameraSplineFields(t *testing.T) {
	wire := []byte{0, 0, 128, 63, 1, 0, 0, 0, 4, 'g', 'a', 't', 'e', 1}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("valid spline wire was rejected: %v", p)
		}
	}()
	buf := bytes.NewBuffer(wire)
	var instruction protocol.CameraSplineInstruction
	instruction.Marshal(protocol.NewReader(buf, 0, true))
	if buf.Len() != 0 {
		t.Fatalf("unread spline bytes: %d", buf.Len())
	}
	var encoded bytes.Buffer
	instruction.Marshal(protocol.NewWriter(&encoded, 0))
	if !bytes.Equal(encoded.Bytes(), wire) {
		t.Fatalf("changed spline wire: %x", encoded.Bytes())
	}
}

func TestProtocol2225VideoCaptureStartWire(t *testing.T) {
	var buf bytes.Buffer
	(&packet.PlayerVideoCapture{Action: 0, FrameRate: 20, FilePrefix: "cap"}).Marshal(protocol.NewWriter(&buf, 0))
	want := []byte{0, 20, 0, 0, 0, 3, 'c', 'a', 'p'}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("start packet=%x want=%x", buf.Bytes(), want)
	}
}

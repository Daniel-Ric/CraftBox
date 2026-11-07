package leveldat

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestMalformedLevelDatHeaderReturnsErrorWithoutPanic(t *testing.T) {
	for _, hdr := range []header{{Version, -1}, {Version, 0}, {Version + 1, 1}} {
		var data bytes.Buffer
		binary.Write(&data, binary.LittleEndian, hdr)
		data.WriteByte(0)
		if _, err := Read(&data); err == nil {
			t.Fatalf("accepted invalid header %#v", hdr)
		}
	}
}

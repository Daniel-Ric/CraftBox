package nbtconv

import (
	"encoding/binary"
	"image/color"
)


func Int32FromRGBA(x color.RGBA) int32 {
	if x.R == 0 && x.G == 0 && x.B == 0 {


		return int32(-0x1000000)
	}
	return int32(binary.BigEndian.Uint32([]byte{x.A, x.R, x.G, x.B}))
}


func RGBAFromInt32(x int32) color.RGBA {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(x))

	return color.RGBA{A: b[0], R: b[1], G: b[2], B: b[3]}
}

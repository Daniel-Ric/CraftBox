//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

func main() {
	if len(os.Args) != 3 {
		panic("expected input PNG and output ICO")
	}
	input, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	source, err := png.Decode(input)
	input.Close()
	if err != nil {
		panic(err)
	}
	if source.Bounds().Dx() != source.Bounds().Dy() {
		panic("icon must be square")
	}
	sizes := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	images := make([][]byte, len(sizes))
	for i, size := range sizes {
		target := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Src, nil)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, target); err != nil {
			panic(err)
		}
		images[i] = encoded.Bytes()
	}
	header := make([]byte, 6+16*len(sizes))
	binary.LittleEndian.PutUint16(header[2:], 1)
	binary.LittleEndian.PutUint16(header[4:], uint16(len(sizes)))
	offset := len(header)
	for i, size := range sizes {
		entry := header[6+16*i:]
		entry[0], entry[1] = byte(size%256), byte(size%256)
		binary.LittleEndian.PutUint16(entry[4:], 1)
		binary.LittleEndian.PutUint16(entry[6:], 32)
		binary.LittleEndian.PutUint32(entry[8:], uint32(len(images[i])))
		binary.LittleEndian.PutUint32(entry[12:], uint32(offset))
		offset += len(images[i])
	}
	var result bytes.Buffer
	result.Write(header)
	for _, encoded := range images {
		result.Write(encoded)
	}
	if err := os.WriteFile(os.Args[2], result.Bytes(), 0644); err != nil {
		panic(err)
	}
}

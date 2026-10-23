package main

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
)

func entries(data []byte, offset uint32) map[uint32]uint32 {
	if uint64(offset)+16 > uint64(len(data)) {
		panic("invalid resource directory")
	}
	count := int(binary.LittleEndian.Uint16(data[offset+12:])) + int(binary.LittleEndian.Uint16(data[offset+14:]))
	result := map[uint32]uint32{}
	for i := 0; i < count; i++ {
		at := int(offset) + 16 + i*8
		if at+8 > len(data) {
			panic("invalid resource entry")
		}
		result[binary.LittleEndian.Uint32(data[at:])] = binary.LittleEndian.Uint32(data[at+4:])
	}
	return result
}
func main() {
	if len(os.Args) != 2 {
		panic("expected Windows exe path")
	}
	file, err := pe.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer file.Close()
	section := file.Section(".rsrc")
	if section == nil {
		panic("missing embedded Windows resources")
	}
	data, err := section.Data()
	if err != nil {
		panic(err)
	}
	root := entries(data, 0)
	for _, kind := range []uint32{3, 14, 24} {
		if _, ok := root[kind]; !ok {
			panic(fmt.Sprintf("missing resource type %d", kind))
		}
	}
	groups := entries(data, root[14]&0x7fffffff)
	if _, ok := groups[3]; !ok {
		panic("Wails icon group ID 3 missing")
	}
	images := entries(data, root[3]&0x7fffffff)
	if len(images) == 0 {
		panic("empty icon image resources")
	}
	fmt.Printf("PASS %s: Wails icon group 3, %d icon images, application manifest\n", os.Args[1], len(images))
}

package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsResourcesRejectUnsupportedArchitecture(t *testing.T) {
	if err := prepareWindowsResources("invalid"); err == nil {
		t.Fatal("unsupported resource architecture accepted")
	}
}

func TestWailsIconResourceKeepsImageReferences(t *testing.T) {

	data := make([]byte, 132)
	binary.LittleEndian.PutUint16(data[2:4], 1)
	copy(data[20:28], ".rsrc")
	binary.LittleEndian.PutUint32(data[36:40], 72)
	binary.LittleEndian.PutUint32(data[40:44], 60)
	resources := data[60:]
	binary.LittleEndian.PutUint16(resources[14:16], 1)
	binary.LittleEndian.PutUint32(resources[16:20], 14)
	binary.LittleEndian.PutUint32(resources[20:24], 0x80000018)
	binary.LittleEndian.PutUint16(resources[38:40], 1)
	binary.LittleEndian.PutUint32(resources[40:44], 2)
	binary.LittleEndian.PutUint32(resources[44:48], 0x80000030)
	resources[64] = 7
	file := filepath.Join(t.TempDir(), "icon.syso")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := setWailsIconResourceID(file); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(actual[100:104]) != 3 || actual[124] != 7 {
		t.Fatal("Wails icon ID or image data incorrect")
	}
}

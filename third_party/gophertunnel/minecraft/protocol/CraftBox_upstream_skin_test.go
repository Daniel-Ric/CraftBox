package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestUpstreamSkinDimensionRepairOnRead(t *testing.T) {
	original := Skin{SkinImageWidth: 128, SkinImageHeight: 128, SkinData: make([]byte, 128*128*4)}
	buf := new(bytes.Buffer)
	original.Marshal(NewWriter(buf, 0))
	raw := buf.Bytes()
	// Empty SkinID and resource patch occupy one byte each before the dimensions.
	binary.LittleEndian.PutUint32(raw[2:6], 64)
	binary.LittleEndian.PutUint32(raw[6:10], 64)
	var got Skin
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("upstream-compatible skin repair missing: %v", p)
		}
	}()
	got.Marshal(NewReader(bytes.NewReader(raw), 0, true))
	if got.SkinImageWidth != 128 || got.SkinImageHeight != 128 || len(got.SkinData) != 128*128*4 {
		t.Fatal("skin dimensions were not repaired")
	}
}

func TestUpstreamCapeRepairIsBounded(t *testing.T) {
	skin := Skin{SkinImageWidth: 64, SkinImageHeight: 64, SkinData: make([]byte, 64*64*4), CapeImageWidth: 64, CapeImageHeight: 32}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("missing cape should be padded: %v", p)
		}
	}()
	skin.Marshal(NewWriter(new(bytes.Buffer), 0))
	if len(skin.CapeData) != 64*32*4 {
		t.Fatal("missing cape remains invalid")
	}
}

func TestSkinDimensionsCannotWrapByteCount(t *testing.T) {
	for _, skin := range []Skin{
		{SkinImageWidth: 1 << 31, SkinImageHeight: 1 << 31},
		{CapeImageWidth: 1 << 31, CapeImageHeight: 1 << 31},
		{Animations: []SkinAnimation{{ImageWidth: 1 << 31, ImageHeight: 1 << 31}}},
	} {
		skin.repairDimensions()
		if err := skin.validate(); err == nil {
			t.Fatal("overflowing dimensions accepted empty image")
		}
	}
}

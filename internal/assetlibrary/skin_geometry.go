package assetlibrary

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"strings"
)

func capturedSkinType(meta map[string]any, slots ...map[string]string) string {
	name, _ := meta["player"].(string)
	if len(name) > 16 && strings.HasSuffix(strings.ToLower(name), " back bling") {
		return "display"
	}
	if strings.Contains(name, " ") {
		for _, equipment := range slots {
			for _, id := range equipment {
				if !strings.HasPrefix(id, "hivehats:") && !strings.HasPrefix(id, "hivebackbling:") {
					continue
				}
				_, label, _ := strings.Cut(id, ":")
				label = strings.ReplaceAll(label, "_", " ")
				if label == strings.ToLower(name) || strings.HasPrefix(label, strings.ToLower(name)+" ") {
					return "display"
				}
			}
		}
	}
	return "player"
}

func declaredSkinArms(meta map[string]any) string {
	for _, key := range []string{"armModel", "armSize", "ArmSize"} {
		value, _ := meta[key].(string)
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "wide", "slim":
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}

func skinPatchSlim(meta map[string]any) bool {
	if declaredSkinArms(meta) == "slim" {
		return true
	}
	if value, _ := meta["geometryId"].(string); strings.EqualFold(value, "geometry.humanoid.customSlim") {
		return true
	}
	value, _ := meta["resourcePatch"].(string)
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		raw = []byte(value)
	}
	var patch struct {
		Geometry struct {
			Default string `json:"default"`
		} `json:"geometry"`
	}
	return json.Unmarshal(raw, &patch) == nil && strings.Contains(strings.ToLower(patch.Geometry.Default), "slim")
}

// Only infer unannotated standard PNGs. Both unused Alex UV strips must be
// completely transparent and both arm fronts must contain substantial texture.
// Explicit metadata and custom geometry take precedence at the call site.
func skinTextureSlim(data []byte) bool {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != config.Height || config.Width < 64 || config.Width > 2048 || config.Width%64 != 0 {
		return false
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	scale := config.Width / 64
	for _, r := range [][4]int{{50, 16, 56, 20}, {54, 20, 56, 32}, {42, 48, 48, 52}, {46, 52, 48, 64}} {
		for y := r[1] * scale; y < r[3]*scale; y++ {
			for x := r[0] * scale; x < r[2]*scale; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a != 0 {
					return false
				}
			}
		}
	}
	for _, r := range [][4]int{{44, 20, 47, 32}, {36, 52, 39, 64}} {
		opaque := 0
		for y := r[1] * scale; y < r[3]*scale; y++ {
			for x := r[0] * scale; x < r[2]*scale; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a >= 0x8000 {
					opaque++
				}
			}
		}
		if opaque < (r[2]-r[0])*(r[3]-r[1])*scale*scale/2 {
			return false
		}
	}
	return true
}

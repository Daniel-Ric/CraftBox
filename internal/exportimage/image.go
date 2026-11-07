package exportimage

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"strconv"
	"strings"
)

type Request struct {
	Name       string `json:"name"`
	Format     string `json:"format"`
	Data       string `json:"data"`
	Quality    int    `json:"quality"`
	Background string `json:"background"`
}

func Extension(format string) (string, error) {
	switch format {
	case "png", "webp", "tiff", "bmp":
		return "." + format, nil
	case "jpeg", "jpg":
		return ".jpg", nil
	}
	return "", fmt.Errorf("Unsupported image format %q", format)
}

func Encode(request Request) ([]byte, error) {
	if _, err := Extension(request.Format); err != nil {
		return nil, err
	}
	if request.Quality < 0 || request.Quality > 100 {
		return nil, fmt.Errorf("Image quality must be between 1 and 100")
	}
	quality := request.Quality
	if quality == 0 {
		quality = 90
	}
	background, err := matte(request.Background)
	if err != nil {
		return nil, err
	}
	prefix := "data:image/png;base64,"
	expected := "png"
	if request.Format == "webp" {
		prefix = "data:image/webp;base64,"
		expected = "webp"
	}
	if len(request.Data) > 192<<20 || !strings.HasPrefix(request.Data, prefix) {
		return nil, fmt.Errorf("Invalid %s image or image exceeds 192 MiB", expected)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(request.Data, prefix))
	if err != nil {
		return nil, fmt.Errorf("Invalid image encoding: %w", err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || format != expected || config.Width < 1 || config.Height < 1 || config.Width > 16384 || config.Height > 16384 || int64(config.Width)*int64(config.Height) > 64<<20 {
		return nil, fmt.Errorf("Image exceeds the 64 MP / 16384-pixel budget or has an invalid header")
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("Invalid image pixels: %w", err)
	}
	if request.Format == "png" || request.Format == "webp" {
		return raw, nil
	}
	if request.Format == "jpeg" || request.Format == "jpg" || request.Format == "bmp" {
		flat := image.NewRGBA(decoded.Bounds())
		draw.Draw(flat, flat.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		draw.Draw(flat, flat.Bounds(), decoded, decoded.Bounds().Min, draw.Over)
		decoded = flat
	}
	var output bytes.Buffer
	switch request.Format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&output, decoded, &jpeg.Options{Quality: quality})
	case "tiff":
		err = tiff.Encode(&output, decoded, &tiff.Options{Compression: tiff.Deflate})
	case "bmp":
		err = bmp.Encode(&output, decoded)
	}
	if err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func matte(value string) (color.NRGBA, error) {
	if value == "" {
		value = "#ffffff"
	}
	if len(value) != 7 || value[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("Choose a background color in #RRGGBB format")
	}
	rgb, err := strconv.ParseUint(value[1:], 16, 24)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("Invalid background color")
	}
	return color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 255}, nil
}

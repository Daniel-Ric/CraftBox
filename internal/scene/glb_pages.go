package scene

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
)

func WriteGLBPages(ctx context.Context, out io.Writer, directory string, next func(PageRequest) (Page, error), progress func(GLBProgress)) error {
	w, err := NewGLBWriter(ctx, directory)
	if err != nil {
		return err
	}
	defer w.Close()
	request := PageRequest{Limit: 8, TerrainOnly: true}
	var atlas *image.NRGBA
	source := ""
	done := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := next(request)
		if err != nil {
			return err
		}
		if atlas == nil {
			if page.AtlasSize < 16 || page.AtlasSize > 8192 {
				return fmt.Errorf("Invalid GLB atlas size")
			}
			atlas = image.NewNRGBA(image.Rect(0, 0, page.AtlasSize, page.AtlasSize))
			source = page.Source
		} else if page.AtlasSize != atlas.Bounds().Dx() {
			return fmt.Errorf("GLB atlas changed size")
		}
		for _, patch := range page.Patches {
			if patch.Size < 1 || patch.Size > 128 || patch.X < 0 || patch.Y < 0 || patch.X+patch.Size > page.AtlasSize || patch.Y+patch.Size > page.AtlasSize {
				return fmt.Errorf("Invalid GLB atlas patch")
			}
			raw, err := base64.StdEncoding.DecodeString(patch.Data)
			if err != nil || len(raw) != patch.Size*patch.Size*4 {
				return fmt.Errorf("Invalid GLB atlas pixels")
			}
			for y := 0; y < patch.Size; y++ {
				offset := atlas.PixOffset(patch.X, patch.Y+y)
				copy(atlas.Pix[offset:offset+patch.Size*4], raw[y*patch.Size*4:(y+1)*patch.Size*4])
			}
		}
		for _, packed := range page.Chunks {
			ch := ChunkMesh{X: packed.X, Z: packed.Z}
			for _, p := range packed.Parts {
				part, err := unpackGLBPart(p)
				if err != nil {
					return err
				}
				ch.Parts = append(ch.Parts, part)
			}
			if err := w.AddChunk(ch); err != nil {
				return err
			}
			done++
			if progress != nil {
				progress(GLBProgress{Done: done, Total: page.Total, Bytes: w.Bytes(), Stage: "geometry"})
			}
		}
		if page.Complete {
			break
		}
		if page.Next == nil {
			return fmt.Errorf("Incomplete model pages")
		}
		request.Target = page.Sequence
		request.Cursor = page.Next
		request.AtlasRevision = page.AtlasRevision
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, atlas); err != nil {
		return err
	}
	if progress != nil {
		progress(GLBProgress{Done: done, Total: done, Bytes: w.Bytes(), Stage: "writing"})
	}
	return w.Finish(out, encoded.Bytes(), source)
}

func unpackGLBPart(p PackedPart) (Part, error) {
	if p.Vertices < 1 || p.Vertices > 4_000_000 || p.Indices < 0 || p.Indices%3 != 0 {
		return Part{}, fmt.Errorf("Invalid packed GLB geometry size")
	}
	normalOffset := p.Vertices * 12
	uvOffset := (normalOffset + p.Vertices*3 + 3) &^ 3
	indexOffset := uvOffset + p.Vertices*8
	expected := uint64(indexOffset) + uint64(p.Indices)*4
	if expected > 64<<20 || uint64(base64.StdEncoding.DecodedLen(len(p.Data))) < expected || uint64(base64.StdEncoding.DecodedLen(len(p.Data))) > expected+2 {
		return Part{}, fmt.Errorf("Invalid packed GLB geometry length")
	}
	raw, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil || uint64(len(raw)) != expected {
		return Part{}, fmt.Errorf("Incomplete packed GLB geometry")
	}
	out := Part{Mode: p.Mode, Positions: make([]float32, p.Vertices*3), Normals: make([]float32, p.Vertices*3), UVs: make([]float32, p.Vertices*2), Indices: make([]uint32, p.Indices)}
	for i := range out.Positions {
		out.Positions[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		out.Normals[i] = float32(int8(raw[normalOffset+i])) / 127
	}
	for i := range out.UVs {
		out.UVs[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[uvOffset+i*4:]))
	}
	for i := range out.Indices {
		out.Indices[i] = binary.LittleEndian.Uint32(raw[indexOffset+i*4:])
	}
	return out, nil
}

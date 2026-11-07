package messages

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CapturePoint keeps original world coordinates, orientation and recording time.
type CapturePoint struct {
	ID       string     `json:"id"`
	Kind     string     `json:"kind"`
	Name     string     `json:"name"`
	Icon     string     `json:"icon"`
	Position [3]float32 `json:"position"`
	Yaw      float32    `json:"yaw"`
	Pitch    float32    `json:"pitch"`
	Created  time.Time  `json:"created"`
	Elapsed  float64    `json:"elapsed"`
}
type EventCapturePoints struct {
	WorldID string
	Points  []CapturePoint
}

var capturePointsMu sync.Mutex

func CapturePointsInBounds(points []CapturePoint, bounds ChunkBounds) []CapturePoint {
	out := []CapturePoint{}
	for _, point := range points {
		if bounds.Contains(int32(math.Floor(float64(point.Position[0])/16)), int32(math.Floor(float64(point.Position[2])/16))) {
			out = append(out, point)
		}
	}
	return out
}

func ReadCapturePoints(folder string) []CapturePoint {
	points, _ := readCapturePoints(folder)
	return points
}
func readCapturePoints(folder string) ([]CapturePoint, error) {
	if folder == "" {
		return nil, nil
	}
	f, e := os.Open(filepath.Join(folder, "craftbox-capture.json"))
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 2<<20 {
		return nil, fmt.Errorf("Capture notes exceed the size limit")
	}
	var points []CapturePoint
	if json.Unmarshal(b, &points) != nil || len(points) > 512 {
		return nil, fmt.Errorf("Capture notes are invalid; the existing file was retained")
	}
	ids := map[string]bool{}
	for _, p := range points {
		if p.ID == "" || ids[p.ID] || len(p.Name) > 400 || len(p.Icon) > 32 {
			return nil, fmt.Errorf("Invalid capture point")
		}
		ids[p.ID] = true
		for _, v := range p.Position {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || math.Abs(float64(v)) > 30_000_000 {
				return nil, fmt.Errorf("Invalid capture point coordinates")
			}
		}
	}
	return points, nil
}
func AddCapturePoint(folder string, point CapturePoint) ([]CapturePoint, error) {
	capturePointsMu.Lock()
	defer capturePointsMu.Unlock()
	if folder == "" {
		return nil, fmt.Errorf("Capture folder is not ready")
	}
	points, e := readCapturePoints(folder)
	if e != nil {
		return nil, e
	}
	if len(points) >= 512 {
		return nil, fmt.Errorf("This capture already contains 512 markers, notes and cameras")
	}
	points = append(points, point)
	b, e := json.MarshalIndent(points, "", "  ")
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(folder, 0755); e != nil {
		return nil, e
	}
	f, e := os.CreateTemp(folder, ".craftbox-capture-*")
	if e != nil {
		return nil, e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return nil, e
	}
	if e = f.Close(); e != nil {
		return nil, e
	}
	if e = os.Rename(name, filepath.Join(folder, "craftbox-capture.json")); e != nil {
		return nil, e
	}
	return points, nil
}

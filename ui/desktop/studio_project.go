package desktop

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"io"
	"os"
	"path/filepath"
)

const maxProjectJSON = 96 << 20
const maxProjectPage = 64 << 20

func (a *App) shouldPreventStudioClose() bool {
	a.mu.Lock()
	dirty, approved, emit := a.studioProjectDirty || a.documentCloseManaged, a.studioCloseApproved, a.emit
	a.studioCloseApproved = false
	a.mu.Unlock()
	if dirty && !approved {
		if emit != nil {
			emit("studio:close-requested", true)
		}
		return true
	}
	return false
}

type studioManifest struct {
	Format   string          `json:"format"`
	Version  int             `json:"version"`
	Document json.RawMessage `json:"document"`
}
type studioArchive struct {
	positions [][2]int32
	zip       *zip.ReadCloser
	pages     []*zip.File
	Document  json.RawMessage
	Total     int
}

func (s *studioArchive) Close() error { return s.zip.Close() }
func (s *studioArchive) Page(request scene.PageRequest) (scene.Page, error) {
	if request.Bounds != nil {
		if err := request.Bounds.Validate(); err != nil {
			return scene.Page{}, err
		}
	}
	index := 0
	if request.Cursor != nil {
		if request.Cursor[1] != -2147483648 {
			return scene.Page{}, fmt.Errorf("Invalid project cursor")
		}
		index = int(request.Cursor[0])
	}
	if index < 0 || index >= len(s.pages) {
		return scene.Page{}, fmt.Errorf("Project page is missing")
	}
	page, err := readProjectPage(s.pages[index])
	if err != nil {
		return page, err
	}
	if request.Bounds != nil {
		filtered := page.Chunks[:0]
		for _, chunk := range page.Chunks {
			if request.Bounds.Contains(chunk.X, chunk.Z) {
				filtered = append(filtered, chunk)
			}
		}
		page.Chunks = filtered
		page.Total = 0
		for _, p := range s.positions {
			if request.Bounds.Contains(p[0], p[1]) {
				page.Total++
			}
		}
	}
	page.Reset = index == 0
	page.Complete = index == len(s.pages)-1
	page.Next = nil
	if !page.Complete {
		page.Next = &[2]int32{int32(index + 1), -2147483648}
	}
	return page, nil
}
func readProjectPage(entry *zip.File) (scene.Page, error) {
	var page scene.Page
	if entry.UncompressedSize64 > maxProjectPage {
		return page, fmt.Errorf("Project page is too large")
	}
	input, err := entry.Open()
	if err != nil {
		return page, err
	}
	defer input.Close()
	raw, err := io.ReadAll(io.LimitReader(input, maxProjectPage+1))
	if err != nil {
		return page, err
	}
	if len(raw) > maxProjectPage {
		return page, fmt.Errorf("Project page is too large")
	}
	if err = json.Unmarshal(raw, &page); err != nil {
		return page, err
	}
	if page.AtlasSize < 16 || page.AtlasSize > 8192 || page.Total < 0 || page.Total > 32768 || len(page.Chunks) > 4096 {
		return page, fmt.Errorf("Invalid map data in project")
	}
	return page, nil
}
func readStudioArchive(path string) (*studioArchive, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	result := &studioArchive{zip: archive}
	success := false
	defer func() {
		if !success {
			archive.Close()
		}
	}()
	if len(archive.File) < 2 || len(archive.File) > 4097 {
		return nil, fmt.Errorf("Invalid project structure")
	}
	entries := map[string]*zip.File{}
	var total uint64
	for _, entry := range archive.File {
		if entries[entry.Name] != nil {
			return nil, fmt.Errorf("Duplicate project entry")
		}
		entries[entry.Name] = entry
		if entry.UncompressedSize64 > 8<<30-total {
			return nil, fmt.Errorf("Project exceeds 8 GiB")
		}
		total += entry.UncompressedSize64
		if total > 8<<30 {
			return nil, fmt.Errorf("Project exceeds 8 GiB")
		}
	}
	manifest := entries["project.json"]
	if manifest == nil || manifest.UncompressedSize64 > maxProjectJSON {
		return nil, fmt.Errorf("Project document is missing or too large")
	}
	input, err := manifest.Open()
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(input, maxProjectJSON+1))
	input.Close()
	if err != nil {
		return nil, err
	}
	var metadata studioManifest
	if len(raw) > maxProjectJSON || json.Unmarshal(raw, &metadata) != nil || metadata.Format != "craftbox-project" || metadata.Version != 1 || !json.Valid(metadata.Document) {
		return nil, fmt.Errorf("Not a supported CraftBox project file")
	}
	result.Document = metadata.Document
	var sequence, scope uint64
	loaded := 0
	for i := 0; i < len(entries)-1; i++ {
		entry := entries[fmt.Sprintf("scene/%06d.json", i)]
		if entry == nil {
			return nil, fmt.Errorf("Map pages are incomplete")
		}
		page, e := readProjectPage(entry)
		if e != nil {
			return nil, e
		}
		if i == 0 {
			sequence = page.Sequence
			scope = page.Scope
			result.Total = page.Total
		}
		if page.Sequence != sequence || page.Scope != scope || page.Complete != (i == len(entries)-2) {
			return nil, fmt.Errorf("Inconsistent map pages")
		}
		result.pages = append(result.pages, entry)
		for _, chunk := range page.Chunks {
			result.positions = append(result.positions, [2]int32{chunk.X, chunk.Z})
		}
		loaded += len(page.Chunks)
		if page.Total != result.Total {
			return nil, fmt.Errorf("Inconsistent map set")
		}
	}
	if loaded != result.Total {
		return nil, fmt.Errorf("Map data in project is incomplete")
	}
	success = true
	return result, nil
}
func writeStudioArchive(path string, document json.RawMessage, getPage func(scene.PageRequest) (scene.Page, error)) (err error) {
	if len(document) > maxProjectJSON || !json.Valid(document) {
		return fmt.Errorf("Invalid project document")
	}
	output, err := os.CreateTemp(filepath.Dir(path), ".craftbox-project-*.tmp")
	if err != nil {
		return err
	}
	temporary := output.Name()
	defer func() { output.Close(); os.Remove(temporary) }()
	archive := zip.NewWriter(output)
	entry, err := archive.Create("project.json")
	if err != nil {
		return err
	}
	if err = json.NewEncoder(entry).Encode(studioManifest{Format: "craftbox-project", Version: 1, Document: document}); err != nil {
		return err
	}
	request := scene.PageRequest{TerrainOnly: true, Limit: 8}
	var sequence, scope uint64
	complete := false
	var total int64
	loaded, expected := 0, 0
	for index := 0; index < 4096; index++ {
		page, e := getPage(request)
		if e != nil {
			return e
		}
		if index == 0 {
			sequence = page.Sequence
			scope = page.Scope
			expected = page.Total
		}
		if page.Total != expected || expected < 0 || expected > 32768 {
			return fmt.Errorf("Unsupported or changed map set")
		}
		loaded += len(page.Chunks)
		if page.Sequence != sequence || page.Scope != scope || (index > 0 && page.Reset) {
			return fmt.Errorf("The map changed while saving. Save again")
		}
		raw, e := json.Marshal(page)
		if e != nil {
			return e
		}
		if len(raw) > maxProjectPage {
			return fmt.Errorf("Map page is too large")
		}
		total += int64(len(raw))
		if total > 8<<30 {
			return fmt.Errorf("Project exceeds 8 GiB")
		}
		entry, e = archive.Create(fmt.Sprintf("scene/%06d.json", index))
		if e != nil {
			return e
		}
		if _, e = entry.Write(raw); e != nil {
			return e
		}
		if page.Complete {
			complete = true
			break
		}
		if page.Next == nil {
			return fmt.Errorf("Map pages are incomplete")
		}
		request.Target = page.Sequence
		request.Scope = page.Scope
		request.Cursor = page.Next
		request.AtlasRevision = page.AtlasRevision
	}
	if !complete {
		return fmt.Errorf("Too many map pages")
	}
	if loaded != expected {
		return fmt.Errorf("Map data was not saved completely")
	}
	if err = archive.Close(); err != nil {
		return err
	}
	if err = output.Sync(); err != nil {
		return err
	}
	if err = output.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

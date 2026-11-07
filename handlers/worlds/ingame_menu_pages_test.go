package worlds

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
)

func TestMenuPagingClampsEmptyAndLastPages(t *testing.T) {
	for _, tc := range []struct{ total, page, start, end, current, pages int }{
		{0, 99, 0, 0, 0, 1}, {7, -1, 0, 6, 0, 2}, {7, 99, 6, 7, 1, 2}, {12, 1, 6, 12, 1, 2},
	} {
		a, b, c, d := menuPage(tc.total, tc.page)
		if a != tc.start || b != tc.end || c != tc.current || d != tc.pages {
			t.Fatalf("%+v got %d %d %d %d", tc, a, b, c, d)
		}
	}
}

func TestPointBrowserFiltersAndSortsWithoutChangingCaptureEvidence(t *testing.T) {
	points := []messages.CapturePoint{
		{ID: "old", Name: "Home base", Kind: "marker", Created: time.Unix(10, 0)},
		{ID: "note", Name: "Home entrance", Kind: "note", Created: time.Unix(30, 0)},
		{ID: "new", Name: "HOME garden", Kind: "marker", Created: time.Unix(20, 0)},
	}
	got := filterMenuPoints(points, "marker", " HOME ")
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Fatalf("wrong filtered order: %v", got)
	}
	if points[0].ID != "old" || points[1].ID != "note" {
		t.Fatal("browser changed original capture evidence")
	}
	if len(filterMenuPoints(points, "camera", "")) != 0 {
		t.Fatal("wrong point type leaked into filter")
	}
}

func TestBlueprintBrowserSkipsInvalidProjectsAndSearchesNames(t *testing.T) {
	folder := t.TempDir()
	for _, tc := range []struct {
		id, name string
		size     []int32
	}{
		{"b", "Tower", []int32{2, 3, 4}}, {"a", "Bridge", []int32{4, 3, 2}}, {"bad-size", "Invalid", []int32{0, 3, 4}},
	} {
		path := filepath.Join(folder, tc.id)
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(path, "recovery.craftbox"))
		if err != nil {
			t.Fatal(err)
		}
		z := zip.NewWriter(f)
		manifest, err := z.Create("project.json")
		if err != nil {
			t.Fatal(err)
		}
		if err = json.NewEncoder(manifest).Encode(map[string]any{"format": "craftbox-structure", "version": 1, "name": tc.name, "size": tc.size}); err != nil {
			t.Fatal(err)
		}
		if err = z.Close(); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(folder, "missing"), 0700); err != nil {
		t.Fatal(err)
	}
	items, skipped, err := readMenuBlueprints(folder, "")
	if err != nil || len(items) != 2 || skipped != 2 || items[0].brief.Name != "Bridge" || items[1].brief.Name != "Tower" {
		t.Fatalf("items=%v skipped=%d err=%v", items, skipped, err)
	}
	items, _, err = readMenuBlueprints(folder, "TOW")
	if err != nil || len(items) != 1 || items[0].id != "b" {
		t.Fatal("blueprint search failed")
	}
}

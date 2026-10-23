package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResultFolderResolvesExportAndDirectoryWithSpaces(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "Cube Craft")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	export := filepath.Join(directory, "Lobby.mcworld")
	if err := os.WriteFile(export, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{directory, export} {
		actual, err := resultFolder(value)
		if err != nil || actual != directory {
			t.Fatalf("folder for %q: %q, %v", value, actual, err)
		}
	}
	if _, err := resultFolder(filepath.Join(directory, "missing.mcworld")); err == nil {
		t.Fatal("missing export accepted")
	}
}

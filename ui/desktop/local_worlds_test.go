package desktop

import (
	"context"
	"errors"
	"github.com/Daniel-Ric/CraftBox/internal/localworlds"
	"testing"
)

func TestLocalWorldExportJobCancellationAndExclusion(t *testing.T) {
	a := &App{localWorldSources: map[string]localworlds.World{"world": {ID: "world", Name: "World"}}}
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := a.runLocalWorldExport(LocalWorldRequest{WorldID: "world", JobID: "first"}, func(string) (string, error) { close(entered); <-release; return "unused.mcworld", nil })
		done <- err
	}()
	<-entered
	if !a.GetLocalWorldExportProgress().Running {
		t.Fatal("job was not registered before destination dialog")
	}
	if _, err := a.runLocalWorldExport(LocalWorldRequest{WorldID: "world", JobID: "second"}, func(string) (string, error) { t.Error("second dialog opened"); return "", nil }); err == nil {
		t.Fatal("concurrent export accepted")
	}
	a.CancelLocalWorldExportJob("stale")
	a.CancelLocalWorldExportJob("first")
	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if a.GetLocalWorldExportProgress().Running {
		t.Fatal("job remained running")
	}
	if result, err := a.runLocalWorldExport(LocalWorldRequest{WorldID: "world"}, func(string) (string, error) { return "", nil }); err != nil || result != nil {
		t.Fatalf("dialog cancellation: %v %v", result, err)
	}
}

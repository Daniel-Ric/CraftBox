//go:build !windows

package diagnostics

import (
	"os/exec"
	"runtime"
)

func OpenPath(path string, folder bool) error {
	program := "xdg-open"
	if runtime.GOOS == "darwin" {
		program = "open"
	}
	cmd := exec.Command(program, path)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

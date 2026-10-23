//go:build !windows

package desktop

import (
	"os/exec"
	"runtime"
)

func openResultDirectory(directory string) error {
	program := "xdg-open"
	if runtime.GOOS == "darwin" {
		program = "open"
	}
	command := exec.Command(program, directory)
	if err := command.Start(); err != nil {
		return err
	}
	go func() { _ = command.Wait() }()
	return nil
}

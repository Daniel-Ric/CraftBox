//go:build windows

package diagnostics

import (
	"os/exec"
)

func OpenPath(path string, folder bool) error {
	program := "notepad.exe"
	if folder {
		program = "explorer.exe"
	}
	cmd := exec.Command(program, path)
	// This is a user-requested GUI window. SW_HIDE would also hide Notepad,
	// making a successful launch appear to do nothing.
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

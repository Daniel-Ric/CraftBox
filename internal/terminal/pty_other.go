//go:build !windows

package terminal

import "errors"

func startPowerShell(Request) (Process, Info, error) {
	return nil, Info{}, errors.New("PowerShell terminals require Windows")
}

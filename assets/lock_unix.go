//go:build !windows

package assets

import (
	"golang.org/x/sys/unix"
	"os"
)

func lockAssets(file *os.File) error { return unix.Flock(int(file.Fd()), unix.LOCK_EX) }

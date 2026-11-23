//go:build !windows

package main

import "os/exec"

func hideDiagnosticProcess(*exec.Cmd) {}

func observeNativeCrash(int) (func() string, func()) { return nil, func() {} }

//go:build !windows

package main

import "os/exec"

func hideDiagnosticProcess(*exec.Cmd) {}

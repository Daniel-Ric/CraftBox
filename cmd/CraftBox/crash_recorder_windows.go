//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func hideDiagnosticProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

func observeNativeCrash(pid int) (func() string, func()) {
	process, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return nil, func() {}
	}
	return func() string {
		result, err := windows.WaitForSingleObject(process, 250)
		if err != nil || result != windows.WAIT_OBJECT_0 {
			return ""
		}
		var status uint32
		if err := windows.GetExitCodeProcess(process, &status); err != nil {
			return ""
		}
		return windowsCrashEvidence(windows.NTStatus(status))
	}, func() { _ = windows.CloseHandle(process) }
}

func windowsCrashEvidence(status windows.NTStatus) string {
	switch status {
	case windows.STATUS_ACCESS_VIOLATION, windows.STATUS_IN_PAGE_ERROR, windows.STATUS_ILLEGAL_INSTRUCTION,
		windows.STATUS_INTEGER_DIVIDE_BY_ZERO, windows.STATUS_STACK_OVERFLOW,
		windows.STATUS_HEAP_CORRUPTION, windows.STATUS_STACK_BUFFER_OVERRUN:
		return fmt.Sprintf("Windows exception: 0x%08X\n", uint32(status))
	default:
		return ""
	}
}

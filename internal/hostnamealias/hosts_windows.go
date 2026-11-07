package hostnamealias

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"unsafe"
)

var replaceFile = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")
var flushResolver = windows.NewLazySystemDLL("dnsapi.dll").NewProc("DnsFlushResolverCache")

func acquirePlatform(ctx context.Context, name string) (*Lease, error) {
	return acquireRemote(ctx, name, func(address, token string) error {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		verb := "runas"
		if windows.GetCurrentProcessToken().IsElevated() {
			verb = "open"
		}
		operation, _ := windows.UTF16PtrFromString(verb)
		file, err := windows.UTF16PtrFromString(executable)
		if err != nil {
			return err
		}
		args, _ := windows.UTF16PtrFromString(helperFlag + " " + windows.EscapeArg(address) + " " + windows.EscapeArg(token))
		return windows.ShellExecute(0, operation, file, args, nil, windows.SW_HIDE)
	})
}

func systemHosts() (string, func(), error) {
	// A named event acts as an exclusive lifetime handle and can be closed from
	// any goroutine. Windows releases it automatically if the process crashes.
	eventName, _ := windows.UTF16PtrFromString(`Global\CraftBox.HostnameAlias.Active.v1`)
	event, err := windows.CreateEvent(nil, 1, 0, eventName)
	if err == windows.ERROR_ALREADY_EXISTS {
		if event != 0 {
			windows.CloseHandle(event)
		}
		return "", nil, fmt.Errorf("another CraftBox session is using destination hostname aliases")
	}
	if err != nil {
		return "", nil, err
	}
	root, err := windows.GetSystemDirectory()
	if err != nil {
		windows.CloseHandle(event)
		return "", nil, fmt.Errorf("Windows system directory is unavailable")
	}
	return filepath.Join(root, "drivers", "etc", "hosts"), func() { windows.CloseHandle(event) }, nil
}
func atomicReplace(path, tmp string) error {
	target, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	source, err := windows.UTF16PtrFromString(tmp)
	if err != nil {
		return err
	}
	backupPath := tmp + ".original"
	backup, err := windows.UTF16PtrFromString(backupPath)
	if err != nil {
		return err
	}
	ok, _, err := replaceFile.Call(uintptr(unsafe.Pointer(target)), uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(backup)), 0, 0, 0)
	if ok == 0 {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			if restoreErr := windows.MoveFileEx(backup, target, windows.MOVEFILE_WRITE_THROUGH); restoreErr != nil {
				return fmt.Errorf("hosts replacement failed (%v); recovery copy: %s; restore failed: %w", err, backupPath, restoreErr)
			}
		}
		return fmt.Errorf("hosts replacement failed: %w; any recovery copy is at %s", err, backupPath)
	}
	os.Remove(backupPath)
	return nil
}
func flushDNS() { flushResolver.Call() }

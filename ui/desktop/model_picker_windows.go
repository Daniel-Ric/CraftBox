package desktop

import (
	"context"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

func init() {
	watchModelPickerKey = func(ctx context.Context, pick func()) {
		go func() {
			user := windows.NewLazySystemDLL("user32.dll")
			key := user.NewProc("GetAsyncKeyState")
			foreground := user.NewProc("GetForegroundWindow")
			owner := user.NewProc("GetWindowThreadProcessId")
			timer := time.NewTicker(40 * time.Millisecond)
			defer timer.Stop()
			down := false
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					value, _, _ := key.Call(0x77)
					pressed := value&0x8000 != 0
					if pressed && !down {
						hwnd, _, _ := foreground.Call()
						var pid uint32
						owner.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
						process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
						if err == nil {
							var path [1024]uint16
							size := uint32(len(path))
							err = windows.QueryFullProcessImageName(process, 0, &path[0], &size)
							windows.CloseHandle(process)
							if err == nil && strings.HasPrefix(strings.ToLower(filepath.Base(windows.UTF16ToString(path[:size]))), "minecraft") {
								go pick()
							}
						}
					}
					down = pressed
				}
			}
		}()
	}
}

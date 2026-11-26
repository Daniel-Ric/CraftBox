package desktop

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

const flarialPackageFamily = "Flarial.Launcher_0jrgakbnj75vr"
const minecraftPackageFamily = "Microsoft.MinecraftUWP_8wekyb3d8bbwe"

var getPackagesByPackageFamily = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetPackagesByPackageFamily")
var getProcessPackageFamily = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetPackageFamilyName")
var isHungAppWindow = windows.NewLazySystemDLL("user32.dll").NewProc("IsHungAppWindow")

type minecraftWindowSearch struct {
	processes map[uint32]bool
	found     bool
}

var findMinecraftChild = windows.NewCallback(func(hwnd windows.HWND, search *minecraftWindowSearch) uintptr {
	var pid uint32
	_, _ = windows.GetWindowThreadProcessId(hwnd, &pid)
	if search.processes[pid] && windows.IsWindowVisible(hwnd) {
		hung, _, _ := isHungAppWindow.Call(uintptr(hwnd))
		search.found = hung == 0
	}
	if search.found {
		return 0
	}
	return 1
})

var findMinecraftWindow = windows.NewCallback(func(hwnd windows.HWND, search *minecraftWindowSearch) uintptr {
	if !windows.IsWindowVisible(hwnd) {
		return 1
	}
	var pid uint32
	_, _ = windows.GetWindowThreadProcessId(hwnd, &pid)
	if search.processes[pid] {
		hung, _, _ := isHungAppWindow.Call(uintptr(hwnd))
		search.found = hung == 0
	} else {
		var class [32]uint16
		if _, err := windows.GetClassName(hwnd, &class[0], int32(len(class))); err == nil && windows.UTF16ToString(class[:]) == "ApplicationFrameWindow" {
			windows.EnumChildWindows(hwnd, findMinecraftChild, unsafe.Pointer(search))
		}
	}
	if search.found {
		return 0
	}
	return 1
})

var flarialChannels = [...]struct{ id, filename string }{
	{"release", "Flarial.Client.Release.dll"},
	{"beta", "Flarial.Client.Beta.dll"},
	{"alpha", "Flarial.Client.Alpha.dll"},
}

func regularFlarialFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func flarialClientDLLs(roots []string) map[string]string {
	clients := make(map[string]string)
	for _, channel := range flarialChannels {
		for _, root := range roots {
			if path := filepath.Join(root, channel.filename); regularFlarialFile(path) {
				clients[channel.id] = path
				break
			}
		}
	}
	return clients
}

func discoverFlarial() (flarialInstallation, bool) {
	var installation flarialInstallation
	var roots []string
	if local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0); err == nil {
		cache := filepath.Join(local, "Packages", flarialPackageFamily, "LocalCache", "Local", "Flarial")
		roots = append(roots, filepath.Join(local, "Flarial", "Launcher"), filepath.Join(local, "Flarial", "Client"), filepath.Join(cache, "Launcher"), filepath.Join(cache, "Client"))
	}
	if documents, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0); err == nil {
		legacy := filepath.Join(documents, "Minecraft", "Flarial")
		roots = append(roots, legacy)
		if path := filepath.Join(legacy, "Flarial.Launcher.exe"); regularFlarialFile(path) {
			installation.portable = path
		}
	}
	if getPackagesByPackageFamily.Find() == nil {
		family, _ := windows.UTF16PtrFromString(flarialPackageFamily)
		var count, length uint32
		status, _, _ := getPackagesByPackageFamily.Call(uintptr(unsafe.Pointer(family)), uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&length)), 0)
		if (status == 0 || status == uintptr(windows.ERROR_INSUFFICIENT_BUFFER)) && count > 0 {
			installation.appID = flarialPackageFamily + "!App"
		}
	}
	installation.clients = flarialClientDLLs(roots)
	return installation, installation.appID != "" || installation.portable != ""
}

func flarialProcesses() (bool, map[uint32]bool, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, nil, err
	}
	defer windows.CloseHandle(snapshot)
	var launcher bool
	minecraft := make(map[uint32]bool)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		switch strings.ToLower(windows.UTF16ToString(entry.ExeFile[:])) {
		case "flarial.launcher.exe":
			launcher = true
		case "minecraft.windows.exe", "minecraft.exe":
			minecraft[entry.ProcessID] = true
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return false, nil, err
	}
	return launcher, minecraft, nil
}

func checkFlarialProcesses() error {
	launcher, minecraft, err := flarialProcesses()
	if err != nil {
		return err
	}
	if launcher {
		return errors.New("Close Flarial Launcher to continue.")
	}
	if len(minecraft) > 0 {
		return errors.New("Close Minecraft to continue.")
	}
	return nil
}

func flarialMinecraftReady(packaged bool) (bool, error) {
	launcher, minecraft, err := flarialProcesses()
	if err != nil || launcher || len(minecraft) == 0 {
		return false, err
	}
	if packaged {
		for pid := range minecraft {
			if !isPackagedMinecraft(pid) {
				delete(minecraft, pid)
			}
		}
		if len(minecraft) == 0 {
			return false, nil
		}
	}
	search := minecraftWindowSearch{processes: minecraft}
	err = windows.EnumWindows(findMinecraftWindow, unsafe.Pointer(&search))
	if search.found {
		return true, nil
	}
	return false, err
}

func isPackagedMinecraft(pid uint32) bool {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(process)
	var family [65]uint16
	length := uint32(len(family))
	status, _, _ := getProcessPackageFamily.Call(uintptr(process), uintptr(unsafe.Pointer(&length)), uintptr(unsafe.Pointer(&family[0])))
	return status == 0 && windows.UTF16ToString(family[:]) == minecraftPackageFamily
}

func launchFlarial(installation flarialInstallation, dll string) error {
	args := []string{"--inject", dll}
	if installation.appID == "" {
		command := exec.Command(installation.portable, args...)
		command.Dir = filepath.Dir(installation.portable)
		if err := command.Start(); err != nil {
			return err
		}
		go func() { _ = command.Wait() }()
		return nil
	}
	return activateFlarialApp(installation.appID, windows.ComposeCommandLine(args))
}

func activateFlarialApp(id, arguments string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		status, ok := err.(*ole.OleError)
		if !ok || status.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()
	activation, err := ole.CreateInstance(ole.NewGUID("{45BA127D-10A8-46EA-8AB7-56EA9078943C}"), ole.NewGUID("{2e941141-7f97-4756-ba1d-9decde894a3d}"))
	if err != nil {
		return err
	}
	defer activation.Release()
	appID, err := windows.UTF16PtrFromString(id)
	if err != nil {
		return err
	}
	args, err := windows.UTF16PtrFromString(arguments)
	if err != nil {
		return err
	}
	var pid uint32
	method := (*[4]uintptr)(unsafe.Pointer(activation.VTable()))[3]
	result, _, _ := syscall.SyscallN(method, uintptr(unsafe.Pointer(activation)), uintptr(unsafe.Pointer(appID)), uintptr(unsafe.Pointer(args)), 0, uintptr(unsafe.Pointer(&pid)))
	if int32(result) < 0 {
		return ole.NewError(result)
	}
	return nil
}

func openMinecraftURL(_ context.Context, value string) error {
	uri, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	// The browser launcher uses SW_SHOWNORMAL, which restores stale normal
	// coordinates even for a maximized Minecraft window. Let Windows maximize
	// against the monitor's work area, including its taskbar and DPI settings.
	return windows.ShellExecute(0, nil, uri, nil, nil, windows.SW_SHOWMAXIMIZED)
}

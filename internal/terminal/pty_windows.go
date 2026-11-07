//go:build windows

package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/UserExistsError/conpty"
	"golang.org/x/sys/windows"
)

type windowsPTY struct {
	p      *conpty.ConPty
	job    windows.Handle
	once   sync.Once
	closed atomic.Bool
}

func startPowerShell(r Request) (Process, Info, error) {
	shell, err := exec.LookPath("pwsh.exe")
	if err != nil {
		shell = filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, Info{}, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		windows.CloseHandle(job)
		return nil, Info{}, err
	}
	p, err := conpty.Start(windows.EscapeArg(shell)+" -NoLogo -NoProfile", conpty.ConPtyDimensions(r.Cols, r.Rows), conpty.ConPtyWorkDir(r.Cwd), conpty.ConPtyEnv(append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")))
	if err != nil {
		windows.CloseHandle(job)
		return nil, Info{}, err
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid()))
	if err == nil {
		err = windows.AssignProcessToJobObject(job, proc)
		windows.CloseHandle(proc)
	}
	if err != nil {
		go io.Copy(io.Discard, p)
		_ = p.Close()
		windows.CloseHandle(job)
		return nil, Info{}, fmt.Errorf("assign terminal process tree: %w", err)
	}
	return &windowsPTY{p: p, job: job}, Info{PID: p.Pid(), Shell: shell}, nil
}
func (p *windowsPTY) Read(b []byte) (int, error) { return p.p.Read(b) }
func (p *windowsPTY) Write(b []byte) (int, error) {
	if p.closed.Load() {
		return 0, os.ErrClosed
	}
	return p.p.Write(b)
}
func (p *windowsPTY) Resize(c, r int) error {
	if p.closed.Load() {
		return os.ErrClosed
	}
	return p.p.Resize(c, r)
}
func (p *windowsPTY) Wait(ctx context.Context) (int, error) {
	n, err := p.p.Wait(ctx)
	return int(n), err
}
func (p *windowsPTY) Close() error {
	var err error
	p.once.Do(func() { p.closed.Store(true); windows.CloseHandle(p.job); err = p.p.Close() })
	return err
}
